package orchestrator

import (
	"sync"
)

var (
	globalOrchestratorService *OrchestratorService
	orchestratorServiceMutex   sync.RWMutex
)

// SetGlobalOrchestratorService sets the global orchestrator service instance
func SetGlobalOrchestratorService(service *OrchestratorService) {
	orchestratorServiceMutex.Lock()
	defer orchestratorServiceMutex.Unlock()
	globalOrchestratorService = service
}

// GetGlobalOrchestratorService returns the global orchestrator service instance
func GetGlobalOrchestratorService() *OrchestratorService {
	orchestratorServiceMutex.RLock()
	defer orchestratorServiceMutex.RUnlock()
	return globalOrchestratorService
}

