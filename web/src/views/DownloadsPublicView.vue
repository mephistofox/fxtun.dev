<script setup lang="ts">
import { ref, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema, useFaqSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'

const { t, tm } = useI18n()

useSeo({ titleKey: 'seo.downloads.title', descriptionKey: 'seo.downloads.description' })

useSubpageSchema({
  path: '/downloads',
  name: t('downloadsPage.heroTitle'),
  description: t('seo.downloads.description'),
  breadcrumbs: [
    { name: t('downloadsPage.heroTitle'), path: '/downloads' },
  ],
})

// FAQ
interface FaqItem {
  q: string
  a: string
}

const faqItems = computed(() => tm('downloadsFaq.items') as FaqItem[])
useFaqSchema(faqItems.value.map(item => ({ question: item.q, answer: item.a })), '-downloads')

const openFaqIndex = ref<number | null>(null)

function toggleFaq(index: number) {
  openFaqIndex.value = openFaqIndex.value === index ? null : index
}

// Requirements data
const requirementRows = computed(() => [
  { label: 'OS', value: t('downloadsPage.requirements.os') },
  { label: 'CPU', value: t('downloadsPage.requirements.arch') },
  { label: 'Disk', value: t('downloadsPage.requirements.disk') },
  { label: 'RAM', value: t('downloadsPage.requirements.ram') },
])

const afterInstallSteps = computed(() => tm('downloadsPage.afterInstallSteps') as string[])
</script>

<template>
  <div class="downloads-page">
    <!-- Navbar -->
    <nav class="downloads-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="downloads-nav-brand">
          <div class="downloads-nav-logo">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="downloads-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="downloads-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <!-- Content -->
    <main class="pt-16">
      <!-- Hero -->
      <section class="py-16 md:py-24">
        <div class="container mx-auto px-4 max-w-4xl text-center">
          <h1 class="text-4xl md:text-5xl font-bold mb-4 text-foreground">{{ t('downloadsPage.heroTitle') }}</h1>
          <p class="text-lg md:text-xl text-muted-foreground mb-4">{{ t('downloadsPage.heroSubtitle') }}</p>
          <div class="flex items-center justify-center gap-3">
            <span class="downloads-badge">v{{ t('downloadsPage.version') }}</span>
            <span class="text-sm text-muted-foreground">{{ t('downloadsPage.updatedDate') }}</span>
          </div>
        </div>
      </section>

      <!-- Description -->
      <section class="pb-16">
        <div class="container mx-auto px-4 max-w-3xl">
          <p class="text-muted-foreground leading-relaxed text-base">{{ t('downloadsPage.description') }}</p>
        </div>
      </section>

      <!-- Cards: CLI + GUI -->
      <section class="pb-16">
        <div class="container mx-auto px-4 max-w-4xl grid md:grid-cols-2 gap-6">
          <!-- CLI Card -->
          <div class="downloads-card">
            <div class="downloads-card-icon">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="4 17 10 11 4 5" /><line x1="12" x2="20" y1="19" y2="19" />
              </svg>
            </div>
            <h2 class="text-xl font-semibold mb-3 text-foreground">{{ t('downloadsPage.cliTitle') }}</h2>
            <p class="text-muted-foreground text-sm leading-relaxed">{{ t('downloadsPage.cliDescription') }}</p>
          </div>

          <!-- GUI Card -->
          <div class="downloads-card">
            <div class="downloads-card-icon">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect width="20" height="14" x="2" y="3" rx="2" /><line x1="8" x2="16" y1="21" y2="21" /><line x1="12" x2="12" y1="17" y2="21" />
              </svg>
            </div>
            <h2 class="text-xl font-semibold mb-3 text-foreground">{{ t('downloadsPage.guiTitle') }}</h2>
            <p class="text-muted-foreground text-sm leading-relaxed">{{ t('downloadsPage.guiDescription') }}</p>
          </div>
        </div>
      </section>

      <!-- Quick Start -->
      <section class="pb-16">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl font-bold mb-6 text-foreground">{{ t('downloadsPage.quickStartTitle') }}</h2>
          <div class="space-y-4">
            <div>
              <p class="text-sm font-medium text-muted-foreground mb-2">Linux / macOS</p>
              <code class="downloads-code">{{ t('downloadsPage.quickStartLinux') }}</code>
            </div>
            <div>
              <p class="text-sm font-medium text-muted-foreground mb-2">Windows (PowerShell)</p>
              <code class="downloads-code">{{ t('downloadsPage.quickStartWindows') }}</code>
            </div>
          </div>
        </div>
      </section>

      <!-- Platforms -->
      <section class="pb-16">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl font-bold mb-6 text-foreground">{{ t('downloadsPage.platformsTitle') }}</h2>
          <ul class="space-y-3">
            <li class="downloads-platform">
              <span class="downloads-platform-icon">&#128039;</span>
              <span class="text-muted-foreground">{{ t('downloadsPage.platforms.linux') }}</span>
            </li>
            <li class="downloads-platform">
              <span class="downloads-platform-icon">&#127822;</span>
              <span class="text-muted-foreground">{{ t('downloadsPage.platforms.macos') }}</span>
            </li>
            <li class="downloads-platform">
              <span class="downloads-platform-icon">&#128187;</span>
              <span class="text-muted-foreground">{{ t('downloadsPage.platforms.windows') }}</span>
            </li>
          </ul>
        </div>
      </section>

      <!-- System Requirements -->
      <section class="pb-16">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl font-bold mb-6 text-foreground">{{ t('downloadsPage.requirementsTitle') }}</h2>
          <div class="downloads-card">
            <div class="space-y-3">
              <div
                v-for="row in requirementRows"
                :key="row.label"
                class="flex items-start gap-3"
              >
                <span class="text-sm font-medium text-primary min-w-[3rem]">{{ row.label }}</span>
                <span class="text-sm text-muted-foreground">{{ row.value }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- After Installation -->
      <section class="pb-16">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl font-bold mb-6 text-foreground">{{ t('downloadsPage.afterInstallTitle') }}</h2>
          <ol class="space-y-4">
            <li
              v-for="(step, index) in afterInstallSteps"
              :key="index"
              class="downloads-step"
            >
              <span class="downloads-step-number">{{ index + 1 }}</span>
              <span class="text-muted-foreground text-sm leading-relaxed">{{ step }}</span>
            </li>
          </ol>
        </div>
      </section>

      <!-- FAQ -->
      <section class="pb-16">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl font-bold mb-8 text-foreground text-center">{{ t('downloadsFaq.title') }}</h2>
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
        </div>
      </section>

      <!-- CTA -->
      <section class="pb-20">
        <div class="container mx-auto px-4 max-w-3xl text-center">
          <h2 class="text-2xl font-bold mb-3 text-foreground">{{ t('downloadsPage.ctaTitle') }}</h2>
          <p class="text-muted-foreground mb-6">{{ t('downloadsPage.ctaText') }}</p>
          <RouterLink to="/register" class="downloads-cta-btn">{{ t('downloadsPage.ctaButton') }}</RouterLink>
        </div>
      </section>
    </main>

    <!-- Footer -->
    <LandingFooter />
  </div>
</template>

<style scoped>
.downloads-page {
  min-height: 100vh;
  background: hsl(var(--background));
}

.downloads-nav {
  @apply fixed top-0 left-0 right-0 z-50;
  background: hsl(var(--background) / 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid hsl(var(--border) / 0.4);
}

.downloads-nav-brand {
  @apply flex items-center gap-3;
}

.downloads-nav-logo {
  @apply w-9 h-9 rounded-xl flex items-center justify-center;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
}

.downloads-nav-link {
  @apply text-sm font-medium transition-colors;
  color: hsl(var(--muted-foreground));
}

.downloads-nav-link:hover {
  color: hsl(var(--foreground));
}

.downloads-nav-cta {
  @apply px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.downloads-nav-cta:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}

.downloads-card {
  @apply p-6 rounded-xl border;
  background: hsl(var(--card));
  border-color: hsl(var(--border));
}

.downloads-card-icon {
  @apply w-10 h-10 rounded-lg flex items-center justify-center mb-4;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
}

.downloads-code {
  @apply block w-full px-4 py-3 rounded-lg text-sm font-mono;
  background: hsl(var(--muted));
  color: hsl(var(--foreground));
  border: 1px solid hsl(var(--border));
}

.downloads-platform {
  @apply flex items-center gap-3 px-4 py-3 rounded-lg;
  background: hsl(var(--muted) / 0.5);
}

.downloads-platform-icon {
  @apply text-lg;
}

.downloads-cta-btn {
  @apply inline-block px-8 py-3 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.downloads-cta-btn:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}

.downloads-badge {
  @apply inline-block px-3 py-1 rounded-full text-xs font-medium;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
  border: 1px solid hsl(var(--primary) / 0.2);
}

.downloads-step {
  @apply flex items-start gap-4 px-4 py-3 rounded-lg;
  background: hsl(var(--muted) / 0.5);
}

.downloads-step-number {
  @apply flex-shrink-0 w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold;
  background: hsl(var(--primary) / 0.1);
  color: hsl(var(--primary));
  border: 1px solid hsl(var(--primary) / 0.2);
}

/* FAQ transitions */
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
