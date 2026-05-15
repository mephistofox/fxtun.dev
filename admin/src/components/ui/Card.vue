<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string | string[]
  variant?: 'default' | 'glass' | 'glass-glow'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
})

const variantClasses = {
  default: 'rounded-xl border bg-card text-card-foreground shadow-sm',
  glass: 'glass-card text-card-foreground',
  'glass-glow': 'glass-card-glow text-card-foreground',
}

const classes = computed(() => cn(variantClasses[props.variant], props.class))
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>
