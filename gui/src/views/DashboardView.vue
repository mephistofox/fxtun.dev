<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTunnelsStore } from '@/stores/tunnels'
import { useBundlesStore } from '@/stores/bundles'
import { toast } from '@/composables/useToast'
import {
  Button, Input, Label, Select, Badge, Tooltip
} from '@/components/ui'
import StatusIndicator from '@/components/StatusIndicator.vue'
import {
  Plus, Copy, X, ExternalLink, Check, RefreshCw, ChevronDown, ChevronUp,
  Zap, Boxes, Globe, Server, Radio, ArrowRight, ArrowUpRight, ArrowDownRight,
  Search, Shield, Database, Gamepad2
} from 'lucide-vue-next'
import { formatBytes } from '@/utils/format'
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

interface Template {
  key: string
  icon: typeof Globe
  type?: TunnelType
  port?: string
  subdomain?: string
}

const templates = computed<Template[]>(() => [
  { key: 'webDev', icon: Globe, type: 'http', port: '3000' },
  { key: 'api', icon: Server, type: 'http', port: '8080', subdomain: 'api' },
  { key: 'ssh', icon: Shield, type: 'tcp', port: '22' },
  { key: 'database', icon: Database, type: 'tcp', port: '5432' },
  { key: 'game', icon: Gamepad2, type: 'tcp', port: '25565' },
  { key: 'custom', icon: Plus },
])

function applyTemplate(tpl: Template) {
  if (tpl.type) {
    tunnelType.value = tpl.type
    localPort.value = tpl.port || ''
    subdomain.value = tpl.subdomain || ''
    remotePort.value = ''
  } else {
    tunnelType.value = 'http'
    localPort.value = ''
    subdomain.value = ''
    remotePort.value = ''
  }
  showQuickConnect.value = true
}

const tunnelTypes = computed(() => [
  { value: 'http', label: 'HTTP' },
  { value: 'tcp', label: 'TCP' },
  { value: 'udp', label: 'UDP' },
])

