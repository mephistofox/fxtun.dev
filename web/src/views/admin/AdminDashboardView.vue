<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { adminApi, type AdminStats } from '@/api/client'

const { t } = useI18n()

const stats = ref<AdminStats | null>(null)
const loading = ref(true)
const error = ref('')

async function loadStats() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.getStats()
    stats.value = response.data
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.failedToLoad')
  } finally {
    loading.value = false
  }
}

onMounted(loadStats)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ t('admin.dashboard.title') }}</h1>
          <p class="text-muted-foreground">{{ t('admin.dashboard.subtitle') }}</p>
        </div>
        <Button @click="loadStats" :loading="loading" variant="outline">{{ t('common.refresh') }}</Button>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="stats" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <!-- Active Clients -->
        <Card class="p-6">
          <div class="flex items-center space-x-4">
            <div class="p-3 bg-blue-100 dark:bg-blue-900 rounded-lg">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-blue-600 dark:text-blue-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            </div>
            <div>
              <p class="text-sm text-muted-foreground">{{ t('admin.dashboard.activeClients') }}</p>
              <p class="text-3xl font-bold">{{ stats.active_clients }}</p>
            </div>
          </div>
        </Card>

        <!-- Active Tunnels -->
        <Card class="p-6">
          <div class="flex items-center space-x-4">
            <div class="p-3 bg-green-100 dark:bg-green-900 rounded-lg">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-green-600 dark:text-green-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M5 12.55a11 11 0 0 1 14.08 0" />
                <path d="M1.42 9a16 16 0 0 1 21.16 0" />
                <path d="M8.53 16.11a6 6 0 0 1 6.95 0" />
                <line x1="12" y1="20" x2="12.01" y2="20" />
              </svg>
            </div>
            <div>
              <p class="text-sm text-muted-foreground">{{ t('admin.dashboard.activeTunnels') }}</p>
              <p class="text-3xl font-bold">{{ stats.active_tunnels }}</p>
            </div>
          </div>
        </Card>

        <!-- Total Users -->
        <Card class="p-6">
          <div class="flex items-center space-x-4">
            <div class="p-3 bg-purple-100 dark:bg-purple-900 rounded-lg">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-purple-600 dark:text-purple-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            </div>
            <div>
              <p class="text-sm text-muted-foreground">{{ t('admin.dashboard.totalUsers') }}</p>
              <p class="text-3xl font-bold">{{ stats.total_users }}</p>
            </div>
          </div>
        </Card>

        <!-- HTTP Tunnels -->
        <Card class="p-6">
          <div class="flex items-center space-x-4">
            <div class="p-3 bg-orange-100 dark:bg-orange-900 rounded-lg">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-orange-600 dark:text-orange-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="2" y1="12" x2="22" y2="12" />
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
              </svg>
            </div>
            <div>
              <p class="text-sm text-muted-foreground">{{ t('admin.dashboard.httpTunnels') }}</p>
              <p class="text-3xl font-bold">{{ stats.http_tunnels }}</p>
            </div>
          </div>
        </Card>

        <!-- TCP Tunnels -->
        <Card class="p-6">
          <div class="flex items-center space-x-4">
            <div class="p-3 bg-cyan-100 dark:bg-cyan-900 rounded-lg">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-cyan-600 dark:text-cyan-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
                <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
                <line x1="6" y1="6" x2="6.01" y2="6" />
                <line x1="6" y1="18" x2="6.01" y2="18" />
              </svg>
            </div>
            <div>
              <p class="text-sm text-muted-foreground">{{ t('admin.dashboard.tcpTunnels') }}</p>
              <p class="text-3xl font-bold">{{ stats.tcp_tunnels }}</p>
            </div>
          </div>
        </Card>

        <!-- UDP Tunnels -->
        <Card class="p-6">
          <div class="flex items-center space-x-4">
            <div class="p-3 bg-pink-100 dark:bg-pink-900 rounded-lg">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-pink-600 dark:text-pink-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
              </svg>
            </div>
            <div>
              <p class="text-sm text-muted-foreground">{{ t('admin.dashboard.udpTunnels') }}</p>
              <p class="text-3xl font-bold">{{ stats.udp_tunnels }}</p>
            </div>
          </div>
        </Card>
      </div>
    </div>
  </Layout>
</template>
