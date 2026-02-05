<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @mousedown.self="$emit('close')">
    <div class="bg-gray-950 border border-gray-800 rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-800">
        <h2 class="text-lg font-semibold text-gray-100">Edit & Replay</h2>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-300 transition">&times;</button>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto px-5 py-4 space-y-4">
        <!-- Method & Path -->
        <div class="flex gap-3">
          <select
            v-model="method"
            class="bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm font-mono text-gray-200 focus:outline-none focus:border-blue-500"
          >
            <option v-for="m in methods" :key="m" :value="m">{{ m }}</option>
          </select>
          <input
            v-model="path"
            type="text"
            class="flex-1 bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm font-mono text-gray-200 focus:outline-none focus:border-blue-500"
            placeholder="/path"
          />
        </div>

        <!-- Headers -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <h3 class="text-sm font-semibold text-gray-400">Headers</h3>
            <button @click="addHeader" class="text-xs text-blue-400 hover:text-blue-300 transition">+ Add Header</button>
          </div>
          <div class="space-y-2">
            <div v-for="(header, index) in headers" :key="index" class="flex gap-2 items-center">
              <input
                v-model="header.key"
                type="text"
                placeholder="Header name"
                class="w-1/3 bg-gray-900 border border-gray-700 rounded px-2 py-1 text-sm font-mono text-gray-200 focus:outline-none focus:border-blue-500"
              />
              <input
                v-model="header.value"
                type="text"
                placeholder="Value"
                class="flex-1 bg-gray-900 border border-gray-700 rounded px-2 py-1 text-sm font-mono text-gray-200 focus:outline-none focus:border-blue-500"
              />
              <button @click="removeHeader(index)" class="text-gray-500 hover:text-red-400 transition text-sm px-1">&times;</button>
            </div>
            <div v-if="headers.length === 0" class="text-sm text-gray-600">No headers</div>
          </div>
        </div>

        <!-- Body -->
        <div>
          <h3 class="text-sm font-semibold text-gray-400 mb-2">Body</h3>
          <textarea
            v-model="body"
            rows="6"
            placeholder="Request body (plain text, will be base64-encoded on send)"
            class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm font-mono text-gray-200 focus:outline-none focus:border-blue-500 resize-y"
          ></textarea>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex justify-end gap-3 px-5 py-4 border-t border-gray-800">
        <button @click="$emit('close')" class="px-4 py-1.5 text-sm bg-gray-800 hover:bg-gray-700 rounded transition text-gray-300">
          Cancel
        </button>
        <button @click="send" class="px-4 py-1.5 text-sm bg-blue-600 hover:bg-blue-500 rounded transition text-white">
          Send
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CapturedExchange, ReplayRequest } from '../../api/client'

const props = defineProps<{
  exchange: CapturedExchange
}>()

const emit = defineEmits<{
  send: [mods: ReplayRequest]
  close: []
}>()

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']

const method = ref(props.exchange.method)
const path = ref(props.exchange.path)
const body = ref(decodeBody(props.exchange.request_body))

interface HeaderRow {
  key: string
  value: string
}

const headers = ref<HeaderRow[]>(buildHeaders(props.exchange.request_headers))

function buildHeaders(raw: Record<string, string[]> | null): HeaderRow[] {
  if (!raw) return []
  const rows: HeaderRow[] = []
  for (const [key, values] of Object.entries(raw)) {
    rows.push({ key, value: values.join(', ') })
  }
  return rows
}

function decodeBody(b: string | null): string {
  if (!b) return ''
  try {
    return atob(b)
  } catch {
    return b
  }
}

function addHeader() {
  headers.value.push({ key: '', value: '' })
}

function removeHeader(index: number) {
  headers.value.splice(index, 1)
}

function send() {
  const hdrs: Record<string, string[]> = {}
  for (const h of headers.value) {
    if (h.key.trim()) {
      hdrs[h.key.trim()] = [h.value]
    }
  }

  const mods: ReplayRequest = {
    method: method.value,
    path: path.value,
    headers: hdrs,
    body: body.value ? btoa(body.value) : undefined,
  }

  emit('send', mods)
}
</script>
