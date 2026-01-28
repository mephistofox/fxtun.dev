<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBundlesStore, type CreateBundleInput } from '@/stores/bundles'
import { useTunnelsStore } from '@/stores/tunnels'
import { toast } from '@/composables/useToast'
import {
  Button, Input, Label, Select, Badge, Switch,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, Tooltip
} from '@/components/ui'
import { Plus, Trash2, Play, Download, Upload, Boxes, Zap, Check, Globe, Server, Radio, ArrowRight, Settings2 } from 'lucide-vue-next'
import type { Bundle, TunnelType } from '@/types'

const { t } = useI18n()
const bundlesStore = useBundlesStore()
const tunnelsStore = useTunnelsStore()

// Check if a bundle is already connected (has active tunnel with same type + localPort)
function isBundleConnected(bundle: Bundle): boolean {
  return tunnelsStore.tunnels.some(
    tunnel => tunnel.type === bundle.type && tunnel.localPort === bundle.localPort
  )
}

const showModal = ref(false)
const showDeleteDialog = ref(false)
const deletingBundleId = ref<number | null>(null)
const editingBundle = ref<Bundle | null>(null)
const formData = ref<CreateBundleInput>({
  name: '',
  type: 'http',
  localPort: 0,
  subdomain: '',
  remotePort: 0,
  autoConnect: false,
})

const tunnelTypes = computed(() => [
  { value: 'http', label: t('tunnelTypes.http') },
  { value: 'tcp', label: t('tunnelTypes.tcp') },
  { value: 'udp', label: t('tunnelTypes.udp') },
])

// Get tunnel icon component
function getTunnelIcon(type: TunnelType) {
  switch (type) {
    case 'http': return Globe
    case 'tcp': return Server
    case 'udp': return Radio
  }
}

// Get tunnel accent color - using theme colors
function getTunnelAccent(type: TunnelType): string {
  switch (type) {
    case 'http': return 'text-type-http'
    case 'tcp': return 'text-type-tcp'
    case 'udp': return 'text-type-udp'
  }
}

// Get tunnel gradient class
function getTunnelGradient(type: TunnelType): string {
  switch (type) {
    case 'http': return 'from-type-http/15 to-type-http/5'
    case 'tcp': return 'from-type-tcp/15 to-type-tcp/5'
    case 'udp': return 'from-type-udp/15 to-type-udp/5'
  }
}

// Get tunnel border color
function getTunnelBorder(type: TunnelType): string {
  switch (type) {
    case 'http': return 'border-type-http/30 hover:border-type-http/60'
    case 'tcp': return 'border-type-tcp/30 hover:border-type-tcp/60'
    case 'udp': return 'border-type-udp/30 hover:border-type-udp/60'
  }
}

// Get tunnel bg for buttons
function getTunnelBgClass(type: TunnelType): string {
  switch (type) {
    case 'http': return 'bg-type-http hover:bg-type-http/90 shadow-lg shadow-type-http/25'
    case 'tcp': return 'bg-type-tcp hover:bg-type-tcp/90 shadow-lg shadow-type-tcp/25'
    case 'udp': return 'bg-type-udp hover:bg-type-udp/90 shadow-lg shadow-type-udp/25'
  }
}

// Get glow color
function getGlowColor(type: TunnelType): string {
  switch (type) {
    case 'http': return 'group-hover:shadow-type-http/20'
    case 'tcp': return 'group-hover:shadow-type-tcp/20'
    case 'udp': return 'group-hover:shadow-type-udp/20'
  }
}

onMounted(() => {
  bundlesStore.loadBundles()
})

function openCreateModal() {
  editingBundle.value = null
  formData.value = {
    name: '',
    type: 'http',
    localPort: 0,
    subdomain: '',
    remotePort: 0,
    autoConnect: false,
  }
  showModal.value = true
}

function openEditModal(bundle: Bundle) {
  editingBundle.value = bundle
  formData.value = {
    name: bundle.name,
    type: bundle.type,
    localPort: bundle.localPort,
    subdomain: bundle.subdomain || '',
    remotePort: bundle.remotePort || 0,
    autoConnect: bundle.autoConnect,
  }
  showModal.value = true
}

