<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const currentStep = ref(0)
let timer: ReturnType<typeof setTimeout> | null = null

const steps = computed(() => [
  { label: t('landing.advanced.customDomains.demo.step1'), duration: 2000 },
  { label: t('landing.advanced.customDomains.demo.step2'), duration: 1800 },
  { label: t('landing.advanced.customDomains.demo.step3'), duration: 2500 },
])

function scheduleNext() {
  const delay = steps.value[currentStep.value].duration
  timer = setTimeout(() => {
    currentStep.value = (currentStep.value + 1) % 3
    scheduleNext()
  }, delay)
}

onMounted(() => {
  scheduleNext()
})

onUnmounted(() => {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
})
</script>

<template>
  <div class="rounded-lg border border-border bg-background overflow-hidden font-mono h-[256px] flex flex-col">
    <!-- Header bar -->
    <div class="flex items-center gap-2.5 px-4 py-2.5 border-b border-border bg-surface">
      <svg class="w-5 h-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
      </svg>
      <span class="text-foreground text-sm font-medium">{{ t('landing.advanced.customDomains.demo.domain') }}</span>
    </div>

    <!-- Steps -->
    <div class="flex-1 p-5 space-y-5 overflow-hidden">
      <!-- Step 0: Configure DNS -->
      <div
        class="flex items-start gap-3 transition-opacity duration-300"
        :class="currentStep < 0 ? 'opacity-30' : ''"
      >
        <div class="flex-shrink-0 mt-0.5">
          <div
            v-if="currentStep > 0"
            class="w-7 h-7 rounded-full bg-emerald-500/20 flex items-center justify-center"
          >
            <svg class="w-4 h-4 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div
            v-else
            class="w-7 h-7 rounded-full border border-primary/50 flex items-center justify-center text-xs font-semibold text-primary"
          >
            1
          </div>
        </div>
        <div>
          <div class="text-sm text-foreground font-medium">{{ steps[0].label }}</div>
          <div
            v-if="currentStep === 0"
            class="mt-1.5 text-xs px-3 py-1.5 rounded bg-surface border border-border text-muted-foreground"
          >
            {{ t('landing.advanced.customDomains.demo.cname') }}
          </div>
        </div>
      </div>

      <!-- Step 1: Verifying -->
      <div
        class="flex items-start gap-3 transition-opacity duration-300"
        :class="currentStep < 1 ? 'opacity-30' : ''"
      >
        <div class="flex-shrink-0 mt-0.5">
          <div
            v-if="currentStep > 1"
            class="w-7 h-7 rounded-full bg-emerald-500/20 flex items-center justify-center"
          >
            <svg class="w-4 h-4 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div
            v-else-if="currentStep === 1"
            class="w-7 h-7 rounded-full border border-amber-500/50 flex items-center justify-center"
          >
            <svg class="w-4 h-4 text-amber-400 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M12 2a10 10 0 0 1 10 10" />
            </svg>
          </div>
          <div
            v-else
            class="w-7 h-7 rounded-full border border-border flex items-center justify-center text-xs font-semibold text-muted-foreground"
          >
            2
          </div>
        </div>
        <div>
          <div class="text-sm font-medium" :class="currentStep === 1 ? 'text-amber-400' : 'text-foreground'">
            {{ currentStep > 1 ? t('landing.advanced.customDomains.demo.verified') : steps[1].label }}
          </div>
        </div>
      </div>

      <!-- Step 2: Certificate issued -->
      <div
        class="flex items-start gap-3 transition-opacity duration-300"
        :class="currentStep < 2 ? 'opacity-30' : ''"
      >
        <div class="flex-shrink-0 mt-0.5">
          <div
            v-if="currentStep >= 2"
            class="w-7 h-7 rounded-full bg-emerald-500/20 flex items-center justify-center"
          >
            <svg class="w-4 h-4 text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div
            v-else
            class="w-7 h-7 rounded-full border border-border flex items-center justify-center text-xs font-semibold text-muted-foreground"
          >
            3
          </div>
        </div>
        <div>
          <div class="text-sm text-foreground font-medium">{{ steps[2].label }}</div>
          <div
            v-if="currentStep === 2"
            class="mt-1.5 inline-block text-xs px-3 py-1 rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 font-medium"
          >
            🔒 {{ t('landing.advanced.customDomains.demo.https') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
