<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { plansApi, subscriptionApi, type Plan } from '@/api/client'

const route = useRoute()
const { t, locale } = useI18n()

const plans = ref<Plan[]>([])
const selectedPlanId = ref<number | null>(null)
const recurring = ref(true)
const loading = ref(false)
const error = ref('')
const submitting = ref(false)

const selectedPlan = computed(() => {
  return plans.value.find(p => p.id === selectedPlanId.value) || null
})

async function loadPlans() {
  loading.value = true
  try {
    const response = await plansApi.listPublic()
    plans.value = response.data.plans.filter(p => p.price > 0)

    // Pre-select plan from query
    const planId = Number(route.query.plan)
    if (planId && plans.value.some(p => p.id === planId)) {
      selectedPlanId.value = planId
    } else if (plans.value.length > 0) {
      // Select recommended or first
      const recommended = plans.value.find(p => p.is_recommended)
      selectedPlanId.value = recommended?.id || plans.value[0].id
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('checkout.failedToLoadPlans')
  } finally {
    loading.value = false
  }
}

async function handleCheckout() {
  if (!selectedPlanId.value) return

  submitting.value = true
  error.value = ''
  try {
    const response = await subscriptionApi.checkout(selectedPlanId.value, recurring.value)
    // Redirect to Robokassa
    window.location.href = response.data.payment_url
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('checkout.paymentFailed')
    submitting.value = false
  }
}

// Format price using backend-calculated price_rub
function formatPrice(plan: Plan) {
  if (locale.value === 'ru') {
    // Use backend price_rub or fallback to price * 75
    const priceRub = plan.price_rub ?? plan.price * 75
    return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB' }).format(priceRub)
  }
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(plan.price)
}

function formatLimit(value: number) {
  return value < 0 ? '\u221E' : value.toString()
}

onMounted(() => {
  loadPlans()
})
</script>

<template>
  <Layout>
    <div class="max-w-4xl mx-auto space-y-6">
      <div class="text-center mb-8">
        <h1 class="text-2xl font-bold">{{ t('checkout.title') }}</h1>
        <p class="text-muted-foreground mt-2">{{ t('checkout.subtitle') }}</p>
      </div>

      <!-- Subscription Warning -->
      <div class="bg-yellow-900/30 border border-yellow-700 rounded-lg p-4 mb-6">
        <div class="flex items-start gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-yellow-400 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/>
            <path d="M12 9v4"/>
            <path d="M12 17h.01"/>
          </svg>
          <div>
            <p class="text-sm text-yellow-200 font-medium">{{ t('checkout.subscriptionWarning') }}</p>
            <p class="text-sm text-yellow-200/80 mt-1">{{ t('checkout.subscriptionWarningHint') }}</p>
          </div>
        </div>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-4 rounded-lg text-sm">
        {{ error }}
      </div>

      <div v-if="loading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>

      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div
          v-for="plan in plans"
          :key="plan.id"
          class="rounded-xl border bg-card text-card-foreground shadow-sm p-6 cursor-pointer transition-all duration-200 hover:border-primary/50"
          :class="{ 'border-primary ring-2 ring-primary/20': selectedPlanId === plan.id }"
          @click="selectedPlanId = plan.id"
        >
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-bold">{{ plan.name }}</h3>
            <span v-if="plan.is_recommended" class="px-2 py-0.5 text-xs font-medium bg-primary/15 text-primary rounded-full">
              {{ t('checkout.recommended') }}
            </span>
          </div>

          <div class="mb-4">
            <span class="text-3xl font-bold">{{ formatPrice(plan) }}</span>
            <span class="text-muted-foreground">/{{ t('checkout.month') }}</span>
          </div>

          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('checkout.tunnels') }}</span>
              <span class="font-mono">{{ formatLimit(plan.max_tunnels) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('checkout.domains') }}</span>
              <span class="font-mono">{{ formatLimit(plan.max_domains) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('checkout.customDomains') }}</span>
              <span class="font-mono">{{ formatLimit(plan.max_custom_domains) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('checkout.tokens') }}</span>
              <span class="font-mono">{{ formatLimit(plan.max_tokens) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('checkout.inspector') }}</span>
              <span :class="plan.inspector_enabled ? 'text-green-400' : 'text-muted-foreground'">
                {{ plan.inspector_enabled ? t('common.yes') : t('common.no') }}
              </span>
            </div>
          </div>

          <div class="mt-4 pt-4 border-t border-border">
            <div
              class="w-5 h-5 rounded-full border-2 mx-auto flex items-center justify-center"
              :class="selectedPlanId === plan.id ? 'border-primary bg-primary' : 'border-muted-foreground'"
            >
              <svg v-if="selectedPlanId === plan.id" xmlns="http://www.w3.org/2000/svg" class="h-3 w-3 text-primary-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Recurring toggle -->
      <Card v-if="selectedPlan" class="p-6 mt-6">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="font-medium">{{ t('checkout.autoRenewal') }}</h3>
            <p class="text-sm text-muted-foreground">{{ t('checkout.autoRenewalHint') }}</p>
          </div>
          <button
            type="button"
            class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
            :class="recurring ? 'bg-primary' : 'bg-muted'"
            @click="recurring = !recurring"
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
              :class="recurring ? 'translate-x-5' : 'translate-x-0'"
            />
          </button>
        </div>
      </Card>

      <!-- Summary and Pay button -->
      <Card v-if="selectedPlan" class="p-6 mt-6">
        <div class="flex items-center justify-between mb-4">
          <span class="text-muted-foreground">{{ t('checkout.selectedPlan') }}</span>
          <span class="font-bold">{{ selectedPlan.name }}</span>
        </div>
        <div class="flex items-center justify-between mb-4">
          <span class="text-muted-foreground">{{ t('checkout.total') }}</span>
          <span class="text-2xl font-bold">{{ formatPrice(selectedPlan) }}</span>
        </div>
        <div class="flex items-center justify-between text-sm text-muted-foreground mb-6">
          <span>{{ t('checkout.paymentType') }}</span>
          <span>{{ recurring ? t('checkout.subscription') : t('checkout.oneTime') }}</span>
        </div>
        <Button
          class="w-full"
          size="lg"
          :loading="submitting"
          :disabled="!selectedPlanId"
          @click="handleCheckout"
        >
          {{ t('checkout.payNow') }}
        </Button>
        <p class="text-xs text-muted-foreground text-center mt-4">
          {{ t('checkout.securePaymentVia') }}
          <a href="https://robokassa.com" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">Robokassa</a>
        </p>
      </Card>
    </div>
  </Layout>
</template>
