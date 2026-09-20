<script setup lang="ts">
/**
 * One page for the search-intent landings.
 *
 * Every one of them is the same shape — a headline, the reason the visitor is
 * stuck, the ways out, the commands that solve it, a FAQ — and differs only in
 * its text. The route names the translation namespace; copying the view per
 * landing would only mean fixing every future change five times.
 *
 * It borrows the home page's vocabulary — the tinted backdrop, the glass
 * cards, the glowing call to action — so a visitor arriving from search lands
 * on the same site, not on a plain document. Getting the client is a pair of
 * buttons rather than an install command: somebody looking for a Minecraft
 * server does not want to be handed a shell pipeline first.
 */
import { ref, computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSeo } from '@/composables/useSeo'
import { useSubpageSchema, useFaqSchema } from '@/composables/useStructuredData'
import LandingFooter from '@/components/landing/LandingFooter.vue'
import Breadcrumbs from '@/components/landing/Breadcrumbs.vue'

interface Card { title: string; desc: string }
interface Step { title: string; command?: string; result?: string; desc: string }
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
const whyParagraphs = computed(() => tm(key('whyParagraphs')) as unknown as string[])

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
  <div class="page-shell">
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

      <!-- Hero -->
      <section class="container mx-auto px-4 pt-10 pb-14 md:pt-16 md:pb-20 max-w-4xl text-center">
        <span class="page-eyebrow">
          <span class="pulse-indicator" aria-hidden="true"></span>
          {{ t(key('eyebrow')) }}
        </span>
        <h1 class="text-3xl md:text-5xl lg:text-6xl font-display font-bold mb-6 leading-tight">
          {{ t(key('h1')) }}
        </h1>
        <p class="text-lg md:text-xl text-muted-foreground leading-relaxed max-w-3xl mx-auto mb-10">
          {{ t(key('intro')) }}
        </p>
        <div class="flex flex-col sm:flex-row gap-4 justify-center">
          <RouterLink to="/register" class="cta-button">
            {{ t(key('ctaButton')) }}
            <svg aria-hidden="true" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
            </svg>
          </RouterLink>
          <RouterLink to="/downloads" class="cta-button-ghost">
            <svg aria-hidden="true" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v12m0 0l-4-4m4 4l4-4M4 17v2a2 2 0 002 2h12a2 2 0 002-2v-2" />
            </svg>
            {{ t('useCase.common.download') }}
          </RouterLink>
        </div>
        <p class="text-sm text-muted-foreground mt-4">{{ t('useCase.common.freeNote') }}</p>
      </section>

      <div class="container mx-auto px-4 max-w-4xl"><div class="section-divider"></div></div>

      <!-- Why it does not work on its own -->
      <section class="container mx-auto px-4 py-14 md:py-20 max-w-3xl">
        <h2 class="text-2xl md:text-4xl font-display font-bold mb-6">{{ t(key('whyTitle')) }}</h2>
        <!-- Paragraphs as a list: the HTML minifier collapses the newlines a
             single string would rely on, and the whole explanation ran together. -->
        <p
          v-for="(para, index) in whyParagraphs"
          :key="index"
          class="text-base md:text-lg text-muted-foreground leading-relaxed mb-5 last:mb-0"
        >{{ para }}</p>
      </section>

      <!-- The ways out -->
      <section class="container mx-auto px-4 pb-14 md:pb-20 max-w-5xl">
        <h2 class="text-2xl md:text-4xl font-display font-bold text-center mb-10">{{ t(key('cardsTitle')) }}</h2>
        <div class="grid gap-5 sm:grid-cols-2">
          <div v-for="(card, index) in cards" :key="index" class="wedge-card">
            <h3 class="text-lg font-display font-semibold mb-3">{{ card.title }}</h3>
            <p class="text-muted-foreground leading-relaxed">{{ card.desc }}</p>
          </div>
        </div>
      </section>

      <div class="container mx-auto px-4 max-w-4xl"><div class="section-divider"></div></div>

      <!-- Step by step -->
      <section class="container mx-auto px-4 py-14 md:py-20 max-w-3xl">
        <h2 class="text-2xl md:text-4xl font-display font-bold text-center mb-12">{{ t(key('stepsTitle')) }}</h2>
        <ol class="space-y-10">
          <li v-for="(step, index) in steps" :key="index" class="flex gap-5">
            <div class="step-number">{{ index + 1 }}</div>
            <div class="flex-1 min-w-0">
              <h3 class="text-lg font-display font-semibold mb-2">{{ step.title }}</h3>
              <p class="text-muted-foreground leading-relaxed mb-4">{{ step.desc }}</p>
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
              <!-- What the command prints. A bare command asks the reader to
                   imagine the result; showing it is the difference between a
                   snippet and a demonstration. -->
              <p v-if="step.result" class="command-result">
                <span class="pulse-indicator" aria-hidden="true"></span>
                {{ step.result }}
              </p>
            </div>
          </li>
        </ol>

        <div class="flex flex-col sm:flex-row gap-4 justify-center mt-12">
          <RouterLink to="/downloads" class="cta-button">
            <svg aria-hidden="true" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v12m0 0l-4-4m4 4l4-4M4 17v2a2 2 0 002 2h12a2 2 0 002-2v-2" />
            </svg>
            {{ t('useCase.common.download') }}
          </RouterLink>
          <RouterLink to="/register" class="cta-button-ghost">{{ t('useCase.common.createAccount') }}</RouterLink>
        </div>
      </section>

      <div class="container mx-auto px-4 max-w-4xl"><div class="section-divider"></div></div>

      <!-- FAQ -->
      <section class="container mx-auto px-4 py-14 md:py-20 max-w-3xl">
        <h2 class="text-2xl md:text-4xl font-display font-bold text-center mb-10">{{ t('landing.faq.title') }}</h2>
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
      <section class="container mx-auto px-4 pb-14 md:pb-20 max-w-4xl">
        <h2 class="text-xl font-display font-semibold mb-6 text-center">{{ t(key('relatedTitle')) }}</h2>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <RouterLink v-for="link in related" :key="link.to" :to="link.to" class="compare-see-also-card">
            <span class="text-sm font-medium">{{ link.label }}</span>
            <svg aria-hidden="true" class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/></svg>
          </RouterLink>
        </div>
      </section>

      <!-- Final call to action -->
      <section class="container mx-auto px-4 pb-20 md:pb-28 max-w-3xl">
        <div class="glass-card p-8 md:p-12 text-center">
          <h2 class="text-2xl md:text-4xl font-display font-bold mb-4">{{ t(key('ctaTitle')) }}</h2>
          <p class="text-muted-foreground mb-8">{{ t('useCase.common.ctaSubtitle') }}</p>
          <div class="flex flex-col sm:flex-row gap-4 justify-center">
            <RouterLink to="/register" class="cta-button">
              {{ t(key('ctaButton')) }}
              <svg aria-hidden="true" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
              </svg>
            </RouterLink>
            <RouterLink to="/pricing" class="cta-button-ghost">{{ t('useCase.common.seePricing') }}</RouterLink>
          </div>
        </div>
      </section>
    </main>

    <LandingFooter />
  </div>
</template>
