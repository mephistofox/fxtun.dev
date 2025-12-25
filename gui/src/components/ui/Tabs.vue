<script setup lang="ts">
import { cn } from '@/lib/utils'

interface Tab {
  value: string
  label: string
}

interface Props {
  modelValue: string
  tabs: Tab[]
  class?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <div :class="cn('inline-flex h-10 items-center justify-center rounded-md bg-muted p-1 text-muted-foreground', props.class)">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      type="button"
      :class="cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
        modelValue === tab.value
          ? 'bg-background text-foreground shadow-sm'
          : 'hover:bg-background/50'
      )"
      @click="emit('update:modelValue', tab.value)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>
