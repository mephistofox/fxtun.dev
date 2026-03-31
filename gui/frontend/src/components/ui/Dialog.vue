<script setup lang="ts">
import { DialogRoot } from 'radix-vue'

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
  <DialogRoot :open="open" :modal="modal" @update:open="emit('update:open', $event)">
    <slot />
  </DialogRoot>
</template>
