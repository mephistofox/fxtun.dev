<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { tunnelsApi, profileApi, type Tunnel, type ProfileResponse } from '@/api/client'

const { t } = useI18n()
const router = useRouter()

const tunnels = ref<Tunnel[]>([])
const loading = ref(true)
const error = ref('')
const serverHost = window.location.hostname
const profile = ref<ProfileResponse | null>(null)

async function loadProfile() {
  try {
    const response = await profileApi.get()
    profile.value = response.data
  } catch {
    // Profile loading is non-critical, silently ignore
  }
}

function usagePercent(used: number, max: number): number {
  if (max <= 0) return 0
  return Math.min(100, Math.round((used / max) * 100))
}

function barColor(percent: number): string {
  if (percent >= 100) return 'bg-red-500'
  if (percent >= 80) return 'bg-orange-500'
  if (percent >= 50) return 'bg-yellow-500'
  return 'bg-primary'
}

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
    return `https://${tunnel.subdomain}.fxtun.dev`
  }
  if (tunnel.remote_port) {
    return `${tunnel.type}://fxtun.dev:${tunnel.remote_port}`
  }
  return '-'
}

const copiedCmd = ref('')

function copyLine(event: MouseEvent) {
  const el = (event.currentTarget as HTMLElement)
  const cmd = el.textContent?.replace(/^\$\s*/, '').trim() || ''
  if (!cmd) return
  navigator.clipboard.writeText(cmd)
  copiedCmd.value = cmd
  setTimeout(() => { copiedCmd.value = '' }, 1500)
}

