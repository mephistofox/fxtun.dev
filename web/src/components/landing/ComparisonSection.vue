<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)

const competitors = ['fxtunnel', 'ngrok', 'localhostrun'] as const

const features = [
  'price',
  'freeSubdomain',
  'requestLimits',
  'sessionTimeout',
  'protocols',
  'guiClient',
  'inspector',
  'selfHosted',
  'customDomains',
] as const

// Highlight fxTunnel advantages (cells where we're better)
const advantages: Record<string, Set<string>> = {
  fxtunnel: new Set(['freeSubdomain', 'requestLimits', 'sessionTimeout', 'protocols', 'guiClient', 'inspector', 'selfHosted']),
}

function isAdvantage(competitor: string, feature: string): boolean {
  return advantages[competitor]?.has(feature) ?? false
}

function isNegative(competitor: string, feature: string): boolean {
  const val = t(`landing.comparison.values.${competitor}.${feature}`)
  return /^(No|Нет|Random only|Только рандомный)$/i.test(val)
}

onMounted(() => {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          isVisible.value = true
          observer.disconnect()
        }
      })
    },
    { threshold: 0.1 }
  )
  if (sectionRef.value) {
    observer.observe(sectionRef.value)
  }
})
</script>

<template>
  <section id="comparison" ref="sectionRef" class="py-16 md:py-32 relative overflow-hidden">
    <!-- Background -->
    <div class="absolute inset-0 bg-gradient-to-b from-background via-surface/20 to-background" />

    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-16">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <span class="text-sm font-medium text-primary">{{ t('landing.comparison.label') }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.comparison.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.comparison.subtitle') }}
        </p>
      </div>

      <!-- Comparison table -->
      <div
        class="max-w-4xl mx-auto reveal reveal-delay-3"
        :class="{ 'visible': isVisible }"
      >
        <!-- Desktop table -->
        <div class="hidden md:block rounded-2xl border border-border overflow-hidden bg-surface/50">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border">
                <th class="text-left p-4 font-medium text-muted-foreground w-1/4">
                  {{ t('landing.comparison.feature') }}
                </th>
                <th
                  v-for="comp in competitors"
                  :key="comp"
                  class="p-4 font-display font-semibold text-center"
                  :class="comp === 'fxtunnel' ? 'text-primary bg-primary/5' : 'text-foreground'"
                >
                  {{ comp === 'fxtunnel' ? 'fxTunnel' : comp === 'ngrok' ? 'ngrok' : 'localhost.run' }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(feat, idx) in features"
                :key="feat"
                :class="idx % 2 === 0 ? 'bg-transparent' : 'bg-surface/30'"
                class="border-b border-border/50 last:border-0"
              >
                <td class="p-4 font-medium text-muted-foreground">
                  {{ t(`landing.comparison.${feat}`) }}
                </td>
                <td
                  v-for="comp in competitors"
                  :key="comp"
                  class="p-4 text-center"
                  :class="[
                    comp === 'fxtunnel' ? 'bg-primary/5' : '',
                    isAdvantage(comp, feat) ? 'text-primary font-medium' : '',
                    isNegative(comp, feat) ? 'text-muted-foreground/50' : '',
                  ]"
                >
                  {{ t(`landing.comparison.values.${comp}.${feat}`) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Mobile cards -->
        <div class="md:hidden space-y-3">
          <div
            v-for="feat in features"
            :key="feat"
            class="rounded-xl border border-border bg-surface/50 p-4"
          >
            <p class="text-xs font-medium text-muted-foreground mb-3">
              {{ t(`landing.comparison.${feat}`) }}
            </p>
            <div class="grid grid-cols-3 gap-2 text-center text-sm">
              <div
                v-for="comp in competitors"
                :key="comp"
                class="py-2 px-1 rounded-lg"
                :class="[
                  comp === 'fxtunnel' ? 'bg-primary/10 text-primary font-medium' : '',
                  isNegative(comp, feat) ? 'text-muted-foreground/50' : '',
                ]"
              >
                <div class="text-[10px] text-muted-foreground mb-1">
                  {{ comp === 'fxtunnel' ? 'fxTunnel' : comp === 'ngrok' ? 'ngrok' : 'lhr' }}
                </div>
                {{ t(`landing.comparison.values.${comp}.${feat}`) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
