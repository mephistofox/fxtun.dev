<script setup lang="ts">
import { ref, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema, useFaqSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'

const { t, tm } = useI18n()

useSeo({
  titleKey: 'seo.ngrokAlternative.title',
  descriptionKey: 'seo.ngrokAlternative.description',
})

interface Wedge {
  title: string
  desc: string
}

interface FaqItem {
  q: string
  a: string
}

const wedges = computed(() => tm('landing.ngrokAlt.wedges') as Wedge[])
const faqItems = computed(() => tm('landing.ngrokAlt.faq') as FaqItem[])

useSubpageSchema({
  path: '/ngrok-alternative',
  name: t('landing.ngrokAlt.h1'),
  description: t('seo.ngrokAlternative.description'),
  breadcrumbs: [
    { name: t('landing.ngrokAlt.h1'), path: '/ngrok-alternative' },
  ],
})

useFaqSchema(
  faqItems.value.map(item => ({ question: item.q, answer: item.a })),
  '-ngrok-alt',
)

const command = 'fxtun http 3000'
const copied = ref(false)

async function copyCommand() {
  try {
    await navigator.clipboard.writeText(command)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    copied.value = false
  }
}

const openFaqIndex = ref<number | null>(null)

function toggleFaq(index: number) {
  openFaqIndex.value = openFaqIndex.value === index ? null : index
}
</script>

<template>
  <div class="ngrok-alt-page">
    <!-- Navbar -->
    <nav class="page-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="page-nav-brand">
          <div class="page-nav-logo">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="page-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="page-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <main class="pt-16">
      <Breadcrumbs :items="[{ name: t('landing.ngrokAlt.h1'), path: '/ngrok-alternative' }]" />

      <!-- Hero -->
      <section class="container mx-auto px-4 pt-8 pb-12 md:pt-12 md:pb-16 max-w-4xl text-center">
        <h1 class="text-3xl md:text-5xl font-display font-bold mb-6">
          {{ t('landing.ngrokAlt.h1') }}
        </h1>
        <p class="text-lg md:text-xl text-muted-foreground leading-relaxed max-w-3xl mx-auto">
          {{ t('landing.ngrokAlt.intro') }}
        </p>
      </section>

      <!-- Wedges -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-5xl">
        <div class="grid gap-5 sm:grid-cols-2">
          <div
            v-for="(wedge, index) in wedges"
            :key="index"
            class="wedge-card"
          >
            <h2 class="text-lg font-display font-semibold mb-2">{{ wedge.title }}</h2>
            <p class="text-muted-foreground leading-relaxed">{{ wedge.desc }}</p>
          </div>
        </div>
      </section>

      <!-- Quick start -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-3xl">
        <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-8">
          {{ t('landing.ngrokAlt.quickStartTitle') }}
        </h2>
        <div class="command-block">
          <code class="command-text">
            <span class="command-prompt">$</span>
            <span>{{ command }}</span>
          </code>
          <button type="button" class="command-copy" @click="copyCommand">
            <svg v-if="!copied" aria-hidden="true" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" />
            </svg>
            <svg v-else aria-hidden="true" class="h-4 w-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
            </svg>
          </button>
        </div>
      </section>

      <!-- Comparison CTA -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-3xl text-center">
        <h2 class="text-2xl md:text-3xl font-display font-bold mb-6">
          {{ t('landing.ngrokAlt.comparisonTitle') }}
        </h2>
        <RouterLink to="/compare/ngrok" class="comparison-link">
          {{ t('landing.ngrokAlt.comparisonButton') }}
          <svg aria-hidden="true" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
          </svg>
        </RouterLink>
      </section>

      <!-- FAQ -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-3xl">
        <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-12">
          {{ t('landing.faq.title') }}
        </h2>
        <div>
          <div
            v-for="(item, index) in faqItems"
            :key="index"
            class="border-b border-border"
            :class="{ 'border-primary/20': openFaqIndex === index }"
          >
            <button
              @click="toggleFaq(index)"
              class="w-full flex items-center justify-between py-5 text-left group"
            >
              <span
                class="text-base font-medium pr-8 group-hover:text-primary transition-colors"
                :class="{ 'text-primary': openFaqIndex === index }"
              >
                {{ item.q }}
              </span>
              <svg
                class="h-5 w-5 flex-shrink-0 text-muted-foreground transition-transform duration-300"
                :class="{ 'rotate-180 text-primary': openFaqIndex === index }"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
              </svg>
            </button>
            <Transition name="faq-expand">
              <div v-if="openFaqIndex === index" class="faq-answer">
                <p class="pb-5 text-muted-foreground leading-relaxed">
                  {{ item.a }}
                </p>
              </div>
            </Transition>
          </div>
        </div>
      </section>

      <!-- Final CTA -->
      <section class="container mx-auto px-4 pb-16 md:pb-24 max-w-3xl text-center">
        <h2 class="text-2xl md:text-3xl font-display font-bold mb-8">
          {{ t('landing.ngrokAlt.ctaTitle') }}
        </h2>
        <RouterLink to="/register" class="cta-button">
          {{ t('landing.ngrokAlt.ctaButton') }}
        </RouterLink>
      </section>
    </main>

    <LandingFooter />
  </div>
</template>

<style scoped>
.ngrok-alt-page {
  min-height: 100vh;
  background: hsl(var(--background));
}

.page-nav {
  @apply fixed top-0 left-0 right-0 z-50;
  background: hsl(var(--background) / 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid hsl(var(--border) / 0.4);
}

.page-nav-brand {
  @apply flex items-center gap-3;
}

.page-nav-logo {
  @apply w-9 h-9 rounded-xl flex items-center justify-center;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
}

.page-nav-link {
  @apply text-sm font-medium transition-colors;
  color: hsl(var(--muted-foreground));
}

.page-nav-link:hover {
  color: hsl(var(--foreground));
}

.page-nav-cta {
  @apply px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.page-nav-cta:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}

.wedge-card {
  @apply rounded-2xl p-6;
  background: hsl(var(--surface) / 0.4);
  border: 1px solid hsl(var(--border) / 0.6);
  transition: border-color 0.2s ease;
}

.wedge-card:hover {
  border-color: hsl(var(--primary) / 0.3);
}

.command-block {
  @apply flex items-center justify-between gap-4 rounded-xl px-5 py-4;
  background: hsl(var(--surface) / 0.6);
  border: 1px solid hsl(var(--border));
}

.command-text {
  @apply flex items-center gap-3 text-sm md:text-base font-mono;
  color: hsl(var(--foreground));
}

.command-prompt {
  color: hsl(var(--primary));
  user-select: none;
}

.command-copy {
  @apply flex-shrink-0 p-2 rounded-lg transition-colors;
  color: hsl(var(--muted-foreground));
}

.command-copy:hover {
  color: hsl(var(--foreground));
  background: hsl(var(--surface));
}

.comparison-link {
  @apply inline-flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-medium transition-all duration-200;
  border: 1px solid hsl(var(--border));
  color: hsl(var(--foreground));
}

.comparison-link:hover {
  border-color: hsl(var(--primary) / 0.4);
  color: hsl(var(--primary));
}

.cta-button {
  @apply inline-flex items-center justify-center px-8 py-4 rounded-xl text-base font-semibold transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.cta-button:hover {
  opacity: 0.9;
  box-shadow: 0 0 24px hsl(var(--primary) / 0.35);
}

.faq-expand-enter-active,
.faq-expand-leave-active {
  transition: all 0.3s ease;
  overflow: hidden;
}

.faq-expand-enter-from,
.faq-expand-leave-to {
  opacity: 0;
  max-height: 0;
}

.faq-expand-enter-to,
.faq-expand-leave-from {
  opacity: 1;
  max-height: 200px;
}
</style>
