<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'

const { t, locale } = useI18n()

useSeo({ titleKey: 'seo.about.title', descriptionKey: 'seo.about.description' })

useSubpageSchema({
  path: '/about',
  name: t('seo.about.title'),
  description: t('seo.about.description'),
  pageType: 'AboutPage',
  breadcrumbs: [
    { name: t('nav.about', 'About'), path: '/about' },
  ],
})
</script>

<template>
  <div class="about-page">
    <!-- Navbar -->
    <nav class="about-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="about-nav-brand">
          <div class="about-nav-logo">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="about-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="about-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <!-- Content -->
    <main class="pt-16">
      <Breadcrumbs :items="[{ name: t('about.title'), path: '/about' }]" />
      <p class="container mx-auto px-4 text-xs text-muted-foreground/60 -mt-1 mb-2">
        {{ t('common.lastUpdated', { date: t('common.updateDateMar2026') }) }}
      </p>
      <div class="container mx-auto px-4 py-16 max-w-4xl">
        <div class="prose prose-neutral dark:prose-invert max-w-none">
          <template v-if="locale === 'ru'">
            <h1>{{ t('about.heroTitle') }}</h1>

            <h2>{{ t('about.projectTitle') }}</h2>
            <p>{{ t('about.projectText1') }}</p>
            <p>{{ t('about.projectText2') }}</p>

            <h2>{{ t('about.whyTitle') }}</h2>
            <p>{{ t('about.whyText') }}</p>

            <h2>{{ t('about.missionTitle') }}</h2>
            <p>{{ t('about.missionText') }}</p>

            <h2>{{ t('about.historyTitle') }}</h2>
            <p>{{ t('about.historyText') }}</p>

            <h2>{{ t('about.developerTitle') }}</h2>
            <p>{{ t('about.developerText1') }}</p>
            <p>
              {{ t('about.developerGithub') }}:
              <a href="https://github.com/mephistofox" target="_blank" rel="noopener">github.com/mephistofox</a>
            </p>

            <h2>{{ t('about.techTitle') }}</h2>
            <p>{{ t('about.techText') }}</p>
            <ul>
              <li><strong>Go 1.24+</strong> — {{ t('about.techItems.backend') }}</li>
              <li><strong>Vue 3 + TypeScript</strong> — {{ t('about.techItems.frontend') }}</li>
              <li><strong>yamux</strong> — {{ t('about.techItems.multiplexing') }}</li>
              <li><strong>SQLite</strong> — {{ t('about.techItems.database') }}</li>
              <li><strong>Wails</strong> — {{ t('about.techItems.desktop') }}</li>
              <li><strong>TLS 1.3, Let's Encrypt</strong> — {{ t('about.techItems.security') }}</li>
            </ul>

            <h2>{{ t('about.openSourceTitle') }}</h2>
            <p>{{ t('about.openSourceText') }}</p>
            <p>
              GitHub:
              <a href="https://github.com/mephistofox/fxTunnel" target="_blank" rel="noopener">github.com/mephistofox/fxTunnel</a>
            </p>

            <h2>{{ t('about.contactTitle') }}</h2>
            <table class="w-full">
              <tbody>
                <tr>
                  <td class="font-medium pr-4 py-1">{{ t('about.contactEmail') }}:</td>
                  <td><a href="mailto:support@fxtun.ru">support@fxtun.ru</a></td>
                </tr>
                <tr>
                  <td class="font-medium pr-4 py-1">GitHub Issues:</td>
                  <td>
                    <a href="https://github.com/mephistofox/fxTunnel/issues" target="_blank" rel="noopener">
                      {{ t('about.contactGithubIssues') }}
                    </a>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>

          <template v-else>
            <h1>{{ t('about.heroTitle') }}</h1>

            <h2>{{ t('about.projectTitle') }}</h2>
            <p>{{ t('about.projectText1') }}</p>
            <p>{{ t('about.projectText2') }}</p>

            <h2>{{ t('about.whyTitle') }}</h2>
            <p>{{ t('about.whyText') }}</p>

            <h2>{{ t('about.missionTitle') }}</h2>
            <p>{{ t('about.missionText') }}</p>

            <h2>{{ t('about.historyTitle') }}</h2>
            <p>{{ t('about.historyText') }}</p>

            <h2>{{ t('about.developerTitle') }}</h2>
            <p>{{ t('about.developerText1') }}</p>
            <p>
              {{ t('about.developerGithub') }}:
              <a href="https://github.com/mephistofox" target="_blank" rel="noopener">github.com/mephistofox</a>
            </p>

            <h2>{{ t('about.techTitle') }}</h2>
            <p>{{ t('about.techText') }}</p>
            <ul>
              <li><strong>Go 1.24+</strong> — {{ t('about.techItems.backend') }}</li>
              <li><strong>Vue 3 + TypeScript</strong> — {{ t('about.techItems.frontend') }}</li>
              <li><strong>yamux</strong> — {{ t('about.techItems.multiplexing') }}</li>
              <li><strong>SQLite</strong> — {{ t('about.techItems.database') }}</li>
              <li><strong>Wails</strong> — {{ t('about.techItems.desktop') }}</li>
              <li><strong>TLS 1.3, Let's Encrypt</strong> — {{ t('about.techItems.security') }}</li>
            </ul>

            <h2>{{ t('about.openSourceTitle') }}</h2>
            <p>{{ t('about.openSourceText') }}</p>
            <p>
              GitHub:
              <a href="https://github.com/mephistofox/fxTunnel" target="_blank" rel="noopener">github.com/mephistofox/fxTunnel</a>
            </p>

            <h2>{{ t('about.contactTitle') }}</h2>
            <table class="w-full">
              <tbody>
                <tr>
                  <td class="font-medium pr-4 py-1">{{ t('about.contactEmail') }}:</td>
                  <td><a href="mailto:support@fxtun.ru">support@fxtun.ru</a></td>
                </tr>
                <tr>
                  <td class="font-medium pr-4 py-1">GitHub Issues:</td>
                  <td>
                    <a href="https://github.com/mephistofox/fxTunnel/issues" target="_blank" rel="noopener">
                      {{ t('about.contactGithubIssues') }}
                    </a>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <LandingFooter />
  </div>
</template>

<style scoped>
.about-page {
  min-height: 100vh;
  background: hsl(var(--background));
}

.about-nav {
  @apply fixed top-0 left-0 right-0 z-50;
  background: hsl(var(--background) / 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid hsl(var(--border) / 0.4);
}

.about-nav-brand {
  @apply flex items-center gap-3;
}

.about-nav-logo {
  @apply w-9 h-9 rounded-xl flex items-center justify-center;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
}

.about-nav-link {
  @apply text-sm font-medium transition-colors;
  color: hsl(var(--muted-foreground));
}

.about-nav-link:hover {
  color: hsl(var(--foreground));
}

.about-nav-cta {
  @apply px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200;
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.about-nav-cta:hover {
  opacity: 0.9;
  box-shadow: 0 0 20px hsl(var(--primary) / 0.3);
}

.prose h1 {
  @apply text-3xl font-bold mb-8 text-foreground;
}

.prose h2 {
  @apply text-xl font-semibold mt-8 mb-4 text-foreground;
}

.prose p {
  @apply mb-4 text-muted-foreground leading-relaxed;
}

.prose ul {
  @apply list-disc pl-6 mb-4 space-y-2 text-muted-foreground;
}

.prose a {
  @apply text-primary hover:underline;
}

.prose table {
  @apply mt-4;
}

.prose td {
  @apply py-2;
}
</style>
