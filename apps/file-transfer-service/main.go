package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	filesvc "file-transfer-service/internal/service"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/health"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"

	_ "github.com/joho/godotenv/autoload"
)

const (
	readHeaderTimeout = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 2 * time.Minute
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	logger.Init()

	logger.Info("=== File Transfer Service Starting ===")

	database.RegisterModels(
		&database.FileTransferCredential{},
		&database.GameServer{},
	)

	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	if err := database.InitRedis(); err != nil {
		logger.Warn("Redis cache initialization failed: %v", err)
	}

	volumeRoot := getenvDefault("FILE_TRANSFER_VOLUME_ROOT", "/var/lib/obiente/volumes")
	sftpPort := getenvDefault("SFTP_PORT", "2222")
	httpPort := getenvDefault("PORT", "3022")
	hostKeyPath := getenvDefault("SFTP_HOST_KEY_PATH", "/var/lib/obiente/file-transfer/ssh_host_key")

	authenticator := filesvc.NewAuthenticator(volumeRoot)
	sftpServer, err := filesvc.NewSFTPServer("0.0.0.0:"+sftpPort, hostKeyPath, authenticator)
	if err != nil {
		logger.Fatalf("failed to initialize SFTP server: %v", err)
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sftpErr := make(chan error, 1)
	go func() {
		sftpErr <- sftpServer.Start()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health.HandleHealth("file-transfer-service", func() (bool, string, map[string]interface{}) {
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			return false, "database unavailable", nil
		}
		return true, "healthy", map[string]interface{}{
			"sftp_port":   sftpPort,
			"volume_root": volumeRoot,
		}
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("file-transfer-service"))
	})

	var handler http.Handler = mux
	handler = middleware.CORSHandler(handler)
	handler = middleware.RequestLogger(handler)

	httpServer := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	httpErr := make(chan error, 1)
	go func() {
		logger.Info("=== File Transfer HTTP Ready - Listening on %s ===", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpErr <- err
		}
	}()

	select {
	case err := <-sftpErr:
		logger.Fatalf("SFTP server failed: %v", err)
	case err := <-httpErr:
		logger.Fatalf("HTTP server failed: %v", err)
	case <-shutdownCtx.Done():
		logger.Info("=== Shutting down file transfer service ===")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Warn("HTTP shutdown failed: %v", err)
		}
		if err := sftpServer.Shutdown(); err != nil {
			logger.Warn("SFTP shutdown failed: %v", err)
		}
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
