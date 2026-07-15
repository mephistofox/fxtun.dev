<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'

const { t, tm } = useI18n()

interface FeatureItem {
  title: string
  desc: string
}

interface FeatureCategory {
  title: string
  items: FeatureItem[]
}

const categories = computed(() => tm('landing.featuresPage.categories') as FeatureCategory[])

useSeo({ titleKey: 'seo.features.title', descriptionKey: 'seo.features.description' })

useSubpageSchema({
  path: '/features',
  name: t('seo.features.title'),
  description: t('seo.features.description'),
  pageType: 'WebPage',
  breadcrumbs: [
    { name: t('landing.nav.features', 'Features'), path: '/features' },
  ],
})
</script>

<template>
  <div class="features-page">
    <!-- Navbar -->
    <nav class="features-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="features-nav-brand">
          <div class="features-nav-logo">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="features-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="features-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <!-- Content -->
    <main class="pt-16">
      <Breadcrumbs :items="[{ name: t('landing.nav.features', 'Features'), path: '/features' }]" />

      <div class="container mx-auto px-4 py-16 max-w-4xl">
        <!-- Intro -->
        <header class="mb-12">
          <h1 class="text-3xl md:text-4xl font-bold mb-4 text-foreground">
            {{ t('landing.featuresPage.h1') }}
          </h1>
          <p class="text-lg text-muted-foreground leading-relaxed">
            {{ t('landing.featuresPage.intro') }}
          </p>
        </header>

        <!-- Categories -->
        <div class="space-y-12">
          <section
            v-for="(category, ci) in categories"
            :key="ci"
            class="features-category"
          >
            <h2 class="text-xl font-semibold mb-6 text-foreground">
              {{ category.title }}
            </h2>
            <div class="grid gap-4 sm:grid-cols-2">
              <div
                v-for="(item, ii) in category.items"
                :key="ii"
                class="features-item"
              >
                <h3 class="text-base font-medium mb-1.5 text-foreground">
                  {{ item.title }}
                </h3>
                <p class="text-sm text-muted-foreground leading-relaxed">
                  {{ item.desc }}
                </p>
              </div>
            </div>
          </section>
        </div>

        <!-- CTA -->
        <div class="features-cta">
          <h2 class="text-2xl font-bold mb-6 text-foreground">
            {{ t('landing.featuresPage.ctaTitle') }}
          </h2>
          <RouterLink to="/register" class="features-cta-button">
            {{ t('landing.featuresPage.ctaButton') }}
          </RouterLink>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <LandingFooter />
  </div>
</template>

<style scoped>
.features-page {
  min-height: 100vh;
  background: hsl(var(--background));
}

.features-nav {
  @apply fixed top-0 left-0 right-0 z-50;
  background: hsl(var(--background) / 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid hsl(var(--border) / 0.4);
}

.features-nav-brand {
  @apply flex items-center gap-3;
}

.features-nav-logo {
  @apply w-9 h-9 rounded-xl flex items-center justify-center;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
}

.features-nav-link {
  @apply text-sm font-medium transition-colors;
  color: hsl(var(--muted-foreground));
}

.features-nav-link:hover {
  color: hsl(var(--foreground));
}

.features-nav-cta {
  @apply px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.features-nav-cta:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}

.features-category {
  @apply pb-2;
}

.features-item {
  @apply p-5 rounded-xl;
  background: hsl(var(--card) / 0.4);
  border: 1px solid hsl(var(--border) / 0.5);
}

.features-cta {
  @apply mt-16 pt-12 text-center border-t border-border;
}

.features-cta-button {
  @apply inline-flex items-center justify-center px-6 py-3 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.features-cta-button:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}
</style>
