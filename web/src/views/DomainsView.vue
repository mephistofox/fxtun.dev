<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { domainsApi, type Domain } from '@/api/client'

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

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

onMounted(loadDomains)
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

      <div v-if="loading" class="text-center py-8 text-muted-foreground">{{ t('common.loading') }}</div>

      <div v-else-if="domains.length === 0" class="text-center py-8">
        <p class="text-muted-foreground">{{ t('domains.noDomains') }}</p>
        <p class="text-sm text-muted-foreground mt-2">
          {{ t('domains.noDomainsHint') }}
        </p>
      </div>

      <div v-else class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card v-for="domain in domains" :key="domain.id" class="p-4">
          <div class="flex items-start justify-between">
            <div class="space-y-1">
              <h3 class="font-medium">{{ domain.subdomain }}.mfdev.ru</h3>
              <p class="text-sm text-muted-foreground">
                {{ t('domains.reserved') }}: {{ formatDate(domain.created_at) }}
              </p>
            </div>
            <Button variant="ghost" size="icon" @click="releaseDomain(domain.id)" :title="t('domains.releaseDomain')">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6" />
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
              </svg>
            </Button>
          </div>
        </Card>
      </div>
    </div>
  </Layout>
</template>
