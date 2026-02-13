<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { domainsApi, customDomainsApi, type Domain, type CustomDomain } from '@/api/client'

const { t, locale } = useI18n()

const domains = ref<Domain[]>([])
const loading = ref(true)
const error = ref('')
const showReserveDialog = ref(false)

const newSubdomain = ref('')
const reserving = ref(false)
const reserveError = ref('')
const checkingAvailability = ref(false)
const isAvailable = ref<boolean | null>(null)

const maxDomains = ref(1)

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

const filteredDomains = computed(() => {
  if (!debouncedSearch.value) return domains.value
  return domains.value.filter(d =>
    d.subdomain.toLowerCase().includes(debouncedSearch.value)
  )
})

const filteredCustomDomains = computed(() => {
  if (!debouncedSearch.value) return customDomains.value
  const q = debouncedSearch.value
  return customDomains.value.filter(cd =>
    cd.domain.toLowerCase().includes(q) ||
    cd.target_subdomain.toLowerCase().includes(q)
  )
})

// Custom domains state
const customDomains = ref<CustomDomain[]>([])
const customLoading = ref(true)
const customError = ref('')
const showAddDialog = ref(false)
const newDomain = ref('')
const newTargetSubdomain = ref('')
const adding = ref(false)
const addError = ref('')
const baseDomain = ref('')
const serverIP = ref('')
const maxCustomDomains = ref(0)

const availableSubdomains = computed(() => {
  const usedTargets = new Set(customDomains.value.map(cd => cd.target_subdomain))
  return domains.value.filter(d => !usedTargets.has(d.subdomain))
})

