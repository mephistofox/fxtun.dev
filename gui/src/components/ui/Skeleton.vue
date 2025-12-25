<script setup lang="ts">
import { cn } from '@/lib/utils'

interface Props {
  class?: string
}

const props = defineProps<Props>()
</script>

<template>
  <div
    :class="cn(
      'animate-pulse rounded-md bg-primary/10',
      props.class
    )"
  />
</template>
