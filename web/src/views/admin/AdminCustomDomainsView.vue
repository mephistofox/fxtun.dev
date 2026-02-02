<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { adminApi, type CustomDomain } from '@/api/client'

const { t, locale } = useI18n()

type AdminCustomDomain = CustomDomain & { user_phone: string; tls_expiry?: string }
type StatusFilter = 'all' | 'verified' | 'pending'

const domains = ref<AdminCustomDomain[]>([])
const loading = ref(true)
const error = ref('')
const total = ref(0)
const page = ref(1)
const limit = 20
const search = ref('')
const statusFilter = ref<StatusFilter>('all')
const confirmingDeleteId = ref<number | null>(null)
const deletingId = ref<number | null>(null)

const filteredDomains = computed(() => {
  let result = domains.value

  if (statusFilter.value === 'verified') {
    result = result.filter((d) => d.verified)
  } else if (statusFilter.value === 'pending') {
    result = result.filter((d) => !d.verified)
  }

  if (search.value.trim()) {
    const q = search.value.trim().toLowerCase()
    result = result.filter(
      (d) =>
        d.domain.toLowerCase().includes(q) ||
        d.target_subdomain.toLowerCase().includes(q) ||
        d.user_phone.toLowerCase().includes(q)
    )
  }

  return result
})

async function loadDomains() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.listCustomDomains(page.value, limit)
    domains.value = response.data.domains || []
    total.value = response.data.total
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.customDomains.failedToLoad')
  } finally {
    loading.value = false
  }
}

function requestDelete(id: number) {
  confirmingDeleteId.value = id
}

function cancelDelete() {
  confirmingDeleteId.value = null
}

