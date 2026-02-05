<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { plansApi, type Plan } from '@/api/client'

const { t } = useI18n()
const router = useRouter()

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)
const plans = ref<Plan[]>([])
const loading = ref(true)

// Check if we're on a RU domain
const isRuDomain = computed(() => {
  const host = window.location.hostname
  return host.endsWith('.ru') || host === 'localhost'
})

// Check if payments are disabled for this domain
const isPaymentsDisabled = computed(() => {
  const host = window.location.hostname
  return host.endsWith('.dev') || host === 'fxtun.dev'
})

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
  if (isRuDomain.value) {
    // Use backend price_rub or fallback to price * 75
    const priceRub = plan.price_rub ?? plan.price * 75
    return `${Math.round(priceRub)} ₽`
  }
  return `$${plan.price}`
}

const sortedPlans = computed(() =>
  [...plans.value].sort((a, b) => a.price - b.price)
)

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
            <ul class="space-y-2 mb-6 text-sm">
              <!-- Tunnels -->
              <li class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span><strong>{{ displayLimit(plan.max_tunnels) }}</strong> {{ t('landing.pricing.tunnels') }}</span>
              </li>

              <!-- Domains (hide when 0) -->
              <li v-if="plan.max_domains !== 0" class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span><strong>{{ displayLimit(plan.max_domains) }}</strong> {{ t('landing.pricing.domains') }}</span>
              </li>

              <!-- Custom Domains (hide when 0) -->
              <li v-if="plan.max_custom_domains !== 0" class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span><strong>{{ displayLimit(plan.max_custom_domains) }}</strong> {{ t('landing.pricing.customDomains') }}</span>
              </li>

              <!-- Tokens -->
              <li class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span><strong>{{ displayLimit(plan.max_tokens) }}</strong> {{ t('landing.pricing.tokens') }}</span>
              </li>

              <!-- Inspector (hide when disabled) -->
              <li v-if="plan.inspector_enabled" class="flex items-start gap-2">
                <svg class="h-4 w-4 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <span>{{ t('landing.pricing.inspector') }}</span>
              </li>
            </ul>

            <!-- CTA Button -->
            <button
              @click="!isPaymentsDisabled && plan.price > 0 ? selectPlan(plan.id) : selectPlan(plan.id)"
              :disabled="isPaymentsDisabled && plan.price > 0"
              :class="[
                'block w-full py-2.5 px-4 rounded-lg text-center text-sm font-medium transition-all duration-300 mt-auto',
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
    </div>
  </section>
</template>
