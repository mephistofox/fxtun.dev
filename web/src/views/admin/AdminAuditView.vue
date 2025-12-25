<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { adminApi, type AuditLog } from '@/api/client'

const { t, locale } = useI18n()

const logs = ref<AuditLog[]>([])
const loading = ref(true)
const error = ref('')
const total = ref(0)
const page = ref(1)
const limit = 20
const selectedLog = ref<AuditLog | null>(null)

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

function formatDate(dateStr: string): string {
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
  if (action.includes('login') || action.includes('register')) return 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
  if (action.includes('delete') || action.includes('disable')) return 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
  if (action.includes('create') || action.includes('enable')) return 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
  if (action.includes('update') || action.includes('change')) return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'
  return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300'
}

function nextPage() {
  if (page.value * limit < total.value) {
    page.value++
    loadLogs()
  }
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    loadLogs()
  }
}

onMounted(loadLogs)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ t('admin.audit.title') }}</h1>
          <p class="text-muted-foreground">{{ t('admin.audit.subtitle') }}</p>
        </div>
        <Button @click="loadLogs" :loading="loading" variant="outline">{{ t('common.refresh') }}</Button>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <!-- Details Modal -->
      <div
        v-if="selectedLog"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
        @click.self="selectedLog = null"
      >
        <Card class="w-full max-w-lg p-6 max-h-[80vh] overflow-y-auto">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-xl font-bold">{{ t('admin.audit.details') }}</h2>
            <Button variant="ghost" size="icon" @click="selectedLog = null">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </Button>
          </div>
          <div class="space-y-3 text-sm">
            <div>
              <span class="font-medium">{{ t('admin.audit.action') }}:</span>
              <span :class="['ml-2 px-2 py-0.5 text-xs font-medium rounded-full', getActionColor(selectedLog.action)]">
                {{ selectedLog.action }}
              </span>
            </div>
            <div>
              <span class="font-medium">{{ t('admin.audit.user') }}:</span>
              <span class="ml-2">{{ selectedLog.user_phone || '-' }}</span>
            </div>
            <div>
              <span class="font-medium">{{ t('admin.audit.ip') }}:</span>
              <span class="ml-2 font-mono">{{ selectedLog.ip_address }}</span>
            </div>
            <div>
              <span class="font-medium">{{ t('admin.audit.time') }}:</span>
              <span class="ml-2">{{ formatDate(selectedLog.created_at) }}</span>
            </div>
            <div v-if="selectedLog.details && Object.keys(selectedLog.details).length > 0">
              <span class="font-medium">{{ t('admin.audit.detailsData') }}:</span>
              <pre class="mt-2 bg-muted p-3 rounded-lg text-xs overflow-x-auto">{{ JSON.stringify(selectedLog.details, null, 2) }}</pre>
            </div>
          </div>
        </Card>
      </div>

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="logs.length === 0" class="text-center py-8">
        <p class="text-muted-foreground">{{ t('admin.audit.noLogs') }}</p>
      </div>

      <div v-else class="space-y-4">
        <Card class="overflow-hidden">
          <table class="w-full">
            <thead class="bg-muted/50">
              <tr>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.audit.time') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.audit.action') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.audit.user') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.audit.ip') }}</th>
                <th class="text-right p-3 text-sm font-medium"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in logs" :key="log.id" class="border-t hover:bg-muted/30 cursor-pointer" @click="selectedLog = log">
                <td class="p-3 text-sm text-muted-foreground">{{ formatDate(log.created_at) }}</td>
                <td class="p-3">
                  <span :class="['px-2 py-0.5 text-xs font-medium rounded-full', getActionColor(log.action)]">
                    {{ log.action }}
                  </span>
                </td>
                <td class="p-3 text-sm font-mono">{{ log.user_phone || '-' }}</td>
                <td class="p-3 text-sm font-mono text-muted-foreground">{{ log.ip_address }}</td>
                <td class="p-3 text-right">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="9 18 15 12 9 6" />
                  </svg>
                </td>
              </tr>
            </tbody>
          </table>
        </Card>

        <!-- Pagination -->
        <div class="flex items-center justify-between">
          <p class="text-sm text-muted-foreground">
            {{ t('admin.pagination.showing', { from: (page - 1) * limit + 1, to: Math.min(page * limit, total), total }) }}
          </p>
          <div class="flex space-x-2">
            <Button variant="outline" size="sm" @click="prevPage" :disabled="page === 1">
              {{ t('admin.pagination.prev') }}
            </Button>
            <Button variant="outline" size="sm" @click="nextPage" :disabled="page * limit >= total">
              {{ t('admin.pagination.next') }}
            </Button>
          </div>
        </div>
      </div>
    </div>
  </Layout>
</template>
