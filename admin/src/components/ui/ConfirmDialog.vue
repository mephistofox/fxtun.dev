<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next'
import Modal from './Modal.vue'

interface Props {
  show: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'default' | 'destructive'
}

const props = withDefaults(defineProps<Props>(), {
  confirmText: 'Подтвердить',
  cancelText: 'Отмена',
  variant: 'default',
})

const emit = defineEmits<{
  'update:show': [value: boolean]
  confirm: []
  cancel: []
}>()

function close() {
  emit('update:show', false)
  emit('cancel')
}

function confirm() {
  emit('update:show', false)
  emit('confirm')
}
</script>

<template>
  <Modal
    :show="show"
    :title="title"
    width="max-w-md"
    @update:show="(val) => emit('update:show', val)"
  >
    <div class="flex gap-4">
      <div
        v-if="variant === 'destructive'"
        class="flex-shrink-0 flex items-center justify-center w-10 h-10 rounded-full bg-destructive/10"
      >
        <AlertTriangle class="h-5 w-5 text-destructive" />
      </div>
      <p class="text-sm text-muted-foreground leading-relaxed">{{ message }}</p>
    </div>

    <template #footer>
      <button
        type="button"
        class="inline-flex items-center justify-center rounded-lg px-4 py-2 text-sm font-medium transition-colors hover:bg-surface-elevated text-muted-foreground"
        @click="close"
      >
        {{ cancelText }}
      </button>
      <button
        type="button"
        class="inline-flex items-center justify-center rounded-lg px-4 py-2 text-sm font-medium transition-all duration-200 active:scale-[0.98]"
        :class="[
          variant === 'destructive'
            ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90 hover:shadow-lg hover:shadow-destructive/25'
            : 'bg-primary text-primary-foreground hover:bg-primary/90 hover:shadow-lg hover:shadow-primary/25',
        ]"
        @click="confirm"
      >
        {{ confirmText }}
      </button>
    </template>
  </Modal>
</template>
