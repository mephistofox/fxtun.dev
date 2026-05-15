<script setup lang="ts">
import { useToast } from '@/composables/useToast'
import { CheckCircle, XCircle, Info, X } from 'lucide-vue-next'

const { toasts } = useToast()

function removeToast(id: number) {
  toasts.value = toasts.value.filter(t => t.id !== id)
}

const iconMap = {
  success: CheckCircle,
  error: XCircle,
  info: Info,
}

const borderColorMap = {
  success: 'border-l-type-http',
  error: 'border-l-destructive',
  info: 'border-l-type-tcp',
}

const iconColorMap = {
  success: 'text-type-http',
  error: 'text-destructive',
  info: 'text-type-tcp',
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 max-w-sm">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="flex items-start gap-3 rounded-lg border border-border bg-card/95 backdrop-blur-xl px-4 py-3 shadow-xl border-l-4"
          :class="borderColorMap[toast.type]"
        >
          <component
            :is="iconMap[toast.type]"
            class="h-5 w-5 flex-shrink-0 mt-0.5"
            :class="iconColorMap[toast.type]"
          />
          <p class="text-sm text-foreground flex-1">{{ toast.message }}</p>
          <button
            type="button"
            class="flex-shrink-0 text-muted-foreground hover:text-foreground transition-colors"
            @click="removeToast(toast.id)"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-enter-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-leave-active {
  transition: all 0.2s ease-in;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.toast-move {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
</style>
