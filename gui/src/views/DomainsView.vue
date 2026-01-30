<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDomainsStore } from '@/stores/domains'
import { useCustomDomainsStore } from '@/stores/customDomains'
import { toast } from '@/composables/useToast'
import {
  Button, Input, Label, Tooltip,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter
} from '@/components/ui'
import { Globe, Plus, Trash2, Copy, Check, RefreshCw, ExternalLink, Calendar, Link, ShieldCheck, Clock } from 'lucide-vue-next'

const { t, locale } = useI18n()
const domainsStore = useDomainsStore()
const customDomainsStore = useCustomDomainsStore()

// --- Reserved subdomains ---
const showReserveDialog = ref(false)
const newSubdomain = ref('')
const isReserving = ref(false)
const isCheckingAvailability = ref(false)
const availabilityResult = ref<{ available: boolean; reason?: string } | null>(null)
const copiedId = ref<number | null>(null)

const canReserve = computed(() =>
  domainsStore.domains.length < domainsStore.maxDomains
)

// --- Custom domains ---
const showAddCustomDialog = ref(false)
const newCustomDomain = ref('')
const newTargetSubdomain = ref('')
const isAddingCustom = ref(false)
const verifyingId = ref<number | null>(null)

const canAddCustom = computed(() =>
  customDomainsStore.domains.length < customDomainsStore.maxDomains
)

const reservedSubdomainOptions = computed(() =>
  domainsStore.domains.map(d => d.subdomain)
)

