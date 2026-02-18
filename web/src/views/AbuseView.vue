<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import { useSeo } from '@/composables/useSeo'

const themeStore = useThemeStore()
const { t } = useI18n()

useSeo({ titleKey: 'seo.abuse.title', descriptionKey: 'seo.abuse.description' })

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
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Theme and Language Switchers -->
    <div class="fixed top-4 right-4 flex items-center space-x-2 z-50">
      <button
        @click="cycleTheme"
        class="p-2 rounded-lg hover:bg-accent/10 transition-colors"
        :title="t(`theme.${themeStore.mode}`)"
      >
        <svg
          v-if="themeStore.mode === 'light'"
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
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
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
        <svg
          v-else
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
          <line x1="8" y1="21" x2="16" y2="21" />
          <line x1="12" y1="17" x2="12" y2="21" />
        </svg>
      </button>
      <button
        @click="toggleLocale"
        class="px-2 py-1 text-sm font-medium rounded-lg hover:bg-accent/10 transition-colors"
      >
        {{ getLocale() === 'en' ? 'RU' : 'EN' }}
      </button>
    </div>

    <!-- Back to landing -->
    <RouterLink
      to="/"
      class="fixed top-4 left-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors z-50"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M9.707 16.707a1 1 0 01-1.414 0l-6-6a1 1 0 010-1.414l6-6a1 1 0 011.414 1.414L5.414 9H17a1 1 0 110 2H5.414l4.293 4.293a1 1 0 010 1.414z" clip-rule="evenodd" />
      </svg>
      {{ t('landing.nav.backToHome') }}
    </RouterLink>

    <div class="container mx-auto px-4 py-16 max-w-4xl">
      <!-- Header -->
      <div class="mb-8">
        <h1 class="text-3xl font-bold mb-4">{{ t('legal.abuseTitle') }}</h1>
      </div>

      <!-- Content -->
      <div class="prose prose-neutral dark:prose-invert max-w-none">

        <p>
          If you have encountered abuse, illegal content, or malicious activity originating from
          the fxtun network, please report it using the information below. We take all reports
          seriously and will investigate promptly.
        </p>

        <h2>1. How to Report</h2>
        <p>Send an email to:</p>
        <div class="my-4 p-4 rounded-lg border border-border bg-surface/50">
          <a href="mailto:abuse@fxtun.dev" class="text-lg font-mono font-semibold text-primary">
            abuse@fxtun.dev
          </a>
        </div>

        <h2>2. What to Include</h2>
        <p>To help us investigate quickly, please provide as much of the following as possible:</p>
        <ul>
          <li><strong>IP address and port</strong> — the public IP and port where the abuse was observed;</li>
          <li><strong>Timestamp</strong> — date and time of the incident (include timezone);</li>
          <li><strong>Subdomain or URL</strong> — if the abuse involved an HTTP tunnel (e.g., <code>malicious.fxtun.dev</code>);</li>
          <li><strong>Description</strong> — what happened, what type of abuse (phishing, malware, scanning, spam, etc.);</li>
          <li><strong>Evidence</strong> — logs, screenshots, packet captures, or any supporting material;</li>
          <li><strong>Your contact information</strong> — so we can follow up if needed.</li>
        </ul>

        <h2>3. Response Time</h2>
        <ul>
          <li><strong>Critical reports</strong> (active attacks, CSAM, imminent threats): we aim to respond and take action within <strong>4 hours</strong>;</li>
          <li><strong>Standard reports</strong> (phishing, spam, policy violations): we aim to respond within <strong>24 hours</strong> on business days;</li>
          <li><strong>Informational reports</strong> (suspicious activity, low-severity concerns): we aim to respond within <strong>72 hours</strong>.</li>
        </ul>

        <h2>4. What We Will Do</h2>
        <p>Upon receiving a valid report, we may:</p>
        <ul>
          <li>Immediately close the offending tunnel(s);</li>
          <li>Suspend or terminate the account responsible;</li>
          <li>Preserve relevant logs for law enforcement if applicable;</li>
          <li>Notify you of the actions taken (subject to legal and privacy constraints).</li>
        </ul>

        <h2>5. Legal & Law Enforcement</h2>
        <p>
          For legal requests, subpoenas, or law enforcement inquiries, please contact us at
          <a href="mailto:abuse@fxtun.dev">abuse@fxtun.dev</a>.
        </p>
        <table class="w-full">
          <tbody>
            <tr>
              <td class="font-medium pr-4 py-1">Company:</td>
              <td>Nocodo LTD</td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Jurisdiction:</td>
              <td>Republic of Cyprus</td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Website:</td>
              <td><a href="https://nocodo.tech">nocodo.tech</a></td>
            </tr>
          </tbody>
        </table>

        <h2>6. Related Policies</h2>
        <ul>
          <li><RouterLink to="/aup">Acceptable Use Policy</RouterLink></li>
          <li><RouterLink to="/terms">Terms of Service</RouterLink></li>
          <li><RouterLink to="/disclaimer">TCP & UDP Risk Disclaimer</RouterLink></li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prose h2 {
  @apply text-xl font-semibold mt-8 mb-4 text-foreground;
}

.prose h3 {
  @apply text-lg font-medium mt-6 mb-3 text-foreground;
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

.prose code {
  @apply text-sm bg-surface px-1.5 py-0.5 rounded;
}
</style>
