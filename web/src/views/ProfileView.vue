<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { useAuthStore } from '@/stores/auth'
import { profileApi, totpApi, subscriptionApi, authApi, type ProfileResponse, type Subscription } from '@/api/client'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const profile = ref<ProfileResponse | null>(null)

// Subscription
const subscription = ref<Subscription | null>(null)
const cancellingSubscription = ref(false)
const subscriptionError = ref('')

// GitHub linking
const githubLinkSuccess = ref(false)
// Google linking
const googleLinkSuccess = ref(false)

// Profile form
const displayName = ref(authStore.user?.display_name || '')
const savingProfile = ref(false)
const profileError = ref('')
const profileSuccess = ref('')

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
    profile.value = response.data
    totpEnabled.value = response.data.totp_enabled
  } catch {
    // Ignore errors
  }
}

async function loadSubscription() {
  try {
    const response = await subscriptionApi.get()
    subscription.value = response.data.subscription
  } catch {
    // Ignore errors
  }
}

async function cancelSubscription() {
  if (!confirm(t('profile.confirmCancelSubscription'))) return

  cancellingSubscription.value = true
  subscriptionError.value = ''
  try {
    await subscriptionApi.cancel()
    await loadSubscription()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    subscriptionError.value = err.response?.data?.error || t('profile.failedToCancelSubscription')
  } finally {
    cancellingSubscription.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'long',
    year: 'numeric'
  })
}

const oauthLinkLoading = ref(false)

async function linkOAuthAccount(provider: string) {
  oauthLinkLoading.value = true
  try {
    const response = await authApi.initOAuthLink(provider)
    window.location.href = response.data.url
  } catch {
    oauthLinkLoading.value = false
  }
}

