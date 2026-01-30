<template>
  <div class="flex flex-col h-full">
    <div class="flex items-center justify-between px-4 py-3 border-b border-border">
      <div class="flex items-center gap-3">
        <router-link to="/dashboard" class="text-muted-foreground hover:text-foreground transition">
          &larr;
        </router-link>
        <h1 class="text-lg font-semibold">Traffic Inspector</h1>
        <span v-if="connected" class="flex items-center gap-1.5 text-sm text-emerald-500">
          <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
          Live
        </span>
      </div>
      <button @click="clearExchanges" class="px-3 py-1.5 text-sm bg-secondary hover:bg-secondary/80 rounded transition">
        Clear
      </button>
    </div>

    <div class="flex flex-1 overflow-hidden">
      <div class="w-1/2 border-r border-border overflow-hidden flex flex-col">
        <div class="px-3 py-2 border-b border-border">
          <input
            v-model="filter"
            type="text"
            placeholder="Filter..."
            class="w-full bg-background border border-border rounded px-3 py-1.5 text-sm focus:outline-none focus:border-primary"
          />
        </div>
        <div class="flex-1 overflow-y-auto">
          <div v-if="filteredExchanges.length === 0" class="px-4 py-8 text-center text-muted-foreground text-sm">
            No requests captured yet
          </div>
          <div
            v-for="ex in filteredExchanges"
            :key="ex.id"
            @click="selectExchange(ex.id)"
            :class="[
              'flex items-center gap-2 px-3 py-1.5 cursor-pointer border-b border-border/50 text-sm hover:bg-accent/50 transition',
              selectedId === ex.id ? 'bg-accent' : ''
            ]"
          >
            <span :class="methodColor(ex.method)" class="font-mono font-semibold w-14 text-xs">{{ ex.method }}</span>
            <span class="flex-1 truncate font-mono text-foreground/80">{{ ex.path }}</span>
            <span :class="statusColor(ex.status_code)" class="font-mono text-xs px-1.5 py-0.5 rounded">{{ ex.status_code }}</span>
            <span class="text-muted-foreground text-xs w-14 text-right">{{ formatDuration(ex.duration_ns) }}</span>
          </div>
        </div>
      </div>

      <div class="w-1/2 overflow-y-auto">
        <div v-if="selectedExchange" class="p-4">
          <div class="flex items-center gap-2 mb-3">
            <span class="font-mono font-bold">{{ selectedExchange.method }}</span>
            <span class="font-mono text-foreground/70">{{ selectedExchange.path }}</span>
            <span :class="statusColor(selectedExchange.status_code)" class="font-mono text-xs px-1.5 py-0.5 rounded">{{ selectedExchange.status_code }}</span>
          </div>

          <div class="flex border-b border-border mb-3">
            <button v-for="tab in ['Request', 'Response']" :key="tab" @click="activeTab = tab"
              :class="['px-3 py-1.5 text-sm border-b-2 -mb-px transition', activeTab === tab ? 'border-primary text-primary' : 'border-transparent text-muted-foreground']">
              {{ tab }}
            </button>
          </div>

          <template v-if="activeTab === 'Request'">
            <h3 class="text-sm font-semibold text-muted-foreground mb-2">Headers</h3>
            <div class="bg-card rounded border border-border mb-3 text-sm">
              <div v-for="(values, name) in selectedExchange.request_headers" :key="name" class="flex border-b border-border/50 last:border-0">
                <span class="px-2 py-1 text-muted-foreground font-mono w-48 shrink-0">{{ name }}</span>
                <span class="px-2 py-1 font-mono break-all">{{ values?.join(', ') }}</span>
              </div>
            </div>
            <h3 class="text-sm font-semibold text-muted-foreground mb-2">Body</h3>
            <pre class="bg-card rounded border border-border p-3 text-sm font-mono overflow-auto max-h-64 whitespace-pre-wrap">{{ formatBody(selectedExchange.request_body, selectedExchange.request_headers) }}</pre>
          </template>

          <template v-if="activeTab === 'Response'">
            <h3 class="text-sm font-semibold text-muted-foreground mb-2">Headers</h3>
            <div class="bg-card rounded border border-border mb-3 text-sm">
              <div v-for="(values, name) in selectedExchange.response_headers" :key="name" class="flex border-b border-border/50 last:border-0">
                <span class="px-2 py-1 text-muted-foreground font-mono w-48 shrink-0">{{ name }}</span>
                <span class="px-2 py-1 font-mono break-all">{{ values?.join(', ') }}</span>
              </div>
            </div>
            <h3 class="text-sm font-semibold text-muted-foreground mb-2">Body</h3>
            <pre class="bg-card rounded border border-border p-3 text-sm font-mono overflow-auto max-h-64 whitespace-pre-wrap">{{ formatBody(selectedExchange.response_body, selectedExchange.response_headers) }}</pre>
          </template>
        </div>
        <div v-else class="flex items-center justify-center h-full text-muted-foreground">
          Select a request to view details
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { List, Get, Clear, Subscribe, Unsubscribe } from '@/wailsjs/go/gui/InspectService'
import { EventsOn, EventsOff } from '@/wailsjs/runtime/runtime'

