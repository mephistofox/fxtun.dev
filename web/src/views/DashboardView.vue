<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Layout from '@/components/Layout.vue'
import Button from '@/components/ui/Button.vue'
import { tunnelsApi, profileApi, type Tunnel, type ProfileResponse } from '@/api/client'

const { t } = useI18n()
const router = useRouter()

const tunnels = ref<Tunnel[]>([])
const loading = ref(true)
const error = ref('')
const serverHost = window.location.hostname
const profile = ref<ProfileResponse | null>(null)
const copiedId = ref('')

async function loadProfile() {
  try {
    const response = await profileApi.get()
    profile.value = response.data
  } catch {
    // Profile loading is non-critical
  }
}

const userName = computed(() => {
  if (!profile.value) return ''
  return profile.value.user.display_name || profile.value.user.phone
})

function usagePercent(used: number, max: number): number {
  if (max <= 0) return 0
  return Math.min(100, Math.round((used / max) * 100))
}

function usageColor(percent: number): string {
  if (percent >= 100) return 'var(--destructive)'
  if (percent >= 80) return '38, 85%, 48%'
  if (percent >= 50) return '38, 85%, 55%'
  return 'var(--primary)'
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

function copyUrl(tunnel: Tunnel) {
  const url = getTunnelUrl(tunnel)
  if (url === '-') return
  navigator.clipboard.writeText(url)
  copiedId.value = tunnel.id
  setTimeout(() => { copiedId.value = '' }, 1500)
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

function tunnelTypeClass(type: string): string {
  switch (type) {
    case 'http': return 'tunnel-badge-http'
    case 'tcp': return 'tunnel-badge-tcp'
    case 'udp': return 'tunnel-badge-udp'
    default: return ''
  }
}

function tunnelGlowClass(type: string): string {
  switch (type) {
    case 'http': return 'tunnel-card-http'
    case 'tcp': return 'tunnel-card-tcp'
    case 'udp': return 'tunnel-card-udp'
    default: return ''
  }
}

const stats = computed(() => {
  if (!profile.value?.plan) return null
  const p = profile.value
  return {
    tunnels: { used: p.tunnel_count, max: p.plan!.max_tunnels },
    domains: { used: p.reserved_domains.length, max: p.plan!.max_domains },
    tokens: { used: p.token_count, max: p.plan!.max_tokens },
  }
})

onMounted(() => {
  loadProfile()
  loadTunnels()
})
</script>

<template>
  <Layout>
    <div class="dashboard-root">
      <!-- ========== HERO HEADER ========== -->
      <div class="dash-hero">
        <div class="dash-hero-content">
          <div class="dash-hero-left">
            <div class="dash-status-badge">
              <span class="dash-status-dot">
                <span class="dash-status-dot-ping"></span>
              </span>
              <span>{{ t('dashboard.statusOnline') }}</span>
            </div>
            <h1 class="dash-title">
              {{ userName ? t('dashboard.welcome') : t('dashboard.welcomeAnon') }}<template v-if="userName">,
              <span class="gradient-text">{{ userName }}</span></template>
            </h1>
            <p class="dash-subtitle">{{ t('dashboard.subtitle') }}</p>
          </div>
          <div class="dash-hero-right">
            <Button @click="loadTunnels" :loading="loading" variant="outline" class="dash-refresh-btn">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
              {{ t('common.refresh') }}
            </Button>
          </div>
        </div>

        <!-- Decorative glow orbs -->
        <div class="dash-hero-orb dash-hero-orb-1"></div>
        <div class="dash-hero-orb dash-hero-orb-2"></div>
      </div>

      <!-- ========== ERROR ========== -->
      <div v-if="error" class="dash-error">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        {{ error }}
      </div>

      <!-- ========== PLAN + STATS ========== -->
      <div v-if="stats" class="dash-stats-grid">
        <!-- Plan card -->
        <div class="dash-plan-card">
          <div class="dash-plan-header">
            <span :class="['dash-plan-badge', profile?.plan?.slug === 'free' ? 'dash-plan-badge-free' : 'dash-plan-badge-paid']">
              {{ profile?.plan?.name }}
            </span>
            <span class="dash-plan-label">{{ profile?.plan?.slug === 'free' ? t('dashboard.plan.freePlan') : t('dashboard.plan.currentPlan') }}</span>
          </div>
          <div class="dash-plan-cta">
            <Button
              v-if="profile?.plan?.slug === 'free'"
              size="sm"
              @click="router.push('/checkout')"
              class="dash-upgrade-btn"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
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

        <!-- Stats cards -->
        <div class="dash-stat-card">
          <div class="dash-stat-icon dash-stat-icon-tunnels">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="16" y="16" width="6" height="6" rx="1"/><rect x="2" y="16" width="6" height="6" rx="1"/><rect x="9" y="2" width="6" height="6" rx="1"/><path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3"/><path d="M12 12V8"/></svg>
          </div>
          <div class="dash-stat-info">
            <span class="dash-stat-label">{{ t('dashboard.stats.tunnels') }}</span>
            <div class="dash-stat-value-row">
              <span class="dash-stat-value">{{ stats.tunnels.used }}</span>
              <span class="dash-stat-max">/ {{ stats.tunnels.max }}</span>
            </div>
          </div>
          <div class="dash-stat-bar">
            <div
              class="dash-stat-bar-fill"
              :style="{
                width: usagePercent(stats.tunnels.used, stats.tunnels.max) + '%',
                background: `hsl(${usageColor(usagePercent(stats.tunnels.used, stats.tunnels.max))})`
              }"
            ></div>
          </div>
        </div>

        <div class="dash-stat-card">
          <div class="dash-stat-icon dash-stat-icon-domains">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </div>
          <div class="dash-stat-info">
            <span class="dash-stat-label">{{ t('dashboard.stats.domains') }}</span>
            <div class="dash-stat-value-row">
              <template v-if="stats.domains.max > 0">
                <span class="dash-stat-value">{{ stats.domains.used }}</span>
                <span class="dash-stat-max">/ {{ stats.domains.max }}</span>
              </template>
              <template v-else>
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                <span class="dash-stat-max">&mdash;</span>
              </template>
            </div>
          </div>
          <div v-if="stats.domains.max > 0" class="dash-stat-bar">
            <div
              class="dash-stat-bar-fill"
              :style="{
                width: usagePercent(stats.domains.used, stats.domains.max) + '%',
                background: `hsl(${usageColor(usagePercent(stats.domains.used, stats.domains.max))})`
              }"
            ></div>
          </div>
        </div>

        <div class="dash-stat-card">
          <div class="dash-stat-icon dash-stat-icon-tokens">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
          </div>
          <div class="dash-stat-info">
            <span class="dash-stat-label">{{ t('dashboard.stats.tokens') }}</span>
            <div class="dash-stat-value-row">
              <span class="dash-stat-value">{{ stats.tokens.used }}</span>
              <span class="dash-stat-max">/ {{ stats.tokens.max }}</span>
            </div>
          </div>
          <div class="dash-stat-bar">
            <div
              class="dash-stat-bar-fill"
              :style="{
                width: usagePercent(stats.tokens.used, stats.tokens.max) + '%',
                background: `hsl(${usageColor(usagePercent(stats.tokens.used, stats.tokens.max))})`
              }"
            ></div>
          </div>
        </div>
      </div>

      <!-- ========== LOADING ========== -->
      <div v-if="loading" class="dash-loading">
        <div class="dash-loading-spinner"></div>
        <span>{{ t('common.loading') }}</span>
      </div>

      <!-- ========== ACTIVE TUNNELS ========== -->
      <template v-else-if="tunnels.length > 0">
        <div class="dash-section-header">
          <h2 class="dash-section-title">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><circle cx="12" cy="20" r="1"/></svg>
            {{ t('dashboard.tunnels.title') }}
          </h2>
          <span class="dash-tunnel-count">{{ tunnels.length }}</span>
        </div>

        <div class="dash-tunnels-grid">
          <div
            v-for="tunnel in tunnels"
            :key="tunnel.id"
            :class="['dash-tunnel-card', tunnelGlowClass(tunnel.type)]"
          >
            <div class="dash-tunnel-top">
              <div class="dash-tunnel-meta">
                <span :class="['dash-tunnel-badge', tunnelTypeClass(tunnel.type)]">
                  {{ tunnel.type.toUpperCase() }}
                </span>
                <span class="dash-tunnel-name">{{ tunnel.name || t('dashboard.unnamed') }}</span>
              </div>
              <div class="dash-tunnel-actions">
                <router-link
                  v-if="tunnel.type === 'http'"
                  :to="`/inspect/${tunnel.id}`"
                  class="dash-tunnel-btn dash-tunnel-btn-inspect"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
                  <span>{{ t('dashboard.inspect') }}</span>
                </router-link>
                <button
                  @click="closeTunnel(tunnel.id)"
                  class="dash-tunnel-btn dash-tunnel-btn-close"
                  :title="t('dashboard.closeTunnel')"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
                </button>
              </div>
            </div>

            <div class="dash-tunnel-url-row">
              <a
                :href="getTunnelUrl(tunnel)"
                target="_blank"
                class="dash-tunnel-url"
              >
                {{ getTunnelUrl(tunnel) }}
              </a>
              <button
                @click="copyUrl(tunnel)"
                class="dash-tunnel-copy"
                :title="t('dashboard.copyUrl')"
              >
                <svg v-if="copiedId !== tunnel.id" xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
              </button>
            </div>

            <div class="dash-tunnel-footer">
              <span class="dash-tunnel-port">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/></svg>
                localhost:{{ tunnel.local_port }}
              </span>
              <span class="dash-tunnel-status">
                <span class="dash-tunnel-status-dot"></span>
                online
              </span>
            </div>
          </div>
        </div>
      </template>

      <!-- ========== EMPTY STATE: CLI Quick Start ========== -->
      <template v-else>
        <div class="dash-empty">
          <!-- Header -->
          <div class="dash-empty-header">
            <div class="dash-empty-icon">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            </div>
            <h2 class="dash-empty-title">{{ t('dashboard.cli.title') }}</h2>
            <p class="dash-empty-subtitle">{{ t('dashboard.cli.subtitle') }}</p>
          </div>

          <!-- Steps -->
          <div class="dash-steps">
            <!-- Step 1: Install -->
            <div class="dash-step">
              <div class="dash-step-number">1</div>
              <div class="dash-step-content">
                <div class="dash-step-header">
                  <h3>{{ t('dashboard.cli.step1') }}</h3>
                  <p>{{ t('dashboard.cli.step1Desc') }}</p>
                </div>
                <div class="dash-terminal">
                  <div class="dash-terminal-header">
                    <span class="dash-terminal-dot" style="background: #ff5f56"></span>
                    <span class="dash-terminal-dot" style="background: #ffbd2e"></span>
                    <span class="dash-terminal-dot" style="background: #27c93f"></span>
                  </div>
                  <div class="dash-terminal-body">
                    <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> curl -fsSL https://{{ serverHost }}/install.sh | sh</div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Step 2: Auth -->
            <div class="dash-step">
              <div class="dash-step-number">2</div>
              <div class="dash-step-content">
                <div class="dash-step-header">
                  <h3>{{ t('dashboard.cli.step2') }}</h3>
                  <p>{{ t('dashboard.cli.step2Desc') }}</p>
                </div>
                <div class="dash-terminal">
                  <div class="dash-terminal-header">
                    <span class="dash-terminal-dot" style="background: #ff5f56"></span>
                    <span class="dash-terminal-dot" style="background: #ffbd2e"></span>
                    <span class="dash-terminal-dot" style="background: #27c93f"></span>
                  </div>
                  <div class="dash-terminal-body">
                    <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> fxtunnel login</div>
                    <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> fxtunnel login --token <span class="dash-token">sk_...</span></div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Step 3: Tunnel -->
            <div class="dash-step">
              <div class="dash-step-number">3</div>
              <div class="dash-step-content">
                <div class="dash-step-header">
                  <h3>{{ t('dashboard.cli.step3') }}</h3>
                  <p>{{ t('dashboard.cli.step3Desc') }}</p>
                </div>
                <div class="dash-protocol-cards">
                  <div class="dash-protocol-card dash-protocol-http">
                    <div class="dash-protocol-top">
                      <span class="dash-tunnel-badge tunnel-badge-http">HTTP</span>
                      <span class="dash-protocol-desc">{{ t('dashboard.cli.httpDesc') }}</span>
                    </div>
                    <div class="dash-terminal dash-terminal-compact">
                      <div class="dash-terminal-body">
                        <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> fxtunnel http 3000</div>
                        <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> fxtunnel http 8080 --subdomain <span class="dash-arg">api</span></div>
                      </div>
                    </div>
                  </div>

                  <div class="dash-protocol-card dash-protocol-tcp">
                    <div class="dash-protocol-top">
                      <span class="dash-tunnel-badge tunnel-badge-tcp">TCP</span>
                      <span class="dash-protocol-desc">{{ t('dashboard.cli.tcpDesc') }}</span>
                    </div>
                    <div class="dash-terminal dash-terminal-compact">
                      <div class="dash-terminal-body">
                        <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> fxtunnel tcp 22</div>
                        <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> fxtunnel tcp 5432</div>
                      </div>
                    </div>
                  </div>

                  <div class="dash-protocol-card dash-protocol-udp">
                    <div class="dash-protocol-top">
                      <span class="dash-tunnel-badge tunnel-badge-udp">UDP</span>
                      <span class="dash-protocol-desc">{{ t('dashboard.cli.udpDesc') }}</span>
                    </div>
                    <div class="dash-terminal dash-terminal-compact">
                      <div class="dash-terminal-body">
                        <div class="dash-cmd" @click="copyLine"><span class="dash-prompt">$</span> fxtunnel udp 53</div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Footer hint -->
          <p class="dash-empty-hint">{{ t('dashboard.cli.hint') }}</p>
        </div>
      </template>

      <!-- ========== QUICK ACTIONS ========== -->
      <div class="dash-quick-grid">
        <router-link to="/domains" class="dash-quick-card">
          <div class="dash-quick-icon dash-quick-icon-domains">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </div>
          <div>
            <span class="dash-quick-label">{{ t('dashboard.quickActions.domains') }}</span>
            <span class="dash-quick-desc">{{ t('dashboard.quickActions.domainsDesc') }}</span>
          </div>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 dash-quick-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
        </router-link>

        <router-link to="/tokens" class="dash-quick-card">
          <div class="dash-quick-icon dash-quick-icon-tokens">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
          </div>
          <div>
            <span class="dash-quick-label">{{ t('dashboard.quickActions.tokens') }}</span>
            <span class="dash-quick-desc">{{ t('dashboard.quickActions.tokensDesc') }}</span>
          </div>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 dash-quick-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
        </router-link>

        <router-link to="/downloads" class="dash-quick-card">
          <div class="dash-quick-icon dash-quick-icon-downloads">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
          </div>
          <div>
            <span class="dash-quick-label">{{ t('dashboard.quickActions.downloads') }}</span>
            <span class="dash-quick-desc">{{ t('dashboard.quickActions.downloadsDesc') }}</span>
          </div>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 dash-quick-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
        </router-link>

        <router-link to="/profile" class="dash-quick-card">
          <div class="dash-quick-icon dash-quick-icon-profile">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          </div>
          <div>
            <span class="dash-quick-label">{{ t('dashboard.quickActions.profile') }}</span>
            <span class="dash-quick-desc">{{ t('dashboard.quickActions.profileDesc') }}</span>
          </div>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 dash-quick-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
        </router-link>
      </div>
    </div>
  </Layout>
