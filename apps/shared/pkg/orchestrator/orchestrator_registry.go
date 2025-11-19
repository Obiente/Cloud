package orchestrator

import (
	"sync"
)

var (
	globalOrchestratorService OrchestratorServiceInterface
	orchestratorServiceMutex   sync.RWMutex
)

// SetGlobalOrchestratorService sets the global orchestrator service instance
func SetGlobalOrchestratorService(service OrchestratorServiceInterface) {
	orchestratorServiceMutex.Lock()
	defer orchestratorServiceMutex.Unlock()
	globalOrchestratorService = service
}

// GetGlobalOrchestratorService returns the global orchestrator service instance
func GetGlobalOrchestratorService() OrchestratorServiceInterface {
	orchestratorServiceMutex.RLock()
	defer orchestratorServiceMutex.RUnlock()
	return globalOrchestratorService
}

