<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTunnelsStore } from '@/stores/tunnels'
import { useBundlesStore } from '@/stores/bundles'
import { toast } from '@/composables/useToast'
import {
  Button, Card, CardHeader, CardTitle, CardContent, Input, Label, Select, Badge, Tooltip
} from '@/components/ui'
import StatusIndicator from '@/components/StatusIndicator.vue'
import { Plus, Copy, X, ExternalLink, Check, RefreshCw, ChevronDown, ChevronUp, Zap, Boxes, Globe, Server, Radio, ArrowRight } from 'lucide-vue-next'
import type { TunnelType, TunnelConfig } from '@/types'

const { t } = useI18n()
const tunnelsStore = useTunnelsStore()
const bundlesStore = useBundlesStore()

const showQuickConnect = ref(true)
const tunnelType = ref<TunnelType>('http')
const localPort = ref('')
const subdomain = ref('')
const remotePort = ref('')
const copiedId = ref<string | null>(null)
const isCreating = ref(false)

const tunnelTypes = computed(() => [
  { value: 'http', label: t('tunnelTypes.http') },
  { value: 'tcp', label: t('tunnelTypes.tcp') },
  { value: 'udp', label: t('tunnelTypes.udp') },
])

// Stats
const stats = computed(() => ({
  total: tunnelsStore.activeTunnels.length,
  http: tunnelsStore.activeTunnels.filter(t => t.type === 'http').length,
  tcp: tunnelsStore.activeTunnels.filter(t => t.type === 'tcp').length,
  udp: tunnelsStore.activeTunnels.filter(t => t.type === 'udp').length,
}))

// Get tunnel icon component
function getTunnelIcon(type: TunnelType) {
  switch (type) {
    case 'http': return Globe
    case 'tcp': return Server
    case 'udp': return Radio
  }
}

// Get tunnel gradient class
function getTunnelGradient(type: TunnelType): string {
  switch (type) {
    case 'http': return 'from-emerald-500/20 to-emerald-500/5'
    case 'tcp': return 'from-blue-500/20 to-blue-500/5'
    case 'udp': return 'from-purple-500/20 to-purple-500/5'
  }
}

// Get tunnel accent color
function getTunnelAccent(type: TunnelType): string {
  switch (type) {
    case 'http': return 'text-emerald-500'
    case 'tcp': return 'text-blue-500'
    case 'udp': return 'text-purple-500'
  }
}

// Get tunnel border color
function getTunnelBorder(type: TunnelType): string {
  switch (type) {
    case 'http': return 'border-emerald-500/30 hover:border-emerald-500/50'
    case 'tcp': return 'border-blue-500/30 hover:border-blue-500/50'
    case 'udp': return 'border-purple-500/30 hover:border-purple-500/50'
  }
}

onMounted(async () => {
  await tunnelsStore.loadTunnels()
  await bundlesStore.loadBundles()
})

async function createQuickTunnel() {
  isCreating.value = true
  const config: TunnelConfig = {
    name: `quick-${tunnelType.value}-${localPort.value}`,
    type: tunnelType.value,
    localPort: parseInt(localPort.value),
    subdomain: tunnelType.value === 'http' ? subdomain.value : undefined,
    remotePort: tunnelType.value !== 'http' ? parseInt(remotePort.value) || undefined : undefined,
  }

  const result = await tunnelsStore.createTunnel(config)
  isCreating.value = false

  if (result) {
    toast({ title: t('toasts.tunnelCreated'), variant: 'success' })
    localPort.value = ''
    subdomain.value = ''
    remotePort.value = ''
  }
}

async function closeTunnel(id: string) {
  await tunnelsStore.closeTunnel(id)
  toast({ title: t('toasts.tunnelClosed'), variant: 'success' })
}

function copyToClipboard(text: string, id: string) {
  navigator.clipboard.writeText(text)
  copiedId.value = id
  toast({ title: t('toasts.urlCopied'), variant: 'success' })
  setTimeout(() => {
    copiedId.value = null
  }, 2000)
}

function getTunnelTypeBadge(type: TunnelType): 'http' | 'tcp' | 'udp' {
  return type
}
</script>

