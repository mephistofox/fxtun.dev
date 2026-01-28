<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDomainsStore } from '@/stores/domains'
import { toast } from '@/composables/useToast'
import {
  Button, Input, Label, Tooltip,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter
} from '@/components/ui'
import { Globe, Plus, Trash2, Copy, Check, RefreshCw, ExternalLink, Calendar } from 'lucide-vue-next'

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
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="h-10 w-10 rounded-xl bg-type-http/20 border border-type-http/30 flex items-center justify-center">
          <Globe class="h-5 w-5 text-type-http" />
        </div>
        <div>
          <h1 class="font-display text-xl font-bold">{{ t('domains.title') }}</h1>
          <p class="text-xs text-muted-foreground">
            {{ t('domains.subtitle') }}
            <span class="ml-1 px-1.5 py-0.5 rounded bg-type-http/10 text-type-http text-[10px] font-medium">
              {{ domainsStore.domains.length }}/{{ domainsStore.maxDomains }}
            </span>
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" @click="domainsStore.loadDomains">
          <RefreshCw class="h-4 w-4" />
        </Button>
        <Button
          size="sm"
          @click="openReserveDialog"
          :disabled="!canReserve"
          class="bg-type-http hover:bg-type-http/90"
        >
          <Plus class="h-4 w-4 mr-1.5" />
          {{ t('domains.reserve') }}
        </Button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="domainsStore.error" class="flex items-center gap-2 p-3 rounded-lg bg-destructive/10 border border-destructive/30 text-sm text-destructive">
      <Globe class="h-4 w-4" />
      {{ domainsStore.error }}
    </div>

    <!-- Loading -->
    <div v-if="domainsStore.isLoading" class="text-center py-8">
      <RefreshCw class="h-8 w-8 text-type-http animate-spin mx-auto" />
      <p class="mt-2 text-sm text-muted-foreground">{{ t('common.loading') }}</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="domainsStore.domains.length === 0" class="rounded-xl border-2 border-dashed border-muted-foreground/20 p-10 text-center">
      <div class="mx-auto mb-4 h-14 w-14 rounded-xl bg-muted/50 flex items-center justify-center">
        <Globe class="h-7 w-7 text-muted-foreground" />
      </div>
      <p class="font-semibold">{{ t('domains.noDomains') }}</p>
      <p class="mt-1 text-sm text-muted-foreground">{{ t('domains.noDomainsHint') }}</p>
      <Button class="mt-4 bg-type-http hover:bg-type-http/90" @click="openReserveDialog">
        <Plus class="mr-2 h-4 w-4" />
        {{ t('domains.reserve') }}
      </Button>
    </div>

    <!-- Domains grid -->
    <TransitionGroup v-else name="list" tag="div" class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="domain in domainsStore.domains"
        :key="domain.id"
        class="group relative overflow-hidden rounded-xl border transition-all duration-200 hover:shadow-lg bg-gradient-to-br from-card to-card/50 border-type-http/30 hover:border-type-http/60"
      >
        <div class="p-4">
          <!-- Header -->
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-type-http/20">
                <Globe class="h-4 w-4 text-type-http" />
              </div>
              <div>
                <h3 class="font-semibold text-sm">{{ domain.subdomain }}</h3>
                <p class="text-[10px] text-muted-foreground">.mfdev.ru</p>
              </div>
            </div>
            <Tooltip :content="t('domains.releaseDomain')">
              <Button
                variant="ghost"
                size="icon"
                class="h-7 w-7 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                @click="releaseDomain(domain.id)"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </Button>
            </Tooltip>
          </div>

          <!-- URL -->
          <div class="flex items-center gap-1.5 p-2 rounded-lg bg-background/60 border border-border/30 mb-3">
            <code class="flex-1 text-xs font-mono text-type-http break-all">
              {{ domain.url }}
            </code>
            <div class="flex items-center gap-0.5 shrink-0">
              <Tooltip :content="copiedId === domain.id ? t('common.copied') : t('domains.copyUrl')">
                <Button variant="ghost" size="icon" class="h-6 w-6" @click="copyUrl(domain.url, domain.id)">
                  <component :is="copiedId === domain.id ? Check : Copy" :class="['h-3 w-3', copiedId === domain.id ? 'text-success' : 'text-muted-foreground']" />
                </Button>
              </Tooltip>
              <Tooltip :content="t('common.open')">
                <Button variant="ghost" size="icon" class="h-6 w-6" @click="openUrl(domain.url)">
                  <ExternalLink class="h-3 w-3 text-muted-foreground" />
                </Button>
              </Tooltip>
            </div>
          </div>

          <!-- Footer -->
          <div class="flex items-center justify-between text-[10px] text-muted-foreground">
            <span class="flex items-center gap-1">
              <Calendar class="h-3 w-3" />
              {{ t('domains.reserved') }}
            </span>
            <span class="font-medium text-type-http">{{ formatDate(domain.createdAt) }}</span>
          </div>
        </div>
      </div>
    </TransitionGroup>

    <!-- Reserve Dialog -->
    <Dialog v-model:open="showReserveDialog">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <Globe class="h-5 w-5 text-type-http" />
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
                class="font-mono"
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
              {{ t('domains.willBeAvailable') }}
              <code class="px-1 py-0.5 rounded bg-type-http/10 text-type-http font-mono">{{ newSubdomain || 'xxx' }}.mfdev.ru</code>
            </p>

            <!-- Availability result -->
            <div v-if="availabilityResult?.available === true" class="flex items-center gap-2 p-2 rounded-lg bg-success/10 border border-success/30">
              <Check class="h-4 w-4 text-success" />
              <span class="text-sm text-success">{{ t('domains.available') }}</span>
            </div>
            <div v-if="availabilityResult?.available === false" class="flex items-center gap-2 p-2 rounded-lg bg-destructive/10 border border-destructive/30">
              <Globe class="h-4 w-4 text-destructive" />
              <span class="text-sm text-destructive">
                {{ t('domains.notAvailable') }}
                <span v-if="availabilityResult.reason" class="text-muted-foreground">({{ availabilityResult.reason }})</span>
              </span>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" @click="showReserveDialog = false">{{ t('common.cancel') }}</Button>
            <Button
              type="submit"
              :loading="isReserving"
              :disabled="!newSubdomain || availabilityResult?.available === false"
              class="bg-type-http hover:bg-type-http/90"
            >
              {{ t('domains.reserve') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
