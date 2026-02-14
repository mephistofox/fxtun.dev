<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import { useSeo } from '@/composables/useSeo'

const themeStore = useThemeStore()
const { t } = useI18n()

useSeo({ titleKey: 'seo.terms.title', descriptionKey: 'seo.terms.description' })

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

const lastUpdated = '13.02.2026'
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
        <h1 class="text-3xl font-bold mb-4">{{ t('legal.termsTitle') }}</h1>
        <span class="text-sm text-muted-foreground">
          {{ t('legal.lastUpdated') }}: {{ lastUpdated }}
        </span>
      </div>

      <!-- Content -->
      <div class="prose prose-neutral dark:prose-invert max-w-none">

        <h2>1. Introduction</h2>
        <p>
          These Terms of Service ("Terms") govern your access to and use of the fxTunnel service
          ("Service"), including the website at <a href="https://fxtun.dev">fxtun.dev</a>,
          desktop applications, command-line tools, and all related APIs.
        </p>
        <p>
          The Service is operated by <strong>Nocodo LTD</strong>, a company incorporated and
          registered in the Republic of Cyprus (hereinafter "Company", "we", "us", or "our").
          Company website: <a href="https://nocodo.tech">nocodo.tech</a>.
        </p>
        <p>
          By creating an account or using the Service, you ("User", "you", or "your") agree to
          be bound by these Terms. If you do not agree, do not use the Service.
        </p>

        <h2>2. Definitions</h2>
        <ul>
          <li><strong>"Tunnel"</strong> — a secure connection that exposes a local service running on your device to the public internet via the fxTunnel infrastructure.</li>
          <li><strong>"Subdomain"</strong> — a unique hostname (e.g., <code>your-app.fxtun.dev</code>) assigned to an HTTP tunnel.</li>
          <li><strong>"Token"</strong> — an API credential used to authenticate tunnel connections.</li>
          <li><strong>"Plan"</strong> — a subscription tier defining the scope of Service available to you (Free, Base, or Pro).</li>
          <li><strong>"Traffic Inspector"</strong> — a built-in tool for monitoring HTTP requests and responses passing through your tunnels.</li>
        </ul>

        <h2>3. Description of the Service</h2>
        <p>
          fxTunnel provides secure reverse tunneling that allows you to expose local HTTP, TCP,
          and UDP services to the internet. The Service includes:
        </p>
        <ul>
          <li>HTTP tunneling with custom or random subdomains under <code>fxtun.dev</code>;</li>
          <li>TCP port forwarding with dynamically allocated public ports;</li>
          <li>UDP port forwarding;</li>
          <li>A web dashboard for managing tunnels, tokens, and reserved subdomains;</li>
          <li>Desktop GUI client for Windows, macOS, and Linux;</li>
          <li>Command-line client for all major platforms;</li>
          <li>Real-time traffic inspection for HTTP tunnels.</li>
        </ul>

        <h2>4. Account Registration</h2>
        <p>
          To use the Service, you must create an account. You agree to provide accurate and
          complete information during registration and to keep your account credentials secure.
        </p>
        <p>
          You are solely responsible for all activity that occurs under your account. You must
          notify us immediately at
          <a href="mailto:support@nocodo.tech">support@nocodo.tech</a> if you suspect
          unauthorized access to your account.
        </p>

        <h2>5. Free and Paid Plans</h2>
        <h3>5.1. Free Plan</h3>
        <p>
          The Free plan provides up to 3 concurrent tunnels with any available subdomain,
          no bandwidth limits, and no session timeout. The Free plan is available indefinitely
          and does not require a credit card.
        </p>
        <h3>5.2. Paid Plans</h3>
        <p>
          Paid plans (Base and Pro) offer additional capacity, reserved subdomains, custom domains,
          and priority features. Current pricing is available at
          <a href="https://fxtun.dev/#pricing">fxtun.dev/#pricing</a>.
        </p>
        <p>
          Subscriptions are billed monthly. Payment is processed through third-party payment
          providers. By subscribing, you authorize recurring charges to your selected payment
          method until you cancel.
        </p>

        <h2>6. Acceptable Use</h2>
        <p>You agree not to use the Service to:</p>
        <ul>
          <li>Violate any applicable law, regulation, or third-party rights;</li>
          <li>Distribute malware, phishing pages, or any form of malicious content;</li>
          <li>Conduct denial-of-service attacks or port scanning against third parties;</li>
          <li>Host or distribute illegal content, including but not limited to CSAM;</li>
          <li>Circumvent access controls or authentication of other systems;</li>
          <li>Relay spam or unsolicited bulk messages;</li>
          <li>Interfere with or disrupt the integrity of the Service or its infrastructure;</li>
          <li>Resell, redistribute, or sublicense access to the Service without prior written consent.</li>
        </ul>
        <p>
          We reserve the right to suspend or terminate your account immediately, without prior
          notice, if we reasonably determine that you have violated these terms.
        </p>

        <h2>7. Intellectual Property</h2>
        <p>
          The Service, including its software, design, logos, documentation, and all related
          intellectual property, is owned by Nocodo LTD or its licensors. These Terms do not
          grant you any right, title, or interest in the Service except for the limited right
          to use it as described herein.
        </p>
        <p>
          The open-source components of fxTunnel are licensed under their respective licenses
          as specified in the source code repositories.
        </p>

        <h2>8. Privacy and Data Processing</h2>
        <h3>8.1. Data We Collect</h3>
        <ul>
          <li><strong>Account data:</strong> email address, hashed password, account preferences;</li>
          <li><strong>Connection metadata:</strong> IP addresses, user agent strings, tunnel session timestamps;</li>
          <li><strong>Usage data:</strong> tunnel count, bandwidth statistics, feature usage;</li>
          <li><strong>Payment data:</strong> processed and stored exclusively by our payment providers — we do not store card details.</li>
        </ul>
        <h3>8.2. How We Use Your Data</h3>
        <ul>
          <li>To provide and maintain the Service;</li>
          <li>To authenticate your identity and manage your account;</li>
          <li>To process payments and manage subscriptions;</li>
          <li>To detect and prevent abuse and ensure platform security;</li>
          <li>To communicate essential service updates.</li>
        </ul>
        <h3>8.3. Data Retention</h3>
        <p>
          We retain your account data for the duration of your account plus 12 months after
          deletion. Connection metadata is retained for up to 90 days. You may request full
          data deletion by contacting <a href="mailto:support@nocodo.tech">support@nocodo.tech</a>.
        </p>
        <h3>8.4. Traffic Content</h3>
        <p>
          We do <strong>not</strong> inspect, log, or store the content of traffic passing through
          your tunnels. The Traffic Inspector feature operates exclusively in your browser session
          and on your device — tunnel payload data is not stored on our servers.
        </p>
        <h3>8.5. Third-Party Processors</h3>
        <p>We may share data with the following categories of third-party processors:</p>
        <ul>
          <li>Payment processors (for billing and subscription management);</li>
          <li>Infrastructure providers (for hosting and content delivery);</li>
          <li>Law enforcement (when required by applicable law or court order).</li>
        </ul>
        <h3>8.6. Cookies</h3>
        <p>
          The Service uses strictly necessary cookies for authentication and session management.
          We do not use tracking or advertising cookies.
        </p>

        <h2>9. Cancellation and Refunds</h2>
        <p>
          You may cancel your subscription at any time from your account dashboard. Upon
          cancellation, you retain access to paid features until the end of the current billing
          period.
        </p>
        <ul>
          <li><strong>Within 7 days of first payment:</strong> full refund, no questions asked.</li>
          <li><strong>After 7 days:</strong> pro-rata refund for unused full days remaining in the billing period.</li>
        </ul>
        <p>
          Refunds are issued to the original payment method within 14 business days.
          To request a refund, contact <a href="mailto:support@nocodo.tech">support@nocodo.tech</a>.
        </p>

        <h2>10. Service Availability and SLA</h2>
        <p>
          We make commercially reasonable efforts to maintain 99.9% uptime for the tunneling
          infrastructure. However, the Service is provided on an "as is" and "as available" basis.
        </p>
        <p>We are not liable for interruptions caused by:</p>
        <ul>
          <li>Scheduled maintenance (announced at least 24 hours in advance);</li>
          <li>Force majeure events, including natural disasters, wars, pandemics, or government actions;</li>
          <li>Third-party service failures (DNS providers, hosting providers, payment processors);</li>
          <li>Your network connectivity or local environment issues.</li>
        </ul>

        <h2>11. Limitation of Liability</h2>
        <p>
          To the maximum extent permitted by applicable law, Nocodo LTD shall not be liable
          for any indirect, incidental, special, consequential, or punitive damages, including
          but not limited to loss of profits, data, business opportunities, or goodwill,
          arising out of or related to your use of the Service.
        </p>
        <p>
          Our total aggregate liability for any claims arising from or related to the Service
          shall not exceed the amount you paid to us in the 12 months preceding the claim.
        </p>

        <h2>12. Indemnification</h2>
        <p>
          You agree to indemnify and hold harmless Nocodo LTD, its officers, directors, employees,
          and agents from any claims, damages, losses, liabilities, and expenses (including
          reasonable legal fees) arising from your use of the Service or violation of these Terms.
        </p>

        <h2>13. Modifications to the Terms</h2>
        <p>
          We may update these Terms from time to time. Material changes will be communicated
          via email or a prominent notice on the Service at least 30 days before they take effect.
          Continued use of the Service after the effective date constitutes acceptance of the
          updated Terms.
        </p>

        <h2>14. Termination</h2>
        <p>
          Either party may terminate this agreement at any time. You may do so by deleting
          your account or contacting support. We may terminate or suspend your access
          immediately for violation of these Terms, without prior notice or liability.
        </p>
        <p>
          Upon termination, your right to use the Service ceases immediately. Sections
          concerning intellectual property, limitation of liability, indemnification,
          and governing law survive termination.
        </p>

        <h2>15. Governing Law and Dispute Resolution</h2>
        <p>
          These Terms are governed by and construed in accordance with the laws of the
          Republic of Cyprus, without regard to its conflict of law provisions.
        </p>
        <p>
          Any disputes arising from or relating to these Terms or the Service shall first be
          attempted to be resolved through good-faith negotiation. If unresolved within 30 days,
          disputes shall be submitted to the exclusive jurisdiction of the courts of the
          Republic of Cyprus.
        </p>

        <h2>16. Severability</h2>
        <p>
          If any provision of these Terms is held to be invalid or unenforceable, the remaining
          provisions shall continue in full force and effect.
        </p>

        <h2>17. Contact Information</h2>
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
            <tr>
              <td class="font-medium pr-4 py-1">Email:</td>
              <td><a href="mailto:support@nocodo.tech">support@nocodo.tech</a></td>
            </tr>
            <tr>
              <td class="font-medium pr-4 py-1">Service:</td>
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