const route = useRoute()
const tunnelId = computed(() => route.params.tunnelId as string)

interface ExchangeSummary {
  id: string; tunnel_id: string; timestamp: string; duration_ns: number
  method: string; path: string; host: string; status_code: number
  request_body_size: number; response_body_size: number; remote_addr: string
}

const exchanges = ref<ExchangeSummary[]>([])
const selectedId = ref<string | null>(null)
const selectedExchange = ref<any>(null)
const filter = ref('')
const activeTab = ref('Request')
const connected = ref(false)

const filteredExchanges = computed(() => {
  if (!filter.value) return exchanges.value
  const q = filter.value.toLowerCase()
  return exchanges.value.filter(ex =>
    ex.path.toLowerCase().includes(q) || ex.method.toLowerCase().includes(q) || String(ex.status_code).includes(q)
  )
})

function methodColor(method: string): string {
  const c: Record<string, string> = { GET: 'text-emerald-500', POST: 'text-blue-500', PUT: 'text-amber-500', DELETE: 'text-red-500' }
  return c[method] || 'text-muted-foreground'
}

function statusColor(status: number): string {
  if (status >= 500) return 'bg-red-500/20 text-red-400'
  if (status >= 400) return 'bg-amber-500/20 text-amber-400'
  if (status >= 200) return 'bg-emerald-500/20 text-emerald-400'
  return 'bg-secondary text-muted-foreground'
}

function formatDuration(ns: number): string {
  const ms = ns / 1_000_000
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatBody(body: any, headers: Record<string, string[]> | null): string {
  if (!body) return '(empty)'
  const str = typeof body === 'string' ? body : JSON.stringify(body)
  try {
    const decoded = atob(str)
    const ct = headers?.['Content-Type']?.[0] || headers?.['content-type']?.[0] || ''
    if (ct.includes('json')) {
      try { return JSON.stringify(JSON.parse(decoded), null, 2) } catch { return decoded }
    }
    return decoded
  } catch {
    return str
  }
}

async function loadExchanges() {
  try {
    const data = await List(tunnelId.value, 0, 50)
    exchanges.value = data.exchanges || []
    connected.value = true
  } catch (e) { console.error(e) }
}

async function selectExchange(id: string) {
  selectedId.value = id
  try {
    selectedExchange.value = await Get(tunnelId.value, id)
  } catch (e) { console.error(e) }
}

async function clearExchanges() {
  try {
    await Clear(tunnelId.value)
    exchanges.value = []
    selectedId.value = null
    selectedExchange.value = null
  } catch (e) { console.error(e) }
}

onMounted(async () => {
  await loadExchanges()
  try {
    await Subscribe(tunnelId.value)
    connected.value = true
  } catch (e) { console.error(e) }

  EventsOn('inspect_exchange', (ex: ExchangeSummary) => {
    exchanges.value.unshift(ex)
  })
  EventsOn('inspect_disconnected', () => {
    connected.value = false
  })
})

onUnmounted(() => {
  Unsubscribe(tunnelId.value)
  EventsOff('inspect_exchange')
  EventsOff('inspect_disconnected')
})
</script>
