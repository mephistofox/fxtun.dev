<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { adminApi, type CustomDomain } from '@/api/client'

const { t, locale } = useI18n()

type AdminCustomDomain = CustomDomain & { user_phone: string; tls_expiry?: string }

const domains = ref<AdminCustomDomain[]>([])
const loading = ref(true)
const error = ref('')
const total = ref(0)
const page = ref(1)
const limit = 20

async function loadDomains() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.listCustomDomains(page.value, limit)
    domains.value = response.data.domains || []
    total.value = response.data.total
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to load custom domains'
  } finally {
    loading.value = false
  }
}

async function deleteDomain(id: number, domain: string) {
  if (!confirm(`Delete custom domain "${domain}"?`)) return
  try {
    await adminApi.deleteCustomDomain(id)
    domains.value = domains.value.filter((d) => d.id !== id)
    total.value--
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to delete'
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function formatExpiry(dateStr?: string): string {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  const now = new Date()
  const daysLeft = Math.ceil((d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  const formatted = formatDate(dateStr)
  if (daysLeft < 30) return `${formatted} (${daysLeft}d)`
  return formatted
}

const totalPages = () => Math.ceil(total.value / limit)

function nextPage() {
  if (page.value < totalPages()) {
    page.value++
    loadDomains()
  }
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    loadDomains()
  }
}

onMounted(loadDomains)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-bold">Custom Domains</h1>
        <p class="text-muted-foreground">
          Manage all custom domains ({{ total }} total)
        </p>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <div v-if="loading" class="text-center py-12 text-muted-foreground">
        <div class="animate-pulse">{{ t('common.loading') }}</div>
      </div>

      <Card v-else-if="domains.length === 0" class="p-8 text-center text-muted-foreground">
        No custom domains registered
      </Card>

      <div v-else>
        <div class="overflow-x-auto rounded-lg border">
          <table class="w-full text-sm">
            <thead class="bg-muted/50">
              <tr>
                <th class="px-4 py-3 text-left font-medium">Domain</th>
                <th class="px-4 py-3 text-left font-medium">Target</th>
                <th class="px-4 py-3 text-left font-medium">User</th>
                <th class="px-4 py-3 text-left font-medium">Status</th>
                <th class="px-4 py-3 text-left font-medium">TLS Expiry</th>
                <th class="px-4 py-3 text-left font-medium">Created</th>
                <th class="px-4 py-3 text-right font-medium">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr v-for="domain in domains" :key="domain.id" class="hover:bg-muted/30 transition-colors">
                <td class="px-4 py-3 font-medium">{{ domain.domain }}</td>
                <td class="px-4 py-3 text-muted-foreground">{{ domain.target_subdomain }}</td>
                <td class="px-4 py-3 text-muted-foreground">{{ domain.user_phone }}</td>
                <td class="px-4 py-3">
                  <span
                    v-if="domain.verified"
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-500/10 text-green-600 dark:text-green-400"
                  >
                    Verified
                  </span>
                  <span
                    v-else
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-500/10 text-yellow-600 dark:text-yellow-400"
                  >
                    Pending
                  </span>
                </td>
                <td class="px-4 py-3 text-muted-foreground">{{ formatExpiry(domain.tls_expiry) }}</td>
                <td class="px-4 py-3 text-muted-foreground">{{ formatDate(domain.created_at) }}</td>
                <td class="px-4 py-3 text-right">
                  <Button variant="ghost" size="sm" @click="deleteDomain(domain.id, domain.domain)" class="text-destructive hover:text-destructive">
                    Delete
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div v-if="totalPages() > 1" class="flex items-center justify-between pt-4">
          <p class="text-sm text-muted-foreground">
            Page {{ page }} of {{ totalPages() }}
          </p>
          <div class="flex gap-2">
            <Button variant="outline" size="sm" @click="prevPage" :disabled="page <= 1">Prev</Button>
            <Button variant="outline" size="sm" @click="nextPage" :disabled="page >= totalPages()">Next</Button>
          </div>
        </div>
      </div>
    </div>
  </Layout>
</template>
