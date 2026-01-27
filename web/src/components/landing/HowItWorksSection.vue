<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const steps = [
  {
    number: '01',
    icon: 'download',
    titleKey: 'landing.howItWorks.step1.title',
    descKey: 'landing.howItWorks.step1.desc',
    code: 'curl -L https://get.fxtunnel.io | sh',
  },
  {
    number: '02',
    icon: 'key',
    titleKey: 'landing.howItWorks.step2.title',
    descKey: 'landing.howItWorks.step2.desc',
    code: 'fxtunnel login --token sk_xxxxx',
  },
  {
    number: '03',
    icon: 'rocket',
    titleKey: 'landing.howItWorks.step3.title',
    descKey: 'landing.howItWorks.step3.desc',
    code: 'fxtunnel http 3000 --subdomain myapp',
  },
]

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)
const cardsContainerRef = ref<HTMLElement | null>(null)

function handleMouseMove(e: MouseEvent) {
  if (!cardsContainerRef.value) return

  const containerRect = cardsContainerRef.value.getBoundingClientRect()
  const cards = cardsContainerRef.value.querySelectorAll<HTMLElement>('[data-card]')

  cards.forEach((card) => {
    const cardRect = card.getBoundingClientRect()
    const x = e.clientX - cardRect.left
    const y = e.clientY - cardRect.top
    card.style.setProperty('--mouse-x', `${x}px`)
    card.style.setProperty('--mouse-y', `${y}px`)
  })
}

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

  if (cardsContainerRef.value) {
    cardsContainerRef.value.addEventListener('mousemove', handleMouseMove)
  }
})

onUnmounted(() => {
  if (cardsContainerRef.value) {
    cardsContainerRef.value.removeEventListener('mousemove', handleMouseMove)
  }
})
</script>

<template>
  <section id="how-it-works" ref="sectionRef" class="py-32 relative overflow-hidden">
    <!-- Background gradient -->
    <div class="absolute inset-0 bg-gradient-to-b from-background via-surface/30 to-background" />

    <!-- Accent glow -->
    <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[400px] rounded-full opacity-20 blur-3xl pointer-events-none" style="background: radial-gradient(ellipse, hsl(var(--primary) / 0.3) 0%, transparent 70%);" />

    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-20">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <span class="text-sm font-medium text-primary">{{ t('landing.howItWorks.label') || 'Quick Start' }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.howItWorks.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.howItWorks.subtitle') }}
        </p>
      </div>

      <!-- Steps -->
      <div class="max-w-5xl mx-auto">
        <!-- Cards container with spotlight effect -->
        <div
          ref="cardsContainerRef"
          class="cards-spotlight grid md:grid-cols-3 gap-4 md:gap-5"
        >
          <div
            v-for="(step, index) in steps"
            :key="index"
            data-card
            class="spotlight-card reveal"
            :class="[
              { 'visible': isVisible },
              `reveal-delay-${3 + index}`
            ]"
          >
            <!-- Card inner content -->
            <div class="spotlight-card-content">
              <!-- Step number badge -->
              <div class="flex items-center justify-between mb-6">
                <div class="w-12 h-12 rounded-xl bg-primary/10 border border-primary/30 flex items-center justify-center relative">
                  <span class="text-lg font-display font-bold text-primary">{{ step.number }}</span>
                  <div v-if="index === 0" class="absolute inset-0 rounded-xl animate-pulse-ring bg-primary/20" />
                </div>

                <div class="w-10 h-10 rounded-lg bg-surface flex items-center justify-center text-muted-foreground">
                  <svg v-if="step.icon === 'download'" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                  </svg>
                  <svg v-else-if="step.icon === 'key'" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                  </svg>
                  <svg v-else-if="step.icon === 'rocket'" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.59 14.37a6 6 0 01-5.84 7.38v-4.8m5.84-2.58a14.98 14.98 0 006.16-12.12A14.98 14.98 0 009.631 8.41m5.96 5.96a14.926 14.926 0 01-5.841 2.58m-.119-8.54a6 6 0 00-7.381 5.84h4.8m2.581-5.84a14.927 14.927 0 00-2.58 5.84m2.699 2.7c-.103.021-.207.041-.311.06a15.09 15.09 0 01-2.448-2.448 14.9 14.9 0 01.06-.312m-2.24 2.39a4.493 4.493 0 00-1.757 4.306 4.493 4.493 0 004.306-1.758M16.5 9a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0z" />
                  </svg>
                </div>
              </div>

              <!-- Content -->
              <h3 class="text-lg font-display font-semibold mb-2">
                {{ t(step.titleKey) }}
              </h3>
              <p class="text-muted-foreground text-sm mb-4">
                {{ t(step.descKey) }}
              </p>

              <!-- Code block -->
              <div class="relative group">
                <div class="bg-[hsl(220,20%,6%)] rounded-lg p-3 font-mono text-sm overflow-x-auto border border-border">
                  <code class="text-primary">{{ step.code }}</code>
                </div>
                <div class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button class="p-1.5 rounded bg-surface/80 text-muted-foreground hover:text-foreground">
                    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Mobile connectors -->
        <div class="md:hidden flex flex-col items-center -mt-2">
          <div v-for="i in 2" :key="i" class="w-px h-8 bg-gradient-to-b from-primary/50 to-border my-2" />
        </div>

        <!-- Result showcase -->
        <div
          class="mt-16 text-center reveal reveal-delay-7"
          :class="{ 'visible': isVisible }"
        >
          <div class="inline-flex items-center gap-3 px-6 py-3 rounded-full bg-type-http/10 border border-type-http/30">
            <div class="pulse-indicator" style="background: hsl(var(--type-http));" />
            <span class="font-mono text-sm text-type-http">
              https://myapp.tunnel.example.com
            </span>
            <span class="text-xs text-muted-foreground">→ localhost:3000</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.spotlight-card {
  --mouse-x: 50%;
  --mouse-y: 50%;
  position: relative;
  border-radius: 1rem;
  background: hsl(var(--background));
  overflow: hidden;
}

/* Animated gradient border */
.spotlight-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1px;
  background: linear-gradient(
    180deg,
    hsl(var(--border)) 0%,
    hsl(var(--border) / 0.3) 100%
  );
  -webkit-mask:
    linear-gradient(#fff 0 0) content-box,
    linear-gradient(#fff 0 0);
  mask:
    linear-gradient(#fff 0 0) content-box,
    linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
  transition: opacity 0.3s;
}

/* Spotlight glow effect on hover */
.spotlight-card::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  opacity: 0;
  transition: opacity 0.4s;
  background: radial-gradient(
    600px circle at var(--mouse-x) var(--mouse-y),
    hsl(var(--primary) / 0.15),
    transparent 40%
  );
  pointer-events: none;
}

.cards-spotlight:hover .spotlight-card::after {
  opacity: 1;
}

/* Border glow on hover */
.spotlight-card:hover::before {
  background: linear-gradient(
    180deg,
    hsl(var(--primary) / 0.5) 0%,
    hsl(var(--primary) / 0.1) 100%
  );
}

.spotlight-card-content {
  position: relative;
  z-index: 10;
  padding: 1.5rem;
  height: 100%;
  background: hsl(var(--background));
  border-radius: inherit;
}

/* Subtle inner shadow for depth */
.spotlight-card-content::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  box-shadow: inset 0 1px 1px hsl(var(--foreground) / 0.03);
  pointer-events: none;
}
</style>