async function loadDomains() {
  loading.value = true
  error.value = ''
  try {
    const response = await domainsApi.list()
    domains.value = response.data.domains || []
    if (response.data.max_domains !== undefined) {
      maxDomains.value = response.data.max_domains
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('domains.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function checkAvailability() {
  if (!newSubdomain.value) return

  checkingAvailability.value = true
  isAvailable.value = null
  try {
    const response = await domainsApi.check(newSubdomain.value)
    isAvailable.value = response.data.available
  } catch {
    isAvailable.value = false
  } finally {
    checkingAvailability.value = false
  }
}

async function reserveDomain() {
  reserving.value = true
  reserveError.value = ''
  try {
    const response = await domainsApi.reserve(newSubdomain.value)
    domains.value.push(response.data)
    newSubdomain.value = ''
    isAvailable.value = null
    showReserveDialog.value = false
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    reserveError.value = err.response?.data?.error || t('domains.failedToReserve')
  } finally {
    reserving.value = false
  }
}

async function releaseDomain(id: number) {
  if (!confirm(t('domains.confirmRelease'))) return

  try {
    await domainsApi.release(id)
    domains.value = domains.value.filter((d) => d.id !== id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('domains.failedToRelease')
  }
}

async function loadCustomDomains() {
  customLoading.value = true
  customError.value = ''
  try {
    const response = await customDomainsApi.list()
    customDomains.value = response.data.domains || []
    baseDomain.value = response.data.base_domain || ''
    serverIP.value = response.data.server_ip || ''
    maxCustomDomains.value = response.data.max_domains || 0
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    customError.value = err.response?.data?.error || t('customDomains.failedToLoad')
  } finally {
    customLoading.value = false
  }
}

async function addCustomDomain() {
  adding.value = true
  addError.value = ''
  try {
    const response = await customDomainsApi.add(newDomain.value, newTargetSubdomain.value)
    customDomains.value.push(response.data)
    newDomain.value = ''
    newTargetSubdomain.value = ''
    showAddDialog.value = false
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    addError.value = err.response?.data?.error || t('customDomains.failedToAdd')
  } finally {
    adding.value = false
  }
}

async function deleteCustomDomain(id: number) {
  if (!confirm(t('customDomains.confirmDelete'))) return

  try {
    await customDomainsApi.delete(id)
    customDomains.value = customDomains.value.filter((d) => d.id !== id)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    customError.value = err.response?.data?.error || t('customDomains.failedToDelete')
  }
}

async function verifyCustomDomain(id: number) {
  try {
    const response = await customDomainsApi.verify(id)
    if (response.data.verified) {
      const domain = customDomains.value.find((d) => d.id === id)
      if (domain) {
        domain.verified = true
        domain.verified_at = new Date().toISOString()
      }
    } else {
      customError.value = response.data.error || t('customDomains.verificationFailed')
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    customError.value = err.response?.data?.error || t('customDomains.failedToVerify')
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

onMounted(() => {
  loadDomains()
  loadCustomDomains()
})
</script>

<template>
  <Layout>
    <div class="domains-root">
      <!-- ========== HERO HEADER ========== -->
      <div class="dom-hero">
        <div class="dom-hero-content">
          <div class="dom-hero-left">
            <div class="dom-status-badge">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              <span>{{ domains.length }}/{{ maxDomains < 0 ? '∞' : maxDomains }}</span>
            </div>
            <h1 class="dom-title">{{ t('domains.title') }}</h1>
            <p class="dom-subtitle">{{ t('domains.subtitle') }}</p>
          </div>
          <div class="dom-hero-right">
            <div class="dom-search-wrapper">
              <svg xmlns="http://www.w3.org/2000/svg" class="dom-search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('domains.searchPlaceholder')"
                class="dom-search-input"
              />
              <button v-if="searchQuery" @click="searchQuery = ''" class="dom-search-clear">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
            <Button
              @click="showReserveDialog = true"
              :disabled="maxDomains >= 0 && domains.length >= maxDomains"
              class="dom-reserve-btn"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              {{ t('domains.reserve') }}
            </Button>
          </div>
        </div>
        <div class="dom-hero-orb dom-hero-orb-1"></div>
        <div class="dom-hero-orb dom-hero-orb-2"></div>
      </div>

      <!-- ========== ERROR ========== -->
      <div v-if="error" class="dom-error">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
        {{ error }}
      </div>

      <!-- ========== RESERVE DIALOG ========== -->
      <Teleport to="body">
        <Transition name="modal">
          <div v-if="showReserveDialog" class="dom-modal-overlay" @click.self="showReserveDialog = false">
            <div class="dom-modal">
              <div class="dom-modal-header">
                <h2>{{ t('domains.reserveTitle') }}</h2>
              </div>
              <form @submit.prevent="reserveDomain" class="dom-modal-body">
                <div v-if="reserveError" class="dom-form-error">{{ reserveError }}</div>

                <div class="dom-form-group">
                  <label>{{ t('domains.subdomain') }}</label>
                  <div class="dom-input-row">
                    <Input
                      v-model="newSubdomain"
                      placeholder="my-app"
                      @input="isAvailable = null"
                      required
                    />
                    <Button type="button" variant="outline" @click="checkAvailability" :loading="checkingAvailability">
                      {{ t('common.check') }}
                    </Button>
                  </div>
                  <div class="dom-domain-preview">
                    <span class="dom-domain-preview-label">{{ newSubdomain || 'xxx' }}</span><span class="dom-domain-preview-base">.{{ baseDomain || 'fxtun.dev' }}</span>
                  </div>
                  <p v-if="isAvailable === true" class="dom-avail dom-avail-yes">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
                    {{ t('domains.available') }}
                  </p>
                  <p v-if="isAvailable === false" class="dom-avail dom-avail-no">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                    {{ t('domains.notAvailable') }}
                  </p>
                </div>

                <div class="dom-modal-actions">
                  <Button type="button" variant="outline" @click="showReserveDialog = false" class="flex-1">
                    {{ t('common.cancel') }}
                  </Button>
                  <Button
                    type="submit"
                    :loading="reserving"
                    :disabled="!newSubdomain || isAvailable === false"
                    class="flex-1"
                  >
                    {{ t('domains.reserve') }}
                  </Button>
                </div>
              </form>
            </div>
          </div>
        </Transition>
      </Teleport>

      <!-- ========== LOADING ========== -->
      <div v-if="loading" class="dom-loading">
        <div class="dom-loading-spinner"></div>
        <span>{{ t('common.loading') }}</span>
      </div>

      <!-- ========== EMPTY STATE ========== -->
      <div v-else-if="domains.length === 0" class="dom-empty">
        <div class="dom-empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
        </div>
        <p class="dom-empty-title">{{ t('domains.noDomains') }}</p>
        <p class="dom-empty-subtitle">{{ t('domains.noDomainsHint') }}</p>
      </div>

      <!-- ========== SUBDOMAIN CARDS ========== -->
      <div v-else>
        <div class="dom-section-header">
          <h2 class="dom-section-title">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
            {{ t('domains.title') }}
          </h2>
          <span class="dom-count-badge">{{ filteredDomains.length }}<template v-if="debouncedSearch">/{{ domains.length }}</template></span>
        </div>

        <div class="dom-grid">
          <div
            v-for="domain in filteredDomains"
            :key="domain.id"
            class="dom-card dom-card-subdomain"
          >
            <div class="dom-card-top">
              <div class="dom-card-icon">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              </div>
              <button
                @click="releaseDomain(domain.id)"
                class="dom-card-delete"
                :title="t('domains.releaseDomain')"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>

            <div class="dom-card-body">
              <h3 class="dom-card-name">{{ domain.subdomain }}</h3>
              <p class="dom-card-base">.{{ baseDomain || 'fxtun.dev' }}</p>
            </div>

            <div class="dom-card-footer">
              <span class="dom-card-date-label">{{ t('domains.reserved') }}</span>
              <span class="dom-card-date">{{ formatDate(domain.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== CUSTOM DOMAINS SECTION ========== -->
      <div class="dom-custom-section">
        <div class="dom-section-header">
          <div>
            <h2 class="dom-section-title">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
              {{ t('customDomains.title') }}
            </h2>
            <p class="dom-section-subtitle">{{ t('customDomains.subtitle') }} ({{ customDomains.length }}/{{ maxCustomDomains < 0 ? '∞' : maxCustomDomains }})</p>
          </div>
          <Button
            @click="showAddDialog = true"
            :disabled="maxCustomDomains >= 0 && customDomains.length >= maxCustomDomains"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            {{ t('customDomains.add') }}
          </Button>
        </div>

        <div v-if="customError" class="dom-error">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
          {{ customError }}
        </div>

        <!-- ADD CUSTOM DOMAIN DIALOG -->
        <Teleport to="body">
          <Transition name="modal">
            <div v-if="showAddDialog" class="dom-modal-overlay" @click.self="showAddDialog = false">
              <div class="dom-modal dom-modal-lg">
                <div class="dom-modal-header">
                  <h2>{{ t('customDomains.addTitle') }}</h2>
                </div>
                <form @submit.prevent="addCustomDomain" class="dom-modal-body">
                  <div v-if="addError" class="dom-form-error">{{ addError }}</div>

                  <div class="dom-form-group">
                    <label>{{ t('customDomains.domain') }}</label>
                    <Input v-model="newDomain" placeholder="example.com" required />
                  </div>

                  <div class="dom-form-group">
                    <label>{{ t('customDomains.targetSubdomain') }}</label>
                    <select
                      v-model="newTargetSubdomain"
                      class="dom-select"
                      required
                    >
                      <option value="" disabled>{{ t('customDomains.selectSubdomain') }}</option>
                      <option v-for="d in domains" :key="d.id" :value="d.subdomain">
                        {{ d.subdomain }}
                      </option>
                    </select>
                  </div>

                  <!-- Reserve subdomain first -->
                  <div v-if="availableSubdomains.length === 0" class="dom-reserve-hint">
                    <p class="dom-reserve-hint-title">{{ t('customDomains.reserveFirst') }}</p>
                    <p class="dom-reserve-hint-text">{{ t('customDomains.reserveFirstHint') }}</p>
                    <Button
                      type="button"
                      size="sm"
                      class="mt-2.5"
                      @click="showAddDialog = false; showReserveDialog = true"
                    >
                      {{ t('customDomains.reserveNow') }}
                    </Button>
                  </div>

                  <!-- DNS Records Section -->
                  <div class="dom-dns-section">
                    <div class="dom-dns-header">
                      <div class="dom-dns-header-icon">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                      </div>
                      <p>{{ t('customDomains.cnameHint') }}</p>
                    </div>

                    <!-- CNAME record -->
                    <div class="dom-dns-record">
                      <div class="dom-dns-record-top">
                        <span class="dom-dns-badge dom-dns-badge-cname">CNAME</span>
                        <span class="dom-dns-record-hint">{{ locale === 'ru' ? '-- для поддоменов (app.example.com)' : '-- for subdomains (app.example.com)' }}</span>
                      </div>
                      <div class="dom-dns-terminal">
                        <span class="dom-dns-val">{{ newDomain || 'app.example.com' }}</span>
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 dom-dns-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
                        <span class="dom-dns-target">{{ newTargetSubdomain || 'my-app' }}.{{ baseDomain || 'fxtun.dev' }}</span>
                      </div>
                    </div>

                    <!-- A record -->
                    <div class="dom-dns-record">
                      <div class="dom-dns-record-top">
                        <span class="dom-dns-badge dom-dns-badge-a">A</span>
                        <span class="dom-dns-record-hint">{{ locale === 'ru' ? '-- для корневых доменов (example.com)' : '-- for root domains (example.com)' }}</span>
                      </div>
                      <div class="dom-dns-terminal">
                        <span class="dom-dns-val">{{ newDomain || 'example.com' }}</span>
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 dom-dns-arrow-tcp" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
                        <span class="dom-dns-ip">{{ serverIP || '...' }}</span>
                      </div>
                    </div>

                    <!-- Steps hint -->
                    <div class="dom-dns-steps">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
                      <div class="whitespace-pre-line">{{ t('customDomains.dnsGuideSteps') }}</div>
                    </div>
                  </div>

                  <div class="dom-modal-actions">
                    <Button type="button" variant="outline" @click="showAddDialog = false" class="flex-1">
                      {{ t('common.cancel') }}
                    </Button>
                    <Button
                      type="submit"
                      :loading="adding"
                      :disabled="!newDomain || !newTargetSubdomain"
                      class="flex-1"
                    >
                      {{ t('customDomains.add') }}
                    </Button>
                  </div>
                </form>
              </div>
            </div>
          </Transition>
        </Teleport>

        <!-- Custom Domains Loading -->
        <div v-if="customLoading" class="dom-loading">
          <div class="dom-loading-spinner"></div>
          <span>{{ t('common.loading') }}</span>
        </div>

        <!-- Custom Domains Empty -->
        <div v-else-if="customDomains.length === 0" class="dom-empty dom-empty-sm">
          <div class="dom-empty-icon dom-empty-icon-custom">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
          </div>
          <p class="dom-empty-title">{{ t('customDomains.noDomains') }}</p>
          <p class="dom-empty-subtitle">{{ t('customDomains.noDomainsHint') }}</p>
        </div>

        <!-- Custom Domain Cards -->
        <div v-else class="dom-grid">
          <div
            v-for="cd in filteredCustomDomains"
            :key="cd.id"
            class="dom-card dom-card-custom"
          >
            <div class="dom-card-top">
              <div class="dom-card-icon dom-card-icon-custom">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
              </div>
              <div class="dom-card-actions">
                <button
                  v-if="!cd.verified"
                  @click="verifyCustomDomain(cd.id)"
                  class="dom-card-verify"
                  :title="t('customDomains.verify')"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                  <span>{{ t('customDomains.verify') }}</span>
                </button>
                <button
                  @click="deleteCustomDomain(cd.id)"
                  class="dom-card-delete"
                  :title="t('customDomains.deleteDomain')"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                </button>
              </div>
            </div>

            <div class="dom-card-body">
              <h3 class="dom-card-name">{{ cd.domain }}</h3>
              <p class="dom-card-target">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
                {{ cd.target_subdomain }}.{{ baseDomain }}
              </p>
            </div>

            <div class="dom-card-footer">
              <span
                :class="['dom-status-pill', cd.verified ? 'dom-status-verified' : 'dom-status-pending']"
              >
                {{ cd.verified ? t('customDomains.verified') : t('customDomains.pending') }}
              </span>
              <span class="dom-card-date">{{ formatDate(cd.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Layout>
</template>

<style scoped>
/* ============================================
   DOMAINS — CYBER COMMAND CENTER
   ============================================ */

.domains-root {
  @apply space-y-6;
}

/* ---- Hero ---- */
.dom-hero {
  @apply relative rounded-2xl overflow-hidden p-6 sm:p-8;
  background:
    radial-gradient(ellipse 60% 80% at 20% 0%, hsl(var(--type-http) / 0.12) 0%, transparent 60%),
    radial-gradient(ellipse 40% 60% at 90% 80%, hsl(var(--accent) / 0.08) 0%, transparent 50%),
    hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dom-hero-content {
  @apply relative z-10 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4;
}

.dom-hero-left {
  @apply space-y-2;
}

.dom-hero-right {
  @apply flex-shrink-0 flex items-center gap-3;
}

/* ---- Search ---- */
.dom-search-wrapper {
  @apply relative hidden sm:flex items-center;
}

.dom-search-icon {
  @apply absolute left-3 w-4 h-4 pointer-events-none;
  color: hsl(var(--muted-foreground));
}

.dom-search-input {
  @apply h-10 pl-9 pr-8 rounded-xl text-sm transition-all duration-200 w-48 lg:w-56;
  background: hsl(var(--background) / 0.6);
  border: 1px solid hsl(var(--border) / 0.6);
  color: hsl(var(--foreground));
  backdrop-filter: blur(8px);
}

.dom-search-input::placeholder {
  color: hsl(var(--muted-foreground) / 0.7);
}

.dom-search-input:focus {
  outline: none;
  border-color: hsl(var(--type-http) / 0.5);
  box-shadow: 0 0 0 3px hsl(var(--type-http) / 0.08), 0 0 16px hsl(var(--type-http) / 0.06);
  width: 16rem;
}

.dom-search-clear {
  @apply absolute right-2 p-1 rounded-full transition-colors;
  color: hsl(var(--muted-foreground));
}

.dom-search-clear:hover {
  color: hsl(var(--foreground));
  background: hsl(var(--muted) / 0.5);
}

.dom-status-badge {
  @apply inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium;
  background: hsl(var(--type-http) / 0.1);
  border: 1px solid hsl(var(--type-http) / 0.2);
  color: hsl(var(--type-http));
}

.dom-title {
  @apply text-2xl sm:text-3xl font-bold tracking-tight font-display;
}

.dom-subtitle {
  @apply text-sm text-muted-foreground;
}

.dom-reserve-btn {
  box-shadow: 0 0 20px hsl(var(--primary) / 0.2);
}

.dom-hero-orb {
  @apply absolute rounded-full pointer-events-none;
  filter: blur(80px);
}

.dom-hero-orb-1 {
  width: 200px;
  height: 200px;
  top: -60px;
  left: -40px;
  background: hsl(var(--type-http) / 0.15);
}

.dom-hero-orb-2 {
  width: 150px;
  height: 150px;
  bottom: -50px;
  right: -30px;
  background: hsl(var(--accent) / 0.1);
}

/* ---- Error ---- */
.dom-error {
  @apply flex items-center gap-2 p-4 rounded-xl text-sm;
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

/* ---- Loading ---- */
.dom-loading {
  @apply flex items-center justify-center gap-3 py-16 text-muted-foreground;
}

.dom-loading-spinner {
  @apply w-5 h-5 rounded-full border-2 border-current border-t-transparent animate-spin;
}

/* ---- Empty State ---- */
.dom-empty {
  @apply text-center py-12 space-y-3;
}

.dom-empty-sm {
  @apply py-8;
}

.dom-empty-icon {
  @apply mx-auto w-16 h-16 rounded-2xl flex items-center justify-center;
  background: hsl(var(--type-http) / 0.1);
  color: hsl(var(--type-http));
  border: 1px solid hsl(var(--type-http) / 0.2);
}

.dom-empty-icon-custom {
  background: hsl(var(--type-tcp) / 0.1);
  color: hsl(var(--type-tcp));
  border: 1px solid hsl(var(--type-tcp) / 0.2);
}

.dom-empty-title {
  @apply text-base font-semibold;
}

.dom-empty-subtitle {
  @apply text-sm text-muted-foreground max-w-sm mx-auto;
}

/* ---- Section Header ---- */
.dom-section-header {
  @apply flex items-center justify-between mb-4;
}

.dom-section-title {
  @apply flex items-center gap-2 text-lg font-bold font-display;
  color: hsl(var(--foreground));
}

.dom-section-title svg {
  color: hsl(var(--primary));
}

.dom-section-subtitle {
  @apply text-sm text-muted-foreground mt-0.5;
}

.dom-count-badge {
  @apply px-2.5 py-0.5 rounded-full text-xs font-bold;
  background: hsl(var(--type-http) / 0.12);
  color: hsl(var(--type-http));
}

/* ---- Domain Grid ---- */
.dom-grid {
  @apply grid gap-4 md:grid-cols-2 lg:grid-cols-3;
}

/* ---- Domain Card ---- */
.dom-card {
  @apply relative rounded-xl p-4 space-y-3 transition-all duration-300 overflow-hidden;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dom-card:hover {
  transform: translateY(-2px);
}

.dom-card::before {
  content: '';
  @apply absolute inset-0 opacity-0 transition-opacity duration-300 pointer-events-none;
}

.dom-card:hover::before {
  opacity: 1;
}

/* Subdomain card — emerald/HTTP glow */
.dom-card-subdomain {
  border-color: hsl(var(--type-http) / 0.15);
}

.dom-card-subdomain:hover {
  border-color: hsl(var(--type-http) / 0.4);
  box-shadow: 0 8px 30px hsl(var(--type-http) / 0.1);
}

.dom-card-subdomain::before {
  background: linear-gradient(135deg, hsl(var(--type-http) / 0.05) 0%, transparent 50%);
}

/* Custom domain card — TCP/blue glow */
.dom-card-custom {
  border-color: hsl(var(--type-tcp) / 0.15);
}

.dom-card-custom:hover {
  border-color: hsl(var(--type-tcp) / 0.4);
  box-shadow: 0 8px 30px hsl(var(--type-tcp) / 0.1);
}

.dom-card-custom::before {
  background: linear-gradient(135deg, hsl(var(--type-tcp) / 0.05) 0%, transparent 50%);
}

.dom-card-top {
  @apply flex items-center justify-between;
}

.dom-card-icon {
  @apply w-10 h-10 rounded-xl flex items-center justify-center;
  background: hsl(var(--type-http) / 0.12);
  color: hsl(var(--type-http));
}

.dom-card-icon-custom {
  background: hsl(var(--type-tcp) / 0.12);
  color: hsl(var(--type-tcp));
}

.dom-card-actions {
  @apply flex items-center gap-1.5;
}

.dom-card-delete {
  @apply p-1.5 rounded-full transition-all duration-200;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted) / 0.5);
  opacity: 0;
}

.dom-card:hover .dom-card-delete {
  opacity: 1;
}

.dom-card-delete:hover {
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 0.12);
  box-shadow: 0 0 12px hsl(var(--destructive) / 0.12);
  transform: translateY(-1px);
}

.dom-card-verify {
  @apply inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold uppercase tracking-wider transition-all duration-200;
  color: hsl(var(--type-tcp));
  background: hsl(var(--type-tcp) / 0.08);
  border: 1px solid hsl(var(--type-tcp) / 0.15);
}

.dom-card-verify:hover {
  background: hsl(var(--type-tcp) / 0.15);
  border-color: hsl(var(--type-tcp) / 0.3);
  box-shadow: 0 0 12px hsl(var(--type-tcp) / 0.15);
  transform: translateY(-1px);
}

.dom-card-body {
  @apply space-y-0.5;
}

.dom-card-name {
  @apply text-lg font-bold font-display truncate;
}

.dom-card-base {
  @apply text-xs text-muted-foreground font-mono;
}

.dom-card-target {
  @apply flex items-center gap-1 text-xs text-muted-foreground font-mono;
}

.dom-card-footer {
  @apply flex items-center justify-between text-xs pt-3;
  border-top: 1px solid hsl(var(--border) / 0.4);
}

.dom-card-date-label {
  @apply text-muted-foreground;
}

.dom-card-date {
  @apply font-medium;
  color: hsl(var(--type-http));
}

.dom-card-custom .dom-card-date {
  color: hsl(var(--type-tcp));
}

/* Status pills */
.dom-status-pill {
  @apply inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase tracking-wider;
}

.dom-status-verified {
  background: hsl(160 84% 45% / 0.12);
  color: hsl(160 84% 45%);
  border: 1px solid hsl(160 84% 45% / 0.2);
}

.dom-status-pending {
  background: hsl(38 85% 55% / 0.12);
  color: hsl(38 85% 55%);
  border: 1px solid hsl(38 85% 55% / 0.2);
}

/* ---- Custom Domains Section ---- */
.dom-custom-section {
  @apply pt-6;
  border-top: 1px solid hsl(var(--border) / 0.4);
}

/* ---- Modal ---- */
.dom-modal-overlay {
  @apply fixed inset-0 flex items-center justify-center z-50 p-4;
  background: hsl(0 0% 0% / 0.6);
  backdrop-filter: blur(4px);
}

.dom-modal {
  @apply w-full max-w-md rounded-2xl overflow-hidden;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
  box-shadow: 0 25px 50px -12px hsl(0 0% 0% / 0.4);
}

.dom-modal-lg {
  @apply max-w-lg;
}

.dom-modal-header {
  @apply px-6 pt-6 pb-0;
}

.dom-modal-header h2 {
  @apply text-xl font-bold font-display;
}

.dom-modal-body {
  @apply p-6 space-y-5;
}

.dom-modal-actions {
  @apply flex gap-2;
}

/* ---- Form Elements ---- */
.dom-form-group {
  @apply space-y-2;
}

.dom-form-group label {
  @apply block text-sm font-medium;
}

.dom-input-row {
  @apply flex gap-2;
}

.dom-form-error {
  @apply p-3 rounded-xl text-sm;
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

.dom-domain-preview {
  @apply font-mono text-sm;
}

.dom-domain-preview-label {
  color: hsl(var(--primary));
  font-weight: 600;
}

.dom-domain-preview-base {
  @apply text-muted-foreground;
}

.dom-avail {
  @apply flex items-center gap-1.5 text-sm font-medium;
}

.dom-avail-yes {
  color: hsl(160 84% 45%);
}

.dom-avail-no {
  color: hsl(var(--destructive));
}

.dom-select {
  @apply flex h-10 w-full rounded-xl px-3 py-2 text-sm transition-all duration-200;
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
  color: hsl(var(--foreground));
}

.dom-select:focus {
  outline: none;
  border-color: hsl(var(--primary) / 0.5);
  box-shadow: 0 0 0 3px hsl(var(--primary) / 0.1);
}

.dom-reserve-hint {
  @apply p-4 rounded-xl;
  background: hsl(var(--primary) / 0.05);
  border: 1px solid hsl(var(--primary) / 0.15);
}

.dom-reserve-hint-title {
  @apply text-sm font-semibold;
}

.dom-reserve-hint-text {
  @apply text-xs text-muted-foreground mt-0.5;
}

/* ---- DNS Section ---- */
.dom-dns-section {
  @apply space-y-3;
}

.dom-dns-header {
  @apply flex items-center gap-2;
}

.dom-dns-header-icon {
  @apply w-5 h-5 rounded-full flex items-center justify-center flex-shrink-0;
  background: hsl(var(--primary) / 0.15);
  color: hsl(var(--primary));
}

.dom-dns-header p {
  @apply text-sm font-semibold;
}

.dom-dns-record {
  @apply rounded-xl p-3 transition-all duration-200;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border) / 0.6);
}

.dom-dns-record:hover {
  border-color: hsl(var(--primary) / 0.3);
}

.dom-dns-record-top {
  @apply flex items-center gap-2 mb-3;
}

.dom-dns-badge {
  @apply inline-flex items-center px-2.5 py-1 rounded-md text-xs font-bold uppercase tracking-wider;
}

.dom-dns-badge-cname {
  background: hsl(var(--primary) / 0.15);
  color: hsl(var(--primary));
}

.dom-dns-badge-a {
  background: hsl(var(--type-tcp) / 0.15);
  color: hsl(var(--type-tcp));
}

.dom-dns-record-hint {
  @apply text-xs text-muted-foreground;
}

.dom-dns-terminal {
  @apply flex items-center gap-2 px-3 py-2 rounded-lg font-mono text-sm;
  background: hsl(220 20% 6%);
}

.dom-dns-val {
  @apply font-medium;
  color: hsl(210 20% 85%);
}

.dom-dns-arrow {
  @apply flex-shrink-0;
  color: hsl(var(--primary));
}

.dom-dns-arrow-tcp {
  @apply flex-shrink-0;
  color: hsl(var(--type-tcp));
}

.dom-dns-target {
  @apply font-medium;
  color: hsl(var(--primary));
}

.dom-dns-ip {
  @apply font-medium;
  color: hsl(var(--type-tcp));
}

.dom-dns-steps {
  @apply flex items-start gap-3 text-[11px] text-muted-foreground p-3 rounded-lg leading-relaxed;
  background: hsl(var(--muted) / 0.3);
}

/* ---- Modal Transitions ---- */
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-active .dom-modal,
.modal-leave-active .dom-modal {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .dom-modal,
.modal-leave-to .dom-modal {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}
</style>
