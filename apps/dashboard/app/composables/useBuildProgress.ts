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
  let incrementalProgressIntervalId: ReturnType<typeof setInterval> | null = null;
  let lastLogUpdateTime = ref(Date.now());

  // Build phase patterns and their progress percentages
  // Order matters: more specific patterns should come first
  const buildPhases = [
    // Early phases - detailed tracking
    {
      pattern: /🚀 Starting deployment/i,
      progress: 1,
      phase: "Starting deployment",
    },
    {
      pattern: /📦 Using build strategy/i,
      progress: 2,
      phase: "Selecting build strategy",
    },
    {
      pattern: /🔨 Building deployment/i,
      progress: 3,
      phase: "Initializing build",
    },
    {
      pattern: /🚀 Obiente Cloud: Starting deployment build/i,
      progress: 4,
      phase: "Starting build process",
    },
    {
      pattern: /📥 Cloning repository/i,
      progress: 5,
      phase: "Cloning repository",
    },
    {
      pattern: /✅ Repository cloned successfully/i,
      progress: 6,
      phase: "Repository cloned",
    },
    {
      pattern: /📁 Build path/i,
      progress: 7,
      phase: "Identifying build path",
    },
    {
      pattern: /🔧 Analyzing project|🔧 Analyzing project and configuring build/i,
      progress: 8,
      phase: "Analyzing project",
    },
    {
      pattern: /📦 Configured install command|🚀 Configured start command|✅ Created\/updated nixpacks\.toml/i,
      progress: 9,
      phase: "Configuring build commands",
    },
    {
      pattern: /✨ Obiente Cloud: Auto-detected|Detected.*project/i,
      progress: 9,
      phase: "Detecting project type",
    },
    {
      pattern: /✅ Obiente Cloud: Build configuration complete/i,
      progress: 10,
      phase: "Configuration complete",
    },
    {
      pattern: /🔨 Building application with/i,
      progress: 11,
      phase: "Starting application build",
    },
    // Nixpacks-specific phases
    {
      pattern: /Nixpacks v\d+\.\d+\.\d+|╔═══════════ Nixpacks/i,
      progress: 12,
      phase: "Nixpacks: Detecting environment",
    },
    {
      pattern: /║ setup|║ install|║ build|║ start/i,
      progress: 13,
      phase: "Nixpacks: Configuring build",
    },
    {
      pattern: /#0 building with.*docker driver|building with "default" instance/i,
      progress: 14,
      phase: "Nixpacks: Starting Docker build",
    },
    {
      pattern: /\[internal\] load.*dockerfile|\[internal\] load metadata/i,
      progress: 15,
      phase: "Nixpacks: Loading build context",
    },
    {
      pattern: /\[stage-0\s+\d+\/\d+\].*FROM|FROM ghcr\.io\/railwayapp\/nixpacks/i,
      progress: 16,
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
      pattern: /╭.*Railpack.*╮|Railpack \d+\.\d+\.\d+/i,
      progress: 12,
      phase: "Railpack: Detecting environment",
    },
    {
      pattern: /Detected Node|Using .* package manager|↳ Detected/i,
      progress: 12,
      phase: "Railpack: Analyzing project",
    },
    {
      pattern: /#1 loading|#1 transferring context/i,
      progress: 14,
      phase: "Railpack: Loading build context",
    },
    {
      pattern: /#2 \[railpack\]|\[railpack\] secrets hash/i,
      progress: 15,
      phase: "Railpack: Configuring secrets",
    },
    {
      pattern: /#3 docker-image.*railpack-builder|resolve.*railpack-builder/i,
      progress: 16,
      phase: "Railpack: Pulling builder image",
    },
    {
      pattern: /#4 docker-image.*railpack-runtime|resolve.*railpack-runtime/i,
      progress: 17,
      phase: "Railpack: Pulling runtime image",
    },
    {
      pattern: /transferring context.*done|loading \./i,
      progress: 18,
      phase: "Railpack: Transferring context",
    },
    {
      pattern: /#5.*mkdir|#6.*create|#7.*copy package|#8.*mkdir|#9.*copy.*lock/i,
      progress: 19,
      phase: "Railpack: Setting up build environment",
    },
    {
      pattern: /#10 install mise packages|mise.*install|install mise packages: node/i,
      progress: 20,
      phase: "Railpack: Installing runtime tools",
    },
    {
      pattern: /#11 pnpm install|#11 npm install|#11 yarn install/i,
      progress: 25,
      phase: "Railpack: Installing dependencies",
    },
    {
      pattern: /#12 copy.*\/app|#12 copy \/ \/app/i,
      progress: 30,
      phase: "Railpack: Copying application files",
    },
    {
      pattern: /#13 pnpm run build|#13 npm run build|#13 yarn build/i,
      progress: 35,
      phase: "Railpack: Building application",
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
      progress: 42,
      phase: "Dependencies installed",
    },
    {
      pattern: /\[stage-0\s+\d+\/\d+\].*RUN.*pnpm run build|\[stage-0\s+\d+\/\d+\].*RUN.*npm run build|\[stage-0\s+\d+\/\d+\].*RUN.*yarn build/i,
      progress: 44,
      phase: "Nixpacks: Building application",
    },
    {
      pattern: /pnpm run build|npm run build|yarn build|go build|\.\/build/i,
      progress: 45,
      phase: "Building application",
    },
    {
      pattern: /\[build\]|\[vite\]|building.*vite|building.*server|building client.*vite/i,
      progress: 50,
      phase: "Compiling application",
    },
    {
      pattern: /\[build\] Rearranging server assets|\[build\] Server built/i,
      progress: 55,
      phase: "Finalizing build",
    },
    {
      pattern: /\[build\] Complete!|built in|Completed in|✓ built/i,
      progress: 58,
      phase: "Build compilation",
    },
    {
      pattern: /#20 copy.*node_modules|#21 copy.*\/app|#22 \[railpack\] merge/i,
      progress: 60,
      phase: "Railpack: Copying files",
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
    // Order matters: more specific patterns should come first
    // npm-specific lockfile errors (most specific)
    {
      pattern: /npm error code EUSAGE|npm ci.*can only install packages when.*package\.json.*package-lock\.json.*are in sync|Missing:.*from lock file/i,
      progress: -1, // Special marker for failure
      phase: "Dependency installation failed",
    },
    // npm errors (general)
    {
      pattern: /npm error|npm ERR!/i,
      progress: -1,
      phase: "npm error",
    },
    // Build failure messages (explicit)
    {
      pattern: /❌ Build failed|❌ Deployment failed|Build failed:|Error: Docker build failed/i,
      progress: -1,
      phase: "Build failed",
    },
    // Docker build errors
    {
      pattern: /ERROR: failed to build|ERROR:.*failed|failed to solve|Docker build failed/i,
      progress: -1,
      phase: "Docker build failed",
    },
    // Process execution errors
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

      // Smooth interpolation with adaptive speed based on jump size
      // For small jumps (< 5%), move 5% of remaining distance per frame
      // For medium jumps (5-15%), move 4% of remaining distance per frame  
      // For large jumps (> 15%), move 3% of remaining distance per frame
      // This ensures smooth animation even for large jumps
      let speed = 0.03; // Default for large jumps (slowest)
      if (Math.abs(diff) < 5) {
        speed = 0.05; // Small jumps can be slightly faster
      } else if (Math.abs(diff) < 15) {
        speed = 0.04; // Medium jumps
      }
      
      // Cap the maximum change per frame to ensure smoothness
      const maxChangePerFrame = 0.5; // Maximum 0.5% per frame
      const change = Math.min(diff * speed, maxChangePerFrame);
      
      progress.value += change;
      
      animationFrameId = requestAnimationFrame(animate);
    };

    animate();
  };

  const isFailed = ref(false);

  const updateProgressFromLog = (logLine: string) => {
    // Update last log update time
    lastLogUpdateTime.value = Date.now();
    
    // Check failure patterns first (they have progress: -1)
    for (const phase of buildPhases) {
      if (phase.pattern.test(logLine) && phase.progress === -1) {
        isFailed.value = true;
        targetProgress.value = progress.value; // Keep current progress, don't animate forward
        currentPhase.value = phase.phase;
        // Stop any ongoing animation and incremental progress
        if (animationFrameId) {
          cancelAnimationFrame(animationFrameId);
          animationFrameId = null;
        }
        stopIncrementalProgress();
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
            
            // Stop incremental progress if we've reached completion
            if (phase.progress >= 90) {
              stopIncrementalProgress();
            }
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
        // For Nixpacks: npm install happens at stage 5-6 (28-42% range)
        // For Railpack: pnpm install is earlier (30-45% range)
        const baseProgress = currentPhase.value.includes("Nixpacks") ? 28 : 30;
        const maxProgress = currentPhase.value.includes("Nixpacks") ? 42 : 45;
        const installProgress = Math.min(maxProgress, baseProgress + Math.floor((added / resolved) * (maxProgress - baseProgress)));
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
      // Nix downloading typically happens at stages 3-4 (18-28% range for Nixpacks)
      if (totalPaths > 0 && targetProgress.value < 28) {
        targetProgress.value = 18;
        currentPhase.value = "Nixpacks: Downloading Nix packages";
        animateProgress();
      }
    }
    
    // Track Nix package copying progress
    const nixCopyMatch = logLine.match(/copying path '\/nix\/store\/.*' from 'https:\/\/cache\.nixos\.org'/i);
    if (nixCopyMatch && targetProgress.value >= 18 && targetProgress.value < 28) {
      // Incrementally update during Nix package copying (18-28% range)
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
    // Match #NUM at start of line or after whitespace, followed by any content
    const buildStepMatch = logLine.match(/(?:^|\s)#(\d+)(?:\s+|$)/);
    if (buildStepMatch && buildStepMatch[1]) {
      const stepNum = parseInt(buildStepMatch[1], 10);
      // Detect Railpack: check phase name, log content, or step patterns
      const isRailpack = 
        currentPhase.value.includes("Railpack") || 
        logLine.includes("[railpack]") || 
        logLine.includes("railpack-builder") || 
        logLine.includes("railpack-runtime");
      
      // Check if it's a Nixpacks stage step
      const nixpacksStageMatch = logLine.match(/#(\d+)\s+\[stage-0\s+(\d+)\/(\d+)\]/i);
      if (nixpacksStageMatch && nixpacksStageMatch[1] && nixpacksStageMatch[2] && nixpacksStageMatch[3]) {
        const stepNum = parseInt(nixpacksStageMatch[1], 10);
        const stageNum = parseInt(nixpacksStageMatch[2], 10);
        const totalStages = parseInt(nixpacksStageMatch[3], 10);
        
        // Nixpacks has ~10 stages, map them to progress ranges
        // Stage 1-2: Initial setup (16-18%)
        // Stage 3-4: Nix setup (18-28%)
        // Stage 5-6: Package install (28-42%)
        // Stage 7-8: Build (42-58%)
        // Stage 9-10: Final copy/export (58-70%)
        let stageProgress = 16;
        if (stageNum <= 2) {
          stageProgress = 16 + Math.floor((stageNum / 2) * 2); // 16-18%
        } else if (stageNum <= 4) {
          stageProgress = 18 + Math.floor(((stageNum - 2) / 2) * 10); // 18-28%
        } else if (stageNum <= 6) {
          stageProgress = 28 + Math.floor(((stageNum - 4) / 2) * 14); // 28-42%
        } else if (stageNum <= 8) {
          stageProgress = 42 + Math.floor(((stageNum - 6) / 2) * 16); // 42-58%
        } else {
          stageProgress = 58 + Math.floor(((stageNum - 8) / 2) * 12); // 58-70%
        }
        
        stageProgress = Math.min(70, stageProgress);
        if (stageProgress > targetProgress.value) {
          targetProgress.value = stageProgress;
          
          // Set phase based on stage
          let phaseName = `Nixpacks: Stage ${stageNum}/${totalStages}`;
          if (stageNum <= 2) {
            phaseName = "Nixpacks: Initializing build";
          } else if (stageNum <= 4) {
            phaseName = "Nixpacks: Installing Nix packages";
          } else if (stageNum === 5 || stageNum === 6) {
            phaseName = "Nixpacks: Installing dependencies";
          } else if (stageNum === 7 || stageNum === 8) {
            phaseName = "Nixpacks: Building application";
          } else {
            phaseName = "Nixpacks: Finalizing build";
          }
          currentPhase.value = phaseName;
          animateProgress();
        }
      } else if (isRailpack) {
        // Railpack-specific step tracking
        // Railpack has ~26 steps total
        // Steps 1-4: Image resolution and context loading (14-18%)
        // Steps 5-13: Setup and mise installation (18-30%)
        // Step 14: Copy files (30%)
        // Step 15: Build (30-50%)
        // Steps 16-24: Copy and setup (50-65%)
        // Step 25: Merge (65%)
        // Step 26: Export (65-70%)
        let stepProgress = 14;
        if (stepNum <= 4) {
          // Image resolution and context loading (#1-#4): 14-18%
          stepProgress = 14 + Math.floor((stepNum / 4) * 4);
        } else if (stepNum <= 13) {
          // Setup and mise installation (#5-#13): 18-30%
          stepProgress = 18 + Math.floor(((stepNum - 4) / 9) * 12);
        } else if (stepNum === 14) {
          // Copy files (#14): 30%
          stepProgress = 30;
        } else if (stepNum === 15) {
          // Build step (#15): 35-45% (build can take time, so we'll update as build progresses)
          stepProgress = 40;
        } else if (stepNum <= 24) {
          // Copy and setup (#16-#24): 40-60% (more gradual progression)
          // Each step adds ~2.2% to prevent jumps
          stepProgress = 40 + Math.floor(((stepNum - 15) / 9) * 20);
        } else if (stepNum === 25) {
          // Merge (#25): 62% (smooth transition from copy steps)
          stepProgress = 62;
        } else {
          // Export (#26+): 65-70%
          stepProgress = Math.min(70, 65 + Math.floor(((stepNum - 25) / 1) * 5));
        }
        
        if (stepProgress > targetProgress.value) {
          targetProgress.value = stepProgress;
          
          // Set phase based on step number
          let phaseName = `Railpack: Step ${stepNum}`;
          if (stepNum <= 4) {
            phaseName = "Railpack: Initializing build";
          } else if (stepNum <= 13) {
            phaseName = "Railpack: Setting up build environment";
          } else if (stepNum === 14) {
            phaseName = "Railpack: Copying application files";
          } else if (stepNum === 15) {
            phaseName = "Railpack: Building application";
          } else if (stepNum <= 24) {
            phaseName = "Railpack: Copying files";
          } else if (stepNum === 25) {
            phaseName = "Railpack: Merging layers";
          } else {
            phaseName = "Railpack: Exporting image";
          }
          
          // Always update phase to ensure Railpack detection works for future steps
          currentPhase.value = phaseName;
          animateProgress();
        }
      } else {
        // Generic Docker step (early Nixpacks setup steps)
        // Early Docker steps (#0-#5) are setup: 14-18%
        // Nixpacks has ~10-15 total steps
        let stepProgress = 14;
        if (stepNum <= 5) {
          // Setup steps (#0-#5): 14-18%
          stepProgress = 14 + Math.floor((stepNum / 5) * 4);
        } else if (stepNum <= 10) {
          // Mid steps (#6-#10): 18-35%
          stepProgress = 18 + Math.floor(((stepNum - 5) / 5) * 17);
        } else if (stepNum <= 15) {
          // Late steps (#11-#15): 35-60%
          stepProgress = 35 + Math.floor(((stepNum - 10) / 5) * 25);
        } else {
          // Very late steps (#16+): 60-70%
          stepProgress = Math.min(70, 60 + Math.floor(((stepNum - 15) / 10) * 10));
        }
        
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
    const imageDownloadMatch = logLine.match(/sha256:.*?(\d+(?:\.\d+)?)\s*(MB|GB|KB|B|MiB|GiB|KiB)\s*\/\s*(\d+(?:\.\d+)?)\s*(MB|GB|KB|B|MiB|GiB|KiB)/i);
    if (imageDownloadMatch && imageDownloadMatch[1] && imageDownloadMatch[2] && imageDownloadMatch[3] && imageDownloadMatch[4]) {
      const downloaded = parseFloat(imageDownloadMatch[1]);
      const downloadedUnit = imageDownloadMatch[2];
      const total = parseFloat(imageDownloadMatch[3]);
      const totalUnit = imageDownloadMatch[4];
      
      // Convert to MB for comparison
      const toMB = (val: number, unit: string) => {
        switch (unit.toUpperCase()) {
          case "GB":
          case "GIB": return val * 1024;
          case "MB":
          case "MIB": return val;
          case "KB":
          case "KIB": return val / 1024;
          case "B": return val / (1024 * 1024);
          default: return val;
        }
      };
      
      const downloadedMB = toMB(downloaded, downloadedUnit);
      const totalMB = toMB(total, totalUnit);
      
      if (totalMB > 0) {
        // Image downloads happen early in Docker build (14-18% range)
        const downloadProgress = Math.min(18, 14 + Math.floor((downloadedMB / totalMB) * 4));
        if (downloadProgress > targetProgress.value) {
          targetProgress.value = downloadProgress;
          currentPhase.value = `Pulling images (${Math.floor((downloadedMB / totalMB) * 100)}%)`;
          animateProgress();
        }
      }
    }

    // Track file copying progress (incremental)
    // Note: Railpack copy steps are handled by step tracking above, so we skip them here
    const copyMatch = logLine.match(/#(\d+)\s+copy.*DONE|#(\d+)\s+copy.*done/i);
    if (copyMatch) {
      const stepNum = copyMatch[1] ? parseInt(copyMatch[1], 10) : (copyMatch[2] ? parseInt(copyMatch[2], 10) : 0);
      const isRailpack = currentPhase.value.includes("Railpack") || logLine.includes("[railpack]") || logLine.includes("railpack-builder") || logLine.includes("railpack-runtime");
      
      if (stepNum > 0) {
        // Skip Railpack copy steps - they're already handled by step tracking above
        // This prevents double-tracking and jumps
        if (isRailpack) {
          // Railpack copy steps are already tracked by step tracking, just ensure phase name is set
          if (stepNum >= 16 && stepNum <= 24 && !currentPhase.value.includes("Copying")) {
            currentPhase.value = "Railpack: Copying files";
          }
        } else if (currentPhase.value.includes("Nixpacks")) {
          // File copying typically happens at stages 9-10 for Nixpacks (58-70%)
          const copyProgress = Math.min(70, 58 + Math.floor(((stepNum - 8) / 2) * 12)); // 58-70% range
          if (copyProgress > targetProgress.value) {
            targetProgress.value = copyProgress;
            currentPhase.value = "Nixpacks: Copying files";
            animateProgress();
          }
        } else if (stepNum >= 15 && stepNum <= 21) {
          // Generic copy steps for other build strategies
          const copyProgress = Math.min(70, 58 + Math.floor(((stepNum - 15) / 6) * 12)); // 58-70% range
          if (copyProgress > targetProgress.value) {
            targetProgress.value = copyProgress;
            currentPhase.value = "Copying files";
            animateProgress();
          }
        }
      }
    }
  };

  // Start incremental progress when no logs are received
  const startIncrementalProgress = () => {
    // Clear any existing interval
    stopIncrementalProgress();
    
    // Don't start if already failed or completed
    if (isFailed.value || targetProgress.value >= 90) {
      return;
    }
    
    incrementalProgressIntervalId = setInterval(() => {
      // Only increment if we're streaming and no logs received recently
      if (!isStreaming.value || isFailed.value) {
        stopIncrementalProgress();
        return;
      }
      
      // If no logs received for 3 seconds, start slowly incrementing
      const timeSinceLastLog = Date.now() - lastLogUpdateTime.value;
      if (timeSinceLastLog >= 3000 && targetProgress.value < 90) {
        // Slowly increment: 0.1% per second (after 3 second delay)
        // This gives a sense of progress even during quiet periods
        const increment = Math.min(0.1, (90 - targetProgress.value) / 100);
        targetProgress.value = Math.min(90, targetProgress.value + increment);
        animateProgress();
      }
    }, 1000); // Check every second
  };
  
  const stopIncrementalProgress = () => {
    if (incrementalProgressIntervalId) {
      clearInterval(incrementalProgressIntervalId);
      incrementalProgressIntervalId = null;
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
    lastLogUpdateTime.value = Date.now();
    currentPhase.value = "Starting deployment...";
    streamController = new AbortController();
    
    // Start incremental progress tracking
    startIncrementalProgress();

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
      stopIncrementalProgress();
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
    stopIncrementalProgress();
    isStreaming.value = false;
  };

  const reset = () => {
    targetProgress.value = 0;
    progress.value = 0;
    isFailed.value = false;
    lastLogUpdateTime.value = Date.now();
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
    isFailed: computed(() => isFailed.value),
    startStreaming,
    stopStreaming,
    reset,
  };
}

