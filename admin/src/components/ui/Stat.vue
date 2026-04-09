<script setup lang="ts">
import { computed, type Component } from 'vue'
import { TrendingUp, TrendingDown } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

interface Props {
  label: string
  value: string | number
  icon?: Component
  trend?: 'up' | 'down'
  trendValue?: string
  class?: string
}

const props = defineProps<Props>()

const trendColor = computed(() => {
  if (props.trend === 'up') return 'text-type-http'
  if (props.trend === 'down') return 'text-destructive'
  return ''
})

const trendBg = computed(() => {
  if (props.trend === 'up') return 'bg-type-http/10'
  if (props.trend === 'down') return 'bg-destructive/10'
  return ''
})
</script>

<template>
  <div
    :class="cn(
      'relative overflow-hidden rounded-xl border border-border p-6 backdrop-blur-xl',
      'bg-card transition-all duration-300 hover:border-primary/20',
      props.class,
    )"
  >
    <div class="flex items-start justify-between">
      <div class="flex-1 min-w-0">
        <p class="text-sm text-muted-foreground truncate">{{ label }}</p>
        <p class="mt-2 text-2xl font-display font-bold text-foreground truncate">{{ value }}</p>
        <div
          v-if="trend && trendValue"
          class="mt-2 inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-medium"
          :class="[trendColor, trendBg]"
        >
          <TrendingUp v-if="trend === 'up'" class="h-3 w-3" />
          <TrendingDown v-if="trend === 'down'" class="h-3 w-3" />
          <span>{{ trendValue }}</span>
        </div>
      </div>
      <div
        v-if="icon"
        class="flex-shrink-0 flex items-center justify-center rounded-lg bg-primary/10 p-2.5 text-primary"
      >
        <component :is="icon" class="h-5 w-5" />
      </div>
    </div>
  </div>
</template>
