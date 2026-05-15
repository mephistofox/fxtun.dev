<script setup lang="ts">
import { TooltipRoot, TooltipTrigger, TooltipPortal, TooltipContent, TooltipArrow } from 'radix-vue'
import { cn } from '@/lib/utils'

interface Props {
  content?: string
  side?: 'top' | 'right' | 'bottom' | 'left'
  sideOffset?: number
  delayDuration?: number
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  side: 'top',
  sideOffset: 4,
  delayDuration: 200,
})
</script>

<template>
  <TooltipRoot :delay-duration="delayDuration">
    <TooltipTrigger as-child>
      <slot />
    </TooltipTrigger>
    <TooltipPortal>
      <TooltipContent
        :side="side"
        :side-offset="sideOffset"
        :class="cn(
          'z-50 overflow-hidden rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground animate-fade-in',
          props.class
        )"
      >
        <slot name="content">
          {{ content }}
        </slot>
        <TooltipArrow class="fill-primary" />
      </TooltipContent>
    </TooltipPortal>
  </TooltipRoot>
</template>
