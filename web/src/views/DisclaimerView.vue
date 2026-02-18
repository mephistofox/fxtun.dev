<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import { useSeo } from '@/composables/useSeo'

const themeStore = useThemeStore()
const { t } = useI18n()

useSeo({ titleKey: 'seo.disclaimer.title', descriptionKey: 'seo.disclaimer.description' })

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
        <h1 class="text-3xl font-bold mb-4">{{ t('legal.disclaimerTitle') }}</h1>
        <span class="text-sm text-muted-foreground">
          {{ t('legal.lastUpdated') }}: {{ lastUpdated }}
        </span>
      </div>

      <!-- Warning banner -->
      <div class="mb-8 p-4 rounded-lg border border-yellow-500/30 bg-yellow-500/5">
        <div class="flex items-start gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-yellow-500 shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
            <line x1="12" y1="9" x2="12" y2="13" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
          <p class="text-sm text-foreground leading-relaxed">
            {{ t('legal.disclaimerBanner') }}
          </p>
        </div>
      </div>

      <!-- Content -->
      <div class="prose prose-neutral dark:prose-invert max-w-none">

        <h2>1. Nature of the Service</h2>
        <p>
          fxtun is a reverse tunneling platform that exposes local network services to the public
          internet via HTTP subdomains, TCP ports, and UDP ports. By using this service, you
          acknowledge that you are making local services accessible from anywhere on the internet.
        </p>
        <p>
          This document describes the inherent risks of TCP and UDP tunneling and your responsibilities
          when using these features.
        </p>

        <h2>2. TCP Tunneling Risks</h2>
        <p>
          TCP tunnels forward raw TCP connections from a dynamically assigned public port to your
          local service. This means:
        </p>
        <ul>
          <li><strong>Full network exposure:</strong> any device on the internet can connect to your
            service via the assigned public port. There is no built-in authentication or access control
            at the tunnel level;</li>
          <li><strong>Protocol transparency:</strong> fxtun forwards TCP traffic as-is without
            inspection, filtering, or modification. If your local service has vulnerabilities, they
            become remotely exploitable;</li>
          <li><strong>Common dangerous services:</strong> exposing SSH (port 22), databases
            (MySQL 3306, PostgreSQL 5432, Redis 6379, MongoDB 27017), RDP (3389), or administrative
            interfaces without proper authentication is extremely risky;</li>
          <li><strong>Brute-force attacks:</strong> publicly exposed services will receive automated
            login attempts within minutes. Weak or default credentials will be compromised;</li>
          <li><strong>Data interception:</strong> TCP connections between the fxtun server and
            external clients are unencrypted unless your service uses TLS. Sensitive data may be
            intercepted in transit.</li>
        </ul>

        <h2>3. UDP Tunneling Risks</h2>
        <p>
          UDP tunnels forward datagrams from a public port to your local UDP service. Additional risks
          include:
        </p>
        <ul>
          <li><strong>Amplification attacks:</strong> UDP services (DNS, NTP, memcached, SSDP) can be
            abused as amplification vectors in DDoS attacks. If your service responds with larger
            payloads than received, it may be exploited;</li>
          <li><strong>Spoofed source addresses:</strong> UDP has no built-in connection state.
            Attackers can send packets with forged source addresses, potentially making your service
            participate in reflection attacks;</li>
          <li><strong>No delivery guarantees:</strong> UDP provides no ordering, retransmission, or
            congestion control. Sensitive applications must handle packet loss at the application level;</li>
          <li><strong>Stateless exposure:</strong> without connection tracking, any party can send
            datagrams to your service at any time without an established session.</li>
        </ul>

        <h2>4. Shared Risks (TCP & UDP)</h2>
        <ul>
          <li><strong>Port scanning:</strong> your assigned public port will be discovered by automated
            scanners (Shodan, Censys, Masscan). Expect reconnaissance traffic within hours;</li>
          <li><strong>No network isolation:</strong> the tunnel bypasses your local firewall, NAT, and
            network policies. Services that were safe behind your network perimeter become directly
            accessible;</li>
          <li><strong>Session persistence:</strong> tunnels remain active until explicitly closed. A
            forgotten tunnel keeps your service exposed indefinitely;</li>
          <li><strong>Shared infrastructure:</strong> fxtun is a shared platform. While we isolate
            tunnel traffic, the public IP addresses and ports are shared infrastructure;</li>
          <li><strong>Service availability:</strong> tunnel connectivity depends on fxtun
            infrastructure availability. Do not rely on tunnels for production or
            mission-critical workloads.</li>
        </ul>

        <h2>5. Recommended Security Measures</h2>
        <p>Before exposing any service via TCP or UDP tunnels, we strongly recommend:</p>
        <ul>
          <li><strong>Authentication:</strong> ensure your service requires strong authentication.
            Never expose services with default or empty credentials;</li>
          <li><strong>Encryption:</strong> use TLS/SSL for TCP services whenever possible.
            For UDP, use DTLS or application-level encryption;</li>
          <li><strong>Firewall rules:</strong> configure your local service to restrict access
            by IP address if possible, even when tunneled;</li>
          <li><strong>Monitoring:</strong> actively monitor tunnel connections and traffic. Use the
            fxtun dashboard to track active tunnels;</li>
          <li><strong>Minimal exposure:</strong> only expose the specific service and port needed.
            Close tunnels when no longer in use;</li>
          <li><strong>Rate limiting:</strong> configure rate limits on your local service to mitigate
            brute-force and abuse attempts;</li>
          <li><strong>Regular updates:</strong> keep exposed services patched and up to date.</li>
        </ul>

        <h2>6. Disclaimer of Liability</h2>
        <p>
          fxtun provides tunneling infrastructure on an "as is" and "as available" basis.
          We do not:
        </p>
        <ul>
          <li>Filter, inspect, or validate traffic passing through TCP and UDP tunnels;</li>
          <li>Provide intrusion detection, intrusion prevention, or firewall services;</li>
          <li>Guarantee protection against attacks, unauthorized access, or data breaches;</li>
          <li>Accept responsibility for the security configuration of your local services.</li>
        </ul>
        <p>
          <strong>You are solely responsible for the security of services you expose through fxtun
          tunnels.</strong> By using TCP or UDP tunneling features, you acknowledge these risks and
          accept full responsibility for any consequences, including but not limited to unauthorized
          access, data loss, service disruption, or third-party claims.
        </p>
        <p>
          For full liability terms, see
          <RouterLink to="/terms" class="text-primary hover:underline">Terms of Service</RouterLink>
          sections 11 and 12.
        </p>

        <h2>7. Abuse Reporting</h2>
        <p>
          If you observe suspicious or malicious activity originating from fxtun infrastructure,
          please report it via our
          <RouterLink to="/abuse" class="text-primary hover:underline">Abuse Contact</RouterLink> page.
        </p>
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
