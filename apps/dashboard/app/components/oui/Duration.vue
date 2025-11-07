<template>
  <span>{{ formattedDuration }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type UnitDisplay = 'narrow' | 'short' | 'long'

interface Props {
  value: number | bigint | string | null | undefined
  unitDisplay?: UnitDisplay
  locale?: string | string[]
}

const props = withDefaults(defineProps<Props>(), {
  unitDisplay: 'short' as UnitDisplay,
  locale: undefined,
})

const msValue = computed<number>(() => {
  if (!props.value) return 0
  if (typeof props.value === 'number') return props.value
  if (typeof props.value === 'bigint') return Number(props.value)
  if (typeof props.value === 'string') {
    const parsed = Number.parseInt(props.value, 10)
    return Number.isNaN(parsed) ? 0 : parsed
  }
  return 0
})

const formattedDuration = computed(() => {
  const ms = msValue.value
  
  // Less than 1 second - show milliseconds
  if (ms < 1000) {
    return `${ms}ms`
  }
  
  // Try to use Intl.DurationFormat if available (Chrome 119+, Node 22+)
  if (typeof Intl !== 'undefined' && 'DurationFormat' in Intl) {
    try {
      const seconds = Math.floor(ms / 1000)
      const minutes = Math.floor(seconds / 60)
      const hours = Math.floor(minutes / 60)
      
      const duration: Record<string, number> = {}
      
      if (hours > 0) {
        duration.hours = hours
      }
      if (minutes % 60 > 0) {
        duration.minutes = minutes % 60
      }
      if (seconds % 60 > 0 && hours === 0) {
        // Only show seconds if less than 1 hour
        duration.seconds = seconds % 60
      }
      
      const formatter = new (Intl as any).DurationFormat(props.locale, {
        style: props.unitDisplay,
      })
      
      return formatter.format(duration)
    } catch (e) {
      // Fall through to manual formatting
    }
  }
  
  // Fallback: Manual formatting
  const seconds = ms / 1000
  
  // Less than 1 minute - show seconds with appropriate precision
  if (seconds < 60) {
    if (seconds < 10) {
      return `${seconds.toFixed(2)}s`
    }
    return `${seconds.toFixed(1)}s`
  }
  
  // 1 minute or more - show minutes and seconds
  const minutes = Math.floor(ms / 60000)
  const remainingSeconds = Math.floor((ms % 60000) / 1000)
  
  if (remainingSeconds === 0) {
    return `${minutes}m`
  }
  return `${minutes}m ${remainingSeconds}s`
})
</script>

