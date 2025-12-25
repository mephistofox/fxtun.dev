<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLogsStore } from '@/stores/logs'
import { toast } from '@/composables/useToast'
import {
  Button, Card, Select, Badge, Input, Tooltip,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter
} from '@/components/ui'
import { Trash2, Download, ArrowDown, FileText, Search, Copy, Check } from 'lucide-vue-next'

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

  // Demo logs for development
  logsStore.addLog({ timestamp: new Date().toISOString(), level: 'info', message: 'Application started' })
  logsStore.addLog({ timestamp: new Date().toISOString(), level: 'debug', message: 'Checking saved credentials...' })
  logsStore.addLog({ timestamp: new Date().toISOString(), level: 'info', message: 'No saved credentials found' })
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
  <div class="flex h-full flex-col space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="flex items-center gap-2 text-2xl font-bold">
        <FileText class="h-6 w-6 text-muted-foreground" />
        {{ t('logs.title') }}
      </h1>
      <div class="flex gap-2">
        <!-- Search -->
        <div class="relative">
          <Search class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="searchQuery"
            :placeholder="t('logs.search')"
            class="w-48 pl-8"
          />
        </div>

        <Select
          v-model="filter"
          :options="filterOptions"
          class="w-32"
        />
        <Tooltip :content="t('logs.export')">
          <Button variant="outline" size="sm" @click="exportLogs">
            <Download class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('logs.export') }}</span>
          </Button>
        </Tooltip>
        <Button
          variant="destructive"
          size="sm"
          :disabled="logsStore.logs.length === 0"
          @click="showClearDialog = true"
        >
          <Trash2 class="h-4 w-4 sm:mr-2" />
          <span class="hidden sm:inline">{{ t('common.clear') }}</span>
        </Button>
      </div>
    </div>

    <!-- Log Viewer -->
    <Card class="flex-1 overflow-hidden">
      <div
        ref="logContainer"
        class="h-full overflow-auto p-4 font-mono text-sm"
      >
        <div v-if="filteredAndSearchedLogs.length === 0" class="flex h-full flex-col items-center justify-center">
          <div class="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
            <FileText class="h-6 w-6 text-muted-foreground" />
          </div>
          <p class="font-medium text-muted-foreground">{{ t('logs.noLogs') }}</p>
          <p v-if="searchQuery" class="mt-1 text-sm text-muted-foreground">
            {{ t('logs.noSearchResults') }}
          </p>
        </div>

        <TransitionGroup v-else name="list" tag="div">
          <div
            v-for="(log, index) in filteredAndSearchedLogs"
            :key="log.timestamp + index"
            class="group flex items-start gap-2 rounded px-2 py-1.5 transition-colors hover:bg-muted/50"
          >
            <span class="shrink-0 text-muted-foreground tabular-nums">
              {{ formatTimestamp(log.timestamp) }}
            </span>
            <Badge
              :variant="getLevelBadgeVariant(log.level)"
              class="min-w-[60px] shrink-0 justify-center text-xs"
            >
              {{ log.level.toUpperCase() }}
            </Badge>
            <span
              class="flex-1"
              :class="{
                'text-destructive': log.level === 'error',
                'text-amber-600 dark:text-amber-400': log.level === 'warn',
                'text-blue-600 dark:text-blue-400': log.level === 'info',
                'text-muted-foreground': log.level === 'debug',
              }"
            >
              {{ log.message }}
            </span>
            <Tooltip :content="copiedIndex === index ? t('common.copied') : t('logs.copyEntry')">
              <Button
                variant="ghost"
                size="icon"
                class="h-6 w-6 shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
                @click="copyLogEntry(log.message, index)"
              >
                <component
                  :is="copiedIndex === index ? Check : Copy"
                  :class="['h-3 w-3', copiedIndex === index ? 'text-success' : '']"
                />
              </Button>
            </Tooltip>
          </div>
        </TransitionGroup>
      </div>
    </Card>

    <!-- Footer -->
    <div class="flex items-center justify-between text-sm text-muted-foreground">
      <span>
        {{ t('logs.entries', { count: filteredAndSearchedLogs.length }) }}
        <span v-if="searchQuery" class="text-xs">
          ({{ t('logs.filtered') }})
        </span>
      </span>
      <div class="flex items-center gap-3">
        <label class="flex cursor-pointer items-center gap-2">
          <input
            v-model="autoScroll"
            type="checkbox"
            class="h-4 w-4 rounded border-input accent-primary"
          />
          {{ t('logs.autoScroll') }}
        </label>
        <Tooltip :content="t('logs.scrollToBottom')">
          <Button variant="ghost" size="sm" @click="scrollToBottom">
            <ArrowDown class="h-4 w-4" />
          </Button>
        </Tooltip>
      </div>
    </div>

    <!-- Clear Logs Dialog -->
    <Dialog v-model:open="showClearDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t('logs.clearLogs') }}</DialogTitle>
          <DialogDescription>
            {{ t('logs.confirmClear') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showClearDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" @click="clearLogs">
            {{ t('common.clear') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
