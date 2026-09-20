<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import { getBlogUrl } from '@/i18n'

const { t, locale } = useI18n()

const currentYear = new Date().getFullYear()

const showOffer = computed(() => locale.value === 'ru')
const blogUrl = computed(() => getBlogUrl())
</script>

<template>
  <footer class="py-12 bg-background border-t border-border relative">
    <!-- Subtle gradient -->
    <div class="absolute inset-0 bg-gradient-to-t from-surface/30 to-transparent pointer-events-none" />

    <div class="container mx-auto px-4 relative z-10">
      <!-- Four columns instead of fifteen links in a row: product, the
           situations people arrive with, the comparisons, and the paperwork.
           A flat list of everything is a list of nothing. -->
      <div class="grid gap-10 md:grid-cols-2 lg:grid-cols-5">
        <div class="lg:col-span-2">
          <div class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center">
              <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <div>
              <span class="font-display font-semibold text-lg">fxTunnel</span>
              <p class="text-xs text-muted-foreground">{{ t('landing.footer.tagline') || 'Secure tunneling' }}</p>
            </div>
          </div>
          <p class="text-sm text-muted-foreground max-w-xs leading-relaxed">{{ t('landing.footer.blurb') }}</p>

          <h3 class="footer-heading mt-8">{{ t('landing.footer.colContacts') }}</h3>
          <ul class="footer-list">
            <li><a href="mailto:dev@fxcode.ru" class="footer-link">dev@fxcode.ru</a></li>
            <li><a href="https://t.me/mephistofx" rel="noopener" class="footer-link">Telegram: @mephistofx</a></li>
            <li><a href="https://github.com/mephistofox/fxtun.dev" rel="noopener" class="footer-link">GitHub</a></li>
          </ul>
          <p class="mt-3 text-xs text-muted-foreground">{{ t('landing.footer.contactsNote') }}</p>
        </div>

        <div>
          <h3 class="footer-heading">{{ t('landing.footer.colProduct') }}</h3>
          <ul class="footer-list">
            <li><RouterLink to="/features" class="footer-link">{{ t('landing.nav.features') }}</RouterLink></li>
            <li><RouterLink to="/pricing" class="footer-link">{{ t('landing.nav.pricing') }}</RouterLink></li>
            <li><RouterLink to="/downloads" class="footer-link">{{ t('landing.nav.download') }}</RouterLink></li>
            <li><a :href="blogUrl" class="footer-link">{{ t('landing.nav.blog') }}</a></li>
            <li><RouterLink to="/about" class="footer-link">{{ t('about.navTitle') }}</RouterLink></li>
          </ul>
        </div>

        <div>
          <h3 class="footer-heading">{{ t('landing.footer.colUseCases') }}</h3>
          <ul class="footer-list">
            <li v-if="showOffer"><RouterLink to="/probros-portov" class="footer-link">{{ t('landing.footer.linkPortForward') }}</RouterLink></li>
            <li><RouterLink to="/ngrok-alternative" class="footer-link">{{ t('landing.footer.linkNgrokAlt') }}</RouterLink></li>
            <li v-if="showOffer"><RouterLink to="/bez-belogo-ip" class="footer-link">{{ t('landing.footer.linkNoWhiteIp') }}</RouterLink></li>
            <li v-if="showOffer"><RouterLink to="/minecraft-server" class="footer-link">{{ t('landing.footer.linkMinecraft') }}</RouterLink></li>
          </ul>

          <h3 class="footer-heading mt-8">{{ t('landing.footer.colCompare') }}</h3>
          <ul class="footer-list">
            <li><RouterLink to="/compare/ngrok" class="footer-link">vs ngrok</RouterLink></li>
            <li><RouterLink to="/compare/cloudpub" class="footer-link">vs CloudPub</RouterLink></li>
            <li><RouterLink to="/compare/cloudflare" class="footer-link">vs Cloudflare</RouterLink></li>
            <li><RouterLink to="/compare/tuna" class="footer-link">vs tuna.am</RouterLink></li>
            <li><RouterLink to="/compare/xtunnel" class="footer-link">vs xTunnel</RouterLink></li>
          </ul>
        </div>

        <div>
          <h3 class="footer-heading">{{ t('landing.footer.colLegal') }}</h3>
          <ul class="footer-list">
            <li v-if="showOffer"><RouterLink to="/offer" class="footer-link">{{ t('legal.offer') }}</RouterLink></li>
            <li v-else><RouterLink to="/terms" class="footer-link">{{ t('legal.terms') }}</RouterLink></li>
            <li><RouterLink to="/privacy" class="footer-link">{{ t('legal.privacy') }}</RouterLink></li>
            <li><RouterLink to="/aup" class="footer-link">{{ t('legal.aup') }}</RouterLink></li>
            <li><RouterLink to="/abuse" class="footer-link">{{ t('legal.abuse') }}</RouterLink></li>
            <li><RouterLink to="/disclaimer" class="footer-link">{{ t('legal.disclaimer') }}</RouterLink></li>
          </ul>
        </div>
      </div>

      <div class="mt-10 pt-6 border-t border-border flex flex-col sm:flex-row items-center justify-between gap-3">
        <p class="text-sm text-muted-foreground">© {{ currentYear }} fxTunnel. {{ t('landing.footer.rights') }}</p>
        <p class="text-sm text-muted-foreground">{{ t('landing.footer.madeIn') }}</p>
      </div>
    </div>
  </footer>
</template>
