<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { plansApi, type Plan } from '@/api/client'

const { t, locale } = useI18n()
const router = useRouter()

// Russian pluralization: 1 туннель, 2 туннеля, 5 туннелей
// Uses ;; separator in i18n to avoid vue-i18n pipe interpretation
function plural(key: string, n: number): string {
  const raw = t(key)
  const forms = raw.split(';;').map(s => s.trim())
  if (locale.value === 'ru' && forms.length >= 3) {
    const abs = Math.abs(n) % 100
    const last = abs % 10
    if (abs > 10 && abs < 20) return forms[2]
    if (last === 1) return forms[0]
    if (last >= 2 && last <= 4) return forms[1]
    return forms[2]
  }
  return forms.length >= 2 && n !== 1 ? forms[1] : forms[0]
}

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)
const plans = ref<Plan[]>([])
const loading = ref(true)

const isRuLocale = computed(() => locale.value === 'ru')

// Payments disabled for non-Russian users (no Paddle yet)
const isPaymentsDisabled = computed(() => locale.value !== 'ru')

// Handle plan selection - save redirect and go to login
function selectPlan(planId: number) {
  localStorage.setItem('authRedirect', `/checkout?plan=${planId}`)
  router.push('/login')
}

function displayLimit(val: number): string {
  return val < 0 ? t('landing.pricing.unlimited') : String(val)
}

// Format price using backend-calculated values
function formatPrice(plan: Plan): string {
  if (plan.price === 0) return ''
  if (isRuLocale.value) {
    const priceRub = plan.price_rub ?? plan.price * 75
    return `${Math.round(priceRub)} ₽`
  }
  return `$${plan.price}`
}

const sortedPlans = computed(() =>
  [...plans.value].sort((a, b) => b.price - a.price)
)

const commonFeatures = [
  { key: 'noLimits', icon: 'infinity' },
  { key: 'noTimeout', icon: 'clock' },
  { key: 'protocols', icon: 'signal' },
  { key: 'guiCli', icon: 'desktop' },
  { key: 'subdomains', icon: 'globe' },
  { key: 'tls', icon: 'lock' },
]

