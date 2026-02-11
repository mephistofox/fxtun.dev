<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { useSeo } from '@/composables/useSeo'
import { useOrganizationSchema, useSoftwareApplicationSchema, useWebSiteSchema, useFaqSchema } from '@/composables/useStructuredData'
import HeroSection from '@/components/landing/HeroSection.vue'
import FeaturesSection from '@/components/landing/FeaturesSection.vue'
import AdvancedFeaturesSection from '@/components/landing/AdvancedFeaturesSection.vue'
import HowItWorksSection from '@/components/landing/HowItWorksSection.vue'
import ProtocolsSection from '@/components/landing/ProtocolsSection.vue'
import UseCasesSection from '@/components/landing/UseCasesSection.vue'
import PricingSection from '@/components/landing/PricingSection.vue'
import ComparisonSection from '@/components/landing/ComparisonSection.vue'
import DownloadSection from '@/components/landing/DownloadSection.vue'
import FaqSection from '@/components/landing/FaqSection.vue'
import LandingFooter from '@/components/landing/LandingFooter.vue'

const themeStore = useThemeStore()
const { t, tm } = useI18n()

useSeo({ titleKey: 'seo.landing.title', descriptionKey: 'seo.landing.description' })
useOrganizationSchema()
useSoftwareApplicationSchema()
useWebSiteSchema()

const faqItems = tm('landing.faq.items') as Array<{ q: string; a: string }>
useFaqSchema(faqItems.map(item => ({ question: item.q, answer: item.a })))

const isScrolled = ref(false)
const isMobileMenuOpen = ref(false)

function toggleLocale() {
  const current = getLocale()
  setLocale(current === 'en' ? 'ru' : 'en')
}

function cycleTheme() {
  const modes: ThemeMode[] = ['light', 'dark', 'system']
  const currentIndex = modes.indexOf(themeStore.mode)
  const nextIndex = (currentIndex + 1) % modes.length
  themeStore.setMode(modes[nextIndex])
}

const blogUrl = computed(() => {
  if (typeof window === 'undefined') return '/blog'
  return `${window.location.protocol}//${window.location.hostname}/blog`
})

