<script setup lang="ts">
import { ProgressRoot, ProgressIndicator } from 'radix-vue'
import { cn } from '@/lib/utils'
import { computed } from 'vue'

interface Props {
  modelValue?: number
  max?: number
  class?: string
  indicatorClass?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: 0,
  max: 100,
})

const percentage = computed(() => {
  return Math.min(100, Math.max(0, (props.modelValue / props.max) * 100))
})
</script>

<template>
  <ProgressRoot
    :model-value="modelValue"
    :max="max"
    :class="cn(
      'relative h-2 w-full overflow-hidden rounded-full bg-primary/20',
      props.class
    )"
  >
    <ProgressIndicator
      :class="cn(
        'h-full w-full flex-1 bg-primary transition-all duration-300 ease-in-out',
        indicatorClass
      )"
      :style="{ transform: `translateX(-${100 - percentage}%)` }"
    />
  </ProgressRoot>
</template>
