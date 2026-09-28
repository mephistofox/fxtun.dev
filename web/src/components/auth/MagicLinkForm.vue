<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { isAxiosError } from 'axios'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/client'
import { getLocale } from '@/i18n'

const props = defineProps<{ mode: 'login' | 'register' }>()

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const step = ref<'email' | 'sent'>('email')
const email = ref('')
const code = ref('')
const totp = ref('')
const needTotp = ref(false)
const loading = ref(false)
const error = ref('')
const info = ref('')

const submitLabel = computed(() =>
  props.mode === 'register' ? t('auth.signUpWithEmail') : t('auth.signInWithEmail'),
)

function lang(): string {
  return getLocale() === 'en' ? 'en' : 'ru'
}

function apiError(e: unknown, fallback: string): string {
  if (isAxiosError(e)) {
    const data = e.response?.data as { error?: string } | undefined
    if (data?.error) return data.error
    if (e.response?.status === 429) return t('auth.magicLinkTooMany')
  }
  return fallback
}

async function sendLink() {
  if (!email.value.trim()) return
  loading.value = true
  error.value = ''
  info.value = ''
  try {
    const res = await authApi.sendMagicLink(email.value.trim(), lang())
    info.value = res.data.message || t('auth.magicLinkSent')
    step.value = 'sent'
  } catch (e) {
    error.value = apiError(e, t('auth.magicLinkSendFailed'))
  } finally {
    loading.value = false
  }
}

function errorCode(e: unknown): string | undefined {
  if (isAxiosError(e)) {
    return (e.response?.data as { code?: string } | undefined)?.code
  }
  return undefined
}

async function verifyCode() {
  if (code.value.trim().length !== 6) return
  if (needTotp.value && totp.value.trim().length < 6) return
  loading.value = true
  error.value = ''
  try {
    const res = await authApi.verifyMagicLink({
      email: email.value.trim(),
      code: code.value.trim(),
      totp_code: needTotp.value ? totp.value.trim() : undefined,
    })
    localStorage.setItem('accessToken', res.data.access_token)
    localStorage.setItem('refreshToken', res.data.refresh_token)
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
    // 2FA-protected account: reveal the TOTP field and let the user retry.
    if (errorCode(e) === 'TOTP_REQUIRED') {
      needTotp.value = true
      error.value = ''
    } else {
      error.value = apiError(e, t('auth.magicLinkInvalid'))
    }
  } finally {
    loading.value = false
  }
}

function reset() {
  step.value = 'email'
  code.value = ''
  totp.value = ''
  needTotp.value = false
  error.value = ''
  info.value = ''
}
</script>

<template>
  <div class="space-y-3">
    <!-- Step 1: enter email -->
    <form v-if="step === 'email'" class="space-y-3" @submit.prevent="sendLink">
      <input
        v-model="email"
        type="email"
        autocomplete="email"
        required
        :placeholder="t('auth.emailPlaceholder')"
        class="w-full rounded-lg border border-border bg-card px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/40"
      />
      <button
        type="submit"
        :disabled="loading"
        class="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground hover:opacity-90 transition disabled:opacity-60"
      >
        <svg v-if="loading" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12a9 9 0 1 1-6.219-8.56" />
        </svg>
        {{ submitLabel }}
      </button>
    </form>

    <!-- Step 2: email sent, enter the 6-digit code (or use the link) -->
    <form v-else class="space-y-3" @submit.prevent="verifyCode">
      <p class="text-sm text-muted-foreground text-center">{{ info || t('auth.magicLinkSent') }}</p>
      <input
        v-model="code"
        inputmode="numeric"
        autocomplete="one-time-code"
        maxlength="6"
        :placeholder="t('auth.magicLinkCodePlaceholder')"
        class="w-full rounded-lg border border-border bg-card px-4 py-2.5 text-center text-lg font-mono tracking-[0.4em] focus:outline-none focus:ring-2 focus:ring-primary/40"
      />
      <input
        v-if="needTotp"
        v-model="totp"
        inputmode="numeric"
        autocomplete="one-time-code"
        maxlength="8"
        :placeholder="t('auth.totpCode')"
        class="w-full rounded-lg border border-border bg-card px-4 py-2.5 text-center text-lg font-mono tracking-[0.3em] focus:outline-none focus:ring-2 focus:ring-primary/40"
      />
      <button
        type="submit"
        :disabled="loading || code.trim().length !== 6"
        class="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground hover:opacity-90 transition disabled:opacity-60"
      >
        <svg v-if="loading" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12a9 9 0 1 1-6.219-8.56" />
        </svg>
        {{ t('auth.magicLinkVerify') }}
      </button>
      <div class="flex items-center justify-between text-xs">
        <button type="button" class="text-muted-foreground hover:text-foreground" @click="reset">
          {{ t('auth.magicLinkChangeEmail') }}
        </button>
        <button type="button" class="text-primary hover:underline" :disabled="loading" @click="sendLink">
          {{ t('auth.magicLinkResend') }}
        </button>
      </div>
    </form>

    <p v-if="error" class="text-sm text-destructive text-center">{{ error }}</p>
  </div>
</template>
