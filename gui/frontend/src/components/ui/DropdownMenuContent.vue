<script setup lang="ts">
import { DropdownMenuPortal, DropdownMenuContent } from 'radix-vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
  sideOffset?: number
  align?: 'start' | 'center' | 'end'
  side?: 'top' | 'right' | 'bottom' | 'left'
}

const props = withDefaults(defineProps<Props>(), {
  sideOffset: 4,
  align: 'center',
  side: 'bottom',
})
</script>

<template>
  <DropdownMenuPortal>
    <DropdownMenuContent
      :side-offset="sideOffset"
      :align="align"
      :side="side"
      :class="cn(
        'z-50 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md',
        'data-[state=open]:animate-scale-in data-[state=closed]:animate-scale-out',
        props.class
      )"
    >
      <slot />
    </DropdownMenuContent>
  </DropdownMenuPortal>
</template>
