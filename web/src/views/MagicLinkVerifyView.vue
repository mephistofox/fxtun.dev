<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { isAxiosError } from 'axios'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/client'
import Card from '@/components/ui/Card.vue'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const error = ref('')
const loading = ref(true)
const needTotp = ref(false)
const totp = ref('')
const token = ref('')

function errorCode(e: unknown): string | undefined {
  if (isAxiosError(e)) {
    return (e.response?.data as { code?: string } | undefined)?.code
  }
  return undefined
}

async function verify() {
  loading.value = true
  error.value = ''
  try {
    const response = await authApi.verifyMagicLink({
      token: token.value,
      totp_code: needTotp.value ? totp.value.trim() : undefined,
    })
    localStorage.setItem('accessToken', response.data.access_token)
    localStorage.setItem('refreshToken', response.data.refresh_token)
    await authStore.refreshProfile()
    authStore.initialized = true

    const savedRedirect = localStorage.getItem('authRedirect')
    if (savedRedirect) {
      localStorage.removeItem('authRedirect')
      const safe = savedRedirect.startsWith('/') && !savedRedirect.startsWith('//') ? savedRedirect : undefined
      router.replace(safe || { name: 'dashboard' })
    } else {
      router.replace({ name: 'dashboard' })
    }
  } catch (e) {
    if (errorCode(e) === 'TOTP_REQUIRED') {
      // 2FA account: show the code field and let the user complete sign-in.
      needTotp.value = true
      loading.value = false
    } else {
      error.value = t('auth.magicLinkInvalid')
      loading.value = false
    }
  }
}

onMounted(() => {
  const params = new URLSearchParams(window.location.search)
  const t0 = params.get('token')

  // Clean URL to remove the token from browser history.
  if (window.history.replaceState) {
    window.history.replaceState({}, document.title, window.location.pathname)
  }

  if (!t0) {
    error.value = t('auth.magicLinkInvalid')
    loading.value = false
    return
  }
  token.value = t0
  verify()
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center hero-gradient p-4">
    <Card variant="glass" class="w-full max-w-md p-8 animate-fade-in-up text-center">
      <div v-if="loading" class="space-y-4">
        <div class="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mx-auto">
          <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-primary animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56" />
          </svg>
        </div>
        <p class="text-muted-foreground">{{ t('auth.signingIn') }}</p>
      </div>

      <!-- 2FA step -->
      <form v-else-if="needTotp" class="space-y-4" @submit.prevent="verify">
        <p class="text-sm text-muted-foreground">{{ t('auth.totpCode') }}</p>
        <input
          v-model="totp"
          inputmode="numeric"
          autocomplete="one-time-code"
          maxlength="8"
          :placeholder="t('auth.totpCode')"
          class="w-full rounded-lg border border-border bg-card px-4 py-2.5 text-center text-lg font-mono tracking-[0.3em] focus:outline-none focus:ring-2 focus:ring-primary/40"
        />
        <button
          type="submit"
          :disabled="totp.trim().length < 6"
          class="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground hover:opacity-90 transition disabled:opacity-60"
        >
          {{ t('auth.magicLinkVerify') }}
        </button>
        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
      </form>

      <div v-else class="space-y-4">
        <div class="w-12 h-12 rounded-xl bg-destructive/10 flex items-center justify-center mx-auto">
          <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </div>
        <p class="text-destructive font-medium">{{ error }}</p>
        <router-link to="/login" class="text-primary hover:underline text-sm">
          {{ t('landing.nav.backToHome') }}
        </router-link>
      </div>
    </Card>
  </div>
</template>
