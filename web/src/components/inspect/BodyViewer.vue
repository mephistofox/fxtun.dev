<template>
  <div class="bg-gray-900 rounded border border-gray-800 overflow-hidden">
    <div v-if="!body" class="px-4 py-3 text-gray-600 text-sm">No body</div>
    <div v-else>
      <div v-if="truncated" class="px-3 py-1 bg-amber-900/30 text-amber-400 text-xs">
        Body truncated (showing {{ formatSize(rawBytes.length) }} of {{ formatSize(bodySize) }})
      </div>
      <!-- JSON -->
      <pre v-if="isJson" class="p-3 text-sm font-mono text-gray-200 overflow-x-auto max-h-96 whitespace-pre-wrap">{{ formattedJson }}</pre>
      <!-- Plain text -->
      <pre v-else-if="isText" class="p-3 text-sm font-mono text-gray-200 overflow-x-auto max-h-96 whitespace-pre-wrap">{{ decodedBody }}</pre>
      <!-- Binary -->
      <div v-else class="p-3 text-sm font-mono text-gray-400">
        <div class="mb-1 text-gray-500">Binary data ({{ formatSize(bodySize) }})</div>
        <pre class="overflow-x-auto max-h-48">{{ hexDump }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  body: string | null
  contentType: string
  bodySize: number
}>()

// Decode base64 to Uint8Array once
const rawBytes = computed(() => {
  if (!props.body) return new Uint8Array(0)
  try {
    const bin = atob(props.body)
    const bytes = new Uint8Array(bin.length)
    for (let i = 0; i < bin.length; i++) {
      bytes[i] = bin.charCodeAt(i)
    }
    return bytes
  } catch {
    return new Uint8Array(0)
  }
})

const truncated = computed(() => props.body && props.bodySize > rawBytes.value.length)

const isJson = computed(() => {
  if (!props.contentType) return false
  return props.contentType.includes('json')
})

const isText = computed(() => {
  if (!props.contentType) return true
  return props.contentType.includes('text') || props.contentType.includes('xml') || props.contentType.includes('html') || props.contentType.includes('form-urlencoded')
})

// Decode bytes as UTF-8 text
const decodedBody = computed(() => {
  if (rawBytes.value.length === 0) return ''
  return new TextDecoder('utf-8').decode(rawBytes.value)
})

const formattedJson = computed(() => {
  try {
    return JSON.stringify(JSON.parse(decodedBody.value), null, 2)
  } catch {
    return decodedBody.value
  }
})

const hexDump = computed(() => {
  const bytes = rawBytes.value
  if (bytes.length === 0) return ''
  const lines: string[] = []
  const limit = Math.min(bytes.length, 256)
  for (let i = 0; i < limit; i += 16) {
    const chunk = bytes.slice(i, i + 16)
    const hex = Array.from(chunk)
      .map(b => b.toString(16).padStart(2, '0'))
      .join(' ')
    const ascii = Array.from(chunk)
      .map(b => (b >= 32 && b < 127) ? String.fromCharCode(b) : '.')
      .join('')
    lines.push(`${i.toString(16).padStart(8, '0')}  ${hex.padEnd(48)}  ${ascii}`)
  }
  if (bytes.length > limit) lines.push('...')
  return lines.join('\n')
})

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>
