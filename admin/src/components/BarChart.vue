<script setup lang="ts">
import { computed } from 'vue'

interface DataPoint {
  date: string
  value: number
}

interface Props {
  data: DataPoint[]
  loading?: boolean
  color?: string
  formatValue?: (v: number) => string
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  color: 'hsl(75 100% 50% / 0.8)',
})

const maxVal = computed(() => {
  if (!props.data || props.data.length === 0) return 1
  return Math.max(...props.data.map(p => p.value), 1)
})

const svgWidth = computed(() => (props.data?.length ?? 0) * 24 + 8)

function barHeight(value: number): number {
  return Math.max((value / maxVal.value) * 180, 2)
}

function barY(value: number): number {
  return 200 - barHeight(value)
}

function tooltip(point: DataPoint): string {
  const val = props.formatValue ? props.formatValue(point.value) : String(point.value)
  return `${point.date}: ${val}`
}
</script>

<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center h-48">
      <svg
        class="h-6 w-6 animate-spin text-primary"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </div>

    <!-- Empty -->
    <div v-else-if="!data || data.length === 0" class="flex items-center justify-center h-48 text-sm text-muted-foreground">
      Нет данных
    </div>

    <!-- Chart -->
    <div v-else class="relative">
      <svg :viewBox="`0 0 ${svgWidth} 200`" class="w-full h-48" preserveAspectRatio="none">
        <rect
          v-for="(point, i) in data"
          :key="i"
          :x="i * 24 + 4"
          :y="barY(point.value)"
          :width="18"
          :height="barHeight(point.value)"
          :fill="color"
          rx="3"
          class="transition-all duration-300"
        >
          <title>{{ tooltip(point) }}</title>
        </rect>
      </svg>
    </div>
  </div>
</template>
