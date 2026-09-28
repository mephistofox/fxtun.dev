<template>
  <div class="min-h-screen bg-gray-950 text-gray-100">
    <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
      <div class="flex items-center gap-4">
        <router-link to="/dashboard" class="text-gray-400 hover:text-white transition">
          &larr; Dashboard
        </router-link>
        <h1 class="text-xl font-semibold">Traffic Inspector</h1>
        <span class="text-sm text-gray-500">Tunnel: {{ tunnelId }}</span>
      </div>
      <div class="flex items-center gap-3">
        <span v-if="connected" class="flex items-center gap-1.5 text-sm text-emerald-400">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          Live
        </span>
        <span v-else class="flex items-center gap-1.5 text-sm text-gray-500">
          <span class="w-2 h-2 rounded-full bg-gray-500"></span>
          Disconnected
        </span>
        <button @click="clearExchanges" class="px-3 py-1.5 text-sm bg-gray-800 hover:bg-gray-700 rounded transition">
          Clear
        </button>
      </div>
    </div>

    <div class="flex h-[calc(100vh-65px)]">
      <!-- Left: Exchange List -->
      <div class="w-1/2 border-r border-gray-800 overflow-hidden flex flex-col">
        <div class="px-4 py-2 border-b border-gray-800">
          <input
            v-model="filter"
            type="text"
            placeholder="Filter by path, method, status..."
            class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm focus:outline-none focus:border-blue-500"
          />
        </div>
        <div class="flex-1 overflow-y-auto">
          <ExchangeList
            :exchanges="filteredExchanges"
            :selected-id="selectedId"
            @select="selectExchange"
          />
        </div>
      </div>

      <!-- Right: Exchange Detail -->
      <div class="w-1/2 overflow-y-auto">
        <ExchangeDetail
          v-if="selectedExchange"
          :exchange="selectedExchange"
          :replaying="replaying"
          :replay-result="replayResponse"
          @replay="replayExchange"
          @edit-replay="editReplayExchange"
        />
        <div v-else class="flex items-center justify-center h-full text-gray-500">
          Select a request to view details
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { inspectApi, type ExchangeSummary, type CapturedExchange, type ReplayRequest, type ReplayResponse } from '../api/client'
import ExchangeList from '../components/inspect/ExchangeList.vue'
import ExchangeDetail from '../components/inspect/ExchangeDetail.vue'

const route = useRoute()
const tunnelId = computed(() => route.params.tunnelId as string)

const exchanges = ref<ExchangeSummary[]>([])
const selectedId = ref<string | null>(null)
const selectedExchange = ref<CapturedExchange | null>(null)
const filter = ref('')
const connected = ref(false)
const replaying = ref(false)
const replayResponse = ref<ReplayResponse | null>(null)

const filteredExchanges = computed(() => {
  if (!filter.value) return exchanges.value
  const q = filter.value.toLowerCase()
  return exchanges.value.filter(ex =>
    ex.path.toLowerCase().includes(q) ||
    ex.method.toLowerCase().includes(q) ||
    String(ex.status_code).includes(q)
  )
})

async function loadExchanges() {
  try {
    const data = await inspectApi.list(tunnelId.value)
    exchanges.value = data.exchanges || []
  } catch (e) {
    console.error('Failed to load exchanges:', e)
  }
}

async function selectExchange(id: string) {
  selectedId.value = id
  replayResponse.value = null
  try {
    selectedExchange.value = await inspectApi.get(tunnelId.value, id)
  } catch (e) {
    console.error('Failed to load exchange:', e)
  }
}

async function clearExchanges() {
  try {
    await inspectApi.clear(tunnelId.value)
    exchanges.value = []
    selectedId.value = null
    selectedExchange.value = null
    replayResponse.value = null
  } catch (e) {
    console.error('Failed to clear:', e)
  }
}

async function replayExchange(exchangeId: string, mods?: ReplayRequest) {
  replaying.value = true
  replayResponse.value = null
  try {
    const response = await inspectApi.replay(tunnelId.value, exchangeId, mods)
    replayResponse.value = response
    // Auto-select the newly created exchange
    if (response.exchange_id) {
      await selectExchange(response.exchange_id)
    }
  } catch (e) {
    console.error('Replay failed:', e)
  } finally {
    replaying.value = false
  }
}

async function editReplayExchange(mods: ReplayRequest) {
  if (!selectedId.value) return
  await replayExchange(selectedId.value, mods)
}

let sseAbort: AbortController | null = null

function connectSSE() {
  const token = localStorage.getItem('accessToken')
  const url = `/api/tunnels/${tunnelId.value}/inspect/stream`

  sseAbort = new AbortController()
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`

  fetch(url, { headers, signal: sseAbort.signal })
    .then(response => {
      if (!response.ok || !response.body) {
        connected.value = false
        return
      }
      connected.value = true
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      function read(): Promise<void> {
        return reader.read().then(({ done, value }) => {
          if (done) {
            connected.value = false
            return
          }
          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''
          let eventType = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) {
              eventType = line.slice(7).trim()
            } else if (line.startsWith('data: ') && eventType === 'exchange') {
              try {
                const ex: ExchangeSummary = JSON.parse(line.slice(6))
                exchanges.value.unshift(ex)
              } catch { /* skip malformed */ }
              eventType = ''
            } else if (line === '') {
              eventType = ''
            }
          }
          return read()
        })
      }
      return read()
    })
    .catch(() => {
      connected.value = false
    })
}

onMounted(() => {
  loadExchanges()
  connectSSE()
})

onUnmounted(() => {
  if (sseAbort) {
    sseAbort.abort()
    sseAbort = null
  }
})
</script>