onMounted(() => {
  loadProfile()
  loadSubscription()
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
    <div class="space-y-6">

      <!-- Hero Profile Card -->
      <Card class="p-6">
        <div class="flex items-center gap-5">
          <!-- Avatar -->
          <div class="flex-shrink-0 w-16 h-16 rounded-full bg-primary/20 text-primary flex items-center justify-center text-2xl font-bold uppercase select-none">
            {{ (authStore.user?.display_name || authStore.user?.phone || '?').charAt(0) }}
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-3 flex-wrap">
              <h1 class="text-xl font-bold truncate">{{ authStore.user?.display_name || authStore.user?.phone }}</h1>
              <span v-if="profile?.plan" class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary/15 text-primary border border-primary/25">
                {{ profile.plan.name }}
              </span>
            </div>
            <p class="text-sm text-muted-foreground mt-0.5 truncate">{{ authStore.user?.email || authStore.user?.phone }}</p>
          </div>
        </div>
      </Card>

      <!-- Two-column grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">

        <!-- LEFT COLUMN -->
        <div class="space-y-6">

          <!-- Profile Card -->
          <Card class="p-6">
            <h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-4">{{ t('profile.profileSection') }}</h2>
            <form @submit.prevent="saveProfile" class="space-y-4">
              <div v-if="profileError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
                {{ profileError }}
              </div>
              <div v-if="profileSuccess" class="bg-green-900/30 text-green-300 p-3 rounded-md text-sm">
                {{ profileSuccess }}
              </div>

              <div class="space-y-2">
                <label class="text-sm font-medium">{{ t('auth.email') }}</label>
                <Input :value="authStore.user?.email || authStore.user?.phone" disabled />
              </div>

              <div class="space-y-2">
                <label class="text-sm font-medium">{{ t('auth.displayName') }}</label>
                <Input v-model="displayName" :placeholder="t('profile.yourName')" />
              </div>

              <Button type="submit" :loading="savingProfile">{{ t('profile.saveChanges') }}</Button>
            </form>
          </Card>

        </div>

        <!-- RIGHT COLUMN -->
        <div class="space-y-6">

          <!-- Plan & Limits Card -->
          <Card v-if="profile?.plan" class="p-6">
            <h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-4">Plan & Limits</h2>
            <div class="text-lg font-bold mb-4">{{ profile.plan.name }}</div>
            <div class="grid grid-cols-2 gap-3">
              <!-- Tunnels -->
              <div class="flex items-center gap-3 rounded-lg bg-muted/50 border border-border px-3 py-2.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"/></svg>
                <div class="min-w-0">
                  <div class="text-xs text-muted-foreground">Tunnels</div>
                  <div class="text-sm font-semibold font-mono">{{ profile.plan.max_tunnels < 0 ? '\u221E' : profile.plan.max_tunnels }}</div>
                </div>
              </div>
              <!-- Domains -->
              <div class="flex items-center gap-3 rounded-lg bg-muted/50 border border-border px-3 py-2.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
                <div class="min-w-0">
                  <div class="text-xs text-muted-foreground">Domains</div>
                  <div class="text-sm font-semibold font-mono">{{ profile.plan.max_domains < 0 ? '\u221E' : profile.plan.max_domains }}</div>
                </div>
              </div>
              <!-- Custom Domains -->
              <div class="flex items-center gap-3 rounded-lg bg-muted/50 border border-border px-3 py-2.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                <div class="min-w-0">
                  <div class="text-xs text-muted-foreground">Custom Domains</div>
                  <div class="text-sm font-semibold font-mono">{{ profile.plan.max_custom_domains < 0 ? '\u221E' : profile.plan.max_custom_domains }}</div>
                </div>
              </div>
              <!-- Tokens -->
              <div class="flex items-center gap-3 rounded-lg bg-muted/50 border border-border px-3 py-2.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                <div class="min-w-0">
                  <div class="text-xs text-muted-foreground">Tokens</div>
                  <div class="text-sm font-semibold font-mono">{{ profile.plan.max_tokens < 0 ? '\u221E' : profile.plan.max_tokens }}</div>
                </div>
              </div>
              <!-- Tunnels/Token -->
              <div class="flex items-center gap-3 rounded-lg bg-muted/50 border border-border px-3 py-2.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 3h5v5"/><path d="M8 3H3v5"/><path d="M12 22v-8.3a4 4 0 0 0-1.172-2.872L3 3"/><path d="m15 9 6-6"/></svg>
                <div class="min-w-0">
                  <div class="text-xs text-muted-foreground">Tunnels/Token</div>
                  <div class="text-sm font-semibold font-mono">{{ profile.plan.max_tunnels_per_token < 0 ? '\u221E' : profile.plan.max_tunnels_per_token }}</div>
                </div>
              </div>
              <!-- Inspector -->
              <div class="flex items-center gap-3 rounded-lg bg-muted/50 border border-border px-3 py-2.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                <div class="min-w-0">
                  <div class="text-xs text-muted-foreground">Inspector</div>
                  <div class="text-sm font-semibold font-mono" :class="profile.plan.inspector_enabled ? 'text-green-400' : 'text-muted-foreground'">{{ profile.plan.inspector_enabled ? 'Enabled' : 'Disabled' }}</div>
                </div>
              </div>
            </div>
          </Card>

          <!-- Subscription Card -->
          <Card class="p-6">
            <h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-4">{{ t('profile.subscriptionSection') }}</h2>

            <div v-if="subscriptionError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm mb-4">
              {{ subscriptionError }}
            </div>

            <!-- Active subscription -->
            <div v-if="subscription && (subscription.status === 'active' || subscription.status === 'cancelled')">
              <div class="flex items-center justify-between mb-4">
                <div>
                  <span class="text-lg font-bold">{{ subscription.plan?.name || 'Plan' }}</span>
                  <span
                    class="ml-2 px-2 py-0.5 text-xs font-medium rounded-full"
                    :class="subscription.status === 'active' ? 'bg-green-900/30 text-green-400' : 'bg-yellow-900/30 text-yellow-400'"
                  >
                    {{ subscription.status === 'active' ? t('profile.subscriptionActive') : t('profile.subscriptionCancelled') }}
                  </span>
                </div>
              </div>

              <div class="space-y-2 text-sm">
                <div v-if="subscription.current_period_end" class="flex justify-between">
                  <span class="text-muted-foreground">{{ subscription.status === 'active' ? t('profile.renewsOn') : t('profile.expiresOn') }}</span>
                  <span class="font-medium">{{ formatDate(subscription.current_period_end) }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-muted-foreground">{{ t('profile.autoRenewal') }}</span>
                  <span class="font-medium" :class="subscription.recurring ? 'text-green-400' : 'text-muted-foreground'">
                    {{ subscription.recurring ? t('common.yes') : t('common.no') }}
                  </span>
                </div>
                <div v-if="subscription.next_plan" class="flex justify-between">
                  <span class="text-muted-foreground">{{ t('profile.nextPlan') }}</span>
                  <span class="font-medium text-primary">{{ subscription.next_plan.name }}</span>
                </div>
              </div>

              <div class="mt-4 pt-4 border-t border-border flex flex-col sm:flex-row gap-2">
                <Button
                  v-if="subscription.status === 'active' && subscription.recurring"
                  variant="destructive"
                  size="sm"
                  :loading="cancellingSubscription"
                  @click="cancelSubscription"
                >
                  {{ t('profile.cancelSubscription') }}
                </Button>
                <Button
                  size="sm"
                  :variant="subscription.status === 'cancelled' ? 'default' : 'outline'"
                  @click="router.push('/checkout')"
                >
                  {{ subscription.status === 'cancelled' ? t('profile.upgradePlan') : t('profile.changePlan') }}
                </Button>
              </div>
            </div>

            <!-- No subscription -->
            <div v-else class="text-center py-4">
              <p class="text-muted-foreground mb-4">{{ t('profile.noSubscription') }}</p>
              <Button @click="router.push('/checkout')">{{ t('profile.upgradePlan') }}</Button>
            </div>
          </Card>

          <!-- Security Card (GitHub + Google + 2FA) -->
          <Card class="p-6">
            <h2 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-4">Security</h2>

            <!-- GitHub row -->
            <div class="space-y-4 divide-y divide-border">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-muted-foreground" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                  </svg>
                  <div>
                    <div class="text-sm font-medium">GitHub</div>
                    <div v-if="githubLinkSuccess" class="text-xs text-green-400">{{ t('profile.githubLinkSuccess') }}</div>
                  </div>
                </div>
                <div v-if="authStore.user?.github_id" class="flex items-center gap-1.5 text-xs text-green-400 font-medium">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline points="22 4 12 14.01 9 11.01" /></svg>
                  {{ t('profile.githubLinked') }}
                </div>
                <button v-else :disabled="oauthLinkLoading" @click="linkOAuthAccount('github')" class="inline-flex items-center gap-1.5 rounded-md border border-border bg-muted/50 px-3 py-1.5 text-xs font-medium hover:bg-muted transition-colors disabled:opacity-50">
                  {{ t('profile.linkGitHub') }}
                </button>
              </div>

              <!-- Google row -->
              <div class="flex items-center justify-between pt-4">
                <div class="flex items-center gap-3">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24">
                    <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
                    <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
                    <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
                    <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
                  </svg>
                  <div>
                    <div class="text-sm font-medium">Google</div>
                    <div v-if="googleLinkSuccess" class="text-xs text-green-400">{{ t('profile.googleLinkSuccess') }}</div>
                  </div>
                </div>
                <div v-if="authStore.user?.google_id" class="flex items-center gap-1.5 text-xs text-green-400 font-medium">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline points="22 4 12 14.01 9 11.01" /></svg>
                  {{ t('profile.googleLinked') }}
                </div>
                <button v-else :disabled="oauthLinkLoading" @click="linkOAuthAccount('google')" class="inline-flex items-center gap-1.5 rounded-md border border-border bg-muted/50 px-3 py-1.5 text-xs font-medium hover:bg-muted transition-colors disabled:opacity-50">
                  {{ t('profile.linkGoogle') }}
                </button>
              </div>

              <!-- 2FA row (temporarily hidden) -->
              <template v-if="false">
              <div class="flex items-center justify-between pt-4">
                <div class="flex items-center gap-3">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  <div>
                    <div class="text-sm font-medium">{{ t('profile.twoFactorSection') }}</div>
                    <div class="text-xs text-muted-foreground">{{ t('profile.twoFactorHint') }}</div>
                  </div>
                </div>
                <div v-if="totpEnabled" class="flex items-center gap-2">
                  <span class="flex items-center gap-1.5 text-xs text-green-400 font-medium">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline points="22 4 12 14.01 9 11.01" /></svg>
                    {{ t('profile.twoFactorEnabled') }}
                  </span>
                  <Button variant="destructive" size="sm" @click="showDisableTotp = true">{{ t('profile.disable2FA') }}</Button>
                </div>
                <Button v-else size="sm" @click="startTotpSetup">{{ t('profile.enable2FA') }}</Button>
              </div>
              </template>
            </div>

            <!-- TOTP error (temporarily hidden) -->
            <template v-if="false">
            <div v-if="totpError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm mt-4">
              {{ totpError }}
            </div>
            </template>

            <!-- TOTP Setup Dialog (temporarily hidden) -->
            <template v-if="false">
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
            </template>

            <!-- Disable TOTP Dialog (temporarily hidden) -->
            <template v-if="false">
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
            </template>
          </Card>

        </div>
      </div>
    </div>
  </Layout>
</template>
