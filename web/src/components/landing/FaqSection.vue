<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t, tm } = useI18n()

const openIndex = ref<number | null>(null)
const isVisible = ref(false)
const sectionRef = ref<HTMLElement | null>(null)

interface FaqItem {
  q: string
  a: string
}

const faqItems = computed(() => {
  return tm('landing.faq.items') as FaqItem[]
})

function toggle(index: number) {
  openIndex.value = openIndex.value === index ? null : index
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
  <section id="faq" ref="sectionRef" class="py-16 md:py-32 bg-background relative overflow-hidden">
    <div class="container mx-auto px-4 relative z-10">
      <!-- Section header -->
      <div class="max-w-3xl mx-auto text-center mb-16">
        <div
          class="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-primary/30 bg-primary/5 mb-6 reveal"
          :class="{ 'visible': isVisible }"
        >
          <svg class="h-4 w-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z" />
          </svg>
          <span class="text-sm font-medium text-primary">{{ t('landing.faq.label') }}</span>
        </div>

        <h2
          class="text-display-lg font-display mb-6 reveal reveal-delay-1"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.faq.title') }}
        </h2>

        <p
          class="text-xl text-muted-foreground reveal reveal-delay-2"
          :class="{ 'visible': isVisible }"
        >
          {{ t('landing.faq.subtitle') }}
        </p>
      </div>

      <!-- FAQ accordion -->
      <div class="max-w-3xl mx-auto">
        <div
          v-for="(item, index) in faqItems"
          :key="index"
          class="reveal"
          :class="[
            { 'visible': isVisible },
            `reveal-delay-${Math.min(3 + Math.floor(index / 2), 7)}`
          ]"
        >
          <div
            class="border-b border-border"
            :class="{ 'border-primary/20': openIndex === index }"
          >
            <button
              @click="toggle(index)"
              class="w-full flex items-center justify-between py-5 text-left group"
            >
              <span class="text-base font-medium pr-8 group-hover:text-primary transition-colors" :class="{ 'text-primary': openIndex === index }">
                {{ item.q }}
              </span>
              <svg
                class="h-5 w-5 flex-shrink-0 text-muted-foreground transition-transform duration-300"
                :class="{ 'rotate-180 text-primary': openIndex === index }"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
              </svg>
            </button>

            <Transition name="faq-expand">
              <div v-if="openIndex === index" class="faq-answer">
                <p class="pb-5 text-muted-foreground leading-relaxed">
                  {{ item.a }}
                </p>
              </div>
            </Transition>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
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
