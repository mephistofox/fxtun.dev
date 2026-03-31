<script setup lang="ts">
import { Separator as RadixSeparator } from 'radix-vue'
import { cn } from '@/lib/utils'

interface Props {
  orientation?: 'horizontal' | 'vertical'
  decorative?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  orientation: 'horizontal',
  decorative: true,
})
</script>

<template>
  <RadixSeparator
    :orientation="orientation"
    :decorative="decorative"
    :class="cn(
      'shrink-0 bg-border',
      orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
      props.class
    )"
  />
</template>
