package proxy

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Route represents a database routing entry
type Route struct {
	DatabaseID       string
	DatabaseType     string // postgresql, mysql, mariadb, mongodb, redis
	ContainerID      string
	ContainerIP      string
	InternalPort     int
	RedisPort        int // Allocated external port for Redis instances
	Username         string
	Password         string // Encrypted password
	OrganizationID   string
	Stopped          bool      // Container is stopped (sleeping or fully stopped)
	DBStatus         int32     // Database status code (5=STOPPED, 12=SLEEPING)
	AutoSleepSeconds int32     // Auto-sleep after inactivity (0 = disabled)
	LastConnectionAt time.Time // Last time a client connected
}

// WakeFunc starts a sleeping database container and returns the new container IP
type WakeFunc func(ctx context.Context, route *Route) (string, error)

// SleepFunc stops a database container for sleeping
type SleepFunc func(ctx context.Context, route *Route) error

// RouteRegistry is a thread-safe in-memory route registry
type RouteRegistry struct {
	mu           sync.RWMutex
	routes       map[string]*Route // keyed by database name (db-{id})
	routesByID   map[string]*Route // keyed by database ID
	dockerClient *docker.Client
	stopRefresh  chan struct{}

	// Wake/sleep callbacks (set by service layer)
	OnWake  WakeFunc
	OnSleep SleepFunc

	// Redis port allocator
	redisMu        sync.Mutex
	redisPortStart int
	redisPortEnd   int
	usedRedisPorts map[int]string // port -> database ID
}

// NewRouteRegistry creates a new route registry
func NewRouteRegistry(dockerClient *docker.Client) *RouteRegistry {
	return &RouteRegistry{
		routes:         make(map[string]*Route),
		routesByID:     make(map[string]*Route),
		dockerClient:   dockerClient,
		stopRefresh:    make(chan struct{}),
		redisPortStart: 16379,
		redisPortEnd:   16478,
		usedRedisPorts: make(map[int]string),
	}
}

// Register adds a route to the registry
func (r *RouteRegistry) Register(route *Route) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[route.DatabaseID] = route
	r.routesByID[route.DatabaseID] = route

	logger.Info("Route registered: %s -> %s:%d (type: %s)", route.DatabaseID, route.ContainerIP, route.InternalPort, route.DatabaseType)
}

// Unregister removes a route from the registry
func (r *RouteRegistry) Unregister(databaseID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	route, ok := r.routesByID[databaseID]
	if !ok {
		return
	}

	// Release Redis port if applicable
	if route.RedisPort > 0 {
		r.redisMu.Lock()
		delete(r.usedRedisPorts, route.RedisPort)
		r.redisMu.Unlock()
	}

	delete(r.routes, route.DatabaseID)
	delete(r.routesByID, databaseID)

	logger.Info("Route unregistered: %s", route.DatabaseID)
}

// Lookup finds a route by database name (the routing key)
func (r *RouteRegistry) Lookup(dbName string) (*Route, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, ok := r.routes[dbName]
	return route, ok
}

// LookupByID finds a route by database ID
func (r *RouteRegistry) LookupByID(databaseID string) (*Route, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, ok := r.routesByID[databaseID]
	return route, ok
}

// LookupByRedisPort finds a route by its allocated Redis port
func (r *RouteRegistry) LookupByRedisPort(port int) (*Route, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.redisMu.Lock()
	dbID, ok := r.usedRedisPorts[port]
	r.redisMu.Unlock()

	if !ok {
		return nil, false
	}

	route, ok := r.routesByID[dbID]
	return route, ok
}

// AllocateRedisPort allocates a port for a Redis instance
func (r *RouteRegistry) AllocateRedisPort(databaseID string) (int, error) {
	r.redisMu.Lock()
	defer r.redisMu.Unlock()

	port, err := r.allocateRedisPortLocked(databaseID, r.usedRedisPorts)
	if err != nil {
		return 0, err
	}
	r.usedRedisPorts[port] = databaseID
	return port, nil
}

// ReleaseRedisPort releases an allocated Redis port
func (r *RouteRegistry) ReleaseRedisPort(port int) {
	r.redisMu.Lock()
	defer r.redisMu.Unlock()
	delete(r.usedRedisPorts, port)
}

