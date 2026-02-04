<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { plansApi, type Plan } from '@/api/client'

const { t } = useI18n()

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)
const plans = ref<Plan[]>([])
const loading = ref(true)

function displayLimit(val: number): string {
  return val < 0 ? t('landing.pricing.unlimited') : String(val)
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
        class="grid gap-8 reveal reveal-delay-3"
        :class="[
          { 'visible': isVisible },
          plans.length === 1 ? 'max-w-md mx-auto' : '',
          plans.length === 2 ? 'md:grid-cols-2 max-w-3xl mx-auto' : '',
          plans.length >= 3 ? 'md:grid-cols-2 lg:grid-cols-3 max-w-5xl mx-auto' : '',
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
              'h-full rounded-2xl p-8 transition-all duration-300',
              plan.is_recommended
                ? 'bg-surface border-2 border-primary shadow-xl shadow-primary/10 group-hover:shadow-2xl group-hover:shadow-primary/20'
                : 'bg-surface/50 border border-border group-hover:border-primary/30 group-hover:shadow-lg',
            ]"
          >
            <!-- Plan name -->
            <h3 class="text-xl font-display font-semibold mb-2">{{ plan.name }}</h3>

            <!-- Price -->
            <div class="mb-6">
              <span v-if="plan.price === 0" class="text-4xl font-display font-bold">
                {{ t('landing.pricing.free') }}
              </span>
              <template v-else>
                <span class="text-4xl font-display font-bold">${{ plan.price }}</span>
                <span class="text-muted-foreground">{{ t('landing.pricing.perMonth') }}</span>
              </template>
            </div>

            <!-- Divider -->
            <div class="h-px bg-border mb-6" />

            <!-- Features list -->
            <ul class="space-y-3 mb-8">
              <!-- Tunnels -->
              <li class="flex items-start gap-3">
                <svg class="h-5 w-5 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <div>
                  <span><strong>{{ displayLimit(plan.max_tunnels) }}</strong> {{ t('landing.pricing.tunnels') }}</span>
                  <p class="text-xs text-muted-foreground mt-0.5">{{ t('landing.pricing.tunnelsDesc') }}</p>
                </div>
              </li>

              <!-- Domains -->
              <li class="flex items-start gap-3">
                <svg class="h-5 w-5 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <div>
                  <span><strong>{{ displayLimit(plan.max_domains) }}</strong> {{ t('landing.pricing.domains') }}</span>
                  <p class="text-xs text-muted-foreground mt-0.5">{{ t('landing.pricing.domainsDesc') }}</p>
                </div>
              </li>

              <!-- Custom Domains -->
              <li class="flex items-start gap-3">
                <svg
                  class="h-5 w-5 flex-shrink-0 mt-0.5"
                  :class="plan.max_custom_domains > 0 ? 'text-primary' : 'text-muted-foreground/50'"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path v-if="plan.max_custom_domains > 0" fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                  <path v-else fill-rule="evenodd" d="M4 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H4.75A.75.75 0 014 10z" clip-rule="evenodd" />
                </svg>
                <div :class="{ 'text-muted-foreground': plan.max_custom_domains === 0 }">
                  <span><strong>{{ displayLimit(plan.max_custom_domains) }}</strong> {{ t('landing.pricing.customDomains') }}</span>
                  <p class="text-xs text-muted-foreground mt-0.5">{{ t('landing.pricing.customDomainsDesc') }}</p>
                </div>
              </li>

              <!-- Tokens -->
              <li class="flex items-start gap-3">
                <svg class="h-5 w-5 text-primary flex-shrink-0 mt-0.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                </svg>
                <div>
                  <span><strong>{{ displayLimit(plan.max_tokens) }}</strong> {{ t('landing.pricing.tokens') }}</span>
                  <p class="text-xs text-muted-foreground mt-0.5">{{ t('landing.pricing.tokensDesc') }}</p>
                </div>
              </li>

              <!-- Inspector -->
              <li class="flex items-start gap-3">
                <svg
                  class="h-5 w-5 flex-shrink-0 mt-0.5"
                  :class="plan.inspector_enabled ? 'text-primary' : 'text-muted-foreground/50'"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path v-if="plan.inspector_enabled" fill-rule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clip-rule="evenodd" />
                  <path v-else fill-rule="evenodd" d="M4 10a.75.75 0 01.75-.75h10.5a.75.75 0 010 1.5H4.75A.75.75 0 014 10z" clip-rule="evenodd" />
                </svg>
                <div :class="{ 'text-muted-foreground': !plan.inspector_enabled }">
                  <span>{{ t('landing.pricing.inspector') }}</span>
                  <p class="text-xs text-muted-foreground mt-0.5">{{ t('landing.pricing.inspectorDesc') }}</p>
                </div>
              </li>
            </ul>

            <!-- CTA Button -->
            <RouterLink
              to="/register"
              :class="[
                'block w-full py-3 px-6 rounded-xl text-center font-medium transition-all duration-300',
                plan.is_recommended
                  ? 'bg-primary text-primary-foreground hover:bg-primary/90 shadow-lg shadow-primary/25 hover:shadow-xl hover:shadow-primary/30'
                  : 'bg-surface border border-border hover:border-primary/50 hover:bg-primary/5',
              ]"
            >
              {{ t('landing.pricing.selectPlan') }}
            </RouterLink>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