onMounted(() => {
  loadTunnels()
  loadProfile()
})
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

      <div v-if="profile?.plan" class="bg-card/80 backdrop-blur border border-border/50 rounded-xl p-4">
        <div class="flex flex-col sm:flex-row sm:items-center gap-4 sm:gap-6">
          <div class="flex items-center gap-3 min-w-0">
            <span
              :class="[
                'px-3 py-1 text-xs font-semibold rounded-full whitespace-nowrap',
                profile.plan.slug === 'free'
                  ? 'bg-muted text-muted-foreground'
                  : 'bg-primary/10 text-primary'
              ]"
            >
              {{ profile.plan.name }}
            </span>
            <span class="text-sm text-muted-foreground truncate">
              {{ profile.user.display_name || profile.user.phone }}
            </span>
          </div>

          <div class="flex flex-wrap items-center gap-4 sm:gap-6 flex-1">
            <div class="flex flex-col gap-1">
              <span class="text-xs text-muted-foreground">{{ t('dashboard.plan.tunnels') }}</span>
              <div class="flex items-center gap-2">
                <div class="h-1.5 w-24 bg-muted rounded-full overflow-hidden">
                  <div
                    :class="['h-full rounded-full transition-all', barColor(usagePercent(profile.tunnel_count, profile.plan.max_tunnels))]"
                    :style="{ width: usagePercent(profile.tunnel_count, profile.plan.max_tunnels) + '%' }"
                  />
                </div>
                <span class="text-xs text-muted-foreground">{{ profile.tunnel_count }}/{{ profile.plan.max_tunnels }}</span>
              </div>
            </div>

            <div class="flex flex-col gap-1">
              <span class="text-xs text-muted-foreground">{{ t('dashboard.plan.domains') }}</span>
              <div class="flex items-center gap-2">
                <template v-if="profile.plan.max_domains > 0">
                  <div class="h-1.5 w-24 bg-muted rounded-full overflow-hidden">
                    <div
                      :class="['h-full rounded-full transition-all', barColor(usagePercent(profile.reserved_domains.length, profile.plan.max_domains))]"
                      :style="{ width: usagePercent(profile.reserved_domains.length, profile.plan.max_domains) + '%' }"
                    />
                  </div>
                  <span class="text-xs text-muted-foreground">{{ profile.reserved_domains.length }}/{{ profile.plan.max_domains }}</span>
                </template>
                <template v-else>
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  <span class="text-xs text-muted-foreground">&mdash;</span>
                </template>
              </div>
            </div>

            <div class="flex flex-col gap-1">
              <span class="text-xs text-muted-foreground">{{ t('dashboard.plan.tokens') }}</span>
              <div class="flex items-center gap-2">
                <div class="h-1.5 w-24 bg-muted rounded-full overflow-hidden">
                  <div
                    :class="['h-full rounded-full transition-all', barColor(usagePercent(profile.token_count, profile.plan.max_tokens))]"
                    :style="{ width: usagePercent(profile.token_count, profile.plan.max_tokens) + '%' }"
                  />
                </div>
                <span class="text-xs text-muted-foreground">{{ profile.token_count }}/{{ profile.plan.max_tokens }}</span>
              </div>
            </div>

            <span
              v-if="profile.plan.slug === 'free' && !profile.plan.inspector_enabled"
              class="inline-flex items-center gap-1 text-xs text-muted-foreground bg-muted/50 px-2 py-0.5 rounded"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              {{ t('dashboard.plan.inspector') }}
            </span>
          </div>

          <div class="flex-shrink-0">
            <Button
              v-if="profile.plan.slug === 'free'"
              size="sm"
              @click="router.push('/checkout')"
            >
              {{ t('dashboard.plan.upgrade') }}
            </Button>
            <Button
              v-else
              variant="outline"
              size="sm"
              @click="router.push('/profile')"
            >
              {{ t('dashboard.plan.manage') }}
            </Button>
          </div>
        </div>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="tunnels.length === 0" class="space-y-6">
        <!-- Hero hint -->
        <div class="text-center py-2">
          <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 border border-primary/20 mb-4">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
            </span>
            <span class="text-xs font-medium text-primary">{{ t('dashboard.cli.title') }}</span>
          </div>
          <p class="text-sm text-muted-foreground max-w-md mx-auto">{{ t('dashboard.cli.subtitle') }}</p>
        </div>

        <!-- Step 0: Install -->
        <div class="cli-section group">
          <div class="cli-section-header">
            <div class="cli-icon bg-cyan-500/15 text-cyan-400 group-hover:bg-cyan-500/25">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            </div>
            <div>
              <h4 class="text-sm font-semibold">{{ t('dashboard.cli.install') }}</h4>
              <p class="text-[11px] text-muted-foreground">{{ t('dashboard.cli.installDesc') }}</p>
            </div>
          </div>
          <div class="cli-terminal">
            <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> curl -fsSL https://{{ serverHost }}/install.sh | sh</div>
          </div>
        </div>

        <!-- Step 1: Auth + Init -->
        <div class="grid gap-4 md:grid-cols-2">
          <div class="cli-section group">
            <div class="cli-section-header">
              <div class="cli-icon bg-emerald-500/15 text-emerald-400 group-hover:bg-emerald-500/25">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>
              </div>
              <div>
                <h4 class="text-sm font-semibold">{{ t('dashboard.cli.auth') }}</h4>
                <p class="text-[11px] text-muted-foreground">{{ t('dashboard.cli.authDesc') }}</p>
              </div>
            </div>
            <div class="cli-terminal">
              <div class="cli-lines-row">
                <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel login</div>
                <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel login --token <span class="cli-placeholder">sk_...</span></div>
              </div>
            </div>
          </div>

          <div class="cli-section group">
            <div class="cli-section-header">
              <div class="cli-icon bg-amber-500/15 text-amber-400 group-hover:bg-amber-500/25">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              </div>
              <div>
                <h4 class="text-sm font-semibold">{{ t('dashboard.cli.authToken') }}</h4>
                <p class="text-[11px] text-muted-foreground">{{ t('dashboard.cli.authTokenDesc') }}</p>
              </div>
            </div>
            <div class="cli-terminal">
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel init</div>
            </div>
          </div>
        </div>

        <!-- Step 2: Tunnels -->
        <div class="grid gap-4 md:grid-cols-3">
          <div class="cli-section group">
            <div class="cli-section-header">
              <div class="cli-icon bg-[hsl(var(--type-http))]/15 text-[hsl(var(--type-http))] group-hover:bg-[hsl(var(--type-http))]/25">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              </div>
              <div>
                <h4 class="text-sm font-semibold">{{ t('dashboard.cli.httpTunnel') }}</h4>
                <p class="text-[11px] text-muted-foreground">Web, API, Webhooks</p>
              </div>
            </div>
            <div class="cli-terminal">
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel http 3000</div>
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel http 8080 --subdomain <span class="cli-arg">api</span></div>
            </div>
          </div>

          <div class="cli-section group">
            <div class="cli-section-header">
              <div class="cli-icon bg-[hsl(var(--type-tcp))]/15 text-[hsl(var(--type-tcp))] group-hover:bg-[hsl(var(--type-tcp))]/25">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
              </div>
              <div>
                <h4 class="text-sm font-semibold">{{ t('dashboard.cli.tcpTunnel') }}</h4>
                <p class="text-[11px] text-muted-foreground">SSH, PostgreSQL, Redis</p>
              </div>
            </div>
            <div class="cli-terminal">
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel tcp 22</div>
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel tcp 5432</div>
            </div>
          </div>

          <div class="cli-section group">
            <div class="cli-section-header">
              <div class="cli-icon bg-[hsl(var(--type-udp))]/15 text-[hsl(var(--type-udp))] group-hover:bg-[hsl(var(--type-udp))]/25">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4.9 19.1C1 15.2 1 8.8 4.9 4.9"/><path d="M7.8 16.2c-2.3-2.3-2.3-6.1 0-8.4"/><circle cx="12" cy="12" r="2"/><path d="M16.2 7.8c2.3 2.3 2.3 6.1 0 8.4"/><path d="M19.1 4.9C23 8.8 23 15.1 19.1 19"/></svg>
              </div>
              <div>
                <h4 class="text-sm font-semibold">{{ t('dashboard.cli.udpTunnel') }}</h4>
                <p class="text-[11px] text-muted-foreground">DNS, Games, VoIP</p>
              </div>
            </div>
            <div class="cli-terminal">
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel udp 53</div>
            </div>
          </div>
        </div>

        <!-- Step 3: Domains -->
        <div class="grid gap-4 md:grid-cols-2">
          <div class="cli-section group">
            <div class="cli-section-header">
              <div class="cli-icon bg-sky-500/15 text-sky-400 group-hover:bg-sky-500/25">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
              </div>
              <div>
                <h4 class="text-sm font-semibold">{{ t('dashboard.cli.domains') }}</h4>
                <p class="text-[11px] text-muted-foreground">{{ t('dashboard.cli.domainsDesc') }}</p>
              </div>
            </div>
            <div class="cli-terminal">
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel domains list</div>
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel domains reserve <span class="cli-arg">my-app</span></div>
            </div>
          </div>

          <div class="cli-section group">
            <div class="cli-section-header">
              <div class="cli-icon bg-violet-500/15 text-violet-400 group-hover:bg-violet-500/25">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
              </div>
              <div>
                <h4 class="text-sm font-semibold">{{ t('dashboard.cli.customDomains') }}</h4>
                <p class="text-[11px] text-muted-foreground">{{ t('dashboard.cli.customDomainsDesc') }}</p>
              </div>
            </div>
            <div class="cli-terminal">
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel domains custom verify <span class="cli-arg">example.com</span></div>
              <div class="cli-line" @click="copyLine"><span class="cli-prompt">$</span> fxtunnel domains custom add <span class="cli-arg">example.com</span> --target <span class="cli-arg">my-app</span></div>
            </div>
          </div>
        </div>

        <!-- Footer hint -->
        <p class="text-xs text-center text-muted-foreground pt-2">
          {{ t('dashboard.cli.hint') }}
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

