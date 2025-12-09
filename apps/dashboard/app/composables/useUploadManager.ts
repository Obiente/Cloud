import { ref, computed, watchEffect } from "vue";

/**
 * Per-file progress tracking
 */
export interface UploadProgressEntry {
  bytesUploaded?: number;
  totalBytes?: number;
  percentComplete?: number;
  speedBytesPerSec?: number;
  etaSeconds?: number;
  chunkIndex?: number;
  totalChunks?: number;
}

/**
 * Centralized upload manager for tracking progress, speed, and ETA across all uploads.
 * This composable manages aggregate metrics that work across all upload locations (FileUploadZone, GameServerFileUploader, etc.)
 * 
 * Features:
 * - Stable progress tracking that doesn't reset between batches
 * - Smoothed network speed (10-sample buffer) for stable ETA calculations
 * - Adaptive chunk size and concurrency recommendations based on current network conditions
 */
export function useUploadManager() {
  // ========== State Management ==========
  
  /** Progress map: fileName -> progress entry */
  const progressMap = ref<Record<string, UploadProgressEntry>>({});
  
  /** Total bytes to upload (stable across batches) */
  const totalBytesToUpload = ref(0);
  
  /** Max bytes ever observed (prevents denominator from shrinking when files complete) */
  const maxObservedTotal = ref(0);
  
  /** Speed tracking for derived speed calculation */
  const lastBytesSnapshot = ref(0);
  const lastTimestamp = ref<number | null>(null);
  const derivedSpeed = ref(0);
  const derivedSpeedSamples = ref<number[]>([]);
  
  /** Larger buffer for smoothed speed (10-sample window) */
  const smoothedSpeedBuffer = ref<number[]>([]);
  
  /** Smoothed ETA for stable time estimates */
  const smoothedEta = ref<number | undefined>(undefined);

  // ========== Computed Properties ==========

  /**
   * Overall progress percentage (0-100)
   */
  const overallProgress = computed(() => {
    const items = Object.values(progressMap.value);

    const loadedInMap = items.reduce(
      (sum, it) => sum + Math.min(it.bytesUploaded || 0, it.totalBytes || 0),
      0
    );

    const totalInMap = items.reduce(
      (sum, it) => sum + (it.totalBytes || 0),
      0
    );

    const inferredTotal = totalInMap;
    const grandTotal =
      totalBytesToUpload.value && totalBytesToUpload.value > 0
        ? totalBytesToUpload.value
        : inferredTotal;

    const stableTotal = Math.max(grandTotal, maxObservedTotal.value || 0);
    const totalLoaded = loadedInMap;
    const safeLoaded = Math.min(totalLoaded, stableTotal);

    if (stableTotal > 0) {
      const percent = (safeLoaded / stableTotal) * 100;
      return Math.round(percent);
    }

    if (items.length === 0) return 0;

    // Fallback: average of per-file percentages
    const avgPercent =
      items.reduce((sum, it) => sum + (it.percentComplete || 0), 0) / items.length;
    return Math.round(avgPercent);
  });

  /**
   * Clamped progress (0-100)
   */
  const overallProgressClamped = computed(() => {
    const num = overallProgress.value;
    return Math.min(100, Math.max(0, num));
  });

  /**
   * Derive overall speed if per-file speeds are missing
   */
  watchEffect(() => {
    const items = Object.values(progressMap.value);
    const loadedInMap = items.reduce(
      (sum, it) => sum + Math.min(it.bytesUploaded || 0, it.totalBytes || 0),
      0
    );
    const totalLoaded = loadedInMap;
    const now = performance.now();
    
    if (lastTimestamp.value !== null) {
      const dtSeconds = (now - lastTimestamp.value) / 1000;
      if (dtSeconds > 0) {
        const deltaBytes = totalLoaded - lastBytesSnapshot.value;
        const instSpeed = Math.max(0, deltaBytes / dtSeconds);
        const samples = derivedSpeedSamples.value.slice(-4);
        samples.push(instSpeed);
        derivedSpeedSamples.value = samples;
        const avg =
          samples.reduce((acc, v) => acc + v, 0) /
          (samples.length || 1);
        derivedSpeed.value = avg;
      }
    }
    
    lastBytesSnapshot.value = totalLoaded;
    lastTimestamp.value = now;
  });

  /**
   * Overall upload speed (sum of per-file speeds, or derived speed as fallback)
   */
  const overallSpeed = computed(() => {
    const items = Object.values(progressMap.value);
    const summedSpeeds = items.reduce(
      (sum, it) => sum + (it.speedBytesPerSec || 0),
      0
    );
    // Prefer summed speeds if present; otherwise fall back to derived speed
    return summedSpeeds > 0 ? summedSpeeds : derivedSpeed.value;
  });

  /**
   * Smoothed network speed (averaged over 10-sample buffer)
   * More stable for ETA and chunk size recommendations
   */
  const smoothedNetworkSpeed = computed(() => {
    const items = Object.values(progressMap.value);
    const summedSpeeds = items.reduce(
      (sum, it) => sum + (it.speedBytesPerSec || 0),
      0
    );
    const currentSpeed = summedSpeeds > 0 ? summedSpeeds : derivedSpeed.value;
    
    // Keep a 10-sample buffer for smoother speed averaging
    if (currentSpeed > 0) {
      smoothedSpeedBuffer.value = smoothedSpeedBuffer.value.slice(-9);
      smoothedSpeedBuffer.value.push(currentSpeed);
    }
    
    if (smoothedSpeedBuffer.value.length === 0) return 0;
    return smoothedSpeedBuffer.value.reduce((a, b) => a + b, 0) / smoothedSpeedBuffer.value.length;
  });

  /**
   * Overall ETA in seconds (using smoothed speed for stability)
   */
  const overallEtaSeconds = computed(() => {
    const items = Object.values(progressMap.value);
    const loadedInMap = items.reduce(
      (sum, it) => sum + Math.min(it.bytesUploaded || 0, it.totalBytes || 0),
      0
    );
    const totalInMap = items.reduce(
      (sum, it) => sum + (it.totalBytes || 0),
      0
    );

    const inferredTotal = totalInMap;
    const grandTotal =
      totalBytesToUpload.value && totalBytesToUpload.value > 0
        ? totalBytesToUpload.value
        : inferredTotal;

    const stableTotal = Math.max(grandTotal, maxObservedTotal.value || 0);
    const totalLoaded = loadedInMap;
    const remaining = Math.max(0, stableTotal - totalLoaded);

    // Use smoothed speed for more stable ETA
    const speed = smoothedNetworkSpeed.value;
    if (remaining === 0) return 0;
    if (speed === 0) return undefined;

    const eta = remaining / speed;
    // Smooth ETA more aggressively to reduce drastic jumps (0.85 weight to previous)
    smoothedEta.value =
      smoothedEta.value === undefined
        ? eta
        : smoothedEta.value * 0.85 + eta * 0.15;

    return Math.round(smoothedEta.value);
  });

  /**
   * Recommendation for chunk size based on current network speed
   * Dynamically scales from 1MB to 100MB+ based on measured bandwidth
   * Returns chunk size in bytes
   */
  const recommendedChunkSize = computed(() => {
    const speed = smoothedNetworkSpeed.value;
    
    // Target: chunk upload time ~2 seconds for optimal throughput with progress updates
    const targetChunkTime = 2; // seconds
    const recommendedBytes = speed * targetChunkTime;
    
    // Clamp to reasonable bounds
    const MIN_CHUNK = 1 * 1024 * 1024;      // 1 MB minimum
    const MAX_CHUNK = 100 * 1024 * 1024;    // 100 MB maximum
    
    return Math.max(MIN_CHUNK, Math.min(MAX_CHUNK, recommendedBytes));
  });

  /**
   * Recommendation for concurrent uploads based on network speed
   * Returns number of files to upload concurrently (1-8 range)
   */
  const recommendedConcurrency = computed(() => {
    const speed = smoothedNetworkSpeed.value;
    
    // Very fast (>100 MB/s), can handle many concurrent uploads
    if (speed > 100 * 1024 * 1024) return 8;
    // Very fast (>50 MB/s)
    if (speed > 50 * 1024 * 1024) return 6;
    // Fast (>20 MB/s)
    if (speed > 20 * 1024 * 1024) return 5;
    // Moderate-fast (>10 MB/s)
    if (speed > 10 * 1024 * 1024) return 4;
    // Moderate (>5 MB/s)
    if (speed > 5 * 1024 * 1024) return 3;
    // Slower (<5 MB/s)
    if (speed > 1 * 1024 * 1024) return 2;
    // Very slow
    return 1;
  });

  // ========== Public Methods ==========

  /**
   * Update a file's progress entry
   */
  const updateProgress = (fileName: string, progress: UploadProgressEntry) => {
    progressMap.value[fileName] = progress;
    
    // Track max observed total for denominator stability
    if (progress.totalBytes && progress.totalBytes > maxObservedTotal.value) {
      maxObservedTotal.value = progress.totalBytes;
    }
  };

  /**
   * Remove a file from progress tracking
   */
  const removeProgress = (fileName: string) => {
    delete progressMap.value[fileName];
  };

  /**
   * Clear all progress
   */
  const clearProgress = () => {
    progressMap.value = {};
    totalBytesToUpload.value = 0;
    maxObservedTotal.value = 0;
    lastBytesSnapshot.value = 0;
    lastTimestamp.value = null;
    derivedSpeed.value = 0;
    derivedSpeedSamples.value = [];
    smoothedSpeedBuffer.value = [];
    smoothedEta.value = undefined;
  };

  /**
   * Reset for a new batch while preserving speed history
   */
  const resetForNewBatch = () => {
    progressMap.value = {};
    // Note: maxObservedTotal and smoothedSpeedBuffer remain to preserve stability across batches
    lastBytesSnapshot.value = 0;
    lastTimestamp.value = null;
  };

  /**
   * Set total bytes to upload (e.g., sum of all file sizes)
   */
  const setTotalBytesToUpload = (bytes: number) => {
    totalBytesToUpload.value = bytes;
    // Update max observed if larger
    if (bytes > maxObservedTotal.value) {
      maxObservedTotal.value = bytes;
    }
  };

  /**
   * Get smoothed network speed (for parent component optimization)
   */
  const getSmoothedNetworkSpeed = () => smoothedNetworkSpeed.value;

  return {
    // State
    progressMap,
    totalBytesToUpload,
    maxObservedTotal,
    smoothedSpeedBuffer,
    smoothedEta,

    // Computed metrics
    overallProgress,
    overallProgressClamped,
    overallSpeed,
    smoothedNetworkSpeed,
    overallEtaSeconds,

    // Recommendations
    recommendedChunkSize,
    recommendedConcurrency,

    // Methods
    updateProgress,
    removeProgress,
    clearProgress,
    resetForNewBatch,
    setTotalBytesToUpload,
    getSmoothedNetworkSpeed,
  };
}
