<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { useSeo } from '@/composables/useSeo'
import PricingSection from '@/components/landing/PricingSection.vue'
import LandingFooter from '@/components/landing/LandingFooter.vue'

const { t } = useI18n()

useSeo({ titleKey: 'landing.pricing.title', descriptionKey: 'landing.pricing.subtitle' })
</script>

<template>
  <div class="pricing-page">
    <!-- Navbar -->
    <nav class="pricing-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="pricing-nav-brand">
          <div class="pricing-nav-logo">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxTunnel</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="pricing-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="pricing-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <!-- Pricing section (reuse landing component) -->
    <main class="pt-16">
      <PricingSection compact />
    </main>

    <!-- Footer -->
    <LandingFooter />
  </div>
</template>

<style scoped>
.pricing-page {
  min-height: 100vh;
  background: hsl(var(--background));
}

.pricing-nav {
  @apply fixed top-0 left-0 right-0 z-50;
  background: hsl(var(--background) / 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid hsl(var(--border) / 0.4);
}

.pricing-nav-brand {
  @apply flex items-center gap-3;
}

.pricing-nav-logo {
  @apply w-9 h-9 rounded-xl flex items-center justify-center;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
}

.pricing-nav-link {
  @apply text-sm font-medium transition-colors;
  color: hsl(var(--muted-foreground));
}

.pricing-nav-link:hover {
  color: hsl(var(--foreground));
}

.pricing-nav-cta {
  @apply px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.pricing-nav-cta:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}
</style>
