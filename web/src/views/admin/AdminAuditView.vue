<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { adminApi, type AuditLog } from '@/api/client'

const { t, locale } = useI18n()

const logs = ref<AuditLog[]>([])
const loading = ref(true)
const error = ref('')
const total = ref(0)
const page = ref(1)
const limit = 30

const search = ref('')
const activeFilter = ref('all')
const expandedId = ref<number | null>(null)

interface FilterCategory {
  key: string
  label: string
  match: (action: string) => boolean
}

const filterCategories: FilterCategory[] = [
  { key: 'all', label: 'admin.filterAll', match: () => true },
  { key: 'auth', label: 'Auth', match: (a) => /^(login|register|logout|auth)/.test(a) },
  { key: 'tokens', label: 'Tokens', match: (a) => /token/.test(a) },
  { key: 'domains', label: 'Domains', match: (a) => /domain/.test(a) },
  { key: 'users', label: 'Users', match: (a) => /user/.test(a) },
  { key: 'other', label: 'Other', match: () => true },
]

function categoryFor(action: string): string {
  for (const cat of filterCategories) {
    if (cat.key === 'all' || cat.key === 'other') continue
    if (cat.match(action)) return cat.key
  }
  return 'other'
}

const filteredLogs = computed(() => {
  let result = logs.value

  if (activeFilter.value !== 'all') {
    if (activeFilter.value === 'other') {
      const specificKeys = filterCategories
        .filter((c) => c.key !== 'all' && c.key !== 'other')
        .map((c) => c.match)
      result = result.filter((log) => !specificKeys.some((fn) => fn(log.action)))
    } else {
      const cat = filterCategories.find((c) => c.key === activeFilter.value)
      if (cat) result = result.filter((log) => cat.match(log.action))
    }
  }

  if (search.value.trim()) {
    const q = search.value.toLowerCase()
    result = result.filter(
      (log) =>
        log.action.toLowerCase().includes(q) ||
        (log.user_phone && log.user_phone.toLowerCase().includes(q)) ||
        log.ip_address.toLowerCase().includes(q)
    )
  }

  return result
})

const filterCounts = computed(() => {
  const counts: Record<string, number> = {}
  for (const cat of filterCategories) {
    counts[cat.key] = 0
  }
  for (const log of logs.value) {
    counts.all++
    const c = categoryFor(log.action)
    counts[c] = (counts[c] || 0) + 1
  }
  return counts
})

const totalPages = computed(() => Math.ceil(total.value / limit))

function relativeTime(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diff = now - then
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  const months = Math.floor(days / 30)
  return `${months}mo ago`
}

function fullDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function getActionColor(action: string): string {
  if (/login|register/.test(action)) return 'bg-green-500/15 text-green-400 border border-green-500/20'
  if (/delete|disable|remove/.test(action)) return 'bg-red-500/15 text-red-400 border border-red-500/20'
  if (/create|enable|add/.test(action)) return 'bg-blue-500/15 text-blue-400 border border-blue-500/20'
  if (/update|change|edit|modify/.test(action)) return 'bg-yellow-500/15 text-yellow-400 border border-yellow-500/20'
  if (/logout/.test(action)) return 'bg-orange-500/15 text-orange-400 border border-orange-500/20'
  return 'bg-muted text-muted-foreground border border-border'
}

function toggleExpand(id: number) {
  expandedId.value = expandedId.value === id ? null : id
}

async function loadLogs() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.listAuditLogs(page.value, limit)
    logs.value = response.data.logs || []
    total.value = response.data.total
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.failedToLoad')
  } finally {
    loading.value = false
  }
}

function goToPage(p: number) {
  page.value = p
  expandedId.value = null
  loadLogs()
}

watch(activeFilter, () => {
  expandedId.value = null
})

onMounted(loadLogs)
</script>

