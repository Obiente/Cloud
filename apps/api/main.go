package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	apisrv "api/internal/server"
)

const (
	defaultPort             = "3001"
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

func main() {
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
