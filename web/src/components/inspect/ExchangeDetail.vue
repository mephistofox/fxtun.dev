<template>
  <div class="p-4">
    <div class="flex items-center gap-3 mb-4">
      <span :class="methodClass(exchange.method)" class="font-mono font-bold text-lg">{{ exchange.method }}</span>
      <span class="font-mono text-gray-300">{{ exchange.path }}</span>
      <span :class="statusBadge(exchange.status_code)" class="font-mono text-sm px-2 py-0.5 rounded">
        {{ exchange.status_code }}
      </span>
      <span class="text-gray-500 text-sm">{{ formatDuration(exchange.duration_ns) }}</span>
      <span v-if="exchange.replay_ref" class="text-xs bg-purple-900/50 text-purple-300 px-2 py-0.5 rounded">
        Replayed from {{ exchange.replay_ref }}
      </span>
      <div class="ml-auto flex gap-2">
        <button
          @click="$emit('replay', exchange.id)"
          :disabled="replaying"
          class="px-3 py-1 text-sm bg-blue-600 hover:bg-blue-500 disabled:opacity-50 rounded transition text-white"
        >
          {{ replaying ? 'Replaying...' : 'Replay' }}
        </button>
        <button
          @click="showEditor = true"
          class="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded transition text-gray-200"
        >
          Edit & Replay
        </button>
      </div>
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

    <!-- Replay Result -->
    <div v-if="replayResult" class="mt-6 border-t border-gray-800 pt-4">
      <h3 class="text-sm font-semibold text-gray-400 mb-3">Replay Result</h3>
      <div class="mb-2">
        <span :class="statusBadge(replayResult.status_code)" class="font-mono text-sm px-2 py-0.5 rounded">
          {{ replayResult.status_code }}
        </span>
      </div>
      <h4 class="text-xs font-semibold text-gray-500 mb-1">Response Headers</h4>
      <HeadersTable :headers="replayResult.response_headers" />
      <h4 class="text-xs font-semibold text-gray-500 mt-3 mb-1">Response Body</h4>
      <BodyViewer
        :body="replayResult.response_body"
        :content-type="getReplayContentType(replayResult.response_headers)"
        :body-size="replayResult.response_body ? replayResult.response_body.length : 0"
      />
    </div>

    <!-- Replay Editor Modal -->
    <ReplayEditor
      v-if="showEditor"
      :exchange="exchange"
      @send="onEditorSend"
      @close="showEditor = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CapturedExchange, ReplayRequest, ReplayResponse } from '../../api/client'
import BodyViewer from './BodyViewer.vue'
import HeadersTable from './HeadersTable.vue'
import ReplayEditor from './ReplayEditor.vue'

defineProps<{
  exchange: CapturedExchange
  replaying?: boolean
  replayResult?: ReplayResponse | null
}>()

const emit = defineEmits<{
  replay: [id: string]
  editReplay: [mods: ReplayRequest]
}>()

const activeTab = ref('Request')
const showEditor = ref(false)

function onEditorSend(mods: ReplayRequest) {
  showEditor.value = false
  emit('editReplay', mods)
}

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

function getReplayContentType(headers: Record<string, string[]> | null): string {
  if (!headers) return ''
  const ct = headers['Content-Type'] || headers['content-type']
  return ct?.[0] || ''
}
</script>
