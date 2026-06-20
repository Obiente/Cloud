package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pkgsftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

type SFTPServer struct {
	address       string
	authenticator *Authenticator
	config        *ssh.ServerConfig
	listener      net.Listener
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func NewSFTPServer(address string, hostKeyPath string, authenticator *Authenticator) (*SFTPServer, error) {
	if address == "" {
		address = "0.0.0.0:2222"
	}
	if authenticator == nil {
		return nil, fmt.Errorf("authenticator is required")
	}

	hostKey, err := loadOrCreateHostKey(hostKeyPath)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := &SFTPServer{
		address:       address,
		authenticator: authenticator,
		ctx:           ctx,
		cancel:        cancel,
	}
	server.config = &ssh.ServerConfig{
		PasswordCallback: server.passwordCallback,
		ServerVersion:    "SSH-2.0-ObienteFileTransfer",
	}
	server.config.AddHostKey(hostKey)

	return server, nil
}

func (s *SFTPServer) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.address, err)
	}
	s.listener = listener
	logger.Info("[FileTransfer] SFTP listening on %s", s.address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				logger.Warn("[FileTransfer] SFTP accept failed: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *SFTPServer) Shutdown() error {
	s.cancel()
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *SFTPServer) passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	session, err := s.authenticator.Authenticate(s.ctx, string(password))
	if err != nil {
		logger.Warn("[FileTransfer] SFTP auth failed for login %q from %s: %v", conn.User(), conn.RemoteAddr(), err)
		return nil, fmt.Errorf("authentication failed")
	}

	logger.Info("[FileTransfer] SFTP auth ok: credential=%s resource=%s:%s user=%s org=%s",
		session.CredentialID, session.ResourceType, session.ResourceID, session.UserID, session.OrganizationID)

	return &ssh.Permissions{
		Extensions: map[string]string{
			"credential_id":   session.CredentialID,
			"user_id":         session.UserID,
			"organization_id": session.OrganizationID,
			"resource_type":   session.ResourceType,
			"resource_id":     session.ResourceID,
			"root_path":       session.RootPath,
			"permissions":     serializePermissions(session.Permissions),
		},
	}, nil
}

func (s *SFTPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	sshConn, channels, requests, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		logger.Warn("[FileTransfer] SSH handshake failed from %s: %v", conn.RemoteAddr(), err)
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(requests)

	extensions := sshConn.Permissions.Extensions
	session := &Session{
		CredentialID:   extensions["credential_id"],
		UserID:         extensions["user_id"],
		OrganizationID: extensions["organization_id"],
		ResourceType:   extensions["resource_type"],
		ResourceID:     extensions["resource_id"],
		RootPath:       extensions["root_path"],
		Permissions:    deserializePermissions(extensions["permissions"]),
	}

	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		accepted, requests, err := channel.Accept()
		if err != nil {
			logger.Warn("[FileTransfer] Accept channel failed: %v", err)
			continue
		}
		s.wg.Add(1)
		go s.handleChannel(accepted, requests, session)
	}
}

func (s *SFTPServer) handleChannel(channel ssh.Channel, requests <-chan *ssh.Request, session *Session) {
	defer s.wg.Done()
	defer channel.Close()

	for req := range requests {
		if req.Type != "subsystem" || parseSubsystem(req.Payload) != "sftp" {
			_ = req.Reply(false, nil)
			continue
		}
		_ = req.Reply(true, nil)

		handler := newSFTPHandler(session)
		server := pkgsftp.NewRequestServer(channel, pkgsftp.Handlers{
			FileGet:  handler,
			FilePut:  handler,
			FileCmd:  handler,
			FileList: handler,
		})
		if err := server.Serve(); err != nil && !errors.Is(err, io.EOF) {
			logger.Warn("[FileTransfer] SFTP session failed for credential=%s: %v", session.CredentialID, err)
		}
		return
	}
}

func parseSubsystem(payload []byte) string {
	if len(payload) < 4 {
		return ""
	}
	size := binary.BigEndian.Uint32(payload[:4])
	if int(size) > len(payload)-4 {
		return ""
	}
	return string(payload[4 : 4+size])
}

func loadOrCreateHostKey(path string) (ssh.Signer, error) {
	if path == "" {
		path = "/var/lib/obiente/file-transfer/ssh_host_key"
	}
	if data, err := os.ReadFile(path); err == nil {
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse host key %s: %w", path, err)
		}
		return signer, nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("generate host key: %w", err)
	}
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create host key directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, fmt.Errorf("write host key: %w", err)
	}
	return ssh.NewSignerFromKey(key)
}

func serializePermissions(permissions []Permission) string {
	parts := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		parts = append(parts, string(permission))
	}
	return strings.Join(parts, ",")
}

func deserializePermissions(serialized string) []Permission {
	parts := strings.Split(serialized, ",")
	permissions := make([]Permission, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			permissions = append(permissions, Permission(part))
		}
	}
	return permissions
}