function getTunnelIcon(type: TunnelType) {
  switch (type) {
    case 'http': return Globe
    case 'tcp': return Server
    case 'udp': return Radio
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
</script>

<template>
  <div class="space-y-5">
    <!-- Quick Connect -->
    <div class="cyber-card overflow-hidden">
      <button
        @click="showQuickConnect = !showQuickConnect"
        class="w-full flex items-center justify-between p-4 hover:bg-muted/30 transition-colors"
      >
        <div class="flex items-center gap-3">
          <div class="h-9 w-9 rounded-lg bg-gradient-to-br from-primary to-accent flex items-center justify-center">
            <Zap class="h-4 w-4 text-primary-foreground" />
          </div>
          <span class="font-semibold">{{ t('dashboard.quickConnect') }}</span>
        </div>
        <component :is="showQuickConnect ? ChevronUp : ChevronDown" class="h-4 w-4 text-muted-foreground" />
      </button>

      <Transition name="slide-up">
        <div v-if="showQuickConnect" class="px-4 pb-4 pt-0">
          <div class="flex flex-wrap items-end gap-3">
            <div class="w-28">
              <Label class="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5 block">{{ t('dashboard.tunnelType') }}</Label>
              <Select v-model="tunnelType" :options="tunnelTypes" class="h-9" />
            </div>

            <div class="w-24">
              <Label class="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5 block">{{ t('dashboard.localPort') }}</Label>
              <Input v-model="localPort" type="number" placeholder="3000" class="h-9 font-mono" />
            </div>

            <div v-if="tunnelType === 'http'" class="flex-1 min-w-[120px]">
              <Label class="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5 block">
                {{ t('dashboard.subdomain') }}
                <span class="text-muted-foreground/50">({{ t('dashboard.optional') }})</span>
              </Label>
              <Input v-model="subdomain" placeholder="my-app" class="h-9" />
            </div>

            <div v-else class="w-24">
              <Label class="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5 block">
                {{ t('dashboard.remotePort') }}
              </Label>
              <Input v-model="remotePort" type="number" placeholder="auto" class="h-9 font-mono" />
            </div>

            <Button
              class="h-9 px-4 bg-gradient-to-r from-primary to-primary hover:to-accent shadow-lg shadow-primary/20"
              :disabled="!localPort"
              :loading="isCreating"
              @click="createQuickTunnel"
            >
              <Plus class="h-4 w-4 mr-1.5" />
              {{ t('dashboard.createTunnel') }}
            </Button>
          </div>
        </div>
      </Transition>
    </div>

    <!-- Active Tunnels -->
    <div>
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
          <h2 class="font-semibold">{{ t('dashboard.activeTunnels') }}</h2>
          <Badge v-if="tunnelsStore.activeTunnels.length" variant="secondary" class="font-mono text-xs">
            {{ tunnelsStore.activeTunnels.length }}
          </Badge>
        </div>
        <Button variant="ghost" size="sm" @click="tunnelsStore.loadTunnels" class="h-8 w-8 p-0">
          <RefreshCw class="h-4 w-4" />
        </Button>
      </div>

      <!-- Empty state — use-case templates -->
      <div v-if="tunnelsStore.activeTunnels.length === 0">
        <p class="text-sm text-muted-foreground mb-3">{{ t('dashboard.templates.title') }}</p>
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
          <button
            v-for="tpl in templates"
            :key="tpl.key"
            @click="applyTemplate(tpl)"
            class="group flex flex-col items-center gap-2 p-4 rounded-xl border border-border/50 bg-card/60 hover:bg-muted/40 hover:border-primary/40 transition-all text-center hover:scale-[1.02]"
          >
            <div class="h-10 w-10 rounded-lg bg-muted/50 group-hover:bg-primary/10 flex items-center justify-center transition-colors">
              <component :is="tpl.icon" class="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
            </div>
            <div>
              <p class="text-sm font-medium">{{ t(`dashboard.templates.${tpl.key}`) }}</p>
              <p class="text-[10px] text-muted-foreground">{{ t(`dashboard.templates.${tpl.key}Hint`) }}</p>
              <p v-if="tpl.type" class="text-[10px] font-mono text-muted-foreground/60 mt-0.5">
                {{ tpl.type.toUpperCase() }} :{{ tpl.port }}
              </p>
            </div>
          </button>
        </div>
      </div>

      <!-- Tunnel cards -->
      <TransitionGroup v-else name="list" tag="div" class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="tunnel in tunnelsStore.activeTunnels"
          :key="tunnel.id"
          :class="[
            'group relative overflow-hidden rounded-xl border transition-all duration-200 hover:shadow-lg',
            tunnel.type === 'http'
              ? 'border-type-http/30 hover:border-type-http/60 bg-gradient-to-br from-type-http/5 to-transparent'
              : tunnel.type === 'tcp'
                ? 'border-type-tcp/30 hover:border-type-tcp/60 bg-gradient-to-br from-type-tcp/5 to-transparent'
                : 'border-type-udp/30 hover:border-type-udp/60 bg-gradient-to-br from-type-udp/5 to-transparent'
          ]"
        >
          <div class="p-4">
            <!-- Header -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <div
                  :class="[
                    'flex h-8 w-8 items-center justify-center rounded-lg',
                    tunnel.type === 'http' ? 'bg-type-http/20' : tunnel.type === 'tcp' ? 'bg-type-tcp/20' : 'bg-type-udp/20'
                  ]"
                >
                  <component
                    :is="getTunnelIcon(tunnel.type)"
                    :class="['h-4 w-4', tunnel.type === 'http' ? 'text-type-http' : tunnel.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp']"
                  />
                </div>
                <div>
                  <div class="flex items-center gap-1.5">
                    <span class="font-medium text-sm truncate max-w-[120px]">{{ tunnel.name }}</span>
                    <StatusIndicator status="connected" size="sm" />
                  </div>
                  <Badge :variant="tunnel.type" class="text-[10px] mt-0.5">{{ tunnel.type.toUpperCase() }}</Badge>
                </div>
              </div>
              <Button
                variant="ghost"
                size="icon"
                class="h-7 w-7 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                @click="closeTunnel(tunnel.id)"
              >
                <X class="h-3.5 w-3.5" />
              </Button>
            </div>

            <!-- Port mapping -->
            <div class="flex items-center gap-2 text-xs mb-3">
              <code class="px-2 py-1 rounded bg-muted/50 font-mono">:{{ tunnel.localPort }}</code>
              <ArrowRight class="h-3 w-3 text-muted-foreground" />
              <code
                :class="[
                  'px-2 py-1 rounded font-mono',
                  tunnel.type === 'http' ? 'bg-type-http/10 text-type-http' : tunnel.type === 'tcp' ? 'bg-type-tcp/10 text-type-tcp' : 'bg-type-udp/10 text-type-udp'
                ]"
              >
                {{ tunnel.type === 'http' ? 'public' : tunnel.remoteAddr?.split(':')[1] || 'auto' }}
              </code>
            </div>

            <!-- Traffic stats -->
            <div v-if="tunnel.bytesSent > 0 || tunnel.bytesReceived > 0" class="flex items-center gap-3 text-xs mb-3">
              <span class="flex items-center gap-0.5 text-type-http">
                <ArrowUpRight class="h-3 w-3" />
                {{ formatBytes(tunnel.bytesSent) }}
              </span>
              <span class="text-muted-foreground/50">/</span>
              <span class="flex items-center gap-0.5 text-type-tcp">
                <ArrowDownRight class="h-3 w-3" />
                {{ formatBytes(tunnel.bytesReceived) }}
              </span>
            </div>

            <!-- URL -->
            <div class="flex items-center gap-1.5 p-2 rounded-lg bg-background/60 border border-border/30">
              <code class="flex-1 truncate text-xs font-mono">{{ tunnel.url || tunnel.remoteAddr }}</code>
              <Tooltip :content="copiedId === tunnel.id ? t('common.copied') : t('dashboard.copyUrl')">
                <Button variant="ghost" size="icon" class="h-6 w-6" @click="copyToClipboard(tunnel.url || tunnel.remoteAddr || '', tunnel.id)">
                  <component :is="copiedId === tunnel.id ? Check : Copy" :class="['h-3 w-3', copiedId === tunnel.id && 'text-success']" />
                </Button>
              </Tooltip>
              <Tooltip v-if="tunnel.type === 'http'" content="Inspect traffic">
                <router-link :to="`/inspect/${tunnel.id}`">
                  <Button variant="ghost" size="icon" class="h-6 w-6">
                    <Search class="h-3 w-3" />
                  </Button>
                </router-link>
              </Tooltip>
              <Tooltip v-if="tunnel.url" :content="t('dashboard.openInBrowser')">
                <Button variant="ghost" size="icon" class="h-6 w-6" @click="tunnelsStore.openUrl(tunnel.url!)">
                  <ExternalLink class="h-3 w-3" />
                </Button>
              </Tooltip>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>

    <!-- Saved Bundles -->
    <div v-if="bundlesStore.bundles.length > 0">
      <div class="flex items-center gap-2 mb-3">
        <Boxes class="h-4 w-4 text-muted-foreground" />
        <h2 class="font-semibold">{{ t('dashboard.savedBundles') }}</h2>
      </div>

      <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <button
          v-for="bundle in bundlesStore.bundles.slice(0, 8)"
          :key="bundle.id"
          @click="bundlesStore.connectBundle(bundle.id)"
          :class="[
            'group flex items-center gap-2.5 p-3 rounded-lg border transition-all text-left hover:scale-[1.02]',
            bundle.type === 'http'
              ? 'border-type-http/20 hover:border-type-http/50 hover:bg-type-http/5'
              : bundle.type === 'tcp'
                ? 'border-type-tcp/20 hover:border-type-tcp/50 hover:bg-type-tcp/5'
                : 'border-type-udp/20 hover:border-type-udp/50 hover:bg-type-udp/5'
          ]"
        >
          <div
            :class="[
              'flex h-8 w-8 items-center justify-center rounded-lg transition-transform group-hover:scale-110',
              bundle.type === 'http' ? 'bg-type-http/15' : bundle.type === 'tcp' ? 'bg-type-tcp/15' : 'bg-type-udp/15'
            ]"
          >
            <component
              :is="getTunnelIcon(bundle.type)"
              :class="['h-4 w-4', bundle.type === 'http' ? 'text-type-http' : bundle.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp']"
            />
          </div>
          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">{{ bundle.name }}</p>
            <p class="text-[10px] text-muted-foreground font-mono">
              :{{ bundle.localPort }} → {{ bundle.subdomain || bundle.remotePort || 'auto' }}
            </p>
          </div>
          <Plus
            :class="['h-4 w-4 opacity-0 group-hover:opacity-100 transition-opacity', bundle.type === 'http' ? 'text-type-http' : bundle.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp']"
          />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cyber-card {
  @apply relative rounded-xl border border-border/50 bg-card/80 backdrop-blur-sm;
}
</style>
