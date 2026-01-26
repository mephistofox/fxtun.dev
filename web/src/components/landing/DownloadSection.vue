<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n()

const platforms = [
  { name: 'Linux', icon: 'linux', architectures: ['x64', 'ARM64'] },
  { name: 'macOS', icon: 'apple', architectures: ['Intel', 'Apple Silicon'] },
  { name: 'Windows', icon: 'windows', architectures: ['x64'] },
]

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)

onMounted(() => {
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
})
</script>

<template>
  <section id="download" ref="sectionRef" class="py-32 relative overflow-hidden">
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

          <!-- Platform icons -->
          <div
            class="flex justify-center gap-6 md:gap-10 mb-10 reveal reveal-delay-4"
            :class="{ 'visible': isVisible }"
          >
            <div
              v-for="platform in platforms"
              :key="platform.name"
              class="text-center group cursor-default"
            >
              <div class="platform-icon mx-auto mb-3 group-hover:shadow-glow-sm transition-all duration-300">
                <!-- Linux -->
                <svg v-if="platform.icon === 'linux'" class="h-8 w-8 text-foreground" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12.504 0c-.155 0-.315.008-.48.021-4.226.333-3.105 4.807-3.17 6.298-.076 1.092-.3 1.953-1.05 3.02-.885 1.051-2.127 2.75-2.716 4.521-.278.832-.41 1.684-.287 2.489a.424.424 0 00-.11.135c-.26.268-.45.6-.663.839-.199.199-.485.267-.797.4-.313.136-.658.269-.864.68-.09.189-.136.394-.132.602 0 .199.027.4.055.536.058.399.116.728.04.97-.249.68-.28 1.145-.106 1.484.174.334.535.47.94.601.81.2 1.91.135 2.774.6.926.466 1.866.67 2.616.47.526-.116.97-.464 1.208-.946.587-.003 1.23-.269 2.26-.334.699-.058 1.574.267 2.577.2.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071.771-.06 1.592-.536 2.257-1.306.631-.765 1.683-1.084 2.378-1.503.348-.199.629-.469.649-.853.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.357.357 0 00-.19-.064c.431-1.278.264-2.55-.173-3.694-.533-1.41-1.465-2.638-2.175-3.483-.796-1.005-1.576-1.957-1.56-3.368.026-2.152.236-6.133-3.544-6.139z"/>
                </svg>
                <!-- Apple -->
                <svg v-else-if="platform.icon === 'apple'" class="h-8 w-8 text-foreground" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>
                </svg>
                <!-- Windows -->
                <svg v-else-if="platform.icon === 'windows'" class="h-8 w-8 text-foreground" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M3 12V6.75l6-1.32v6.48L3 12m17-9v8.75l-10 .15V5.21L20 3m-10 15.32l10 1.38V12l-10 .09v6.23m-7-.42v-5.9h6v6.23l-6-1.33z"/>
                </svg>
              </div>
              <p class="text-sm font-medium text-foreground">{{ platform.name }}</p>
              <p class="text-xs text-muted-foreground">{{ platform.architectures.join(' / ') }}</p>
            </div>
          </div>

          <!-- CTA Buttons -->
          <div
            class="flex flex-col sm:flex-row gap-4 justify-center reveal reveal-delay-5"
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
            class="text-sm text-muted-foreground mt-8 reveal reveal-delay-6"
            :class="{ 'visible': isVisible }"
          >
            {{ t('landing.download.note') }}
          </p>
        </div>

        <!-- Trust indicators -->
        <div
          class="mt-16 text-center reveal reveal-delay-7"
          :class="{ 'visible': isVisible }"
        >
          <p class="text-sm text-muted-foreground mb-6">{{ t('landing.download.openSource') || 'Open source and self-hosted' }}</p>
          <div class="flex items-center justify-center gap-8">
            <a
              href="https://github.com/mephistofox/fxTunnel"
              target="_blank"
              rel="noopener"
              class="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <svg class="h-6 w-6" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
              </svg>
              <span class="font-medium">GitHub</span>
            </a>
            <div class="flex items-center gap-2 text-muted-foreground">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
              </svg>
              <span>{{ t('landing.download.secure') || 'Self-hosted & Secure' }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
