<script setup lang="ts">
import { cn } from '@/lib/utils'

interface Tab {
  key: string
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

function selectTab(key: string) {
  emit('update:modelValue', key)
}
</script>

<template>
  <div :class="cn('w-full', props.class)">
    <div class="flex border-b border-border">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        class="relative px-4 py-2.5 text-sm font-medium transition-colors duration-200 outline-none"
        :class="[
          modelValue === tab.key
            ? 'text-primary'
            : 'text-muted-foreground hover:text-foreground',
        ]"
        @click="selectTab(tab.key)"
      >
        {{ tab.label }}
        <span
          v-if="modelValue === tab.key"
          class="absolute bottom-0 left-0 right-0 h-0.5 bg-primary transition-all duration-200"
        />
      </button>
    </div>
    <div class="mt-4">
      <slot />
    </div>
  </div>
</template>
