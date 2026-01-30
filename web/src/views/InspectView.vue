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
        <ExchangeDetail v-if="selectedExchange" :exchange="selectedExchange" />
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
import { inspectApi, type ExchangeSummary, type CapturedExchange } from '../api/client'
import ExchangeList from '../components/inspect/ExchangeList.vue'
import ExchangeDetail from '../components/inspect/ExchangeDetail.vue'

const route = useRoute()
const tunnelId = computed(() => route.params.tunnelId as string)

const exchanges = ref<ExchangeSummary[]>([])
const selectedId = ref<string | null>(null)
const selectedExchange = ref<CapturedExchange | null>(null)
const filter = ref('')
const connected = ref(false)
let eventSource: EventSource | null = null

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
  } catch (e) {
    console.error('Failed to clear:', e)
  }
}

function connectSSE() {
  const token = localStorage.getItem('access_token')
  const url = `/api/tunnels/${tunnelId.value}/inspect/stream${token ? `?token=${token}` : ''}`
  eventSource = new EventSource(url)

  eventSource.onopen = () => { connected.value = true }
  eventSource.onerror = () => { connected.value = false }

  eventSource.addEventListener('exchange', (event: MessageEvent) => {
    const ex: ExchangeSummary = JSON.parse(event.data)
    exchanges.value.unshift(ex)
  })
}

onMounted(() => {
  loadExchanges()
  connectSSE()
})

onUnmounted(() => {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
})
</script>
