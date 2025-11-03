import { ref, computed, onUnmounted } from "vue";
import { DeploymentService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

export interface BuildProgressOptions {
  deploymentId: string;
  organizationId: string;
}

/**
 * Tracks build progress by analyzing build log patterns
 */
export function useBuildProgress(options: BuildProgressOptions) {
  const client = useConnectClient(DeploymentService);
  const targetProgress = ref(0); // Target progress value from log analysis
  const progress = ref(0); // Actual displayed progress (smoothly animated)
  const currentPhase = ref<string>("Starting deployment...");
  const isStreaming = ref(false);
  let streamController: AbortController | null = null;
  let animationFrameId: number | null = null;

  // Build phase patterns and their progress percentages
  // Order matters: more specific patterns should come first
  const buildPhases = [
    // Early phases
    {
      pattern: /🚀 Starting deployment/i,
      progress: 2,
      phase: "Starting deployment",
    },
    {
      pattern: /📦 Using build strategy/i,
      progress: 5,
      phase: "Preparing build",
    },
    {
      pattern: /🔨 Building deployment/i,
      progress: 8,
      phase: "Initializing build",
    },
    // Nixpacks-specific phases
    {
      pattern: /Nixpacks v\d+\.\d+\.\d+|╔═══════════ Nixpacks/i,
      progress: 10,
      phase: "Nixpacks: Detecting environment",
    },
    {
      pattern: /║ setup|║ install|║ build|║ start/i,
      progress: 12,
      phase: "Nixpacks: Configuring build",
    },
    {
      pattern: /#0 building with.*docker driver|building with "default" instance/i,
      progress: 12,
      phase: "Nixpacks: Starting Docker build",
    },
    {
      pattern: /\[internal\] load.*dockerfile|\[internal\] load metadata/i,
      progress: 15,
      phase: "Nixpacks: Loading build context",
    },
    {
      pattern: /\[stage-0\s+\d+\/\d+\].*FROM|FROM ghcr\.io\/railwayapp\/nixpacks/i,
      progress: 18,
      phase: "Nixpacks: Pulling base image",
    },
    {
      pattern: /nix-env -if|installing.*nix|these \d+ derivations will be built/i,
      progress: 20,
      phase: "Nixpacks: Installing Nix packages",
    },
    {
      pattern: /copying path.*from 'https:\/\/cache\.nixos\.org'/i,
      progress: 22,
      phase: "Nixpacks: Downloading Nix packages",
    },
    {
      pattern: /building '\/nix\/store\/|these \d+ paths will be fetched/i,
      progress: 25,
      phase: "Nixpacks: Building Nix packages",
    },
    // Railpack-specific phases
    {
      pattern: /Railpack \d+\.\d+\.\d+|Detected Node|Using .* package manager/i,
      progress: 10,
      phase: "Railpack: Detecting environment",
    },
    {
      pattern: /docker-image.*railpack-builder|resolve.*railpack-builder/i,
      progress: 12,
      phase: "Railpack: Pulling builder image",
    },
    {
      pattern: /docker-image.*railpack-runtime|resolve.*railpack-runtime/i,
      progress: 15,
      phase: "Railpack: Pulling runtime image",
    },
    {
      pattern: /transferring context|loading \./i,
      progress: 18,
      phase: "Preparing build context",
    },
    {
      pattern: /extracting sha256|sha256.*done/i,
      progress: 20,
      phase: "Extracting base images",
    },
    {
      pattern: /install mise packages|mise.*install/i,
      progress: 25,
      phase: "Installing runtime tools",
    },
    {
      pattern: /\[stage-0\s+\d+\/\d+\].*RUN.*pnpm i|\[stage-0\s+\d+\/\d+\].*RUN.*npm i|\[stage-0\s+\d+\/\d+\].*RUN.*yarn/i,
      progress: 28,
      phase: "Nixpacks: Installing dependencies",
    },
    {
      pattern: /pnpm install|npm install|yarn install|pip install/i,
      progress: 30,
      phase: "Installing dependencies",
    },
    {
      pattern: /Progress:.*resolved|Packages:.*\+/i,
      progress: 35,
      phase: "Installing packages",
    },
    {
      pattern: /Done in.*using (pnpm|npm|yarn)/i,
      progress: 40,
      phase: "Dependencies installed",
    },
    {
      pattern: /\[stage-0\s+\d+\/\d+\].*RUN.*pnpm run build|\[stage-0\s+\d+\/\d+\].*RUN.*npm run build|\[stage-0\s+\d+\/\d+\].*RUN.*yarn build/i,
      progress: 43,
      phase: "Nixpacks: Building application",
    },
    {
      pattern: /pnpm run build|npm run build|yarn build|go build|\.\/build/i,
      progress: 45,
      phase: "Building application",
    },
    {
      pattern: /\[build\]|\[vite\]|building.*vite|building.*server/i,
      progress: 50,
      phase: "Compiling application",
    },
    {
      pattern: /built in|Completed in|✓ built/i,
      progress: 55,
      phase: "Build compilation",
    },
    {
      pattern: /copy.*node_modules|copy.*app|copy.*\/app/i,
      progress: 60,
      phase: "Copying files",
    },
    {
      pattern: /exporting to docker image|exporting layers/i,
      progress: 70,
      phase: "Exporting image",
    },
    {
      pattern: /exporting manifest|exporting config/i,
      progress: 75,
      phase: "Finalizing image",
    },
    {
      pattern: /sending tarball/i,
      progress: 80,
      phase: "Uploading image",
    },
    {
      pattern: /Successfully built image|Loaded image/i,
      progress: 85,
      phase: "Image ready",
    },
    {
      pattern: /✅ Build completed successfully/i,
      progress: 90,
      phase: "Build completed",
    },
    // Generic build patterns (fallback)
    {
      pattern: /🐳 Deploying Docker Compose/i,
      progress: 30,
      phase: "Deploying Compose",
    },
    {
      pattern: /pulling|pulling image|pulling.*from/i,
      progress: 20,
      phase: "Pulling images",
    },
    {
      pattern: /building.*image|step \d+\/\d+|Step \d+\/\d+/i,
      progress: 40,
      phase: "Building image",
    },
    // Deployment phases
    {
      pattern: /🚀 Deploying to orchestrator/i,
      progress: 92,
      phase: "Deploying containers",
    },
    {
      pattern: /🔍 Verifying containers/i,
      progress: 95,
      phase: "Verifying containers",
    },
    {
      pattern: /📊 Calculating storage/i,
      progress: 97,
      phase: "Calculating storage",
    },
    // Completion
    {
      pattern: /✅ Deployment completed successfully/i,
      progress: 100,
      phase: "Deployment complete",
    },
    // Failure patterns - these should be checked early to catch failures
    {
      pattern: /❌ Build failed|❌ Deployment failed|Build failed:|ERROR:.*failed|ERROR: failed to build/i,
      progress: -1, // Special marker for failure
      phase: "Build failed",
    },
    {
      pattern: /ERROR:.*exit code|exit status 1|Command failed|did not complete successfully/i,
      progress: -1,
      phase: "Build error",
    },
  ];

  // Smoothly animate progress towards target
  const animateProgress = () => {
    if (animationFrameId) {
      cancelAnimationFrame(animationFrameId);
    }

    const animate = () => {
      const diff = targetProgress.value - progress.value;
      
      if (Math.abs(diff) < 0.1) {
        // Close enough, set directly
        progress.value = targetProgress.value;
        animationFrameId = null;
        return;
      }

      // Smooth interpolation: move 15% of the remaining distance per frame
      // This creates a smooth deceleration effect
      progress.value += diff * 0.15;
      
      animationFrameId = requestAnimationFrame(animate);
    };

    animate();
  };

  const isFailed = ref(false);

  const updateProgressFromLog = (logLine: string) => {
    // Check failure patterns first (they have progress: -1)
    for (const phase of buildPhases) {
      if (phase.pattern.test(logLine) && phase.progress === -1) {
        isFailed.value = true;
        targetProgress.value = progress.value; // Keep current progress, don't animate forward
        currentPhase.value = phase.phase;
        // Stop any ongoing animation
        if (animationFrameId) {
          cancelAnimationFrame(animationFrameId);
          animationFrameId = null;
        }
        return; // Don't process further on failure
      }
    }

    // Check each phase pattern in order (skip if already failed)
    if (!isFailed.value) {
      for (const phase of buildPhases) {
        if (phase.pattern.test(logLine) && phase.progress >= 0) {
          if (targetProgress.value < phase.progress) {
            targetProgress.value = phase.progress;
            currentPhase.value = phase.phase;
            animateProgress();
          }
          break; // Use first match
        }
      }
    }

    // Additional incremental progress tracking for specific phases
    
    // Track package installation progress (pnpm/npm/yarn) - works for both Railpack and Nixpacks
    const packageProgressMatch = logLine.match(/Progress:.*resolved (\d+),.*(?:reused|downloaded) (\d+),.*added (\d+)/i);
    if (packageProgressMatch && packageProgressMatch[1] && packageProgressMatch[3]) {
      const resolved = parseInt(packageProgressMatch[1], 10);
      const downloaded = packageProgressMatch[2] ? parseInt(packageProgressMatch[2], 10) : 0;
      const added = parseInt(packageProgressMatch[3], 10);
      // Estimate progress based on added packages
      if (resolved > 0 && added > 0) {
        // For Nixpacks: pnpm install happens after Nix setup (30-40% range)
        // For Railpack: pnpm install is earlier (30-45% range)
        const installProgress = Math.min(45, Math.floor((added / resolved) * 15) + 30);
        if (installProgress > targetProgress.value) {
          targetProgress.value = installProgress;
          const phasePrefix = currentPhase.value.includes("Nixpacks") ? "Nixpacks: " : "";
          currentPhase.value = `${phasePrefix}Installing packages (${added}/${resolved})`;
          animateProgress();
        }
      }
    }
    
    // Track Nix package downloading progress
    const nixDownloadMatch = logLine.match(/these (\d+) paths will be fetched.*\(([\d.]+)\s*(MB|GB|KB|MiB|GiB|KiB)\s*download/i);
    if (nixDownloadMatch && nixDownloadMatch[1]) {
      const totalPaths = parseInt(nixDownloadMatch[1], 10);
      // Nix downloading typically happens in 20-28% range for Nixpacks
      if (totalPaths > 0 && targetProgress.value < 28) {
        targetProgress.value = 22;
        currentPhase.value = "Nixpacks: Downloading Nix packages";
        animateProgress();
      }
    }
    
    // Track Nix package copying progress
    const nixCopyMatch = logLine.match(/copying path '\/nix\/store\/.*' from 'https:\/\/cache\.nixos\.org'/i);
    if (nixCopyMatch && targetProgress.value >= 20 && targetProgress.value < 30) {
      // Incrementally update during Nix package copying (22-28% range)
      const incrementalProgress = Math.min(28, targetProgress.value + 0.5);
      if (incrementalProgress > targetProgress.value) {
        targetProgress.value = incrementalProgress;
        if (!currentPhase.value.includes("Nixpacks")) {
          currentPhase.value = "Nixpacks: Downloading Nix packages";
        }
        animateProgress();
      }
    }

    // Track Docker build steps (#1, #2, #3, etc.) - works for both Railpack and Nixpacks
    const buildStepMatch = logLine.match(/#(\d+)\s+(DONE|CACHED|\.\.\.|\[internal\]|\[stage-0)/i);
    if (buildStepMatch && buildStepMatch[1]) {
      const stepNum = parseInt(buildStepMatch[1], 10);
      
      // Check if it's a Nixpacks stage step
      const nixpacksStageMatch = logLine.match(/#(\d+)\s+\[stage-0\s+(\d+)\/(\d+)\]/i);
      if (nixpacksStageMatch && nixpacksStageMatch[1] && nixpacksStageMatch[2] && nixpacksStageMatch[3]) {
        const stepNum = parseInt(nixpacksStageMatch[1], 10);
        const stageNum = parseInt(nixpacksStageMatch[2], 10);
        const totalStages = parseInt(nixpacksStageMatch[3], 10);
        
        // Nixpacks has ~10 stages, map them to progress ranges
        // Stage 1-3: Nix setup (20-30%)
        // Stage 4-6: Package install (30-45%)
        // Stage 7-8: Build (45-60%)
        // Stage 9-10: Final copy/export (60-70%)
        let stageProgress = 20;
        if (stageNum <= 3) {
          stageProgress = 20 + Math.floor((stageNum / 3) * 10); // 20-30%
        } else if (stageNum <= 6) {
          stageProgress = 30 + Math.floor(((stageNum - 3) / 3) * 15); // 30-45%
        } else if (stageNum <= 8) {
          stageProgress = 45 + Math.floor(((stageNum - 6) / 2) * 15); // 45-60%
        } else {
          stageProgress = 60 + Math.floor(((stageNum - 8) / 2) * 10); // 60-70%
        }
        
        stageProgress = Math.min(70, stageProgress);
        if (stageProgress > targetProgress.value) {
          targetProgress.value = stageProgress;
          
          // Set phase based on stage
          let phaseName = `Nixpacks: Stage ${stageNum}/${totalStages}`;
          if (stageNum <= 3) {
            phaseName = "Nixpacks: Installing Nix packages";
          } else if (stageNum === 4 || stageNum === 5) {
            phaseName = "Nixpacks: Installing dependencies";
          } else if (stageNum === 6 || stageNum === 7) {
            phaseName = "Nixpacks: Building application";
          } else {
            phaseName = "Nixpacks: Finalizing build";
          }
          currentPhase.value = phaseName;
          animateProgress();
        }
      } else {
        // Generic Docker step (Railpack or early Nixpacks)
        // Railpack builds typically have 20-25 steps, Nixpacks has ~15 steps
        // Estimate progress based on step number
        const stepProgress = Math.min(70, 20 + Math.floor((stepNum / 25) * 50)); // 20-70% range
        if (stepProgress > targetProgress.value) {
          targetProgress.value = stepProgress;
          if (!currentPhase.value.includes("Building") && !currentPhase.value.includes("Nixpacks")) {
            currentPhase.value = `Building (step ${stepNum})`;
          }
          animateProgress();
        }
      }
    }

    // Track Docker image download progress
    const imageDownloadMatch = logLine.match(/sha256:.*?(\d+(?:\.\d+)?)\s*(MB|GB|KB|B)\s*\/\s*(\d+(?:\.\d+)?)\s*(MB|GB|KB|B)/i);
    if (imageDownloadMatch && imageDownloadMatch[1] && imageDownloadMatch[2] && imageDownloadMatch[3] && imageDownloadMatch[4]) {
      const downloaded = parseFloat(imageDownloadMatch[1]);
      const downloadedUnit = imageDownloadMatch[2];
      const total = parseFloat(imageDownloadMatch[3]);
      const totalUnit = imageDownloadMatch[4];
      
      // Convert to MB for comparison
      const toMB = (val: number, unit: string) => {
        switch (unit.toUpperCase()) {
          case "GB": return val * 1024;
          case "MB": return val;
          case "KB": return val / 1024;
          case "B": return val / (1024 * 1024);
          default: return val;
        }
      };
      
      const downloadedMB = toMB(downloaded, downloadedUnit);
      const totalMB = toMB(total, totalUnit);
      
      if (totalMB > 0) {
        const downloadProgress = Math.min(25, 12 + Math.floor((downloadedMB / totalMB) * 8)); // 12-20% range
        if (downloadProgress > targetProgress.value) {
          targetProgress.value = downloadProgress;
          currentPhase.value = `Pulling images (${Math.floor((downloadedMB / totalMB) * 100)}%)`;
          animateProgress();
        }
      }
    }

    // Track file copying progress (incremental)
    const copyMatch = logLine.match(/#(\d+)\s+copy.*DONE|#(\d+)\s+copy.*done/i);
    if (copyMatch) {
      const stepNum = copyMatch[1] ? parseInt(copyMatch[1], 10) : (copyMatch[2] ? parseInt(copyMatch[2], 10) : 0);
      if (stepNum > 0) {
        // File copying typically happens in steps 15-21 for Railpack
        if (stepNum >= 15 && stepNum <= 21) {
          const copyProgress = Math.min(70, 55 + Math.floor(((stepNum - 15) / 6) * 15)); // 55-70% range
          if (copyProgress > targetProgress.value) {
            targetProgress.value = copyProgress;
            currentPhase.value = "Copying files";
            animateProgress();
          }
        }
      }
    }
  };

  const startStreaming = async () => {
    if (isStreaming.value || streamController) {
      return;
    }

    isStreaming.value = true;
    targetProgress.value = 0;
    progress.value = 0;
    isFailed.value = false;
    currentPhase.value = "Starting deployment...";
    streamController = new AbortController();

    try {
      const stream = await (client as any).streamBuildLogs(
        {
          organizationId: options.organizationId,
          deploymentId: options.deploymentId,
        },
        { signal: streamController.signal }
      );

      for await (const update of stream) {
        if (streamController?.signal.aborted) {
          break;
        }
        if (update.line) {
          updateProgressFromLog(update.line);
        }
      }
    } catch (err: any) {
      if (err.name === "AbortError") {
        return;
      }
      // Suppress benign stream errors
      const isBenignError =
        err.message?.toLowerCase().includes("missing trailer") ||
        err.message?.toLowerCase().includes("trailer") ||
        err.code === "unknown";

      if (!isBenignError) {
        console.error("Failed to stream build logs for progress:", err);
      }
    } finally {
      isStreaming.value = false;
      streamController = null;
    }
  };

  const stopStreaming = () => {
    if (streamController) {
      streamController.abort();
      streamController = null;
    }
    if (animationFrameId) {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    }
    isStreaming.value = false;
  };

  const reset = () => {
    targetProgress.value = 0;
    progress.value = 0;
    isFailed.value = false;
    currentPhase.value = "Starting deployment...";
    stopStreaming();
  };

  // Note: Auto-start is handled by the caller, not here
  // This allows more control over when streaming starts/stops

  // Cleanup on unmount
  onUnmounted(() => {
    stopStreaming();
  });

  return {
    progress: computed(() => progress.value),
    currentPhase: computed(() => currentPhase.value),
    isStreaming: computed(() => isStreaming.value),
    startStreaming,
    stopStreaming,
    reset,
  };
}

