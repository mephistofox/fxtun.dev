<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { downloadsApi, type Download } from '@/api/client'

const { t } = useI18n()

// State
const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)
const activeOS = ref<'linux' | 'macos' | 'windows'>('linux')
const downloads = ref<Download[]>([])
const loading = ref(true)
const copied = ref(false)

const installCommand = 'curl -fsSL https://fxtun.dev/install.sh | sh'

const osTabs = [
  { key: 'linux' as const, label: 'Linux', osMatch: 'Linux' },
  { key: 'macos' as const, label: 'macOS', osMatch: 'macOS' },
  { key: 'windows' as const, label: 'Windows', osMatch: 'Windows' },
]

// Auto-detect platform
function detectPlatform(): 'linux' | 'macos' | 'windows' {
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('mac')) return 'macos'
  if (ua.includes('win')) return 'windows'
  return 'linux'
}

// Filtered downloads for active OS
const filteredDownloads = computed(() => {
  const osName = osTabs.find(t => t.key === activeOS.value)?.osMatch || 'Linux'
  return downloads.value.filter(d => d.os === osName)
})

function formatSize(bytes: number): string {
  if (bytes === 0) return ''
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

async function copyInstallCommand() {
  try {
    await navigator.clipboard.writeText(installCommand)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // fallback
  }
}

function downloadFile(dl: Download) {
  window.location.href = dl.url
}

onMounted(async () => {
  activeOS.value = detectPlatform()

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          isVisible.value = true
          observer.disconnect()
        }
      })
    },
    { threshold: 0.15 }
  )
  if (sectionRef.value) {
    observer.observe(sectionRef.value)
  }

  try {
    const resp = await downloadsApi.list()
    downloads.value = [...(resp.data.cli || []), ...(resp.data.gui || [])]
  } catch {
    // silent — section will show "no builds"
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section id="download" ref="sectionRef" class="py-16 md:py-32 relative overflow-hidden">
    <!-- Background -->
    <div class="absolute inset-0 bg-gradient-to-b from-background via-surface/30 to-background" />

    <!-- Glow effect -->
    <div class="absolute top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] rounded-full opacity-20 blur-3xl pointer-events-none" style="background: radial-gradient(ellipse, hsl(var(--primary) / 0.3) 0%, transparent 60%);" />

    <div class="container mx-auto px-4 relative z-10">
      <div class="max-w-4xl mx-auto">
        <!-- Main CTA card -->
        <div
          class="glass-card-glow p-8 md:p-12 text-center reveal"
          :class="{ 'visible': isVisible }"
        >
          <!-- Badge -->
          <div
            class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-8 reveal reveal-delay-1"
            :class="{ 'visible': isVisible }"
          >
            <span class="pulse-indicator" />
            <span class="text-sm font-medium text-primary">{{ t('landing.download.badge') || 'Ready to start?' }}</span>
          </div>

          <!-- Headline -->
          <h2
            class="text-display-lg font-display mb-4 reveal reveal-delay-2"
            :class="{ 'visible': isVisible }"
          >
            {{ t('landing.download.title') }}
          </h2>

          <p
            class="text-xl text-muted-foreground mb-10 max-w-2xl mx-auto reveal reveal-delay-3"
            :class="{ 'visible': isVisible }"
          >
            {{ t('landing.download.subtitle') }}
          </p>

          <!-- One-liner install command -->
          <div
            class="max-w-xl mx-auto mb-8 reveal reveal-delay-4"
            :class="{ 'visible': isVisible }"
          >
            <p class="text-sm font-medium text-muted-foreground mb-3">{{ t('landing.download.installCommand') }}</p>
            <div class="relative group">
              <div class="bg-code rounded-xl p-4 font-mono text-sm border border-border flex items-center justify-between gap-3 overflow-hidden">
                <code class="text-primary truncate select-all">{{ installCommand }}</code>
                <button
                  @click="copyInstallCommand"
                  class="flex-shrink-0 p-2 rounded-lg bg-surface/50 text-muted-foreground hover:text-foreground hover:bg-surface transition-all"
                  :title="copied ? t('landing.download.copied') : t('common.copy')"
                  :aria-label="copied ? t('landing.download.copied') : t('common.copy')"
                >
                  <svg v-if="!copied" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                  </svg>
                  <svg v-else class="h-4 w-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Divider -->
          <div
            class="flex items-center gap-4 max-w-md mx-auto mb-8 reveal reveal-delay-4"
            :class="{ 'visible': isVisible }"
          >
            <div class="flex-1 h-px bg-border" />
            <span class="text-xs text-muted-foreground uppercase tracking-wider">{{ t('landing.download.orManual') }}</span>
            <div class="flex-1 h-px bg-border" />
          </div>

          <!-- Platform tabs -->
          <div
            class="flex justify-center mb-8 reveal reveal-delay-5"
            :class="{ 'visible': isVisible }"
          >
            <div class="inline-flex p-1.5 rounded-2xl bg-surface border border-border">
              <button
                v-for="tab in osTabs"
                :key="tab.key"
                @click="activeOS = tab.key"
                class="relative px-5 py-2.5 rounded-xl text-sm font-medium transition-all duration-300"
                :class="[
                  activeOS === tab.key
                    ? 'bg-primary/15 text-primary shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                ]"
              >
                <span class="flex items-center gap-2">
                  <!-- Linux -->
                  <svg v-if="tab.key === 'linux'" class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12.504 0c-.155 0-.315.008-.48.021-4.226.333-3.105 4.807-3.17 6.298-.076 1.092-.3 1.953-1.05 3.02-.885 1.051-2.127 2.75-2.716 4.521-.278.832-.41 1.684-.287 2.489a.424.424 0 00-.11.135c-.26.268-.45.6-.663.839-.199.199-.485.267-.797.4-.313.136-.658.269-.864.68-.09.189-.136.394-.132.602 0 .199.027.4.055.536.058.399.116.728.04.97-.249.68-.28 1.145-.106 1.484.174.334.535.47.94.601.81.2 1.91.135 2.774.6.926.466 1.866.67 2.616.47.526-.116.97-.464 1.208-.946.587-.003 1.23-.269 2.26-.334.699-.058 1.574.267 2.577.2.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071.771-.06 1.592-.536 2.257-1.306.631-.765 1.683-1.084 2.378-1.503.348-.199.629-.469.649-.853.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.357.357 0 00-.19-.064c.431-1.278.264-2.55-.173-3.694-.533-1.41-1.465-2.638-2.175-3.483-.796-1.005-1.576-1.957-1.56-3.368.026-2.152.236-6.133-3.544-6.139z"/>
                  </svg>
                  <!-- macOS -->
                  <svg v-else-if="tab.key === 'macos'" class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>
                  </svg>
                  <!-- Windows -->
                  <svg v-else-if="tab.key === 'windows'" class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M3 12V6.75l6-1.32v6.48L3 12m17-9v8.75l-10 .15V5.21L20 3m-10 15.32l10 1.38V12l-10 .09v6.23m-7-.42v-5.9h6v6.23l-6-1.33z"/>
                  </svg>
                  {{ tab.label }}
                </span>
              </button>
            </div>
          </div>

          <!-- Download cards -->
          <div
            class="mb-10 reveal reveal-delay-5"
            :class="{ 'visible': isVisible }"
          >
            <div v-if="loading" class="flex justify-center py-8">
              <svg class="h-5 w-5 animate-spin text-muted-foreground" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </div>

            <div v-else-if="filteredDownloads.length > 0" class="flex flex-wrap justify-center gap-4">
              <button
                v-for="dl in filteredDownloads"
                :key="dl.platform"
                @click="downloadFile(dl)"
                class="group flex items-center gap-3 px-5 py-3 rounded-xl bg-surface border border-border hover:border-primary/40 hover:bg-primary/5 transition-all duration-300"
              >
                <svg class="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                </svg>
                <div class="text-left">
                  <div class="text-sm font-medium text-foreground group-hover:text-primary transition-colors">
                    {{ dl.client_type === 'gui' ? 'GUI' : 'CLI' }} — {{ dl.arch.toUpperCase() }}
                  </div>
                  <div v-if="dl.size" class="text-xs text-muted-foreground">{{ formatSize(dl.size) }}</div>
                </div>
              </button>
            </div>

            <p v-else class="text-sm text-muted-foreground py-4">
              {{ t('landing.download.noBuilds') }}
            </p>
          </div>

          <!-- CTA Buttons -->
          <div
            class="flex flex-col sm:flex-row gap-4 justify-center reveal reveal-delay-6"
            :class="{ 'visible': isVisible }"
          >
            <RouterLink to="/register" class="btn-glow inline-flex items-center justify-center gap-2 text-lg">
              {{ t('landing.download.createAccount') }}
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </RouterLink>
            <RouterLink to="/login" class="btn-ghost inline-flex items-center justify-center gap-2">
              {{ t('landing.download.signIn') }}
            </RouterLink>
          </div>

          <!-- Note -->
          <p
            class="text-sm text-muted-foreground mt-8 reveal reveal-delay-7"
            :class="{ 'visible': isVisible }"
          >
            {{ t('landing.download.note') }}
          </p>
        </div>

      </div>
    </div>
  </section>
</template>
