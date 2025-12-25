<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDomainsStore } from '@/stores/domains'
import { toast } from '@/composables/useToast'
import {
  Button, Input, Label, Tooltip,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter
} from '@/components/ui'
import { Globe, Plus, Trash2, Copy, Check, RefreshCw, ExternalLink } from 'lucide-vue-next'

const { t, locale } = useI18n()
const domainsStore = useDomainsStore()

const showReserveDialog = ref(false)
const newSubdomain = ref('')
const isReserving = ref(false)
const isCheckingAvailability = ref(false)
const availabilityResult = ref<{ available: boolean; reason?: string } | null>(null)
const copiedId = ref<number | null>(null)

const canReserve = computed(() =>
  domainsStore.domains.length < domainsStore.maxDomains
)

onMounted(async () => {
  await domainsStore.loadDomains()
})

async function checkAvailability() {
  if (!newSubdomain.value) return

  isCheckingAvailability.value = true
  availabilityResult.value = null

  const result = await domainsStore.checkAvailability(newSubdomain.value)
  availabilityResult.value = result
  isCheckingAvailability.value = false
}

async function reserveDomain() {
  if (!newSubdomain.value) return

  isReserving.value = true
  const domain = await domainsStore.reserveDomain(newSubdomain.value)
  isReserving.value = false

  if (domain) {
    toast({ title: t('toasts.domainReserved'), variant: 'success' })
    showReserveDialog.value = false
    newSubdomain.value = ''
    availabilityResult.value = null
  }
}

async function releaseDomain(id: number) {
  const success = await domainsStore.releaseDomain(id)
  if (success) {
    toast({ title: t('toasts.domainReleased'), variant: 'success' })
  }
}

function copyUrl(url: string, id: number) {
  navigator.clipboard.writeText(url)
  copiedId.value = id
  toast({ title: t('toasts.urlCopied'), variant: 'success' })
  setTimeout(() => {
    copiedId.value = null
  }, 2000)
}

