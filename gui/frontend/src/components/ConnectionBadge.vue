<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import StatusIndicator from './StatusIndicator.vue'
import { cn } from '@/lib/utils'

type Status = 'connected' | 'connecting' | 'disconnected' | 'error'

interface Props {
  status: Status
  server?: string
  class?: string
}

const props = defineProps<Props>()
const { t } = useI18n()

const statusText = computed(() => {
  switch (props.status) {
    case 'connected':
      return t('status.connected')
    case 'connecting':
      return t('status.connecting')
    case 'disconnected':
      return t('status.disconnected')
    case 'error':
      return t('errors.general')
    default:
      return ''
  }
})

const badgeClasses = computed(() => {
  switch (props.status) {
    case 'connected':
      return 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20'
    case 'connecting':
      return 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20'
    case 'disconnected':
      return 'bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20'
    case 'error':
      return 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20'
    default:
      return ''
  }
})
</script>

<template>
  <div
    :class="cn(
      'inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium transition-colors',
      badgeClasses,
      props.class
    )"
  >
    <StatusIndicator :status="status" size="sm" />
    <span>{{ statusText }}</span>
    <span v-if="server && status === 'connected'" class="text-muted-foreground">
      &bull; {{ server }}
    </span>
  </div>
</template>
