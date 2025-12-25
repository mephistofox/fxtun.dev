<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { adminApi, type AdminTunnel } from '@/api/client'

const { t } = useI18n()

const tunnels = ref<AdminTunnel[]>([])
const loading = ref(true)
const error = ref('')

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

async function closeTunnel(tunnel: AdminTunnel) {
  if (!confirm(t('admin.tunnels.confirmClose'))) return

  try {
    await adminApi.closeTunnel(tunnel.id)
    tunnels.value = tunnels.value.filter((t) => t.id !== tunnel.id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.tunnels.failedToClose')
  }
}

function getTunnelUrl(tunnel: AdminTunnel): string {
  if (tunnel.url) return tunnel.url
  if (tunnel.type === 'http' && tunnel.subdomain) {
    return `https://${tunnel.subdomain}.mfdev.ru`
  }
  if (tunnel.remote_port) {
    return `${tunnel.type}://mfdev.ru:${tunnel.remote_port}`
  }
  return '-'
}

onMounted(loadTunnels)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ t('admin.tunnels.title') }}</h1>
          <p class="text-muted-foreground">{{ t('admin.tunnels.subtitle') }}</p>
        </div>
        <Button @click="loadTunnels" :loading="loading" variant="outline">{{ t('common.refresh') }}</Button>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="tunnels.length === 0" class="text-center py-8">
        <p class="text-muted-foreground">{{ t('admin.tunnels.noTunnels') }}</p>
      </div>

      <div v-else class="space-y-4">
        <Card class="overflow-hidden">
          <table class="w-full">
            <thead class="bg-muted/50">
              <tr>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.tunnels.type') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.tunnels.url') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.tunnels.localPort') }}</th>
                <th class="text-left p-3 text-sm font-medium">{{ t('admin.tunnels.owner') }}</th>
                <th class="text-right p-3 text-sm font-medium">{{ t('admin.users.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="tunnel in tunnels" :key="tunnel.id" class="border-t">
                <td class="p-3">
                  <span
                    :class="[
                      'px-2 py-0.5 text-xs font-medium rounded-full uppercase',
                      tunnel.type === 'http'
                        ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                        : tunnel.type === 'tcp'
                          ? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
                          : 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
                    ]"
                  >
                    {{ tunnel.type }}
                  </span>
                </td>
                <td class="p-3">
                  <a
                    :href="getTunnelUrl(tunnel)"
                    target="_blank"
                    class="text-primary hover:underline text-sm"
                  >
                    {{ getTunnelUrl(tunnel) }}
                  </a>
                </td>
                <td class="p-3 text-sm">{{ tunnel.local_port }}</td>
                <td class="p-3 text-sm font-mono">{{ tunnel.user_phone || '-' }}</td>
                <td class="p-3">
                  <div class="flex justify-end">
                    <Button variant="ghost" size="icon" @click="closeTunnel(tunnel)" :title="t('admin.tunnels.close')">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18" />
                        <line x1="6" y1="6" x2="18" y2="18" />
                      </svg>
                    </Button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </Card>

        <p class="text-sm text-muted-foreground text-center">
          {{ t('admin.tunnels.total', { count: tunnels.length }) }}
        </p>
      </div>
    </div>
  </Layout>
</template>
