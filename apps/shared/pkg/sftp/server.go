package sftp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Permission represents SFTP access permissions
type Permission string

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
)

// AuthValidator validates API keys and returns user info and permissions
type AuthValidator interface {
	ValidateAPIKey(ctx context.Context, apiKey string) (userID string, orgID string, permissions []Permission, err error)
}

// AuditLogger logs SFTP operations for audit trail
type AuditLogger interface {
	LogOperation(ctx context.Context, entry AuditEntry) error
}

// AuditEntry represents an SFTP operation for auditing
type AuditEntry struct {
	UserID       string
	OrgID        string
	Operation    string // "upload", "download", "delete", "mkdir", "rename", "list"
	Path         string
	Success      bool
	ErrorMessage string
	BytesWritten int64
	BytesRead    int64
}

// Server is an SFTP server with API key authentication
type Server struct {
	listener      net.Listener
	config        *ssh.ServerConfig
	authValidator AuthValidator
	auditLogger   AuditLogger
	basePath      string
	hostKey       ssh.Signer
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// Config holds SFTP server configuration
type Config struct {
	Address       string        // Address to listen on (e.g., "0.0.0.0:2222")
	BasePath      string        // Base directory for SFTP files
	HostKeyPath   string        // Path to host private key (optional, will generate if missing)
	AuthValidator AuthValidator // API key validator
	AuditLogger   AuditLogger   // Audit logger
}

// NewServer creates a new SFTP server
func NewServer(cfg *Config) (*Server, error) {
	if cfg.AuthValidator == nil {
		return nil, fmt.Errorf("auth validator is required")
	}
	if cfg.BasePath == "" {
		return nil, fmt.Errorf("base path is required")
	}
	if cfg.Address == "" {
		cfg.Address = "0.0.0.0:2222"
	}

	// Ensure base path exists
	if err := os.MkdirAll(cfg.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	// Load or generate host key
	hostKey, err := loadOrGenerateHostKey(cfg.HostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load host key: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	srv := &Server{
		authValidator: cfg.AuthValidator,
		auditLogger:   cfg.AuditLogger,
		basePath:      cfg.BasePath,
		hostKey:       hostKey,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Configure SSH server
	srv.config = &ssh.ServerConfig{
		PasswordCallback: srv.passwordCallback,
	}
	srv.config.AddHostKey(hostKey)

	// Start listening
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	srv.listener = listener

	logger.Info("[SFTP] Server listening on %s", cfg.Address)

	return srv, nil
}

// Start starts accepting SFTP connections
func (s *Server) Start() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				logger.Error("[SFTP] Failed to accept connection: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	logger.Info("[SFTP] Shutting down server")
	s.cancel()
	
	if s.listener != nil {
		s.listener.Close()
	}
	
	s.wg.Wait()
	logger.Info("[SFTP] Server shutdown complete")
	return nil
}

// passwordCallback validates API keys as passwords
func (s *Server) passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	apiKey := string(password)
	
	// Validate API key
	userID, orgID, permissions, err := s.authValidator.ValidateAPIKey(s.ctx, apiKey)
	if err != nil {
		logger.Warn("[SFTP] Authentication failed for user %s: %v", conn.User(), err)
		return nil, fmt.Errorf("authentication failed")
	}

	logger.Info("[SFTP] User authenticated: %s (org: %s, permissions: %v)", userID, orgID, permissions)

	// Store user info and permissions in SSH permissions
	perms := &ssh.Permissions{
		Extensions: map[string]string{
			"user_id":     userID,
			"org_id":      orgID,
			"permissions": serializePermissions(permissions),
		},
	}

	return perms, nil
}

// handleConnection handles a single SSH connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		logger.Error("[SFTP] SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	// Discard all out-of-band requests
	go ssh.DiscardRequests(reqs)

	// Extract user info from permissions
	userID := sshConn.Permissions.Extensions["user_id"]
	orgID := sshConn.Permissions.Extensions["org_id"]
	permissions := deserializePermissions(sshConn.Permissions.Extensions["permissions"])

	logger.Info("[SFTP] Connection established for user %s", userID)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			logger.Error("[SFTP] Failed to accept channel: %v", err)
			continue
		}

		go s.handleChannel(channel, requests, userID, orgID, permissions)
	}
}

// handleChannel handles SFTP requests on a channel
func (s *Server) handleChannel(channel ssh.Channel, requests <-chan *ssh.Request, userID, orgID string, permissions []Permission) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "subsystem":
			if string(req.Payload[4:]) == "sftp" {
				req.Reply(true, nil)
				
				// Create user-specific handler
				handler := newUserHandler(s.basePath, orgID, userID, permissions, s.auditLogger)
				
				// Create handlers struct
				handlers := sftp.Handlers{
					FileGet:  handler,
					FilePut:  handler,
					FileCmd:  handler,
					FileList: handler,
				}
				
				// Start SFTP server
				server := sftp.NewRequestServer(channel, handlers)
				if err := server.Serve(); err != nil && err != io.EOF {
					logger.Error("[SFTP] Server error for user %s: %v", userID, err)
				}
				logger.Info("[SFTP] Session ended for user %s", userID)
				return
			}
		}
		
		if req.WantReply {
			req.Reply(false, nil)
		}
	}
}

// loadOrGenerateHostKey loads or generates an SSH host key
func loadOrGenerateHostKey(keyPath string) (ssh.Signer, error) {
	// Try to load existing key
	if keyPath != "" {
		keyBytes, err := os.ReadFile(keyPath)
		if err == nil {
			key, err := ssh.ParsePrivateKey(keyBytes)
			if err == nil {
				logger.Info("[SFTP] Loaded host key from %s", keyPath)
				return key, nil
			}
		}
	}

	// Generate new key
	logger.Info("[SFTP] Generating new host key")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Save key if path provided
	if keyPath != "" {
		keyDir := filepath.Dir(keyPath)
		if err := os.MkdirAll(keyDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create key directory: %w", err)
		}

		privateKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})

		if err := os.WriteFile(keyPath, privateKeyPEM, 0600); err != nil {
			return nil, fmt.Errorf("failed to save host key: %w", err)
		}
		logger.Info("[SFTP] Saved host key to %s", keyPath)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return signer, nil
}

func serializePermissions(perms []Permission) string {
	result := ""
	for i, p := range perms {
		if i > 0 {
			result += ","
		}
		result += string(p)
	}
	return result
}

func deserializePermissions(s string) []Permission {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	perms := make([]Permission, len(parts))
	for i, p := range parts {
		perms[i] = Permission(p)
	}
	return perms
}
