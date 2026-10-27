package main

import (
	"context"
	gameserverorchestrator "gameservers-service/internal/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/health"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
	"sync/atomic"
	"time"
)

func newRuntimeHandler(shutdownCtx context.Context, mux *http.ServeMux, readyManager *atomic.Pointer[gameserverorchestrator.GameServerManager]) http.Handler {
	readyHandler := health.HandleHealth("gameservers-service", func() (bool, string, map[string]interface{}) {
		manager := readyManager.Load()
		if manager == nil {
			return false, "initializing Docker", nil
		}
		ctx, cancel := context.WithTimeout(shutdownCtx, 3*time.Second)
		defer cancel()
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.PingContext(ctx) != nil {
			return false, "database unavailable", nil
		}
		if err := manager.CheckReady(ctx); err != nil {
			return false, "Docker unavailable", nil
		}
		return true, "ready", nil
	})
	mux.HandleFunc("/health", readyHandler)
	mux.HandleFunc("/readyz", readyHandler)
	mux.HandleFunc("/livez", health.HandleHealth("gameservers-service", func() (bool, string, map[string]interface{}) {
		return shutdownCtx.Err() == nil, "alive", nil
	}))

	// Wrap with h2c for HTTP/2
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/livez" && r.URL.Path != "/readyz" && r.URL.Path != "/health" && readyManager.Load() == nil {
			http.Error(w, "Service is starting", http.StatusServiceUnavailable)
			return
		}
		mux.ServeHTTP(w, r)
	}), &http2.Server{})

}
