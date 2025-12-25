<script setup lang="ts">
import { DropdownMenuRoot } from 'radix-vue'

interface Props {
  open?: boolean
  modal?: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()
</script>

<template>
  <DropdownMenuRoot :open="open" :modal="modal" @update:open="emit('update:open', $event)">
    <slot />
  </DropdownMenuRoot>
</template>
