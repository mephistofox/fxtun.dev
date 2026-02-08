<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface MockExchange {
  id: number
  method: string
  path: string
  status: number
  durationMs: number
  timestamp: Date
  isReplay: boolean
  reqHeaders: Record<string, string>
  reqBody: string | null
  resHeaders: Record<string, string>
  resBody: string
}

const mockPool: Omit<MockExchange, 'id' | 'timestamp' | 'isReplay'>[] = [
  {
    method: 'POST', path: '/api/webhooks/stripe', status: 200, durationMs: 23,
    reqHeaders: { 'Content-Type': 'application/json', 'Stripe-Signature': 'whsec_t2Lk9x...mN3Q' },
    reqBody: '{\n  "type": "checkout.session.completed",\n  "data": {\n    "object": {\n      "id": "cs_live_a1B2c3",\n      "amount_total": 2500,\n      "currency": "usd"\n    }\n  }\n}',
    resHeaders: { 'Content-Type': 'application/json', 'X-Request-Id': 'req_8kL2mN' },
    resBody: '{ "received": true }'
  },
  {
    method: 'GET', path: '/api/users/me', status: 200, durationMs: 12,
    reqHeaders: { 'Authorization': 'Bearer sk_live_4eC39H...', 'Accept': 'application/json' },
    reqBody: null,
    resHeaders: { 'Content-Type': 'application/json', 'Cache-Control': 'no-cache' },
    resBody: '{\n  "id": 1,\n  "name": "Alex",\n  "email": "alex@example.com",\n  "plan": "pro"\n}'
  },
  {
    method: 'PUT', path: '/api/settings', status: 401, durationMs: 8,
    reqHeaders: { 'Content-Type': 'application/json' },
    reqBody: '{ "theme": "dark" }',
    resHeaders: { 'Content-Type': 'application/json', 'WWW-Authenticate': 'Bearer' },
    resBody: '{ "error": "Unauthorized", "message": "Token expired" }'
  },
  {
    method: 'POST', path: '/api/orders', status: 201, durationMs: 45,
    reqHeaders: { 'Content-Type': 'application/json', 'Authorization': 'Bearer sk_live_4eC39H...', 'Idempotency-Key': 'ord_9xK2mN' },
    reqBody: '{\n  "items": [{"sku": "PROD-001", "qty": 2}],\n  "shipping": "express"\n}',
    resHeaders: { 'Content-Type': 'application/json', 'Location': '/api/orders/847' },
    resBody: '{\n  "id": 847,\n  "status": "created",\n  "total": "$49.98"\n}'
  },
  {
    method: 'DELETE', path: '/api/sessions/old', status: 500, durationMs: 120,
    reqHeaders: { 'Authorization': 'Bearer sk_live_4eC39H...' },
    reqBody: null,
    resHeaders: { 'Content-Type': 'application/json' },
    resBody: '{\n  "error": "Internal Server Error",\n  "trace_id": "tr_xK2mN9"\n}'
  },
  {
    method: 'GET', path: '/health', status: 200, durationMs: 2,
    reqHeaders: { 'Accept': 'application/json', 'User-Agent': 'kube-probe/1.28' },
    reqBody: null,
    resHeaders: { 'Content-Type': 'application/json' },
    resBody: '{ "status": "ok", "uptime": "4d 12h" }'
  },
  {
    method: 'POST', path: '/api/webhooks/github', status: 200, durationMs: 34,
    reqHeaders: { 'Content-Type': 'application/json', 'X-GitHub-Event': 'push', 'X-Hub-Signature-256': 'sha256=d4f5...' },
    reqBody: '{\n  "ref": "refs/heads/main",\n  "commits": [{\n    "message": "fix: resolve auth issue",\n    "author": { "name": "alex" }\n  }]\n}',
    resHeaders: { 'Content-Type': 'application/json' },
    resBody: '{ "ok": true, "deployed": true }'
  },
  {
    method: 'PATCH', path: '/api/users/42', status: 404, durationMs: 15,
    reqHeaders: { 'Content-Type': 'application/json', 'Authorization': 'Bearer sk_live_4eC39H...' },
    reqBody: '{ "name": "Updated Name" }',
    resHeaders: { 'Content-Type': 'application/json' },
    resBody: '{ "error": "Not Found" }'
  },
]

