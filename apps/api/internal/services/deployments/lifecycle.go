package deployments

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"
	"api/internal/quota"

	"connectrpc.com/connect"
	"github.com/google/uuid"
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

	// Get user ID for build record
	userInfo, _ := auth.GetUserFromContext(ctx)
	triggeredBy := "system"
	if userInfo != nil {
		triggeredBy = userInfo.Id
	}

	// Start async rebuild with log streaming
	go func() {
		// Recover from panics to ensure deployment status is always updated
		defer func() {
			if r := recover(); r != nil {
				logger.Error("[TriggerDeployment] PANIC in build goroutine for deployment %s: %v", deploymentID, r)
				// Ensure deployment status is updated even on panic
				_ = s.repo.UpdateStatus(context.Background(), deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
			}
		}()

		buildCtx := context.Background()
		buildStartTime := time.Now()

		// Get or create build log streamer
		streamer := GetBuildLogStreamer(deploymentID)
		defer streamer.Close()

		// Create build record
		buildID := uuid.New().String()
		buildNumber, err := s.buildHistoryRepo.GetNextBuildNumber(buildCtx, deploymentID)
		if err != nil {
			logger.Warn("[TriggerDeployment] Failed to get next build number: %v", err)
			buildNumber = 1 // Fallback to 1
		}

		buildRecord := &database.BuildHistory{
			ID:             buildID,
			DeploymentID:   deploymentID,
			OrganizationID: dbDeployment.OrganizationID,
			BuildNumber:    buildNumber,
			Status:         1, // BUILD_PENDING
			StartedAt:      buildStartTime,
			TriggeredBy:    triggeredBy,
			BuildStrategy:  dbDeployment.BuildStrategy,
			Branch:         dbDeployment.Branch,
		}

		// Capture build configuration snapshot
		if dbDeployment.RepositoryURL != nil {
			buildRecord.RepositoryURL = dbDeployment.RepositoryURL
		}
		if dbDeployment.BuildCommand != nil {
			buildRecord.BuildCommand = dbDeployment.BuildCommand
		}
		if dbDeployment.InstallCommand != nil {
			buildRecord.InstallCommand = dbDeployment.InstallCommand
		}
		if dbDeployment.StartCommand != nil {
			buildRecord.StartCommand = dbDeployment.StartCommand
		}
		if dbDeployment.DockerfilePath != nil {
			buildRecord.DockerfilePath = dbDeployment.DockerfilePath
		}
		if dbDeployment.ComposeFilePath != nil {
			buildRecord.ComposeFilePath = dbDeployment.ComposeFilePath
		}

		if err := s.buildHistoryRepo.CreateBuild(buildCtx, buildRecord); err != nil {
			logger.Warn("[TriggerDeployment] Failed to create build record: %v", err)
			// Still update deployment status to BUILDING even if build record creation fails
			_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_BUILDING))
		} else {
			// Set build ID on streamer so logs are saved to database
			streamer.SetBuildID(buildID)
			// Update build history status to BUILDING
			_ = s.buildHistoryRepo.UpdateBuildStatus(buildCtx, buildID, 2, 0, nil) // BUILD_BUILDING = 2
			// Update deployment status to BUILDING when build actually starts
			_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_BUILDING))
		}

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

			// Fallback to RAILPACK if still unspecified
			if buildStrategy == deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED || buildStrategy == 0 {
				buildStrategy = deploymentsv1.BuildStrategy_RAILPACK
				dbDeployment.BuildStrategy = int32(buildStrategy)
				s.repo.Update(buildCtx, dbDeployment)
			}
		}

		strategy, err := s.buildRegistry.Get(buildStrategy)
		if err != nil {
			logger.Warn("[TriggerDeployment] Invalid build strategy %v: %v", buildStrategy, err)
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
		startCmd := ""
		if dbDeployment.StartCommand != nil {
			startCmd = *dbDeployment.StartCommand
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

		// Get configurable paths
		buildPath := ""
		if dbDeployment.BuildPath != nil {
			buildPath = *dbDeployment.BuildPath
		}
		buildOutputPath := ""
		if dbDeployment.BuildOutputPath != nil {
			buildOutputPath = *dbDeployment.BuildOutputPath
		}
		// Static deployments always use nginx
		useNginx := strategy.Name() == "Static"
		nginxConfig := ""
		if dbDeployment.NginxConfig != nil {
			nginxConfig = *dbDeployment.NginxConfig
		}

		buildConfig := &BuildConfig{
			DeploymentID:    deploymentID,
			RepositoryURL:   repoURL,
			Branch:          dbDeployment.Branch,
			GitHubToken:     githubToken,
			BuildCommand:    buildCmd,
			InstallCommand:  installCmd,
			StartCommand:    startCmd,
			DockerfilePath:  dockerfilePath,
			ComposeFilePath: composeFilePath,
			BuildPath:       buildPath,
			BuildOutputPath: buildOutputPath,
			UseNginx:        useNginx, // Always true for static deployments
			NginxConfig:     nginxConfig,
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

		// Helper function to update build status and calculate build time
		updateBuildStatus := func(status int32, errorMsg *string) {
			buildTime := int32(time.Since(buildStartTime).Seconds())
			if err := s.buildHistoryRepo.UpdateBuildStatus(buildCtx, buildID, status, buildTime, errorMsg); err != nil {
				logger.Warn("[TriggerDeployment] Failed to update build status: %v", err)
			}
		}

		// Capture build result for storage calculation
		var buildResult *BuildResult

		// Handle compose-based deployments
		if dbDeployment.ComposeYaml != "" {
			streamer.Write([]byte("🐳 Deploying Docker Compose configuration...\n"))
			if err := s.manager.DeployComposeFile(buildCtx, deploymentID, dbDeployment.ComposeYaml); err != nil {
				logger.Warn("[TriggerDeployment] Compose deployment failed: %v", err)
				streamer.WriteStderr([]byte(fmt.Sprintf("❌ Deployment failed: %v\n", err)))
				errorMsg := err.Error()
				updateBuildStatus(4, &errorMsg) // BUILD_FAILED = 4
				_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
				return
			}
			// Update build with compose yaml
			composeYaml := dbDeployment.ComposeYaml
			_ = s.buildHistoryRepo.UpdateBuildResults(buildCtx, buildID, nil, &composeYaml, nil)
			// Note: buildResult will be nil for compose deployments, which is fine
			buildResult = nil
			// Containers are automatically registered by DeployComposeFile
		} else {
			// Build using strategy
			streamer.Write([]byte(fmt.Sprintf("🔨 Building deployment with %s...\n", strategy.Name())))
			result, err := strategy.Build(buildCtx, dbDeployment, buildConfig)
			if err != nil || !result.Success {
				logger.Error("[TriggerDeployment] Build failed: %v", err)
				if result != nil && result.Error != nil {
					err = result.Error
				}
				streamer.WriteStderr([]byte(fmt.Sprintf("❌ Build failed: %v\n", err)))
				errorMsg := err.Error()
				updateBuildStatus(4, &errorMsg) // BUILD_FAILED = 4
				_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
				return
			}

			buildResult = result

			streamer.Write([]byte("✅ Build completed successfully\n"))
			
			// Update deployment start command if it was modified by the build strategy
			// (e.g., for Astro projects that need "pnpm build && pnpm preview --host")
			// Also update if start command was extracted from image (e.g., railpack images)
			if buildConfig.StartCommand != "" {
				if startCmd == "" || buildConfig.StartCommand != startCmd {
					dbDeployment.StartCommand = &buildConfig.StartCommand
					logger.Info("[TriggerDeployment] Updated start command to: %s", buildConfig.StartCommand)
				}
			}
			
			// Detect port from build logs (prioritize over repo detection)
			// Build logs are more accurate as they show what the app actually started on
			logPort := streamer.DetectPortFromLogs(200) // Check first 200 log lines for better coverage
			if logPort > 0 {
				result.Port = logPort
				logger.Info("[TriggerDeployment] Detected port %d from build logs", logPort)
				streamer.Write([]byte(fmt.Sprintf("🔍 Detected port %d from build logs\n", logPort)))
			} else if result.Port == 0 {
				// Fallback: if no port detected from logs and build didn't detect one, use default
				// But prefer to use repo detection result which should already be set
				logger.Debug("[TriggerDeployment] No port detected from logs, using build result port: %d", result.Port)
			}
			
			// If in Swarm mode, push image to local registry
			if result.ImageName != "" {
				// Check if we're in Swarm mode
				enableSwarm := os.Getenv("ENABLE_SWARM")
				isSwarmMode := false
				if enableSwarm != "" {
					enabled, err := strconv.ParseBool(strings.ToLower(enableSwarm))
					if err == nil {
						isSwarmMode = enabled
					} else {
						lower := strings.ToLower(strings.TrimSpace(enableSwarm))
						isSwarmMode = lower == "true" || lower == "1" || lower == "yes" || lower == "on"
					}
				}
				
				if isSwarmMode {
					registryURL := os.Getenv("REGISTRY_URL")
					if registryURL == "" {
						domain := os.Getenv("DOMAIN")
						if domain == "" {
							domain = "obiente.cloud"
						}
						registryURL = fmt.Sprintf("https://registry.%s", domain)
					} else {
						// Handle unexpanded docker-compose variables (e.g., "https://registry.${DOMAIN:-obiente.cloud}")
						if strings.Contains(registryURL, "${DOMAIN") {
							domain := os.Getenv("DOMAIN")
							if domain == "" {
								domain = "obiente.cloud"
							}
							registryURL = strings.ReplaceAll(registryURL, "${DOMAIN:-obiente.cloud}", domain)
							registryURL = strings.ReplaceAll(registryURL, "${DOMAIN}", domain)
						}
					}
					
					// Strip protocol from registry URL for image name (Docker doesn't use protocols in image names)
					registryHost := strings.TrimPrefix(registryURL, "https://")
					registryHost = strings.TrimPrefix(registryHost, "http://")
					registryImageName := fmt.Sprintf("%s/%s", registryHost, result.ImageName)
					streamer.Write([]byte(fmt.Sprintf("📤 Pushing image to registry: %s\n", registryImageName)))
					
					registryUsername := os.Getenv("REGISTRY_USERNAME")
					registryPassword := os.Getenv("REGISTRY_PASSWORD")
					if registryUsername == "" {
						registryUsername = "obiente"
					}
					if registryPassword != "" {
						streamer.Write([]byte("🔐 Authenticating with registry...\n"))
						loginCmd := exec.CommandContext(buildCtx, "docker", "login", registryURL, "-u", registryUsername, "-p", registryPassword)
						var loginStderr bytes.Buffer
						loginCmd.Stderr = &loginStderr
						if err := loginCmd.Run(); err != nil {
							logger.Warn("[TriggerDeployment] Failed to authenticate with registry: %v (stderr: %s)", err, loginStderr.String())
							streamer.WriteStderr([]byte(fmt.Sprintf("⚠️  Warning: Failed to authenticate with registry: %v\n", err)))
							streamer.WriteStderr([]byte(fmt.Sprintf("   Error: %s\n", loginStderr.String())))
							streamer.WriteStderr([]byte("   Continuing without authentication (may fail if registry requires auth).\n"))
						} else {
							streamer.Write([]byte("✅ Authenticated with registry\n"))
						}
					} else {
						logger.Warn("[TriggerDeployment] REGISTRY_PASSWORD not set - pushing without authentication")
						streamer.WriteStderr([]byte("⚠️  Warning: REGISTRY_PASSWORD not set - pushing without authentication\n"))
					}
					
					tagCmd := exec.CommandContext(buildCtx, "docker", "tag", result.ImageName, registryImageName)
					if err := tagCmd.Run(); err != nil {
						logger.Warn("[TriggerDeployment] Failed to tag image %s as %s: %v", result.ImageName, registryImageName, err)
						streamer.WriteStderr([]byte(fmt.Sprintf("⚠️  Warning: Failed to tag image for registry: %v\n", err)))
					} else {
						// Push to registry
						pushCmd := exec.CommandContext(buildCtx, "docker", "push", registryImageName)
						var pushStdout bytes.Buffer
						var pushStderr bytes.Buffer
						pushCmd.Stdout = &pushStdout
						pushCmd.Stderr = &pushStderr
						if err := pushCmd.Run(); err != nil {
							logger.Warn("[TriggerDeployment] Failed to push image %s to registry: %v (stderr: %s)", registryImageName, err, pushStderr.String())
							streamer.WriteStderr([]byte(fmt.Sprintf("⚠️  Warning: Failed to push image to registry: %v\n", err)))
							streamer.WriteStderr([]byte("   Registry may not be available or authentication failed. Continuing with local image.\n"))
						} else {
							logger.Info("[TriggerDeployment] Successfully pushed image %s to registry", registryImageName)
							streamer.Write([]byte("✅ Image pushed to registry successfully\n"))
							// Update image name to use registry URL
							result.ImageName = registryImageName
						}
					}
				}
			}
			
			streamer.Write([]byte("🚀 Deploying to orchestrator...\n"))

			// Update build with build results
			var imageName *string
			var composeYaml *string
			var size *string
			if result.ImageName != "" {
				imageName = &result.ImageName
			}
			if result.ComposeYaml != "" {
				composeYaml = &result.ComposeYaml
			}
			// Calculate size from image if available (store as bytes string)
			if result.ImageSizeBytes > 0 {
				sizeStr := fmt.Sprintf("%d", result.ImageSizeBytes)
				size = &sizeStr
			} else if dbDeployment.Size != "" {
				size = &dbDeployment.Size
			}
			_ = s.buildHistoryRepo.UpdateBuildResults(buildCtx, buildID, imageName, composeYaml, size)

			// Deploy using build result
			// Get manager - use service's manager if available, otherwise try to get from global orchestrator
			manager := s.manager
			if manager == nil {
				// Try to get manager from global orchestrator service as fallback
				orchService := orchestrator.GetGlobalOrchestratorService()
				if orchService != nil {
					manager = orchService.GetDeploymentManager()
					logger.Debug("[TriggerDeployment] Manager was nil, retrieved from global orchestrator service")
				}
			}
			
			// If still nil, try to create a new one as last resort
			if manager == nil {
				logger.Warn("[TriggerDeployment] WARNING: Creating deployment manager as last resort...")
				var err error
				manager, err = orchestrator.NewDeploymentManager("least-loaded", 50)
				if err != nil {
					logger.Error("[TriggerDeployment] CRITICAL: Failed to create deployment manager: %v", err)
					streamer.WriteStderr([]byte(fmt.Sprintf("❌ Deployment failed: deployment manager is not available (orchestrator not initialized): %v\n", err)))
					errorMsg := fmt.Sprintf("deployment manager is not available (orchestrator not initialized): %v", err)
					updateBuildStatus(4, &errorMsg) // BUILD_FAILED = 4
					_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
					return
				}
				logger.Info("[TriggerDeployment] Successfully created deployment manager as last resort")
			}
			
			if err := deployResultToOrchestrator(buildCtx, manager, dbDeployment, result); err != nil {
				logger.Error("[TriggerDeployment] Deployment failed: %v", err)
				streamer.WriteStderr([]byte(fmt.Sprintf("❌ Deployment failed: %v\n", err)))
				errorMsg := err.Error()
				updateBuildStatus(4, &errorMsg) // BUILD_FAILED = 4
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

		// Update build status in build history as successful (build completed)
		// buildTime is calculated inside updateBuildStatus
		updateBuildStatus(3, nil) // BUILD_SUCCESS = 3
		
		// Verify containers are running
		streamer.Write([]byte("🔍 Verifying containers are running...\n"))
		if err := s.verifyContainersRunning(buildCtx, deploymentID); err != nil {
			logger.Warn("[TriggerDeployment] WARNING: Containers not running: %v", err)
			streamer.WriteStderr([]byte(fmt.Sprintf("⚠️  Warning: %v\n", err)))
			// Don't mark build as failed - build itself succeeded, just container verification failed
			_ = s.repo.UpdateStatus(buildCtx, deploymentID, int32(deploymentsv1.DeploymentStatus_FAILED))
		} else {
			streamer.Write([]byte("✅ Deployment completed successfully!\n"))
			
			// Calculate and update storage usage
			streamer.Write([]byte("📊 Calculating storage usage...\n"))
			if err := s.updateDeploymentStorage(buildCtx, deploymentID, buildResult); err != nil {
				logger.Warn("[TriggerDeployment] Warning: Failed to update storage: %v", err)
				// Don't fail deployment if storage calculation fails
			}
			
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
				logger.Warn("[StartDeployment] Failed to deploy compose file for deployment %s: %v", deploymentID, err)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to deploy compose file: %w", err))
			}
			logger.Info("[StartDeployment] Successfully deployed compose file for deployment %s", deploymentID)

			// Verify containers are actually running before setting status
			if err := s.verifyContainersRunning(ctx, deploymentID); err != nil {
				logger.Warn("[StartDeployment] WARNING: Containers not running for deployment %s: %v", deploymentID, err)
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
				// Get port from routing configuration if available, otherwise use deployment port
				port := 8080
				if dbDep.Port != nil {
					port = int(*dbDep.Port)
				}
				
				// Check routing configuration for target port (takes precedence)
				routings, err := database.GetDeploymentRoutings(deploymentID)
				if err == nil && len(routings) > 0 {
					// Track if we found a routing rule
					foundRouting := false
					// Find routing rule for "default" service (or first one if no service name specified)
					for _, routing := range routings {
						if routing.ServiceName == "" || routing.ServiceName == "default" {
							port = routing.TargetPort
							logger.Info("[StartDeployment] Using target port %d from routing configuration (default service) for deployment %s", port, deploymentID)
							foundRouting = true
							break
						}
					}
					// If no default service routing found, use first routing's target port
					if !foundRouting {
						port = routings[0].TargetPort
						logger.Info("[StartDeployment] Using target port %d from first routing rule for deployment %s", port, deploymentID)
					}
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
					logger.Error("[StartDeployment] Failed to create containers for deployment %s: %v", deploymentID, err)
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create containers: %w", err))
				}
				logger.Info("[StartDeployment] Successfully created containers for deployment %s", deploymentID)

				// Verify containers are actually running before setting status
				if err := s.verifyContainersRunning(ctx, deploymentID); err != nil {
					logger.Warn("[StartDeployment] WARNING: Containers not running for deployment %s: %v", deploymentID, err)
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
				logger.Warn("[StartDeployment] WARNING: Containers not running for deployment %s after start: %v", deploymentID, err)
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
			logger.Warn("[StopDeployment] Failed to stop compose deployment %s: %v", deploymentID, err)
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
			logger.Warn("[RestartDeployment] Failed to restart compose deployment %s: %v", deploymentID, err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restart compose deployment: %w", err))
		}
	} else if s.manager != nil {
		if err := s.manager.RestartDeployment(ctx, deploymentID); err != nil {
			logger.Warn("[RestartDeployment] Failed to restart deployment %s: %v", deploymentID, err)
			// Don't return error immediately - try to start deployment if restart failed
			// This handles the case where containers don't exist
			if err := s.manager.StartDeployment(ctx, deploymentID); err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restart deployment: %w", err))
			}
			logger.Info("[RestartDeployment] Successfully started deployment %s after restart failure", deploymentID)
		}
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment manager not available"))
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

// updateDeploymentStorage calculates and updates storage usage for a deployment
func (s *Service) updateDeploymentStorage(ctx context.Context, deploymentID string, result *BuildResult) error {
	// Get all container locations for this deployment
	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment locations: %w", err)
	}

	// Collect container IDs
	containerIDs := make([]string, 0, len(locations))
	for _, loc := range locations {
		if loc.ContainerID != "" {
			containerIDs = append(containerIDs, loc.ContainerID)
		}
	}

	// Determine image name
	imageName := ""
	if result != nil && result.ImageName != "" {
		imageName = result.ImageName
	} else {
		// Try to get from deployment
		deployment, err := s.repo.GetByID(ctx, deploymentID)
		if err == nil && deployment.Image != nil {
			imageName = *deployment.Image
		}
	}

	// Calculate storage
	storageInfo, err := CalculateStorage(ctx, imageName, containerIDs)
	if err != nil {
		return fmt.Errorf("failed to calculate storage: %w", err)
	}

	// Update deployment with storage information
	if err := s.repo.UpdateStorage(ctx, deploymentID, storageInfo.TotalStorage); err != nil {
		return fmt.Errorf("failed to update storage: %w", err)
	}

	logger.Debug("[updateDeploymentStorage] Updated storage for deployment %s: Image=%d bytes, Volumes=%d bytes, Container=%d bytes, Total=%d bytes",
		deploymentID, storageInfo.ImageSize, storageInfo.VolumeSize, storageInfo.ContainerDisk, storageInfo.TotalStorage)

	return nil
}
