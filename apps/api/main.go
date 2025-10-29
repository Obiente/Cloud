package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"api/internal/database"
	apisrv "api/internal/server"

	_ "github.com/joho/godotenv/autoload"
)

const (
	defaultPort             = "3001"
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

func main() {
	// Set log output and flags
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	log.Println("=== Obiente Cloud API Starting ===")
	log.Printf("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))
	log.Printf("CORS_ORIGIN: %s", os.Getenv("CORS_ORIGIN"))

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	log.Println("✓ Database initialized")

	// Initialize Redis
	if err := database.InitRedis(); err != nil {
		log.Printf("Redis initialization failed: %v", err)
	} else {
		log.Println("✓ Redis initialized")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Println("✓ Creating HTTP server with middleware...")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           apisrv.New(),
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	log.Printf("=== Server Ready - Listening on %s ===", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Print(gracefulShutdownMessage)
			return
		}
		log.Fatalf("server failed: %v", err)
	}
}
