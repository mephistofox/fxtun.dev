<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

type Status = 'connected' | 'connecting' | 'disconnected' | 'error'

interface Props {
  status: Status
  size?: 'sm' | 'md' | 'lg'
  pulse?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  pulse: true,
})

const sizeClasses = {
  sm: 'h-2 w-2',
  md: 'h-2.5 w-2.5',
  lg: 'h-3 w-3',
}

const statusClasses = {
  connected: 'bg-emerald-500',
  connecting: 'bg-amber-500',
  disconnected: 'bg-slate-400 dark:bg-slate-600',
  error: 'bg-red-500',
}

const pingClasses = {
  connected: 'bg-emerald-400',
  connecting: 'bg-amber-400',
  disconnected: '',
  error: 'bg-red-400',
}

const shouldPulse = computed(() => {
  return props.pulse && (props.status === 'connected' || props.status === 'connecting')
})
</script>

<template>
  <span :class="cn('relative inline-flex', sizeClasses[size], props.class)">
    <span
      v-if="shouldPulse"
      :class="cn(
        'absolute inline-flex h-full w-full animate-ping rounded-full opacity-75',
        pingClasses[status]
      )"
    />
    <span
      :class="cn(
        'relative inline-flex h-full w-full rounded-full',
        statusClasses[status]
      )"
    />
  </span>
</template>