async function confirmDelete(id: number) {
  deletingId.value = id
  try {
    await adminApi.deleteCustomDomain(id)
    domains.value = domains.value.filter((d) => d.id !== id)
    total.value--
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.customDomains.failedToDelete')
  } finally {
    deletingId.value = null
    confirmingDeleteId.value = null
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function tlsDaysLeft(dateStr?: string): number | null {
  if (!dateStr) return null
  const d = new Date(dateStr)
  const now = new Date()
  return Math.ceil((d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

function tlsExpiryClass(dateStr?: string): string {
  const days = tlsDaysLeft(dateStr)
  if (days === null) return 'text-muted-foreground'
  if (days <= 0) return 'text-red-500'
  if (days <= 30) return 'text-yellow-500'
  return 'text-muted-foreground'
}

const totalPages = computed(() => Math.ceil(total.value / limit))

function goToPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  loadDomains()
}

watch(statusFilter, () => {
  page.value = 1
})

onMounted(loadDomains)
</script>

<template>
  <Layout>
    <div class="space-y-5">
      <!-- Header -->
      <div>
        <h1 class="text-2xl font-bold text-foreground">{{ t('admin.customDomains.title') }}</h1>
        <p class="text-sm text-muted-foreground mt-1">
          {{ t('admin.customDomains.subtitle', { total }) }}
        </p>
      </div>

      <!-- Error -->
      <div v-if="error" class="bg-destructive/10 text-destructive px-4 py-3 rounded-lg text-sm border border-destructive/20">
        {{ error }}
      </div>

      <!-- Toolbar: Search + Status Filter -->
      <div class="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
        <div class="relative flex-1 w-full sm:max-w-sm">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <Input
            v-model="search"
            :placeholder="t('admin.searchDomains')"
            class="pl-9"
          />
        </div>

        <div class="flex gap-1 rounded-lg border bg-muted/30 p-1">
          <button
            v-for="filter in (['all', 'verified', 'pending'] as StatusFilter[])"
            :key="filter"
            @click="statusFilter = filter"
            :class="[
              'px-3 py-1.5 text-xs font-medium rounded-md transition-all',
              statusFilter === filter
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            ]"
          >
            {{
              filter === 'all'
                ? t('admin.filterAll')
                : filter === 'verified'
                  ? t('admin.customDomains.verified')
                  : t('admin.customDomains.pending')
            }}
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-16 text-muted-foreground">
        <svg class="h-6 w-6 animate-spin mx-auto mb-3 text-primary" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        {{ t('common.loading') }}
      </div>

      <!-- Empty State -->
      <Card v-else-if="domains.length === 0 && !search && statusFilter === 'all'" class="py-16 text-center">
        <div class="flex flex-col items-center gap-3">
          <div class="h-12 w-12 rounded-full bg-muted flex items-center justify-center">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10" />
              <line x1="2" y1="12" x2="22" y2="12" />
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
            </svg>
          </div>
          <p class="text-muted-foreground text-sm">{{ t('admin.customDomains.noDomains') }}</p>
        </div>
      </Card>

      <!-- No search results -->
      <Card v-else-if="filteredDomains.length === 0" class="py-12 text-center">
        <p class="text-muted-foreground text-sm">{{ t('admin.noResults') }}</p>
      </Card>

      <!-- Table -->
      <div v-else>
        <div class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-xs uppercase tracking-wider text-muted-foreground">
              <tr>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.customDomains.domain') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.customDomains.target') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.customDomains.user') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.customDomains.status') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.customDomains.tlsExpiry') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.customDomains.created') }}</th>
                <th class="px-4 py-3 text-right font-medium">{{ t('admin.users.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <tr
                v-for="domain in filteredDomains"
                :key="domain.id"
                class="hover:bg-muted/30 transition-colors"
              >
                <!-- Domain -->
                <td class="px-4 py-3 font-semibold text-foreground">
                  {{ domain.domain }}
                </td>

                <!-- Target -->
                <td class="px-4 py-3 text-muted-foreground">
                  <span class="inline-flex items-center gap-1.5">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="5" y1="12" x2="19" y2="12" />
                      <polyline points="12 5 19 12 12 19" />
                    </svg>
                    {{ domain.target_subdomain }}
                  </span>
                </td>

                <!-- User -->
                <td class="px-4 py-3">
                  <span class="font-mono text-xs text-muted-foreground">{{ domain.user_phone }}</span>
                </td>

                <!-- Status -->
                <td class="px-4 py-3">
                  <span
                    v-if="domain.verified"
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-500/10 text-green-600 dark:text-green-400"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                      <polyline points="20 6 9 17 4 12" />
                    </svg>
                    {{ t('admin.customDomains.verified') }}
                  </span>
                  <span
                    v-else
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-500/10 text-yellow-600 dark:text-yellow-400"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="12" y1="8" x2="12" y2="12" />
                      <line x1="12" y1="16" x2="12.01" y2="16" />
                    </svg>
                    {{ t('admin.customDomains.pending') }}
                  </span>
                </td>

                <!-- TLS Expiry -->
                <td class="px-4 py-3">
                  <template v-if="domain.tls_expiry">
                    <span :class="tlsExpiryClass(domain.tls_expiry)" class="text-xs">
                      {{ formatDate(domain.tls_expiry) }}
                      <template v-if="tlsDaysLeft(domain.tls_expiry) !== null && tlsDaysLeft(domain.tls_expiry)! <= 30">
                        <span
                          :class="[
                            'ml-1 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold',
                            tlsDaysLeft(domain.tls_expiry)! <= 0
                              ? 'bg-red-500/15 text-red-500'
                              : 'bg-yellow-500/15 text-yellow-600 dark:text-yellow-400',
                          ]"
                        >
                          {{ tlsDaysLeft(domain.tls_expiry) }}d
                        </span>
                      </template>
                    </span>
                  </template>
                  <span v-else class="text-muted-foreground text-xs">&mdash;</span>
                </td>

                <!-- Created -->
                <td class="px-4 py-3 text-xs text-muted-foreground">
                  {{ formatDate(domain.created_at) }}
                </td>

                <!-- Actions -->
                <td class="px-4 py-3 text-right">
                  <template v-if="confirmingDeleteId === domain.id">
                    <span class="inline-flex items-center gap-1.5">
                      <Button
                        variant="destructive"
                        size="xs"
                        :loading="deletingId === domain.id"
                        @click="confirmDelete(domain.id)"
                      >
                        {{ t('common.confirm') }}
                      </Button>
                      <Button
                        variant="ghost"
                        size="xs"
                        @click="cancelDelete"
                      >
                        {{ t('common.cancel') }}
                      </Button>
                    </span>
                  </template>
                  <template v-else>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8 text-muted-foreground hover:text-destructive"
                      @click="requestDelete(domain.id)"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                      </svg>
                    </Button>
                  </template>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div v-if="totalPages > 1" class="flex items-center justify-between pt-4">
          <p class="text-xs text-muted-foreground">
            {{ t('admin.pagination.page', { page, totalPages }) }}
          </p>
          <div class="flex gap-1">
            <Button variant="outline" size="xs" :disabled="page <= 1" @click="goToPage(page - 1)">
              {{ t('admin.pagination.prev') }}
            </Button>
            <Button variant="outline" size="xs" :disabled="page >= totalPages" @click="goToPage(page + 1)">
              {{ t('admin.pagination.next') }}
            </Button>
          </div>
        </div>
      </div>
    </div>
  </Layout>
</template>
