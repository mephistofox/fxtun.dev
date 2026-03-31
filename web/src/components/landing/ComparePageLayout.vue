<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'

const props = defineProps<{
  competitorName: string
  competitorSlug: string
}>()

const { t } = useI18n()

const allCompetitors = [
  { slug: 'ngrok', name: 'ngrok' },
  { slug: 'cloudflare', name: 'Cloudflare Tunnel' },
  { slug: 'tuna', name: 'tuna.am' },
  { slug: 'xtunnel', name: 'xTunnel' },
]
const otherCompetitors = allCompetitors.filter(c => c.slug !== props.competitorSlug)

const seoKey = `compare${props.competitorSlug.charAt(0).toUpperCase() + props.competitorSlug.slice(1).replace(/-./g, m => m[1].toUpperCase())}`

useSeo({
  titleKey: `seo.${seoKey}.title`,
  descriptionKey: `seo.${seoKey}.description`,
})

useSubpageSchema({
  path: `/compare/${props.competitorSlug}`,
  name: t(`seo.${seoKey}.title`),
  description: t(`seo.${seoKey}.description`),
  dateModified: '2026-03-27',
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
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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

    <!-- Breadcrumbs -->
    <div class="pt-16">
      <Breadcrumbs :items="[
        { name: t('common.breadcrumbCompare'), path: `/compare/${competitorSlug}` },
        { name: t(`compare.${competitorSlug}.title`), path: `/compare/${competitorSlug}` },
      ]" />
    </div>

    <!-- Hero -->
    <section class="pt-8 pb-16">
      <div class="container mx-auto px-4 text-center">
        <h1 class="text-4xl md:text-5xl font-display font-bold mb-4">
          {{ t(`compare.${competitorSlug}.title`) }}
        </h1>
        <p class="text-xl text-muted-foreground max-w-2xl mx-auto">
          {{ t('compare.heroSubtitle') }}
        </p>
        <p class="mt-3 text-sm text-muted-foreground/60">
          {{ t('common.lastUpdated', { date: t('common.updateDateMar2026') }) }}
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

    <!-- See Also -->
    <section class="py-12">
      <div class="container mx-auto px-4">
        <div class="max-w-4xl mx-auto">
          <h2 class="text-xl font-display font-semibold mb-6 text-center">{{ t('compare.seeAlsoTitle') }}</h2>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <RouterLink
              v-for="c in otherCompetitors"
              :key="c.slug"
              :to="'/compare/' + c.slug"
              class="compare-see-also-card"
            >
              <span class="text-sm font-medium">fxTunnel vs {{ c.name }}</span>
              <svg aria-hidden="true" class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/></svg>
            </RouterLink>
          </div>
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

.compare-see-also-card {
  @apply flex items-center justify-between px-5 py-4 rounded-xl transition-all duration-200;
  background: hsl(var(--surface) / 0.5);
  border: 1px solid hsl(var(--border) / 0.4);
}

.compare-see-also-card:hover {
  border-color: hsl(var(--primary) / 0.4);
  background: hsl(var(--surface) / 0.8);
}
</style>