</template>

<style scoped>
/* ============================================
   DASHBOARD — CYBER COMMAND CENTER
   ============================================ */

.dashboard-root {
  @apply space-y-6;
}

/* ---- Hero ---- */
.dash-hero {
  @apply relative rounded-2xl overflow-hidden p-6 sm:p-8;
  background:
    radial-gradient(ellipse 60% 80% at 20% 0%, hsl(var(--primary) / 0.12) 0%, transparent 60%),
    radial-gradient(ellipse 40% 60% at 90% 80%, hsl(var(--accent) / 0.08) 0%, transparent 50%),
    hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dash-hero-content {
  @apply relative z-10 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4;
}

.dash-hero-left {
  @apply space-y-2;
}

.dash-status-badge {
  @apply inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
  color: hsl(var(--primary));
}

.dash-status-dot {
  @apply relative flex h-2 w-2;
}

.dash-status-dot-ping {
  @apply absolute inline-flex h-full w-full rounded-full opacity-75;
  background: hsl(var(--primary));
  animation: ping 2s cubic-bezier(0, 0, 0.2, 1) infinite;
}

.dash-status-dot::after {
  content: '';
  @apply relative inline-flex rounded-full h-2 w-2;
  background: hsl(var(--primary));
}

@keyframes ping {
  75%, 100% {
    transform: scale(2);
    opacity: 0;
  }
}

.dash-title {
  @apply text-2xl sm:text-3xl font-bold tracking-tight font-display;
}

.dash-subtitle {
  @apply text-sm text-muted-foreground;
}

.dash-hero-right {
  @apply flex-shrink-0;
}

.dash-refresh-btn {
  @apply flex items-center;
}

.dash-hero-orb {
  @apply absolute rounded-full pointer-events-none;
  filter: blur(80px);
}

.dash-hero-orb-1 {
  width: 200px;
  height: 200px;
  top: -60px;
  left: -40px;
  background: hsl(var(--primary) / 0.15);
}

.dash-hero-orb-2 {
  width: 150px;
  height: 150px;
  bottom: -50px;
  right: -30px;
  background: hsl(var(--accent) / 0.1);
}

/* ---- Error ---- */
.dash-error {
  @apply flex items-center gap-2 p-4 rounded-xl text-sm;
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

/* ---- Stats Grid ---- */
.dash-stats-grid {
  @apply grid gap-3;
  grid-template-columns: 1fr;
}

@media (min-width: 640px) {
  .dash-stats-grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (min-width: 1024px) {
  .dash-stats-grid {
    grid-template-columns: auto 1fr 1fr 1fr;
  }
}

.dash-plan-card {
  @apply flex flex-col justify-between gap-3 p-4 rounded-xl;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dash-plan-header {
  @apply flex items-center gap-3;
}

.dash-plan-badge {
  @apply px-3 py-1 text-xs font-bold rounded-full uppercase tracking-wider;
}

.dash-plan-badge-free {
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
}

.dash-plan-badge-paid {
  background: hsl(var(--primary) / 0.15);
  color: hsl(var(--primary));
  border: 1px solid hsl(var(--primary) / 0.3);
}

.dash-plan-label {
  @apply text-xs text-muted-foreground;
}

.dash-plan-cta {
  @apply flex;
}

.dash-upgrade-btn {
  box-shadow: 0 0 20px hsl(var(--primary) / 0.25);
}

/* Stat card */
.dash-stat-card {
  @apply flex flex-col gap-3 p-4 rounded-xl transition-all duration-300;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dash-stat-card:hover {
  border-color: hsl(var(--primary) / 0.3);
  box-shadow: 0 0 20px hsl(var(--primary) / 0.06);
}

.dash-stat-icon {
  @apply w-10 h-10 rounded-xl flex items-center justify-center;
}

.dash-stat-icon-tunnels {
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
}

.dash-stat-icon-domains {
  background: hsl(var(--type-http) / 0.12);
  color: hsl(var(--type-http));
}

.dash-stat-icon-tokens {
  background: hsl(var(--accent) / 0.12);
  color: hsl(var(--accent));
}

.dash-stat-info {
  @apply flex flex-col gap-0.5;
}

.dash-stat-label {
  @apply text-xs text-muted-foreground font-medium;
}

.dash-stat-value-row {
  @apply flex items-baseline gap-1;
}

.dash-stat-value {
  @apply text-2xl font-bold font-display;
}

.dash-stat-max {
  @apply text-sm text-muted-foreground;
}

.dash-stat-bar {
  @apply h-1.5 rounded-full overflow-hidden;
  background: hsl(var(--muted));
}

.dash-stat-bar-fill {
  @apply h-full rounded-full transition-all duration-500;
}

/* ---- Loading ---- */
.dash-loading {
  @apply flex items-center justify-center gap-3 py-16 text-muted-foreground;
}

.dash-loading-spinner {
  @apply w-5 h-5 rounded-full border-2 border-current border-t-transparent animate-spin;
}

/* ---- Tunnels Section ---- */
.dash-section-header {
  @apply flex items-center justify-between;
}

.dash-section-title {
  @apply flex items-center gap-2 text-lg font-bold font-display;
}

.dash-tunnel-count {
  @apply px-2.5 py-0.5 rounded-full text-xs font-bold;
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
}

.dash-tunnels-grid {
  @apply grid gap-4 md:grid-cols-2 lg:grid-cols-3;
}

.dash-tunnel-card {
  @apply relative rounded-xl p-4 space-y-3 transition-all duration-300 overflow-hidden;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dash-tunnel-card:hover {
  transform: translateY(-2px);
}

.dash-tunnel-card::before {
  content: '';
  @apply absolute inset-0 opacity-0 transition-opacity duration-300 pointer-events-none;
}

.dash-tunnel-card:hover::before {
  opacity: 1;
}

.tunnel-card-http { border-color: hsl(var(--type-http) / 0.2); }
.tunnel-card-http:hover { border-color: hsl(var(--type-http) / 0.4); box-shadow: 0 8px 30px hsl(var(--type-http) / 0.1); }
.tunnel-card-http::before { background: linear-gradient(135deg, hsl(var(--type-http) / 0.05) 0%, transparent 50%); }

.tunnel-card-tcp { border-color: hsl(var(--type-tcp) / 0.2); }
.tunnel-card-tcp:hover { border-color: hsl(var(--type-tcp) / 0.4); box-shadow: 0 8px 30px hsl(var(--type-tcp) / 0.1); }
.tunnel-card-tcp::before { background: linear-gradient(135deg, hsl(var(--type-tcp) / 0.05) 0%, transparent 50%); }

.tunnel-card-udp { border-color: hsl(var(--type-udp) / 0.2); }
.tunnel-card-udp:hover { border-color: hsl(var(--type-udp) / 0.4); box-shadow: 0 8px 30px hsl(var(--type-udp) / 0.1); }
.tunnel-card-udp::before { background: linear-gradient(135deg, hsl(var(--type-udp) / 0.05) 0%, transparent 50%); }

.dash-tunnel-top {
  @apply flex items-center justify-between;
}

.dash-tunnel-meta {
  @apply flex items-center gap-2;
}

.dash-tunnel-badge {
  @apply px-2.5 py-0.5 text-[10px] font-bold rounded-full uppercase tracking-widest;
}

.tunnel-badge-http {
  background: hsl(var(--type-http) / 0.15);
  color: hsl(var(--type-http));
  border: 1px solid hsl(var(--type-http) / 0.3);
}

.tunnel-badge-tcp {
  background: hsl(var(--type-tcp) / 0.15);
  color: hsl(var(--type-tcp));
  border: 1px solid hsl(var(--type-tcp) / 0.3);
}

.tunnel-badge-udp {
  background: hsl(var(--type-udp) / 0.15);
  color: hsl(var(--type-udp));
  border: 1px solid hsl(var(--type-udp) / 0.3);
}

.dash-tunnel-name {
  @apply text-sm font-semibold truncate;
}

.dash-tunnel-actions {
  @apply flex items-center gap-1.5;
}

/* Shared button base */
.dash-tunnel-btn {
  @apply inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold uppercase tracking-wider transition-all duration-200 no-underline select-none;
  border: 1px solid transparent;
}

/* Inspect button */
.dash-tunnel-btn-inspect {
  color: hsl(var(--type-tcp));
  background: hsl(var(--type-tcp) / 0.08);
  border-color: hsl(var(--type-tcp) / 0.15);
}

.dash-tunnel-btn-inspect:hover {
  background: hsl(var(--type-tcp) / 0.15);
  border-color: hsl(var(--type-tcp) / 0.3);
  box-shadow: 0 0 12px hsl(var(--type-tcp) / 0.15);
  transform: translateY(-1px);
}

.dash-tunnel-btn-inspect:active {
  transform: translateY(0);
}

/* Close button — icon-only, round */
.dash-tunnel-btn-close {
  @apply p-1.5;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted) / 0.5);
  border-color: transparent;
  border-radius: 9999px;
}

.dash-tunnel-btn-close:hover {
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 0.12);
  border-color: hsl(var(--destructive) / 0.2);
  box-shadow: 0 0 12px hsl(var(--destructive) / 0.12);
  transform: translateY(-1px);
}

.dash-tunnel-btn-close:active {
  transform: translateY(0);
}

.dash-tunnel-url-row {
  @apply flex items-center gap-2;
}

.dash-tunnel-url {
  @apply text-sm font-mono truncate transition-colors;
  color: hsl(var(--primary));
}

.dash-tunnel-url:hover {
  text-decoration: underline;
}

.dash-tunnel-copy {
  @apply flex-shrink-0 p-1 rounded transition-all duration-150 text-muted-foreground;
}

.dash-tunnel-copy:hover {
  color: hsl(var(--primary));
  background: hsl(var(--primary) / 0.1);
}

.dash-tunnel-footer {
  @apply flex items-center justify-between text-xs text-muted-foreground;
}

.dash-tunnel-port {
  @apply flex items-center gap-1 font-mono;
}

.dash-tunnel-status {
  @apply flex items-center gap-1.5;
}

.dash-tunnel-status-dot {
  @apply w-1.5 h-1.5 rounded-full;
  background: hsl(160 84% 45%);
  box-shadow: 0 0 6px hsl(160 84% 45% / 0.5);
}

/* ---- Empty State ---- */
.dash-empty {
  @apply space-y-8;
}

.dash-empty-header {
  @apply text-center space-y-3;
}

.dash-empty-icon {
  @apply mx-auto w-16 h-16 rounded-2xl flex items-center justify-center;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
  border: 1px solid hsl(var(--primary) / 0.2);
}

.dash-empty-title {
  @apply text-xl font-bold font-display;
}

.dash-empty-subtitle {
  @apply text-sm text-muted-foreground max-w-md mx-auto;
}

/* Steps */
.dash-steps {
  @apply space-y-6;
}

.dash-step {
  @apply flex gap-4;
}

.dash-step-number {
  @apply flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold;
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
  border: 1px solid hsl(var(--primary) / 0.2);
}

.dash-step-content {
  @apply flex-1 min-w-0 space-y-3;
}

.dash-step-header h3 {
  @apply text-sm font-semibold;
}

.dash-step-header p {
  @apply text-xs text-muted-foreground;
}

/* Terminal blocks */
.dash-terminal {
  @apply rounded-xl overflow-hidden;
  background: hsl(220 20% 6%);
  border: 1px solid hsl(220 15% 15%);
}

.dash-terminal-header {
  @apply flex items-center gap-1.5 px-3 py-2;
  background: hsl(220 20% 8%);
  border-bottom: 1px solid hsl(220 15% 15%);
}

.dash-terminal-dot {
  @apply w-2.5 h-2.5 rounded-full;
}

.dash-terminal-body {
  @apply p-3 space-y-1;
}

.dash-terminal-compact .dash-terminal-body {
  @apply p-2.5 space-y-0.5;
}

.dash-cmd {
  @apply font-mono text-[13px] leading-relaxed px-2.5 py-1 rounded-lg relative cursor-pointer transition-all duration-150 select-none;
  color: hsl(210 20% 80%);
}

.dash-cmd:hover {
  background: hsl(220 20% 10%);
  box-shadow: 0 0 0 1px hsl(75 100% 50% / 0.15);
}

.dash-cmd:active {
  transform: scale(0.995);
}

.dash-cmd::after {
  content: '';
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 12px;
  height: 12px;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23555' stroke-width='2'%3E%3Crect x='9' y='9' width='13' height='13' rx='2' ry='2'/%3E%3Cpath d='M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1'/%3E%3C/svg%3E");
  background-size: contain;
  opacity: 0;
  transition: opacity 0.15s;
}

.dash-cmd:hover::after {
  opacity: 1;
}

.dash-prompt {
  color: hsl(75 100% 50%);
  margin-right: 0.5em;
  font-weight: 600;
}

.dash-arg {
  color: hsl(280 100% 65%);
}

.dash-token {
  color: hsl(38 85% 55%);
  font-style: italic;
}

/* Protocol cards in step 3 */
.dash-protocol-cards {
  @apply grid gap-3 sm:grid-cols-3;
}

.dash-protocol-card {
  @apply rounded-xl overflow-hidden transition-all duration-300;
  border: 1px solid hsl(var(--border) / 0.5);
  background: hsl(var(--card));
}

.dash-protocol-card:hover {
  transform: translateY(-2px);
}

.dash-protocol-http:hover { border-color: hsl(var(--type-http) / 0.4); box-shadow: 0 4px 20px hsl(var(--type-http) / 0.1); }
.dash-protocol-tcp:hover { border-color: hsl(var(--type-tcp) / 0.4); box-shadow: 0 4px 20px hsl(var(--type-tcp) / 0.1); }
.dash-protocol-udp:hover { border-color: hsl(var(--type-udp) / 0.4); box-shadow: 0 4px 20px hsl(var(--type-udp) / 0.1); }

.dash-protocol-top {
  @apply flex items-center gap-2 px-3 py-2.5;
}

.dash-protocol-desc {
  @apply text-[11px] text-muted-foreground;
}

.dash-empty-hint {
  @apply text-xs text-center text-muted-foreground pt-2;
}

/* ---- Quick Actions ---- */
.dash-quick-grid {
  @apply grid gap-3 sm:grid-cols-2 lg:grid-cols-4;
}

.dash-quick-card {
  @apply flex items-center gap-3 p-3.5 rounded-xl transition-all duration-300 no-underline;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
  color: hsl(var(--foreground));
}

.dash-quick-card:hover {
  border-color: hsl(var(--primary) / 0.3);
  box-shadow: 0 4px 20px hsl(var(--primary) / 0.06);
  transform: translateY(-1px);
}

.dash-quick-icon {
  @apply flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center transition-all duration-300;
}

.dash-quick-icon-domains {
  background: hsl(var(--type-http) / 0.12);
  color: hsl(var(--type-http));
}

.dash-quick-icon-tokens {
  background: hsl(var(--accent) / 0.12);
  color: hsl(var(--accent));
}

.dash-quick-icon-downloads {
  background: hsl(var(--type-tcp) / 0.12);
  color: hsl(var(--type-tcp));
}

.dash-quick-icon-profile {
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
}

.dash-quick-card:hover .dash-quick-icon {
  transform: scale(1.1);
}

.dash-quick-label {
  @apply block text-sm font-semibold;
}

.dash-quick-desc {
  @apply block text-xs text-muted-foreground;
}

.dash-quick-arrow {
  @apply ml-auto flex-shrink-0 text-muted-foreground transition-transform duration-300;
}

.dash-quick-card:hover .dash-quick-arrow {
  transform: translateX(3px);
  color: hsl(var(--primary));
}
</style>
