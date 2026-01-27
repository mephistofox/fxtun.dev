<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import AnimatedTerminal from './AnimatedTerminal.vue'

const { t } = useI18n()

const isVisible = ref(false)

onMounted(() => {
  setTimeout(() => {
    isVisible.value = true
  }, 100)
})
</script>

<template>
  <section class="hero-section">
    <!-- Grid overlay -->
    <div class="grid-overlay" />

    <!-- Animated gradient orbs -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
      <div
        class="absolute w-[600px] h-[600px] rounded-full opacity-30 blur-3xl animate-particle-float"
        style="background: radial-gradient(circle, hsl(var(--primary) / 0.4) 0%, transparent 70%); top: -20%; left: -10%;"
      />
      <div
        class="absolute w-[400px] h-[400px] rounded-full opacity-20 blur-3xl animate-particle-float"
        style="background: radial-gradient(circle, hsl(var(--accent) / 0.4) 0%, transparent 70%); bottom: 10%; right: -5%; animation-delay: -4s;"
      />
    </div>

    <!-- Main content -->
    <div class="container mx-auto px-4 pt-32 pb-20 relative z-10">
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
            <span class="pulse-indicator" />
            <span class="text-sm font-medium text-muted-foreground">
              {{ t('landing.hero.badge') || 'Self-hosted tunneling' }}
            </span>
          </div>

          <!-- Headline -->
          <div class="space-y-4">
            <h1
              class="text-display-lg font-display"
              :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.2s forwards; opacity: 0' : ''"
            >
              <span class="block text-foreground">{{ t('landing.hero.titleLine1') || 'Expose your' }}</span>
              <span class="block gradient-text">{{ t('landing.hero.titleLine2') || 'localhost' }}</span>
              <span class="block text-foreground">{{ t('landing.hero.titleLine3') || 'to the world' }}</span>
            </h1>

            <p
              class="text-xl text-muted-foreground max-w-xl leading-relaxed"
              :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.35s forwards; opacity: 0' : ''"
            >
              {{ t('landing.hero.subtitle') }}
            </p>
          </div>

          <!-- CTA Buttons -->
          <div
            class="flex flex-wrap gap-4"
            :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.5s forwards; opacity: 0' : ''"
          >
            <RouterLink to="/register" class="btn-glow inline-flex items-center gap-2">
              {{ t('landing.hero.getStarted') }}
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </RouterLink>
            <a
              href="https://github.com/mephistofox/fxTunnel"
              target="_blank"
              rel="noopener"
              class="btn-ghost inline-flex items-center gap-2"
            >
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
              </svg>
              GitHub
            </a>
            <a href="#features" class="btn-ghost inline-flex items-center gap-2">
              {{ t('landing.hero.learnMore') }}
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </a>
          </div>

          <!-- Protocol indicators -->
          <div
            class="flex flex-wrap gap-3 pt-4"
            :style="isVisible ? 'animation: fade-in-up 0.8s ease-out 0.65s forwards; opacity: 0' : ''"
          >
            <div class="flex items-center gap-2 px-4 py-2 rounded-lg bg-type-http/10 border border-type-http/30">
              <div class="w-2 h-2 rounded-full bg-type-http" />
              <span class="text-sm font-medium text-type-http">HTTP</span>
            </div>
            <div class="flex items-center gap-2 px-4 py-2 rounded-lg bg-type-tcp/10 border border-type-tcp/30">
              <div class="w-2 h-2 rounded-full bg-type-tcp" />
              <span class="text-sm font-medium text-type-tcp">TCP</span>
            </div>
            <div class="flex items-center gap-2 px-4 py-2 rounded-lg bg-type-udp/10 border border-type-udp/30">
              <div class="w-2 h-2 rounded-full bg-type-udp" />
              <span class="text-sm font-medium text-type-udp">UDP</span>
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
        </div>
      </div>
    </div>

    <!-- Bottom gradient fade -->
    <div class="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-background to-transparent pointer-events-none" />
  </section>
</template>
