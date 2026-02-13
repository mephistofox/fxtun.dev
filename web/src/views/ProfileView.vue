<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { useAuthStore } from '@/stores/auth'
import { profileApi, subscriptionApi, authApi, type ProfileResponse, type Subscription } from '@/api/client'

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
    <div class="prof-root">
      <!-- ========== HERO HEADER + SUBSCRIPTION ========== -->
      <div class="prof-hero">
        <div class="prof-hero-content">
          <!-- Top row: avatar + info + subscription -->
          <div class="prof-hero-main">
            <div class="prof-avatar">
              {{ (authStore.user?.display_name || authStore.user?.phone || '?').charAt(0) }}
            </div>
            <div class="prof-hero-info">
              <div class="prof-hero-name-row">
                <h1 class="prof-hero-name">{{ authStore.user?.display_name || authStore.user?.phone }}</h1>
                <span v-if="profile?.plan" class="prof-plan-badge">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
                  {{ profile.plan.name }}
                </span>
              </div>
              <p class="prof-hero-email">{{ authStore.user?.email || authStore.user?.phone }}</p>
            </div>
          </div>

          <!-- Subscription strip -->
          <div class="prof-hero-sub">
            <div v-if="subscriptionError" class="prof-alert prof-alert-error">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
              {{ subscriptionError }}
            </div>

            <template v-if="subscription && (subscription.status === 'active' || subscription.status === 'cancelled')">
              <div class="prof-hero-sub-row">
                <div class="prof-hero-sub-details">
                  <span
                    :class="['prof-sub-status', subscription.status === 'active' ? 'prof-sub-active' : 'prof-sub-cancelled']"
                  >
                    {{ subscription.status === 'active' ? t('profile.subscriptionActive') : t('profile.subscriptionCancelled') }}
                  </span>
                  <span v-if="subscription.current_period_end" class="prof-hero-sub-meta">
                    {{ subscription.status === 'active' ? t('profile.renewsOn') : t('profile.expiresOn') }}
                    <strong>{{ formatDate(subscription.current_period_end) }}</strong>
                  </span>
                  <span class="prof-hero-sub-meta">
                    {{ t('profile.autoRenewal') }}:
                    <strong :class="subscription.recurring ? 'prof-val-on' : ''">{{ subscription.recurring ? t('common.yes') : t('common.no') }}</strong>
                  </span>
                  <span v-if="subscription.next_plan" class="prof-hero-sub-meta">
                    {{ t('profile.nextPlan') }}: <strong class="prof-val-primary">{{ subscription.next_plan.name }}</strong>
                  </span>
                </div>
                <div class="prof-hero-sub-actions">
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
            </template>

            <template v-else>
              <div class="prof-hero-sub-row">
                <span class="prof-hero-sub-meta">{{ t('profile.noSubscription') }}</span>
                <Button @click="router.push('/checkout')" size="sm">{{ t('profile.upgradePlan') }}</Button>
              </div>
            </template>
          </div>
        </div>
        <div class="prof-hero-orb prof-hero-orb-1"></div>
        <div class="prof-hero-orb prof-hero-orb-2"></div>
      </div>

      <!-- ========== TWO COLUMN GRID ========== -->
      <div class="prof-grid">

        <!-- ======== LEFT COLUMN ======== -->
        <div class="prof-col">

          <!-- Profile Form Section -->
          <div class="prof-section">
            <div class="prof-section-header">
              <div class="prof-section-icon prof-section-icon-primary">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              </div>
              <h2>{{ t('profile.profileSection') }}</h2>
            </div>
            <form @submit.prevent="saveProfile" class="prof-form">
              <div v-if="profileError" class="prof-alert prof-alert-error">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                {{ profileError }}
              </div>
              <div v-if="profileSuccess" class="prof-alert prof-alert-success">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                {{ profileSuccess }}
              </div>

              <div class="prof-field">
                <label>{{ t('auth.email') }}</label>
                <Input :value="authStore.user?.email || authStore.user?.phone" disabled />
              </div>

              <div class="prof-field">
                <label>{{ t('auth.displayName') }}</label>
                <Input v-model="displayName" :placeholder="t('profile.yourName')" />
              </div>

              <Button type="submit" :loading="savingProfile" class="prof-save-btn">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
                {{ t('profile.saveChanges') }}
              </Button>
            </form>
          </div>

          <!-- Linked Accounts Section -->
          <div class="prof-section">
            <div class="prof-section-header">
              <div class="prof-section-icon prof-section-icon-tcp">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
              </div>
              <h2>{{ t('profile.linkedAccounts') }}</h2>
            </div>

            <div class="prof-oauth-list">
              <!-- GitHub row -->
              <div class="prof-oauth-row">
                <div class="prof-oauth-left">
                  <div class="prof-oauth-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                    </svg>
                  </div>
                  <div>
                    <div class="prof-oauth-name">GitHub</div>
                    <div v-if="githubLinkSuccess" class="prof-oauth-success">{{ t('profile.githubLinkSuccess') }}</div>
                  </div>
                </div>
                <div v-if="authStore.user?.github_id" class="prof-oauth-linked">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline points="22 4 12 14.01 9 11.01" /></svg>
                  {{ t('profile.githubLinked') }}
                </div>
                <button v-else :disabled="oauthLinkLoading" @click="linkOAuthAccount('github')" class="prof-oauth-link-btn">
                  {{ t('profile.linkGitHub') }}
                </button>
              </div>

              <!-- Google row -->
              <div class="prof-oauth-row">
                <div class="prof-oauth-left">
                  <div class="prof-oauth-icon prof-oauth-icon-google">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24">
                      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
                      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
                      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
                      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
                    </svg>
                  </div>
                  <div>
                    <div class="prof-oauth-name">Google</div>
                    <div v-if="googleLinkSuccess" class="prof-oauth-success">{{ t('profile.googleLinkSuccess') }}</div>
                  </div>
                </div>
                <div v-if="authStore.user?.google_id" class="prof-oauth-linked">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline points="22 4 12 14.01 9 11.01" /></svg>
                  {{ t('profile.googleLinked') }}
                </div>
                <button v-else :disabled="oauthLinkLoading" @click="linkOAuthAccount('google')" class="prof-oauth-link-btn">
                  {{ t('profile.linkGoogle') }}
                </button>
              </div>
            </div>
          </div>

        </div>

        <!-- ======== RIGHT COLUMN ======== -->
        <div class="prof-col">

          <!-- Plan & Limits Section -->
          <div v-if="profile?.plan" class="prof-section">
            <div class="prof-section-header">
              <div class="prof-section-icon prof-section-icon-http">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"/></svg>
              </div>
              <h2>{{ t('profile.planAndLimits') }}</h2>
              <span class="prof-plan-name-badge">{{ profile.plan.name }}</span>
            </div>

            <div class="prof-limits-grid">
              <div class="prof-limit-card">
                <svg xmlns="http://www.w3.org/2000/svg" class="prof-limit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"/></svg>
                <span class="prof-limit-label">{{ t('profile.tunnels') }}</span>
                <span class="prof-limit-value">{{ profile.plan.max_tunnels < 0 ? '\u221E' : profile.plan.max_tunnels }}</span>
              </div>
              <div class="prof-limit-card">
                <svg xmlns="http://www.w3.org/2000/svg" class="prof-limit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
                <span class="prof-limit-label">{{ t('profile.domains') }}</span>
                <span class="prof-limit-value">{{ profile.plan.max_domains < 0 ? '\u221E' : profile.plan.max_domains }}</span>
              </div>
              <div class="prof-limit-card">
                <svg xmlns="http://www.w3.org/2000/svg" class="prof-limit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                <span class="prof-limit-label">{{ t('profile.customDomains') }}</span>
                <span class="prof-limit-value">{{ profile.plan.max_custom_domains < 0 ? '\u221E' : profile.plan.max_custom_domains }}</span>
              </div>
              <div class="prof-limit-card">
                <svg xmlns="http://www.w3.org/2000/svg" class="prof-limit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                <span class="prof-limit-label">{{ t('profile.tokens') }}</span>
                <span class="prof-limit-value">{{ profile.plan.max_tokens < 0 ? '\u221E' : profile.plan.max_tokens }}</span>
              </div>
              <div class="prof-limit-card">
                <svg xmlns="http://www.w3.org/2000/svg" class="prof-limit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 3h5v5"/><path d="M8 3H3v5"/><path d="M12 22v-8.3a4 4 0 0 0-1.172-2.872L3 3"/><path d="m15 9 6-6"/></svg>
                <span class="prof-limit-label">{{ t('profile.tunnelsPerToken') }}</span>
                <span class="prof-limit-value">{{ profile.plan.max_tunnels_per_token < 0 ? '\u221E' : profile.plan.max_tunnels_per_token }}</span>
              </div>
              <div class="prof-limit-card">
                <svg xmlns="http://www.w3.org/2000/svg" class="prof-limit-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                <span class="prof-limit-label">{{ t('profile.inspector') }}</span>
                <span :class="['prof-limit-value', profile.plan.inspector_enabled ? 'prof-limit-val-on' : 'prof-limit-val-off']">
                  {{ profile.plan.inspector_enabled ? t('profile.enabled') : t('profile.disabled') }}
                </span>
              </div>
            </div>
          </div>

        </div>
      </div>
    </div>
  </Layout>
