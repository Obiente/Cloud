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

	listenerMu    sync.Mutex
	pgListener    net.Listener
	mysqlListener net.Listener
	mongoListener net.Listener

	redisManager *RedisPortManager

	activeConns     sync.WaitGroup
	activeListeners atomic.Int32
	running         atomic.Bool
	stopCh          chan struct{}
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
	p.listenerMu.Lock()
	p.stopCh = make(chan struct{})
	p.listenerMu.Unlock()
	p.activeListeners.Store(0)
	p.running.Store(false)

	var err error

	// PostgreSQL listener on 5432
	p.pgListener, err = net.Listen("tcp", ":5432")
	if err != nil {
		logger.Warn("Failed to start PostgreSQL proxy listener: %v", err)
	} else {
		p.activeListeners.Add(1)
		go p.acceptLoop(p.pgListener, "postgresql")
		logger.Info("PostgreSQL proxy listening on :5432")
	}

	// MySQL listener on 3306
	p.mysqlListener, err = net.Listen("tcp", ":3306")
	if err != nil {
		logger.Warn("Failed to start MySQL proxy listener: %v", err)
	} else {
		p.activeListeners.Add(1)
		go p.acceptLoop(p.mysqlListener, "mysql")
		logger.Info("MySQL proxy listening on :3306")
	}

	// MongoDB listener on 27017
	p.mongoListener, err = net.Listen("tcp", ":27017")
	if err != nil {
		logger.Warn("Failed to start MongoDB proxy listener: %v", err)
	} else {
		p.activeListeners.Add(1)
		go p.acceptLoop(p.mongoListener, "mongodb")
		logger.Info("MongoDB proxy listening on :27017")
	}

	// Start Redis listeners for existing routes
	p.startRedisListeners()

	if p.activeListeners.Load() == 0 {
		return fmt.Errorf("failed to start any proxy listeners")
	}

	p.running.Store(true)

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

	p.listenerMu.Lock()
	if p.pgListener != nil {
		p.pgListener.Close()
		p.pgListener = nil
	}
	if p.mysqlListener != nil {
		p.mysqlListener.Close()
		p.mysqlListener = nil
	}
	if p.mongoListener != nil {
		p.mongoListener.Close()
		p.mongoListener = nil
	}
	p.listenerMu.Unlock()

	p.redisManager.StopAll()
	p.registry.StopIPRefresh()
	p.activeListeners.Store(0)

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
	return p.running.Load() && p.activeListeners.Load() > 0
}

func (p *Proxy) acceptLoop(ln net.Listener, protocol string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				logger.Warn("Temporary accept error on %s: %v", protocol, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			select {
			case <-p.stopCh:
				return
			default:
				logger.Error("Accept loop stopped on %s: %v", protocol, err)
				p.handleListenerFailure(protocol, ln)
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
		_, _ = io.Copy(backend, client)
		if tc, ok := backend.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(client, backend)
		if tc, ok := client.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	<-done
	_ = client.Close()
	_ = backend.Close()
	<-done
}

func (p *Proxy) handleListenerFailure(protocol string, failedListener net.Listener) {
	p.listenerMu.Lock()
	defer p.listenerMu.Unlock()

	switch protocol {
	case "postgresql":
		if p.pgListener == failedListener {
			p.pgListener = nil
			p.activeListeners.Add(-1)
		}
	case "mysql":
		if p.mysqlListener == failedListener {
			p.mysqlListener = nil
			p.activeListeners.Add(-1)
		}
	case "mongodb":
		if p.mongoListener == failedListener {
			p.mongoListener = nil
			p.activeListeners.Add(-1)
		}
	}

	if p.activeListeners.Load() <= 0 {
		p.activeListeners.Store(0)
		p.running.Store(false)
	}
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
