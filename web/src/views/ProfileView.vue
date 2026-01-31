<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { useAuthStore } from '@/stores/auth'
import { profileApi, totpApi } from '@/api/client'

const route = useRoute()
const authStore = useAuthStore()
const { t } = useI18n()

// GitHub linking
const githubLinkSuccess = ref(false)
// Google linking
const googleLinkSuccess = ref(false)

// Profile form
const displayName = ref(authStore.user?.display_name || '')
const savingProfile = ref(false)
const profileError = ref('')
const profileSuccess = ref('')

// Password form
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const savingPassword = ref(false)
const passwordError = ref('')
const passwordSuccess = ref('')

// TOTP
const totpEnabled = ref(false)
const showTotpSetup = ref(false)
const totpSecret = ref('')
const totpQrCode = ref('')
const totpCode = ref('')
const totpBackupCodes = ref<string[]>([])
const enablingTotp = ref(false)
const totpError = ref('')
const showDisableTotp = ref(false)
const disableCode = ref('')
const disablingTotp = ref(false)

async function saveProfile() {
  savingProfile.value = true
  profileError.value = ''
  profileSuccess.value = ''
  try {
    await profileApi.update({ display_name: displayName.value })
    await authStore.refreshProfile()
    profileSuccess.value = t('profile.profileUpdated')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    profileError.value = err.response?.data?.error || t('profile.failedToUpdate')
  } finally {
    savingProfile.value = false
  }
}

async function changePassword() {
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = t('auth.passwordsDoNotMatch')
    return
  }
  if (newPassword.value.length < 8) {
    passwordError.value = t('auth.passwordTooShort')
    return
  }

  savingPassword.value = true
  passwordError.value = ''
  passwordSuccess.value = ''
  try {
    await profileApi.changePassword({
      current_password: currentPassword.value,
      new_password: newPassword.value,
    })
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordSuccess.value = t('profile.passwordChanged')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    passwordError.value = err.response?.data?.error || t('profile.failedToChangePassword')
  } finally {
    savingPassword.value = false
  }
}

async function startTotpSetup() {
  totpError.value = ''
  try {
    const response = await totpApi.enable()
    totpSecret.value = response.data.secret
    totpQrCode.value = response.data.qr_code
    showTotpSetup.value = true
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    totpError.value = err.response?.data?.error || t('profile.failedToSetup')
  }
}

async function verifyTotp() {
  enablingTotp.value = true
  totpError.value = ''
  try {
    const response = await totpApi.verify(totpCode.value)
    totpBackupCodes.value = response.data.backup_codes
    totpEnabled.value = true
    await authStore.refreshProfile()
    totpCode.value = ''
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    totpError.value = err.response?.data?.error || t('profile.failedToVerify')
  } finally {
    enablingTotp.value = false
  }
}

async function disableTotp() {
  disablingTotp.value = true
  totpError.value = ''
  try {
    await totpApi.disable(disableCode.value)
    totpEnabled.value = false
    await authStore.refreshProfile()
    showDisableTotp.value = false
    disableCode.value = ''
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    totpError.value = err.response?.data?.error || t('profile.failedToDisable')
  } finally {
    disablingTotp.value = false
  }
}

function closeTotpSetup() {
  showTotpSetup.value = false
  totpSecret.value = ''
  totpQrCode.value = ''
  totpCode.value = ''
  totpBackupCodes.value = []
}

async function loadProfile() {
  try {
    const response = await profileApi.get()
    totpEnabled.value = response.data.totp_enabled
  } catch {
    // Ignore errors
  }
}

function getGitHubLinkUrl() {
  const token = localStorage.getItem('accessToken')
  return `/api/auth/github?link=true&token=${token}`
}

function getGoogleLinkUrl() {
  const token = localStorage.getItem('accessToken')
  return `/api/auth/google?link=true&token=${token}`
}

onMounted(() => {
  loadProfile()
  if (route.query.github_linked === 'true') {
    githubLinkSuccess.value = true
    authStore.refreshProfile()
  }
  if (route.query.google_linked === 'true') {
    googleLinkSuccess.value = true
    authStore.refreshProfile()
  }
})
</script>

