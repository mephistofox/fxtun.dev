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

// Get tunnel accent color
function getTunnelAccent(type: TunnelType): string {
  switch (type) {
    case 'http': return 'text-emerald-500'
    case 'tcp': return 'text-blue-500'
    case 'udp': return 'text-purple-500'
  }
}

// Get tunnel gradient class
function getTunnelGradient(type: TunnelType): string {
  switch (type) {
    case 'http': return 'from-emerald-500/15 to-emerald-500/5'
    case 'tcp': return 'from-blue-500/15 to-blue-500/5'
    case 'udp': return 'from-purple-500/15 to-purple-500/5'
  }
}

// Get tunnel border color
function getTunnelBorder(type: TunnelType): string {
  switch (type) {
    case 'http': return 'border-emerald-500/30 hover:border-emerald-500/50'
    case 'tcp': return 'border-blue-500/30 hover:border-blue-500/50'
    case 'udp': return 'border-purple-500/30 hover:border-purple-500/50'
  }
}

// Get tunnel bg for buttons
function getTunnelBg(type: TunnelType): string {
  switch (type) {
    case 'http': return 'bg-emerald-500 hover:bg-emerald-600'
    case 'tcp': return 'bg-blue-500 hover:bg-blue-600'
    case 'udp': return 'bg-purple-500 hover:bg-purple-600'
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
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10">
          <Boxes class="h-5 w-5 text-primary" />
        </div>
        <div>
          <h1 class="text-2xl font-bold">{{ t('bundles.title') }}</h1>
          <p class="text-sm text-muted-foreground">{{ t('bundles.subtitle') || 'Manage your saved tunnel configurations' }}</p>
        </div>
      </div>
      <div class="flex gap-2">
        <Tooltip :content="t('bundles.import')">
          <Button variant="outline" size="sm" @click="importBundles">
            <Upload class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('bundles.import') }}</span>
          </Button>
        </Tooltip>
        <Tooltip :content="t('bundles.export')">
          <Button variant="outline" size="sm" @click="exportBundles">
            <Download class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t('bundles.export') }}</span>
          </Button>
        </Tooltip>
        <Button @click="openCreateModal">
          <Plus class="h-4 w-4 sm:mr-2" />
          <span class="hidden sm:inline">{{ t('bundles.newBundle') }}</span>
        </Button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="bundlesStore.bundles.length === 0" class="rounded-2xl border-2 border-dashed border-muted-foreground/20 p-12 text-center">
      <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
        <Boxes class="h-8 w-8 text-muted-foreground" />
      </div>
      <p class="text-lg font-semibold text-muted-foreground">{{ t('bundles.noSaved') }}</p>
      <p class="mt-2 text-sm text-muted-foreground max-w-md mx-auto">
        {{ t('bundles.noSavedHint') }}
      </p>
      <Button class="mt-6" size="lg" @click="openCreateModal">
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
          'group relative overflow-hidden rounded-2xl border-2 bg-gradient-to-br transition-all duration-300 hover:shadow-xl hover:scale-[1.02]',
          getTunnelGradient(bundle.type),
          getTunnelBorder(bundle.type)
        ]"
      >
        <!-- Top accent line -->
        <div :class="['absolute top-0 left-0 right-0 h-1', bundle.type === 'http' ? 'bg-emerald-500' : bundle.type === 'tcp' ? 'bg-blue-500' : 'bg-purple-500']" />

        <!-- Connected indicator -->
        <div v-if="isBundleConnected(bundle)" class="absolute top-3 right-3">
          <div class="flex items-center gap-1.5 px-2 py-1 rounded-full bg-emerald-500/20 text-emerald-500 text-xs font-medium">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
            </span>
            {{ t('bundles.active') || 'Active' }}
          </div>
        </div>

        <div class="p-5">
          <!-- Header with Icon -->
          <div class="flex items-start gap-4 mb-5">
            <div :class="[
              'flex h-14 w-14 items-center justify-center rounded-2xl transition-transform group-hover:scale-110',
              bundle.type === 'http' ? 'bg-emerald-500/20' : bundle.type === 'tcp' ? 'bg-blue-500/20' : 'bg-purple-500/20'
            ]">
              <component :is="getTunnelIcon(bundle.type)" :class="['h-7 w-7', getTunnelAccent(bundle.type)]" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="text-lg font-bold truncate">{{ bundle.name }}</h3>
              <Badge :variant="getTunnelTypeBadge(bundle.type)" class="mt-1">
                {{ bundle.type.toUpperCase() }}
              </Badge>
            </div>
          </div>

          <!-- Port mapping visualization -->
          <div class="flex items-center gap-3 p-4 rounded-xl bg-background/60 border border-border/30 mb-4">
            <div class="flex-1 text-center">
              <p class="text-xs text-muted-foreground mb-1">{{ t('bundles.localPort') }}</p>
              <p class="font-mono text-lg font-bold">:{{ bundle.localPort }}</p>
            </div>
            <div class="flex flex-col items-center">
              <ArrowRight :class="['h-5 w-5', getTunnelAccent(bundle.type)]" />
            </div>
            <div class="flex-1 text-center">
              <p class="text-xs text-muted-foreground mb-1">{{ bundle.type === 'http' ? t('bundles.subdomain') : t('bundles.remotePort') }}</p>
              <p :class="['font-mono text-lg font-bold', getTunnelAccent(bundle.type)]">
                {{ bundle.type === 'http' ? (bundle.subdomain || 'auto') : (bundle.remotePort || 'auto') }}
              </p>
            </div>
          </div>

          <!-- Auto-connect badge -->
          <div v-if="bundle.autoConnect" class="flex items-center gap-2 mb-4 px-3 py-2 rounded-lg bg-amber-500/10 border border-amber-500/20">
            <Zap class="h-4 w-4 text-amber-500" />
            <span class="text-sm text-amber-600 dark:text-amber-400 font-medium">{{ t('bundles.autoConnectEnabled') || 'Auto-connect enabled' }}</span>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-2">
            <Button
              v-if="isBundleConnected(bundle)"
              variant="outline"
              class="flex-1"
              disabled
            >
              <Check class="mr-2 h-4 w-4 text-emerald-500" />
              {{ t('bundles.connected') }}
            </Button>
            <Button
              v-else
              :class="['flex-1 text-white', getTunnelBg(bundle.type)]"
              @click="connectBundle(bundle.id)"
            >
              <Play class="mr-2 h-4 w-4" />
              {{ t('bundles.connect') }}
            </Button>

            <Tooltip :content="t('common.edit')">
              <Button
                variant="outline"
                size="icon"
                class="h-10 w-10"
                @click="openEditModal(bundle)"
              >
                <Settings2 class="h-4 w-4" />
              </Button>
            </Tooltip>
            <Tooltip :content="t('common.delete')">
              <Button
                variant="outline"
                size="icon"
                class="h-10 w-10 hover:border-destructive hover:text-destructive hover:bg-destructive/10"
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
      <DialogContent>
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
              <Boxes class="h-4 w-4 text-primary" />
            </div>
            {{ editingBundle ? t('bundles.editBundle') : t('bundles.createBundle') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('bundles.bundleDescription') }}
          </DialogDescription>
        </DialogHeader>

        <form @submit.prevent="saveBundle" class="space-y-4">
          <div class="space-y-2">
            <Label>{{ t('bundles.name') }}</Label>
            <Input v-model="formData.name" :placeholder="t('bundles.namePlaceholder')" />
          </div>

          <div class="space-y-2">
            <Label>{{ t('bundles.type') }}</Label>
            <Select
              v-model="formData.type"
              :options="tunnelTypes"
            />
          </div>

          <div class="space-y-2">
            <Label>{{ t('bundles.localPort') }}</Label>
            <Input
              v-model.number="formData.localPort"
              type="number"
              placeholder="3000"
            />
          </div>

          <div v-if="formData.type === 'http'" class="space-y-2">
            <Label>{{ t('bundles.subdomain') }} <span class="text-muted-foreground text-xs">{{ t('dashboard.optional') }}</span></Label>
            <Input v-model="formData.subdomain" :placeholder="t('bundles.subdomainPlaceholder')" />
          </div>

          <div v-else class="space-y-2">
            <Label>{{ t('bundles.remotePort') }} <span class="text-muted-foreground text-xs">{{ t('dashboard.optional') }}</span></Label>
            <Input
              v-model.number="formData.remotePort"
              type="number"
              placeholder="0"
            />
            <p class="text-xs text-muted-foreground">{{ t('bundles.remotePortHint') }}</p>
          </div>

          <div class="flex items-center justify-between p-3 rounded-lg bg-amber-500/5 border border-amber-500/20">
            <div class="flex items-center gap-2">
              <Zap class="h-4 w-4 text-amber-500" />
              <Label class="cursor-pointer">{{ t('bundles.autoConnectOnStartup') }}</Label>
            </div>
            <Switch v-model="formData.autoConnect" />
          </div>

          <DialogFooter class="pt-4">
            <Button type="button" variant="outline" @click="showModal = false">
              {{ t('common.cancel') }}
            </Button>
            <Button type="submit" :disabled="!formData.name || !formData.localPort">
              {{ editingBundle ? t('common.save') : t('common.create') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-destructive">
            <Trash2 class="h-5 w-5" />
            {{ t('bundles.deleteBundle') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('bundles.confirmDelete') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showDeleteDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" @click="deleteBundle">
            {{ t('common.delete') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
