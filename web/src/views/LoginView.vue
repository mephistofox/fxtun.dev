<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Card from '@/components/ui/Card.vue'

const authStore = useAuthStore()
const themeStore = useThemeStore()
const { t } = useI18n()

const phone = ref('')
const password = ref('')
const totpCode = ref('')
const showTotp = ref(false)
const error = ref('')

async function handleSubmit() {
  error.value = ''
  try {
    await authStore.login({
      phone: phone.value,
      password: password.value,
      totp_code: showTotp.value ? totpCode.value : undefined,
    })
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string; code?: string } } }
    if (err.response?.data?.code === 'totp_required') {
      showTotp.value = true
    } else {
      error.value = err.response?.data?.error || t('auth.loginFailed')
    }
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
  <div class="min-h-screen flex items-center justify-center bg-background p-4">
    <!-- Theme and Language Switchers -->
    <div class="fixed top-4 right-4 flex items-center space-x-2">
      <button
        @click="cycleTheme"
        class="p-2 rounded-md hover:bg-accent transition-colors"
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
        class="px-2 py-1 text-sm font-medium rounded-md hover:bg-accent transition-colors"
      >
        {{ getLocale() === 'en' ? 'RU' : 'EN' }}
      </button>
    </div>

    <Card class="w-full max-w-md p-6">
      <div class="text-center mb-6">
        <h1 class="text-2xl font-bold">fxTunnel</h1>
        <p class="text-muted-foreground mt-1">{{ t('auth.signInTitle') }}</p>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
          {{ error }}
        </div>

        <div class="space-y-2">
          <label class="text-sm font-medium">{{ t('auth.phone') }}</label>
          <Input v-model="phone" type="tel" placeholder="+7 (999) 123-45-67" required />
        </div>

        <div class="space-y-2">
          <label class="text-sm font-medium">{{ t('auth.password') }}</label>
          <Input v-model="password" type="password" :placeholder="t('auth.password')" required />
        </div>

        <div v-if="showTotp" class="space-y-2">
          <label class="text-sm font-medium">{{ t('auth.totpCode') }}</label>
          <Input v-model="totpCode" type="text" placeholder="123456" maxlength="6" required />
        </div>

        <Button type="submit" class="w-full" :loading="authStore.loading">{{ t('auth.signIn') }}</Button>
      </form>

      <p class="text-center text-sm text-muted-foreground mt-4">
        {{ t('auth.noAccount') }}
        <RouterLink to="/register" class="text-primary hover:underline">{{ t('auth.signUp') }}</RouterLink>
      </p>
    </Card>
  </div>
</template>