<style scoped>
.cli-section {
  @apply relative rounded-xl border border-border/60 bg-card/80 overflow-hidden transition-all duration-300;
}
.cli-section:hover {
  border-color: hsl(var(--primary) / 0.3);
  box-shadow: 0 0 24px hsl(var(--primary) / 0.06);
}
.cli-section-header {
  @apply flex items-center gap-3 px-4 py-3;
}
.cli-icon {
  @apply flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center transition-all duration-300;
}
.cli-terminal {
  @apply px-4 pb-3 space-y-1;
}
.cli-lines-row {
  @apply flex flex-wrap gap-1;
}
.cli-lines-row .cli-line {
  flex: 1 1 auto;
  min-width: fit-content;
}
.cli-line {
  @apply font-mono text-[13px] leading-relaxed px-3 pr-8 py-1.5 rounded-lg bg-[hsl(220,20%,6%)] text-[hsl(210,20%,80%)] relative cursor-pointer transition-all duration-150 select-none;
}
.cli-line:hover {
  background: hsl(220, 20%, 10%);
  box-shadow: 0 0 0 1px hsl(var(--primary) / 0.2);
}
.cli-line:active {
  transform: scale(0.995);
}
.cli-line::after {
  content: '';
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 14px;
  height: 14px;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23666' stroke-width='2'%3E%3Crect x='9' y='9' width='13' height='13' rx='2' ry='2'/%3E%3Cpath d='M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1'/%3E%3C/svg%3E");
  background-size: contain;
  opacity: 0;
  transition: opacity 0.15s;
}
.cli-line:hover::after {
  opacity: 1;
}
.cli-prompt {
  color: hsl(var(--primary));
  margin-right: 0.5em;
  font-weight: 600;
}
.cli-arg {
  color: hsl(var(--accent));
}
.cli-placeholder {
  color: hsl(38, 85%, 55%);
  font-style: italic;
}
</style>
