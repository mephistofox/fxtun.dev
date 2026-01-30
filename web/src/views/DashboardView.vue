<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { tunnelsApi, type Tunnel } from '@/api/client'

const { t } = useI18n()

const tunnels = ref<Tunnel[]>([])
const loading = ref(true)
const error = ref('')

async function loadTunnels() {
  loading.value = true
  error.value = ''
  try {
    const response = await tunnelsApi.list()
    tunnels.value = response.data.tunnels || []
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('dashboard.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function closeTunnel(id: string) {
  try {
    await tunnelsApi.close(id)
    tunnels.value = tunnels.value.filter((t) => t.id !== id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('dashboard.failedToClose')
  }
}

function getTunnelUrl(tunnel: Tunnel): string {
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
          <h1 class="text-2xl font-bold">{{ t('dashboard.title') }}</h1>
          <p class="text-muted-foreground">{{ t('dashboard.subtitle') }}</p>
        </div>
        <Button @click="loadTunnels" :loading="loading" variant="outline">{{ t('common.refresh') }}</Button>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="tunnels.length === 0" class="text-center py-8">
        <p class="text-muted-foreground">{{ t('dashboard.noTunnels') }}</p>
        <p class="text-sm text-muted-foreground mt-2">
          {{ t('dashboard.noTunnelsHint') }} <code class="bg-muted px-2 py-1 rounded">fxtunnel http 3000</code>
        </p>
      </div>

      <div v-else class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card v-for="tunnel in tunnels" :key="tunnel.id" class="p-4">
          <div class="flex items-start justify-between">
            <div class="space-y-1">
              <div class="flex items-center space-x-2">
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
                <span class="font-medium">{{ tunnel.name || t('dashboard.unnamed') }}</span>
              </div>
              <p class="text-sm text-muted-foreground">
                {{ t('dashboard.localPort') }}: {{ tunnel.local_port }}
              </p>
              <p class="text-sm">
                <a
                  :href="getTunnelUrl(tunnel)"
                  target="_blank"
                  class="text-primary hover:underline break-all"
                >
                  {{ getTunnelUrl(tunnel) }}
                </a>
              </p>
            </div>
            <div class="flex items-center gap-1">
              <router-link
                v-if="tunnel.type === 'http'"
                :to="`/inspect/${tunnel.id}`"
                class="text-sm text-blue-400 hover:text-blue-300 transition"
              >
                Inspect
              </router-link>
            <Button variant="ghost" size="icon" @click="closeTunnel(tunnel.id)" :title="t('dashboard.closeTunnel')">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </Button>
            </div>
          </div>
        </Card>
      </div>
    </div>
  </Layout>
</template>
