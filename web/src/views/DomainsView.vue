<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
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

const MAX_DOMAINS = 3

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
const maxCustomDomains = ref(0)

async function loadDomains() {
  loading.value = true
  error.value = ''
  try {
    const response = await domainsApi.list()
    domains.value = response.data.domains || []
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
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ t('domains.title') }}</h1>
          <p class="text-muted-foreground">
            {{ t('domains.subtitle') }} ({{ domains.length }}/{{ MAX_DOMAINS }})
          </p>
        </div>
        <Button
          @click="showReserveDialog = true"
          :disabled="domains.length >= MAX_DOMAINS"
        >
          {{ t('domains.reserve') }}
        </Button>
      </div>

      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
        {{ error }}
      </div>

      <!-- Reserve Domain Dialog -->
      <div
        v-if="showReserveDialog"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      >
        <Card class="w-full max-w-md p-6">
          <h2 class="text-xl font-bold mb-4">{{ t('domains.reserveTitle') }}</h2>
          <form @submit.prevent="reserveDomain" class="space-y-4">
            <div v-if="reserveError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
              {{ reserveError }}
            </div>

            <div class="space-y-2">
              <label class="text-sm font-medium">{{ t('domains.subdomain') }}</label>
              <div class="flex space-x-2">
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
              <p class="text-xs text-muted-foreground">
                {{ t('domains.willBeAvailable') }} {{ newSubdomain || 'xxx' }}.mfdev.ru
              </p>
              <p v-if="isAvailable === true" class="text-sm text-green-600 dark:text-green-400">
                {{ t('domains.available') }}
              </p>
              <p v-if="isAvailable === false" class="text-sm text-red-600 dark:text-red-400">
                {{ t('domains.notAvailable') }}
              </p>
            </div>

            <div class="flex space-x-2">
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
        </Card>
      </div>

      <div v-if="loading" class="text-center py-12 text-muted-foreground">
        <div class="animate-pulse">{{ t('common.loading') }}</div>
      </div>

      <div v-else-if="domains.length === 0" class="text-center py-12">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-emerald-500/10 mb-4">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="2" y1="12" x2="22" y2="12" />
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
          </svg>
        </div>
        <p class="text-muted-foreground font-medium">{{ t('domains.noDomains') }}</p>
        <p class="text-sm text-muted-foreground mt-2">
          {{ t('domains.noDomainsHint') }}
        </p>
      </div>

      <div v-else class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="domain in domains"
          :key="domain.id"
          class="group relative overflow-hidden rounded-xl border-2 bg-gradient-to-br from-emerald-500/20 to-emerald-500/5 border-emerald-500/20 hover:border-emerald-500/40 transition-all duration-300 hover:shadow-lg hover:scale-[1.02]"
        >
          <!-- Top accent line -->
          <div class="absolute top-0 left-0 right-0 h-1 bg-emerald-500" />

          <div class="p-4 pt-5">
            <div class="flex items-start justify-between">
              <div class="flex items-start gap-3">
                <!-- Icon container -->
                <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-emerald-500/10 text-emerald-500">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10" />
                    <line x1="2" y1="12" x2="22" y2="12" />
                    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
                  </svg>
                </div>
                <div class="space-y-1 min-w-0">
                  <h3 class="font-semibold text-foreground truncate">{{ domain.subdomain }}</h3>
                  <p class="text-xs text-muted-foreground">.mfdev.ru</p>
                </div>
              </div>
              <Button variant="ghost" size="icon" @click="releaseDomain(domain.id)" :title="t('domains.releaseDomain')" class="opacity-0 group-hover:opacity-100 transition-opacity">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6" />
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                </svg>
              </Button>
            </div>

            <div class="mt-4 pt-3 border-t border-emerald-500/20">
              <div class="flex items-center justify-between text-xs">
                <span class="text-muted-foreground">{{ t('domains.reserved') }}</span>
                <span class="font-medium text-emerald-600 dark:text-emerald-400">{{ formatDate(domain.created_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Custom Domains Section -->
      <div class="border-t pt-6">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-2xl font-bold">{{ t('customDomains.title') }}</h2>
            <p class="text-muted-foreground">
              {{ t('customDomains.subtitle') }} ({{ customDomains.length }}/{{ maxCustomDomains }})
            </p>
          </div>
          <Button
            @click="showAddDialog = true"
            :disabled="maxCustomDomains > 0 && customDomains.length >= maxCustomDomains"
          >
            {{ t('customDomains.add') }}
          </Button>
        </div>

        <div v-if="customError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm mt-4">
          {{ customError }}
        </div>

        <!-- Add Custom Domain Dialog -->
        <div
          v-if="showAddDialog"
          class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
        >
          <Card class="w-full max-w-md p-6">
            <h2 class="text-xl font-bold mb-4">{{ t('customDomains.addTitle') }}</h2>
            <form @submit.prevent="addCustomDomain" class="space-y-4">
              <div v-if="addError" class="bg-destructive/10 text-destructive p-3 rounded-md text-sm">
                {{ addError }}
              </div>

              <div class="space-y-2">
                <label class="text-sm font-medium">{{ t('customDomains.domain') }}</label>
                <Input
                  v-model="newDomain"
                  placeholder="example.com"
                  required
                />
              </div>

              <div class="space-y-2">
                <label class="text-sm font-medium">{{ t('customDomains.targetSubdomain') }}</label>
                <select
                  v-model="newTargetSubdomain"
                  class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  required
                >
                  <option value="" disabled>{{ t('customDomains.selectSubdomain') }}</option>
                  <option v-for="d in domains" :key="d.id" :value="d.subdomain">
                    {{ d.subdomain }}
                  </option>
                </select>
              </div>

              <div v-if="newDomain" class="bg-blue-500/10 text-blue-700 dark:text-blue-300 p-3 rounded-md text-sm">
                {{ t('customDomains.cnameHint') }}
                <code class="block mt-1 font-mono text-xs bg-blue-500/10 px-2 py-1 rounded">
                  {{ newDomain }} CNAME {{ newTargetSubdomain || '...' }}.{{ baseDomain }}
                </code>
              </div>

              <div class="flex space-x-2">
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
          </Card>
        </div>

        <div v-if="customLoading" class="text-center py-12 text-muted-foreground mt-4">
          <div class="animate-pulse">{{ t('common.loading') }}</div>
        </div>

        <div v-else-if="customDomains.length === 0" class="text-center py-12 mt-4">
          <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-blue-500/10 mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <line x1="2" y1="12" x2="22" y2="12" />
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
            </svg>
          </div>
          <p class="text-muted-foreground font-medium">{{ t('customDomains.noDomains') }}</p>
          <p class="text-sm text-muted-foreground mt-2">
            {{ t('customDomains.noDomainsHint') }}
          </p>
        </div>

        <div v-else class="grid gap-4 md:grid-cols-2 lg:grid-cols-3 mt-4">
          <div
            v-for="cd in customDomains"
            :key="cd.id"
            class="group relative overflow-hidden rounded-xl border-2 bg-gradient-to-br from-blue-500/20 to-blue-500/5 border-blue-500/20 hover:border-blue-500/40 transition-all duration-300 hover:shadow-lg hover:scale-[1.02]"
          >
            <!-- Top accent line -->
            <div class="absolute top-0 left-0 right-0 h-1 bg-blue-500" />

            <div class="p-4 pt-5">
              <div class="flex items-start justify-between">
                <div class="flex items-start gap-3">
                  <!-- Icon container -->
                  <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-blue-500/10 text-blue-500">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="2" y1="12" x2="22" y2="12" />
                      <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
                    </svg>
                  </div>
                  <div class="space-y-1 min-w-0">
                    <h3 class="font-semibold text-foreground truncate">{{ cd.domain }}</h3>
                    <p class="text-xs text-muted-foreground">→ {{ cd.target_subdomain }}.{{ baseDomain }}</p>
                  </div>
                </div>
                <div class="flex items-center gap-1">
                  <Button
                    v-if="!cd.verified"
                    variant="ghost"
                    size="icon"
                    @click="verifyCustomDomain(cd.id)"
                    :title="t('customDomains.verify')"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                      <polyline points="22 4 12 14.01 9 11.01" />
                    </svg>
                  </Button>
                  <Button variant="ghost" size="icon" @click="deleteCustomDomain(cd.id)" :title="t('customDomains.deleteDomain')" class="opacity-0 group-hover:opacity-100 transition-opacity">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="3 6 5 6 21 6" />
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                    </svg>
                  </Button>
                </div>
              </div>

              <div class="mt-4 pt-3 border-t border-blue-500/20">
                <div class="flex items-center justify-between text-xs">
                  <span>
                    <span
                      v-if="cd.verified"
                      class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-500/10 text-green-600 dark:text-green-400"
                    >
                      {{ t('customDomains.verified') }}
                    </span>
                    <span
                      v-else
                      class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-500/10 text-yellow-600 dark:text-yellow-400"
                    >
                      {{ t('customDomains.pending') }}
                    </span>
                  </span>
                  <span class="font-medium text-blue-600 dark:text-blue-400">{{ formatDate(cd.created_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Layout>
</template>
