<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'

const { t, locale } = useI18n()

const currentYear = new Date().getFullYear()

const showOffer = computed(() => locale.value === 'ru')
const blogUrl = computed(() => {
  if (typeof window === 'undefined') return '/blog'
  return `${window.location.protocol}//${window.location.hostname}/blog`
})
</script>

<template>
  <footer class="py-12 bg-background border-t border-border relative">
    <!-- Subtle gradient -->
    <div class="absolute inset-0 bg-gradient-to-t from-surface/30 to-transparent pointer-events-none" />

    <div class="container mx-auto px-4 relative z-10">
      <div class="flex flex-col md:flex-row items-center justify-between gap-6">
        <!-- Logo & brand -->
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <div>
            <span class="font-display font-semibold text-lg">fxTunnel</span>
            <p class="text-xs text-muted-foreground">{{ t('landing.footer.tagline') || 'Secure tunneling' }}</p>
          </div>
        </div>

        <!-- Links -->
        <div class="flex items-center gap-6">
          <RouterLink
            v-if="showOffer"
            to="/offer"
            class="text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            {{ t('legal.offer') }}
          </RouterLink>
          <a
            :href="blogUrl"
            class="text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            {{ t('landing.nav.blog') }}
          </a>
        </div>

        <!-- Copyright -->
        <p class="text-sm text-muted-foreground">
          © {{ currentYear }} fxTunnel. {{ t('landing.footer.rights') }}
        </p>
      </div>
    </div>
  </footer>
</template>
