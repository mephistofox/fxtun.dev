<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  variant?: 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning' | 'info' | 'http' | 'tcp' | 'udp'
  pulse?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  pulse: false,
})

const variantClasses = {
  default: 'border-transparent bg-primary text-primary-foreground',
  secondary: 'border-transparent bg-secondary text-secondary-foreground',
  destructive: 'border-transparent bg-destructive text-destructive-foreground',
  outline: 'text-foreground border-border',
  success: 'border-transparent bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  warning: 'border-transparent bg-amber-500/10 text-amber-600 dark:text-amber-400',
  info: 'border-transparent bg-blue-500/10 text-blue-600 dark:text-blue-400',
  http: 'border-transparent bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  tcp: 'border-transparent bg-blue-500/10 text-blue-600 dark:text-blue-400',
  udp: 'border-transparent bg-purple-500/10 text-purple-600 dark:text-purple-400',
}

const classes = computed(() =>
  cn(
    'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors',
    variantClasses[props.variant],
    props.pulse && 'animate-pulse-soft',
    props.class
  )
)
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>
