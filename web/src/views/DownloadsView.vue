<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Button from '@/components/ui/Button.vue'
import { downloadsApi, type Download } from '@/api/client'
import { siLinux, siApple } from 'simple-icons'

const { t } = useI18n()

// Platform icons
const platformIcons: Record<string, { path: string; hex: string }> = {
  Linux: { path: siLinux.path, hex: siLinux.hex },
  macOS: { path: siApple.path, hex: siApple.hex },
  Windows: {
    path: 'M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801',
    hex: '0078D4'
  },
}

function getPlatformIcon(os: string) {
  return platformIcons[os] || null
}

// State
const cliDownloads = ref<Download[]>([])
const guiDownloads = ref<Download[]>([])
const loading = ref(true)
const error = ref('')
const activeTab = ref<'gui' | 'cli'>('gui')
const copiedCommand = ref('')
const activeOsTab = ref<'linux' | 'macos' | 'windows'>('linux')

// Format file size
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Get architecture badge
function getArchBadge(download: Download): string {
  return download.arch.toUpperCase()
}

// Load downloads from API
async function loadDownloads() {
  loading.value = true
  error.value = ''
  try {
    const response = await downloadsApi.list()
    cliDownloads.value = response.data.cli || []
    guiDownloads.value = response.data.gui || []
    // Set default tab based on available downloads
    if (guiDownloads.value.length === 0 && cliDownloads.value.length > 0) {
      activeTab.value = 'cli'
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('downloads.failedToLoad')
  } finally {
    loading.value = false
  }
}

// Download client
function downloadClient(download: Download) {
  window.location.href = download.url
}

// Copy command to clipboard
async function copyCommand(command: string) {
  try {
    await navigator.clipboard.writeText(command)
    copiedCommand.value = command
    setTimeout(() => {
      copiedCommand.value = ''
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

// Quick start commands by OS
const serverHost = window.location.hostname

const quickStartCommands = computed(() => ({
  linux: {
    install: `curl -fsSL https://${serverHost}/install.sh | sh`,
    run: 'fxtunnel http 3000 --token YOUR_TOKEN'
  },
  macos: {
    install: `curl -fsSL https://${serverHost}/install.sh | sh`,
    run: 'fxtunnel http 3000 --token YOUR_TOKEN'
  },
  windows: {
    install: `curl -fsSL https://${serverHost}/install.sh | sh`,
    run: 'fxtunnel.exe http 3000 --token YOUR_TOKEN'
  }
}))

// Current downloads based on active tab
const currentDownloads = computed(() =>
  activeTab.value === 'gui' ? guiDownloads.value : cliDownloads.value
)

const hasDownloads = computed(() =>
  cliDownloads.value.length > 0 || guiDownloads.value.length > 0
)

// macOS GUI placeholder cards
const macGuiPlaceholders = [
  { os: 'macOS', arch: 'ARM64', label: 'Apple Silicon' },
  { os: 'macOS', arch: 'AMD64', label: 'Intel' },
]

onMounted(loadDownloads)
</script>

<template>
  <Layout>
    <div class="dl-root">
      <!-- ========== HERO ========== -->
      <div class="dl-hero">
        <div class="dl-hero-content">
          <div class="dl-hero-left">
            <div class="dl-status-badge">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
              <span>{{ (cliDownloads.length + guiDownloads.length) }} {{ t('downloads.download').toLowerCase() }}</span>
            </div>
            <h1 class="dl-title">{{ t('downloads.title') }}</h1>
            <p class="dl-subtitle">{{ t('downloads.subtitle') }}</p>
          </div>
          <div class="dl-hero-icon">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
          </div>
        </div>
        <div class="dl-hero-orb dl-hero-orb-1"></div>
        <div class="dl-hero-orb dl-hero-orb-2"></div>
      </div>

      <!-- ========== ERROR ========== -->
      <div v-if="error" class="dl-error">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        {{ error }}
      </div>

      <!-- ========== LOADING ========== -->
      <div v-if="loading" class="dl-loading">
        <div class="dl-loading-spinner"></div>
        <span>{{ t('common.loading') }}</span>
      </div>

      <template v-else-if="hasDownloads">
        <!-- ========== TABS ========== -->
        <div class="dl-tabs">
          <button
            v-if="guiDownloads.length > 0 || macGuiPlaceholders.length > 0"
            @click="activeTab = 'gui'"
            :class="['dl-tab', activeTab === 'gui' && 'dl-tab-active']"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
            {{ t('downloads.guiTitle') }}
          </button>
          <button
            v-if="cliDownloads.length > 0"
            @click="activeTab = 'cli'"
            :class="['dl-tab', activeTab === 'cli' && 'dl-tab-active']"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            {{ t('downloads.cliTitle') }}
          </button>
        </div>

        <!-- ========== SECTION HEADER ========== -->
        <div class="dl-section-header">
          <div class="dl-section-icon">
            <svg v-if="activeTab === 'gui'" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
          </div>
          <div>
            <h2 class="dl-section-title">
              {{ activeTab === 'gui' ? t('downloads.guiTitle') : t('downloads.cliTitle') }}
            </h2>
            <p class="dl-section-subtitle">
              {{ activeTab === 'gui' ? t('downloads.guiSubtitle') : t('downloads.cliSubtitle') }}
            </p>
          </div>
        </div>

        <!-- ========== DOWNLOADS GRID ========== -->
        <transition name="fade" mode="out-in">
          <div :key="activeTab" class="dl-grid">
            <!-- Real downloads -->
            <div
              v-for="download in currentDownloads"
              :key="download.platform"
              class="dl-card"
              @click="downloadClient(download)"
            >
              <div class="dl-card-inner">
                <div class="dl-card-platform-icon">
                  <svg
                    v-if="getPlatformIcon(download.os)"
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-8 w-8"
                    viewBox="0 0 24 24"
                    :fill="'#' + getPlatformIcon(download.os)!.hex"
                  >
                    <path :d="getPlatformIcon(download.os)!.path" />
                  </svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                </div>

                <div class="dl-card-info">
                  <h3 class="dl-card-os">{{ download.os }}</h3>
                  <div class="dl-card-badges">
                    <span class="dl-card-arch">{{ getArchBadge(download) }}</span>
                    <span class="dl-card-size">{{ formatSize(download.size) }}</span>
                  </div>
                </div>

                <Button class="dl-card-btn" @click.stop="downloadClient(download)">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                  {{ t('downloads.download') }}
                </Button>
              </div>
            </div>

            <!-- macOS GUI placeholder cards -->
            <template v-if="activeTab === 'gui'">
              <div
                v-for="ph in macGuiPlaceholders"
                :key="ph.label"
                class="dl-card dl-card-disabled"
              >
                <div class="dl-card-inner">
                  <!-- Coming soon ribbon -->
                  <div class="dl-card-ribbon">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
                    {{ t('downloads.macGuiComingSoon') }}
                  </div>

                  <div class="dl-card-platform-icon dl-card-platform-icon-dim">
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      class="h-8 w-8"
                      viewBox="0 0 24 24"
                      :fill="'#' + siApple.hex"
                      style="opacity: 0.35"
                    >
                      <path :d="siApple.path" />
                    </svg>
                  </div>

                  <div class="dl-card-info">
                    <h3 class="dl-card-os dl-card-os-dim">{{ ph.os }}</h3>
                    <div class="dl-card-badges">
                      <span class="dl-card-arch dl-card-arch-dim">{{ ph.arch }}</span>
                      <span class="dl-card-size">{{ ph.label }}</span>
                    </div>
                  </div>

                  <div class="dl-card-disabled-hint">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
                    <span>{{ t('downloads.macGuiComingSoonHint') }}</span>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </transition>
      </template>

      <!-- ========== NO DOWNLOADS ========== -->
      <div v-else-if="!loading" class="dl-empty">
        <div class="dl-empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
        </div>
        <p class="dl-empty-title">{{ t('downloads.noDownloads') }}</p>
        <p class="dl-empty-subtitle">{{ t('downloads.noDownloadsHint') }}</p>
      </div>

      <!-- ========== QUICK START ========== -->
      <div class="dl-quickstart">
        <div class="dl-quickstart-header">
          <div class="dl-quickstart-icon">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
          </div>
          <div>
            <h2 class="dl-quickstart-title">{{ t('downloads.quickStart') }}</h2>
            <p class="dl-quickstart-subtitle">{{ t('downloads.quickStartDesc') || 'Get started in seconds' }}</p>
          </div>
        </div>

        <!-- OS Tabs -->
        <div class="dl-os-tabs">
          <button
            @click="activeOsTab = 'linux'"
            :class="['dl-os-tab', activeOsTab === 'linux' && 'dl-os-tab-active']"
          >Linux</button>
          <button
            @click="activeOsTab = 'macos'"
            :class="['dl-os-tab', activeOsTab === 'macos' && 'dl-os-tab-active']"
          >macOS</button>
          <button
            @click="activeOsTab = 'windows'"
            :class="['dl-os-tab', activeOsTab === 'windows' && 'dl-os-tab-active']"
          >Windows</button>
        </div>

        <transition name="fade" mode="out-in">
          <div :key="activeOsTab" class="dl-steps">
            <!-- Step 1 -->
            <div class="dl-step">
              <span class="dl-step-num">1</span>
              <div class="dl-step-content">
                <p class="dl-step-label">{{ t('downloads.step1') }}</p>
                <div class="dl-code-block">
                  <code>{{ quickStartCommands[activeOsTab].install }}</code>
                  <button @click="copyCommand(quickStartCommands[activeOsTab].install)" class="dl-copy-btn">
                    <svg v-if="copiedCommand !== quickStartCommands[activeOsTab].install" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                    <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
                  </button>
                </div>
              </div>
            </div>

            <!-- Step 2 -->
            <div class="dl-step">
              <span class="dl-step-num">2</span>
              <div class="dl-step-content">
                <p class="dl-step-label">
                  <i18n-t keypath="downloads.step2" tag="span">
                    <template #link>
                      <router-link to="/tokens" class="dl-link">{{ t('downloads.step2TokenLink') }}</router-link>
                    </template>
                  </i18n-t>
                </p>
                <div class="dl-code-block">
                  <code>{{ quickStartCommands[activeOsTab].run }}</code>
                  <button @click="copyCommand(quickStartCommands[activeOsTab].run)" class="dl-copy-btn">
                    <svg v-if="copiedCommand !== quickStartCommands[activeOsTab].run" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                    <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
                  </button>
                </div>
              </div>
            </div>

            <!-- Token Hint -->
            <div class="dl-token-hint">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
              <p>
                <i18n-t keypath="downloads.tokenHint" tag="span">
                  <template #link>
                    <router-link to="/tokens" class="dl-link">{{ t('downloads.tokenHintLink') }}</router-link>
                  </template>
                </i18n-t>
              </p>
            </div>
          </div>
        </transition>
      </div>
    </div>
  </Layout>
</template>

<style scoped>
/* ============================================
   DOWNLOADS — CYBER COMMAND CENTER
   ============================================ */

.dl-root {
  @apply space-y-6;
}

/* ---- Hero ---- */
.dl-hero {
  @apply relative rounded-2xl overflow-hidden p-6 sm:p-8;
  background:
    radial-gradient(ellipse 60% 80% at 20% 0%, hsl(var(--type-tcp) / 0.12) 0%, transparent 60%),
    radial-gradient(ellipse 40% 60% at 90% 80%, hsl(var(--primary) / 0.08) 0%, transparent 50%),
    hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dl-hero-content {
  @apply relative z-10 flex flex-col md:flex-row items-center justify-between gap-6;
}

.dl-hero-left {
  @apply space-y-2 text-center md:text-left;
}

.dl-status-badge {
  @apply inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium;
  background: hsl(var(--type-tcp) / 0.1);
  border: 1px solid hsl(var(--type-tcp) / 0.2);
  color: hsl(var(--type-tcp));
}

.dl-title {
  @apply text-2xl sm:text-3xl font-bold tracking-tight font-display;
}

.dl-subtitle {
  @apply text-sm text-muted-foreground max-w-xl;
}

.dl-hero-icon {
  @apply flex items-center justify-center w-20 h-20 rounded-2xl;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
  color: hsl(var(--primary));
}

.dl-hero-orb {
  @apply absolute rounded-full pointer-events-none;
  filter: blur(80px);
}

.dl-hero-orb-1 {
  width: 200px;
  height: 200px;
  top: -60px;
  left: -40px;
  background: hsl(var(--type-tcp) / 0.15);
}

.dl-hero-orb-2 {
  width: 150px;
  height: 150px;
  bottom: -50px;
  right: -30px;
  background: hsl(var(--primary) / 0.1);
}

/* ---- Error ---- */
.dl-error {
  @apply flex items-center gap-2 p-4 rounded-xl text-sm;
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

/* ---- Loading ---- */
.dl-loading {
  @apply flex items-center justify-center gap-3 py-16 text-muted-foreground;
}

.dl-loading-spinner {
  @apply w-5 h-5 rounded-full border-2 border-current border-t-transparent animate-spin;
}

/* ---- Tabs ---- */
.dl-tabs {
  @apply flex items-center gap-2 p-1 rounded-xl w-fit;
  background: hsl(var(--muted) / 0.5);
}

.dl-tab {
  @apply flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg transition-all duration-200;
  color: hsl(var(--muted-foreground));
}

.dl-tab:hover {
  color: hsl(var(--foreground));
}

.dl-tab-active {
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  box-shadow: 0 1px 3px hsl(0 0% 0% / 0.1);
}

/* ---- Section Header ---- */
.dl-section-header {
  @apply flex items-center gap-3;
}

.dl-section-icon {
  @apply flex items-center justify-center w-10 h-10 rounded-xl;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
}

.dl-section-title {
  @apply text-xl font-bold font-display;
}

.dl-section-subtitle {
  @apply text-sm text-muted-foreground;
}

/* ---- Downloads Grid ---- */
.dl-grid {
  @apply grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4;
}

/* ---- Download Card ---- */
.dl-card {
  @apply relative rounded-xl overflow-hidden cursor-pointer transition-all duration-300;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dl-card:hover {
  border-color: hsl(var(--primary) / 0.3);
  box-shadow: 0 8px 30px hsl(var(--primary) / 0.08);
  transform: translateY(-2px);
}

.dl-card::before {
  content: '';
  @apply absolute inset-0 opacity-0 transition-opacity duration-300 pointer-events-none;
  background: linear-gradient(135deg, hsl(var(--primary) / 0.04) 0%, transparent 50%);
}

.dl-card:hover::before {
  opacity: 1;
}

.dl-card-inner {
  @apply flex flex-col items-center text-center p-6 space-y-4;
}

.dl-card-platform-icon {
  @apply w-14 h-14 rounded-2xl flex items-center justify-center;
  background: hsl(var(--muted) / 0.5);
}

.dl-card-platform-icon-dim {
  opacity: 0.5;
}

.dl-card-info {
  @apply space-y-2;
}

.dl-card-os {
  @apply font-bold text-lg;
}

.dl-card-os-dim {
  @apply text-muted-foreground;
}

.dl-card-badges {
  @apply flex items-center justify-center gap-2;
}

.dl-card-arch {
  @apply inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
}

.dl-card-arch-dim {
  background: hsl(var(--muted) / 0.5);
  color: hsl(var(--muted-foreground));
}

.dl-card-size {
  @apply text-sm text-muted-foreground;
}

.dl-card-btn {
  @apply w-full;
}

/* ---- Disabled Card ---- */
.dl-card-disabled {
  cursor: default;
  opacity: 0.65;
}

.dl-card-disabled:hover {
  transform: none;
  border-color: hsl(var(--border) / 0.6);
  box-shadow: none;
}

.dl-card-disabled::before {
  display: none;
}

.dl-card-ribbon {
  @apply absolute top-3 right-3 inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider;
  background: hsl(38 85% 55% / 0.12);
  color: hsl(38 85% 55%);
  border: 1px solid hsl(38 85% 55% / 0.2);
}

.dl-card-disabled-hint {
  @apply flex items-start gap-2 text-xs text-muted-foreground text-left w-full p-3 rounded-lg;
  background: hsl(var(--muted) / 0.3);
}

/* ---- Empty State ---- */
.dl-empty {
  @apply text-center py-12 space-y-3;
}

.dl-empty-icon {
  @apply mx-auto w-16 h-16 rounded-2xl flex items-center justify-center;
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
}

.dl-empty-title {
  @apply text-base font-semibold;
}

.dl-empty-subtitle {
  @apply text-sm text-muted-foreground;
}

/* ---- Quick Start ---- */
.dl-quickstart {
  @apply p-6 md:p-8 rounded-xl;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dl-quickstart-header {
  @apply flex items-center gap-3 mb-6;
}

.dl-quickstart-icon {
  @apply flex items-center justify-center w-10 h-10 rounded-xl;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
}

.dl-quickstart-title {
  @apply text-xl font-bold font-display;
}

.dl-quickstart-subtitle {
  @apply text-sm text-muted-foreground;
}

/* OS Tabs */
.dl-os-tabs {
  @apply flex items-center gap-1 p-1 rounded-lg w-fit mb-6;
  background: hsl(var(--muted) / 0.5);
}

.dl-os-tab {
  @apply px-4 py-2 text-sm font-medium rounded-md transition-all duration-200;
  color: hsl(var(--muted-foreground));
}

.dl-os-tab:hover {
  color: hsl(var(--foreground));
}

.dl-os-tab-active {
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  box-shadow: 0 1px 3px hsl(0 0% 0% / 0.1);
}

/* Steps */
.dl-steps {
  @apply space-y-6;
}

.dl-step {
  @apply flex items-start gap-3;
}

.dl-step-num {
  @apply flex-shrink-0 w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.dl-step-content {
  @apply flex-1 min-w-0 space-y-2;
}

.dl-step-label {
  @apply font-medium text-sm;
}

.dl-code-block {
  @apply relative px-4 py-3 rounded-xl font-mono text-sm;
  background: hsl(220 20% 6%);
  border: 1px solid hsl(220 15% 15%);
  color: hsl(210 20% 80%);
}

.dl-copy-btn {
  @apply absolute top-2 right-2 p-1.5 rounded-lg transition-all duration-150;
  color: hsl(var(--muted-foreground));
}

.dl-copy-btn:hover {
  background: hsl(220 20% 12%);
  color: hsl(var(--primary));
}

.dl-token-hint {
  @apply flex items-start gap-3 p-4 rounded-xl;
  background: hsl(var(--primary) / 0.05);
  border: 1px solid hsl(var(--primary) / 0.1);
}

.dl-token-hint svg {
  color: hsl(var(--primary));
  margin-top: 2px;
}

.dl-token-hint p {
  @apply text-sm text-muted-foreground;
}

.dl-link {
  color: hsl(var(--primary));
  font-weight: 600;
}

.dl-link:hover {
  text-decoration: underline;
}

/* ---- Transitions ---- */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
