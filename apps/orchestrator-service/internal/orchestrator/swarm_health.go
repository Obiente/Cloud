package orchestrator

import (
	"context"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

var (
	swarmManager           = promauto.NewGauge(prometheus.GaugeOpts{Name: "obiente_swarm_manager", Help: "Whether this daemon can inspect Swarm services."})
	swarmCollectionSuccess = promauto.NewGauge(prometheus.GaugeOpts{Name: "obiente_swarm_collection_success", Help: "Whether the latest Swarm inventory read succeeded."})
	swarmUpdatePaused      = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "obiente_swarm_service_update_paused", Help: "Whether a service update or rollback is paused."}, []string{"swarm_service"})
	swarmMissingReplicas   = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "obiente_swarm_service_missing_replicas", Help: "Desired service tasks that are not running."}, []string{"swarm_service"})
)

func (os *OrchestratorService) monitorSwarmHealth() {
	cli, ok := os.deploymentManager.GetDockerClient().(client.APIClient)
	if !ok {
		logger.Warn("Swarm monitoring client unavailable")
		return
	}
	collect := func() {
		ctx, cancel := context.WithTimeout(os.ctx, 10*time.Second)
		defer cancel()
		info, err := cli.Info(ctx, client.InfoOptions{})
		if err != nil {
			swarmCollectionSuccess.Set(0)
			return
		}
		if !info.Info.Swarm.ControlAvailable {
			swarmManager.Set(0)
			swarmUpdatePaused.Reset()
			swarmMissingReplicas.Reset()
			return
		}
		swarmManager.Set(1)
		services, err := cli.ServiceList(ctx, client.ServiceListOptions{Status: true})
		if err != nil {
			swarmCollectionSuccess.Set(0)
			return
		}
		swarmUpdatePaused.Reset()
		swarmMissingReplicas.Reset()
		for _, service := range services.Items {
			paused, missing := swarmServiceHealth(service)
			swarmUpdatePaused.WithLabelValues(service.Spec.Name).Set(paused)
			swarmMissingReplicas.WithLabelValues(service.Spec.Name).Set(missing)
		}
		swarmCollectionSuccess.Set(1)
	}
	collect()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-os.ctx.Done():
			return
		case <-ticker.C:
			collect()
		}
	}
}

func swarmServiceHealth(service swarm.Service) (paused, missing float64) {
	if service.UpdateStatus != nil && (service.UpdateStatus.State == swarm.UpdateStatePaused || service.UpdateStatus.State == swarm.UpdateStateRollbackPaused) {
		paused = 1
	}
	if service.ServiceStatus != nil && service.Spec.Mode.ReplicatedJob == nil && service.Spec.Mode.GlobalJob == nil && service.ServiceStatus.DesiredTasks > service.ServiceStatus.RunningTasks {
		missing = float64(service.ServiceStatus.DesiredTasks - service.ServiceStatus.RunningTasks)
	}
	return
}
