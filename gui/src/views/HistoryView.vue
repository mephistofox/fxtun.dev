<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useHistoryStore } from '@/stores/history'
import { toast } from '@/composables/useToast'
import {
  Button, Badge,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter
} from '@/components/ui'
import { Trash2, RefreshCw, History, ArrowUpRight, ArrowDownRight, Clock, Globe, Server, Radio } from 'lucide-vue-next'
import { formatBytes } from '@/utils/format'
import type { TunnelType } from '@/types'

const { t } = useI18n()
const historyStore = useHistoryStore()

const showClearDialog = ref(false)
const filterType = ref<TunnelType | 'all'>('all')

onMounted(async () => {
  await historyStore.loadHistory()
  historyStore.verifyActiveEntries()
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
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div class="flex items-center gap-3">
        <div class="h-10 w-10 rounded-xl bg-gradient-to-br from-primary/20 to-accent/20 border border-primary/30 flex items-center justify-center">
          <History class="h-5 w-5 text-primary" />
        </div>
        <div>
          <h1 class="font-display text-xl font-bold">{{ t('history.title') }}</h1>
          <p class="text-xs text-muted-foreground">{{ t('history.subtitle') }}</p>
        </div>
      </div>

      <div class="flex items-center gap-2 flex-wrap">
        <!-- Filter buttons -->
        <div class="flex items-center gap-0.5 rounded-lg border border-border/50 bg-muted/30 p-0.5">
          <Button
            :variant="filterType === 'all' ? 'default' : 'ghost'"
            size="xs"
            class="h-7 px-2 text-xs"
            @click="filterType = 'all'"
          >
            {{ t('common.all') }}
          </Button>
          <Button
            :variant="filterType === 'http' ? 'default' : 'ghost'"
            :class="filterType === 'http' ? 'bg-type-http hover:bg-type-http/90' : ''"
            size="xs"
            class="h-7 px-2 text-xs"
            @click="filterType = 'http'"
          >
            HTTP
          </Button>
          <Button
            :variant="filterType === 'tcp' ? 'default' : 'ghost'"
            :class="filterType === 'tcp' ? 'bg-type-tcp hover:bg-type-tcp/90' : ''"
            size="xs"
            class="h-7 px-2 text-xs"
            @click="filterType = 'tcp'"
          >
            TCP
          </Button>
          <Button
            :variant="filterType === 'udp' ? 'default' : 'ghost'"
            :class="filterType === 'udp' ? 'bg-type-udp hover:bg-type-udp/90' : ''"
            size="xs"
            class="h-7 px-2 text-xs"
            @click="filterType = 'udp'"
          >
            UDP
          </Button>
        </div>

        <Button variant="outline" size="sm" @click="historyStore.loadHistory()">
          <RefreshCw class="h-4 w-4" />
        </Button>
        <Button
          variant="destructive"
          size="sm"
          :disabled="historyStore.entries.length === 0"
          @click="showClearDialog = true"
        >
          <Trash2 class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <!-- History Table -->
    <div class="rounded-xl border border-border/50 bg-card/80 overflow-hidden">
      <div v-if="filteredEntries.length === 0" class="p-10 text-center">
        <div class="mx-auto mb-4 h-14 w-14 rounded-xl bg-muted/50 flex items-center justify-center">
          <History class="h-7 w-7 text-muted-foreground" />
        </div>
        <p class="font-semibold">{{ t('history.noHistory') }}</p>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('history.noHistoryHint') }}</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="border-b border-border/50 bg-muted/30">
            <tr>
              <th class="px-3 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.bundle') }}
              </th>
              <th class="px-3 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.type') }}
              </th>
              <th class="px-3 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.localPort') }}
              </th>
              <th class="px-3 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.remote') }}
              </th>
              <th class="px-3 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.connected') }}
              </th>
              <th class="px-3 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.duration') }}
              </th>
              <th class="px-3 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ t('history.traffic') }}
              </th>
            </tr>
          </thead>
          <TransitionGroup name="list" tag="tbody" class="divide-y divide-border/30">
            <tr
              v-for="entry in filteredEntries"
              :key="entry.id"
              class="transition-colors hover:bg-muted/30"
            >
              <td class="px-3 py-2.5">
                <div class="flex items-center gap-2">
                  <div :class="[
                    'flex h-6 w-6 items-center justify-center rounded',
                    entry.tunnelType === 'http' ? 'bg-type-http/20' : entry.tunnelType === 'tcp' ? 'bg-type-tcp/20' : 'bg-type-udp/20'
                  ]">
                    <component
                      :is="getTunnelIcon(entry.tunnelType)"
                      :class="[
                        'h-3 w-3',
                        entry.tunnelType === 'http' ? 'text-type-http' : entry.tunnelType === 'tcp' ? 'text-type-tcp' : 'text-type-udp'
                      ]"
                    />
                  </div>
                  <span class="font-medium text-xs truncate max-w-[100px]">{{ entry.bundleName || '-' }}</span>
                </div>
              </td>
              <td class="px-3 py-2.5">
                <Badge :variant="getTunnelTypeBadge(entry.tunnelType)" class="text-[10px]">
                  {{ entry.tunnelType.toUpperCase() }}
                </Badge>
              </td>
              <td class="px-3 py-2.5">
                <code class="text-xs font-mono">:{{ entry.localPort }}</code>
              </td>
              <td class="px-3 py-2.5">
                <code class="text-xs font-mono truncate max-w-[150px] block">
                  {{ entry.url || entry.remoteAddr || '-' }}
                </code>
              </td>
              <td class="px-3 py-2.5">
                <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Clock class="h-3 w-3" />
                  {{ formatDate(entry.connectedAt) }}
                </div>
              </td>
              <td class="px-3 py-2.5">
                <span v-if="entry.disconnectedAt" class="text-xs text-muted-foreground">
                  {{ formatDuration(entry.connectedAt, entry.disconnectedAt) }}
                </span>
                <div v-else class="flex items-center gap-1">
                  <span class="relative flex h-1.5 w-1.5">
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-success"></span>
                  </span>
                  <span class="text-xs font-medium text-success">{{ t('history.active') }}</span>
                </div>
              </td>
              <td class="px-3 py-2.5">
                <div class="flex items-center gap-2 text-xs">
                  <span class="flex items-center gap-0.5 text-type-http">
                    <ArrowUpRight class="h-3 w-3" />
                    {{ formatBytes(historyStore.getLiveTraffic(entry).bytesSent) }}
                  </span>
                  <span class="text-muted-foreground/50">/</span>
                  <span class="flex items-center gap-0.5 text-type-tcp">
                    <ArrowDownRight class="h-3 w-3" />
                    {{ formatBytes(historyStore.getLiveTraffic(entry).bytesReceived) }}
                  </span>
                </div>
              </td>
            </tr>
          </TransitionGroup>
        </table>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="historyStore.totalCount > 50" class="flex items-center justify-center gap-3 text-sm">
      <Button variant="outline" size="sm">{{ t('common.previous') }}</Button>
      <span class="text-muted-foreground">
        {{ t('history.showing', { count: historyStore.entries.length, total: historyStore.totalCount }) }}
      </span>
      <Button variant="outline" size="sm">{{ t('common.next') }}</Button>
    </div>

    <!-- Clear History Dialog -->
    <Dialog v-model:open="showClearDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-destructive">
            <Trash2 class="h-5 w-5" />
            {{ t('history.clearHistory') }}
          </DialogTitle>
          <DialogDescription>{{ t('history.confirmClear') }}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showClearDialog = false">{{ t('common.cancel') }}</Button>
          <Button variant="destructive" @click="clearHistory">{{ t('common.clear') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
