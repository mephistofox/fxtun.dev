<script setup lang="ts">
/**
 * One page for the search-intent landings.
 *
 * Every one of them is the same shape — a headline, the reason the visitor is
 * stuck, the ways out, the command that solves it, a FAQ — and differs only in
 * its text. The route names the translation namespace; copying the 300-line
 * view per landing would only mean fixing every future change five times.
 */
import { ref, computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema, useFaqSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'

interface Card { title: string; desc: string }
interface Step { title: string; command?: string; desc: string }
interface FaqItem { q: string; a: string }
interface RelatedLink { to: string; label: string }

const { t, tm } = useI18n()
const route = useRoute()

const ns = route.meta.ns as string
const seoKey = route.meta.seoKey as string
const path = route.path

const key = (name: string) => `useCase.${ns}.${name}`

useSeo({ titleKey: `seo.${seoKey}.title`, descriptionKey: `seo.${seoKey}.description` })

const cards = computed(() => tm(key('cards')) as Card[])
const steps = computed(() => tm(key('steps')) as Step[])
const faqItems = computed(() => tm(key('faq')) as FaqItem[])
const related = computed(() => tm(key('related')) as RelatedLink[])

useSubpageSchema({
  path,
  name: t(key('h1')),
  description: t(`seo.${seoKey}.description`),
  breadcrumbs: [{ name: t(key('h1')), path }],
})

useFaqSchema(
  faqItems.value.map(item => ({ question: item.q, answer: item.a })),
  `-${ns}`,
)

const openFaqIndex = ref<number | null>(null)
function toggleFaq(index: number) {
  openFaqIndex.value = openFaqIndex.value === index ? null : index
}

const copiedStep = ref<number | null>(null)
async function copyCommand(command: string, index: number) {
  try {
    await navigator.clipboard.writeText(command)
    copiedStep.value = index
    setTimeout(() => { copiedStep.value = null }, 2000)
  } catch {
    copiedStep.value = null
  }
}
</script>

<template>
  <div class="ngrok-alt-page">
    <nav class="page-nav">
      <div class="container mx-auto px-4 flex items-center justify-between h-16">
        <RouterLink to="/" class="page-nav-brand">
          <div class="page-nav-logo">
            <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <span class="font-display font-semibold text-lg">fxtun</span>
        </RouterLink>
        <div class="flex items-center gap-4">
          <RouterLink to="/login" class="page-nav-link">{{ t('auth.signIn') }}</RouterLink>
          <RouterLink to="/register" class="page-nav-cta">{{ t('landing.hero.getStarted') }}</RouterLink>
        </div>
      </div>
    </nav>

    <main class="pt-16">
      <Breadcrumbs :items="[{ name: t(key('h1')), path }]" />

      <section class="container mx-auto px-4 pt-8 pb-10 md:pt-12 md:pb-14 max-w-4xl text-center">
        <h1 class="text-3xl md:text-5xl font-display font-bold mb-6">{{ t(key('h1')) }}</h1>
        <p class="text-lg md:text-xl text-muted-foreground leading-relaxed max-w-3xl mx-auto">
          {{ t(key('intro')) }}
        </p>
      </section>

      <!-- Why it does not work on its own -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-3xl">
        <h2 class="text-2xl md:text-3xl font-display font-bold mb-5">{{ t(key('whyTitle')) }}</h2>
        <p class="text-muted-foreground leading-relaxed whitespace-pre-line">{{ t(key('whyText')) }}</p>
      </section>

      <!-- The ways out -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-5xl">
        <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-8">{{ t(key('cardsTitle')) }}</h2>
        <div class="grid gap-5 sm:grid-cols-2">
          <div v-for="(card, index) in cards" :key="index" class="wedge-card">
            <h3 class="text-lg font-display font-semibold mb-2">{{ card.title }}</h3>
            <p class="text-muted-foreground leading-relaxed">{{ card.desc }}</p>
          </div>
        </div>
      </section>

      <!-- Step by step -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-3xl">
        <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-8">{{ t(key('stepsTitle')) }}</h2>
        <ol class="space-y-6">
          <li v-for="(step, index) in steps" :key="index">
            <h3 class="text-lg font-display font-semibold mb-2">
              <span class="text-primary mr-2">{{ index + 1 }}.</span>{{ step.title }}
            </h3>
            <p class="text-muted-foreground leading-relaxed mb-3">{{ step.desc }}</p>
            <div v-if="step.command" class="command-block">
              <code class="command-text">
                <span class="command-prompt">$</span>
                <span>{{ step.command }}</span>
              </code>
              <button type="button" class="command-copy" :aria-label="t('common.copy')" @click="copyCommand(step.command, index)">
                <svg v-if="copiedStep !== index" aria-hidden="true" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" />
                </svg>
                <svg v-else aria-hidden="true" class="h-4 w-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                </svg>
              </button>
            </div>
          </li>
        </ol>
      </section>

      <!-- FAQ -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-3xl">
        <h2 class="text-2xl md:text-3xl font-display font-bold text-center mb-10">{{ t('landing.faq.title') }}</h2>
        <div>
          <div
            v-for="(item, index) in faqItems"
            :key="index"
            class="border-b border-border"
            :class="{ 'border-primary/20': openFaqIndex === index }"
          >
            <button
              class="w-full flex items-center justify-between py-5 text-left group"
              @click="toggleFaq(index)"
            >
              <span
                class="text-base font-medium pr-8 group-hover:text-primary transition-colors"
                :class="{ 'text-primary': openFaqIndex === index }"
              >{{ item.q }}</span>
              <svg
                class="h-5 w-5 flex-shrink-0 text-muted-foreground transition-transform duration-300"
                :class="{ 'rotate-180 text-primary': openFaqIndex === index }"
                fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
              </svg>
            </button>
            <Transition name="faq-expand">
              <div v-if="openFaqIndex === index" class="faq-answer">
                <p class="pb-5 text-muted-foreground leading-relaxed">{{ item.a }}</p>
              </div>
            </Transition>
          </div>
        </div>
      </section>

      <!-- Where to go next -->
      <section class="container mx-auto px-4 pb-12 md:pb-16 max-w-4xl">
        <h2 class="text-xl font-display font-semibold mb-6 text-center">{{ t(key('relatedTitle')) }}</h2>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <RouterLink v-for="link in related" :key="link.to" :to="link.to" class="compare-see-also-card">
            <span class="text-sm font-medium">{{ link.label }}</span>
            <svg aria-hidden="true" class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/></svg>
          </RouterLink>
        </div>
      </section>

      <section class="container mx-auto px-4 pb-16 md:pb-24 max-w-3xl text-center">
        <h2 class="text-2xl md:text-3xl font-display font-bold mb-8">{{ t(key('ctaTitle')) }}</h2>
        <RouterLink to="/register" class="cta-button">{{ t(key('ctaButton')) }}</RouterLink>
      </section>
    </main>

    <LandingFooter />
  </div>
</template>
