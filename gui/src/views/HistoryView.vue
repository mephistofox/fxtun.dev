<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useHistoryStore } from '@/stores/history'
import { toast } from '@/composables/useToast'
import {
  Button, Card, Badge, Tooltip,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter
} from '@/components/ui'
import { Trash2, RefreshCw, History, ArrowUpRight, ArrowDownRight, Clock } from 'lucide-vue-next'
import type { TunnelType } from '@/types'

const { t } = useI18n()
const historyStore = useHistoryStore()

const showClearDialog = ref(false)
const filterType = ref<TunnelType | 'all'>('all')

onMounted(() => {
  historyStore.loadHistory()
})

const filteredEntries = computed(() => {
  if (filterType.value === 'all') {
    return historyStore.entries
  }
  return historyStore.entries.filter(e => e.tunnelType === filterType.value)
})

async function clearHistory() {
  await historyStore.clearHistory()
  showClearDialog.value = false
  toast({ title: t('toasts.historyCleared'), variant: 'success' })
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDuration(start: string, end?: string): string {
  const startDate = new Date(start)
  const endDate = end ? new Date(end) : new Date()
  const diff = endDate.getTime() - startDate.getTime()

  const hours = Math.floor(diff / 3600000)
  const minutes = Math.floor((diff % 3600000) / 60000)
  const seconds = Math.floor((diff % 60000) / 1000)

  if (hours > 0) {
    return `${hours}${t('time.hoursShort')} ${minutes}${t('time.minutesShort')}`
  } else if (minutes > 0) {
    return `${minutes}${t('time.minutesShort')} ${seconds}${t('time.secondsShort')}`
  } else {
    return `${seconds}${t('time.secondsShort')}`
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function getTunnelTypeBadge(type: TunnelType): 'http' | 'tcp' | 'udp' {
  return type
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="flex items-center gap-2 text-2xl font-bold">
        <History class="h-6 w-6 text-muted-foreground" />
        {{ t('history.title') }}
      </h1>
      <div class="flex gap-2">
        <!-- Filter -->
        <div class="flex items-center gap-1 rounded-md border bg-muted/30 p-1">
          <Tooltip :content="t('history.filterAll')">
            <Button
              :variant="filterType === 'all' ? 'secondary' : 'ghost'"
              size="xs"
              @click="filterType = 'all'"
            >
              {{ t('common.all') }}
            </Button>
          </Tooltip>
          <Tooltip :content="t('history.filterHttp')">
            <Button
              :variant="filterType === 'http' ? 'secondary' : 'ghost'"
              size="xs"
              @click="filterType = 'http'"
            >
              HTTP
            </Button>
          </Tooltip>
          <Tooltip :content="t('history.filterTcp')">
            <Button
              :variant="filterType === 'tcp' ? 'secondary' : 'ghost'"
              size="xs"
              @click="filterType = 'tcp'"
            >
              TCP
            </Button>
          </Tooltip>
          <Tooltip :content="t('history.filterUdp')">
            <Button
              :variant="filterType === 'udp' ? 'secondary' : 'ghost'"
              size="xs"
              @click="filterType = 'udp'"
            >
              UDP
            </Button>
          </Tooltip>
        </div>

        <Tooltip :content="t('common.refresh')">
          <Button variant="outline" size="sm" @click="historyStore.loadHistory()">
            <RefreshCw class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('common.refresh') }}</span>
          </Button>
        </Tooltip>
        <Button
          variant="destructive"
          size="sm"
          :disabled="historyStore.entries.length === 0"
          @click="showClearDialog = true"
        >
          <Trash2 class="h-4 w-4 sm:mr-2" />
          <span class="hidden sm:inline">{{ t('history.clearHistory') }}</span>
        </Button>
      </div>
    </div>

    <!-- History Table -->
    <Card class="overflow-hidden">
      <div v-if="filteredEntries.length === 0" class="p-8 text-center">
        <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
          <History class="h-6 w-6 text-muted-foreground" />
        </div>
        <p class="font-medium text-muted-foreground">{{ t('history.noHistory') }}</p>
        <p class="mt-1 text-sm text-muted-foreground">
          {{ t('history.noHistoryHint') }}
        </p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b bg-muted/50">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.bundle') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.type') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.localPort') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.remote') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.connected') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.duration') }}
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.traffic') }}
              </th>
            </tr>
          </thead>
          <TransitionGroup name="list" tag="tbody" class="divide-y">
            <tr
              v-for="entry in filteredEntries"
              :key="entry.id"
              class="transition-colors hover:bg-muted/30"
            >
              <td class="px-4 py-3 text-sm font-medium">
                {{ entry.bundleName || '-' }}
              </td>
              <td class="px-4 py-3">
                <Badge :variant="getTunnelTypeBadge(entry.tunnelType)">
                  {{ entry.tunnelType.toUpperCase() }}
                </Badge>
              </td>
              <td class="px-4 py-3 text-sm">
                <code class="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">{{ entry.localPort }}</code>
              </td>
              <td class="px-4 py-3 text-sm">
                <code class="max-w-[200px] truncate rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                  {{ entry.url || entry.remoteAddr || '-' }}
                </code>
              </td>
              <td class="px-4 py-3 text-sm text-muted-foreground">
                <div class="flex items-center gap-1.5">
                  <Clock class="h-3.5 w-3.5" />
                  {{ formatDate(entry.connectedAt) }}
                </div>
              </td>
              <td class="px-4 py-3 text-sm">
                <span v-if="entry.disconnectedAt" class="text-muted-foreground">
                  {{ formatDuration(entry.connectedAt, entry.disconnectedAt) }}
                </span>
                <Badge v-else variant="success" class="animate-pulse">
                  {{ t('history.active') }}
                </Badge>
              </td>
              <td class="px-4 py-3 text-sm">
                <div class="flex items-center gap-2">
                  <span class="flex items-center gap-0.5 text-emerald-600 dark:text-emerald-400">
                    <ArrowUpRight class="h-3 w-3" />
                    {{ formatBytes(entry.bytesSent) }}
                  </span>
                  <span class="text-muted-foreground">/</span>
                  <span class="flex items-center gap-0.5 text-blue-600 dark:text-blue-400">
                    <ArrowDownRight class="h-3 w-3" />
                    {{ formatBytes(entry.bytesReceived) }}
                  </span>
                </div>
              </td>
            </tr>
          </TransitionGroup>
        </table>
      </div>
    </Card>

    <!-- Pagination -->
    <div v-if="historyStore.totalCount > 50" class="flex items-center justify-center gap-4">
      <Button variant="outline" size="sm">
        {{ t('common.previous') }}
      </Button>
      <span class="text-sm text-muted-foreground">
        {{ t('history.showing', { count: historyStore.entries.length, total: historyStore.totalCount }) }}
      </span>
      <Button variant="outline" size="sm">
        {{ t('common.next') }}
      </Button>
    </div>

    <!-- Clear History Dialog -->
    <Dialog v-model:open="showClearDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t('history.clearHistory') }}</DialogTitle>
          <DialogDescription>
            {{ t('history.confirmClear') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showClearDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" @click="clearHistory">
            {{ t('common.clear') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
