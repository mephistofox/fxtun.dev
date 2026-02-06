<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { adminApi, type AdminTunnel } from '@/api/client'

const { t } = useI18n()

const tunnels = ref<AdminTunnel[]>([])
const loading = ref(true)
const error = ref('')
const search = ref('')
const typeFilter = ref<'all' | 'http' | 'tcp' | 'udp'>('all')
const confirmingId = ref<string | null>(null)
let confirmTimer: ReturnType<typeof setTimeout> | null = null

const typeFilters = ['all', 'http', 'tcp', 'udp'] as const

const filteredTunnels = computed(() => {
  let result = tunnels.value
  if (typeFilter.value !== 'all') {
    result = result.filter((t) => t.type === typeFilter.value)
  }
  const q = search.value.toLowerCase().trim()
  if (q) {
    result = result.filter((t) =>
      (t.url && t.url.toLowerCase().includes(q)) ||
      (t.subdomain && t.subdomain.toLowerCase().includes(q)) ||
      (t.user_phone && t.user_phone.toLowerCase().includes(q)) ||
      (t.name && t.name.toLowerCase().includes(q))
    )
  }
  return result
})

const stats = computed(() => {
  const all = tunnels.value
  return {
    total: all.length,
    http: all.filter((t) => t.type === 'http').length,
    tcp: all.filter((t) => t.type === 'tcp').length,
    udp: all.filter((t) => t.type === 'udp').length,
  }
})

async function loadTunnels() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.listTunnels()
    tunnels.value = response.data.tunnels || []
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.failedToLoad')
  } finally {
    loading.value = false
  }
}

function getTunnelUrl(tunnel: AdminTunnel): string {
  if (tunnel.url) return tunnel.url
  if (tunnel.type === 'http' && tunnel.subdomain) {
    return `https://${tunnel.subdomain}.fxtun.dev`
  }
  if (tunnel.remote_port) {
    return `${tunnel.type}://fxtun.dev:${tunnel.remote_port}`
  }
  return '-'
}

function handleClose(tunnel: AdminTunnel) {
  if (confirmingId.value === tunnel.id) {
    doClose(tunnel)
    return
  }
  confirmingId.value = tunnel.id
  if (confirmTimer) clearTimeout(confirmTimer)
  confirmTimer = setTimeout(() => {
    confirmingId.value = null
  }, 3000)
}

