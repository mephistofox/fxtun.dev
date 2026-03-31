<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLogsStore } from '@/stores/logs'
import { toast } from '@/composables/useToast'
import {
  Button, Select, Badge, Input, Tooltip,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter
} from '@/components/ui'
import { Trash2, Download, ArrowDown, Search, Copy, Check, Terminal } from 'lucide-vue-next'

const { t } = useI18n()
const logsStore = useLogsStore()
const logContainer = ref<HTMLElement | null>(null)
const autoScroll = ref(true)
const searchQuery = ref('')
const showClearDialog = ref(false)
const copiedIndex = ref<number | null>(null)

const filterOptions = computed(() => [
  { value: 'all', label: t('logs.allLevels') },
  { value: 'debug', label: t('logs.debug') },
  { value: 'info', label: t('logs.info') },
  { value: 'warn', label: t('logs.warning') },
  { value: 'error', label: t('logs.error') },
])

const filter = computed({
  get: () => logsStore.filter,
  set: (value) => logsStore.setFilter(value as typeof logsStore.filter),
})

const filteredAndSearchedLogs = computed(() => {
  if (!searchQuery.value) {
    return logsStore.filteredLogs
  }
  const query = searchQuery.value.toLowerCase()
  return logsStore.filteredLogs.filter(log =>
    log.message.toLowerCase().includes(query)
  )
})

onMounted(() => {
  logsStore.init()
})

// Auto-scroll when new logs arrive
watch(() => logsStore.filteredLogs.length, () => {
  if (autoScroll.value) {
    scrollToBottom()
  }
})

function getLevelBadgeVariant(level: string): 'default' | 'secondary' | 'destructive' | 'outline' | 'warning' | 'info' {
  switch (level) {
    case 'error': return 'destructive'
    case 'warn': return 'warning'
    case 'info': return 'info'
    default: return 'outline'
  }
}

function getLevelColor(level: string): string {
  switch (level) {
    case 'error': return 'text-destructive'
    case 'warn': return 'text-warning'
    case 'info': return 'text-info'
    default: return 'text-muted-foreground'
  }
}

