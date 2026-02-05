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
            <span v-if="ex.replay_ref" class="text-amber-400 text-xs" title="Replayed request">&#8635;</span>
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
            <span v-if="selectedExchange.replay_ref" class="text-xs bg-amber-500/20 text-amber-400 px-1.5 py-0.5 rounded font-mono">
              replay of {{ selectedExchange.replay_ref.slice(0, 8) }}
            </span>
            <div class="ml-auto flex items-center gap-2">
              <button
                @click="openEditReplay(selectedExchange)"
                :disabled="replaying"
                class="px-3 py-1 text-xs bg-secondary hover:bg-secondary/80 disabled:opacity-50 rounded transition"
              >
                Edit &amp; Replay
              </button>
              <button
                @click="replayExchange(selectedExchange.id)"
                :disabled="replaying"
                class="px-3 py-1 text-xs bg-primary hover:bg-primary/80 disabled:opacity-50 rounded transition text-primary-foreground"
              >
                {{ replaying ? 'Replaying...' : 'Replay' }}
              </button>
            </div>
          </div>

          <!-- Replay editor panel -->
          <div v-if="editReplayVisible" class="mb-4 bg-card rounded border border-border p-3">
            <div class="flex items-center justify-between mb-3">
              <h3 class="text-sm font-semibold">Edit &amp; Replay</h3>
              <button @click="editReplayVisible = false" class="text-muted-foreground hover:text-foreground text-xs">Cancel</button>
            </div>
            <div class="flex gap-2 mb-2">
              <select v-model="editMods.method" class="bg-background border border-border rounded px-2 py-1 text-sm font-mono">
                <option v-for="m in ['GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS']" :key="m" :value="m">{{ m }}</option>
              </select>
              <input v-model="editMods.path" class="flex-1 bg-background border border-border rounded px-2 py-1 text-sm font-mono" placeholder="/path" />
            </div>
            <div class="mb-2">
              <label class="text-xs text-muted-foreground">Headers (JSON)</label>
              <textarea v-model="editMods.headersRaw" rows="3" class="w-full bg-background border border-border rounded px-2 py-1 text-sm font-mono mt-1" placeholder='{"Content-Type": ["application/json"]}'></textarea>
            </div>
            <div class="mb-3">
              <label class="text-xs text-muted-foreground">Body</label>
              <textarea v-model="editMods.body" rows="4" class="w-full bg-background border border-border rounded px-2 py-1 text-sm font-mono mt-1" placeholder="Request body..."></textarea>
            </div>
            <button
              @click="sendEditReplay"
              :disabled="replaying"
              class="px-3 py-1.5 text-sm bg-primary hover:bg-primary/80 disabled:opacity-50 rounded transition text-primary-foreground"
            >
              {{ replaying ? 'Sending...' : 'Send Modified Request' }}
            </button>
          </div>

          <!-- Replay response inline -->
          <div v-if="replayResult" class="mb-4 bg-card rounded border border-amber-500/30 p-3">
            <div class="flex items-center gap-2 mb-2">
              <h3 class="text-sm font-semibold text-amber-400">Replay Response</h3>
              <span :class="statusColor(replayResult.status_code)" class="font-mono text-xs px-1.5 py-0.5 rounded">{{ replayResult.status_code }}</span>
              <button
                v-if="replayResult.exchange_id"
                @click="selectExchange(replayResult.exchange_id)"
                class="ml-auto text-xs text-primary hover:text-primary/80 transition"
              >
                View full exchange
              </button>
            </div>
            <div v-if="replayResult.response_headers" class="mb-2">
              <h4 class="text-xs text-muted-foreground mb-1">Headers</h4>
              <div class="bg-background rounded border border-border text-xs">
                <div v-for="(values, name) in replayResult.response_headers" :key="name" class="flex border-b border-border/50 last:border-0">
                  <span class="px-2 py-0.5 text-muted-foreground font-mono w-40 shrink-0">{{ name }}</span>
                  <span class="px-2 py-0.5 font-mono break-all">{{ values?.join(', ') }}</span>
                </div>
              </div>
            </div>
            <div>
              <h4 class="text-xs text-muted-foreground mb-1">Body</h4>
              <pre class="bg-background rounded border border-border p-2 text-xs font-mono overflow-auto max-h-48 whitespace-pre-wrap">{{ formatBody(replayResult.response_body, replayResult.response_headers) }}</pre>
            </div>
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
import { List, Get, Clear, Replay, Subscribe, Unsubscribe } from '@/wailsjs/wailsjs/go/gui/InspectService'
import { EventsOn, EventsOff } from '@/wailsjs/wailsjs/runtime/runtime'

