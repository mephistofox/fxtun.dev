<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  variant?: 'default' | 'success' | 'destructive' | 'warning' | 'info' | 'accent' | 'outline'
  size?: 'sm' | 'default'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  size: 'default',
})

const variantClasses: Record<NonNullable<Props['variant']>, string> = {
  default: 'bg-secondary text-secondary-foreground border-secondary',
  success: 'bg-type-http/10 text-type-http border-type-http/20',
  destructive: 'bg-destructive/10 text-destructive border-destructive/20',
  warning: 'bg-[hsl(var(--warning)/0.1)] text-[hsl(var(--warning))] border-[hsl(var(--warning)/0.2)]',
  info: 'bg-type-tcp/10 text-type-tcp border-type-tcp/20',
  accent: 'bg-accent/10 text-accent border-accent/20',
  outline: 'border border-border text-muted-foreground bg-transparent',
}

const sizeClasses: Record<NonNullable<Props['size']>, string> = {
  sm: 'px-1.5 py-0 text-[10px]',
  default: 'px-2 py-0.5 text-xs',
}

const classes = computed(() =>
  cn(
    'inline-flex items-center rounded-md font-medium border whitespace-nowrap',
    variantClasses[props.variant!],
    sizeClasses[props.size!],
    props.class,
  ),
)
</script>

<template>
  <span :class="classes">
    <slot />
  </span>
</template>
