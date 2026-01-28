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
  Zap, Boxes, Globe, Server, Radio, ArrowRight, Activity
} from 'lucide-vue-next'
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

const stats = computed(() => ({
  total: tunnelsStore.activeTunnels.length,
  http: tunnelsStore.activeTunnels.filter(t => t.type === 'http').length,
  tcp: tunnelsStore.activeTunnels.filter(t => t.type === 'tcp').length,
  udp: tunnelsStore.activeTunnels.filter(t => t.type === 'udp').length,
}))

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
  <div class="space-y-6">
    <!-- Stats Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="cyber-card group p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-muted-foreground uppercase tracking-wider">{{ t('dashboard.stats.totalTunnels') }}</p>
            <p class="text-3xl font-display font-bold mt-1 gradient-text">{{ stats.total }}</p>
          </div>
          <div class="h-12 w-12 rounded-xl bg-primary/10 flex items-center justify-center group-hover:scale-110 transition-transform">
            <Activity class="h-6 w-6 text-primary" />
          </div>
        </div>
      </div>

      <div class="cyber-card group p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-muted-foreground uppercase tracking-wider">HTTP</p>
            <p class="text-3xl font-display font-bold mt-1 text-type-http">{{ stats.http }}</p>
          </div>
          <div class="h-12 w-12 rounded-xl bg-type-http/10 flex items-center justify-center group-hover:scale-110 transition-transform">
            <Globe class="h-6 w-6 text-type-http" />
          </div>
        </div>
      </div>

      <div class="cyber-card group p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-muted-foreground uppercase tracking-wider">TCP</p>
            <p class="text-3xl font-display font-bold mt-1 text-type-tcp">{{ stats.tcp }}</p>
          </div>
          <div class="h-12 w-12 rounded-xl bg-type-tcp/10 flex items-center justify-center group-hover:scale-110 transition-transform">
            <Server class="h-6 w-6 text-type-tcp" />
          </div>
        </div>
      </div>

      <div class="cyber-card group p-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs text-muted-foreground uppercase tracking-wider">UDP</p>
            <p class="text-3xl font-display font-bold mt-1 text-type-udp">{{ stats.udp }}</p>
          </div>
          <div class="h-12 w-12 rounded-xl bg-type-udp/10 flex items-center justify-center group-hover:scale-110 transition-transform">
            <Radio class="h-6 w-6 text-type-udp" />
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Connect -->
    <div class="cyber-card overflow-hidden">
      <div class="p-4 border-b border-border/50">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="h-10 w-10 rounded-xl bg-gradient-to-br from-primary to-accent flex items-center justify-center">
              <Zap class="h-5 w-5 text-primary-foreground" />
            </div>
            <div>
              <h2 class="font-display font-semibold">{{ t('dashboard.quickConnect') }}</h2>
              <p class="text-xs text-muted-foreground">{{ t('dashboard.quickConnectDesc') || 'Create a tunnel instantly' }}</p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            @click="showQuickConnect = !showQuickConnect"
            class="h-8 w-8 p-0"
          >
            <component :is="showQuickConnect ? ChevronUp : ChevronDown" class="h-4 w-4" />
          </Button>
        </div>
      </div>

      <Transition name="slide-up">
        <div v-if="showQuickConnect" class="p-4 bg-muted/20">
          <div class="grid gap-4 md:grid-cols-5">
            <div class="space-y-2">
              <Label class="text-xs uppercase tracking-wider text-muted-foreground">{{ t('dashboard.tunnelType') }}</Label>
              <Select
                v-model="tunnelType"
                :options="tunnelTypes"
                class="bg-background/50"
              />
            </div>

            <div class="space-y-2">
              <Label class="text-xs uppercase tracking-wider text-muted-foreground">{{ t('dashboard.localPort') }}</Label>
              <Input
                v-model="localPort"
                type="number"
                :placeholder="t('dashboard.localPortPlaceholder')"
                class="bg-background/50 font-mono"
              />
            </div>

            <div v-if="tunnelType === 'http'" class="space-y-2">
              <Label class="text-xs uppercase tracking-wider text-muted-foreground">
                {{ t('dashboard.subdomain') }}
                <span class="text-muted-foreground/50 normal-case">{{ t('dashboard.optional') }}</span>
              </Label>
              <Input
                v-model="subdomain"
                :placeholder="t('dashboard.subdomainPlaceholder')"
                class="bg-background/50"
              />
            </div>

            <div v-else class="space-y-2">
              <Label class="text-xs uppercase tracking-wider text-muted-foreground">
                {{ t('dashboard.remotePort') }}
                <span class="text-muted-foreground/50 normal-case">{{ t('dashboard.optional') }}</span>
              </Label>
              <Input
                v-model="remotePort"
                type="number"
                :placeholder="t('dashboard.remotePortPlaceholder')"
                class="bg-background/50 font-mono"
              />
            </div>

            <div class="flex items-end">
              <Button
                class="w-full h-10 bg-gradient-to-r from-primary to-primary hover:to-accent transition-all duration-300 shadow-lg shadow-primary/20"
                :disabled="!localPort"
                :loading="isCreating"
                @click="createQuickTunnel"
              >
                <Plus class="h-4 w-4 mr-2" />
                {{ t('dashboard.createTunnel') }}
              </Button>
            </div>
          </div>
        </div>
      </Transition>
    </div>

    <!-- Active Tunnels -->
    <div>
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-3">
          <h2 class="font-display font-semibold text-lg">{{ t('dashboard.activeTunnels') }}</h2>
          <Badge v-if="tunnelsStore.activeTunnels.length" variant="secondary" class="font-mono">
            {{ tunnelsStore.activeTunnels.length }}
          </Badge>
        </div>
        <Tooltip :content="t('common.refresh')">
          <Button variant="outline" size="sm" @click="tunnelsStore.loadTunnels" class="h-8 w-8 p-0">
            <RefreshCw class="h-4 w-4" />
          </Button>
        </Tooltip>
      </div>

      <!-- Empty state -->
      <div v-if="tunnelsStore.activeTunnels.length === 0" class="cyber-card p-12 text-center">
        <div class="mx-auto mb-4 h-16 w-16 rounded-2xl bg-muted/50 flex items-center justify-center">
          <Zap class="h-8 w-8 text-muted-foreground" />
        </div>
        <p class="font-display font-medium text-lg">{{ t('dashboard.noTunnels') }}</p>
        <p class="mt-2 text-sm text-muted-foreground max-w-sm mx-auto">
          {{ t('dashboard.noTunnelsHint') }}
        </p>
      </div>

      <!-- Tunnel cards -->
      <TransitionGroup v-else name="list" tag="div" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="tunnel in tunnelsStore.activeTunnels"
          :key="tunnel.id"
          :class="[
            'group relative overflow-hidden rounded-xl border-2 transition-all duration-300 hover:scale-[1.02]',
            tunnel.type === 'http'
              ? 'border-type-http/30 hover:border-type-http/60 bg-gradient-to-br from-type-http/5 to-transparent'
              : tunnel.type === 'tcp'
                ? 'border-type-tcp/30 hover:border-type-tcp/60 bg-gradient-to-br from-type-tcp/5 to-transparent'
                : 'border-type-udp/30 hover:border-type-udp/60 bg-gradient-to-br from-type-udp/5 to-transparent'
          ]"
        >
          <!-- Top accent line -->
          <div
            :class="[
              'absolute top-0 left-0 right-0 h-1',
              tunnel.type === 'http' ? 'bg-type-http' : tunnel.type === 'tcp' ? 'bg-type-tcp' : 'bg-type-udp'
            ]"
          />

          <div class="p-5">
            <!-- Header -->
            <div class="flex items-start justify-between mb-4">
              <div class="flex items-center gap-3">
                <div
                  :class="[
                    'flex h-12 w-12 items-center justify-center rounded-xl transition-all duration-300 group-hover:scale-110 group-hover:shadow-lg',
                    tunnel.type === 'http'
                      ? 'bg-type-http/20 group-hover:shadow-type-http/20'
                      : tunnel.type === 'tcp'
                        ? 'bg-type-tcp/20 group-hover:shadow-type-tcp/20'
                        : 'bg-type-udp/20 group-hover:shadow-type-udp/20'
                  ]"
                >
                  <component
                    :is="getTunnelIcon(tunnel.type)"
                    :class="[
                      'h-6 w-6',
                      tunnel.type === 'http' ? 'text-type-http' : tunnel.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp'
                    ]"
                  />
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <span class="font-display font-semibold truncate">{{ tunnel.name }}</span>
                    <StatusIndicator status="connected" size="sm" />
                  </div>
                  <Badge :variant="tunnel.type" class="mt-1 font-mono text-[10px]">
                    {{ tunnel.type.toUpperCase() }}
                  </Badge>
                </div>
              </div>
              <Tooltip :content="t('dashboard.closeTunnel')">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 opacity-0 group-hover:opacity-100 transition-all text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                  @click="closeTunnel(tunnel.id)"
                >
                  <X class="h-4 w-4" />
                </Button>
              </Tooltip>
            </div>

            <!-- Port mapping -->
            <div class="flex items-center gap-2 mb-4 text-sm">
              <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-background/50 border border-border/50">
                <span class="text-muted-foreground text-xs">localhost:</span>
                <span class="font-mono font-bold">{{ tunnel.localPort }}</span>
              </div>
              <ArrowRight class="h-4 w-4 text-muted-foreground" />
              <div
                :class="[
                  'flex items-center gap-2 px-3 py-1.5 rounded-lg',
                  tunnel.type === 'http'
                    ? 'bg-type-http/10 border border-type-http/20'
                    : tunnel.type === 'tcp'
                      ? 'bg-type-tcp/10 border border-type-tcp/20'
                      : 'bg-type-udp/10 border border-type-udp/20'
                ]"
              >
                <component
                  :is="getTunnelIcon(tunnel.type)"
                  :class="[
                    'h-3.5 w-3.5',
                    tunnel.type === 'http' ? 'text-type-http' : tunnel.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp'
                  ]"
                />
                <span
                  :class="[
                    'font-mono font-bold text-xs',
                    tunnel.type === 'http' ? 'text-type-http' : tunnel.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp'
                  ]"
                >
                  {{ tunnel.type === 'http' ? 'public' : 'auto' }}
                </span>
              </div>
            </div>

            <!-- URL -->
            <div class="flex items-center gap-2 p-3 rounded-xl bg-background/80 border border-border/50">
              <code class="flex-1 truncate text-xs font-mono">
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
                      :class="['h-3.5 w-3.5', copiedId === tunnel.id && 'text-success']"
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

    <!-- Saved Bundles -->
    <div v-if="bundlesStore.bundles.length > 0">
      <div class="flex items-center gap-3 mb-4">
        <Boxes class="h-5 w-5 text-muted-foreground" />
        <h2 class="font-display font-semibold text-lg">{{ t('dashboard.savedBundles') }}</h2>
      </div>

      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <button
          v-for="bundle in bundlesStore.bundles.slice(0, 8)"
          :key="bundle.id"
          @click="bundlesStore.connectBundle(bundle.id)"
          :class="[
            'group flex items-center gap-3 p-4 rounded-xl border-2 transition-all duration-200 text-left hover:scale-[1.02]',
            bundle.type === 'http'
              ? 'border-type-http/20 hover:border-type-http/50 hover:bg-type-http/5'
              : bundle.type === 'tcp'
                ? 'border-type-tcp/20 hover:border-type-tcp/50 hover:bg-type-tcp/5'
                : 'border-type-udp/20 hover:border-type-udp/50 hover:bg-type-udp/5'
          ]"
        >
          <div
            :class="[
              'flex h-10 w-10 items-center justify-center rounded-xl transition-all duration-200 group-hover:scale-110',
              bundle.type === 'http'
                ? 'bg-type-http/15'
                : bundle.type === 'tcp'
                  ? 'bg-type-tcp/15'
                  : 'bg-type-udp/15'
            ]"
          >
            <component
              :is="getTunnelIcon(bundle.type)"
              :class="[
                'h-5 w-5',
                bundle.type === 'http' ? 'text-type-http' : bundle.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp'
              ]"
            />
          </div>
          <div class="flex-1 min-w-0">
            <p class="font-medium truncate">{{ bundle.name }}</p>
            <p class="text-xs text-muted-foreground font-mono">
              :{{ bundle.localPort }} → {{ bundle.subdomain || bundle.remotePort || 'auto' }}
            </p>
          </div>
          <Plus
            :class="[
              'h-5 w-5 opacity-0 group-hover:opacity-100 transition-all',
              bundle.type === 'http' ? 'text-type-http' : bundle.type === 'tcp' ? 'text-type-tcp' : 'text-type-udp'
            ]"
          />
        </button>

        <button
          v-if="bundlesStore.bundles.length > 8"
          @click="$router.push('/bundles')"
          class="flex items-center justify-center gap-2 p-4 rounded-xl border-2 border-dashed border-muted-foreground/20 hover:border-primary/40 hover:bg-primary/5 transition-all text-muted-foreground hover:text-foreground"
        >
          <Boxes class="h-5 w-5" />
          <span class="font-medium">+{{ bundlesStore.bundles.length - 8 }} {{ t('common.more') }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cyber-card {
  @apply relative rounded-xl border border-border/50 bg-card/80 backdrop-blur-sm;
}

.cyber-card::before {
  content: '';
  position: absolute;
  inset: 0;
  padding: 1px;
  border-radius: inherit;
  background: linear-gradient(
    135deg,
    hsl(var(--primary) / 0.2),
    transparent 40%,
    transparent 60%,
    hsl(var(--accent) / 0.2)
  );
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
}

.gradient-text {
  background: linear-gradient(135deg, hsl(var(--primary)) 0%, hsl(var(--accent)) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
</style>
