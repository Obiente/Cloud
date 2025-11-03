package deployments

import (
	"context"
	"fmt"
	"log"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"
	"api/internal/quota"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TriggerDeployment triggers a rebuild and redeployment
func (s *Service) TriggerDeployment(ctx context.Context, req *connect.Request[deploymentsv1.TriggerDeploymentRequest]) (*connect.Response[deploymentsv1.TriggerDeploymentResponse], error) {
	// Check if user has deploy permission for this deployment
	deploymentID := req.Msg.GetDeploymentId()
	if err := s.checkDeploymentPermission(ctx, deploymentID, "deploy"); err != nil {
		return nil, err
	}

	// Get deployment
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %w", err))
	}

	// Update deployment status to deploying
	if err := s.repo.UpdateStatus(ctx, deploymentID, int32(deploymentsv1.DeploymentStatus_DEPLOYING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deployment: %w", err))
	}

	// Start async rebuild with log streaming
	go func() {
		buildCtx := context.Background()

		// Get or create build log streamer
		streamer := GetBuildLogStreamer(deploymentID)
		defer streamer.Close()

		// Get build strategy - handle UNSPECIFIED by auto-detecting
		buildStrategy := deploymentsv1.BuildStrategy(dbDeployment.BuildStrategy)
		if buildStrategy == deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED || buildStrategy == 0 {
			// Auto-detect build strategy if not set
			if dbDeployment.RepositoryURL != nil && *dbDeployment.RepositoryURL != "" {
				buildDir, err := ensureBuildDir(deploymentID + "-detect")
				if err == nil {
					// Get GitHub token if integration ID is set
					githubToken := ""
					if dbDeployment.GitHubIntegrationID != nil && *dbDeployment.GitHubIntegrationID != "" {
						var integration database.GitHubIntegration
						if err := database.DB.Where("id = ?", *dbDeployment.GitHubIntegrationID).First(&integration).Error; err == nil {
							githubToken = integration.Token
						}
					}
					if err := cloneRepository(buildCtx, *dbDeployment.RepositoryURL, dbDeployment.Branch, buildDir, githubToken); err == nil {
						if detected, _ := s.buildRegistry.AutoDetect(buildCtx, buildDir); detected != deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED {
							buildStrategy = detected
							// Update deployment with detected strategy
							dbDeployment.BuildStrategy = int32(buildStrategy)
							s.repo.Update(buildCtx, dbDeployment)
						}
					}
				}
			}

			// Fallback to NIXPACKS if still unspecified
			if buildStrategy == deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED || buildStrategy == 0 {
				buildStrategy = deploymentsv1.BuildStrategy_NIXPACKS
				dbDeployment.BuildStrategy = int32(buildStrategy)
				s.repo.Update(buildCtx, dbDeployment)
			}
		}

		strategy, err := s.buildRegistry.Get(buildStrategy)
		if err != nil {
			log.Printf("[TriggerDeployment] Invalid build strategy %v: %v", buildStrategy, err)
			streamer.WriteStderr([]byte(fmt.Sprintf("Error: Invalid build strategy: %v\n", err)))
			_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
			return
		}

		// Prepare build config with log writers
		repoURL := ""
		if dbDeployment.RepositoryURL != nil {
			repoURL = *dbDeployment.RepositoryURL
		}
		buildCmd := ""
		if dbDeployment.BuildCommand != nil {
			buildCmd = *dbDeployment.BuildCommand
		}
		installCmd := ""
		if dbDeployment.InstallCommand != nil {
			installCmd = *dbDeployment.InstallCommand
		}

		// Set defaults for optional fields
		port := 8080
		if dbDeployment.Port != nil {
			port = int(*dbDeployment.Port)
		}
		memoryBytes := int64(512 * 1024 * 1024) // 512MB default
		if dbDeployment.MemoryBytes != nil {
			memoryBytes = *dbDeployment.MemoryBytes
		}
		cpuShares := int64(256) // Default CPU shares
		if dbDeployment.CPUShares != nil {
			cpuShares = *dbDeployment.CPUShares
		}

		dockerfilePath := ""
		if dbDeployment.DockerfilePath != nil {
			dockerfilePath = *dbDeployment.DockerfilePath
		}
		composeFilePath := ""
		if dbDeployment.ComposeFilePath != nil {
			composeFilePath = *dbDeployment.ComposeFilePath
		}

		// Get GitHub token if integration ID is set
		githubToken := ""
		if dbDeployment.GitHubIntegrationID != nil && *dbDeployment.GitHubIntegrationID != "" {
			var integration database.GitHubIntegration
			if err := database.DB.Where("id = ?", *dbDeployment.GitHubIntegrationID).First(&integration).Error; err == nil {
				githubToken = integration.Token
			}
		}

		buildConfig := &BuildConfig{
			DeploymentID:    deploymentID,
			RepositoryURL:   repoURL,
			Branch:          dbDeployment.Branch,
			GitHubToken:     githubToken,
			BuildCommand:    buildCmd,
			InstallCommand:  installCmd,
			DockerfilePath:  dockerfilePath,
			ComposeFilePath: composeFilePath,
			EnvVars:         parseEnvVars(dbDeployment.EnvVars),
			Port:            port,
			MemoryBytes:     memoryBytes,
			CPUShares:       cpuShares,
			LogWriter:       streamer,                  // Stream stdout
			LogWriterErr:    NewStderrWriter(streamer), // Stream stderr
		}

		// Write initial build message
		streamer.Write([]byte(fmt.Sprintf("🚀 Starting deployment rebuild for %s...\n", deploymentID)))
		streamer.Write([]byte(fmt.Sprintf("📦 Using build strategy: %s\n", strategy.Name())))

		// Handle compose-based deployments
		if dbDeployment.ComposeYaml != "" {
			streamer.Write([]byte("🐳 Deploying Docker Compose configuration...\n"))
			if err := s.manager.DeployComposeFile(buildCtx, deploymentID, dbDeployment.ComposeYaml); err != nil {
				log.Printf("[TriggerDeployment] Compose deployment failed: %v", err)
				streamer.WriteStderr([]byte(fmt.Sprintf("❌ Deployment failed: %v\n", err)))
				_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
				return
			}
			// Containers are automatically registered by DeployComposeFile
		} else {
			// Build using strategy
			streamer.Write([]byte(fmt.Sprintf("🔨 Building deployment with %s...\n", strategy.Name())))
			result, err := strategy.Build(buildCtx, dbDeployment, buildConfig)
			if err != nil || !result.Success {
				log.Printf("[TriggerDeployment] Build failed: %v", err)
				if result != nil && result.Error != nil {
					err = result.Error
				}
				streamer.WriteStderr([]byte(fmt.Sprintf("❌ Build failed: %v\n", err)))
				_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
				return
			}

			streamer.Write([]byte("✅ Build completed successfully\n"))
			streamer.Write([]byte("🚀 Deploying to orchestrator...\n"))

			// Deploy using build result
			if err := deployResultToOrchestrator(buildCtx, s.manager, dbDeployment, result); err != nil {
				log.Printf("[TriggerDeployment] Deployment failed: %v", err)
				streamer.WriteStderr([]byte(fmt.Sprintf("❌ Deployment failed: %v\n", err)))
				_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
				return
			}

			// Update deployment with build results
			if result.ImageName != "" {
				dbDeployment.Image = &result.ImageName
			}
			if result.ComposeYaml != "" {
				dbDeployment.ComposeYaml = result.ComposeYaml
			}
			if result.Port > 0 {
				port := int32(result.Port)
				dbDeployment.Port = &port
			}
			s.repo.Update(buildCtx, dbDeployment)
		}

		// Verify containers are running
		streamer.Write([]byte("🔍 Verifying containers are running...\n"))
		if err := s.verifyContainersRunning(buildCtx, deploymentID); err != nil {
			log.Printf("[TriggerDeployment] WARNING: Containers not running: %v", err)
			streamer.WriteStderr([]byte(fmt.Sprintf("⚠️  Warning: %v\n", err)))
			_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
		} else {
			streamer.Write([]byte("✅ Deployment completed successfully!\n"))
			_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_RUNNING))
		}
	}()

	res := connect.NewResponse(&deploymentsv1.TriggerDeploymentResponse{
		DeploymentId: req.Msg.GetDeploymentId(),
		Status:       "DEPLOYING",
	})
	return res, nil
}

