<script setup lang="ts">
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'
import { X, CheckCircle, AlertCircle, AlertTriangle, Info } from 'lucide-vue-next'

const toastVariants = cva(
  'group pointer-events-auto relative flex w-full items-center justify-between space-x-2 overflow-hidden rounded-lg border p-4 pr-6 shadow-lg transition-all animate-slide-in-from-right',
  {
    variants: {
      variant: {
        default: 'border bg-background text-foreground',
        success: 'border-success/50 bg-success/10 text-success',
        destructive: 'border-destructive/50 bg-destructive/10 text-destructive',
        warning: 'border-warning/50 bg-warning/10 text-warning',
        info: 'border-info/50 bg-info/10 text-info',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

type ToastVariants = VariantProps<typeof toastVariants>

interface Props {
  id: string
  title?: string
  description?: string
  variant?: ToastVariants['variant']
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
})

const emit = defineEmits<{
  dismiss: [id: string]
}>()

const iconMap = {
  default: null,
  success: CheckCircle,
  destructive: AlertCircle,
  warning: AlertTriangle,
  info: Info,
}
</script>

<template>
  <div :class="cn(toastVariants({ variant }), props.class)">
    <div class="flex items-start gap-3">
      <component
        :is="iconMap[variant || 'default']"
        v-if="iconMap[variant || 'default']"
        class="h-5 w-5 shrink-0"
      />
      <div class="grid gap-1">
        <div v-if="title" class="text-sm font-semibold">{{ title }}</div>
        <div v-if="description" class="text-sm opacity-90">{{ description }}</div>
      </div>
    </div>
    <button
      class="absolute right-1 top-1 rounded-md p-1 text-foreground/50 opacity-0 transition-opacity hover:text-foreground focus:opacity-100 focus:outline-none group-hover:opacity-100"
      @click="emit('dismiss', id)"
    >
      <X class="h-4 w-4" />
    </button>
  </div>
</template>