function getLevelBgColor(level: string): string {
  switch (level) {
    case 'error': return 'bg-destructive/10 border-destructive/30'
    case 'warn': return 'bg-warning/10 border-warning/30'
    case 'info': return 'bg-info/10 border-info/30'
    default: return 'bg-muted/30 border-border/30'
  }
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function clearLogs() {
  logsStore.clearLogs()
  showClearDialog.value = false
  toast({ title: t('toasts.logsCleared'), variant: 'success' })
}

function exportLogs() {
  const data = logsStore.logs.map(log =>
    `[${log.timestamp}] [${log.level.toUpperCase()}] ${log.message}`
  ).join('\n')

  const blob = new Blob([data], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `fxtunnel-logs-${new Date().toISOString().split('T')[0]}.txt`
  a.click()
  URL.revokeObjectURL(url)
  toast({ title: t('toasts.logsExported'), variant: 'success' })
}

function scrollToBottom() {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

function copyLogEntry(message: string, index: number) {
  navigator.clipboard.writeText(message)
  copiedIndex.value = index
  toast({ title: t('toasts.logCopied'), variant: 'success' })
  setTimeout(() => {
    copiedIndex.value = null
  }, 2000)
}
</script>

<template>
  <div class="flex h-full flex-col space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <div class="relative">
          <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-20 blur-lg" />
          <div class="relative flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/20 to-accent/20 border border-primary/30">
            <Terminal class="h-7 w-7 text-primary" />
          </div>
        </div>
        <div>
          <h1 class="font-display text-2xl font-bold tracking-tight">{{ t('logs.title') }}</h1>
          <p class="text-sm text-muted-foreground">Application logs and events</p>
        </div>
      </div>
      <div class="flex gap-2">
        <!-- Search -->
        <div class="relative">
          <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="searchQuery"
            :placeholder="t('logs.search')"
            class="w-48 pl-9 bg-muted/30 border-border/50"
          />
        </div>

        <Select
          v-model="filter"
          :options="filterOptions"
          class="w-32 bg-muted/30"
        />
        <Tooltip :content="t('logs.export')">
          <Button variant="outline" size="sm" class="border-border/50 hover:border-primary/50 hover:bg-primary/5" @click="exportLogs">
            <Download class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('logs.export') }}</span>
          </Button>
        </Tooltip>
        <Button
          variant="destructive"
          size="sm"
          class="shadow-lg shadow-destructive/25"
          :disabled="logsStore.logs.length === 0"
          @click="showClearDialog = true"
        >
          <Trash2 class="h-4 w-4 sm:mr-2" />
          <span class="hidden sm:inline">{{ t('common.clear') }}</span>
        </Button>
      </div>
    </div>

    <!-- Log Viewer -->
    <div class="cyber-card flex-1 rounded-2xl overflow-hidden">
      <div
        ref="logContainer"
        class="h-full overflow-auto p-4 font-mono text-sm"
      >
        <div v-if="filteredAndSearchedLogs.length === 0" class="flex h-full flex-col items-center justify-center">
          <div class="relative mx-auto mb-6 w-fit">
            <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-10 blur-xl" />
            <div class="relative flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50 border border-border/30">
              <Terminal class="h-8 w-8 text-muted-foreground" />
            </div>
          </div>
          <p class="font-display font-semibold text-lg text-muted-foreground">{{ t('logs.noLogs') }}</p>
          <p v-if="searchQuery" class="mt-2 text-sm text-muted-foreground">
            {{ t('logs.noSearchResults') }}
          </p>
        </div>

        <TransitionGroup v-else name="list" tag="div" class="space-y-1">
          <div
            v-for="(log, index) in filteredAndSearchedLogs"
            :key="log.timestamp + index"
            :class="[
              'group flex items-start gap-3 rounded-xl px-3 py-2 transition-all duration-200 border',
              getLevelBgColor(log.level),
              'hover:scale-[1.01]'
            ]"
          >
            <span class="shrink-0 text-muted-foreground tabular-nums text-xs pt-0.5">
              {{ formatTimestamp(log.timestamp) }}
            </span>
            <Badge
              :variant="getLevelBadgeVariant(log.level)"
              class="min-w-[60px] shrink-0 justify-center text-[10px] uppercase tracking-wider"
            >
              {{ log.level }}
            </Badge>
            <span
              :class="['flex-1 break-all', getLevelColor(log.level)]"
            >
              {{ log.message }}
            </span>
            <Tooltip :content="copiedIndex === index ? t('common.copied') : t('logs.copyEntry')">
              <Button
                variant="ghost"
                size="icon"
                class="h-6 w-6 shrink-0 opacity-0 transition-all group-hover:opacity-100"
                @click="copyLogEntry(log.message, index)"
              >
                <component
                  :is="copiedIndex === index ? Check : Copy"
                  :class="['h-3 w-3', copiedIndex === index ? 'text-success' : 'text-muted-foreground']"
                />
              </Button>
            </Tooltip>
          </div>
        </TransitionGroup>
      </div>
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between rounded-xl bg-muted/30 border border-border/30 px-4 py-3">
      <span class="text-sm text-muted-foreground">
        {{ t('logs.entries', { count: filteredAndSearchedLogs.length }) }}
        <span v-if="searchQuery" class="text-xs ml-1 px-2 py-0.5 rounded-full bg-primary/10 text-primary">
          {{ t('logs.filtered') }}
        </span>
      </span>
      <div class="flex items-center gap-4">
        <label class="flex cursor-pointer items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors">
          <div class="relative">
            <input
              v-model="autoScroll"
              type="checkbox"
              class="peer sr-only"
            />
            <div class="h-5 w-5 rounded-md border-2 border-border/50 bg-muted/30 transition-all peer-checked:border-primary peer-checked:bg-primary/20" />
            <svg
              class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 h-3 w-3 text-primary opacity-0 transition-opacity peer-checked:opacity-100"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="3"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          {{ t('logs.autoScroll') }}
        </label>
        <Tooltip :content="t('logs.scrollToBottom')">
          <Button variant="outline" size="sm" class="border-border/50 hover:border-primary/50" @click="scrollToBottom">
            <ArrowDown class="h-4 w-4" />
          </Button>
        </Tooltip>
      </div>
    </div>

    <!-- Clear Logs Dialog -->
    <Dialog v-model:open="showClearDialog">
      <DialogContent class="border-destructive/30 bg-card/95 backdrop-blur-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-3 font-display">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-destructive/20 border border-destructive/30">
              <Trash2 class="h-5 w-5 text-destructive" />
            </div>
            {{ t('logs.clearLogs') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('logs.confirmClear') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" class="border-border/50" @click="showClearDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" class="shadow-lg shadow-destructive/25" @click="clearLogs">
            {{ t('common.clear') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