<template>
  <Layout>
    <div class="space-y-5">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold tracking-tight">{{ t('admin.audit.title') }}</h1>
          <p class="text-sm text-muted-foreground mt-1">{{ t('admin.audit.subtitle') }}</p>
        </div>
        <Button variant="outline" size="sm" :loading="loading" @click="loadLogs">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" />
            <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15" />
          </svg>
          {{ t('common.refresh') }}
        </Button>
      </div>

      <!-- Error -->
      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-lg text-sm font-mono">
        {{ error }}
      </div>

      <!-- Search + Filters -->
      <div class="space-y-3">
        <div class="relative">
          <svg xmlns="http://www.w3.org/2000/svg" class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <Input
            v-model="search"
            :placeholder="t('admin.searchAudit')"
            class="pl-10 font-mono text-sm bg-muted/30"
          />
        </div>

        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="cat in filterCategories"
            :key="cat.key"
            @click="activeFilter = cat.key"
            :class="[
              'px-3 py-1 rounded-full text-xs font-medium transition-all duration-150',
              activeFilter === cat.key
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground',
            ]"
          >
            {{ cat.key === 'all' ? t(cat.label) : cat.label }}
            <span
              v-if="filterCounts[cat.key]"
              :class="[
                'ml-1.5 text-[10px]',
                activeFilter === cat.key ? 'opacity-75' : 'opacity-50',
              ]"
            >{{ filterCounts[cat.key] }}</span>
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="py-16 text-center">
        <svg class="h-6 w-6 animate-spin text-muted-foreground mx-auto mb-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        <p class="text-sm text-muted-foreground">{{ t('common.loading') }}</p>
      </div>

      <!-- Empty state -->
      <div v-else-if="filteredLogs.length === 0" class="py-16 text-center">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-muted-foreground/40 mx-auto mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
          <polyline points="14 2 14 8 20 8" />
          <line x1="16" y1="13" x2="8" y2="13" />
          <line x1="16" y1="17" x2="8" y2="17" />
          <polyline points="10 9 9 9 8 9" />
        </svg>
        <p class="text-muted-foreground text-sm">
          {{ search.trim() || activeFilter !== 'all' ? t('admin.noResults') : t('admin.audit.noLogs') }}
        </p>
      </div>

      <!-- Timeline log entries -->
      <div v-else class="space-y-1">
        <Card class="overflow-hidden divide-y divide-border">
          <div
            v-for="log in filteredLogs"
            :key="log.id"
            class="group"
          >
            <!-- Row -->
            <div
              class="flex items-center gap-3 px-4 py-2.5 cursor-pointer transition-colors hover:bg-muted/40"
              :class="{ 'bg-muted/30': expandedId === log.id }"
              @click="toggleExpand(log.id)"
            >
              <!-- Expand indicator -->
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-3.5 w-3.5 text-muted-foreground/50 shrink-0 transition-transform duration-200"
                :class="{ 'rotate-90': expandedId === log.id }"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="9 18 15 12 9 6" />
              </svg>

              <!-- Time -->
              <span
                class="text-xs text-muted-foreground font-mono w-16 shrink-0 tabular-nums"
                :title="fullDate(log.created_at)"
              >
                {{ relativeTime(log.created_at) }}
              </span>

              <!-- Action badge -->
              <span
                :class="[
                  'px-2 py-0.5 text-[11px] font-semibold rounded-md font-mono shrink-0',
                  getActionColor(log.action),
                ]"
              >
                {{ log.action }}
              </span>

              <!-- User -->
              <span class="text-xs font-mono text-foreground truncate min-w-0 flex-1">
                {{ log.user_phone || '—' }}
              </span>

              <!-- IP -->
              <span class="text-xs font-mono text-muted-foreground shrink-0 hidden sm:block">
                {{ log.ip_address }}
              </span>

              <!-- Details indicator -->
              <span
                v-if="log.details && Object.keys(log.details).length > 0"
                class="text-[10px] text-muted-foreground/40 shrink-0"
                :title="t('admin.audit.detailsData')"
              >
                { }
              </span>
            </div>

            <!-- Expanded details -->
            <Transition
              enter-active-class="transition-all duration-200 ease-out"
              enter-from-class="opacity-0 max-h-0"
              enter-to-class="opacity-100 max-h-96"
              leave-active-class="transition-all duration-150 ease-in"
              leave-from-class="opacity-100 max-h-96"
              leave-to-class="opacity-0 max-h-0"
            >
              <div
                v-if="expandedId === log.id"
                class="overflow-hidden border-t border-border/50"
              >
                <div class="px-4 py-3 bg-muted/20 space-y-2">
                  <!-- Full date -->
                  <div class="flex items-center gap-2 text-xs">
                    <span class="text-muted-foreground">{{ t('admin.audit.time') }}:</span>
                    <span class="font-mono text-foreground">{{ fullDate(log.created_at) }}</span>
                  </div>

                  <!-- User -->
                  <div class="flex items-center gap-2 text-xs">
                    <span class="text-muted-foreground">{{ t('admin.audit.user') }}:</span>
                    <span class="font-mono text-foreground">{{ log.user_phone || '—' }}</span>
                    <span v-if="log.user_id" class="text-muted-foreground/50 font-mono">(ID: {{ log.user_id }})</span>
                  </div>

                  <!-- IP -->
                  <div class="flex items-center gap-2 text-xs">
                    <span class="text-muted-foreground">{{ t('admin.audit.ip') }}:</span>
                    <span class="font-mono text-foreground">{{ log.ip_address }}</span>
                  </div>

                  <!-- JSON details -->
                  <div v-if="log.details && Object.keys(log.details).length > 0">
                    <span class="text-xs text-muted-foreground">{{ t('admin.audit.detailsData') }}:</span>
                    <pre class="mt-1.5 bg-background/80 border border-border/50 rounded-lg p-3 text-xs font-mono text-foreground overflow-x-auto leading-relaxed">{{ JSON.stringify(log.details, null, 2) }}</pre>
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </Card>

        <!-- Pagination -->
        <div class="flex items-center justify-between pt-2">
          <p class="text-xs text-muted-foreground font-mono tabular-nums">
            {{ t('admin.pagination.showing', { from: (page - 1) * limit + 1, to: Math.min(page * limit, total), total }) }}
          </p>
          <div class="flex items-center gap-1">
            <Button variant="ghost" size="xs" :disabled="page === 1" @click="goToPage(page - 1)">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="15 18 9 12 15 6" />
              </svg>
              {{ t('admin.pagination.prev') }}
            </Button>
            <span class="text-xs text-muted-foreground font-mono px-2 tabular-nums">
              {{ page }} / {{ totalPages }}
            </span>
            <Button variant="ghost" size="xs" :disabled="page * limit >= total" @click="goToPage(page + 1)">
              {{ t('admin.pagination.next') }}
              <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="9 18 15 12 9 6" />
              </svg>
            </Button>
          </div>
        </div>
      </div>
    </div>
  </Layout>
</template>