function handleScroll() {
  isScrolled.value = window.scrollY > 20
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll, { passive: true })
  handleScroll()
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <!-- Navigation -->
    <nav
      class="fixed top-0 left-0 right-0 z-50 transition-all duration-300"
      :class="[
        isScrolled || isMobileMenuOpen
          ? 'bg-background/80 backdrop-blur-xl border-b border-border shadow-sm'
          : 'bg-transparent'
      ]"
    >
      <div class="container mx-auto px-4">
        <div class="flex items-center justify-between h-16 lg:h-20">
          <!-- Logo -->
          <RouterLink to="/" class="flex items-center gap-3 group">
            <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center transition-all duration-300 group-hover:bg-primary/20 group-hover:border-primary/40">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <span class="font-display font-semibold text-xl">fxTunnel</span>
          </RouterLink>

          <!-- Desktop Nav Links -->
          <div class="hidden lg:flex items-center gap-8">
            <a href="#how-it-works" class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
              {{ t('landing.nav.howItWorks') }}
            </a>
            <a href="#protocols" class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
              {{ t('landing.nav.protocols') }}
            </a>
            <a href="#features" class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
              {{ t('landing.nav.features') }}
            </a>
            <a href="#pricing" class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
              {{ t('landing.nav.pricing') }}
            </a>
            <a href="#download" class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
              {{ t('landing.nav.download') }}
            </a>
            <a href="#faq" class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
              {{ t('landing.nav.faq') }}
            </a>
            <a :href="blogUrl" class="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
              {{ t('landing.nav.blog') }}
            </a>
          </div>

          <!-- Right Controls -->
          <div class="flex items-center gap-2">
            <!-- Theme toggle -->
            <button
              @click="cycleTheme"
              class="p-2.5 rounded-xl hover:bg-surface transition-colors"
              :title="t(`theme.${themeStore.mode}`)"
            >
              <svg
                v-if="themeStore.mode === 'light'"
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 text-muted-foreground"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <circle cx="12" cy="12" r="5" />
                <line x1="12" y1="1" x2="12" y2="3" />
                <line x1="12" y1="21" x2="12" y2="23" />
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                <line x1="1" y1="12" x2="3" y2="12" />
                <line x1="21" y1="12" x2="23" y2="12" />
                <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
              </svg>
              <svg
                v-else-if="themeStore.mode === 'dark'"
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 text-muted-foreground"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
              </svg>
              <svg
                v-else
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 text-muted-foreground"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
                <line x1="8" y1="21" x2="16" y2="21" />
                <line x1="12" y1="17" x2="12" y2="21" />
              </svg>
            </button>

            <!-- Language toggle -->
            <button
              @click="toggleLocale"
              class="px-3 py-2 text-sm font-medium rounded-xl hover:bg-surface transition-colors text-muted-foreground"
            >
              {{ getLocale() === 'en' ? 'RU' : 'EN' }}
            </button>

            <!-- Sign in button -->
            <RouterLink
              to="/login"
              class="hidden sm:inline-flex ml-2 px-5 py-2.5 text-sm font-medium rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 transition-colors shadow-sm hover:shadow-glow-sm"
            >
              {{ t('auth.signIn') }}
            </RouterLink>

            <!-- Mobile menu button -->
            <button
              @click="isMobileMenuOpen = !isMobileMenuOpen"
              class="lg:hidden p-2.5 rounded-xl hover:bg-surface transition-colors"
              :aria-label="isMobileMenuOpen ? t('common.close') : t('common.menu')"
              :aria-expanded="isMobileMenuOpen"
            >
              <svg v-if="!isMobileMenuOpen" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
              <svg v-else class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        <!-- Mobile menu -->
        <Transition name="slide-up">
          <div v-if="isMobileMenuOpen" class="lg:hidden pb-6">
            <div class="flex flex-col gap-2 py-4">
              <a
                href="#how-it-works"
                @click="isMobileMenuOpen = false"
                class="px-4 py-3 rounded-xl text-muted-foreground hover:text-foreground hover:bg-surface transition-colors"
              >
                {{ t('landing.nav.howItWorks') }}
              </a>
              <a
                href="#protocols"
                @click="isMobileMenuOpen = false"
                class="px-4 py-3 rounded-xl text-muted-foreground hover:text-foreground hover:bg-surface transition-colors"
              >
                {{ t('landing.nav.protocols') }}
              </a>
              <a
                href="#features"
                @click="isMobileMenuOpen = false"
                class="px-4 py-3 rounded-xl text-muted-foreground hover:text-foreground hover:bg-surface transition-colors"
              >
                {{ t('landing.nav.features') }}
              </a>
              <a
                href="#pricing"
                @click="isMobileMenuOpen = false"
                class="px-4 py-3 rounded-xl text-muted-foreground hover:text-foreground hover:bg-surface transition-colors"
              >
                {{ t('landing.nav.pricing') }}
              </a>
              <a
                href="#download"
                @click="isMobileMenuOpen = false"
                class="px-4 py-3 rounded-xl text-muted-foreground hover:text-foreground hover:bg-surface transition-colors"
              >
                {{ t('landing.nav.download') }}
              </a>
              <a
                href="#faq"
                @click="isMobileMenuOpen = false"
                class="px-4 py-3 rounded-xl text-muted-foreground hover:text-foreground hover:bg-surface transition-colors"
              >
                {{ t('landing.nav.faq') }}
              </a>
              <a
                :href="blogUrl"
                @click="isMobileMenuOpen = false"
                class="px-4 py-3 rounded-xl text-muted-foreground hover:text-foreground hover:bg-surface transition-colors"
              >
                {{ t('landing.nav.blog') }}
              </a>
              <RouterLink
                to="/login"
                class="mt-2 px-4 py-3 rounded-xl bg-primary text-primary-foreground text-center font-medium"
              >
                {{ t('auth.signIn') }}
              </RouterLink>
            </div>
          </div>
        </Transition>
      </div>
    </nav>

    <!-- Main Content -->
    <main>
      <HeroSection />
      <HowItWorksSection />

      <!-- Mid-page CTA: after How It Works -->
      <div class="py-12 text-center">
        <p class="text-lg text-muted-foreground mb-6 max-w-xl mx-auto px-4">{{ t('landing.midCta.afterHowItWorks') }}</p>
        <RouterLink to="/register" class="btn-glow inline-flex items-center gap-2">
          {{ t('landing.midCta.startFree') }}
          <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
        </RouterLink>
      </div>

      <ProtocolsSection />
      <FeaturesSection />
      <UseCasesSection />

      <!-- Mid-page CTA: after Use Cases -->
      <div class="py-12 text-center">
        <p class="text-lg text-muted-foreground mb-6 max-w-xl mx-auto px-4">{{ t('landing.midCta.afterUseCases') }}</p>
        <RouterLink to="/register" class="btn-glow inline-flex items-center gap-2">
          {{ t('landing.midCta.startFree') }}
          <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
        </RouterLink>
      </div>

      <AdvancedFeaturesSection />
      <ComparisonSection />

      <!-- Mid-page CTA: after Comparison -->
      <div class="py-12 text-center">
        <p class="text-lg text-muted-foreground mb-6 max-w-xl mx-auto px-4">{{ t('landing.midCta.afterComparison') }}</p>
        <div class="flex flex-wrap justify-center gap-4">
          <RouterLink to="/register" class="btn-glow inline-flex items-center gap-2">
            {{ t('landing.midCta.startFree') }}
            <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
          </RouterLink>
          <a href="#pricing" class="btn-ghost inline-flex items-center gap-2">
            {{ t('landing.midCta.seePricing') }}
          </a>
        </div>
      </div>

      <PricingSection />
      <DownloadSection />
      <FaqSection />
    </main>

    <LandingFooter />
  </div>
</template>
