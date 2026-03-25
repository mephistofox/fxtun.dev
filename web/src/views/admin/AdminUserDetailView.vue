<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { adminApi, type AdminUserDetail } from '@/api/client'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const detail = ref<AdminUserDetail | null>(null)
const loading = ref(true)
const error = ref('')

const userId = computed(() => Number(route.params.id))

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.getUserDetail(userId.value)
    detail.value = response.data
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to load user'
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr?: string | null): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function statusColor(status: string): string {
  switch (status) {
    case 'success': return 'bg-green-500/10 text-green-600 dark:text-green-400'
    case 'failed': return 'bg-red-500/10 text-red-600 dark:text-red-400'
    case 'pending': return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400'
    case 'active': return 'bg-green-500/10 text-green-600 dark:text-green-400'
    case 'cancelled': return 'bg-orange-500/10 text-orange-600 dark:text-orange-400'
    case 'expired': return 'bg-gray-500/10 text-gray-600 dark:text-gray-400'
    default: return 'bg-muted text-muted-foreground'
  }
}

function tunnelTypeColor(type: string): string {
  switch (type) {
    case 'http': return 'bg-blue-500/10 text-blue-600 dark:text-blue-400'
    case 'tcp': return 'bg-purple-500/10 text-purple-600 dark:text-purple-400'
    case 'udp': return 'bg-orange-500/10 text-orange-600 dark:text-orange-400'
    default: return 'bg-muted text-muted-foreground'
  }
}