onMounted(async () => {
  await Promise.all([
    domainsStore.loadDomains(),
    customDomainsStore.loadDomains(),
  ])
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

// --- Custom domain functions ---
function openAddCustomDialog() {
  newCustomDomain.value = ''
  newTargetSubdomain.value = reservedSubdomainOptions.value[0] || ''
  showAddCustomDialog.value = false
  showAddCustomDialog.value = true
}

async function addCustomDomain() {
  if (!newCustomDomain.value || !newTargetSubdomain.value) return

  isAddingCustom.value = true
  const result = await customDomainsStore.addDomain(newCustomDomain.value, newTargetSubdomain.value)
  isAddingCustom.value = false

  if (result) {
    toast({ title: t('toasts.customDomainAdded'), variant: 'success' })
    showAddCustomDialog.value = false
    newCustomDomain.value = ''
    newTargetSubdomain.value = ''
  }
}

async function deleteCustomDomain(id: number) {
  const success = await customDomainsStore.deleteDomain(id)
  if (success) {
    toast({ title: t('toasts.customDomainDeleted'), variant: 'success' })
  }
}

async function verifyCustomDomain(id: number) {
  verifyingId.value = id
  const result = await customDomainsStore.verifyDomain(id)
  verifyingId.value = null

  if (result?.verified) {
    toast({ title: t('toasts.customDomainVerified'), variant: 'success' })
  } else if (result?.error) {
    toast({ title: result.error, variant: 'destructive' })
  }
}
</script>

<template>
  <div class="space-y-8">
    <!-- ===== RESERVED SUBDOMAINS SECTION ===== -->
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
            <div class="flex items-center gap-3 mb-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-type-http/20">
                <Globe class="h-5 w-5 text-type-http" />
              </div>
              <div class="flex-1 min-w-0">
                <h3 class="font-semibold truncate">{{ domain.subdomain }}</h3>
                <p class="text-[10px] text-muted-foreground">.mfdev.ru</p>
              </div>
            </div>

            <!-- Date -->
            <div class="flex items-center gap-1.5 mb-3 px-2 py-1 rounded-lg bg-type-http/5 border border-type-http/20 text-xs">
              <Calendar class="h-3 w-3 text-type-http" />
              <span class="text-type-http font-medium">{{ formatDate(domain.createdAt) }}</span>
            </div>

            <!-- Actions -->
            <div class="flex items-center gap-2">
              <Tooltip :content="copiedId === domain.id ? t('common.copied') : t('domains.copyUrl')">
                <Button
                  variant="outline"
                  size="sm"
                  class="flex-1 h-8 border-type-http/30 text-type-http hover:bg-type-http/10"
                  @click="copyUrl(domain.url, domain.id)"
                >
                  <component :is="copiedId === domain.id ? Check : Copy" class="mr-1.5 h-3.5 w-3.5" />
                  {{ copiedId === domain.id ? t('common.copied') : t('domains.copyUrl') }}
                </Button>
              </Tooltip>
              <Tooltip :content="t('common.open')">
                <Button variant="outline" size="icon" class="h-8 w-8" @click="openUrl(domain.url)">
                  <ExternalLink class="h-3.5 w-3.5" />
                </Button>
              </Tooltip>
              <Tooltip :content="t('domains.releaseDomain')">
                <Button variant="outline" size="icon" class="h-8 w-8 hover:border-destructive hover:text-destructive hover:bg-destructive/10" @click="releaseDomain(domain.id)">
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </Tooltip>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>

    <!-- ===== CUSTOM DOMAINS SECTION ===== -->
    <div class="space-y-5">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 rounded-xl bg-blue-500/20 border border-blue-500/30 flex items-center justify-center">
            <Link class="h-5 w-5 text-blue-500" />
          </div>
          <div>
            <h1 class="font-display text-xl font-bold">{{ t('customDomains.title') }}</h1>
            <p class="text-xs text-muted-foreground">
              {{ t('customDomains.subtitle') }}
              <span class="ml-1 px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500 text-[10px] font-medium">
                {{ customDomainsStore.domains.length }}/{{ customDomainsStore.maxDomains }}
              </span>
            </p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" @click="customDomainsStore.loadDomains">
            <RefreshCw class="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            @click="openAddCustomDialog"
            :disabled="!canAddCustom || reservedSubdomainOptions.length === 0"
            class="bg-blue-500 hover:bg-blue-500/90 text-white"
          >
            <Plus class="h-4 w-4 mr-1.5" />
            {{ t('customDomains.add') }}
          </Button>
        </div>
      </div>

      <!-- Error -->
      <div v-if="customDomainsStore.error" class="flex items-center gap-2 p-3 rounded-lg bg-destructive/10 border border-destructive/30 text-sm text-destructive">
        <Link class="h-4 w-4" />
        {{ customDomainsStore.error }}
      </div>

      <!-- Loading -->
      <div v-if="customDomainsStore.isLoading" class="text-center py-8">
        <RefreshCw class="h-8 w-8 text-blue-500 animate-spin mx-auto" />
        <p class="mt-2 text-sm text-muted-foreground">{{ t('common.loading') }}</p>
      </div>

      <!-- Empty state -->
      <div v-else-if="customDomainsStore.domains.length === 0" class="rounded-xl border-2 border-dashed border-muted-foreground/20 p-10 text-center">
        <div class="mx-auto mb-4 h-14 w-14 rounded-xl bg-muted/50 flex items-center justify-center">
          <Link class="h-7 w-7 text-muted-foreground" />
        </div>
        <p class="font-semibold">{{ t('customDomains.noDomains') }}</p>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('customDomains.noDomainsHint') }}</p>
        <Button
          class="mt-4 bg-blue-500 hover:bg-blue-500/90 text-white"
          @click="openAddCustomDialog"
          :disabled="reservedSubdomainOptions.length === 0"
        >
          <Plus class="mr-2 h-4 w-4" />
          {{ t('customDomains.add') }}
        </Button>
      </div>

      <!-- Custom domains grid -->
      <TransitionGroup v-else name="list" tag="div" class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="cd in customDomainsStore.domains"
          :key="cd.id"
          class="group relative overflow-hidden rounded-xl border transition-all duration-200 hover:shadow-lg bg-gradient-to-br from-card to-card/50"
          :class="cd.verified ? 'border-blue-500/30 hover:border-blue-500/60' : 'border-yellow-500/30 hover:border-yellow-500/60'"
        >
          <div class="p-4">
            <!-- Header -->
            <div class="flex items-center gap-3 mb-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-xl" :class="cd.verified ? 'bg-blue-500/20' : 'bg-yellow-500/20'">
                <Link class="h-5 w-5" :class="cd.verified ? 'text-blue-500' : 'text-yellow-500'" />
              </div>
              <div class="flex-1 min-w-0">
                <h3 class="font-semibold truncate">{{ cd.domain }}</h3>
                <p class="text-[10px] text-muted-foreground">→ {{ cd.targetSubdomain }}.{{ customDomainsStore.baseDomain || 'mfdev.ru' }}</p>
              </div>
            </div>

            <!-- Status -->
            <div class="flex items-center gap-1.5 mb-3 px-2 py-1 rounded-lg text-xs" :class="cd.verified ? 'bg-blue-500/5 border border-blue-500/20' : 'bg-yellow-500/5 border border-yellow-500/20'">
              <component :is="cd.verified ? ShieldCheck : Clock" class="h-3 w-3" :class="cd.verified ? 'text-blue-500' : 'text-yellow-500'" />
              <span class="font-medium" :class="cd.verified ? 'text-blue-500' : 'text-yellow-500'">
                {{ cd.verified ? t('customDomains.verified') : t('customDomains.pending') }}
              </span>
              <span v-if="cd.verified && cd.verifiedAt" class="text-muted-foreground ml-auto">{{ formatDate(cd.verifiedAt) }}</span>
            </div>

            <!-- DNS hint for unverified -->
            <div v-if="!cd.verified" class="mb-3 p-2 rounded-lg bg-muted/50 text-[10px] text-muted-foreground font-mono break-all">
              → {{ cd.targetSubdomain }}.{{ customDomainsStore.baseDomain || 'mfdev.ru' }}
            </div>

            <!-- Actions -->
            <div class="flex items-center gap-2">
              <Button
                v-if="!cd.verified"
                variant="outline"
                size="sm"
                class="flex-1 h-8 border-yellow-500/30 text-yellow-500 hover:bg-yellow-500/10"
                @click="verifyCustomDomain(cd.id)"
                :disabled="verifyingId === cd.id"
              >
                <RefreshCw v-if="verifyingId === cd.id" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
                <ShieldCheck v-else class="mr-1.5 h-3.5 w-3.5" />
                {{ verifyingId === cd.id ? t('customDomains.verifying') : t('customDomains.verify') }}
              </Button>
              <Tooltip v-if="cd.verified" :content="t('common.open')">
                <Button variant="outline" size="sm" class="flex-1 h-8" @click="openUrl('https://' + cd.domain)">
                  <ExternalLink class="mr-1.5 h-3.5 w-3.5" />
                  {{ t('common.open') }}
                </Button>
              </Tooltip>
              <Tooltip :content="t('customDomains.delete')">
                <Button variant="outline" size="icon" class="h-8 w-8 hover:border-destructive hover:text-destructive hover:bg-destructive/10" @click="deleteCustomDomain(cd.id)">
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </Tooltip>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>

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

    <!-- Add Custom Domain Dialog -->
    <Dialog v-model:open="showAddCustomDialog">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <Link class="h-5 w-5 text-blue-500" />
            {{ t('customDomains.addTitle') }}
          </DialogTitle>
        </DialogHeader>

        <form @submit.prevent="addCustomDomain" class="space-y-4">
          <div class="space-y-2">
            <Label>{{ t('customDomains.domain') }}</Label>
            <Input
              v-model="newCustomDomain"
              :placeholder="t('customDomains.domainPlaceholder')"
              class="font-mono"
              required
            />
          </div>

          <div class="space-y-2">
            <Label>{{ t('customDomains.targetSubdomain') }}</Label>
            <select
              v-model="newTargetSubdomain"
              class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 font-mono"
              required
            >
              <option v-for="sub in reservedSubdomainOptions" :key="sub" :value="sub">
                {{ sub }}
              </option>
            </select>
          </div>

          <div v-if="newCustomDomain && newTargetSubdomain" class="p-4 rounded-lg bg-blue-500/5 border border-blue-500/20 text-xs space-y-3">
            <p class="font-semibold text-blue-500">{{ t('customDomains.cnameHint') }}</p>

            <div class="space-y-2">
              <div class="bg-blue-500/10 px-3 py-2 rounded">
                <p class="font-medium text-muted-foreground mb-1">{{ t('customDomains.dnsGuideSubdomain') }}:</p>
                <code class="block text-blue-500 font-mono">{{ newCustomDomain }} → CNAME → {{ newTargetSubdomain }}.{{ customDomainsStore.baseDomain || 'mfdev.ru' }}</code>
              </div>
              <div class="bg-blue-500/10 px-3 py-2 rounded">
                <p class="font-medium text-muted-foreground mb-1">{{ t('customDomains.dnsGuideApex') }}:</p>
                <code class="block text-blue-500 font-mono">{{ newCustomDomain }} → A → {{ customDomainsStore.serverIP || '...' }}</code>
              </div>
            </div>

            <p class="text-muted-foreground whitespace-pre-line">{{ t('customDomains.dnsGuideSteps') }}</p>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" @click="showAddCustomDialog = false">{{ t('common.cancel') }}</Button>
            <Button
              type="submit"
              :loading="isAddingCustom"
              :disabled="!newCustomDomain || !newTargetSubdomain"
              class="bg-blue-500 hover:bg-blue-500/90 text-white"
            >
              {{ t('customDomains.add') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