onMounted(async () => {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          isVisible.value = true
          observer.disconnect()
        }
      })
    },
    { threshold: 0.15 }
  )
  if (sectionRef.value) {
    observer.observe(sectionRef.value)
  }

  try {
    const resp = await plansApi.listPublic()
    plans.value = resp.data.plans || []
  } catch {
    // silent - section will show empty state
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section id="pricing" ref="sectionRef" class="py-16 md:py-32 relative overflow-hidden">
    <!-- Background -->
    <div class="absolute inset-0 bg-gradient-to-b from-background via-surface/20 to-background" />

    <!-- Grid pattern -->
    <div class="absolute inset-0 opacity-20">
      <div class="absolute inset-0 bg-grid-pattern bg-grid-60" style="mask-image: radial-gradient(ellipse 60% 50% at 50% 50%, black 20%, transparent 70%);" />
    </div>

    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-16">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <span class="text-sm font-medium text-primary">{{ t('landing.pricing.label') }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.pricing.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.pricing.subtitle') }}
        </p>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-16">
        <svg class="h-8 w-8 animate-spin text-muted-foreground" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>

      <!-- Empty state -->
      <div v-else-if="plans.length === 0" class="text-center py-16">
        <p class="text-muted-foreground">{{ t('landing.pricing.noPlans') }}</p>
      </div>

      <!-- Pricing cards -->
      <div
        v-else
        class="grid gap-6 reveal reveal-delay-3"
        :class="[
          { 'visible': isVisible },
          plans.length === 1 ? 'max-w-md mx-auto' : '',
          plans.length === 2 ? 'md:grid-cols-2 max-w-3xl mx-auto' : '',
          plans.length === 3 ? 'md:grid-cols-2 lg:grid-cols-3 max-w-5xl mx-auto' : '',
          plans.length >= 4 ? 'md:grid-cols-2 lg:grid-cols-4 max-w-7xl mx-auto' : '',
        ]"
      >
        <div
          v-for="plan in sortedPlans"
          :key="plan.id"
          class="relative group"
          :class="{ 'md:scale-105 md:z-10': plan.is_recommended }"
        >
          <!-- Recommended badge -->
          <div
            v-if="plan.is_recommended"
            class="absolute -top-4 left-1/2 -translate-x-1/2 z-20"
          >
            <span class="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-full bg-primary text-primary-foreground text-sm font-medium shadow-lg shadow-primary/25">
              <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10.868 2.884c-.321-.772-1.415-.772-1.736 0l-1.83 4.401-4.753.381c-.833.067-1.171 1.107-.536 1.651l3.62 3.102-1.106 4.637c-.194.813.691 1.456 1.405 1.02L10 15.591l4.069 2.485c.713.436 1.598-.207 1.404-1.02l-1.106-4.637 3.62-3.102c.635-.544.297-1.584-.536-1.65l-4.752-.382-1.831-4.401z" clip-rule="evenodd" />
              </svg>
              {{ t('landing.pricing.popular') }}
            </span>
          </div>

          <!-- Card -->
          <div
            :class="[
              'h-full rounded-2xl p-6 transition-all duration-300 flex flex-col',
              plan.is_recommended
                ? 'bg-surface border-2 border-primary shadow-xl shadow-primary/10 group-hover:shadow-2xl group-hover:shadow-primary/20'
                : 'bg-surface/50 border border-border group-hover:border-primary/30 group-hover:shadow-lg',
            ]"
          >
            <!-- Plan name -->
            <h3 class="text-lg font-display font-semibold mb-2">{{ plan.name }}</h3>

            <!-- Price -->
            <div class="mb-4">
              <span v-if="plan.price === 0" class="text-3xl font-display font-bold">
                {{ t('landing.pricing.free') }}
              </span>
              <template v-else>
                <span class="text-3xl font-display font-bold">{{ formatPrice(plan) }}</span>
                <span class="text-sm text-muted-foreground">{{ t('landing.pricing.perMonth') }}</span>
              </template>
            </div>

            <!-- Divider -->
            <div class="h-px bg-border mb-4" />

            <!-- Features list -->
            <ul class="space-y-2.5 mb-6 text-sm flex-1">
              <!-- Tunnels -->
              <li class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span><strong>{{ displayLimit(plan.max_tunnels) }}</strong> {{ plural('landing.pricing.tunnels', plan.max_tunnels) }}</span>
              </li>

              <!-- Any subdomain (show on all plans) -->
              <li class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span>{{ t('landing.pricing.anySubdomain') }}</span>
              </li>

              <!-- Reserved domains -->
              <li class="flex items-start gap-2">
                <svg v-if="plan.max_domains > 0" class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <svg v-else class="h-4 w-4 text-muted-foreground/40 flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H4.75A.75.75 0 014 10z" clip-rule="evenodd" />
                </svg>
                <span :class="plan.max_domains === 0 ? 'text-muted-foreground/50' : ''">
                  <template v-if="plan.max_domains > 0">
                    <strong>{{ displayLimit(plan.max_domains) }}</strong> {{ plural('landing.pricing.domains', plan.max_domains) }}
                  </template>
                  <template v-else>
                    0 {{ plural('landing.pricing.domains', 0) }}
                  </template>
                </span>
              </li>

              <!-- Custom Domains -->
              <li class="flex items-start gap-2">
                <svg v-if="plan.max_custom_domains > 0" class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <svg v-else class="h-4 w-4 text-muted-foreground/40 flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H4.75A.75.75 0 014 10z" clip-rule="evenodd" />
                </svg>
                <span :class="plan.max_custom_domains === 0 ? 'text-muted-foreground/50' : ''">
                  <template v-if="plan.max_custom_domains > 0">
                    <strong>{{ displayLimit(plan.max_custom_domains) }}</strong> {{ plural('landing.pricing.customDomains', plan.max_custom_domains) }}
                  </template>
                  <template v-else>
                    0 {{ plural('landing.pricing.customDomains', 0) }}
                  </template>
                </span>
              </li>

              <!-- Tokens -->
              <li class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span><strong>{{ displayLimit(plan.max_tokens) }}</strong> {{ plural('landing.pricing.tokens', plan.max_tokens) }}</span>
              </li>

              <!-- Inspector -->
              <li class="flex items-start gap-2">
                <svg v-if="plan.inspector_enabled" class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <svg v-else class="h-4 w-4 text-muted-foreground/40 flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H4.75A.75.75 0 014 10z" clip-rule="evenodd" />
                </svg>
                <span :class="!plan.inspector_enabled ? 'text-muted-foreground/50' : ''">
                  {{ plan.inspector_enabled ? t('landing.pricing.inspectorUnlimited') : t('landing.pricing.inspector') }}
                </span>
              </li>
            </ul>

            <!-- CTA Button -->
            <button
              @click="selectPlan(plan.id)"
              :disabled="isPaymentsDisabled && plan.price > 0"
              :class="[
                'block w-full py-2.5 px-4 rounded-lg text-center text-sm font-medium transition-all duration-300',
                isPaymentsDisabled && plan.price > 0
                  ? 'bg-muted text-muted-foreground cursor-not-allowed'
                  : plan.is_recommended
                    ? 'bg-primary text-primary-foreground hover:bg-primary/90 shadow-lg shadow-primary/25 hover:shadow-xl hover:shadow-primary/30 cursor-pointer'
                    : 'bg-surface border border-border hover:border-primary/50 hover:bg-primary/5 cursor-pointer',
              ]"
            >
              <template v-if="isPaymentsDisabled && plan.price > 0">
                {{ t('landing.pricing.paymentsComingSoon') }}
              </template>
              <template v-else>
                {{ t('landing.pricing.selectPlan') }}
              </template>
            </button>
          </div>
        </div>
      </div>

      <!-- Common features -->
      <div
        v-if="plans.length > 0"
        class="mt-16 max-w-4xl mx-auto reveal reveal-delay-5"
        :class="{ 'visible': isVisible }"
      >
        <p class="text-center text-sm font-medium text-muted-foreground mb-6">
          {{ t('landing.pricing.common.title') }}
        </p>
        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <div
            v-for="feat in commonFeatures"
            :key="feat.key"
            class="flex flex-col items-center gap-2 p-3 rounded-xl bg-surface/50 border border-border text-center"
          >
            <div class="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
              <!-- infinity -->
              <svg v-if="feat.icon === 'infinity'" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
              </svg>
              <!-- clock -->
              <svg v-else-if="feat.icon === 'clock'" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
              </svg>
              <!-- signal -->
              <svg v-else-if="feat.icon === 'signal'" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8.288 15.038a5.25 5.25 0 0 1 7.424 0M5.106 11.856c3.807-3.808 9.98-3.808 13.788 0M1.924 8.674c5.565-5.565 14.587-5.565 20.152 0M12.53 18.22l-.53.53-.53-.53a.75.75 0 0 1 1.06 0z" />
              </svg>
              <!-- desktop -->
              <svg v-else-if="feat.icon === 'desktop'" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25m18 0A2.25 2.25 0 0 0 18.75 3H5.25A2.25 2.25 0 0 0 3 5.25m18 0V12a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 12V5.25" />
              </svg>
              <!-- globe -->
              <svg v-else-if="feat.icon === 'globe'" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
              </svg>
              <!-- lock -->
              <svg v-else-if="feat.icon === 'lock'" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z" />
              </svg>
            </div>
            <span class="text-xs text-muted-foreground leading-tight">{{ t(`landing.pricing.common.${feat.key}`) }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
