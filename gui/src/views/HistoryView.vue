<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useHistoryStore } from '@/stores/history'
import { toast } from '@/composables/useToast'
import {
  Button, Badge, Tooltip,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter
} from '@/components/ui'
import { Trash2, RefreshCw, History, ArrowUpRight, ArrowDownRight, Clock, Globe, Server, Radio } from 'lucide-vue-next'
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

function getTunnelIcon(type: TunnelType) {
  switch (type) {
    case 'http': return Globe
    case 'tcp': return Server
    case 'udp': return Radio
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <div class="relative">
          <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-20 blur-lg" />
          <div class="relative flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/20 to-accent/20 border border-primary/30">
            <History class="h-7 w-7 text-primary" />
          </div>
        </div>
        <div>
          <h1 class="font-display text-2xl font-bold tracking-tight">{{ t('history.title') }}</h1>
          <p class="text-sm text-muted-foreground">View your tunnel connection history</p>
        </div>
      </div>
      <div class="flex gap-2">
        <!-- Filter -->
        <div class="flex items-center gap-1 rounded-xl border border-border/50 bg-muted/30 p-1">
          <Tooltip :content="t('history.filterAll')">
            <Button
              :variant="filterType === 'all' ? 'default' : 'ghost'"
              :class="filterType === 'all' ? 'bg-gradient-to-r from-primary to-primary shadow-md shadow-primary/25' : ''"
              size="xs"
              @click="filterType = 'all'"
            >
              {{ t('common.all') }}
            </Button>
          </Tooltip>
          <Tooltip :content="t('history.filterHttp')">
            <Button
              :variant="filterType === 'http' ? 'default' : 'ghost'"
              :class="filterType === 'http' ? 'bg-type-http shadow-md shadow-type-http/25' : ''"
              size="xs"
              @click="filterType = 'http'"
            >
              HTTP
            </Button>
          </Tooltip>
          <Tooltip :content="t('history.filterTcp')">
            <Button
              :variant="filterType === 'tcp' ? 'default' : 'ghost'"
              :class="filterType === 'tcp' ? 'bg-type-tcp shadow-md shadow-type-tcp/25' : ''"
              size="xs"
              @click="filterType = 'tcp'"
            >
              TCP
            </Button>
          </Tooltip>
          <Tooltip :content="t('history.filterUdp')">
            <Button
              :variant="filterType === 'udp' ? 'default' : 'ghost'"
              :class="filterType === 'udp' ? 'bg-type-udp shadow-md shadow-type-udp/25' : ''"
              size="xs"
              @click="filterType = 'udp'"
            >
              UDP
            </Button>
          </Tooltip>
        </div>

        <Tooltip :content="t('common.refresh')">
          <Button variant="outline" size="sm" class="border-border/50 hover:border-primary/50 hover:bg-primary/5" @click="historyStore.loadHistory()">
            <RefreshCw class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('common.refresh') }}</span>
          </Button>
        </Tooltip>
        <Button
          variant="destructive"
          size="sm"
          class="shadow-lg shadow-destructive/25"
          :disabled="historyStore.entries.length === 0"
          @click="showClearDialog = true"
        >
          <Trash2 class="h-4 w-4 sm:mr-2" />
          <span class="hidden sm:inline">{{ t('history.clearHistory') }}</span>
        </Button>
      </div>
    </div>

    <!-- History Table -->
    <div class="cyber-card rounded-2xl overflow-hidden">
      <div v-if="filteredEntries.length === 0" class="p-12 text-center">
        <div class="relative mx-auto mb-6 w-fit">
          <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-10 blur-xl" />
          <div class="relative flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50 border border-border/30">
            <History class="h-8 w-8 text-muted-foreground" />
          </div>
        </div>
        <p class="font-display font-semibold text-lg text-muted-foreground">{{ t('history.noHistory') }}</p>
        <p class="mt-2 text-sm text-muted-foreground max-w-md mx-auto">
          {{ t('history.noHistoryHint') }}
        </p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="border-b border-border/50 bg-muted/30">
            <tr>
              <th class="px-4 py-4 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.bundle') }}
              </th>
              <th class="px-4 py-4 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.type') }}
              </th>
              <th class="px-4 py-4 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.localPort') }}
              </th>
              <th class="px-4 py-4 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.remote') }}
              </th>
              <th class="px-4 py-4 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.connected') }}
              </th>
              <th class="px-4 py-4 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.duration') }}
              </th>
              <th class="px-4 py-4 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.traffic') }}
              </th>
            </tr>
          </thead>
          <TransitionGroup name="list" tag="tbody" class="divide-y divide-border/30">
            <tr
              v-for="entry in filteredEntries"
              :key="entry.id"
              class="transition-all duration-200 hover:bg-primary/5"
            >
              <td class="px-4 py-4">
                <div class="flex items-center gap-3">
                  <div :class="[
                    'flex h-8 w-8 items-center justify-center rounded-lg',
                    entry.tunnelType === 'http' ? 'bg-type-http/20' : entry.tunnelType === 'tcp' ? 'bg-type-tcp/20' : 'bg-type-udp/20'
                  ]">
                    <component
                      :is="getTunnelIcon(entry.tunnelType)"
                      :class="[
                        'h-4 w-4',
                        entry.tunnelType === 'http' ? 'text-type-http' : entry.tunnelType === 'tcp' ? 'text-type-tcp' : 'text-type-udp'
                      ]"
                    />
                  </div>
                  <span class="font-medium">{{ entry.bundleName || '-' }}</span>
                </div>
              </td>
              <td class="px-4 py-4">
                <Badge :variant="getTunnelTypeBadge(entry.tunnelType)">
                  {{ entry.tunnelType.toUpperCase() }}
                </Badge>
              </td>
              <td class="px-4 py-4">
                <code class="rounded-lg bg-muted/50 px-2 py-1 text-xs font-mono border border-border/30">{{ entry.localPort }}</code>
              </td>
              <td class="px-4 py-4">
                <code class="max-w-[200px] truncate rounded-lg bg-muted/50 px-2 py-1 text-xs font-mono border border-border/30">
                  {{ entry.url || entry.remoteAddr || '-' }}
                </code>
              </td>
              <td class="px-4 py-4">
                <div class="flex items-center gap-2 text-sm text-muted-foreground">
                  <Clock class="h-3.5 w-3.5" />
                  {{ formatDate(entry.connectedAt) }}
                </div>
              </td>
              <td class="px-4 py-4">
                <span v-if="entry.disconnectedAt" class="text-sm text-muted-foreground">
                  {{ formatDuration(entry.connectedAt, entry.disconnectedAt) }}
                </span>
                <div v-else class="flex items-center gap-1.5">
                  <span class="relative flex h-2 w-2">
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-2 w-2 bg-success"></span>
                  </span>
                  <span class="text-sm font-medium text-success">{{ t('history.active') }}</span>
                </div>
              </td>
              <td class="px-4 py-4">
                <div class="flex items-center gap-3 text-sm">
                  <span class="flex items-center gap-1 text-type-http">
                    <ArrowUpRight class="h-3.5 w-3.5" />
                    {{ formatBytes(entry.bytesSent) }}
                  </span>
                  <span class="text-muted-foreground/50">/</span>
                  <span class="flex items-center gap-1 text-type-tcp">
                    <ArrowDownRight class="h-3.5 w-3.5" />
                    {{ formatBytes(entry.bytesReceived) }}
                  </span>
                </div>
              </td>
            </tr>
          </TransitionGroup>
        </table>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="historyStore.totalCount > 50" class="flex items-center justify-center gap-4">
      <Button variant="outline" size="sm" class="border-border/50">
        {{ t('common.previous') }}
      </Button>
      <span class="text-sm text-muted-foreground">
        {{ t('history.showing', { count: historyStore.entries.length, total: historyStore.totalCount }) }}
      </span>
      <Button variant="outline" size="sm" class="border-border/50">
        {{ t('common.next') }}
      </Button>
    </div>

    <!-- Clear History Dialog -->
    <Dialog v-model:open="showClearDialog">
      <DialogContent class="border-destructive/30 bg-card/95 backdrop-blur-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-3 font-display">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-destructive/20 border border-destructive/30">
              <Trash2 class="h-5 w-5 text-destructive" />
            </div>
            {{ t('history.clearHistory') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('history.confirmClear') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" class="border-border/50" @click="showClearDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" class="shadow-lg shadow-destructive/25" @click="clearHistory">
            {{ t('common.clear') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