async function saveBundle() {
  if (editingBundle.value) {
    await bundlesStore.updateBundle({
      ...editingBundle.value,
      ...formData.value,
    })
    toast({ title: t('toasts.bundleUpdated'), variant: 'success' })
  } else {
    await bundlesStore.createBundle(formData.value)
    toast({ title: t('toasts.bundleCreated'), variant: 'success' })
  }
  showModal.value = false
}

function confirmDeleteBundle(id: number) {
  deletingBundleId.value = id
  showDeleteDialog.value = true
}

async function deleteBundle() {
  if (deletingBundleId.value !== null) {
    await bundlesStore.deleteBundle(deletingBundleId.value)
    toast({ title: t('toasts.bundleDeleted'), variant: 'success' })
  }
  showDeleteDialog.value = false
  deletingBundleId.value = null
}

async function connectBundle(id: number) {
  await bundlesStore.connectBundle(id)
  toast({ title: t('toasts.tunnelCreated'), variant: 'success' })
}

async function exportBundles() {
  const data = await bundlesStore.exportBundles()
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'fxtunnel-bundles.json'
  a.click()
  URL.revokeObjectURL(url)
  toast({ title: t('toasts.bundlesExported'), variant: 'success' })
}

async function importBundles() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.json'
  input.onchange = async (e) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (file) {
      const text = await file.text()
      await bundlesStore.importBundles(text)
      toast({ title: t('toasts.bundlesImported'), variant: 'success' })
    }
  }
  input.click()
}