<template>
  <div class="space-y-6">
    <!-- Stats Bar -->
    <div v-if="stats.total > 0" class="grid grid-cols-4 gap-4">
      <Card class="p-3">
        <div class="flex items-center justify-between">
          <span class="text-xs text-muted-foreground">{{ t('dashboard.stats.totalTunnels') }}</span>
          <span class="text-lg font-bold text-primary">{{ stats.total }}</span>
        </div>
      </Card>
      <Card class="p-3">
        <div class="flex items-center justify-between">
          <span class="text-xs text-muted-foreground">{{ t('dashboard.stats.httpTunnels') }}</span>
          <Badge variant="http">{{ stats.http }}</Badge>
        </div>
      </Card>
      <Card class="p-3">
        <div class="flex items-center justify-between">
          <span class="text-xs text-muted-foreground">{{ t('dashboard.stats.tcpTunnels') }}</span>
          <Badge variant="tcp">{{ stats.tcp }}</Badge>
        </div>
      </Card>
      <Card class="p-3">
        <div class="flex items-center justify-between">
          <span class="text-xs text-muted-foreground">{{ t('dashboard.stats.udpTunnels') }}</span>
          <Badge variant="udp">{{ stats.udp }}</Badge>
        </div>
      </Card>
    </div>

    <!-- Quick Connect -->
    <Card>
      <CardHeader class="pb-3">
        <div class="flex items-center justify-between">
          <CardTitle class="flex items-center gap-2 text-base">
            <Zap class="h-4 w-4 text-amber-500" />
            {{ t('dashboard.quickConnect') }}
          </CardTitle>
          <Button
            variant="ghost"
            size="sm"
            @click="showQuickConnect = !showQuickConnect"
          >
            <component :is="showQuickConnect ? ChevronUp : ChevronDown" class="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>

      <Transition name="slide-up">
        <CardContent v-if="showQuickConnect" class="grid gap-4 md:grid-cols-5">
          <div class="space-y-2">
            <Label>{{ t('dashboard.tunnelType') }}</Label>
            <Select
              v-model="tunnelType"
              :options="tunnelTypes"
            />
          </div>

          <div class="space-y-2">
            <Label>{{ t('dashboard.localPort') }}</Label>
            <Input
              v-model="localPort"
              type="number"
              :placeholder="t('dashboard.localPortPlaceholder')"
            />
          </div>

          <div v-if="tunnelType === 'http'" class="space-y-2">
            <Label>{{ t('dashboard.subdomain') }} <span class="text-muted-foreground text-xs">{{ t('dashboard.optional') }}</span></Label>
            <Input
              v-model="subdomain"
              :placeholder="t('dashboard.subdomainPlaceholder')"
            />
          </div>

          <div v-else class="space-y-2">
            <Label>{{ t('dashboard.remotePort') }} <span class="text-muted-foreground text-xs">{{ t('dashboard.optional') }}</span></Label>
            <Input
              v-model="remotePort"
              type="number"
              :placeholder="t('dashboard.remotePortPlaceholder')"
            />
          </div>

          <div class="flex items-end">
            <Button
              class="w-full"
              :disabled="!localPort"
              :loading="isCreating"
              @click="createQuickTunnel"
            >
              <Plus class="h-4 w-4" />
              {{ t('dashboard.createTunnel') }}
            </Button>
          </div>
        </CardContent>
      </Transition>
    </Card>

    <!-- Active Tunnels -->
    <div>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="flex items-center gap-2 text-lg font-semibold">
          {{ t('dashboard.activeTunnels') }}
          <Badge v-if="tunnelsStore.activeTunnels.length" variant="secondary">
            {{ tunnelsStore.activeTunnels.length }}
          </Badge>
        </h2>
        <Tooltip :content="t('common.refresh')">
          <Button variant="outline" size="sm" @click="tunnelsStore.loadTunnels">
            <RefreshCw class="h-4 w-4" />
          </Button>
        </Tooltip>
      </div>

      <div v-if="tunnelsStore.activeTunnels.length === 0" class="rounded-lg border border-dashed p-8 text-center">
        <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
          <Zap class="h-6 w-6 text-muted-foreground" />
        </div>
        <p class="font-medium text-muted-foreground">{{ t('dashboard.noTunnels') }}</p>
        <p class="mt-1 text-sm text-muted-foreground">
          {{ t('dashboard.noTunnelsHint') }}
        </p>
      </div>

      <TransitionGroup v-else name="list" tag="div" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="tunnel in tunnelsStore.activeTunnels"
          :key="tunnel.id"
          :class="[
            'group relative overflow-hidden rounded-xl border-2 bg-gradient-to-br transition-all duration-300 hover:shadow-lg hover:scale-[1.02]',
            getTunnelGradient(tunnel.type),
            getTunnelBorder(tunnel.type)
          ]"
        >
          <!-- Top accent line -->
          <div :class="['absolute top-0 left-0 right-0 h-1', tunnel.type === 'http' ? 'bg-emerald-500' : tunnel.type === 'tcp' ? 'bg-blue-500' : 'bg-purple-500']" />

          <div class="p-5">
            <!-- Header -->
            <div class="flex items-start justify-between mb-4">
              <div class="flex items-center gap-3">
                <div :class="[
                  'flex h-10 w-10 items-center justify-center rounded-xl transition-transform group-hover:scale-110',
                  tunnel.type === 'http' ? 'bg-emerald-500/20' : tunnel.type === 'tcp' ? 'bg-blue-500/20' : 'bg-purple-500/20'
                ]">
                  <component :is="getTunnelIcon(tunnel.type)" :class="['h-5 w-5', getTunnelAccent(tunnel.type)]" />
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <span class="font-semibold truncate">{{ tunnel.name }}</span>
                    <StatusIndicator status="connected" size="sm" />
                  </div>
                  <Badge :variant="getTunnelTypeBadge(tunnel.type)" class="mt-1">
                    {{ tunnel.type.toUpperCase() }}
                  </Badge>
                </div>
              </div>
              <Tooltip :content="t('dashboard.closeTunnel')">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                  @click="closeTunnel(tunnel.id)"
                >
                  <X class="h-4 w-4" />
                </Button>
              </Tooltip>
            </div>

            <!-- Port info -->
            <div class="flex items-center gap-2 mb-4 text-sm">
              <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-background/50">
                <span class="text-muted-foreground">localhost:</span>
                <span class="font-mono font-semibold">{{ tunnel.localPort }}</span>
              </div>
              <ArrowRight class="h-4 w-4 text-muted-foreground" />
              <div :class="['flex items-center gap-1.5 px-2.5 py-1 rounded-lg', tunnel.type === 'http' ? 'bg-emerald-500/10' : tunnel.type === 'tcp' ? 'bg-blue-500/10' : 'bg-purple-500/10']">
                <Globe v-if="tunnel.type === 'http'" :class="['h-3.5 w-3.5', getTunnelAccent(tunnel.type)]" />
                <Server v-else-if="tunnel.type === 'tcp'" :class="['h-3.5 w-3.5', getTunnelAccent(tunnel.type)]" />
                <Radio v-else :class="['h-3.5 w-3.5', getTunnelAccent(tunnel.type)]" />
                <span :class="['font-mono font-semibold', getTunnelAccent(tunnel.type)]">
                  {{ tunnel.type === 'http' ? 'public' : 'auto' }}
                </span>
              </div>
            </div>

            <!-- URL -->
            <div class="flex items-center gap-2 p-3 rounded-lg bg-background/80 border border-border/50">
              <code class="flex-1 truncate text-xs font-medium">
                {{ tunnel.url || tunnel.remoteAddr }}
              </code>
              <div class="flex items-center gap-1">
                <Tooltip :content="copiedId === tunnel.id ? t('common.copied') : t('dashboard.copyUrl')">
                  <Button
                    variant="ghost"
                    size="icon"
                    class="h-7 w-7"
                    @click="copyToClipboard(tunnel.url || tunnel.remoteAddr || '', tunnel.id)"
                  >
                    <component
                      :is="copiedId === tunnel.id ? Check : Copy"
                      :class="['h-3.5 w-3.5', copiedId === tunnel.id ? 'text-emerald-500' : '']"
                    />
                  </Button>
                </Tooltip>
                <Tooltip v-if="tunnel.url" :content="t('dashboard.openInBrowser')">
                  <Button
                    variant="ghost"
                    size="icon"
                    class="h-7 w-7"
                    @click="tunnelsStore.openUrl(tunnel.url!)"
                  >
                    <ExternalLink class="h-3.5 w-3.5" />
                  </Button>
                </Tooltip>
              </div>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>

    <!-- Saved Bundles (Quick Access) -->
    <div v-if="bundlesStore.bundles.length > 0">
      <h2 class="mb-4 flex items-center gap-2 text-lg font-semibold">
        <Boxes class="h-5 w-5 text-muted-foreground" />
        {{ t('dashboard.savedBundles') }}
      </h2>
      <div class="grid gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
        <button
          v-for="bundle in bundlesStore.bundles.slice(0, 8)"
          :key="bundle.id"
          @click="bundlesStore.connectBundle(bundle.id)"
          :class="[
            'group flex items-center gap-3 p-3 rounded-xl border-2 transition-all duration-200 text-left',
            'hover:shadow-md hover:scale-[1.02]',
            bundle.type === 'http' ? 'border-emerald-500/20 hover:border-emerald-500/40 hover:bg-emerald-500/5' :
            bundle.type === 'tcp' ? 'border-blue-500/20 hover:border-blue-500/40 hover:bg-blue-500/5' :
            'border-purple-500/20 hover:border-purple-500/40 hover:bg-purple-500/5'
          ]"
        >
          <div :class="[
            'flex h-9 w-9 items-center justify-center rounded-lg transition-transform group-hover:scale-110',
            bundle.type === 'http' ? 'bg-emerald-500/15' : bundle.type === 'tcp' ? 'bg-blue-500/15' : 'bg-purple-500/15'
          ]">
            <component :is="getTunnelIcon(bundle.type)" :class="['h-4 w-4', getTunnelAccent(bundle.type)]" />
          </div>
          <div class="flex-1 min-w-0">
            <p class="font-medium truncate">{{ bundle.name }}</p>
            <p class="text-xs text-muted-foreground">
              :{{ bundle.localPort }} <span class="opacity-50">→</span> {{ bundle.subdomain || bundle.remotePort || 'auto' }}
            </p>
          </div>
          <Plus :class="['h-4 w-4 opacity-0 group-hover:opacity-100 transition-opacity', getTunnelAccent(bundle.type)]" />
        </button>
        <button
          v-if="bundlesStore.bundles.length > 8"
          @click="$router.push('/bundles')"
          class="flex items-center justify-center gap-2 p-3 rounded-xl border-2 border-dashed border-muted-foreground/20 hover:border-muted-foreground/40 transition-colors text-muted-foreground hover:text-foreground"
        >
          <Boxes class="h-4 w-4" />
          <span class="text-sm font-medium">+{{ bundlesStore.bundles.length - 8 }} {{ t('common.more') }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