// StreamDeploymentStatus streams deployment status updates
func (s *Service) StreamDeploymentStatus(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentStatusRequest], stream *connect.ServerStream[deploymentsv1.DeploymentStatusUpdate]) error {
	updates := []deploymentsv1.DeploymentStatusUpdate{
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_DEPLOYING,
			HealthStatus: "starting",
			Message:      proto.String("Build started"),
			Timestamp:    timestamppb.Now(),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_DEPLOYING,
			HealthStatus: "verifying",
			Message:      proto.String("Running smoke tests"),
			Timestamp:    timestamppb.New(time.Now().Add(5 * time.Second)),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_RUNNING,
			HealthStatus: "healthy",
			Message:      proto.String("Deployment complete"),
			Timestamp:    timestamppb.New(time.Now().Add(10 * time.Second)),
		},
	}

	for i := range updates {
		if err := stream.Send(&updates[i]); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

// StartDeployment starts a stopped deployment
func (s *Service) StartDeployment(ctx context.Context, req *connect.Request[deploymentsv1.StartDeploymentRequest]) (*connect.Response[deploymentsv1.StartDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.start", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Check if this is a compose-based deployment
	if dbDep.ComposeYaml != "" {
		// Deploy using Docker Compose
		if s.manager != nil {
			if err := s.manager.DeployComposeFile(ctx, deploymentID, dbDep.ComposeYaml); err != nil {
				log.Printf("[StartDeployment] Failed to deploy compose file for deployment %s: %v", deploymentID, err)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to deploy compose file: %w", err))
			}
			log.Printf("[StartDeployment] Successfully deployed compose file for deployment %s", deploymentID)

			// Verify containers are actually running before setting status
			if err := s.verifyContainersRunning(ctx, deploymentID); err != nil {
				log.Printf("[StartDeployment] WARNING: Containers not running for deployment %s: %v", deploymentID, err)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deployment started but no containers are running: %w", err))
			}
		} else {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("compose deployment requires orchestrator"))
		}
	} else {
		// Regular container-based deployment
		// Check if deployment has containers created
		locations, err := database.GetDeploymentLocations(deploymentID)
		if err != nil || len(locations) == 0 {
			// No containers exist - need to create them first
			// This happens if CreateDeployment failed or was never called
			// We should trigger container creation
			if s.manager != nil {
				// Get deployment config from database
				image := ""
				if dbDep.Image != nil {
					image = *dbDep.Image
				}
				port := 8080
				if dbDep.Port != nil {
					port = int(*dbDep.Port)
				}
				memory := int64(512 * 1024 * 1024) // Default 512MB
				if dbDep.MemoryBytes != nil {
					memory = *dbDep.MemoryBytes
				}
				cpuShares := int64(1024) // Default
				if dbDep.CPUShares != nil {
					cpuShares = *dbDep.CPUShares
				}
				replicas := 1 // Default
				if dbDep.Replicas != nil {
					replicas = int(*dbDep.Replicas)
				}

				// Recreate containers using deployment config
				cfg := &orchestrator.DeploymentConfig{
					DeploymentID: deploymentID,
					Image:        image,
					Domain:       dbDep.Domain,
					Port:         port,
					EnvVars:      parseEnvVars(dbDep.EnvVars),
					Labels:       map[string]string{},
					Memory:       memory,
					CPUShares:    cpuShares,
					Replicas:     replicas,
				}
				if err := s.manager.CreateDeployment(ctx, cfg); err != nil {
					log.Printf("[StartDeployment] Failed to create containers for deployment %s: %v", deploymentID, err)
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create containers: %w", err))
				}
				log.Printf("[StartDeployment] Successfully created containers for deployment %s", deploymentID)

				// Verify containers are actually running before setting status
				if err := s.verifyContainersRunning(ctx, deploymentID); err != nil {
					log.Printf("[StartDeployment] WARNING: Containers not running for deployment %s: %v", deploymentID, err)
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("containers created but not running: %w", err))
				}
			} else {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment has no containers and orchestrator is not available"))
			}
		} else {
			// Containers exist - start them
			if s.manager != nil {
				if err := s.manager.StartDeployment(ctx, deploymentID); err != nil {
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start containers: %w", err))
				}
			}

			// Verify containers are actually running after starting
			if err := s.verifyContainersRunning(ctx, deploymentID); err != nil {
				log.Printf("[StartDeployment] WARNING: Containers not running for deployment %s after start: %v", deploymentID, err)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start containers: %w", err))
			}
		}
	}

	// Update status to running
	if err := s.repo.UpdateStatus(ctx, deploymentID, int32(deploymentsv1.DeploymentStatus_RUNNING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment status: %w", err))
	}

	dbDep, _ = s.repo.GetByID(ctx, deploymentID)
	res := connect.NewResponse(&deploymentsv1.StartDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

// StopDeployment stops a running deployment
func (s *Service) StopDeployment(ctx context.Context, req *connect.Request[deploymentsv1.StopDeploymentRequest]) (*connect.Response[deploymentsv1.StopDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.stop", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Check if this is a compose-based deployment
	if dbDep.ComposeYaml != "" && s.manager != nil {
		if err := s.manager.StopComposeDeployment(ctx, deploymentID); err != nil {
			log.Printf("[StopDeployment] Failed to stop compose deployment %s: %v", deploymentID, err)
			// Continue to update status even if stop failed
		}
	} else if s.manager != nil {
		_ = s.manager.StopDeployment(ctx, deploymentID)
	}

	dbDep.Status = int32(deploymentsv1.DeploymentStatus_STOPPED)
	if err := s.repo.Update(ctx, dbDep); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop deployment: %w", err))
	}
	res := connect.NewResponse(&deploymentsv1.StopDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

// RestartDeployment restarts a deployment
func (s *Service) RestartDeployment(ctx context.Context, req *connect.Request[deploymentsv1.RestartDeploymentRequest]) (*connect.Response[deploymentsv1.RestartDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.restart", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Check if this is a compose-based deployment
	if dbDep.ComposeYaml != "" && s.manager != nil {
		// For compose deployments, restart by stopping and starting again
		_ = s.manager.StopComposeDeployment(ctx, deploymentID)
		if err := s.manager.DeployComposeFile(ctx, deploymentID, dbDep.ComposeYaml); err != nil {
			log.Printf("[RestartDeployment] Failed to restart compose deployment %s: %v", deploymentID, err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restart compose deployment: %w", err))
		}
	} else if s.manager != nil {
		_ = s.manager.RestartDeployment(ctx, deploymentID)
	}

	res := connect.NewResponse(&deploymentsv1.RestartDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

// ScaleDeployment scales a deployment
func (s *Service) ScaleDeployment(ctx context.Context, req *connect.Request[deploymentsv1.ScaleDeploymentRequest]) (*connect.Response[deploymentsv1.ScaleDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.scale", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	// Quota check: replicas delta
	newReplicas := int(req.Msg.GetReplicas())
	if newReplicas <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("replicas must be > 0"))
	}
	if err := s.quotaChecker.CanAllocate(ctx, orgID, quota.RequestedResources{Replicas: newReplicas}); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if s.manager != nil {
		_ = s.manager.ScaleDeployment(ctx, deploymentID, newReplicas)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	res := connect.NewResponse(&deploymentsv1.ScaleDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}
