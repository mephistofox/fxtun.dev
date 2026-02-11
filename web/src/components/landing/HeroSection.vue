<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import AnimatedTerminal from './AnimatedTerminal.vue'
import TopoBackground from './TopoBackground.vue'

const { t } = useI18n()

const isVisible = ref(false)
const copied = ref(false)

const quickCommand = 'fxtunnel http 3000'

function copyCommand() {
  navigator.clipboard.writeText(quickCommand)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

onMounted(() => {
  setTimeout(() => {
    isVisible.value = true
  }, 100)
})
</script>

<template>
  <section class="hero-section">
    <!-- Animated topography contour overlay -->
    <TopoBackground />

    <!-- Animated gradient orbs -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
      <div
        class="absolute w-[600px] h-[600px] rounded-full opacity-30 blur-2xl animate-particle-float"
        style="background: radial-gradient(circle, hsl(var(--primary) / 0.4) 0%, transparent 70%); top: -20%; left: -10%;"
      />
      <div
        class="absolute w-[400px] h-[400px] rounded-full opacity-20 blur-2xl animate-particle-float"
        style="background: radial-gradient(circle, hsl(var(--accent) / 0.4) 0%, transparent 70%); bottom: 10%; right: -5%; animation-delay: -4s;"
      />
    </div>

    <!-- Main content -->
    <div class="container mx-auto px-4 pt-24 md:pt-32 pb-12 md:pb-20 relative z-10">
      <div class="grid lg:grid-cols-12 gap-12 lg:gap-8 items-center min-h-[calc(100vh-12rem)]">

        <!-- Left: Text Content (7 cols) -->
        <div
          class="lg:col-span-7 space-y-8"
          :class="{ 'opacity-0': !isVisible }"
          :style="isVisible ? 'animation: fade-in-left 0.8s ease-out forwards' : ''"
        >
          <!-- Badge -->
          <div
            class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-border bg-surface/50 backdrop-blur-sm"
            :style="isVisible ? 'animation: fade-in-up 0.6s ease-out 0.1s forwards; opacity: 0' : ''"
          >
            <span class="flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full bg-type-http" />
              <span class="w-2 h-2 rounded-full bg-type-tcp" />
              <span class="w-2 h-2 rounded-full bg-type-udp" />
            </span>
            <span class="text-sm font-medium text-muted-foreground">
              {{ t('landing.hero.badge') }}
            </span>
          </div>

          <!-- Headline -->
          <div class="space-y-4">
            <h1
              class="text-display-lg font-display"
              :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.2s forwards; opacity: 0' : ''"
            >
              <span class="block text-foreground">{{ t('landing.hero.titleLine1') }}</span>
              <span class="block gradient-text">{{ t('landing.hero.titleLine2') }}</span>
              <span class="block text-foreground">{{ t('landing.hero.titleLine3') }}</span>
            </h1>

            <p
              class="text-xl text-muted-foreground max-w-xl leading-relaxed"
              :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.35s forwards; opacity: 0' : ''"
            >
              {{ t('landing.hero.subtitle') }}
            </p>
          </div>

          <!-- Quick start command -->
          <div
            :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.45s forwards; opacity: 0' : ''"
          >
            <p class="text-sm text-muted-foreground mb-2">{{ t('landing.hero.quickStart') }}</p>
            <div
              class="inline-flex items-center gap-3 px-5 py-3 rounded-xl bg-code border border-border font-mono text-sm cursor-pointer group hover:border-primary/40 transition-colors"
              @click="copyCommand"
            >
              <span class="text-muted-foreground select-none">$</span>
              <span class="text-foreground/90">{{ quickCommand }}</span>
              <button class="ml-2 text-muted-foreground hover:text-primary transition-colors" :aria-label="t('common.copy')">
                <svg v-if="!copied" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-type-http" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              </button>
            </div>
          </div>

          <!-- CTA Buttons -->
          <div
            class="flex flex-wrap gap-4"
            :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.55s forwards; opacity: 0' : ''"
          >
            <RouterLink to="/register" class="btn-glow inline-flex items-center gap-2">
              {{ t('landing.hero.getStarted') }}
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </RouterLink>
            <a href="#how-it-works" class="btn-ghost inline-flex items-center gap-2">
              {{ t('landing.hero.learnMore') }}
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </a>
          </div>

          <!-- Trust badge -->
          <p
            class="text-sm text-muted-foreground flex items-center gap-2"
            :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.6s forwards; opacity: 0' : ''"
          >
            <svg class="h-4 w-4 text-type-http flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {{ t('landing.hero.trustBadge') }}
          </p>

          <!-- Stats row -->
          <div
            class="flex flex-wrap gap-6 pt-2"
            :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.7s forwards; opacity: 0' : ''"
          >
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-primary/10 border border-primary/20 flex items-center justify-center">
                <svg class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8.288 15.038a5.25 5.25 0 017.424 0M5.106 11.856c3.807-3.808 9.98-3.808 13.788 0M1.924 8.674c5.565-5.565 14.587-5.565 20.152 0M12.53 18.22l-.53.53-.53-.53a.75.75 0 011.06 0z" />
                </svg>
              </div>
              <div>
                <p class="text-sm font-semibold text-foreground">{{ t('landing.hero.stats.protocols') }}</p>
                <p class="text-xs text-muted-foreground">{{ t('landing.hero.stats.protocolsDesc') }}</p>
              </div>
            </div>

            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-primary/10 border border-primary/20 flex items-center justify-center">
                <svg class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M9 17.25v1.007a3 3 0 01-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0115 18.257V17.25m6-12V15a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 15V5.25m18 0A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25m18 0V12a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 12V5.25" />
                </svg>
              </div>
              <div>
                <p class="text-sm font-semibold text-foreground">{{ t('landing.hero.stats.platforms') }}</p>
                <p class="text-xs text-muted-foreground">{{ t('landing.hero.stats.platformsDesc') }}</p>
              </div>
            </div>

            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-primary/10 border border-primary/20 flex items-center justify-center">
                <svg class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
                </svg>
              </div>
              <div>
                <p class="text-sm font-semibold text-foreground">{{ t('landing.hero.stats.encryption') }}</p>
                <p class="text-xs text-muted-foreground">{{ t('landing.hero.stats.encryptionDesc') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Right: Terminal Demo (5 cols) -->
        <div
          class="lg:col-span-5 relative"
          :class="{ 'opacity-0': !isVisible }"
          :style="isVisible ? 'animation: fade-in-right 0.8s ease-out 0.3s forwards' : ''"
        >
          <!-- Glow effect behind terminal -->
          <div class="absolute -inset-8 bg-gradient-to-br from-primary/20 via-transparent to-accent/10 rounded-3xl blur-2xl opacity-60" />

          <!-- Terminal wrapper with perspective -->
          <div class="relative transform lg:rotate-1 lg:hover:rotate-0 transition-transform duration-500">
            <AnimatedTerminal class="relative z-10 shadow-2xl" />
          </div>

          <!-- Terminal caption -->
          <p class="relative z-10 text-center text-xs text-muted-foreground/60 mt-4 italic">
            {{ t('landing.hero.terminalCaption') }}
          </p>
        </div>
      </div>
    </div>

    <!-- Bottom gradient fade -->
    <div class="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-background to-transparent pointer-events-none" />
  </section>
</template>
