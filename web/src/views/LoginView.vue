<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'
import { setLocale, getLocale } from '@/i18n'
import { useSeo } from '@/composables/useSeo'
import Card from '@/components/ui/Card.vue'

const themeStore = useThemeStore()
const authStore = useAuthStore()
const { t, locale } = useI18n()

useSeo({ titleKey: 'seo.login.title', descriptionKey: 'seo.login.description', path: '/login' })

const showOffer = computed(() => locale.value === 'ru')

const email = ref('')
const password = ref('')
const totpCode = ref('')
const needTotp = ref(false)
const formError = ref('')
const submitting = ref(false)

async function handleEmailLogin() {
  formError.value = ''
  submitting.value = true

  try {
    await authStore.login({
      phone: email.value,
      password: password.value,
      ...(needTotp.value && totpCode.value ? { totp_code: totpCode.value } : {}),
    })
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    const msg = err.response?.data?.error || ''
    if (msg === 'TOTP_REQUIRED') {
      needTotp.value = true
      formError.value = ''
    } else {
      formError.value = msg || t('auth.loginFailed')
    }
  } finally {
    submitting.value = false
  }
}

function toggleLocale() {
  const current = getLocale()
  setLocale(current === 'en' ? 'ru' : 'en')
}

function cycleTheme() {
  const modes: ThemeMode[] = ['light', 'dark', 'system']
  const currentIndex = modes.indexOf(themeStore.mode)
  const nextIndex = (currentIndex + 1) % modes.length
  themeStore.setMode(modes[nextIndex])
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center hero-gradient p-4">
    <!-- Theme and Language Switchers -->
    <div class="fixed top-4 right-4 flex items-center space-x-2">
      <button
        @click="cycleTheme"
        class="p-2 rounded-lg hover:bg-accent/10 transition-colors"
        :title="t(`theme.${themeStore.mode}`)"
      >
        <svg
          v-if="themeStore.mode === 'light'"
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
        <svg
          v-else-if="themeStore.mode === 'dark'"
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
        <svg
          v-else
          xmlns="http://www.w3.org/2000/svg"
          class="h-5 w-5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
          <line x1="8" y1="21" x2="16" y2="21" />
          <line x1="12" y1="17" x2="12" y2="21" />
        </svg>
      </button>
      <button
        @click="toggleLocale"
        class="px-2 py-1 text-sm font-medium rounded-lg hover:bg-accent/10 transition-colors"
      >
        {{ getLocale() === 'en' ? 'RU' : 'EN' }}
      </button>
    </div>

    <!-- Back to landing -->
    <RouterLink
      to="/"
      class="fixed top-4 left-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M9.707 16.707a1 1 0 01-1.414 0l-6-6a1 1 0 010-1.414l6-6a1 1 0 011.414 1.414L5.414 9H17a1 1 0 110 2H5.414l4.293 4.293a1 1 0 010 1.414z" clip-rule="evenodd" />
      </svg>
      {{ t('landing.nav.backToHome') }}
    </RouterLink>

    <Card variant="glass" class="w-full max-w-md p-8 animate-fade-in-up">
      <div class="text-center mb-8">
        <div class="flex items-center justify-center gap-3 mb-2">
          <div class="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <h1 class="text-2xl font-bold">fxTunnel</h1>
        </div>
        <p class="text-muted-foreground mt-2">{{ t('auth.signInTitle') }}</p>
      </div>

      <div class="space-y-3">
        <a
          href="/api/auth/github"
          class="w-full inline-flex items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 py-2.5 text-sm font-medium hover:bg-accent/10 transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
          </svg>
          {{ t('auth.signInWithGitHub') }}
        </a>

        <a
          href="/api/auth/google"
          class="w-full inline-flex items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 py-2.5 text-sm font-medium hover:bg-accent/10 transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
          </svg>
          {{ t('auth.signInWithGoogle') }}
        </a>
      </div>

      <!-- Divider -->
      <div class="flex items-center gap-3 my-5">
        <div class="flex-1 h-px bg-border"></div>
        <span class="text-xs text-muted-foreground">{{ t('auth.or') }}</span>
        <div class="flex-1 h-px bg-border"></div>
      </div>

      <!-- Email + Password form -->
      <form @submit.prevent="handleEmailLogin" class="space-y-3">
        <div v-if="formError" class="rounded-lg bg-destructive/10 border border-destructive/20 px-3 py-2 text-sm text-destructive">
          {{ formError }}
        </div>

        <template v-if="!needTotp">
          <input
            v-model="email"
            type="email"
            required
            :placeholder="t('auth.email')"
            class="w-full rounded-lg border border-border bg-card px-4 py-2.5 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-colors"
          />
          <input
            v-model="password"
            type="password"
            required
            :placeholder="t('auth.password')"
            class="w-full rounded-lg border border-border bg-card px-4 py-2.5 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-colors"
          />
        </template>

        <template v-else>
          <input
            v-model="totpCode"
            type="text"
            inputmode="numeric"
            autocomplete="one-time-code"
            required
            :placeholder="t('auth.totpCode')"
            class="w-full rounded-lg border border-border bg-card px-4 py-2.5 text-sm text-center tracking-widest placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-colors"
          />
        </template>

        <button
          type="submit"
          :disabled="submitting"
          class="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {{ submitting ? '...' : t('auth.signIn') }}
        </button>
      </form>

      <!-- Tunnel illustration -->
      <div class="my-8 flex justify-center">
        <div class="relative w-full max-w-xs">
          <!-- Tunnel line - absolute, spans full width at icon center height -->
          <div class="absolute left-6 right-6 top-6 h-0.5">
            <!-- Line background with solid center, transparent edges -->
            <div class="absolute inset-0 rounded-full tunnel-line-bg"></div>
            <!-- Moving dots - outgoing green -->
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-out dot-1"></div>
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-out dot-2"></div>
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-out dot-3"></div>
            <!-- Moving dots - incoming violet -->
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-in dot-4"></div>
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-in dot-5"></div>
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-in dot-6"></div>
            <!-- Moving dots - blue -->
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-blue dot-7"></div>
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-blue dot-8"></div>
            <div class="absolute top-1/2 w-1 h-1 rounded-full dot-blue-in dot-9"></div>
          </div>
          <div class="flex items-center justify-between relative z-10">
            <!-- Local server -->
            <div class="flex flex-col items-center">
              <div class="w-12 h-12 rounded-lg bg-surface border border-primary/20 flex items-center justify-center">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" />
                </svg>
              </div>
              <span class="text-xs text-muted-foreground mt-2">localhost</span>
            </div>
            <!-- Internet/Globe -->
            <div class="flex flex-col items-center">
              <div class="w-12 h-12 rounded-lg bg-surface border border-primary/20 flex items-center justify-center">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" />
                </svg>
              </div>
              <span class="text-xs text-muted-foreground mt-2">internet</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Features list -->
      <div class="grid grid-cols-3 gap-3 text-center text-xs">
        <div class="p-3 rounded-lg bg-muted/30">
          <div class="text-primary font-semibold">HTTP</div>
          <div class="text-muted-foreground">TCP / UDP</div>
        </div>
        <div class="p-3 rounded-lg bg-muted/30">
          <div class="text-primary font-semibold">TLS</div>
          <div class="text-muted-foreground">{{ t('auth.encryption') }}</div>
        </div>
        <div class="p-3 rounded-lg bg-muted/30">
          <div class="text-primary font-semibold">CLI</div>
          <div class="text-muted-foreground">{{ t('auth.crossPlatform') }}</div>
        </div>
      </div>

      <!-- Terms agreement (only on fxtun.ru) -->
      <p v-if="showOffer" class="mt-6 text-center text-xs text-muted-foreground">
        {{ t('legal.offerAgreement') }}
        <RouterLink to="/offer" class="text-primary hover:underline">
          {{ t('legal.offer') }}
        </RouterLink>
      </p>

    </Card>
  </div>
</template>

<style scoped>
.tunnel-line-bg {
  background: linear-gradient(to right,
    transparent 0%,
    hsl(var(--muted-foreground) / 0.3) 15%,
    hsl(var(--muted-foreground) / 0.3) 85%,
    transparent 100%
  );
}

.dot-out {
  animation: move-out-green var(--duration) ease-in-out infinite;
  animation-delay: var(--delay);
}

.dot-in {
  animation: move-in-violet var(--duration) ease-in-out infinite;
  animation-delay: var(--delay);
}

.dot-blue {
  animation: move-out-blue var(--duration) ease-in-out infinite;
  animation-delay: var(--delay);
}

.dot-blue-in {
  animation: move-in-cyan var(--duration) ease-in-out infinite;
  animation-delay: var(--delay);
}

/* Color shifting variants */
.dot-1 { animation-name: move-out-green-violet-blue; }
.dot-3 { animation-name: move-out-green-to-cyan; }
.dot-5 { animation-name: move-in-violet-blue-green; }
.dot-6 { animation-name: move-in-pink-to-violet; }
.dot-7 { animation-name: move-out-blue-green-violet; }
.dot-9 { animation-name: move-in-cyan-to-pink; }

.dot-1 { --duration: 2.2s; --delay: 0s; }
.dot-2 { --duration: 2.4s; --delay: 0.25s; }
.dot-3 { --duration: 2s; --delay: 0.5s; }
.dot-4 { --duration: 2.3s; --delay: 0.15s; }
.dot-5 { --duration: 2.1s; --delay: 0.4s; }
.dot-6 { --duration: 2.5s; --delay: 0.65s; }
.dot-7 { --duration: 2.2s; --delay: 0.1s; }
.dot-8 { --duration: 2.3s; --delay: 0.35s; }
.dot-9 { --duration: 2s; --delay: 0.55s; }

/* Green outgoing */
@keyframes move-out-green {
  0% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(var(--primary));
    box-shadow: 0 0 4px 1px hsl(var(--primary));
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(var(--primary));
    box-shadow: 0 0 4px 1px hsl(var(--primary));
  }
}

