package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"databases-service/internal/secrets"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

const (
	dialTimeout = 10 * time.Second
)

// Proxy is the main TCP proxy server
type Proxy struct {
	registry      *RouteRegistry
	dockerClient  *docker.Client
	secretManager *secrets.SecretManager

	pgListener    net.Listener
	mysqlListener net.Listener
	mongoListener net.Listener

	redisManager *RedisPortManager

	activeConns sync.WaitGroup
	running     atomic.Bool
	stopCh      chan struct{}
}

// NewProxy creates a new proxy instance
func NewProxy(registry *RouteRegistry, dockerClient *docker.Client, secretManager *secrets.SecretManager) *Proxy {
	p := &Proxy{
		registry:      registry,
		dockerClient:  dockerClient,
		secretManager: secretManager,
		stopCh:        make(chan struct{}),
	}
	p.redisManager = NewRedisPortManager(p)
	return p
}

// Start starts all proxy listeners
func (p *Proxy) Start(ctx context.Context) error {
	p.running.Store(true)

	var err error

	// PostgreSQL listener on 5432
	p.pgListener, err = net.Listen("tcp", ":5432")
	if err != nil {
		logger.Warn("Failed to start PostgreSQL proxy listener: %v", err)
	} else {
		go p.acceptLoop(p.pgListener, "postgresql")
		logger.Info("PostgreSQL proxy listening on :5432")
	}

	// MySQL listener on 3306
	p.mysqlListener, err = net.Listen("tcp", ":3306")
	if err != nil {
		logger.Warn("Failed to start MySQL proxy listener: %v", err)
	} else {
		go p.acceptLoop(p.mysqlListener, "mysql")
		logger.Info("MySQL proxy listening on :3306")
	}

	// MongoDB listener on 27017
	p.mongoListener, err = net.Listen("tcp", ":27017")
	if err != nil {
		logger.Warn("Failed to start MongoDB proxy listener: %v", err)
	} else {
		go p.acceptLoop(p.mongoListener, "mongodb")
		logger.Info("MongoDB proxy listening on :27017")
	}

	// Start Redis listeners for existing routes
	p.startRedisListeners()

	// Start IP refresh
	p.registry.StartIPRefresh(ctx)

	// Start auto-sleep monitor
	go p.autoSleepMonitor(ctx)

	return nil
}

// Stop stops the proxy and drains active connections
func (p *Proxy) Stop() {
	p.running.Store(false)
	close(p.stopCh)

	if p.pgListener != nil {
		p.pgListener.Close()
	}
	if p.mysqlListener != nil {
		p.mysqlListener.Close()
	}
	if p.mongoListener != nil {
		p.mongoListener.Close()
	}

	p.redisManager.StopAll()
	p.registry.StopIPRefresh()

	// Wait for active connections with timeout
	done := make(chan struct{})
	go func() {
		p.activeConns.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All proxy connections drained")
	case <-time.After(30 * time.Second):
		logger.Warn("Proxy drain timeout, some connections may be interrupted")
	}
}

// StartRedisListener starts a Redis listener for a specific route
func (p *Proxy) StartRedisListener(route *Route) error {
	if route.RedisPort <= 0 {
		return nil
	}
	return p.redisManager.StartListener(route.RedisPort, route)
}

// StopRedisListener stops a Redis listener
func (p *Proxy) StopRedisListener(port int) {
	p.redisManager.StopListener(port)
}

// Healthy returns true if the proxy is running
func (p *Proxy) Healthy() bool {
	return p.running.Load()
}

func (p *Proxy) acceptLoop(ln net.Listener, protocol string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-p.stopCh:
				return
			default:
				logger.Debug("Accept error on %s: %v", protocol, err)
				return
			}
		}

		p.activeConns.Add(1)
		go func() {
			defer p.activeConns.Done()
			p.handleConnection(conn, protocol)
		}()
	}
}

func (p *Proxy) handleConnection(conn net.Conn, protocol string) {
	switch protocol {
	case "postgresql":
		p.handlePostgres(conn)
	case "mysql":
		p.handleMySQL(conn)
	case "mongodb":
		p.handleMongoDB(conn)
	}
}

// bidirectionalCopy performs bidirectional TCP forwarding
func (p *Proxy) bidirectionalCopy(client, backend net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(backend, client)
		if tc, ok := backend.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	go func() {
		io.Copy(client, backend)
		if tc, ok := client.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	// Wait for either direction to finish
	<-done
}

// wakeAndConnect wakes a sleeping database and returns the backend address
func (p *Proxy) wakeAndConnect(route *Route) (string, error) {
	if p.registry.OnWake == nil {
		return "", fmt.Errorf("no wake function configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ip, err := p.registry.OnWake(ctx, route)
	if err != nil {
		return "", fmt.Errorf("failed to wake database: %w", err)
	}

	return net.JoinHostPort(ip, fmt.Sprintf("%d", route.InternalPort)), nil
}

// autoSleepMonitor periodically checks for idle databases and puts them to sleep
func (p *Proxy) autoSleepMonitor(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			routes := p.registry.GetSleepableRoutes()
			for _, route := range routes {
				if p.registry.OnSleep == nil {
					continue
				}
				sleepCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := p.registry.OnSleep(sleepCtx, route); err != nil {
					logger.Error("Failed to auto-sleep database %s: %v", route.DatabaseID, err)
				}
				cancel()
			}
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (p *Proxy) startRedisListeners() {
	p.registry.mu.RLock()
	defer p.registry.mu.RUnlock()

	for _, route := range p.registry.routesByID {
		if route.DatabaseType == "redis" && route.RedisPort > 0 {
			if err := p.redisManager.StartListener(route.RedisPort, route); err != nil {
				logger.Error("Failed to start Redis listener for %s: %v", route.DatabaseID, err)
			}
		}
	}
}
