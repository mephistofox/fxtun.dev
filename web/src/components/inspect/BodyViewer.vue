<template>
  <div class="bg-gray-900 rounded border border-gray-800 overflow-hidden">
    <div v-if="!body" class="px-4 py-3 text-gray-600 text-sm">No body</div>
    <div v-else>
      <div v-if="truncated" class="px-3 py-1 bg-amber-900/30 text-amber-400 text-xs">
        Body truncated (showing {{ formatSize(body.length) }} of {{ formatSize(bodySize) }})
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

const truncated = computed(() => props.body && props.bodySize > props.body.length)

const isJson = computed(() => {
  if (!props.contentType) return false
  return props.contentType.includes('json')
})

const isText = computed(() => {
  if (!props.contentType) return true
  return props.contentType.includes('text') || props.contentType.includes('xml') || props.contentType.includes('html') || props.contentType.includes('form-urlencoded')
})

const decodedBody = computed(() => {
  if (!props.body) return ''
  try {
    return atob(props.body)
  } catch {
    return props.body
  }
})

const formattedJson = computed(() => {
  try {
    return JSON.stringify(JSON.parse(decodedBody.value), null, 2)
  } catch {
    return decodedBody.value
  }
})

const hexDump = computed(() => {
  if (!props.body) return ''
  try {
    const raw = atob(props.body)
    const lines: string[] = []
    const limit = Math.min(raw.length, 256)
    for (let i = 0; i < limit; i += 16) {
      const hex = Array.from(raw.slice(i, i + 16))
        .map(c => c.charCodeAt(0).toString(16).padStart(2, '0'))
        .join(' ')
      const ascii = Array.from(raw.slice(i, i + 16))
        .map(c => { const code = c.charCodeAt(0); return code >= 32 && code < 127 ? c : '.' })
        .join('')
      lines.push(`${i.toString(16).padStart(8, '0')}  ${hex.padEnd(48)}  ${ascii}`)
    }
    if (raw.length > limit) lines.push('...')
    return lines.join('\n')
  } catch {
    return 'Unable to decode'
  }
})

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>