/* Green -> Violet -> Blue (rainbow out) */
@keyframes move-out-green-violet-blue {
  0% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(var(--primary));
    box-shadow: 0 0 4px 1px hsl(var(--primary));
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  35% {
    background-color: hsl(280 70% 65%);
    box-shadow: 0 0 4px 1px hsl(280 70% 65%);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
  }
  65% {
    background-color: hsl(210 80% 60%);
    box-shadow: 0 0 4px 1px hsl(210 80% 60%);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(180 70% 50%);
    box-shadow: 0 0 4px 1px hsl(180 70% 50%);
  }
}

/* Green to Cyan */
@keyframes move-out-green-to-cyan {
  0% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(var(--primary));
    box-shadow: 0 0 4px 1px hsl(var(--primary));
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
    background-color: hsl(180 70% 50%);
    box-shadow: 0 0 4px 1px hsl(180 70% 50%);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(180 70% 50%);
    box-shadow: 0 0 4px 1px hsl(180 70% 50%);
  }
}

/* Violet incoming */
@keyframes move-in-violet {
  0% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(280 70% 65%);
    box-shadow: 0 0 4px 1px hsl(280 70% 65%);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(280 70% 65%);
    box-shadow: 0 0 4px 1px hsl(280 70% 65%);
  }
}

