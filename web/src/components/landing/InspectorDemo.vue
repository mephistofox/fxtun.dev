<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface MockRequest {
  id: number
  method: string
  path: string
  status: number
  duration: string
  reqHeaders: string
  resBody: string
}

const mockPool: Omit<MockRequest, 'id'>[] = [
  { method: 'POST', path: '/api/webhooks/stripe', status: 200, duration: '23ms',
    reqHeaders: 'Content-Type: application/json\nStripe-Signature: whsec_...', resBody: '{ "received": true }' },
  { method: 'GET', path: '/api/users/me', status: 200, duration: '12ms',
    reqHeaders: 'Authorization: Bearer sk_...', resBody: '{ "id": 1, "name": "Alex" }' },
  { method: 'PUT', path: '/api/settings', status: 401, duration: '8ms',
    reqHeaders: 'Content-Type: application/json', resBody: '{ "error": "Unauthorized" }' },
  { method: 'POST', path: '/api/orders', status: 201, duration: '45ms',
    reqHeaders: 'Content-Type: application/json\nAuthorization: Bearer sk_...', resBody: '{ "id": 847, "status": "created" }' },
  { method: 'DELETE', path: '/api/sessions/old', status: 500, duration: '120ms',
    reqHeaders: 'Authorization: Bearer sk_...', resBody: '{ "error": "Internal Server Error" }' },
  { method: 'GET', path: '/health', status: 200, duration: '2ms',
    reqHeaders: 'Accept: application/json', resBody: '{ "status": "ok" }' },
  { method: 'POST', path: '/api/webhooks/github', status: 200, duration: '34ms',
    reqHeaders: 'Content-Type: application/json\nX-Hub-Signature-256: sha256=...', resBody: '{ "ok": true }' },
  { method: 'PATCH', path: '/api/users/42', status: 404, duration: '15ms',
    reqHeaders: 'Content-Type: application/json', resBody: '{ "error": "Not Found" }' },
]

let nextId = 0
const requests = ref<MockRequest[]>([])
const selectedId = ref<number | null>(null)
const replayingId = ref<number | null>(null)
let intervalHandle: ReturnType<typeof setInterval> | null = null

const selectedRequest = computed(() =>
  requests.value.find(r => r.id === selectedId.value) ?? null
)

function pickRandom(): MockRequest {
  const item = mockPool[Math.floor(Math.random() * mockPool.length)]
  return { ...item, id: nextId++ }
}

function addRequest() {
  requests.value.unshift(pickRandom())
  if (requests.value.length > 6) {
    requests.value.pop()
  }
}

function selectRow(id: number) {
  selectedId.value = id
}

function replay(id: number) {
  replayingId.value = id
  setTimeout(() => {
    replayingId.value = null
  }, 800)
}

function methodColor(method: string): string {
  switch (method) {
    case 'GET': return 'text-sky-400'
    case 'POST': return 'text-emerald-400'
    case 'PUT': return 'text-amber-400'
    case 'PATCH': return 'text-orange-400'
    case 'DELETE': return 'text-red-400'
    default: return 'text-muted-foreground'
  }
}

function statusColor(status: number): string {
  if (status >= 200 && status < 300) return 'text-emerald-400'
  if (status >= 400 && status < 500) return 'text-yellow-400'
  if (status >= 500) return 'text-red-400'
  return 'text-muted-foreground'
}

onMounted(() => {
  for (let i = 0; i < 3; i++) {
    requests.value.push(pickRandom())
  }
  selectedId.value = requests.value[0]?.id ?? null
  intervalHandle = setInterval(addRequest, 2500)
})

onUnmounted(() => {
  if (intervalHandle) {
    clearInterval(intervalHandle)
  }
})
</script>

<template>
  <div class="rounded-lg border border-border bg-background overflow-hidden font-mono text-xs select-none h-[256px] flex flex-col">
    <!-- Header -->
    <div class="flex items-center gap-2 px-3 py-2 border-b border-border bg-surface text-sm">
      <span class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
        <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
      </span>
      <span class="text-muted-foreground">Live</span>
      <span class="text-foreground/60">&mdash;</span>
      <span class="text-foreground">my-app.tunnel.dev</span>
    </div>

    <div class="flex">
      <!-- Request list -->
      <div class="border-r border-border" :class="selectedRequest ? 'w-full lg:w-1/2' : 'w-full'">
        <div class="flex-1 overflow-hidden relative">
          <TransitionGroup name="req" tag="div" class="relative">
            <div
              v-for="req in requests"
              :key="req.id"
              class="req-row flex items-center gap-2 px-3 py-1.5 border-b border-border/50 transition-colors"
              :class="selectedId === req.id ? 'bg-primary/10' : 'hover:bg-surface'"
              @mousedown.prevent
              @click="selectRow(req.id)"
            >
              <span class="w-11 font-semibold shrink-0" :class="methodColor(req.method)">{{ req.method }}</span>
              <span class="text-foreground/80 truncate flex-1">{{ req.path }}</span>
              <span class="w-8 text-right shrink-0" :class="statusColor(req.status)">{{ req.status }}</span>
              <span class="w-11 text-right text-muted-foreground shrink-0">{{ req.duration }}</span>
              <button
                class="shrink-0 px-2 py-1 rounded text-[10px] border border-border text-muted-foreground hover:text-foreground hover:border-foreground/30 transition-colors"
                @click.stop="replay(req.id)"
              >
                <svg
                  v-if="replayingId === req.id"
                  class="animate-spin h-3 w-3 inline-block"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <span v-else>{{ t('landing.advanced.inspector.demo.replay') }}</span>
              </button>
            </div>
          </TransitionGroup>
        </div>
      </div>

      <!-- Detail panel -->
      <div v-if="selectedRequest" class="hidden lg:block lg:w-1/2 overflow-auto p-3 space-y-3">
        <div class="flex items-center gap-2">
          <span class="font-semibold" :class="methodColor(selectedRequest.method)">{{ selectedRequest.method }}</span>
          <span class="text-foreground/80 truncate">{{ selectedRequest.path }}</span>
          <span class="ml-auto font-semibold" :class="statusColor(selectedRequest.status)">{{ selectedRequest.status }}</span>
        </div>
        <div>
          <p class="text-[10px] text-muted-foreground uppercase tracking-wider mb-1">Request Headers</p>
          <pre class="text-[11px] text-foreground/70 bg-surface/80 rounded p-2 whitespace-pre-wrap">{{ selectedRequest.reqHeaders }}</pre>
        </div>
        <div>
          <p class="text-[10px] text-muted-foreground uppercase tracking-wider mb-1">Response Body</p>
          <pre class="text-[11px] text-foreground/70 bg-surface/80 rounded p-2 whitespace-pre-wrap">{{ selectedRequest.resBody }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.req-row {
  position: relative;
  z-index: 1;
  cursor: pointer;
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
