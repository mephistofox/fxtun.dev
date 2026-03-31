<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute } from 'vue-router'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema, useFaqSchema, useProductSchema } from '@/composables/useStructuredData'
import { getDomainLocale } from '@/i18n'
import PricingSection from '@/components/landing/PricingSection.vue'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'

const { t, tm, locale } = useI18n()
const route = useRoute()

useSeo({ titleKey: 'seo.pricing.title', descriptionKey: 'seo.pricing.description' })

useSubpageSchema({
  path: '/pricing',
  name: t('seo.pricing.title'),
  description: t('seo.pricing.description'),
  breadcrumbs: [
    { name: t('landing.pricing.title'), path: '/pricing' },
  ],
})

useProductSchema()

const effectiveLocale = computed(() => {
  if (route.meta.forcedLocale) return route.meta.forcedLocale as string
  if (import.meta.env.SSR) return locale.value
  return getDomainLocale() ?? locale.value
})
const isRuDomain = computed(() => effectiveLocale.value === 'ru')

// Feature matrix data
const matrixFeatures = [
  'tunnels', 'anySubdomain', 'reservedSubdomains', 'customDomains',
  'accessKeys', 'inspector', 'httpTunnels', 'tcpTunnels', 'udpTunnels',
  'autoReconnect', 'desktopApp', 'noRequestLimits', 'noBandwidthLimits',
  'noSessionTimeout', 'selfHostable',
] as const

// Plan column values for the matrix
const planColumns = computed(() => [
  {
    name: 'Free',
    slug: 'free',
    values: {
      tunnels: '1',
      anySubdomain: true,
      reservedSubdomains: '0',
      customDomains: '0',
      accessKeys: '1',
      inspector: false,
      httpTunnels: true,
      tcpTunnels: true,
      udpTunnels: true,
      autoReconnect: true,
      desktopApp: true,
      noRequestLimits: true,
      noBandwidthLimits: true,
      noSessionTimeout: true,
      selfHostable: true,
    },
  },
  {
    name: 'Starter',
    slug: 'starter',
    price: isRuDomain.value ? '200 ₽' : '$2.50',
    values: {
      tunnels: '3',
      anySubdomain: true,
      reservedSubdomains: '1',
      customDomains: '1',
      accessKeys: '1',
      inspector: true,
      httpTunnels: true,
      tcpTunnels: true,
      udpTunnels: true,
      autoReconnect: true,
      desktopApp: true,
      noRequestLimits: true,
      noBandwidthLimits: true,
      noSessionTimeout: true,
      selfHostable: true,
    },
  },
  {
    name: 'Base',
    slug: 'base',
    price: isRuDomain.value ? '400 ₽' : '$5',
    values: {
      tunnels: '5',
      anySubdomain: true,
      reservedSubdomains: '5',
      customDomains: '1',
      accessKeys: '5',
      inspector: true,
      httpTunnels: true,
      tcpTunnels: true,
      udpTunnels: true,
      autoReconnect: true,
      desktopApp: true,
      noRequestLimits: true,
      noBandwidthLimits: true,
      noSessionTimeout: true,
      selfHostable: true,
    },
  },
  {
    name: 'Pro',
    slug: 'pro',
    price: isRuDomain.value ? '600 ₽' : '$7.50',
    values: {
      tunnels: '15',
      anySubdomain: true,
      reservedSubdomains: '15',
      customDomains: '5',
      accessKeys: '10',
      inspector: true,
      httpTunnels: true,
      tcpTunnels: true,
      udpTunnels: true,
      autoReconnect: true,
      desktopApp: true,
      noRequestLimits: true,
      noBandwidthLimits: true,
      noSessionTimeout: true,
      selfHostable: true,
    },
  },
])

// FAQ
interface FaqItem {
  q: string
  a: string
}

const faqItems = computed(() => tm('pricingFaq.items') as FaqItem[])
useFaqSchema(faqItems.value.map(item => ({ question: item.q, answer: item.a })), '-pricing')

const openFaqIndex = ref<number | null>(null)

function toggleFaq(index: number) {
  openFaqIndex.value = openFaqIndex.value === index ? null : index
}
</script>