<template>
  <Layout>
    <div class="max-w-2xl space-y-6">
      <div>
        <h1 class="text-2xl font-bold">{{ t('profile.title') }}</h1>
        <p class="text-muted-foreground">{{ t('profile.subtitle') }}</p>
      </div>

      <!-- Profile Section -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold mb-4">{{ t('profile.profileSection') }}</h2>
        <form @submit.prevent="saveProfile" class="space-y-4">
          <div v-if="profileError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
            {{ profileError }}
          </div>
          <div v-if="profileSuccess" class="bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 p-3 rounded-md text-sm">
            {{ profileSuccess }}
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium">{{ t('auth.phone') }}</label>
            <Input :value="authStore.user?.phone" disabled />
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium">{{ t('auth.displayName') }}</label>
            <Input v-model="displayName" :placeholder="t('profile.yourName')" />
          </div>

          <Button type="submit" :loading="savingProfile">{{ t('profile.saveChanges') }}</Button>
        </form>
      </Card>

      <!-- Password Section -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold mb-4">{{ t('profile.passwordSection') }}</h2>
        <form @submit.prevent="changePassword" class="space-y-4">
          <div v-if="passwordError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
            {{ passwordError }}
          </div>
          <div v-if="passwordSuccess" class="bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 p-3 rounded-md text-sm">
            {{ passwordSuccess }}
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium">{{ t('profile.currentPassword') }}</label>
            <Input v-model="currentPassword" type="password" required />
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium">{{ t('profile.newPassword') }}</label>
            <Input v-model="newPassword" type="password" :placeholder="t('profile.minChars')" required />
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium">{{ t('profile.confirmNewPassword') }}</label>
            <Input v-model="confirmPassword" type="password" required />
          </div>

          <Button type="submit" :loading="savingPassword">{{ t('profile.changePassword') }}</Button>
        </form>
      </Card>

      <!-- GitHub Section -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold mb-4">{{ t('profile.githubSection') }}</h2>

        <div v-if="githubLinkSuccess" class="bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 p-3 rounded-md text-sm mb-4">
          {{ t('profile.githubLinkSuccess') }}
        </div>

        <div v-if="authStore.user?.github_id" class="flex items-center text-sm text-green-600 dark:text-green-400">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
            <polyline points="22 4 12 14.01 9 11.01" />
          </svg>
          {{ t('profile.githubLinked') }}
        </div>

        <div v-else class="space-y-4">
          <p class="text-sm text-muted-foreground">{{ t('profile.githubNotLinked') }}</p>
          <a
            :href="getGitHubLinkUrl()"
            class="inline-flex items-center gap-2 rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium hover:bg-accent/10 transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
            </svg>
            {{ t('profile.linkGitHub') }}
          </a>
        </div>
      </Card>

      <!-- Google Section -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold mb-4">{{ t('profile.googleSection') }}</h2>

        <div v-if="googleLinkSuccess" class="bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 p-3 rounded-md text-sm mb-4">
          {{ t('profile.googleLinkSuccess') }}
        </div>

        <div v-if="authStore.user?.google_id" class="flex items-center text-sm text-green-600 dark:text-green-400">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
            <polyline points="22 4 12 14.01 9 11.01" />
          </svg>
          {{ t('profile.googleLinked') }}
        </div>

        <div v-else class="space-y-4">
          <p class="text-sm text-muted-foreground">{{ t('profile.googleNotLinked') }}</p>
          <a
            :href="getGoogleLinkUrl()"
            class="inline-flex items-center gap-2 rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium hover:bg-accent/10 transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
            </svg>
            {{ t('profile.linkGoogle') }}
          </a>
        </div>
      </Card>

      <!-- 2FA Section -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold mb-4">{{ t('profile.twoFactorSection') }}</h2>

        <div v-if="totpError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm mb-4">
          {{ totpError }}
        </div>

        <div v-if="totpEnabled" class="space-y-4">
          <p class="text-sm text-green-600 dark:text-green-400 flex items-center">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
              <polyline points="22 4 12 14.01 9 11.01" />
            </svg>
            {{ t('profile.twoFactorEnabled') }}
          </p>
          <Button variant="destructive" @click="showDisableTotp = true">{{ t('profile.disable2FA') }}</Button>
        </div>

        <div v-else class="space-y-4">
          <p class="text-sm text-muted-foreground">
            {{ t('profile.twoFactorHint') }}
          </p>
          <Button @click="startTotpSetup">{{ t('profile.enable2FA') }}</Button>
        </div>

        <!-- TOTP Setup Dialog -->
        <div
          v-if="showTotpSetup"
          class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
        >
          <Card class="w-full max-w-md p-6">
            <template v-if="totpBackupCodes.length === 0">
              <h2 class="text-xl font-bold mb-4">{{ t('profile.setup2FA') }}</h2>
              <div class="space-y-4">
                <p class="text-sm">{{ t('profile.scanQR') }}</p>
                <div class="flex justify-center">
                  <img :src="totpQrCode" alt="TOTP QR Code" class="border rounded" />
                </div>
                <p class="text-xs text-muted-foreground text-center">
                  {{ t('profile.manualEntry') }} <code class="bg-muted px-1">{{ totpSecret }}</code>
                </p>
                <div class="space-y-2">
                  <label class="text-sm font-medium">{{ t('profile.verificationCode') }}</label>
                  <Input v-model="totpCode" placeholder="123456" maxlength="6" />
                </div>
                <div class="flex space-x-2">
                  <Button variant="outline" @click="closeTotpSetup" class="flex-1">{{ t('common.cancel') }}</Button>
                  <Button @click="verifyTotp" :loading="enablingTotp" class="flex-1">{{ t('profile.verify') }}</Button>
                </div>
              </div>
            </template>
            <template v-else>
              <h2 class="text-xl font-bold mb-4">{{ t('profile.backupCodes') }}</h2>
              <div class="space-y-4">
                <div class="bg-yellow-50 dark:bg-yellow-900/30 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
                  <p class="text-sm text-yellow-800 dark:text-yellow-200">
                    {{ t('profile.backupCodesHint') }}
                  </p>
                </div>
                <div class="bg-muted p-4 rounded font-mono text-sm grid grid-cols-2 gap-2">
                  <span v-for="code in totpBackupCodes" :key="code">{{ code }}</span>
                </div>
                <Button @click="closeTotpSetup" class="w-full">{{ t('common.done') }}</Button>
              </div>
            </template>
          </Card>
        </div>

        <!-- Disable TOTP Dialog -->
        <div
          v-if="showDisableTotp"
          class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
        >
          <Card class="w-full max-w-md p-6">
            <h2 class="text-xl font-bold mb-4">{{ t('profile.disable2FATitle') }}</h2>
            <div class="space-y-4">
              <p class="text-sm">{{ t('profile.disable2FAHint') }}</p>
              <Input v-model="disableCode" placeholder="123456" maxlength="6" />
              <div class="flex space-x-2">
                <Button variant="outline" @click="showDisableTotp = false" class="flex-1">{{ t('common.cancel') }}</Button>
                <Button variant="destructive" @click="disableTotp" :loading="disablingTotp" class="flex-1">{{ t('profile.disable2FA') }}</Button>
              </div>
            </div>
          </Card>
        </div>
      </Card>
    </div>
  </Layout>
</template>