/* Violet -> Blue -> Green (rainbow in) */
@keyframes move-in-violet-blue-green {
  0% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(280 70% 65%);
    box-shadow: 0 0 4px 1px hsl(280 70% 65%);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  35% {
    background-color: hsl(210 80% 60%);
    box-shadow: 0 0 4px 1px hsl(210 80% 60%);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
  }
  65% {
    background-color: hsl(var(--primary));
    box-shadow: 0 0 4px 1px hsl(var(--primary));
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(var(--primary));
    box-shadow: 0 0 4px 1px hsl(var(--primary));
  }
}

/* Pink to Violet */
@keyframes move-in-pink-to-violet {
  0% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(330 70% 60%);
    box-shadow: 0 0 4px 1px hsl(330 70% 60%);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
    background-color: hsl(280 70% 65%);
    box-shadow: 0 0 4px 1px hsl(280 70% 65%);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(260 70% 60%);
    box-shadow: 0 0 4px 1px hsl(260 70% 60%);
  }
}

/* Blue outgoing */
@keyframes move-out-blue {
  0% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(210 80% 60%);
    box-shadow: 0 0 4px 1px hsl(210 80% 60%);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(210 80% 60%);
    box-shadow: 0 0 4px 1px hsl(210 80% 60%);
  }
}

/* Blue -> Green -> Violet */
@keyframes move-out-blue-green-violet {
  0% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(210 80% 60%);
    box-shadow: 0 0 4px 1px hsl(210 80% 60%);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  35% {
    background-color: hsl(var(--primary));
    box-shadow: 0 0 4px 1px hsl(var(--primary));
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
  }
  65% {
    background-color: hsl(280 70% 65%);
    box-shadow: 0 0 4px 1px hsl(280 70% 65%);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(330 70% 60%);
    box-shadow: 0 0 4px 1px hsl(330 70% 60%);
  }
}

/* Cyan incoming */
@keyframes move-in-cyan {
  0% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(180 70% 50%);
    box-shadow: 0 0 4px 1px hsl(180 70% 50%);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(180 70% 50%);
    box-shadow: 0 0 4px 1px hsl(180 70% 50%);
  }
}

/* Cyan to Pink */
@keyframes move-in-cyan-to-pink {
  0% {
    left: calc(100% - 4px);
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(180 70% 50%);
    box-shadow: 0 0 4px 1px hsl(180 70% 50%);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  50% {
    transform: translateY(-50%) scaleX(2.5);
    background-color: hsl(210 80% 60%);
    box-shadow: 0 0 4px 1px hsl(210 80% 60%);
  }
  70% {
    opacity: 1;
    transform: translateY(-50%) scaleX(1.5);
  }
  100% {
    left: 0;
    opacity: 0;
    transform: translateY(-50%) scaleX(1);
    background-color: hsl(330 70% 60%);
    box-shadow: 0 0 4px 1px hsl(330 70% 60%);
  }
}
</style>
