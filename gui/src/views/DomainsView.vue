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
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <div class="relative">
          <div class="absolute inset-0 rounded-2xl bg-type-http opacity-20 blur-lg" />
          <div class="relative flex h-14 w-14 items-center justify-center rounded-2xl bg-type-http/20 border border-type-http/30">
            <Globe class="h-7 w-7 text-type-http" />
          </div>
        </div>
        <div>
          <h1 class="font-display text-2xl font-bold tracking-tight">{{ t('domains.title') }}</h1>
          <p class="text-sm text-muted-foreground">
            {{ t('domains.subtitle') }}
            <span class="inline-flex items-center gap-1 ml-2 px-2 py-0.5 rounded-full bg-type-http/10 text-type-http text-xs font-medium">
              {{ domainsStore.domains.length }}/{{ domainsStore.maxDomains }}
            </span>
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Tooltip :content="t('common.refresh')">
          <Button variant="outline" size="sm" class="border-border/50 hover:border-type-http/50 hover:bg-type-http/5" @click="domainsStore.loadDomains">
            <RefreshCw class="h-4 w-4" />
          </Button>
        </Tooltip>
        <Button
          @click="openReserveDialog"
          :disabled="!canReserve"
          class="bg-type-http hover:bg-type-http/90 shadow-lg shadow-type-http/25 transition-all duration-300"
        >
          <Plus class="h-4 w-4 mr-2" />
          {{ t('domains.reserve') }}
        </Button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="domainsStore.error" class="flex items-center gap-3 p-4 rounded-xl bg-destructive/10 border border-destructive/30 text-sm text-destructive">
      <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-destructive/20">
        <Globe class="h-4 w-4" />
      </div>
      {{ domainsStore.error }}
    </div>

    <!-- Loading -->
    <div v-if="domainsStore.isLoading" class="text-center py-12">
      <div class="relative mx-auto w-fit">
        <div class="absolute inset-0 rounded-full bg-type-http/30 blur-lg animate-pulse" />
        <div class="relative flex h-16 w-16 items-center justify-center rounded-full bg-type-http/20 border border-type-http/30">
          <RefreshCw class="h-8 w-8 text-type-http animate-spin" />
        </div>
      </div>
      <p class="mt-4 text-muted-foreground">{{ t('common.loading') }}</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="domainsStore.domains.length === 0" class="cyber-card rounded-2xl border-2 border-dashed border-type-http/20 p-12 text-center">
      <div class="relative mx-auto mb-6 w-fit">
        <div class="absolute inset-0 rounded-2xl bg-type-http opacity-20 blur-xl" />
        <div class="relative flex h-20 w-20 items-center justify-center rounded-2xl bg-type-http/10 border border-type-http/20">
          <Globe class="h-10 w-10 text-type-http" />
        </div>
      </div>
      <p class="font-display text-xl font-semibold">{{ t('domains.noDomains') }}</p>
      <p class="mt-3 text-sm text-muted-foreground max-w-md mx-auto">
        {{ t('domains.noDomainsHint') }}
      </p>
      <Button class="mt-6 bg-type-http hover:bg-type-http/90 shadow-lg shadow-type-http/25" @click="openReserveDialog">
        <Plus class="h-4 w-4 mr-2" />
        {{ t('domains.reserve') }}
      </Button>
    </div>

    <!-- Domains grid -->
    <TransitionGroup v-else name="list" tag="div" class="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="domain in domainsStore.domains"
        :key="domain.id"
        class="group relative overflow-hidden rounded-2xl border-2 bg-gradient-to-br from-type-http/15 to-type-http/5 border-type-http/30 hover:border-type-http/60 transition-all duration-300 hover:shadow-2xl hover:shadow-type-http/10"
      >
        <!-- Top accent line -->
        <div class="absolute top-0 left-0 right-0 h-1 bg-type-http" />

        <!-- Hover glow effect -->
        <div class="absolute inset-0 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none bg-gradient-to-br from-type-http/10 to-transparent" />

        <div class="relative p-5 pt-6">
          <!-- Header -->
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-type-http/20 transition-all duration-300 group-hover:scale-110 group-hover:shadow-lg group-hover:shadow-type-http/30">
                <Globe class="h-6 w-6 text-type-http" />
              </div>
              <div>
                <h3 class="font-display font-bold text-lg">{{ domain.subdomain }}</h3>
                <p class="text-xs text-muted-foreground">.mfdev.ru</p>
              </div>
            </div>
            <Tooltip :content="t('domains.releaseDomain')">
              <Button
                variant="ghost"
                size="icon"
                class="h-8 w-8 opacity-0 group-hover:opacity-100 transition-all text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                @click="releaseDomain(domain.id)"
              >
                <Trash2 class="h-4 w-4" />
              </Button>
            </Tooltip>
          </div>

          <!-- URL -->
          <div class="flex items-center gap-2 p-3 rounded-xl bg-background/60 backdrop-blur-sm border border-border/30">
            <code class="flex-1 truncate text-xs font-mono font-medium text-type-http">
              {{ domain.url }}
            </code>
            <div class="flex items-center gap-1">
              <Tooltip :content="copiedId === domain.id ? t('common.copied') : t('domains.copyUrl')">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7 hover:bg-type-http/10"
                  @click="copyUrl(domain.url, domain.id)"
                >
                  <component
                    :is="copiedId === domain.id ? Check : Copy"
                    :class="['h-3.5 w-3.5', copiedId === domain.id ? 'text-type-http' : 'text-muted-foreground']"
                  />
                </Button>
              </Tooltip>
              <Tooltip :content="t('common.open')">
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7 hover:bg-type-http/10"
                  @click="openUrl(domain.url)"
                >
                  <ExternalLink class="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
              </Tooltip>
            </div>
          </div>

          <!-- Footer -->
          <div class="mt-4 pt-3 border-t border-type-http/20">
            <div class="flex items-center justify-between text-xs">
              <span class="flex items-center gap-1.5 text-muted-foreground">
                <Calendar class="h-3.5 w-3.5" />
                {{ t('domains.reserved') }}
              </span>
              <span class="font-medium text-type-http">
                {{ formatDate(domain.createdAt) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </TransitionGroup>

    <!-- Reserve Dialog -->
    <Dialog v-model:open="showReserveDialog">
      <DialogContent class="sm:max-w-md border-type-http/30 bg-card/95 backdrop-blur-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-3 font-display">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-type-http/20 border border-type-http/30">
              <Globe class="h-5 w-5 text-type-http" />
            </div>
            {{ t('domains.reserveTitle') }}
          </DialogTitle>
        </DialogHeader>

        <form @submit.prevent="reserveDomain" class="space-y-4">
          <div class="space-y-3">
            <Label class="text-xs uppercase tracking-wider text-muted-foreground">{{ t('domains.subdomain') }}</Label>
            <div class="flex gap-2">
              <Input
                v-model="newSubdomain"
                placeholder="my-app"
                class="bg-muted/30 border-border/50 font-mono"
                @input="availabilityResult = null"
                required
              />
              <Button
                type="button"
                variant="outline"
                class="border-type-http/30 hover:border-type-http/50 hover:bg-type-http/5"
                @click="checkAvailability"
                :loading="isCheckingAvailability"
                :disabled="!newSubdomain"
              >
                {{ t('common.check') }}
              </Button>
            </div>
            <p class="text-xs text-muted-foreground">
              {{ t('domains.willBeAvailable') }}
              <code class="px-1.5 py-0.5 rounded bg-type-http/10 text-type-http font-mono">{{ newSubdomain || 'xxx' }}.mfdev.ru</code>
            </p>

            <!-- Availability result -->
            <div v-if="availabilityResult?.available === true" class="flex items-center gap-2 p-3 rounded-xl bg-success/10 border border-success/30">
              <Check class="h-4 w-4 text-success" />
              <span class="text-sm font-medium text-success">{{ t('domains.available') }}</span>
            </div>
            <div v-if="availabilityResult?.available === false" class="flex items-center gap-2 p-3 rounded-xl bg-destructive/10 border border-destructive/30">
              <Globe class="h-4 w-4 text-destructive" />
              <span class="text-sm text-destructive">
                {{ t('domains.notAvailable') }}
                <span v-if="availabilityResult.reason" class="text-muted-foreground">
                  ({{ availabilityResult.reason }})
                </span>
              </span>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" class="border-border/50" @click="showReserveDialog = false">
              {{ t('common.cancel') }}
            </Button>
            <Button
              type="submit"
              :loading="isReserving"
              :disabled="!newSubdomain || availabilityResult?.available === false"
              class="bg-type-http hover:bg-type-http/90 shadow-lg shadow-type-http/25"
            >
              {{ t('domains.reserve') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