function getTunnelTypeBadge(type: TunnelType): 'http' | 'tcp' | 'udp' {
  return type
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <!-- Icon with glow -->
        <div class="relative">
          <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-20 blur-lg" />
          <div class="relative flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/20 to-accent/20 border border-primary/30">
            <Boxes class="h-7 w-7 text-primary" />
          </div>
        </div>
        <div>
          <h1 class="font-display text-2xl font-bold tracking-tight">{{ t('bundles.title') }}</h1>
          <p class="text-sm text-muted-foreground">{{ t('bundles.subtitle') || 'Manage your saved tunnel configurations' }}</p>
        </div>
      </div>
      <div class="flex gap-2">
        <Tooltip :content="t('bundles.import')">
          <Button variant="outline" size="sm" class="border-border/50 hover:border-primary/50 hover:bg-primary/5" @click="importBundles">
            <Upload class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('bundles.import') }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('bundles.export')">
          <Button variant="outline" size="sm" class="border-border/50 hover:border-primary/50 hover:bg-primary/5" @click="exportBundles">
            <Download class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('bundles.export') }}</span>
          </Button>
        </Tooltip>
        <Button class="bg-gradient-to-r from-primary to-primary hover:to-accent shadow-lg shadow-primary/25 transition-all duration-300" @click="openCreateModal">
          <Plus class="h-4 w-4 sm:mr-2" />
          <span class="hidden sm:inline">{{ t('bundles.newBundle') }}</span>
        </Button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="bundlesStore.bundles.length === 0" class="cyber-card rounded-2xl border-2 border-dashed border-primary/20 p-12 text-center">
      <div class="relative mx-auto mb-6 w-fit">
        <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-20 blur-xl" />
        <div class="relative flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/10 to-accent/10 border border-primary/20">
          <Boxes class="h-10 w-10 text-primary" />
        </div>
      </div>
      <p class="font-display text-xl font-semibold">{{ t('bundles.noSaved') }}</p>
      <p class="mt-3 text-sm text-muted-foreground max-w-md mx-auto">
        {{ t('bundles.noSavedHint') }}
      </p>
      <Button class="mt-6 bg-gradient-to-r from-primary to-accent shadow-lg shadow-primary/25" size="lg" @click="openCreateModal">
        <Plus class="mr-2 h-5 w-5" />
        {{ t('bundles.createBundle') }}
      </Button>
    </div>

    <!-- Bundles Grid -->
    <TransitionGroup v-else name="list" tag="div" class="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="bundle in bundlesStore.bundles"
        :key="bundle.id"
        :class="[
          'group relative overflow-hidden rounded-2xl border-2 bg-gradient-to-br transition-all duration-300 hover:shadow-2xl',
          getTunnelGradient(bundle.type),
          getTunnelBorder(bundle.type),
          getGlowColor(bundle.type)
        ]"
      >
        <!-- Animated border glow on hover -->
        <div class="absolute inset-0 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"
             :class="bundle.type === 'http' ? 'bg-gradient-to-br from-type-http/10 to-transparent' : bundle.type === 'tcp' ? 'bg-gradient-to-br from-type-tcp/10 to-transparent' : 'bg-gradient-to-br from-type-udp/10 to-transparent'" />

        <!-- Top accent line -->
        <div :class="['absolute top-0 left-0 right-0 h-1', bundle.type === 'http' ? 'bg-type-http' : bundle.type === 'tcp' ? 'bg-type-tcp' : 'bg-type-udp']" />

        <!-- Connected indicator -->
        <div v-if="isBundleConnected(bundle)" class="absolute top-4 right-4 z-10">
          <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-success/20 text-success text-xs font-medium border border-success/30">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-success"></span>
            </span>
            {{ t('bundles.active') || 'Active' }}
          </div>
        </div>

        <div class="relative p-5 pt-6">
          <!-- Header with Icon -->
          <div class="flex items-start gap-4 mb-5">
            <div :class="[
              'flex h-14 w-14 items-center justify-center rounded-2xl transition-all duration-300 group-hover:scale-110 group-hover:shadow-lg',
              bundle.type === 'http' ? 'bg-type-http/20 group-hover:shadow-type-http/30' : bundle.type === 'tcp' ? 'bg-type-tcp/20 group-hover:shadow-type-tcp/30' : 'bg-type-udp/20 group-hover:shadow-type-udp/30'
            ]">
              <component :is="getTunnelIcon(bundle.type)" :class="['h-7 w-7', getTunnelAccent(bundle.type)]" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-display text-lg font-bold truncate">{{ bundle.name }}</h3>
              <Badge :variant="getTunnelTypeBadge(bundle.type)" class="mt-1">
                {{ bundle.type.toUpperCase() }}
              </Badge>
            </div>
          </div>

          <!-- Port mapping visualization -->
          <div class="flex items-center gap-3 p-4 rounded-xl bg-background/60 backdrop-blur-sm border border-border/30 mb-4">
            <div class="flex-1 text-center">
              <p class="text-xs text-muted-foreground mb-1 uppercase tracking-wider">{{ t('bundles.localPort') }}</p>
              <p class="font-mono text-xl font-bold">:{{ bundle.localPort }}</p>
            </div>
            <div class="flex flex-col items-center">
              <ArrowRight :class="['h-6 w-6 transition-transform group-hover:translate-x-1', getTunnelAccent(bundle.type)]" />
            </div>
            <div class="flex-1 text-center">
              <p class="text-xs text-muted-foreground mb-1 uppercase tracking-wider">{{ bundle.type === 'http' ? t('bundles.subdomain') : t('bundles.remotePort') }}</p>
              <p :class="['font-mono text-xl font-bold', getTunnelAccent(bundle.type)]">
                {{ bundle.type === 'http' ? (bundle.subdomain || 'auto') : (bundle.remotePort || 'auto') }}
              </p>
            </div>
          </div>

          <!-- Auto-connect badge -->
          <div v-if="bundle.autoConnect" class="flex items-center gap-2 mb-4 px-3 py-2 rounded-xl bg-warning/10 border border-warning/20">
            <Zap class="h-4 w-4 text-warning" />
            <span class="text-sm text-warning font-medium">{{ t('bundles.autoConnectEnabled') || 'Auto-connect enabled' }}</span>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-2">
            <Button
              v-if="isBundleConnected(bundle)"
              variant="outline"
              class="flex-1 border-success/30 text-success bg-success/5"
              disabled
            >
              <Check class="mr-2 h-4 w-4" />
              {{ t('bundles.connected') }}
            </Button>
            <Button
              v-else
              :class="['flex-1 text-white transition-all duration-300', getTunnelBgClass(bundle.type)]"
              @click="connectBundle(bundle.id)"
            >
              <Play class="mr-2 h-4 w-4" />
              {{ t('bundles.connect') }}
            </Button>

            <Tooltip :content="t('common.edit')">
              <Button
                variant="outline"
                size="icon"
                class="h-10 w-10 border-border/50 hover:border-primary/50 hover:bg-primary/10"
                @click="openEditModal(bundle)"
              >
                <Settings2 class="h-4 w-4" />
              </Button>
            </Tooltip>
            <Tooltip :content="t('common.delete')">
              <Button
                variant="outline"
                size="icon"
                class="h-10 w-10 border-border/50 hover:border-destructive hover:text-destructive hover:bg-destructive/10"
                @click="confirmDeleteBundle(bundle.id)"
              >
                <Trash2 class="h-4 w-4" />
              </Button>
            </Tooltip>
          </div>
        </div>
      </div>
    </TransitionGroup>

    <!-- Create/Edit Modal -->
    <Dialog v-model:open="showModal">
      <DialogContent class="border-border/50 bg-card/95 backdrop-blur-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-3 font-display">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-primary/20 to-accent/20 border border-primary/30">
              <Boxes class="h-5 w-5 text-primary" />
            </div>
            {{ editingBundle ? t('bundles.editBundle') : t('bundles.createBundle') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('bundles.bundleDescription') }}
          </DialogDescription>
        </DialogHeader>

        <form @submit.prevent="saveBundle" class="space-y-4">
          <div class="space-y-2">
            <Label class="text-xs uppercase tracking-wider text-muted-foreground">{{ t('bundles.name') }}</Label>
            <Input v-model="formData.name" :placeholder="t('bundles.namePlaceholder')" class="bg-muted/30 border-border/50" />
          </div>

          <div class="space-y-2">
            <Label class="text-xs uppercase tracking-wider text-muted-foreground">{{ t('bundles.type') }}</Label>
            <Select
              v-model="formData.type"
              :options="tunnelTypes"
              class="bg-muted/30"
            />
          </div>

          <div class="space-y-2">
            <Label class="text-xs uppercase tracking-wider text-muted-foreground">{{ t('bundles.localPort') }}</Label>
            <Input
              v-model.number="formData.localPort"
              type="number"
              placeholder="3000"
              class="bg-muted/30 border-border/50 font-mono"
            />
          </div>

          <div v-if="formData.type === 'http'" class="space-y-2">
            <Label class="text-xs uppercase tracking-wider text-muted-foreground">
              {{ t('bundles.subdomain') }}
              <span class="text-muted-foreground/60 lowercase">{{ t('dashboard.optional') }}</span>
            </Label>
            <Input v-model="formData.subdomain" :placeholder="t('bundles.subdomainPlaceholder')" class="bg-muted/30 border-border/50 font-mono" />
          </div>

          <div v-else class="space-y-2">
            <Label class="text-xs uppercase tracking-wider text-muted-foreground">
              {{ t('bundles.remotePort') }}
              <span class="text-muted-foreground/60 lowercase">{{ t('dashboard.optional') }}</span>
            </Label>
            <Input
              v-model.number="formData.remotePort"
              type="number"
              placeholder="0"
              class="bg-muted/30 border-border/50 font-mono"
            />
            <p class="text-xs text-muted-foreground">{{ t('bundles.remotePortHint') }}</p>
          </div>

          <div class="flex items-center justify-between p-4 rounded-xl bg-warning/5 border border-warning/20">
            <div class="flex items-center gap-3">
              <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-warning/20">
                <Zap class="h-4 w-4 text-warning" />
              </div>
              <Label class="cursor-pointer font-medium">{{ t('bundles.autoConnectOnStartup') }}</Label>
            </div>
            <Switch v-model="formData.autoConnect" />
          </div>

          <DialogFooter class="pt-4">
            <Button type="button" variant="outline" class="border-border/50" @click="showModal = false">
              {{ t('common.cancel') }}
            </Button>
            <Button type="submit" class="bg-gradient-to-r from-primary to-primary hover:to-accent shadow-lg shadow-primary/25" :disabled="!formData.name || !formData.localPort">
              {{ editingBundle ? t('common.save') : t('common.create') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <DialogContent class="border-destructive/30 bg-card/95 backdrop-blur-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-3 text-destructive font-display">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-destructive/20 border border-destructive/30">
              <Trash2 class="h-5 w-5" />
            </div>
            {{ t('bundles.deleteBundle') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('bundles.confirmDelete') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" class="border-border/50" @click="showDeleteDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" class="shadow-lg shadow-destructive/25" @click="deleteBundle">
            {{ t('common.delete') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
