import { ref, computed } from 'vue'

export interface Toast {
  id: string
  title?: string
  description?: string
  variant?: 'default' | 'success' | 'destructive' | 'warning' | 'info'
  duration?: number
}

const toasts = ref<Toast[]>([])
const TOAST_LIMIT = 5
const TOAST_REMOVE_DELAY = 5000

let toastCount = 0

function generateId(): string {
  return `toast-${++toastCount}-${Date.now()}`
}

export function useToast() {
  function toast(options: Omit<Toast, 'id'>) {
    const id = generateId()
    const duration = options.duration ?? TOAST_REMOVE_DELAY

    const newToast: Toast = {
      id,
      ...options,
    }

    toasts.value = [newToast, ...toasts.value].slice(0, TOAST_LIMIT)

    if (duration > 0) {
      setTimeout(() => {
        dismiss(id)
      }, duration)
    }

    return id
  }

  function dismiss(id: string) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  function dismissAll() {
    toasts.value = []
  }

  return {
    toasts: computed(() => toasts.value),
    toast,
    dismiss,
    dismissAll,
  }
}

// Singleton for global access
const globalToast = useToast()

export function toast(options: Omit<Toast, 'id'>) {
  return globalToast.toast(options)
}

export function dismissToast(id: string) {
  return globalToast.dismiss(id)
}

export function dismissAllToasts() {
  return globalToast.dismissAll()
}

export { toasts }
