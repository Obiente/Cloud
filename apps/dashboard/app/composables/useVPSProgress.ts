import { ref, computed, onUnmounted, type ComputedRef } from "vue";
import { VPSService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

export interface VPSProgressOptions {
  vpsId: string;
  organizationId: string;
}

/**
 * Tracks VPS provisioning progress by analyzing log patterns
 */
export function useVPSProgress(options: VPSProgressOptions) {
  const client = useConnectClient(VPSService);
  const targetProgress = ref(0); // Target progress value from log analysis
  const progress = ref(0); // Actual displayed progress (smoothly animated)
  const currentPhase = ref<string>("Starting server setup...");
  const isStreaming = ref(false);
  let streamController: AbortController | null = null;
  let animationFrameId: number | null = null;
  let incrementalProgressIntervalId: ReturnType<typeof setInterval> | null = null;
  let lastLogUpdateTime = ref(Date.now());
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  let reconnectAttempts = 0;
  const MAX_RECONNECT_ATTEMPTS = 5;
  const RECONNECT_DELAY = 2000;
  let isReconnecting = false;

  // VPS provisioning phase patterns and their progress percentages
  // Order matters: more specific patterns should come first
  const provisioningPhases = [
    {
      pattern: /Starting server setup/i,
      progress: 5,
      phase: "Starting server setup",
    },
    {
      pattern: /Setting up secure access/i,
      progress: 10,
      phase: "Setting up secure access",
    },
    {
      pattern: /Secure access configured/i,
      progress: 15,
      phase: "Secure access configured",
    },
    {
      pattern: /Assigning network address/i,
      progress: 20,
      phase: "Assigning network address",
    },
    {
      pattern: /Network address assigned/i,
      progress: 25,
      phase: "Network address assigned",
    },
    {
      pattern: /Creating server/i,
      progress: 30,
      phase: "Creating server",
    },
    {
      pattern: /Selecting server location/i,
      progress: 35,
      phase: "Selecting server location",
    },
    {
      pattern: /Server location selected/i,
      progress: 40,
      phase: "Server location selected",
    },
    {
      pattern: /Preparing storage/i,
      progress: 45,
      phase: "Preparing storage",
    },
    {
      pattern: /Storage ready/i,
      progress: 50,
      phase: "Storage ready",
    },
    {
      pattern: /Setting up operating system/i,
      progress: 55,
      phase: "Setting up operating system",
    },
    {
      pattern: /Operating system installed/i,
      progress: 70,
      phase: "Operating system installed",
    },
    {
      pattern: /Configuring security settings/i,
      progress: 80,
      phase: "Configuring security settings",
    },
    {
      pattern: /Security configured/i,
      progress: 85,
      phase: "Security configured",
    },
    {
      pattern: /Starting server/i,
      progress: 90,
      phase: "Starting server",
    },
    {
      pattern: /Server started successfully/i,
      progress: 95,
      phase: "Server started successfully",
    },
    {
      pattern: /Server setup complete/i,
      progress: 100,
      phase: "Server setup complete",
    },
  ];

  const isFailed = ref(false);

  const updateProgressFromLog = (logLine: string) => {
    // Update last log update time
    lastLogUpdateTime.value = Date.now();

    // Check for failure patterns
    if (
      /error|failed|failure/i.test(logLine) &&
      !/warning/i.test(logLine) // Warnings are OK
    ) {
      isFailed.value = true;
      currentPhase.value = "Setup failed";
      targetProgress.value = Math.max(targetProgress.value, 0);
      return;
    }

    // Check for success completion
    if (/Server setup complete/i.test(logLine)) {
      targetProgress.value = 100;
      currentPhase.value = "Server setup complete";
      return;
    }

    // Match log line against phase patterns
    for (const phase of provisioningPhases) {
      if (phase.pattern.test(logLine)) {
        if (phase.progress > targetProgress.value) {
          targetProgress.value = phase.progress;
          currentPhase.value = phase.phase;
        }
        break; // Stop after first match
      }
    }
  };

  // Smooth animation of progress bar
  const animateProgress = () => {
    if (targetProgress.value > progress.value) {
      // Smoothly animate progress towards target
      const diff = targetProgress.value - progress.value;
      const step = Math.max(1, diff * 0.1); // 10% of remaining distance per frame
      progress.value = Math.min(targetProgress.value, progress.value + step);
    } else if (targetProgress.value < progress.value) {
      // Allow progress to decrease if needed (shouldn't happen normally)
      progress.value = targetProgress.value;
    }

    if (Math.abs(targetProgress.value - progress.value) > 0.1) {
      animationFrameId = requestAnimationFrame(animateProgress);
    }
  };

  // Incremental progress when no log updates (prevents progress from stalling)
  const startIncrementalProgress = () => {
    if (incrementalProgressIntervalId) {
      return;
    }

    incrementalProgressIntervalId = setInterval(() => {
      const timeSinceLastUpdate = Date.now() - lastLogUpdateTime.value;
      const staleThreshold = 10000; // 10 seconds

      // If we haven't received logs in a while but are still streaming, increment slowly
      if (timeSinceLastUpdate > staleThreshold && isStreaming.value && !isFailed.value) {
        // Only increment if we're not at 100% and not too close to target
        if (targetProgress.value < 95 && progress.value < targetProgress.value - 5) {
          // Very slow increment (0.1% per interval)
          targetProgress.value = Math.min(95, targetProgress.value + 0.1);
          animateProgress();
        }
      }
    }, 2000); // Check every 2 seconds
  };

  const stopIncrementalProgress = () => {
    if (incrementalProgressIntervalId) {
      clearInterval(incrementalProgressIntervalId);
      incrementalProgressIntervalId = null;
    }
  };

  const scheduleReconnect = () => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      console.error("[useVPSProgress] Max reconnect attempts reached, stopping stream");
      isStreaming.value = false;
      isReconnecting = false;
      return;
    }
    reconnectAttempts++;
    const delay = Math.min(RECONNECT_DELAY * Math.pow(2, reconnectAttempts - 1), 30000);
    isReconnecting = true;
    reconnectTimeout = setTimeout(async () => {
      reconnectTimeout = null;
      // Only reconnect if we're still supposed to be streaming
      // (isStreaming might be false if explicitly stopped, but isReconnecting indicates we want to reconnect)
      if (!isReconnecting) {
        return;
      }
      console.log(`[useVPSProgress] Attempting to reconnect stream (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`);
      await startStreamingInternal(true); // Pass true to indicate this is a reconnect
    }, delay);
  };

  const startStreamingInternal = async (isReconnect = false) => {
    if (isStreaming.value || (streamController && !isReconnect)) {
      return;
    }

    // Only reset progress if this is a fresh start, not a reconnect
    if (!isReconnect) {
      targetProgress.value = 0;
      progress.value = 0;
      isFailed.value = false;
      currentPhase.value = "Starting server setup...";
      reconnectAttempts = 0;
    }

    isStreaming.value = true;
    isReconnecting = false;
    lastLogUpdateTime.value = Date.now();
    
    // Abort previous stream if reconnecting
    if (streamController && isReconnect) {
      streamController.abort();
    }
    
    streamController = new AbortController();
    
    // Start incremental progress tracking
    startIncrementalProgress();

    try {
      const stream = await (client as any).streamVPSLogs(
        {
          organizationId: options.organizationId,
          vpsId: options.vpsId,
        },
        { signal: streamController.signal }
      );

      // Reset reconnect attempts on successful connection
      reconnectAttempts = 0;

      for await (const update of stream) {
        if (streamController?.signal.aborted) {
          break;
        }
        // Ignore empty lines (keepalive heartbeats)
        if (update.line && update.line.trim() !== "") {
          updateProgressFromLog(update.line);
          animateProgress();
        }
      }
    } catch (err: any) {
      if (err.name === "AbortError" || streamController?.signal.aborted) {
        return;
      }
      // Suppress benign stream errors
      const isBenignError =
        err.message?.toLowerCase().includes("missing trailer") ||
        err.message?.toLowerCase().includes("trailer") ||
        err.code === "unknown";

      if (!isBenignError) {
        console.error("Failed to stream VPS logs for progress:", err);
      }
    } finally {
      const wasAborted = streamController?.signal.aborted || false;
      const shouldReconnect = isReconnecting || (!wasAborted && isStreaming.value);
      streamController = null;
      stopIncrementalProgress();
      
      // Only attempt to reconnect if the stream wasn't explicitly aborted
      // and we're still supposed to be streaming (or were in the process of reconnecting)
      if (shouldReconnect) {
        // Stream ended unexpectedly, attempt to reconnect
        // Don't set isStreaming to false yet - let reconnect handle it
        scheduleReconnect();
      } else {
        isStreaming.value = false;
        isReconnecting = false;
      }
    }
  };

  const startStreaming = async () => {
    await startStreamingInternal(false);
  };

  const stopStreaming = () => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }
    reconnectAttempts = 0;
    isReconnecting = false;
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
    currentPhase.value = "Starting server setup...";
    stopStreaming();
  };

  // Cleanup on unmount
  onUnmounted(() => {
    stopStreaming();
  });

  const progressComputed = computed(() => Math.round(progress.value));
  const currentPhaseComputed = computed(() => currentPhase.value);
  const isStreamingComputed = computed(() => isStreaming.value);
  const isFailedComputed = computed(() => isFailed.value);

  return {
    progress: progressComputed,
    currentPhase: currentPhaseComputed,
    isStreaming: isStreamingComputed,
    isFailed: isFailedComputed,
    startStreaming,
    stopStreaming,
    reset,
  };
}