let nextId = 0
const exchanges = ref<MockExchange[]>([])
const selectedId = ref<number | null>(null)
const replayingId = ref<number | null>(null)
const replayResult = ref<{ status: number; body: string } | null>(null)
const activeTab = ref<'request' | 'response'>('request')
const showReplayResult = ref(false)
const editMode = ref(false)
const editMethod = ref('')
const editPath = ref('')
const editBody = ref('')
let intervalHandle: ReturnType<typeof setInterval> | null = null

const selectedExchange = computed(() =>
  exchanges.value.find(r => r.id === selectedId.value) ?? null
)

function pickRandom(): MockExchange {
  const item = mockPool[Math.floor(Math.random() * mockPool.length)]
  return {
    ...item,
    id: nextId++,
    timestamp: new Date(),
    isReplay: false,
  }
}

function addExchange() {
  exchanges.value.unshift(pickRandom())
  if (exchanges.value.length > 8) {
    exchanges.value.pop()
  }
}

function selectRow(id: number) {
  selectedId.value = id
  replayResult.value = null
  showReplayResult.value = false
  editMode.value = false
  activeTab.value = 'request'
}

function openEditor() {
  if (!selectedExchange.value) return
  editMethod.value = selectedExchange.value.method
  editPath.value = selectedExchange.value.path
  editBody.value = selectedExchange.value.reqBody || ''
  editMode.value = true
  showReplayResult.value = false
}

function sendEditReplay() {
  if (!selectedExchange.value) return
  const id = selectedExchange.value.id
  replayingId.value = id
  editMode.value = false
  setTimeout(() => {
    const original = exchanges.value.find(e => e.id === id)
    if (original) {
      replayResult.value = {
        status: original.status,
        body: original.resBody,
      }
      showReplayResult.value = true
      const replayExchange: MockExchange = {
        ...original,
        method: editMethod.value,
        path: editPath.value,
        reqBody: editBody.value || null,
        id: nextId++,
        timestamp: new Date(),
        isReplay: true,
      }
      exchanges.value.unshift(replayExchange)
      if (exchanges.value.length > 8) exchanges.value.pop()
    }
    replayingId.value = null
  }, 600)
}

function replay(id: number) {
  replayingId.value = id
  showReplayResult.value = false
  setTimeout(() => {
    const original = exchanges.value.find(e => e.id === id)
    if (original) {
      // Show replay result
      replayResult.value = {
        status: original.status,
        body: original.resBody,
      }
      showReplayResult.value = true
      // Add replay exchange to list
      const replayExchange: MockExchange = {
        ...original,
        id: nextId++,
        timestamp: new Date(),
        isReplay: true,
      }
      exchanges.value.unshift(replayExchange)
      if (exchanges.value.length > 8) exchanges.value.pop()
    }
    replayingId.value = null
  }, 600)
}

function methodColor(method: string): string {
  const colors: Record<string, string> = {
    GET: 'text-emerald-400',
    POST: 'text-blue-400',
    PUT: 'text-amber-400',
    PATCH: 'text-orange-400',
    DELETE: 'text-red-400',
  }
  return colors[method] || 'text-muted-foreground'
}

function statusBadgeClass(status: number): string {
  if (status >= 500) return 'bg-red-500/15 text-red-400 border-red-500/20'
  if (status >= 400) return 'bg-amber-500/15 text-amber-400 border-amber-500/20'
  if (status >= 200 && status < 300) return 'bg-emerald-500/15 text-emerald-400 border-emerald-500/20'
  return 'bg-muted/50 text-muted-foreground border-border'
}

