<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { tokensApi, type APIToken } from '@/api/client'

const { t, locale } = useI18n()

const tokens = ref<APIToken[]>([])
const maxTokens = ref(-1)
const loading = ref(true)
const error = ref('')
const showCreateDialog = ref(false)
const newTokenVisible = ref<string | null>(null)

// Search
const searchQuery = ref('')
const debouncedSearch = ref('')
let searchTimer: ReturnType<typeof setTimeout> | null = null

watch(searchQuery, (val) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    debouncedSearch.value = val.toLowerCase().trim()
  }, 200)
})

const filteredTokens = computed(() => {
  if (!debouncedSearch.value) return tokens.value
  const q = debouncedSearch.value
  return tokens.value.filter(t =>
    t.name.toLowerCase().includes(q) ||
    (t.allowed_subdomains && t.allowed_subdomains.some(s => s.toLowerCase().includes(q)))
  )
})

const createForm = ref({
  name: '',
  allowed_subdomains: '*',
  max_tunnels: 10,
})
const creating = ref(false)
const createError = ref('')

async function loadTokens() {
  loading.value = true
  error.value = ''
  try {
    const response = await tokensApi.list()
    tokens.value = response.data.tokens || []
    if (response.data.max_tokens !== undefined) {
      maxTokens.value = response.data.max_tokens
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('tokens.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function createToken() {
  creating.value = true
  createError.value = ''
  try {
    const subdomains = createForm.value.allowed_subdomains
      .split(',')
      .map((s) => s.trim())
      .filter((s) => s)

    const response = await tokensApi.create({
      name: createForm.value.name,
      allowed_subdomains: subdomains,
      max_tunnels: createForm.value.max_tunnels,
    })

    newTokenVisible.value = response.data.token
    tokens.value.push(response.data.info)

    createForm.value = {
      name: '',
      allowed_subdomains: '*',
      max_tunnels: 10,
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    createError.value = err.response?.data?.error || t('tokens.failedToCreate')
  } finally {
    creating.value = false
  }
}

async function deleteToken(id: number) {
  if (!confirm(t('tokens.confirmDelete'))) return

  try {
    await tokensApi.delete(id)
    tokens.value = tokens.value.filter((t) => t.id !== id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('tokens.failedToDelete')
  }
}

const copiedWhat = ref<'token' | 'command' | ''>('')

async function copyToken() {
  if (newTokenVisible.value) {
    await navigator.clipboard.writeText(newTokenVisible.value)
    copiedWhat.value = 'token'
    setTimeout(() => (copiedWhat.value = ''), 2000)
  }
}

async function copyCommand() {
  if (newTokenVisible.value) {
    await navigator.clipboard.writeText(`fxtunnel login --token ${newTokenVisible.value}`)
    copiedWhat.value = 'command'
    setTimeout(() => (copiedWhat.value = ''), 2000)
  }
}

function closeNewTokenDialog() {
  newTokenVisible.value = null
  showCreateDialog.value = false
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function formatSubdomains(subs: string[]): string {
  if (!subs || subs.length === 0 || (subs.length === 1 && subs[0] === '*')) {
    return t('tokens.allSubdomains')
  }
  return subs.join(', ')
}

onMounted(loadTokens)
</script>

<template>
  <Layout>
    <div class="tokens-root">
      <!-- ========== HERO HEADER ========== -->
      <div class="tok-hero">
        <div class="tok-hero-content">
          <div class="tok-hero-left">
            <div class="tok-status-badge">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
              <span>{{ tokens.length }}/{{ maxTokens < 0 ? '∞' : maxTokens }}</span>
            </div>
            <h1 class="tok-title">{{ t('tokens.title') }}</h1>
            <p class="tok-subtitle">{{ t('tokens.subtitle') }}</p>
          </div>
          <div class="tok-hero-right">
            <div class="tok-search-wrapper">
              <svg xmlns="http://www.w3.org/2000/svg" class="tok-search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('tokens.searchPlaceholder')"
                class="tok-search-input"
              />
              <button v-if="searchQuery" @click="searchQuery = ''" class="tok-search-clear">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
            <Button
              @click="showCreateDialog = true"
              :disabled="maxTokens >= 0 && tokens.length >= maxTokens"
              class="tok-create-btn"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              {{ t('tokens.createToken') }}
            </Button>
          </div>
        </div>
        <div class="tok-hero-orb tok-hero-orb-1"></div>
        <div class="tok-hero-orb tok-hero-orb-2"></div>
      </div>

      <!-- ========== ERROR ========== -->
      <div v-if="error" class="tok-error">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        {{ error }}
      </div>

      <!-- ========== NEW TOKEN DISPLAY DIALOG ========== -->
      <Teleport to="body">
        <Transition name="modal">
          <div v-if="newTokenVisible" class="tok-modal-overlay" @click.self="closeNewTokenDialog">
            <div class="tok-modal tok-modal-lg">
              <div class="tok-modal-body tok-token-display">
                <!-- Success icon -->
                <div class="tok-success-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                </div>
                <h2 class="tok-token-display-title">{{ t('tokens.tokenCreated') }}</h2>

                <!-- Warning -->
                <div class="tok-token-warning">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
                  <p>{{ t('tokens.copyWarning') }}</p>
                </div>

                <!-- Token value -->
                <div class="tok-token-value" @click="copyToken">
                  <code>{{ newTokenVisible }}</code>
                  <span class="tok-token-value-hint">
                    {{ copiedWhat === 'token' ? t('common.copied') : t('common.copy') }}
                  </span>
                </div>

                <!-- CLI command -->
                <div class="tok-token-cmd" @click="copyCommand">
                  <p class="tok-token-cmd-label">{{ t('tokens.copyUsage') }}</p>
                  <code>fxtunnel login --token {{ newTokenVisible }}</code>
                  <span class="tok-token-cmd-hint">
                    {{ copiedWhat === 'command' ? t('common.copied') : t('common.copy') }}
                  </span>
                </div>

                <!-- Actions -->
                <div class="tok-modal-actions tok-modal-actions-3">
                  <Button @click="copyToken" variant="outline" class="flex-1">
                    {{ copiedWhat === 'token' ? t('common.copied') : t('tokens.copyKey') }}
                  </Button>
                  <Button @click="copyCommand" variant="outline" class="flex-1">
                    {{ copiedWhat === 'command' ? t('common.copied') : t('tokens.copyCommand') }}
                  </Button>
                  <Button @click="closeNewTokenDialog" class="flex-1">{{ t('common.done') }}</Button>
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>

      <!-- ========== CREATE TOKEN DIALOG ========== -->
      <Teleport to="body">
        <Transition name="modal">
          <div v-if="showCreateDialog && !newTokenVisible" class="tok-modal-overlay" @click.self="showCreateDialog = false">
            <div class="tok-modal">
              <div class="tok-modal-header">
                <h2>{{ t('tokens.createTitle') }}</h2>
                <p>{{ t('tokens.createHint') }}</p>
              </div>
              <form @submit.prevent="createToken" class="tok-modal-body">
                <div v-if="createError" class="tok-form-error">{{ createError }}</div>

                <div class="tok-form-group">
                  <label>{{ t('tokens.tokenName') }}</label>
                  <Input v-model="createForm.name" :placeholder="t('tokens.tokenNamePlaceholder')" required />
                  <p class="tok-form-hint">{{ t('tokens.tokenNameHint') }}</p>
                </div>

                <details class="tok-advanced group">
                  <summary>
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 tok-advanced-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
                    {{ t('tokens.advancedSettings') }}
                  </summary>
                  <div class="tok-advanced-body">
                    <div class="tok-form-group">
                      <label>{{ t('tokens.allowedSubdomains') }}</label>
                      <Input
                        v-model="createForm.allowed_subdomains"
                        :placeholder="t('tokens.allowedSubdomainsPlaceholder')"
                      />
                      <p class="tok-form-hint">{{ t('tokens.allowedSubdomainsHint') }}</p>
                    </div>

                    <div class="tok-form-group">
                      <label>{{ t('tokens.maxTunnels') }}</label>
                      <input
                        v-model.number="createForm.max_tunnels"
                        type="number"
                        min="1"
                        max="100"
                        class="tok-number-input"
                      />
                      <p class="tok-form-hint">{{ t('tokens.maxTunnelsHint') }}</p>
                    </div>
                  </div>
                </details>

                <div class="tok-modal-actions">
                  <Button type="button" variant="outline" @click="showCreateDialog = false" class="flex-1">
                    {{ t('common.cancel') }}
                  </Button>
                  <Button type="submit" :loading="creating" class="flex-1">{{ t('common.create') }}</Button>
                </div>
              </form>
            </div>
          </div>
        </Transition>
      </Teleport>

      <!-- ========== LOADING ========== -->
      <div v-if="loading" class="tok-loading">
        <div class="tok-loading-spinner"></div>
        <span>{{ t('common.loading') }}</span>
      </div>

      <!-- ========== EMPTY STATE ========== -->
      <div v-else-if="tokens.length === 0" class="tok-empty-wrapper">
        <div class="tok-empty">
          <div class="tok-empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          </div>
          <h2 class="tok-empty-title">{{ t('tokens.noTokens') }}</h2>
          <p class="tok-empty-subtitle">{{ t('tokens.noTokensHint') }}</p>
        </div>

        <!-- How-to card -->
        <div class="tok-howto">
          <h3 class="tok-howto-title">{{ t('tokens.howToUse') }}</h3>
          <ol class="tok-howto-steps">
            <li>
              <span class="tok-howto-num">1</span>
              <span>{{ t('tokens.howToUseStep1') }}</span>
            </li>
            <li>
              <span class="tok-howto-num">2</span>
              <span>{{ t('tokens.howToUseStep2') }}</span>
            </li>
            <li>
              <span class="tok-howto-num">3</span>
              <span>{{ t('tokens.howToUseStep3') }}</span>
            </li>
          </ol>
        </div>
      </div>

      <!-- ========== TOKEN LIST ========== -->
      <template v-else>
        <div class="tok-section-header">
          <h2 class="tok-section-title">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
            {{ t('tokens.title') }}
          </h2>
          <span class="tok-count-badge">{{ filteredTokens.length }}<template v-if="debouncedSearch">/{{ tokens.length }}</template></span>
        </div>

        <div class="tok-list">
          <div
            v-for="token in filteredTokens"
            :key="token.id"
            class="tok-card"
          >
            <div class="tok-card-top">
              <div class="tok-card-meta">
                <div class="tok-card-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
                </div>
                <h3 class="tok-card-name">{{ token.name }}</h3>
              </div>
              <button
                @click="deleteToken(token.id)"
                class="tok-card-delete"
                :title="t('tokens.deleteToken')"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>

            <div class="tok-card-details">
              <div class="tok-detail">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
                <span class="tok-detail-label">{{ t('tokens.subdomains') }}:</span>
                <span class="tok-detail-value">{{ formatSubdomains(token.allowed_subdomains) }}</span>
              </div>
              <div class="tok-detail">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 20V10"/><path d="M12 20V4"/><path d="M6 20v-6"/></svg>
                <span class="tok-detail-label">{{ t('tokens.tunnelsLimit') }}:</span>
                <span class="tok-detail-value">{{ token.max_tunnels }}</span>
              </div>
              <div class="tok-detail">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
                <span class="tok-detail-label">{{ t('tokens.created') }}:</span>
                <span class="tok-detail-value">{{ formatDate(token.created_at) }}</span>
              </div>
              <div class="tok-detail">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
                <span class="tok-detail-label">{{ t('tokens.lastUsed') }}:</span>
                <span :class="['tok-detail-value', !token.last_used_at && 'tok-detail-dim']">
                  {{ token.last_used_at ? formatDate(token.last_used_at) : t('tokens.neverUsed') }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </Layout>
</template>

<style scoped>
/* ============================================
   TOKENS — CYBER COMMAND CENTER
   ============================================ */

.tokens-root {
  @apply space-y-6;
}

/* ---- Hero ---- */
.tok-hero {
  @apply relative rounded-2xl overflow-hidden p-6 sm:p-8;
  background:
    radial-gradient(ellipse 60% 80% at 20% 0%, hsl(var(--accent) / 0.12) 0%, transparent 60%),
    radial-gradient(ellipse 40% 60% at 90% 80%, hsl(var(--primary) / 0.08) 0%, transparent 50%),
    hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.tok-hero-content {
  @apply relative z-10 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4;
}

.tok-hero-left {
  @apply space-y-2;
}

.tok-hero-right {
  @apply flex-shrink-0 flex items-center gap-3;
}

/* ---- Search ---- */
.tok-search-wrapper {
  @apply relative hidden sm:flex items-center;
}

.tok-search-icon {
  @apply absolute left-3 w-4 h-4 pointer-events-none;
  color: hsl(var(--muted-foreground));
}

.tok-search-input {
  @apply h-10 pl-9 pr-8 rounded-xl text-sm transition-all duration-200 w-48 lg:w-56;
  background: hsl(var(--background) / 0.6);
  border: 1px solid hsl(var(--border) / 0.6);
  color: hsl(var(--foreground));
  backdrop-filter: blur(8px);
}

.tok-search-input::placeholder {
  color: hsl(var(--muted-foreground) / 0.7);
}

.tok-search-input:focus {
  outline: none;
  border-color: hsl(var(--accent) / 0.5);
  box-shadow: 0 0 0 3px hsl(var(--accent) / 0.08), 0 0 16px hsl(var(--accent) / 0.06);
  width: 16rem;
}

.tok-search-clear {
  @apply absolute right-2 p-1 rounded-full transition-colors;
  color: hsl(var(--muted-foreground));
}

.tok-search-clear:hover {
  color: hsl(var(--foreground));
  background: hsl(var(--muted) / 0.5);
}

.tok-status-badge {
  @apply inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium;
  background: hsl(var(--accent) / 0.1);
  border: 1px solid hsl(var(--accent) / 0.2);
  color: hsl(var(--accent));
}

.tok-title {
  @apply text-2xl sm:text-3xl font-bold tracking-tight font-display;
}

.tok-subtitle {
  @apply text-sm text-muted-foreground;
}

.tok-create-btn {
  box-shadow: 0 0 20px hsl(var(--primary) / 0.2);
}

.tok-hero-orb {
  @apply absolute rounded-full pointer-events-none;
  filter: blur(80px);
}

.tok-hero-orb-1 {
  width: 200px;
  height: 200px;
  top: -60px;
  left: -40px;
  background: hsl(var(--accent) / 0.15);
}

.tok-hero-orb-2 {
  width: 150px;
  height: 150px;
  bottom: -50px;
  right: -30px;
  background: hsl(var(--primary) / 0.1);
}

/* ---- Error ---- */
.tok-error {
  @apply flex items-center gap-2 p-4 rounded-xl text-sm;
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

/* ---- Loading ---- */
.tok-loading {
  @apply flex items-center justify-center gap-3 py-16 text-muted-foreground;
}

.tok-loading-spinner {
  @apply w-5 h-5 rounded-full border-2 border-current border-t-transparent animate-spin;
}

/* ---- Empty State ---- */
.tok-empty-wrapper {
  @apply space-y-6;
}

.tok-empty {
  @apply text-center space-y-3;
}

.tok-empty-icon {
  @apply mx-auto w-16 h-16 rounded-2xl flex items-center justify-center;
  background: hsl(var(--accent) / 0.1);
  color: hsl(var(--accent));
  border: 1px solid hsl(var(--accent) / 0.2);
}

.tok-empty-title {
  @apply text-lg font-bold font-display;
}

.tok-empty-subtitle {
  @apply text-sm text-muted-foreground max-w-sm mx-auto;
}

/* How-to card */
.tok-howto {
  @apply max-w-lg mx-auto p-5 rounded-xl;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.tok-howto-title {
  @apply text-sm font-bold mb-4;
}

.tok-howto-steps {
  @apply space-y-3;
}

.tok-howto-steps li {
  @apply flex gap-3 items-start text-sm text-muted-foreground;
}

.tok-howto-num {
  @apply flex-shrink-0 w-5 h-5 rounded-full text-xs font-bold flex items-center justify-center;
  background: hsl(var(--primary) / 0.12);
  color: hsl(var(--primary));
}

/* ---- Section Header ---- */
.tok-section-header {
  @apply flex items-center justify-between;
}

.tok-section-title {
  @apply flex items-center gap-2 text-lg font-bold font-display;
}

.tok-section-title svg {
  color: hsl(var(--accent));
}

.tok-count-badge {
  @apply px-2.5 py-0.5 rounded-full text-xs font-bold;
  background: hsl(var(--accent) / 0.12);
  color: hsl(var(--accent));
}

/* ---- Token List ---- */
.tok-list {
  @apply space-y-3;
}

/* ---- Token Card ---- */
.tok-card {
  @apply relative rounded-xl p-4 space-y-3 transition-all duration-300 overflow-hidden;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.tok-card:hover {
  border-color: hsl(var(--accent) / 0.3);
  box-shadow: 0 8px 30px hsl(var(--accent) / 0.06);
  transform: translateY(-1px);
}

.tok-card::before {
  content: '';
  @apply absolute inset-0 opacity-0 transition-opacity duration-300 pointer-events-none;
  background: linear-gradient(135deg, hsl(var(--accent) / 0.04) 0%, transparent 50%);
}

.tok-card:hover::before {
  opacity: 1;
}

.tok-card-top {
  @apply flex items-center justify-between;
}

.tok-card-meta {
  @apply flex items-center gap-3 min-w-0;
}

.tok-card-icon {
  @apply flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center;
  background: hsl(var(--accent) / 0.12);
  color: hsl(var(--accent));
}

.tok-card-name {
  @apply text-base font-bold truncate;
}

.tok-card-delete {
  @apply p-1.5 rounded-full transition-all duration-200;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted) / 0.5);
  opacity: 0;
}

.tok-card:hover .tok-card-delete {
  opacity: 1;
}

.tok-card-delete:hover {
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 0.12);
  box-shadow: 0 0 12px hsl(var(--destructive) / 0.12);
  transform: translateY(-1px);
}

.tok-card-details {
  @apply flex flex-wrap gap-x-5 gap-y-1.5 pt-3;
  border-top: 1px solid hsl(var(--border) / 0.4);
}

.tok-detail {
  @apply flex items-center gap-1.5 text-xs;
}

.tok-detail svg {
  @apply flex-shrink-0 text-muted-foreground;
}

.tok-detail-label {
  @apply text-muted-foreground;
}

.tok-detail-value {
  @apply font-medium;
}

.tok-detail-dim {
  @apply text-muted-foreground italic;
}

/* ---- Modal ---- */
.tok-modal-overlay {
  @apply fixed inset-0 flex items-center justify-center z-50 p-4;
  background: hsl(0 0% 0% / 0.6);
  backdrop-filter: blur(4px);
}

.tok-modal {
  @apply w-full max-w-md rounded-2xl overflow-hidden;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
  box-shadow: 0 25px 50px -12px hsl(0 0% 0% / 0.4);
}

.tok-modal-lg {
  @apply max-w-lg;
}

.tok-modal-header {
  @apply px-6 pt-6 pb-0;
}

.tok-modal-header h2 {
  @apply text-xl font-bold font-display;
}

.tok-modal-header p {
  @apply text-sm text-muted-foreground mt-1;
}

.tok-modal-body {
  @apply p-6 space-y-5;
}

.tok-modal-actions {
  @apply flex gap-2;
}

.tok-modal-actions-3 {
  @apply flex-wrap;
}

/* ---- Form Elements ---- */
.tok-form-group {
  @apply space-y-1.5;
}

.tok-form-group label {
  @apply block text-sm font-medium;
}

.tok-form-hint {
  @apply text-xs text-muted-foreground;
}

.tok-form-error {
  @apply p-3 rounded-xl text-sm;
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

.tok-number-input {
  @apply flex h-10 w-full rounded-xl px-3 py-2 text-sm transition-all duration-200;
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
  color: hsl(var(--foreground));
}

.tok-number-input:focus {
  outline: none;
  border-color: hsl(var(--primary) / 0.5);
  box-shadow: 0 0 0 3px hsl(var(--primary) / 0.1);
}

/* ---- Advanced Settings ---- */
.tok-advanced summary {
  @apply text-sm font-medium cursor-pointer text-muted-foreground select-none flex items-center gap-1.5 transition-colors;
}

.tok-advanced summary:hover {
  color: hsl(var(--foreground));
}

.tok-advanced-chevron {
  @apply transition-transform;
}

.tok-advanced[open] .tok-advanced-chevron {
  transform: rotate(90deg);
}

.tok-advanced-body {
  @apply space-y-5 mt-4 pl-0.5;
}

/* ---- Token Display ---- */
.tok-token-display {
  @apply text-center;
}

.tok-success-icon {
  @apply mx-auto w-12 h-12 rounded-full flex items-center justify-center mb-4;
  background: hsl(160 84% 45% / 0.12);
  color: hsl(160 84% 45%);
  border: 1px solid hsl(160 84% 45% / 0.2);
}

.tok-token-display-title {
  @apply text-xl font-bold font-display mb-5;
}

.tok-token-warning {
  @apply flex items-start gap-3 p-4 rounded-xl text-left mb-5;
  background: hsl(38 85% 55% / 0.1);
  border: 1px solid hsl(38 85% 55% / 0.2);
  color: hsl(38 85% 55%);
}

.tok-token-warning p {
  @apply text-sm;
}

.tok-token-value {
  @apply relative p-4 rounded-xl font-mono text-sm break-all text-left mb-3 cursor-pointer transition-all duration-200;
  background: hsl(220 20% 6%);
  border: 1px solid hsl(220 15% 15%);
  color: hsl(75 100% 50%);
}

.tok-token-value:hover {
  border-color: hsl(var(--primary) / 0.3);
  box-shadow: 0 0 12px hsl(var(--primary) / 0.1);
}

.tok-token-value-hint {
  @apply absolute top-2 right-3 text-xs opacity-0 transition-opacity;
  color: hsl(var(--muted-foreground));
}

.tok-token-value:hover .tok-token-value-hint {
  opacity: 1;
}

.tok-token-cmd {
  @apply relative p-3 rounded-xl text-left mb-5 cursor-pointer transition-all duration-200;
  background: hsl(var(--muted) / 0.3);
  border: 1px solid hsl(var(--border) / 0.5);
}

.tok-token-cmd:hover {
  border-color: hsl(var(--primary) / 0.3);
}

.tok-token-cmd-label {
  @apply text-xs text-muted-foreground mb-1.5;
}

.tok-token-cmd code {
  @apply text-xs font-mono;
  color: hsl(var(--primary));
}

.tok-token-cmd-hint {
  @apply absolute top-2 right-3 text-xs opacity-0 transition-opacity;
  color: hsl(var(--muted-foreground));
}

.tok-token-cmd:hover .tok-token-cmd-hint {
  opacity: 1;
}

/* ---- Modal Transitions ---- */
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-active .tok-modal,
.modal-leave-active .tok-modal {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .tok-modal,
.modal-leave-to .tok-modal {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}
</style>
