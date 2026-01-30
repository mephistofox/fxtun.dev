<template>
  <div class="p-4">
    <div class="flex items-center gap-3 mb-4">
      <span :class="methodClass(exchange.method)" class="font-mono font-bold text-lg">{{ exchange.method }}</span>
      <span class="font-mono text-gray-300">{{ exchange.path }}</span>
      <span :class="statusBadge(exchange.status_code)" class="font-mono text-sm px-2 py-0.5 rounded">
        {{ exchange.status_code }}
      </span>
      <span class="text-gray-500 text-sm">{{ formatDuration(exchange.duration_ns) }}</span>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-gray-800 mb-4">
      <button
        v-for="tab in ['Request', 'Response']"
        :key="tab"
        @click="activeTab = tab"
        :class="[
          'px-4 py-2 text-sm font-medium transition border-b-2 -mb-px',
          activeTab === tab
            ? 'border-blue-500 text-blue-400'
            : 'border-transparent text-gray-500 hover:text-gray-300'
        ]"
      >
        {{ tab }}
      </button>
    </div>

    <!-- Request Tab -->
    <div v-if="activeTab === 'Request'">
      <h3 class="text-sm font-semibold text-gray-400 mb-2">Headers</h3>
      <HeadersTable :headers="exchange.request_headers" />

      <h3 class="text-sm font-semibold text-gray-400 mt-4 mb-2">
        Body
        <span class="text-gray-600 font-normal">
          ({{ formatSize(exchange.request_body_size) }})
        </span>
      </h3>
      <BodyViewer
        :body="exchange.request_body"
        :content-type="getContentType(exchange.request_headers)"
        :body-size="exchange.request_body_size"
      />
    </div>

    <!-- Response Tab -->
    <div v-if="activeTab === 'Response'">
      <h3 class="text-sm font-semibold text-gray-400 mb-2">Headers</h3>
      <HeadersTable :headers="exchange.response_headers" />

      <h3 class="text-sm font-semibold text-gray-400 mt-4 mb-2">
        Body
        <span class="text-gray-600 font-normal">
          ({{ formatSize(exchange.response_body_size) }})
        </span>
      </h3>
      <BodyViewer
        :body="exchange.response_body"
        :content-type="getContentType(exchange.response_headers)"
        :body-size="exchange.response_body_size"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CapturedExchange } from '../../api/client'
import BodyViewer from './BodyViewer.vue'
import HeadersTable from './HeadersTable.vue'

defineProps<{
  exchange: CapturedExchange
}>()

const activeTab = ref('Request')

function methodClass(method: string): string {
  const colors: Record<string, string> = {
    GET: 'text-emerald-400', POST: 'text-blue-400', PUT: 'text-amber-400',
    PATCH: 'text-orange-400', DELETE: 'text-red-400',
  }
  return colors[method] || 'text-gray-400'
}

function statusBadge(status: number): string {
  if (status >= 500) return 'bg-red-900/50 text-red-300'
  if (status >= 400) return 'bg-amber-900/50 text-amber-300'
  if (status >= 200) return 'bg-emerald-900/50 text-emerald-300'
  return 'bg-gray-800 text-gray-400'
}

function formatDuration(ns: number): string {
  const ms = ns / 1_000_000
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatSize(bytes: number): string {
  if (bytes === 0) return 'empty'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function getContentType(headers: Record<string, string[]> | null): string {
  if (!headers) return ''
  const ct = headers['Content-Type'] || headers['content-type']
  return ct?.[0] || ''
}
</script>