// LoadFromDatabase populates routes from the database_instances table
func (r *RouteRegistry) LoadFromDatabase(ctx context.Context) error {
	var instances []database.DatabaseInstance
	if err := database.DB.WithContext(ctx).
		Where("status IN (?, ?, ?) AND deleted_at IS NULL", 3, 5, 12). // RUNNING, STOPPED, SLEEPING
		Find(&instances).Error; err != nil {
		return fmt.Errorf("failed to load database instances: %w", err)
	}

	var connections []database.DatabaseConnection
	if err := database.DB.WithContext(ctx).Find(&connections).Error; err != nil {
		return fmt.Errorf("failed to load database connections: %w", err)
	}

	connMap := make(map[string]*database.DatabaseConnection)
	for i := range connections {
		connMap[connections[i].DatabaseID] = &connections[i]
	}

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].ID < instances[j].ID
	})

	r.mu.RLock()
	existingRoutes := make(map[string]*Route, len(r.routesByID))
	for id, route := range r.routesByID {
		existingRoutes[id] = route
	}
	r.mu.RUnlock()

	newRoutes := make(map[string]*Route, len(instances))
	newRoutesByID := make(map[string]*Route, len(instances))
	newUsedRedisPorts := make(map[int]string)

	for _, inst := range instances {
		dbType := databaseTypeIntToString(inst.Type)
		internalPort := standardPort(dbType)
		lastConnectionAt := time.Now()
		if existing, ok := existingRoutes[inst.ID]; ok && !existing.LastConnectionAt.IsZero() {
			lastConnectionAt = existing.LastConnectionAt
		}

		route := &Route{
			DatabaseID:       inst.ID,
			DatabaseType:     dbType,
			InternalPort:     internalPort,
			OrganizationID:   inst.OrganizationID,
			DBStatus:         inst.Status,
			AutoSleepSeconds: inst.AutoSleepSeconds,
			LastConnectionAt: lastConnectionAt,
		}

		// Mark sleeping/stopped routes
		if inst.Status == 5 || inst.Status == 12 { // STOPPED or SLEEPING
			route.Stopped = true
		}

		if inst.InstanceID != nil {
			route.ContainerID = *inst.InstanceID
		}

		// Load connection credentials
		if conn, ok := connMap[inst.ID]; ok {
			route.Username = conn.Username
			route.Password = conn.Password
		}

		// Only resolve IPs for running databases
		if route.Stopped {
			if dbType == "redis" {
				port, err := r.allocateRedisPortLocked(inst.ID, newUsedRedisPorts)
				if err != nil {
					logger.Error("Failed to allocate Redis port for %s: %v", inst.ID, err)
				} else {
					route.RedisPort = port
					newUsedRedisPorts[port] = inst.ID
				}
			}
			newRoutes[route.DatabaseID] = route
			newRoutesByID[route.DatabaseID] = route
			continue
		}

		// Ensure container is on the correct network, then resolve IP
		if route.ContainerID != "" && r.dockerClient != nil {
			r.reconcileContainerNetwork(ctx, route.ContainerID)

			if ip, err := r.resolveContainerIP(ctx, route.ContainerID); err == nil {
				route.ContainerIP = ip
			} else {
				logger.Warn("Failed to resolve IP for container %s: %v", route.ContainerID, err)
				// Try by container name
				containerName := fmt.Sprintf("obiente-%s", inst.ID)
				if ip, err := r.resolveContainerIPByName(ctx, containerName); err == nil {
					route.ContainerIP = ip
				}
			}
		}

		// Allocate Redis port if needed
		if dbType == "redis" {
			port, err := r.allocateRedisPortLocked(inst.ID, newUsedRedisPorts)
			if err != nil {
				logger.Error("Failed to allocate Redis port for %s: %v", inst.ID, err)
				continue
			}
			route.RedisPort = port
			newUsedRedisPorts[port] = inst.ID
		}

		newRoutes[route.DatabaseID] = route
		newRoutesByID[route.DatabaseID] = route
	}

	r.mu.Lock()
	r.routes = newRoutes
	r.routesByID = newRoutesByID
	r.mu.Unlock()

	r.redisMu.Lock()
	r.usedRedisPorts = newUsedRedisPorts
	r.redisMu.Unlock()

	logger.Info("Loaded %d routes from database", len(instances))
	return nil
}

// StartIPRefresh starts a background goroutine that refreshes container IPs
func (r *RouteRegistry) StartIPRefresh(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.refreshIPs(ctx)
			case <-r.stopRefresh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// StopIPRefresh stops the IP refresh goroutine
func (r *RouteRegistry) StopIPRefresh() {
	close(r.stopRefresh)
}

func (r *RouteRegistry) refreshIPs(ctx context.Context) {
	r.mu.RLock()
	routes := make([]*Route, 0, len(r.routesByID))
	for _, route := range r.routesByID {
		routes = append(routes, route)
	}
	r.mu.RUnlock()

	for _, route := range routes {
		if route.ContainerID == "" {
			continue
		}

		ip, err := r.resolveContainerIP(ctx, route.ContainerID)
		if err != nil {
			// Try by container name
			containerName := fmt.Sprintf("obiente-%s", route.DatabaseID)
			ip, err = r.resolveContainerIPByName(ctx, containerName)
			if err != nil {
				continue
			}
		}

		if ip != route.ContainerIP {
			r.mu.Lock()
			route.ContainerIP = ip
			r.mu.Unlock()
			logger.Debug("Updated IP for %s: %s", route.DatabaseID, ip)
		}
	}
}

// reconcileContainerNetwork ensures a container is attached to obiente-databases.
func (r *RouteRegistry) reconcileContainerNetwork(ctx context.Context, containerID string) {
	inspect, err := r.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return
	}

	if inspect.NetworkSettings != nil {
		if _, ok := inspect.NetworkSettings.Networks["obiente-databases"]; !ok {
			logger.Info("Reconcile: connecting container %s to obiente-databases network", containerID)
			if err := r.dockerClient.NetworkConnect(ctx, "obiente-databases", containerID); err != nil {
				logger.Warn("Reconcile: failed to connect container %s to obiente-databases: %v", containerID, err)
			}
		}
	}
}

