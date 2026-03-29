package proxy

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// RedisPortManager manages Redis port-per-instance listeners
type RedisPortManager struct {
	mu        sync.RWMutex
	listeners map[int]net.Listener // port -> listener
	proxy     *Proxy
	stopCh    map[int]chan struct{} // port -> stop channel
}

// NewRedisPortManager creates a new Redis port manager
func NewRedisPortManager(proxy *Proxy) *RedisPortManager {
	return &RedisPortManager{
		listeners: make(map[int]net.Listener),
		proxy:     proxy,
		stopCh:    make(map[int]chan struct{}),
	}
}

// StartListener starts a TCP listener for a Redis instance on the given port
func (m *RedisPortManager) StartListener(port int, route *Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.listeners[port]; exists {
		return fmt.Errorf("listener already exists on port %d", port)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	m.listeners[port] = ln
	m.proxy.activeListeners.Add(1)
	stopCh := make(chan struct{})
	m.stopCh[port] = stopCh

	go m.acceptLoop(ln, port, stopCh)

	logger.Info("Redis listener started on port %d for %s", port, route.DatabaseID)
	return nil
}

// StopListener stops the Redis listener on the given port
func (m *RedisPortManager) StopListener(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, ok := m.stopCh[port]; ok {
		close(ch)
		delete(m.stopCh, port)
	}

	if ln, ok := m.listeners[port]; ok {
		ln.Close()
		delete(m.listeners, port)
		m.proxy.activeListeners.Add(-1)
		logger.Info("Redis listener stopped on port %d", port)
	}
}

// StopAll stops all Redis listeners
func (m *RedisPortManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for port, ch := range m.stopCh {
		close(ch)
		delete(m.stopCh, port)
	}

	for port, ln := range m.listeners {
		ln.Close()
		delete(m.listeners, port)
		m.proxy.activeListeners.Add(-1)
	}
}

func (m *RedisPortManager) acceptLoop(ln net.Listener, port int, stopCh chan struct{}) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				logger.Warn("Temporary Redis accept error on port %d: %v", port, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			select {
			case <-stopCh:
				return
			default:
				logger.Error("Redis accept loop stopped on port %d: %v", port, err)
				if m.proxy.activeListeners.Add(-1) <= 0 {
					m.proxy.activeListeners.Store(0)
					m.proxy.running.Store(false)
				}
				return
			}
		}

		m.proxy.activeConns.Add(1)
		go func() {
			defer m.proxy.activeConns.Done()
			m.handleRedisConn(conn, port)
		}()
	}
}

func (m *RedisPortManager) handleRedisConn(clientConn net.Conn, port int) {
	defer clientConn.Close()

	route, ok := m.proxy.registry.LookupByRedisPort(port)
	if !ok {
		logger.Debug("No route for Redis port %d", port)
		return
	}

	// Handle sleeping/stopped databases
	var backendAddr string
	if route.Stopped {
		if route.DBStatus == 5 { // STOPPED - no auto-wake
			logger.Debug("Redis route %s is stopped", route.DatabaseID)
			return
		}
		// SLEEPING (12) - wake on connect
		addr, err := m.proxy.wakeAndConnect(route)
		if err != nil {
			logger.Error("Failed to wake Redis database %s: %v", route.DatabaseID, err)
			return
		}
		backendAddr = addr
	} else {
		if route.ContainerIP == "" {
			logger.Debug("Redis route %s has no IP", route.DatabaseID)
			return
		}
		backendAddr = net.JoinHostPort(route.ContainerIP, fmt.Sprintf("%d", route.InternalPort))
	}

	m.proxy.registry.TouchRoute(route.DatabaseID)

	backendConn, err := net.DialTimeout("tcp", backendAddr, dialTimeout)
	if err != nil {
		logger.Error("Failed to connect to Redis backend %s: %v", backendAddr, err)
		return
	}
	defer backendConn.Close()

	m.proxy.bidirectionalCopy(clientConn, backendConn)
}
