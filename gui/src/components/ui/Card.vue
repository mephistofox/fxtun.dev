<script setup lang="ts">
import { type ClassValue } from 'clsx'
import { cn } from '@/lib/utils'

interface Props {
  class?: ClassValue
}

const props = defineProps<Props>()
</script>

<template>
  <div :class="cn('rounded-lg border bg-card text-card-foreground shadow-sm', props.class)">
    <slot />
  </div>
</template>