async function doClose(tunnel: AdminTunnel) {
  confirmingId.value = null
  if (confirmTimer) clearTimeout(confirmTimer)
  try {
    await adminApi.closeTunnel(tunnel.id)
    tunnels.value = tunnels.value.filter((t) => t.id !== tunnel.id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.tunnels.failedToClose')
  }
}

function typeBadgeClass(type: string): string {
  switch (type) {
    case 'http': return 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
    case 'tcp': return 'bg-blue-500/15 text-blue-600 dark:text-blue-400'
    case 'udp': return 'bg-purple-500/15 text-purple-600 dark:text-purple-400'
    default: return 'bg-muted text-muted-foreground'
  }
}

function typePillClass(type: string, active: boolean): string {
  if (!active) return 'border border-border text-muted-foreground hover:text-foreground hover:border-foreground/30'
  switch (type) {
    case 'all': return 'bg-foreground text-background'
    case 'http': return 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
    case 'tcp': return 'bg-blue-500/15 text-blue-600 dark:text-blue-400 border border-blue-500/30'
    case 'udp': return 'bg-purple-500/15 text-purple-600 dark:text-purple-400 border border-purple-500/30'
    default: return ''
  }
}

onMounted(loadTunnels)
</script>

<template>
  <Layout>
    <div class="space-y-4">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-foreground">{{ t('admin.tunnels.title') }}</h1>
          <p class="text-sm text-muted-foreground">{{ t('admin.tunnels.subtitle') }}</p>
        </div>
        <Button @click="loadTunnels" :loading="loading" variant="outline" size="sm">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10" />
            <polyline points="1 20 1 14 7 14" />
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
          </svg>
          {{ t('common.refresh') }}
        </Button>
      </div>

      <!-- Search + Filter pills -->
      <div class="flex flex-col sm:flex-row gap-3">
        <div class="relative flex-1">
          <svg xmlns="http://www.w3.org/2000/svg" class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <Input
            v-model="search"
            :placeholder="t('admin.searchTunnels')"
            class="pl-9"
          />
        </div>
        <div class="flex items-center gap-1.5">
          <button
            v-for="tf in typeFilters"
            :key="tf"
            @click="typeFilter = tf"
            :class="[
              'px-3 py-1.5 rounded-full text-xs font-medium uppercase tracking-wide transition-all cursor-pointer',
              typePillClass(tf, typeFilter === tf),
            ]"
          >
            {{ tf === 'all' ? t('admin.filterAll') : tf }}
          </button>
        </div>
      </div>

      <!-- Stats bar -->
      <div v-if="!loading && tunnels.length > 0" class="flex items-center gap-3 text-xs text-muted-foreground font-mono">
        <span>{{ t('admin.tunnels.total', { count: stats.total }) }}</span>
        <span class="text-border">|</span>
        <span class="text-emerald-600 dark:text-emerald-400">{{ stats.http }} HTTP</span>
        <span class="text-border">&middot;</span>
        <span class="text-blue-600 dark:text-blue-400">{{ stats.tcp }} TCP</span>
        <span class="text-border">&middot;</span>
        <span class="text-purple-600 dark:text-purple-400">{{ stats.udp }} UDP</span>
      </div>

      <!-- Error -->
      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-12 text-muted-foreground text-sm">
        {{ t('common.loading') }}
      </div>

      <!-- Empty state (no tunnels at all) -->
      <div v-else-if="tunnels.length === 0" class="text-center py-16">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto text-muted-foreground/40 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="16" y="16" width="6" height="6" rx="1" />
          <rect x="2" y="16" width="6" height="6" rx="1" />
          <rect x="9" y="2" width="6" height="6" rx="1" />
          <path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3" />
          <path d="M12 12V8" />
        </svg>
        <p class="text-muted-foreground text-sm">{{ t('admin.tunnels.noTunnels') }}</p>
      </div>

      <!-- Table -->
      <template v-else>
        <!-- No results after filter -->
        <div v-if="filteredTunnels.length === 0" class="text-center py-12">
          <p class="text-muted-foreground text-sm">{{ t('admin.noResults') }}</p>
        </div>

        <Card v-else class="overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b bg-muted/40">
                  <th class="text-left px-3 py-2 font-medium text-muted-foreground text-xs uppercase tracking-wider">{{ t('admin.tunnels.type') }}</th>
                  <th class="text-left px-3 py-2 font-medium text-muted-foreground text-xs uppercase tracking-wider">{{ t('admin.tunnels.url') }}</th>
                  <th class="text-left px-3 py-2 font-medium text-muted-foreground text-xs uppercase tracking-wider">{{ t('admin.tunnels.localPort') }}</th>
                  <th class="text-left px-3 py-2 font-medium text-muted-foreground text-xs uppercase tracking-wider">{{ t('admin.tunnels.owner') }}</th>
                  <th class="text-right px-3 py-2 font-medium text-muted-foreground text-xs uppercase tracking-wider">{{ t('admin.users.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="tunnel in filteredTunnels"
                  :key="tunnel.id"
                  class="border-b border-border/50 hover:bg-muted/20 transition-colors"
                >
                  <td class="px-3 py-2">
                    <span
                      :class="[
                        'inline-block px-2 py-0.5 text-[10px] font-bold rounded uppercase tracking-wider',
                        typeBadgeClass(tunnel.type),
                      ]"
                    >
                      {{ tunnel.type }}
                    </span>
                  </td>
                  <td class="px-3 py-2 max-w-xs truncate">
                    <a
                      v-if="tunnel.type === 'http'"
                      :href="getTunnelUrl(tunnel)"
                      target="_blank"
                      rel="noopener"
                      class="text-primary hover:underline font-mono text-xs"
                    >
                      {{ getTunnelUrl(tunnel) }}
                    </a>
                    <span v-else class="font-mono text-xs text-foreground">{{ getTunnelUrl(tunnel) }}</span>
                  </td>
                  <td class="px-3 py-2 font-mono text-xs text-foreground">{{ tunnel.local_port }}</td>
                  <td class="px-3 py-2 font-mono text-xs text-muted-foreground">{{ tunnel.user_phone || '-' }}</td>
                  <td class="px-3 py-2">
                    <div class="flex justify-end gap-1">
                      <router-link
                        v-if="tunnel.type === 'http'"
                        :to="`/inspect/${tunnel.id}`"
                        class="inline-flex items-center justify-center h-7 w-7 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition"
                        title="Inspect"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <circle cx="11" cy="11" r="8" />
                          <line x1="21" y1="21" x2="16.65" y2="16.65" />
                        </svg>
                      </router-link>
                      <Button
                        :variant="confirmingId === tunnel.id ? 'destructive' : 'ghost'"
                        size="icon"
                        @click="handleClose(tunnel)"
                        :title="confirmingId === tunnel.id ? t('admin.tunnels.confirmClose') : t('admin.tunnels.close')"
                        class="h-7 w-7"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <line x1="18" y1="6" x2="6" y2="18" />
                          <line x1="6" y1="6" x2="18" y2="18" />
                        </svg>
                      </Button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      </template>
    </div>
  </Layout>
</template>
