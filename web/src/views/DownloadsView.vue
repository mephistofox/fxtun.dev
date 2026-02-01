<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
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

onMounted(loadDownloads)
</script>

<template>
  <Layout>
    <div class="space-y-8">
      <!-- Hero Section -->
      <div class="hero-gradient rounded-2xl p-8 md:p-12 relative z-10">
        <div class="flex flex-col md:flex-row items-center justify-between gap-6">
          <div class="text-center md:text-left">
            <h1 class="text-3xl md:text-4xl font-bold mb-3">{{ t('downloads.title') }}</h1>
            <p class="text-muted-foreground text-lg max-w-xl">{{ t('downloads.subtitle') }}</p>
          </div>
          <div class="flex items-center justify-center w-24 h-24 rounded-2xl bg-primary/10 border border-primary/20">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
          </div>
        </div>
      </div>

      <!-- Error Message -->
      <div v-if="error" class="bg-destructive/10 text-destructive p-4 rounded-lg text-sm flex items-center gap-3">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        {{ error }}
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="text-center py-12">
        <div class="inline-flex items-center gap-3 text-muted-foreground">
          <svg class="h-5 w-5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          {{ t('common.loading') }}
        </div>
      </div>

      <template v-else-if="hasDownloads">
        <!-- Tabs -->
        <div class="flex items-center gap-2 p-1 bg-muted/50 rounded-xl w-fit">
          <button
            v-if="guiDownloads.length > 0"
            @click="activeTab = 'gui'"
            :class="['tab-button flex items-center gap-2', activeTab === 'gui' && 'tab-active']"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
              <line x1="8" y1="21" x2="16" y2="21" />
              <line x1="12" y1="17" x2="12" y2="21" />
            </svg>
            {{ t('downloads.guiTitle') }}
          </button>
          <button
            v-if="cliDownloads.length > 0"
            @click="activeTab = 'cli'"
            :class="['tab-button flex items-center gap-2', activeTab === 'cli' && 'tab-active']"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5" />
              <line x1="12" y1="19" x2="20" y2="19" />
            </svg>
            {{ t('downloads.cliTitle') }}
          </button>
        </div>

        <!-- Section Description -->
        <div class="flex items-center gap-3">
          <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-primary/10">
            <svg v-if="activeTab === 'gui'" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
              <line x1="8" y1="21" x2="16" y2="21" />
              <line x1="12" y1="17" x2="12" y2="21" />
            </svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5" />
              <line x1="12" y1="19" x2="20" y2="19" />
            </svg>
          </div>
          <div>
            <h2 class="text-xl font-semibold">
              {{ activeTab === 'gui' ? t('downloads.guiTitle') : t('downloads.cliTitle') }}
            </h2>
            <p class="text-sm text-muted-foreground">
              {{ activeTab === 'gui' ? t('downloads.guiSubtitle') : t('downloads.cliSubtitle') }}
            </p>
          </div>
        </div>

        <!-- Downloads Grid -->
        <transition name="fade" mode="out-in">
          <div :key="activeTab" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <Card
              v-for="download in currentDownloads"
              :key="download.platform"
              class="download-card p-6 cursor-pointer border-2 border-transparent"
              @click="downloadClient(download)"
            >
              <div class="flex flex-col items-center text-center space-y-4">
                <!-- Platform Icon -->
                <div class="platform-icon">
                  <svg
                    v-if="getPlatformIcon(download.os)"
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-8 w-8"
                    viewBox="0 0 24 24"
                    :fill="'#' + getPlatformIcon(download.os)!.hex"
                  >
                    <path :d="getPlatformIcon(download.os)!.path" />
                  </svg>
                  <svg
                    v-else
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-8 w-8 text-primary"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                </div>

                <!-- Info -->
                <div class="space-y-1">
                  <h3 class="font-semibold text-lg">{{ download.os }}</h3>
                  <div class="flex items-center justify-center gap-2">
                    <span class="inline-flex items-center px-2 py-0.5 rounded-md bg-primary/10 text-primary text-xs font-medium">
                      {{ getArchBadge(download) }}
                    </span>
                    <span class="text-sm text-muted-foreground">{{ formatSize(download.size) }}</span>
                  </div>
                </div>

                <!-- Download Button -->
                <Button class="w-full group" @click.stop="downloadClient(download)">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 transition-transform group-hover:translate-y-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                  {{ t('downloads.download') }}
                </Button>
              </div>
            </Card>
          </div>
        </transition>
      </template>

      <!-- No Downloads -->
      <div v-else-if="!loading" class="text-center py-12">
        <div class="inline-flex flex-col items-center gap-4">
          <div class="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
          </div>
          <div>
            <p class="text-muted-foreground font-medium">{{ t('downloads.noDownloads') }}</p>
            <p class="text-sm text-muted-foreground mt-1">{{ t('downloads.noDownloadsHint') }}</p>
          </div>
        </div>
      </div>

      <!-- Quick Start Section -->
      <Card class="p-6 md:p-8">
        <div class="flex items-center gap-3 mb-6">
          <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-primary/10">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
            </svg>
          </div>
          <div>
            <h2 class="text-xl font-semibold">{{ t('downloads.quickStart') }}</h2>
            <p class="text-sm text-muted-foreground">{{ t('downloads.quickStartDesc') || 'Get started in seconds' }}</p>
          </div>
        </div>

        <!-- OS Tabs for Quick Start -->
        <div class="flex items-center gap-1 p-1 bg-muted/50 rounded-lg w-fit mb-6">
          <button
            @click="activeOsTab = 'linux'"
            :class="['px-4 py-2 text-sm font-medium rounded-md transition-all duration-200',
                     activeOsTab === 'linux' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground']"
          >
            Linux
          </button>
          <button
            @click="activeOsTab = 'macos'"
            :class="['px-4 py-2 text-sm font-medium rounded-md transition-all duration-200',
                     activeOsTab === 'macos' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground']"
          >
            macOS
          </button>
          <button
            @click="activeOsTab = 'windows'"
            :class="['px-4 py-2 text-sm font-medium rounded-md transition-all duration-200',
                     activeOsTab === 'windows' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground']"
          >
            Windows
          </button>
        </div>

        <transition name="fade" mode="out-in">
          <div :key="activeOsTab" class="space-y-6">
            <!-- Step 1: Install -->
            <div class="space-y-2">
              <div class="flex items-center gap-2">
                <span class="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-xs font-bold">1</span>
                <p class="font-medium">{{ t('downloads.step1') }}</p>
              </div>
              <div class="relative">
                <div class="code-block">
                  <code>{{ quickStartCommands[activeOsTab].install }}</code>
                </div>
                <button
                  @click="copyCommand(quickStartCommands[activeOsTab].install)"
                  class="copy-button"
                  :title="t('common.copy') || 'Copy'"
                >
                  <svg v-if="copiedCommand !== quickStartCommands[activeOsTab].install" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                  </svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Step 2: Token + Run -->
            <div class="space-y-2">
              <div class="flex items-center gap-2">
                <span class="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-xs font-bold">2</span>
                <p class="font-medium">
                  <i18n-t keypath="downloads.step2" tag="span">
                    <template #link>
                      <router-link to="/tokens" class="text-primary hover:underline font-semibold">{{ t('downloads.step2TokenLink') }}</router-link>
                    </template>
                  </i18n-t>
                </p>
              </div>
              <div class="relative">
                <div class="code-block">
                  <code>{{ quickStartCommands[activeOsTab].run }}</code>
                </div>
                <button
                  @click="copyCommand(quickStartCommands[activeOsTab].run)"
                  class="copy-button"
                  :title="t('common.copy') || 'Copy'"
                >
                  <svg v-if="copiedCommand !== quickStartCommands[activeOsTab].run" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                  </svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Info Box -->
            <div class="flex items-start gap-3 p-4 rounded-lg bg-primary/5 border border-primary/10">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
              </svg>
              <p class="text-sm text-muted-foreground">
                <i18n-t keypath="downloads.tokenHint" tag="span">
                  <template #link>
                    <router-link to="/tokens" class="text-primary hover:underline font-medium">{{ t('downloads.tokenHintLink') }}</router-link>
                  </template>
                </i18n-t>
              </p>
            </div>
          </div>
        </transition>
      </Card>
    </div>
  </Layout>
</template>