</template>

<style scoped>
/* ============================================
   PROFILE — CYBER COMMAND CENTER
   ============================================ */

.prof-root {
  @apply space-y-6;
}

/* ---- Hero ---- */
.prof-hero {
  @apply relative rounded-2xl overflow-hidden p-6 sm:p-8;
  background:
    radial-gradient(ellipse 60% 80% at 20% 0%, hsl(var(--primary) / 0.12) 0%, transparent 60%),
    radial-gradient(ellipse 40% 60% at 90% 80%, hsl(var(--accent) / 0.08) 0%, transparent 50%),
    hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.prof-hero-content {
  @apply relative z-10 space-y-0;
}

.prof-hero-main {
  @apply flex items-center gap-5;
}

.prof-avatar {
  @apply flex-shrink-0 w-16 h-16 sm:w-20 sm:h-20 rounded-2xl flex items-center justify-center text-2xl sm:text-3xl font-bold uppercase select-none font-display;
  background: hsl(var(--primary) / 0.15);
  color: hsl(var(--primary));
  border: 2px solid hsl(var(--primary) / 0.25);
  box-shadow: 0 0 30px hsl(var(--primary) / 0.1);
}

.prof-hero-info {
  @apply min-w-0 flex-1;
}

.prof-hero-name-row {
  @apply flex items-center gap-3 flex-wrap;
}

.prof-hero-name {
  @apply text-2xl sm:text-3xl font-bold tracking-tight font-display truncate;
}

.prof-plan-badge {
  @apply inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider;
  background: hsl(var(--primary) / 0.1);
  border: 1px solid hsl(var(--primary) / 0.2);
  color: hsl(var(--primary));
}

.prof-hero-email {
  @apply text-sm text-muted-foreground mt-1 truncate;
}

/* ---- Subscription strip in hero ---- */
.prof-hero-sub {
  @apply mt-5 pt-5;
  border-top: 1px solid hsl(var(--border) / 0.3);
}

.prof-hero-sub-row {
  @apply flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3;
}

.prof-hero-sub-details {
  @apply flex flex-wrap items-center gap-x-4 gap-y-2;
}

.prof-hero-sub-meta {
  @apply text-sm text-muted-foreground;
}

.prof-hero-sub-meta strong {
  @apply font-semibold;
  color: hsl(var(--foreground));
}

.prof-val-on {
  color: hsl(160 84% 45%);
}

.prof-val-primary {
  color: hsl(var(--primary));
}

.prof-hero-sub-actions {
  @apply flex gap-2 flex-shrink-0;
}

.prof-hero-orb {
  @apply absolute rounded-full pointer-events-none;
  filter: blur(80px);
}

.prof-hero-orb-1 {
  width: 200px;
  height: 200px;
  top: -60px;
  left: -40px;
  background: hsl(var(--primary) / 0.15);
}

.prof-hero-orb-2 {
  width: 150px;
  height: 150px;
  bottom: -50px;
  right: -30px;
  background: hsl(var(--accent) / 0.1);
}

/* ---- Grid ---- */
.prof-grid {
  @apply grid grid-cols-1 md:grid-cols-2 gap-6;
}

.prof-col {
  @apply space-y-6;
}

/* ---- Section Card ---- */
.prof-section {
  @apply rounded-xl p-5 sm:p-6;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.prof-section-header {
  @apply flex items-center gap-3 mb-5;
}

.prof-section-header h2 {
  @apply text-sm font-bold uppercase tracking-wider;
  color: hsl(var(--muted-foreground));
}

.prof-section-icon {
  @apply flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center;
}

.prof-section-icon-primary {
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
}

.prof-section-icon-accent {
  background: hsl(var(--accent) / 0.12);
  color: hsl(var(--accent));
}

.prof-section-icon-http {
  background: hsl(var(--type-http) / 0.12);
  color: hsl(var(--type-http));
}

.prof-section-icon-tcp {
  background: hsl(var(--type-tcp) / 0.12);
  color: hsl(var(--type-tcp));
}

.prof-plan-name-badge {
  @apply ml-auto px-2.5 py-0.5 rounded-full text-xs font-bold;
  background: hsl(var(--type-http) / 0.12);
  color: hsl(var(--type-http));
}

/* ---- Alerts ---- */
.prof-alert {
  @apply flex items-center gap-2 p-3 rounded-xl text-sm mb-4;
}

.prof-alert-error {
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

.prof-alert-success {
  background: hsl(160 84% 45% / 0.1);
  color: hsl(160 84% 45%);
  border: 1px solid hsl(160 84% 45% / 0.2);
}

.prof-alert-warning {
  background: hsl(38 85% 55% / 0.1);
  color: hsl(38 85% 55%);
  border: 1px solid hsl(38 85% 55% / 0.2);
}

/* ---- Form ---- */
.prof-form {
  @apply space-y-4;
}

.prof-field {
  @apply space-y-2;
}

.prof-field label {
  @apply block text-sm font-medium;
}

.prof-save-btn {
  box-shadow: 0 0 20px hsl(var(--primary) / 0.15);
}

/* ---- Limits Grid ---- */
.prof-limits-grid {
  @apply grid grid-cols-2 gap-2.5;
}

.prof-limit-card {
  @apply flex flex-col gap-1.5 p-3 rounded-xl transition-all duration-200;
  background: hsl(var(--muted) / 0.3);
  border: 1px solid hsl(var(--border) / 0.4);
}

.prof-limit-card:hover {
  border-color: hsl(var(--primary) / 0.2);
  background: hsl(var(--muted) / 0.5);
}

.prof-limit-icon {
  @apply h-4 w-4;
  color: hsl(var(--muted-foreground));
}

.prof-limit-label {
  @apply text-[11px] text-muted-foreground uppercase tracking-wider font-medium;
}

.prof-limit-value {
  @apply text-lg font-bold font-mono leading-none;
}

.prof-limit-val-on {
  color: hsl(160 84% 45%);
}

.prof-limit-val-off {
  @apply text-muted-foreground;
}

/* ---- Subscription status pills ---- */
.prof-sub-status {
  @apply px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase tracking-wider;
}

.prof-sub-active {
  background: hsl(160 84% 45% / 0.12);
  color: hsl(160 84% 45%);
  border: 1px solid hsl(160 84% 45% / 0.2);
}

.prof-sub-cancelled {
  background: hsl(38 85% 55% / 0.12);
  color: hsl(38 85% 55%);
  border: 1px solid hsl(38 85% 55% / 0.2);
}

/* ---- OAuth Rows ---- */
.prof-oauth-list {
  @apply space-y-0 divide-y;
  border-color: hsl(var(--border) / 0.4);
}

.prof-oauth-list > * + * {
  border-top: 1px solid hsl(var(--border) / 0.4);
}

.prof-oauth-row {
  @apply flex items-center justify-between py-3.5;
}

.prof-oauth-row:first-child {
  @apply pt-0;
}

.prof-oauth-row:last-child {
  @apply pb-0;
}

.prof-oauth-left {
  @apply flex items-center gap-3;
}

.prof-oauth-icon {
  @apply w-10 h-10 rounded-xl flex items-center justify-center;
  background: hsl(var(--muted) / 0.5);
  color: hsl(var(--muted-foreground));
}

.prof-oauth-icon-google {
  background: hsl(var(--muted) / 0.3);
}

.prof-oauth-name {
  @apply text-sm font-semibold;
}

.prof-oauth-success {
  @apply text-xs;
  color: hsl(160 84% 45%);
}

.prof-oauth-linked {
  @apply flex items-center gap-1.5 text-xs font-bold;
  color: hsl(160 84% 45%);
}

.prof-oauth-link-btn {
  @apply inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-semibold transition-all duration-200;
  background: hsl(var(--muted) / 0.5);
  border: 1px solid hsl(var(--border) / 0.6);
  color: hsl(var(--foreground));
}

.prof-oauth-link-btn:hover {
  background: hsl(var(--muted));
  border-color: hsl(var(--primary) / 0.3);
  box-shadow: 0 0 12px hsl(var(--primary) / 0.08);
}

.prof-oauth-link-btn:disabled {
  @apply opacity-50 cursor-not-allowed;
}

/* ---- Modal ---- */
.prof-modal-overlay {
  @apply fixed inset-0 flex items-center justify-center z-50 p-4;
  background: hsl(0 0% 0% / 0.6);
  backdrop-filter: blur(4px);
}

.prof-modal {
  @apply w-full max-w-md rounded-2xl overflow-hidden;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
  box-shadow: 0 25px 50px -12px hsl(0 0% 0% / 0.4);
}

.prof-modal-body {
  @apply p-6;
}

.prof-modal-actions {
  @apply flex gap-2;
}

.prof-backup-codes {
  @apply p-4 rounded-xl font-mono text-sm grid grid-cols-2 gap-2;
  background: hsl(220 20% 6%);
  border: 1px solid hsl(220 15% 15%);
  color: hsl(75 100% 50%);
}

/* ---- Modal Transitions ---- */
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-active .prof-modal,
.modal-leave-active .prof-modal {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .prof-modal,
.modal-leave-to .prof-modal {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}
</style>