onMounted(loadDetail)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex items-center gap-3">
        <Button variant="ghost" size="sm" @click="router.push({ name: 'admin-users' })">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
          {{ t('admin.userDetail.back') }}
        </Button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-12 text-muted-foreground text-sm">{{ t('common.loading') }}</div>

      <!-- Error -->
      <div v-else-if="error" class="bg-destructive/10 text-destructive px-4 py-3 rounded-md text-sm">{{ error }}</div>

      <!-- Content -->
      <template v-else-if="detail">
        <!-- User Info Card -->
        <Card class="p-5">
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-4">
              <!-- Avatar -->
              <div class="w-14 h-14 rounded-full bg-muted flex items-center justify-center overflow-hidden shrink-0">
                <img v-if="detail.user.avatar_url" :src="detail.user.avatar_url" class="w-full h-full object-cover" />
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-7 w-7 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              </div>
              <div>
                <h1 class="text-xl font-bold text-foreground">{{ detail.user.display_name || detail.user.email || detail.user.phone }}</h1>
                <div class="flex items-center gap-2 mt-1 text-sm text-muted-foreground">
                  <span class="font-mono">ID: {{ detail.user.id }}</span>
                  <span v-if="detail.user.email">{{ detail.user.email }}</span>
                  <span v-if="detail.user.phone">{{ detail.user.phone }}</span>
                </div>
                <div class="flex items-center gap-2 mt-2">
                  <span :class="['inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full', detail.user.is_active ? 'bg-green-500/10 text-green-600 dark:text-green-400' : 'bg-red-500/10 text-red-600 dark:text-red-400']">
                    <span :class="['w-1.5 h-1.5 rounded-full mr-1', detail.user.is_active ? 'bg-green-500' : 'bg-red-500']" />
                    {{ detail.user.is_active ? t('admin.users.active') : t('admin.users.blocked') }}
                  </span>
                  <span v-if="detail.user.is_admin" class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full bg-purple-500/10 text-purple-600 dark:text-purple-400">
                    {{ t('admin.users.admin') }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Info Grid -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mt-6 pt-4 border-t border-border/50">
            <div>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.createdAt') }}</p>
              <p class="text-sm font-medium mt-0.5">{{ formatDate(detail.user.created_at) }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.lastLogin') }}</p>
              <p class="text-sm font-medium mt-0.5">{{ detail.user.last_login_at ? formatDate(detail.user.last_login_at) : t('admin.userDetail.never') }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.plan') }}</p>
              <p class="text-sm font-medium mt-0.5">{{ detail.user.plan?.name || '-' }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.tokens') }} / {{ t('admin.userDetail.domains') }}</p>
              <p class="text-sm font-medium mt-0.5">{{ detail.token_count }} / {{ detail.domain_count }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.github') }}</p>
              <p class="text-sm font-medium mt-0.5">{{ detail.user.github_id ? t('admin.userDetail.linked') : t('admin.userDetail.notLinked') }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.google') }}</p>
              <p class="text-sm font-medium mt-0.5">{{ detail.user.google_id ? t('admin.userDetail.linked') : t('admin.userDetail.notLinked') }}</p>
            </div>
          </div>
        </Card>

        <!-- Tunnel Stats -->
        <Card v-if="detail.tunnel_stats" class="p-5">
          <h2 class="text-sm font-semibold text-foreground mb-3">{{ t('admin.userDetail.tunnelStats') }}</h2>
          <div class="grid grid-cols-3 gap-4">
            <div class="text-center">
              <p class="text-2xl font-bold text-foreground">{{ detail.tunnel_stats.total_connections }}</p>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.connections') }}</p>
            </div>
            <div class="text-center">
              <p class="text-2xl font-bold text-foreground">{{ formatBytes(detail.tunnel_stats.total_bytes_sent) }}</p>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.totalBytesSent') }}</p>
            </div>
            <div class="text-center">
              <p class="text-2xl font-bold text-foreground">{{ formatBytes(detail.tunnel_stats.total_bytes_received) }}</p>
              <p class="text-xs text-muted-foreground">{{ t('admin.userDetail.totalBytesReceived') }}</p>
            </div>
          </div>
        </Card>

        <!-- Payments -->
        <Card class="overflow-hidden">
          <div class="px-5 py-3 border-b border-border/50">
            <h2 class="text-sm font-semibold text-foreground">{{ t('admin.userDetail.payments') }} ({{ detail.payments?.length || 0 }})</h2>
          </div>
          <div v-if="!detail.payments || detail.payments.length === 0" class="px-5 py-6 text-center text-sm text-muted-foreground">
            {{ t('admin.userDetail.noPayments') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-muted/30">
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.invoiceId') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.amount') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.provider') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.status') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.recurring') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.date') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="payment in detail.payments" :key="payment.id" class="border-b border-border/50 hover:bg-muted/10">
                  <td class="px-3 py-2 font-mono text-xs">#{{ payment.invoice_id }}</td>
                  <td class="px-3 py-2 font-medium">{{ payment.amount }} {{ payment.currency }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ payment.provider || '-' }}</td>
                  <td class="px-3 py-2">
                    <span :class="['inline-flex items-center px-1.5 py-0.5 text-[11px] font-medium rounded-full', statusColor(payment.status)]">
                      {{ t('admin.userDetail.' + payment.status) }}
                    </span>
                  </td>
                  <td class="px-3 py-2 text-xs">{{ payment.is_recurring ? t('admin.userDetail.yes') : t('admin.userDetail.no') }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ formatDate(payment.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>

        <!-- Subscriptions -->
        <Card class="overflow-hidden">
          <div class="px-5 py-3 border-b border-border/50">
            <h2 class="text-sm font-semibold text-foreground">{{ t('admin.userDetail.subscriptions') }} ({{ detail.subscriptions?.length || 0 }})</h2>
          </div>
          <div v-if="!detail.subscriptions || detail.subscriptions.length === 0" class="px-5 py-6 text-center text-sm text-muted-foreground">
            {{ t('admin.userDetail.noSubscriptions') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-muted/30">
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">ID</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.plan') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.status') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.recurring') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.period') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.date') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="sub in detail.subscriptions" :key="sub.id" class="border-b border-border/50 hover:bg-muted/10">
                  <td class="px-3 py-2 font-mono text-xs">#{{ sub.id }}</td>
                  <td class="px-3 py-2 font-medium text-xs">{{ sub.plan?.name || '-' }}</td>
                  <td class="px-3 py-2">
                    <span :class="['inline-flex items-center px-1.5 py-0.5 text-[11px] font-medium rounded-full', statusColor(sub.status)]">
                      {{ sub.status }}
                    </span>
                  </td>
                  <td class="px-3 py-2 text-xs">{{ sub.recurring ? t('admin.userDetail.yes') : t('admin.userDetail.no') }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">
                    {{ sub.current_period_start ? formatDate(sub.current_period_start) : '-' }}
                    <span v-if="sub.current_period_end"> &mdash; {{ formatDate(sub.current_period_end) }}</span>
                  </td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ formatDate(sub.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>

        <!-- Tunnel History -->
        <Card class="overflow-hidden">
          <div class="px-5 py-3 border-b border-border/50">
            <h2 class="text-sm font-semibold text-foreground">{{ t('admin.userDetail.tunnelHistory') }} ({{ detail.tunnel_history?.length || 0 }})</h2>
          </div>
          <div v-if="!detail.tunnel_history || detail.tunnel_history.length === 0" class="px-5 py-6 text-center text-sm text-muted-foreground">
            {{ t('admin.userDetail.noHistory') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-muted/30">
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.tunnelType') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.url') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.localPort') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.connectedAt') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.disconnectedAt') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.bytesSent') }}</th>
                  <th class="text-left px-3 py-2 text-xs font-medium text-muted-foreground">{{ t('admin.userDetail.bytesReceived') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in detail.tunnel_history" :key="entry.id" class="border-b border-border/50 hover:bg-muted/10">
                  <td class="px-3 py-2">
                    <span :class="['inline-flex items-center px-1.5 py-0.5 text-[11px] font-medium rounded-full uppercase', tunnelTypeColor(entry.tunnel_type)]">
                      {{ entry.tunnel_type }}
                    </span>
                  </td>
                  <td class="px-3 py-2 font-mono text-xs max-w-[200px] truncate" :title="entry.url || entry.remote_addr || '-'">
                    {{ entry.url || entry.remote_addr || '-' }}
                  </td>
                  <td class="px-3 py-2 font-mono text-xs">{{ entry.local_port }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ formatDate(entry.connected_at) }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ entry.disconnected_at ? formatDate(entry.disconnected_at) : '-' }}</td>
                  <td class="px-3 py-2 text-xs">{{ formatBytes(entry.bytes_sent) }}</td>
                  <td class="px-3 py-2 text-xs">{{ formatBytes(entry.bytes_received) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      </template>
    </div>
  </Layout>
</template>
