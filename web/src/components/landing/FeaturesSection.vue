<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// Core value propositions - 3 main features
const coreFeatures = [
  {
    icon: 'control',
    titleKey: 'landing.features.control.title',
    descKey: 'landing.features.control.desc',
    highlights: [
      'landing.features.control.highlight1',
      'landing.features.control.highlight2',
      'landing.features.control.highlight3',
    ],
  },
  {
    icon: 'protocols',
    titleKey: 'landing.features.protocols.title',
    descKey: 'landing.features.protocols.desc',
    highlights: [
      'landing.features.protocols.highlight1',
      'landing.features.protocols.highlight2',
      'landing.features.protocols.highlight3',
    ],
  },
  {
    icon: 'security',
    titleKey: 'landing.features.security.title',
    descKey: 'landing.features.security.desc',
    highlights: [
      'landing.features.security.highlight1',
      'landing.features.security.highlight2',
      'landing.features.security.highlight3',
    ],
  },
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
    { threshold: 0.2 }
  )

  if (sectionRef.value) {
    observer.observe(sectionRef.value)
  }
})
</script>

<template>
  <section id="features" ref="sectionRef" class="py-32 bg-background relative overflow-hidden">
    <!-- Subtle background pattern -->
    <div class="absolute inset-0 opacity-30">
      <div class="absolute inset-0 bg-grid-pattern bg-grid-60" style="mask-image: radial-gradient(ellipse 60% 50% at 50% 50%, black 20%, transparent 70%);" />
    </div>

    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-20">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <span class="text-sm font-medium text-primary">{{ t('landing.features.label') || 'Why fxTunnel' }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.features.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.features.subtitle') }}
        </p>
      </div>

      <!-- Features grid - 3 large cards -->
      <div class="grid lg:grid-cols-3 gap-8">
        <div
          v-for="(feature, index) in coreFeatures"
          :key="index"
          class="feature-card group reveal"
          :class="[
            { 'visible': isVisible },
            `reveal-delay-${3 + index}`
          ]"
        >
          <!-- Icon -->
          <div class="feature-icon">
            <!-- Control icon -->
            <svg v-if="feature.icon === 'control'" xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
            </svg>
            <!-- Protocols icon -->
            <svg v-else-if="feature.icon === 'protocols'" xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.288 15.038a5.25 5.25 0 017.424 0M5.106 11.856c3.807-3.808 9.98-3.808 13.788 0M1.924 8.674c5.565-5.565 14.587-5.565 20.152 0M12.53 18.22l-.53.53-.53-.53a.75.75 0 011.06 0z" />
            </svg>
            <!-- Security icon -->
            <svg v-else-if="feature.icon === 'security'" xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
            </svg>
          </div>

          <!-- Title -->
          <h3 class="text-xl font-display font-semibold mb-3">
            {{ t(feature.titleKey) || feature.titleKey }}
          </h3>

          <!-- Description -->
          <p class="text-muted-foreground mb-6 leading-relaxed">
            {{ t(feature.descKey) || feature.descKey }}
          </p>

          <!-- Highlights list -->
          <ul class="space-y-3">
            <li
              v-for="(highlight, hIndex) in feature.highlights"
              :key="hIndex"
              class="flex items-start gap-3 text-sm"
            >
              <svg class="h-5 w-5 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
              </svg>
              <span class="text-foreground/80">{{ t(highlight) || highlight }}</span>
            </li>
          </ul>
        </div>
      </div>

      <!-- Additional features row -->
      <div
        class="mt-16 grid sm:grid-cols-2 lg:grid-cols-4 gap-6 reveal reveal-delay-7"
        :class="{ 'visible': isVisible }"
      >
        <div class="flex items-center gap-4 p-4 rounded-xl bg-surface/50 border border-border">
          <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-sm">{{ t('landing.features.fast.title') || 'Lightning Fast' }}</p>
            <p class="text-xs text-muted-foreground">{{ t('landing.features.fast.short') || 'Go + yamux' }}</p>
          </div>
        </div>

        <div class="flex items-center gap-4 p-4 rounded-xl bg-surface/50 border border-border">
          <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-sm">{{ t('landing.features.multiUser.title') || 'Multi-User' }}</p>
            <p class="text-xs text-muted-foreground">{{ t('landing.features.multiUser.short') || 'Admin panel' }}</p>
          </div>
        </div>

        <div class="flex items-center gap-4 p-4 rounded-xl bg-surface/50 border border-border">
          <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-sm">{{ t('landing.features.subdomains.title') || 'Custom Subdomains' }}</p>
            <p class="text-xs text-muted-foreground">{{ t('landing.features.subdomains.short') || 'Reserve yours' }}</p>
          </div>
        </div>

        <div class="flex items-center gap-4 p-4 rounded-xl bg-surface/50 border border-border">
          <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 17.25v1.007a3 3 0 01-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0115 18.257V17.25m6-12V15a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 15V5.25m18 0A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25m18 0V12a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 12V5.25" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-sm">{{ t('landing.features.gui.title') || 'GUI & CLI' }}</p>
            <p class="text-xs text-muted-foreground">{{ t('landing.features.gui.short') || 'Your choice' }}</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