function formatDuration(ms: number): string {
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

onMounted(() => {
  for (let i = 0; i < 4; i++) {
    exchanges.value.push(pickRandom())
  }
  selectedId.value = exchanges.value[0]?.id ?? null
  intervalHandle = setInterval(addExchange, 3000)
})

onUnmounted(() => {
  if (intervalHandle) clearInterval(intervalHandle)
})
</script>

<template>
  <div class="rounded-xl border border-border bg-code overflow-hidden font-mono text-xs h-[360px] flex flex-col shadow-2xl">
    <!-- Toolbar -->
    <div class="flex items-center gap-3 px-3 py-2 border-b border-border/70 bg-code-header">
      <div class="flex items-center gap-2">
        <span class="relative flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
          <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
        </span>
        <span class="text-emerald-400 text-[10px] font-medium">Live</span>
      </div>
      <div class="h-3 w-px bg-border/50" />
      <span class="text-foreground/60 text-[11px]">my-app.fxtun.dev</span>
      <div class="ml-auto flex items-center gap-1.5">
        <span class="text-[10px] text-muted-foreground/60 tabular-nums">{{ exchanges.length }} {{ t('landing.advanced.inspector.demo.requests') }}</span>
      </div>
    </div>

    <!-- Split panel -->
    <div class="flex flex-1 overflow-hidden min-h-0">
      <!-- Left: Exchange List -->
      <div class="w-full lg:w-[45%] border-r border-border/40 flex flex-col overflow-hidden">
        <div class="flex-1 overflow-y-auto">
          <TransitionGroup name="req" tag="div" class="relative">
            <div
              v-for="ex in exchanges"
              :key="ex.id"
              class="req-row flex items-center gap-1.5 px-2.5 py-[7px] border-b border-border/30 transition-colors cursor-pointer"
              :class="selectedId === ex.id ? 'bg-primary/10 border-l-2 border-l-primary' : 'border-l-2 border-l-transparent hover:bg-foreground/[0.03]'"
              @click="selectRow(ex.id)"
            >
              <!-- Method -->
              <span class="w-12 font-bold text-[10px] shrink-0" :class="methodColor(ex.method)">{{ ex.method }}</span>
              <!-- Path -->
              <span class="text-foreground/70 truncate flex-1 min-w-0 text-[11px]">{{ ex.path }}</span>
              <!-- Status badge -->
              <span
                class="shrink-0 text-[10px] font-medium px-1.5 py-0.5 rounded border"
                :class="statusBadgeClass(ex.status)"
              >{{ ex.status }}</span>
              <!-- Duration -->
              <span class="hidden sm:inline shrink-0 w-10 text-right text-[10px] text-muted-foreground/60 tabular-nums">{{ formatDuration(ex.durationMs) }}</span>
              <!-- Replay indicator -->
              <span
                v-if="ex.isReplay"
                class="shrink-0 text-purple-400 text-xs"
                title="Replayed"
              >&#8635;</span>
            </div>
          </TransitionGroup>
        </div>
      </div>

      <!-- Right: Detail Panel -->
      <div class="hidden lg:flex lg:w-[55%] flex-col overflow-hidden">
        <template v-if="selectedExchange">
          <!-- Detail header -->
          <div class="flex items-center gap-2 px-3 py-2 border-b border-border/40 bg-code-header">
            <span class="font-bold text-[11px]" :class="methodColor(selectedExchange.method)">{{ selectedExchange.method }}</span>
            <span class="text-foreground/70 text-[11px] truncate flex-1">{{ selectedExchange.path }}</span>
            <span
              class="text-[10px] font-medium px-1.5 py-0.5 rounded border"
              :class="statusBadgeClass(selectedExchange.status)"
            >{{ selectedExchange.status }}</span>
            <span class="text-[10px] text-muted-foreground/60 tabular-nums">{{ formatDuration(selectedExchange.durationMs) }}</span>
          </div>

          <!-- Tabs: Request / Response -->
          <div class="flex border-b border-border/40">
            <button
              @click="activeTab = 'request'"
              class="flex-1 py-1.5 text-[10px] font-medium transition-colors border-b-2 -mb-px"
              :class="activeTab === 'request' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
            >
              Request
            </button>
            <button
              @click="activeTab = 'response'"
              class="flex-1 py-1.5 text-[10px] font-medium transition-colors border-b-2 -mb-px"
              :class="activeTab === 'response' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
            >
              Response
            </button>
          </div>

          <!-- Tab content / Edit mode -->
          <div class="flex-1 overflow-y-auto p-2.5 space-y-2">
            <!-- Edit & Replay mode -->
            <template v-if="editMode">
              <p class="text-[9px] text-muted-foreground/60 uppercase tracking-wider font-semibold">Method</p>
              <select
                v-model="editMethod"
                class="w-full text-[10px] bg-foreground/[0.03] border border-border/40 rounded-md px-2 py-1.5 text-foreground/80 focus:outline-none focus:border-primary/40"
              >
                <option v-for="m in ['GET','POST','PUT','PATCH','DELETE']" :key="m" :value="m">{{ m }}</option>
              </select>

              <p class="text-[9px] text-muted-foreground/60 uppercase tracking-wider font-semibold pt-1">Path</p>
              <input
                v-model="editPath"
                class="w-full text-[10px] bg-foreground/[0.03] border border-border/40 rounded-md px-2 py-1.5 text-foreground/80 focus:outline-none focus:border-primary/40"
              />

              <p class="text-[9px] text-muted-foreground/60 uppercase tracking-wider font-semibold pt-1">Body</p>
              <textarea
                v-model="editBody"
                rows="4"
                class="w-full text-[10px] bg-foreground/[0.03] border border-border/40 rounded-md px-2 py-1.5 text-foreground/80 focus:outline-none focus:border-primary/40 resize-none leading-relaxed"
              />
            </template>

            <!-- Request tab -->
            <template v-else-if="activeTab === 'request'">
              <p class="text-[9px] text-muted-foreground/60 uppercase tracking-wider font-semibold">Headers</p>
              <div class="rounded-md bg-foreground/[0.02] border border-border/30 overflow-hidden">
                <div
                  v-for="(value, key) in selectedExchange.reqHeaders"
                  :key="key"
                  class="flex gap-2 px-2 py-[3px] border-b border-border/20 last:border-0"
                >
                  <span class="text-[10px] text-primary/70 shrink-0">{{ key }}</span>
                  <span class="text-[10px] text-foreground/60 truncate">{{ value }}</span>
                </div>
              </div>

              <template v-if="selectedExchange.reqBody">
                <p class="text-[9px] text-muted-foreground/60 uppercase tracking-wider font-semibold pt-1">Body</p>
                <pre class="text-[10px] text-foreground/60 bg-foreground/[0.02] border border-border/30 rounded-md p-2 whitespace-pre-wrap overflow-x-auto leading-relaxed">{{ selectedExchange.reqBody }}</pre>
              </template>
            </template>

            <!-- Response tab -->
            <template v-else>
              <p class="text-[9px] text-muted-foreground/60 uppercase tracking-wider font-semibold">Headers</p>
              <div class="rounded-md bg-foreground/[0.02] border border-border/30 overflow-hidden">
                <div
                  v-for="(value, key) in selectedExchange.resHeaders"
                  :key="key"
                  class="flex gap-2 px-2 py-[3px] border-b border-border/20 last:border-0"
                >
                  <span class="text-[10px] text-primary/70 shrink-0">{{ key }}</span>
                  <span class="text-[10px] text-foreground/60 truncate">{{ value }}</span>
                </div>
              </div>

              <p class="text-[9px] text-muted-foreground/60 uppercase tracking-wider font-semibold pt-1">Body</p>
              <pre class="text-[10px] text-foreground/60 bg-foreground/[0.02] border border-border/30 rounded-md p-2 whitespace-pre-wrap overflow-x-auto leading-relaxed">{{ selectedExchange.resBody }}</pre>
            </template>
          </div>

          <!-- Bottom: Replay buttons + result -->
          <div class="border-t border-border/40 px-2.5 py-2 bg-code-header">
            <div v-if="showReplayResult && replayResult" class="mb-2 px-2 py-1.5 rounded-md bg-foreground/[0.02] border border-border/30">
              <div class="flex items-center gap-2 mb-1">
                <span class="text-[9px] text-purple-400 font-semibold uppercase tracking-wider">Replay Result</span>
                <span class="text-[10px] font-medium px-1.5 py-0.5 rounded border" :class="statusBadgeClass(replayResult.status)">{{ replayResult.status }}</span>
              </div>
              <pre class="text-[10px] text-foreground/50 whitespace-pre-wrap leading-relaxed">{{ replayResult.body }}</pre>
            </div>
            <div class="flex gap-2">
              <!-- Edit mode: Cancel + Send -->
              <template v-if="editMode">
                <button
                  @click="editMode = false"
                  class="flex-1 py-1.5 rounded-md text-[10px] font-medium bg-foreground/[0.04] text-foreground/60 border border-border/40 hover:bg-foreground/[0.08] transition-colors"
                >
                  {{ t('landing.advanced.inspector.demo.cancel') }}
                </button>
                <button
                  @click="sendEditReplay"
                  :disabled="replayingId !== null"
                  class="flex-1 py-1.5 rounded-md text-[10px] font-medium bg-primary/15 text-primary border border-primary/20 hover:bg-primary/25 disabled:opacity-50 transition-colors flex items-center justify-center gap-1.5"
                >
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
                  </svg>
                  {{ t('landing.advanced.inspector.demo.send') }}
                </button>
              </template>
              <!-- Normal mode: Replay + Edit & Replay -->
              <template v-else>
                <button
                  @click="replay(selectedExchange.id)"
                  :disabled="replayingId !== null"
                  class="flex-1 py-1.5 rounded-md text-[10px] font-medium bg-primary/15 text-primary border border-primary/20 hover:bg-primary/25 disabled:opacity-50 transition-colors flex items-center justify-center gap-1.5"
                >
                  <svg
                    v-if="replayingId === selectedExchange.id"
                    class="animate-spin h-3 w-3"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  <svg v-else class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
                  </svg>
                  {{ t('landing.advanced.inspector.demo.replay') }}
                </button>
                <button
                  @click="openEditor"
                  class="flex-1 py-1.5 rounded-md text-[10px] font-medium bg-foreground/[0.04] text-foreground/60 border border-border/40 hover:bg-foreground/[0.08] transition-colors flex items-center justify-center gap-1.5"
                >
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                  {{ t('landing.advanced.inspector.demo.editReplay') }}
                </button>
              </template>
            </div>
          </div>
        </template>

        <!-- Empty state -->
        <div v-else class="flex-1 flex items-center justify-center text-muted-foreground/40 text-[11px]">
          {{ t('landing.advanced.inspector.demo.selectRequest') }}
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.req-row {
  position: relative;
  z-index: 1;
}

.req-enter-active {
  transition: all 0.35s ease-out;
}
.req-leave-active {
  transition: all 0.2s ease-in;
  pointer-events: none !important;
  z-index: 0 !important;
  position: absolute !important;
  left: 0;
  right: 0;
}
.req-enter-from {
  opacity: 0;
  transform: translateY(-12px);
}
.req-leave-to {
  opacity: 0;
}
.req-move {
  transition: transform 0.3s ease;
}
</style>
