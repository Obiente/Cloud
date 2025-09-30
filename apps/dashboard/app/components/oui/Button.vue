<template>
  <button
    :class="[
      // Base button composition (reusable)
      'oui-btn-base oui-text',
      // Size utilities applied directly  
      {
        'px-2 py-1 text-xs': size === 'xs',
        'px-3 py-1.5 text-sm': size === 'sm', 
        'px-4 py-2 text-sm': size === 'md',
        'px-6 py-3 text-base': size === 'lg',
        'px-8 py-4 text-lg': size === 'xl',
      },
      // Variant styles applied directly
      {
        'bg-surface-raised text-primary hover:bg-interactive-hover active:bg-interactive-active': variant === 'primary',
        'bg-secondary text-primary hover:bg-secondary-dark active:bg-secondary-darker': variant === 'secondary',
        'bg-transparent border border-primary text-primary hover:bg-primary hover:text-white': variant === 'outline',
        'bg-transparent text-primary hover:bg-primary/10 active:bg-primary/20': variant === 'ghost',
        'bg-danger text-white hover:bg-danger-dark active:bg-danger-darker': variant === 'danger',
      },
      {
        'opacity-50 cursor-not-allowed': disabled || loading,
        'animate-pulse': loading,
      }
    ]"
    :disabled="disabled || loading"
    v-bind="$attrs"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger'
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  disabled?: boolean
  loading?: boolean
}

withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
})

defineOptions({
  inheritAttrs: false
})
</script>