<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'

const props = defineProps<{
  competitorName: string
  competitorSlug: string
}>()

const { t } = useI18n()

const seoKey = `compare${props.competitorSlug.charAt(0).toUpperCase() + props.competitorSlug.slice(1).replace(/-./g, m => m[1].toUpperCase())}`

useSeo({
  titleKey: `seo.${seoKey}.title`,
  descriptionKey: `seo.${seoKey}.description`,
})

useSubpageSchema({
  path: `/compare/${props.competitorSlug}`,
  name: t(`seo.${seoKey}.title`),
  description: t(`seo.${seoKey}.description`),
  breadcrumbs: [
    { name: t('compare.breadcrumbCompare', 'Compare'), path: `/compare/${props.competitorSlug}` },
  ],
})
</script>

<template>
  <div class="compare-page">
    <!-- Navbar -->
    <nav class="compare-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="compare-nav-brand">
          <div class="compare-nav-logo">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="compare-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="compare-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <!-- Hero -->
    <section class="pt-32 pb-16">
      <div class="container mx-auto px-4 text-center">
        <h1 class="text-4xl md:text-5xl font-display font-bold mb-4">
          {{ t(`compare.${competitorSlug}.title`) }}
        </h1>
        <p class="text-xl text-muted-foreground max-w-2xl mx-auto">
          {{ t('compare.heroSubtitle') }}
        </p>
      </div>
    </section>

    <!-- Comparison Table -->
    <slot name="table" />

    <!-- Feature Details -->
    <slot name="details" />

    <!-- CTA -->
    <section class="py-16 md:py-24">
      <div class="container mx-auto px-4 text-center">
        <div class="max-w-2xl mx-auto p-8 md:p-12 rounded-2xl compare-cta-card">
          <h2 class="text-2xl md:text-3xl font-display font-bold mb-4">
            {{ t('compare.ctaTitle') }}
          </h2>
          <p class="text-muted-foreground mb-8">
            {{ t('compare.ctaSubtitle') }}
          </p>
          <RouterLink to="/register" class="compare-cta-button">
            {{ t('compare.ctaButton') }}
          </RouterLink>
        </div>
      </div>
    </section>

    <!-- FAQ -->
    <slot name="faq" />

    <!-- Footer -->
    <LandingFooter />
  </div>
</template>

<style scoped>
.compare-page {
  min-height: 100vh;
  background: hsl(var(--background));
}

.compare-nav {
  @apply fixed top-0 left-0 right-0 z-50;
  background: hsl(var(--background) / 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid hsl(var(--border) / 0.4);
}

.compare-nav-brand {
  @apply flex items-center gap-3;
}

.compare-nav-logo {
  @apply w-9 h-9 rounded-xl flex items-center justify-center;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
}

.compare-nav-link {
  @apply text-sm font-medium transition-colors;
  color: hsl(var(--muted-foreground));
}

.compare-nav-link:hover {
  color: hsl(var(--foreground));
}

.compare-nav-cta {
  @apply px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.compare-nav-cta:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}

.compare-cta-card {
  background: hsl(var(--surface) / 0.5);
  border: 1px solid hsl(var(--border) / 0.5);
}

.compare-cta-button {
  @apply inline-flex items-center px-6 py-3 rounded-lg text-base font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.compare-cta-button:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}
</style>
