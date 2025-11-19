package orchestrator

// OrchestratorServiceInterface defines the interface for orchestrator service
// that other services can use. The actual implementation is in orchestrator-service.
type OrchestratorServiceInterface interface {
	GetDeploymentManager() *DeploymentManager
	GetServiceRegistry() interface{} // *registry.ServiceRegistry - using interface{} to avoid circular dependency
	GetHealthChecker() interface{}   // *registry.HealthChecker
	GetMetricsStreamer() *MetricsStreamer
	GetGameServerManager() *GameServerManager
}
