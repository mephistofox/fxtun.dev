<script setup lang="ts">
import { DropdownMenuSeparator } from 'radix-vue'
import { cn } from '@/lib/utils'

interface Props {
  class?: string
}

const props = defineProps<Props>()
</script>

<template>
  <DropdownMenuSeparator :class="cn('-mx-1 my-1 h-px bg-muted', props.class)" />
</template>