function openUrl(url: string) {
  window.open(url, '_blank')
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function openReserveDialog() {
  newSubdomain.value = ''
  availabilityResult.value = null
  showReserveDialog.value = true
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold flex items-center gap-2">
          <Globe class="h-6 w-6 text-emerald-500" />
          {{ t('domains.title') }}
        </h1>
        <p class="text-muted-foreground mt-1">
          {{ t('domains.subtitle') }} ({{ domainsStore.domains.length }}/{{ domainsStore.maxDomains }})
        </p>
      </div>
      <div class="flex items-center gap-2">
        <Tooltip :content="t('common.refresh')">
          <Button variant="outline" size="sm" @click="domainsStore.loadDomains">
            <RefreshCw class="h-4 w-4" />
          </Button>
        </Tooltip>
        <Button @click="openReserveDialog" :disabled="!canReserve">
          <Plus class="h-4 w-4 mr-2" />
          {{ t('domains.reserve') }}
        </Button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="domainsStore.error" class="bg-destructive/10 text-destructive p-4 rounded-lg text-sm">
      {{ domainsStore.error }}
    </div>

    <!-- Loading -->
    <div v-if="domainsStore.isLoading" class="text-center py-12 text-muted-foreground">
      <div class="animate-pulse">{{ t('common.loading') }}</div>
    </div>

    <!-- Empty state -->
    <div v-else-if="domainsStore.domains.length === 0" class="rounded-lg border border-dashed p-12 text-center">
      <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/10">
        <Globe class="h-8 w-8 text-emerald-500" />
      </div>
      <p class="font-semibold text-lg text-muted-foreground">{{ t('domains.noDomains') }}</p>
      <p class="mt-2 text-sm text-muted-foreground">
        {{ t('domains.noDomainsHint') }}
      </p>
      <Button class="mt-4" @click="openReserveDialog">
        <Plus class="h-4 w-4 mr-2" />
        {{ t('domains.reserve') }}
      </Button>
    </div>

    <!-- Domains grid -->
    <TransitionGroup v-else name="list" tag="div" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="domain in domainsStore.domains"
        :key="domain.id"
        class="group relative overflow-hidden rounded-xl border-2 bg-gradient-to-br from-emerald-500/20 to-emerald-500/5 border-emerald-500/20 hover:border-emerald-500/40 transition-all duration-300 hover:shadow-lg hover:scale-[1.02]"
      >
        <!-- Top accent line -->
        <div class="absolute top-0 left-0 right-0 h-1 bg-emerald-500" />

        <div class="p-5 pt-6">
          <!-- Header -->
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/20 transition-transform group-hover:scale-110">
                <Globe class="h-5 w-5 text-emerald-500" />
              </div>
              <div>
                <h3 class="font-semibold text-foreground">{{ domain.subdomain }}</h3>
                <p class="text-xs text-muted-foreground">.mfdev.ru</p>
              </div>
            </div>
            <Tooltip :content="t('domains.releaseDomain')">
              <Button
                variant="ghost"
                size="icon"
                class="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                @click="releaseDomain(domain.id)"
              >
                <Trash2 class="h-4 w-4" />
              </Button>
            </Tooltip>
          </div>

          <!-- URL -->
          <div class="flex items-center gap-2 p-3 rounded-lg bg-background/80 border border-border/50">
            <code class="flex-1 truncate text-xs font-medium">
              {{ domain.url }}
            </code>
            <div class="flex items-center gap-1">
              <Tooltip :content="copiedId === domain.id ? t('common.copied') : t('domains.copyUrl')">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7"
                  @click="copyUrl(domain.url, domain.id)"
                >
                  <component
                    :is="copiedId === domain.id ? Check : Copy"
                    :class="['h-3.5 w-3.5', copiedId === domain.id ? 'text-emerald-500' : '']"
                  />
                </Button>
              </Tooltip>
              <Tooltip :content="t('common.open')">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7"
                  @click="openUrl(domain.url)"
                >
                  <ExternalLink class="h-3.5 w-3.5" />
                </Button>
              </Tooltip>
            </div>
          </div>

          <!-- Footer -->
          <div class="mt-4 pt-3 border-t border-emerald-500/20">
            <div class="flex items-center justify-between text-xs">
              <span class="text-muted-foreground">{{ t('domains.reserved') }}</span>
              <span class="font-medium text-emerald-600 dark:text-emerald-400">
                {{ formatDate(domain.createdAt) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </TransitionGroup>

    <!-- Reserve Dialog -->
    <Dialog v-model:open="showReserveDialog">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <Globe class="h-5 w-5 text-emerald-500" />
            {{ t('domains.reserveTitle') }}
          </DialogTitle>
        </DialogHeader>

        <form @submit.prevent="reserveDomain" class="space-y-4">
          <div class="space-y-2">
            <Label>{{ t('domains.subdomain') }}</Label>
            <div class="flex gap-2">
              <Input
                v-model="newSubdomain"
                placeholder="my-app"
                @input="availabilityResult = null"
                required
              />
              <Button
                type="button"
                variant="outline"
                @click="checkAvailability"
                :loading="isCheckingAvailability"
                :disabled="!newSubdomain"
              >
                {{ t('common.check') }}
              </Button>
            </div>
            <p class="text-xs text-muted-foreground">
              {{ t('domains.willBeAvailable') }} {{ newSubdomain || 'xxx' }}.mfdev.ru
            </p>
            <p v-if="availabilityResult?.available === true" class="text-sm text-emerald-600 dark:text-emerald-400">
              {{ t('domains.available') }}
            </p>
            <p v-if="availabilityResult?.available === false" class="text-sm text-red-600 dark:text-red-400">
              {{ t('domains.notAvailable') }}
              <span v-if="availabilityResult.reason" class="text-muted-foreground">
                ({{ availabilityResult.reason }})
              </span>
            </p>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" @click="showReserveDialog = false">
              {{ t('common.cancel') }}
            </Button>
            <Button
              type="submit"
              :loading="isReserving"
              :disabled="!newSubdomain || availabilityResult?.available === false"
            >
              {{ t('domains.reserve') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
