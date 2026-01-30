<template>
  <div>
    <div v-if="exchanges.length === 0" class="px-4 py-8 text-center text-gray-500 text-sm">
      No requests captured yet
    </div>
    <div
      v-for="ex in exchanges"
      :key="ex.id"
      @click="$emit('select', ex.id)"
      :class="[
        'flex items-center gap-3 px-4 py-2 cursor-pointer border-b border-gray-800/50 text-sm hover:bg-gray-900/50 transition',
        selectedId === ex.id ? 'bg-gray-800/70' : ''
      ]"
    >
      <span :class="methodClass(ex.method)" class="font-mono font-semibold w-16 text-xs">
        {{ ex.method }}
      </span>
      <span class="flex-1 truncate font-mono text-gray-300">{{ ex.path }}</span>
      <span :class="statusClass(ex.status_code)" class="font-mono text-xs px-1.5 py-0.5 rounded">
        {{ ex.status_code }}
      </span>
      <span class="text-gray-500 text-xs w-16 text-right">{{ formatDuration(ex.duration_ns) }}</span>
      <span class="text-gray-600 text-xs w-16 text-right">{{ formatTime(ex.timestamp) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ExchangeSummary } from '../../api/client'

defineProps<{
  exchanges: ExchangeSummary[]
  selectedId: string | null
}>()

defineEmits<{
  select: [id: string]
}>()

function methodClass(method: string): string {
  const colors: Record<string, string> = {
    GET: 'text-emerald-400',
    POST: 'text-blue-400',
    PUT: 'text-amber-400',
    PATCH: 'text-orange-400',
    DELETE: 'text-red-400',
  }
  return colors[method] || 'text-gray-400'
}

function statusClass(status: number): string {
  if (status >= 500) return 'bg-red-900/50 text-red-300'
  if (status >= 400) return 'bg-amber-900/50 text-amber-300'
  if (status >= 300) return 'bg-blue-900/50 text-blue-300'
  if (status >= 200) return 'bg-emerald-900/50 text-emerald-300'
  return 'bg-gray-800 text-gray-400'
}

function formatDuration(ns: number): string {
  const ms = ns / 1_000_000
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatTime(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
</script>
