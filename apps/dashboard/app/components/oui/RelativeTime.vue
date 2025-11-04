<template>
  <ClientOnly>
  <span>{{ formattedRelativeTime }}</span>
    <template #fallback>
      <span>{{ fallbackText }}</span>
    </template>
  </ClientOnly>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type RelativeTimeStyle = 'long' | 'short' | 'narrow'
type RelativeTimeNumeric = 'always' | 'auto'

interface Props {
  value: string | Date | number | null | undefined
  style?: RelativeTimeStyle
  numeric?: RelativeTimeNumeric
  locale?: string | string[]
}

const props = withDefaults(defineProps<Props>(), {
  style: 'long' as RelativeTimeStyle,
  numeric: 'auto' as RelativeTimeNumeric,
  locale: undefined,
})

const dateValue = computed<Date>(() => {
  if (!props.value) return new Date()
  if (props.value instanceof Date) return props.value
  if (typeof props.value === 'number') return new Date(props.value)
  return new Date(props.value)
})

// Fallback text for SSR - use a simple format that matches Intl.RelativeTimeFormat short style (without period)
const fallbackText = computed(() => {
  if (!props.value) return ''
  const date = dateValue.value
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.round(Math.abs(diffMs) / (1000 * 60))
  
  if (diffMins < 1) {
    return 'just now'
  }
  if (diffMins < 60) {
    // Match short style format: "49 min ago" (no period)
    return `${diffMins} min ago`
  }
  const diffHours = Math.round(Math.abs(diffMs) / (1000 * 60 * 60))
  if (diffHours < 24) {
    return `${diffHours} hr ago`
  }
  const diffDays = Math.round(Math.abs(diffMs) / (1000 * 60 * 60 * 24))
  if (diffDays === 1) {
    return '1 day ago'
  }
  return `${diffDays} days ago`
})

const formattedRelativeTime = computed(() => {
  const now = Date.now()
  const diffMilliseconds = dateValue.value.getTime() - now

  if (!Number.isFinite(diffMilliseconds)) {
    return ''
  }

  const thresholds: Array<{ unit: Intl.RelativeTimeFormatUnit; divisor: number; limit: number }> = [
    { unit: 'second', divisor: 1000, limit: 60 },
    { unit: 'minute', divisor: 60 * 1000, limit: 60 },
    { unit: 'hour', divisor: 60 * 60 * 1000, limit: 24 },
    { unit: 'day', divisor: 24 * 60 * 60 * 1000, limit: 7 },
    { unit: 'week', divisor: 7 * 24 * 60 * 60 * 1000, limit: 4 },
    { unit: 'month', divisor: 30 * 24 * 60 * 60 * 1000, limit: 12 },
    { unit: 'year', divisor: 365 * 24 * 60 * 60 * 1000, limit: Number.POSITIVE_INFINITY },
  ]

  let unit: Intl.RelativeTimeFormatUnit = 'second'
  let value = diffMilliseconds / 1000

  for (const threshold of thresholds) {
    const relativeValue = diffMilliseconds / threshold.divisor
    if (Math.abs(relativeValue) < threshold.limit) {
      unit = threshold.unit
      value = relativeValue
      break
    }
  }

  const formatter = new Intl.RelativeTimeFormat(props.locale, {
    style: props.style,
    numeric: props.numeric,
  })

  return formatter.format(Math.round(value), unit)
})
</script>

