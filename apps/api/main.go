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
	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	// Initialize Redis
	if err := database.InitRedis(); err != nil {
		log.Printf("Redis initialization failed: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           apisrv.New(),
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	log.Printf("Connect RPC API listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Print(gracefulShutdownMessage)
			return
		}
		log.Fatalf("server failed: %v", err)
	}
}