const route = useRoute()
const tunnelId = computed(() => route.params.tunnelId as string)

interface ExchangeSummary {
  id: string; tunnel_id: string; timestamp: string; duration_ns: number
  method: string; path: string; host: string; status_code: number
  request_body_size: number; response_body_size: number; remote_addr: string
  replay_ref?: string
}

interface ReplayResponseData {
  status_code: number
  response_headers: Record<string, string[]>
  response_body: any
  exchange_id: string
}

const exchanges = ref<ExchangeSummary[]>([])
const selectedId = ref<string | null>(null)
const selectedExchange = ref<any>(null)
const filter = ref('')
const activeTab = ref('Request')
const connected = ref(false)
const replaying = ref(false)
const replayResult = ref<ReplayResponseData | null>(null)
const editReplayVisible = ref(false)
const editMods = ref({
  method: 'GET',
  path: '/',
  headersRaw: '',
  body: ''
})

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
  replayResult.value = null
  editReplayVisible.value = false
  try {
    selectedExchange.value = await Get(tunnelId.value, id)
  } catch (e) { console.error(e) }
}

async function replayExchange(exchangeId: string) {
  replaying.value = true
  replayResult.value = null
  try {
    const result = await Replay(tunnelId.value, exchangeId, null) as ReplayResponseData
    replayResult.value = result
    // Auto-select the new exchange if it appeared in the list
    if (result.exchange_id) {
      // Give SSE a moment to deliver the new exchange, then select it
      setTimeout(() => {
        const found = exchanges.value.find(ex => ex.id === result.exchange_id)
        if (found) {
          selectExchange(result.exchange_id)
        }
      }, 500)
    }
  } catch (e) { console.error('Replay failed:', e) }
  finally { replaying.value = false }
}

function openEditReplay(exchange: any) {
  editMods.value = {
    method: exchange.method || 'GET',
    path: exchange.path || '/',
    headersRaw: exchange.request_headers ? JSON.stringify(exchange.request_headers, null, 2) : '',
    body: formatBody(exchange.request_body, exchange.request_headers)
  }
  if (editMods.value.body === '(empty)') editMods.value.body = ''
  editReplayVisible.value = true
}

async function sendEditReplay() {
  if (!selectedExchange.value) return
  replaying.value = true
  replayResult.value = null
  try {
    const mods: any = {}
    if (editMods.value.method !== selectedExchange.value.method) {
      mods.method = editMods.value.method
    }
    if (editMods.value.path !== selectedExchange.value.path) {
      mods.path = editMods.value.path
    }
    if (editMods.value.headersRaw.trim()) {
      try {
        mods.headers = JSON.parse(editMods.value.headersRaw)
      } catch { /* ignore parse errors, send without header mods */ }
    }
    if (editMods.value.body) {
      mods.body = btoa(editMods.value.body)
    }

    const result = await Replay(tunnelId.value, selectedExchange.value.id, mods) as ReplayResponseData
    replayResult.value = result
    editReplayVisible.value = false
    if (result.exchange_id) {
      setTimeout(() => {
        const found = exchanges.value.find(ex => ex.id === result.exchange_id)
        if (found) {
          selectExchange(result.exchange_id)
        }
      }, 500)
    }
  } catch (e) { console.error('Edit & Replay failed:', e) }
  finally { replaying.value = false }
}

async function clearExchanges() {
  try {
    await Clear(tunnelId.value)
    exchanges.value = []
    selectedId.value = null
    selectedExchange.value = null
    replayResult.value = null
    editReplayVisible.value = false
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
