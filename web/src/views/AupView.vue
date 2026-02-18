<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import { useSeo } from '@/composables/useSeo'

const themeStore = useThemeStore()
const { t } = useI18n()

useSeo({ titleKey: 'seo.aup.title', descriptionKey: 'seo.aup.description' })

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

const lastUpdated = '18.02.2026'
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
        <h1 class="text-3xl font-bold mb-4">{{ t('legal.aupTitle') }}</h1>
        <span class="text-sm text-muted-foreground">
          {{ t('legal.lastUpdated') }}: {{ lastUpdated }}
        </span>
      </div>

      <!-- Content -->
      <div class="prose prose-neutral dark:prose-invert max-w-none">

        <h2>1. Purpose</h2>
        <p>
          This Acceptable Use Policy ("AUP") defines the rules and boundaries for using the fxTunnel
          tunneling service. It supplements the
          <RouterLink to="/terms" class="text-primary hover:underline">Terms of Service</RouterLink>
          and applies to all users regardless of their subscription plan.
        </p>
        <p>
          fxTunnel provides infrastructure for exposing local services to the internet. We are committed
          to keeping the platform safe, reliable, and free from abuse. Violation of this policy may
          result in immediate account suspension or termination.
        </p>

        <h2>2. Prohibited Activities</h2>
        <p>You must not use fxTunnel to:</p>

        <h3>2.1. Malicious Content & Attacks</h3>
        <ul>
          <li>Host or distribute malware, ransomware, spyware, viruses, or other malicious software;</li>
          <li>Create phishing pages, credential harvesting forms, or social engineering attack infrastructure;</li>
          <li>Conduct denial-of-service (DoS/DDoS) attacks or amplification attacks against any target;</li>
          <li>Perform unauthorized port scanning, vulnerability scanning, or penetration testing of third-party systems;</li>
          <li>Operate command-and-control (C2) infrastructure for botnets or compromised devices.</li>
        </ul>

        <h3>2.2. Illegal Content</h3>
        <ul>
          <li>Host, transmit, or distribute child sexual abuse material (CSAM) in any form;</li>
          <li>Distribute content that violates intellectual property rights (piracy, counterfeit goods);</li>
          <li>Facilitate illegal activities including but not limited to fraud, money laundering, or trafficking;</li>
          <li>Publish content that incites violence, hatred, or terrorism.</li>
        </ul>

        <h3>2.3. Spam & Unsolicited Communications</h3>
        <ul>
          <li>Relay spam, bulk unsolicited emails, or automated messages;</li>
          <li>Operate open mail relays or proxy servers for unsolicited communications;</li>
          <li>Harvest email addresses or personal data from third-party websites.</li>
        </ul>

        <h3>2.4. Infrastructure Abuse</h3>
        <ul>
          <li>Consume excessive resources that degrade service quality for other users;</li>
          <li>Attempt to circumvent service limits, rate limits, or access controls;</li>
          <li>Probe, scan, or test the vulnerability of fxTunnel infrastructure without authorization;</li>
          <li>Reverse-engineer, decompile, or interfere with the fxTunnel platform;</li>
          <li>Resell, sublicense, or redistribute fxTunnel access without written permission.</li>
        </ul>

        <h3>2.5. TCP & UDP Tunnel Specific</h3>
        <ul>
          <li>Expose unprotected or known-vulnerable services (e.g., open databases, unpatched SSH) to the public internet;</li>
          <li>Use TCP/UDP tunnels to bypass network security policies of your organization;</li>
          <li>Operate anonymous proxy or VPN services through fxTunnel tunnels;</li>
          <li>Run open public proxies (HTTP, SOCKS, or any other protocol) accessible to arbitrary third parties;</li>
          <li>Tunnel protocols designed for network attacks (e.g., ARP spoofing, DHCP starvation).</li>
        </ul>

        <h2>3. Your Responsibilities</h2>
        <ul>
          <li>You are solely responsible for all content and traffic passing through your tunnels;</li>
          <li>Secure local services before exposing them via tunnels — use authentication, firewalls, and encryption;</li>
          <li>Monitor your tunnels for unexpected or unauthorized use;</li>
          <li>Keep your API tokens and account credentials secure — do not share them with unauthorized parties;</li>
          <li>Promptly respond to abuse reports or security inquiries from our team.</li>
        </ul>

        <h2>4. Enforcement</h2>
        <p>We may take the following actions in response to AUP violations:</p>
        <ul>
          <li><strong>Warning:</strong> for first-time minor violations, we will notify you and request corrective action;</li>
          <li><strong>Tunnel termination:</strong> individual tunnels violating this policy may be closed immediately;</li>
          <li><strong>Account suspension:</strong> repeated or serious violations will result in temporary account suspension;</li>
          <li><strong>Account termination:</strong> severe violations (malware, CSAM, attacks) result in permanent ban without prior notice;</li>
          <li><strong>Legal action:</strong> we reserve the right to report illegal activities to law enforcement and cooperate with investigations.</li>
        </ul>
        <p>
          We may monitor connection metadata (IP addresses, bandwidth, port usage) to detect abuse patterns.
          We do <strong>not</strong> inspect tunnel payload content — see our
          <RouterLink to="/terms" class="text-primary hover:underline">Terms of Service</RouterLink> section 8.4.
        </p>

        <h2>5. Reporting Abuse</h2>
        <p>
          If you believe an fxTunnel user is violating this policy, please report it to us.
          See our <RouterLink to="/abuse" class="text-primary hover:underline">Abuse Contact</RouterLink> page
          for reporting instructions.
        </p>

        <h2>6. Changes to This Policy</h2>
        <p>
          We may update this AUP from time to time. Material changes will be communicated via the
          Service at least 14 days before taking effect. Continued use of fxTunnel after the effective
          date constitutes acceptance.
        </p>

        <h2>7. Contact</h2>
        <table class="w-full">
          <tbody>
            <tr>
              <td class="font-medium pr-4 py-1">Company:</td>
              <td>Nocodo LTD</td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Email:</td>
              <td><a href="mailto:abuse@fxtun.dev">abuse@fxtun.dev</a></td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Website:</td>
              <td><a href="https://fxtun.dev">fxtun.dev</a></td>
            </tr>
          </tbody>
        </table>
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
