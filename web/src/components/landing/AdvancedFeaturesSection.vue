<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import InspectorDemo from './InspectorDemo.vue'
import DomainSetupDemo from './DomainSetupDemo.vue'

const { t } = useI18n()

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
    { threshold: 0.2 }
  )

  if (sectionRef.value) {
    observer.observe(sectionRef.value)
  }
})
</script>

<template>
  <section id="advanced" ref="sectionRef" class="py-16 md:py-32 bg-surface/30 relative overflow-hidden">
    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-20">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <span class="text-sm font-medium text-primary">{{ t('landing.advanced.label') }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.advanced.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.advanced.subtitle') }}
        </p>
      </div>

      <!-- 2 large feature cards -->
      <div class="grid lg:grid-cols-2 gap-8">
        <!-- Custom Domains -->
        <div
          class="feature-card group reveal reveal-delay-3"
          :class="{ 'visible': isVisible }"
        >
          <div class="feature-icon">
            <!-- Globe icon -->
            <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" />
            </svg>
          </div>

          <h3 class="text-xl font-display font-semibold mb-3">
            {{ t('landing.advanced.customDomains.title') }}
          </h3>

          <p class="text-muted-foreground leading-relaxed">
            {{ t('landing.advanced.customDomains.desc') }}
          </p>

          <div class="mt-auto pt-6">
            <DomainSetupDemo />
          </div>
        </div>

        <!-- HTTP Inspector & Replay -->
        <div
          class="feature-card group reveal reveal-delay-4"
          :class="{ 'visible': isVisible }"
        >
          <div class="feature-icon">
            <!-- Magnifying glass / document icon -->
            <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m5.231 13.481L15 17.25m-4.5-15H5.625c-.621 0-1.125.504-1.125 1.125v16.5c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9zm3.75 11.625a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z" />
            </svg>
          </div>

          <h3 class="text-xl font-display font-semibold mb-3">
            {{ t('landing.advanced.inspector.title') }}
          </h3>

          <p class="text-muted-foreground leading-relaxed">
            {{ t('landing.advanced.inspector.desc') }}
          </p>

          <div class="mt-auto pt-6">
            <InspectorDemo />
          </div>
        </div>
      </div>

      <!-- 3 compact cards -->
      <div
        class="mt-16 grid sm:grid-cols-2 lg:grid-cols-3 gap-6 reveal reveal-delay-5"
        :class="{ 'visible': isVisible }"
      >
        <!-- WebSocket Support -->
        <div class="flex items-start gap-4 p-5 rounded-xl bg-surface/50 border border-border">
          <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary flex-shrink-0">
            <!-- Bidirectional arrows icon -->
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-sm">{{ t('landing.advanced.websocket.title') }}</p>
            <p class="text-xs text-muted-foreground mt-1">{{ t('landing.advanced.websocket.desc') }}</p>
          </div>
        </div>

        <!-- Low Latency -->
        <div class="flex items-start gap-4 p-5 rounded-xl bg-surface/50 border border-border">
          <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary flex-shrink-0">
            <!-- Lightning bolt icon -->
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-sm">{{ t('landing.advanced.lowLatency.title') }}</p>
            <p class="text-xs text-muted-foreground mt-1">{{ t('landing.advanced.lowLatency.desc') }}</p>
          </div>
        </div>

        <!-- Prometheus Metrics -->
        <div class="flex items-start gap-4 p-5 rounded-xl bg-surface/50 border border-border">
          <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary flex-shrink-0">
            <!-- Bar chart icon -->
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-sm">{{ t('landing.advanced.metrics.title') }}</p>
            <p class="text-xs text-muted-foreground mt-1">{{ t('landing.advanced.metrics.desc') }}</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