<template>
  <div class="pricing-page">
    <!-- Navbar -->
    <nav class="pricing-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="pricing-nav-brand">
          <div class="pricing-nav-logo">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="pricing-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="pricing-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <!-- Pricing section (reuse landing component) -->
    <main class="pt-16">
      <Breadcrumbs :items="[{ name: t('landing.pricing.label'), path: '/pricing' }]" />
      <p class="container mx-auto px-4 text-xs text-muted-foreground/60">
        {{ t('common.lastUpdated', { date: t('common.updateDateMar2026') }) }}
      </p>
      <PricingSection compact />

      <!-- Feature Comparison Matrix -->
      <section class="py-16 md:py-24">
        <div class="container mx-auto px-4">
          <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-12">
            {{ t('pricingPage.matrixTitle') }}
          </h2>
          <div class="max-w-5xl mx-auto overflow-x-auto">
            <table class="matrix-table w-full">
              <thead>
                <tr>
                  <th class="text-left">{{ t('compare.feature') }}</th>
                  <th
                    v-for="plan in planColumns"
                    :key="plan.slug"
                    class="text-center"
                    :class="{ 'matrix-recommended': plan.slug === 'base' }"
                  >
                    <div class="font-semibold">{{ plan.name }}</div>
                    <div v-if="plan.price" class="text-xs text-muted-foreground font-normal mt-0.5">
                      {{ plan.price }}{{ t('landing.pricing.perMonth') }}
                    </div>
                    <div v-else class="text-xs text-muted-foreground font-normal mt-0.5">
                      {{ t('landing.pricing.free') }}
                    </div>
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="feature in matrixFeatures" :key="feature">
                  <td class="font-medium text-sm">{{ t(`pricingPage.matrixFeatures.${feature}`) }}</td>
                  <td
                    v-for="plan in planColumns"
                    :key="plan.slug"
                    class="text-center"
                    :class="{ 'matrix-recommended': plan.slug === 'base' }"
                  >
                    <template v-if="typeof plan.values[feature] === 'boolean'">
                      <svg
                        v-if="plan.values[feature]"
                        class="h-5 w-5 mx-auto text-primary"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                      <svg
                        v-else
                        class="h-5 w-5 mx-auto text-muted-foreground/40"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </template>
                    <template v-else>
                      <span class="text-sm">{{ plan.values[feature] }}</span>
                    </template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <!-- Pricing FAQ -->
      <section class="py-16 md:py-24">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-12">
            {{ t('pricingFaq.title') }}
          </h2>
          <div>
            <div
              v-for="(item, index) in faqItems"
              :key="index"
              class="border-b border-border"
              :class="{ 'border-primary/20': openFaqIndex === index }"
            >
              <button
                @click="toggleFaq(index)"
                class="w-full flex items-center justify-between py-5 text-left group"
              >
                <span
                  class="text-base font-medium pr-8 group-hover:text-primary transition-colors"
                  :class="{ 'text-primary': openFaqIndex === index }"
                >
                  {{ item.q }}
                </span>
                <svg
                  class="h-5 w-5 flex-shrink-0 text-muted-foreground transition-transform duration-300"
                  :class="{ 'rotate-180 text-primary': openFaqIndex === index }"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="1.5"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
                </svg>
              </button>
              <Transition name="faq-expand">
                <div v-if="openFaqIndex === index" class="faq-answer">
                  <p class="pb-5 text-muted-foreground leading-relaxed">
                    {{ item.a }}
                  </p>
                </div>
              </Transition>
            </div>
          </div>
        </div>
      </section>

      <!-- Payment Methods -->
      <section class="py-16 md:py-24">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-8">
            {{ t('pricingPage.paymentTitle') }}
          </h2>
          <div class="grid md:grid-cols-2 gap-6">
            <!-- Russia / YooKassa -->
            <div class="payment-card">
              <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center">
                  <svg aria-hidden="true" class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z" />
                  </svg>
                </div>
                <span class="font-semibold">YooKassa</span>
              </div>
              <p class="text-sm text-muted-foreground">{{ t('pricingPage.paymentRussia') }}</p>
            </div>

            <!-- International / Creem -->
            <div class="payment-card">
              <div class="flex items-center gap-3 mb-3">
                <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center">
                  <svg aria-hidden="true" class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" />
                  </svg>
                </div>
                <span class="font-semibold">Creem</span>
              </div>
              <p class="text-sm text-muted-foreground">{{ t('pricingPage.paymentInternational') }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Which plan is right for you? -->
      <section class="py-16 md:py-24">
        <div class="container mx-auto px-4">
          <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-12">
            {{ t('pricingPage.whoIsItForTitle') }}
          </h2>
          <div class="grid md:grid-cols-2 gap-6 max-w-5xl mx-auto">
            <div
              v-for="tier in ['free', 'starter', 'base', 'pro']"
              :key="tier"
              class="plan-card"
              :class="{ 'plan-card--recommended': tier === 'base' }"
            >
              <h3 class="text-lg font-display font-semibold mb-3">
                {{ t(`pricingPage.whoIsItFor.${tier}.title`) }}
              </h3>
              <p class="text-sm text-muted-foreground leading-relaxed">
                {{ t(`pricingPage.whoIsItFor.${tier}.text`) }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- Subdomains explanation -->
      <section class="py-16 md:py-24">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-8">
            {{ t('pricingPage.subdomainTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed">
            {{ t('pricingPage.subdomainText') }}
          </p>
        </div>
      </section>

      <!-- Common features -->
      <section class="py-16 md:py-24">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-8">
            {{ t('pricingPage.commonFeaturesTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed">
            {{ t('pricingPage.commonFeaturesText') }}
          </p>
        </div>
      </section>
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

/* Feature matrix table */
.matrix-table {
  border-collapse: separate;
  border-spacing: 0;
}

.matrix-table th {
  @apply py-4 px-3;
  border-bottom: 2px solid hsl(var(--border));
  color: hsl(var(--foreground));
}

.matrix-table td {
  @apply py-3 px-3;
  border-bottom: 1px solid hsl(var(--border) / 0.5);
  color: hsl(var(--foreground));
}

.matrix-table tbody tr:hover {
  background: hsl(var(--surface) / 0.3);
}

.matrix-recommended {
  background: hsl(var(--primary) / 0.05);
}

/* Plan cards */
.plan-card {
  @apply p-6 rounded-xl;
  background: hsl(var(--surface) / 0.5);
  border: 1px solid hsl(var(--border) / 0.5);
  transition: border-color 0.2s ease;
}

.plan-card:hover {
  border-color: hsl(var(--primary) / 0.3);
}

.plan-card--recommended {
  border-color: hsl(var(--primary) / 0.4);
  background: hsl(var(--primary) / 0.05);
}

/* Payment cards */
.payment-card {
  @apply p-6 rounded-xl;
  background: hsl(var(--surface) / 0.5);
  border: 1px solid hsl(var(--border) / 0.5);
}

/* FAQ transitions */
.faq-expand-enter-active,
.faq-expand-leave-active {
  transition: all 0.3s ease;
  overflow: hidden;
}

.faq-expand-enter-from,
.faq-expand-leave-to {
  opacity: 0;
  max-height: 0;
}

.faq-expand-enter-to,
.faq-expand-leave-from {
  opacity: 1;
  max-height: 200px;
}
</style>