func (r *RouteRegistry) resolveContainerIP(ctx context.Context, containerID string) (string, error) {
	if r.dockerClient == nil {
		return "", fmt.Errorf("docker client not available")
	}

	inspect, err := r.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	// Look for IP on obiente-databases network first
	if netSettings := inspect.NetworkSettings; netSettings != nil {
		if nw, ok := netSettings.Networks["obiente-databases"]; ok && nw.IPAddress.IsValid() {
			return nw.IPAddress.String(), nil
		}
		// Fallback to any network
		for _, nw := range netSettings.Networks {
			if nw.IPAddress.IsValid() {
				return nw.IPAddress.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no IP address found for container %s", containerID)
}

func (r *RouteRegistry) resolveContainerIPByName(ctx context.Context, containerName string) (string, error) {
	// Container name on overlay network acts as DNS name
	// Return the container name so Go's dialer resolves it via Docker DNS
	return containerName, nil
}

// MarkStopped marks a route as stopped (sleeping or fully stopped)
func (r *RouteRegistry) MarkStopped(databaseID string, status int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	route, ok := r.routesByID[databaseID]
	if !ok {
		return
	}
	route.Stopped = true
	route.DBStatus = status
	route.ContainerIP = ""
	logger.Info("Route marked stopped: %s (status: %d)", route.DatabaseID, status)
}

// MarkRunning marks a route as running with a new IP
func (r *RouteRegistry) MarkRunning(databaseID string, ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	route, ok := r.routesByID[databaseID]
	if !ok {
		return
	}
	route.Stopped = false
	route.DBStatus = 3 // RUNNING
	route.ContainerIP = ip
	logger.Info("Route marked running: %s (ip: %s)", route.DatabaseID, ip)
}

// TouchRoute updates the last connection time for a route
func (r *RouteRegistry) TouchRoute(databaseID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if route, ok := r.routesByID[databaseID]; ok {
		route.LastConnectionAt = time.Now()
	}
}

// GetSleepableRoutes returns routes that are eligible for auto-sleep
func (r *RouteRegistry) GetSleepableRoutes() []*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var routes []*Route
	now := time.Now()
	for _, route := range r.routesByID {
		if route.AutoSleepSeconds > 0 && !route.Stopped && !route.LastConnectionAt.IsZero() {
			if now.Sub(route.LastConnectionAt) > time.Duration(route.AutoSleepSeconds)*time.Second {
				routes = append(routes, route)
			}
		}
	}
	return routes
}

// RouteCount returns the number of registered routes
func (r *RouteRegistry) RouteCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.routes)
}

func (r *RouteRegistry) RedisRoutesSnapshot() map[int]*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make(map[int]*Route)
	for _, route := range r.routesByID {
		if route.DatabaseType == "redis" && route.RedisPort > 0 {
			routes[route.RedisPort] = route
		}
	}
	return routes
}

func (r *RouteRegistry) allocateRedisPortLocked(databaseID string, usedPorts map[int]string) (int, error) {
	span := r.redisPortEnd - r.redisPortStart + 1
	if span <= 0 {
		return 0, fmt.Errorf("invalid Redis port range %d-%d", r.redisPortStart, r.redisPortEnd)
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(databaseID))
	offset := int(hasher.Sum32() % uint32(span))

	for probe := 0; probe < span; probe++ {
		port := r.redisPortStart + ((offset + probe) % span)
		if assignedID, used := usedPorts[port]; !used || assignedID == databaseID {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available Redis ports in range %d-%d", r.redisPortStart, r.redisPortEnd)
}

func databaseTypeIntToString(t int32) string {
	switch t {
	case 1:
		return "postgresql"
	case 2:
		return "mysql"
	case 3:
		return "mongodb"
	case 4:
		return "redis"
	case 5:
		return "mariadb"
	default:
		return "unknown"
	}
}

func standardPort(dbType string) int {
	switch dbType {
	case "postgresql":
		return 5432
	case "mysql", "mariadb":
		return 3306
	case "mongodb":
		return 27017
	case "redis":
		return 6379
	default:
		return 0
	}
}
