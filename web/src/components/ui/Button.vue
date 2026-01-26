<script setup lang="ts">
import { computed } from 'vue'
import { type ClassValue } from 'clsx'
import { cn } from '@/lib/utils'

interface Props {
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'success' | 'glow'
  size?: 'default' | 'sm' | 'lg' | 'icon' | 'xs'
  disabled?: boolean
  loading?: boolean
  class?: ClassValue
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  size: 'default',
  disabled: false,
  loading: false,
})

const variantClasses = {
  default: 'bg-primary text-primary-foreground hover:bg-primary/90 hover:shadow-lg hover:shadow-primary/25 active:scale-[0.98]',
  destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90 hover:shadow-lg hover:shadow-destructive/25 active:scale-[0.98]',
  outline: 'border border-input bg-background hover:bg-accent/10 hover:border-primary/50 hover:text-foreground active:scale-[0.98]',
  secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80 active:scale-[0.98]',
  ghost: 'hover:bg-accent/10 hover:text-accent-foreground active:scale-[0.98]',
  link: 'text-primary underline-offset-4 hover:underline',
  success: 'bg-type-http text-white hover:bg-type-http/90 hover:shadow-lg hover:shadow-type-http/25 active:scale-[0.98]',
  glow: 'btn-glow text-primary-foreground font-semibold active:scale-[0.98]',
}

const sizeClasses = {
  default: 'h-10 px-4 py-2',
  xs: 'h-7 rounded-md px-2 text-xs',
  sm: 'h-9 rounded-md px-3',
  lg: 'h-11 rounded-md px-8 text-base',
  icon: 'h-10 w-10',
}

const classes = computed(() =>
  cn(
    'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg text-sm font-medium ring-offset-background transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
    variantClasses[props.variant],
    sizeClasses[props.size],
    props.class
  )
)
</script>

<template>
  <button :class="classes" :disabled="disabled || loading">
    <svg
      v-if="loading"
      class="h-4 w-4 animate-spin"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
    <slot />
  </button>
</template>
