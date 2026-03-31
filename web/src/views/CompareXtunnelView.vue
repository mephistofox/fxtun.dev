<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ComparePageLayout from '@/components/landing/ComparePageLayout.vue'
import { useFaqSchema } from '@/composables/useStructuredData'

const { t, tm } = useI18n()

const tableRows = [
  'price', 'protocols', 'tunnels', 'sessionTimeout', 'requestLimits',
  'bandwidth', 'subdomains', 'customDomains', 'gui', 'inspector',
  'selfHosted', 'permanentAddress',
] as const

interface FaqItem {
  q: string
  a: string
}

const faqItems = computed(() => tm('compare.xtunnel.faq') as FaqItem[])
useFaqSchema(faqItems.value.map(item => ({ question: item.q, answer: item.a })), '-xtunnel')

const openFaqIndex = ref<number | null>(null)

function toggleFaq(index: number) {
  openFaqIndex.value = openFaqIndex.value === index ? null : index
}

const sectionKeys = [
  'pricing', 'protocols', 'limits', 'gui', 'inspector', 'selfHosted',
] as const
</script>

<template>
  <ComparePageLayout competitor-name="xTunnel" competitor-slug="xtunnel">
    <!-- Comparison Table -->
    <template #table>
      <section class="py-8 md:py-16">
        <div class="container mx-auto px-4">
          <div class="max-w-4xl mx-auto overflow-x-auto">
            <table class="compare-table w-full">
              <thead>
                <tr>
                  <th class="text-left">{{ t('compare.feature') }}</th>
                  <th class="text-center compare-highlight">fxTunnel</th>
                  <th class="text-center">xTunnel</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in tableRows" :key="row">
                  <td class="font-medium">{{ t(`compare.xtunnel.table.${row}.label`) }}</td>
                  <td class="text-center compare-highlight">{{ t(`compare.xtunnel.table.${row}.fxtunnel`) }}</td>
                  <td class="text-center">{{ t(`compare.xtunnel.table.${row}.competitor`) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </template>

    <!-- Feature Details -->
    <template #details>
      <section class="py-8 md:py-16">
        <div class="container mx-auto px-4 max-w-4xl">
          <div v-for="key in sectionKeys" :key="key" class="compare-detail-section">
            <h2 class="text-2xl font-display font-semibold mb-4">
              {{ t(`compare.xtunnel.sections.${key}Title`) }}
            </h2>
            <p class="text-muted-foreground leading-relaxed">
              {{ t(`compare.xtunnel.sections.${key}Text`) }}
            </p>
          </div>

          <!-- Verdict -->
          <div class="compare-verdict mt-12 border-l-4 border-primary rounded-r-lg bg-primary/5 p-6">
            <h2 class="text-2xl font-display font-semibold mb-4">
              {{ t('compare.xtunnel.sections.verdictTitle') }}
            </h2>
            <p class="text-muted-foreground leading-relaxed">
              {{ t('compare.xtunnel.sections.verdictText') }}
            </p>
          </div>
        </div>
      </section>
    </template>

    <!-- FAQ -->
    <template #faq>
      <section class="py-8 md:py-16">
        <div class="container mx-auto px-4 max-w-3xl">
          <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-12">
            {{ t('compare.faqTitle') }}
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
    </template>
  </ComparePageLayout>
</template>

<style scoped>
.compare-table {
  border-collapse: separate;
  border-spacing: 0;
}

.compare-table th {
  @apply py-4 px-4 text-sm font-semibold uppercase tracking-wider;
  color: hsl(var(--muted-foreground));
  border-bottom: 2px solid hsl(var(--border));
}

.compare-table td {
  @apply py-3.5 px-4 text-sm;
  border-bottom: 1px solid hsl(var(--border) / 0.5);
  color: hsl(var(--foreground));
}

.compare-table tbody tr:hover {
  background: hsl(var(--surface) / 0.3);
}

.compare-highlight {
  background: hsl(var(--primary) / 0.05);
}

.compare-detail-section {
  @apply mb-12;
}

.compare-detail-section:last-child {
  @apply mb-0;
}

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
