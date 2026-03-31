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
  { value: 'http', label: 'HTTP' },
  { value: 'tcp', label: 'TCP' },
  { value: 'udp', label: 'UDP' },
])

function getTunnelIcon(type: TunnelType) {
  switch (type) {
    case 'http': return Globe
    case 'tcp': return Server
    case 'udp': return Radio
  }
}

function getTunnelAccent(type: TunnelType): string {
  switch (type) {
    case 'http': return 'text-type-http'
    case 'tcp': return 'text-type-tcp'
    case 'udp': return 'text-type-udp'
  }
}

function getTunnelBorder(type: TunnelType): string {
  switch (type) {
    case 'http': return 'border-type-http/30 hover:border-type-http/60'
    case 'tcp': return 'border-type-tcp/30 hover:border-type-tcp/60'
    case 'udp': return 'border-type-udp/30 hover:border-type-udp/60'
  }
}

function getTunnelBgClass(type: TunnelType): string {
  switch (type) {
    case 'http': return 'bg-type-http hover:bg-type-http/90'
    case 'tcp': return 'bg-type-tcp hover:bg-type-tcp/90'
    case 'udp': return 'bg-type-udp hover:bg-type-udp/90'
  }
}

function getTunnelBg(type: TunnelType): string {
  switch (type) {
    case 'http': return 'bg-type-http/20'
    case 'tcp': return 'bg-type-tcp/20'
    case 'udp': return 'bg-type-udp/20'
  }
}

onMounted(() => {
  bundlesStore.loadBundles()
})

function openCreateModal() {
  editingBundle.value = null
  formData.value = { name: '', type: 'http', localPort: 0, subdomain: '', remotePort: 0, autoConnect: false }
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
    await bundlesStore.updateBundle({ ...editingBundle.value, ...formData.value })
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
</script>

<template>
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="h-10 w-10 rounded-xl bg-gradient-to-br from-primary/20 to-accent/20 border border-primary/30 flex items-center justify-center">
          <Boxes class="h-5 w-5 text-primary" />
        </div>
        <div>
          <h1 class="font-display text-xl font-bold">{{ t('bundles.title') }}</h1>
          <p class="text-xs text-muted-foreground">{{ t('bundles.subtitle') }}</p>
        </div>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" size="sm" @click="importBundles">
          <Upload class="h-4 w-4 mr-1.5" />
          {{ t('bundles.import') }}
        </Button>
        <Button variant="outline" size="sm" @click="exportBundles">
          <Download class="h-4 w-4 mr-1.5" />
          {{ t('bundles.export') }}
        </Button>
        <Button size="sm" class="bg-gradient-to-r from-primary to-primary hover:to-accent" @click="openCreateModal">
          <Plus class="h-4 w-4 mr-1.5" />
          {{ t('bundles.newBundle') }}
        </Button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="bundlesStore.bundles.length === 0" class="rounded-xl border-2 border-dashed border-muted-foreground/20 p-10 text-center">
      <div class="mx-auto mb-4 h-14 w-14 rounded-xl bg-muted/50 flex items-center justify-center">
        <Boxes class="h-7 w-7 text-muted-foreground" />
      </div>
      <p class="font-semibold">{{ t('bundles.noSaved') }}</p>
      <p class="mt-1 text-sm text-muted-foreground">{{ t('bundles.noSavedHint') }}</p>
      <Button class="mt-4" @click="openCreateModal">
        <Plus class="mr-2 h-4 w-4" />
        {{ t('bundles.createBundle') }}
      </Button>
    </div>

    <!-- Bundles Grid -->
    <TransitionGroup v-else name="list" tag="div" class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="bundle in bundlesStore.bundles"
        :key="bundle.id"
        :class="[
          'group relative overflow-hidden rounded-xl border transition-all duration-200 hover:shadow-lg bg-gradient-to-br from-card to-card/50',
          getTunnelBorder(bundle.type)
        ]"
      >
        <!-- Connected indicator -->
        <div v-if="isBundleConnected(bundle)" class="absolute top-2 right-2 z-10">
          <div class="flex items-center gap-1 px-2 py-0.5 rounded-full bg-success/20 text-success text-[10px] font-medium">
            <span class="relative flex h-1.5 w-1.5">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75"></span>
              <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-success"></span>
            </span>
            {{ t('bundles.active') }}
          </div>
        </div>

        <div class="p-4">
          <!-- Header -->
          <div class="flex items-center gap-3 mb-3">
            <div :class="['flex h-10 w-10 items-center justify-center rounded-xl', getTunnelBg(bundle.type)]">
              <component :is="getTunnelIcon(bundle.type)" :class="['h-5 w-5', getTunnelAccent(bundle.type)]" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold truncate">{{ bundle.name }}</h3>
              <Badge :variant="bundle.type" class="text-[10px]">{{ bundle.type.toUpperCase() }}</Badge>
            </div>
          </div>

          <!-- Port mapping -->
          <div class="flex items-center gap-2 p-2.5 rounded-lg bg-background/60 border border-border/30 mb-3 text-sm">
            <div class="flex-1 text-center">
              <p class="text-[10px] text-muted-foreground uppercase">{{ t('bundles.localPort') }}</p>
              <p class="font-mono font-bold">:{{ bundle.localPort }}</p>
            </div>
            <ArrowRight :class="['h-4 w-4', getTunnelAccent(bundle.type)]" />
            <div class="flex-1 text-center">
              <p class="text-[10px] text-muted-foreground uppercase">{{ bundle.type === 'http' ? t('bundles.subdomain') : t('bundles.remotePort') }}</p>
              <p :class="['font-mono font-bold truncate', getTunnelAccent(bundle.type)]">
                {{ bundle.type === 'http' ? (bundle.subdomain || 'auto') : (bundle.remotePort || 'auto') }}
              </p>
            </div>
          </div>

          <!-- Auto-connect badge -->
          <div v-if="bundle.autoConnect" class="flex items-center gap-1.5 mb-3 px-2 py-1 rounded-lg bg-warning/10 border border-warning/20 text-xs">
            <Zap class="h-3 w-3 text-warning" />
            <span class="text-warning font-medium">{{ t('bundles.autoConnect') }}</span>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-2">
            <Button
              v-if="isBundleConnected(bundle)"
              variant="outline"
              size="sm"
              class="flex-1 h-8 border-success/30 text-success"
              disabled
            >
              <Check class="mr-1.5 h-3.5 w-3.5" />
              {{ t('bundles.connected') }}
            </Button>
            <Button
              v-else
              size="sm"
              :class="['flex-1 h-8 text-white', getTunnelBgClass(bundle.type)]"
              @click="connectBundle(bundle.id)"
            >
              <Play class="mr-1.5 h-3.5 w-3.5" />
              {{ t('bundles.connect') }}
            </Button>

            <Tooltip :content="t('common.edit')">
              <Button variant="outline" size="icon" class="h-8 w-8" @click="openEditModal(bundle)">
                <Settings2 class="h-3.5 w-3.5" />
              </Button>
            </Tooltip>
            <Tooltip :content="t('common.delete')">
              <Button variant="outline" size="icon" class="h-8 w-8 hover:border-destructive hover:text-destructive hover:bg-destructive/10" @click="confirmDeleteBundle(bundle.id)">
                <Trash2 class="h-3.5 w-3.5" />
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
            <Boxes class="h-5 w-5 text-primary" />
            {{ editingBundle ? t('bundles.editBundle') : t('bundles.createBundle') }}
          </DialogTitle>
          <DialogDescription>{{ t('bundles.bundleDescription') }}</DialogDescription>
        </DialogHeader>

        <form @submit.prevent="saveBundle" class="space-y-4">
          <div class="space-y-1.5">
            <Label>{{ t('bundles.name') }}</Label>
            <Input v-model="formData.name" :placeholder="t('bundles.namePlaceholder')" />
          </div>

          <div class="space-y-1.5">
            <Label>{{ t('bundles.type') }}</Label>
            <Select v-model="formData.type" :options="tunnelTypes" />
          </div>

          <div class="space-y-1.5">
            <Label>{{ t('bundles.localPort') }}</Label>
            <Input v-model.number="formData.localPort" type="number" placeholder="3000" class="font-mono" />
          </div>

          <div v-if="formData.type === 'http'" class="space-y-1.5">
            <Label>{{ t('bundles.subdomain') }} <span class="text-muted-foreground text-xs">({{ t('dashboard.optional') }})</span></Label>
            <Input v-model="formData.subdomain" :placeholder="t('bundles.subdomainPlaceholder')" class="font-mono" />
          </div>

          <div v-else class="space-y-1.5">
            <Label>{{ t('bundles.remotePort') }} <span class="text-muted-foreground text-xs">({{ t('dashboard.optional') }})</span></Label>
            <Input v-model.number="formData.remotePort" type="number" placeholder="0" class="font-mono" />
          </div>

          <div class="flex items-center justify-between p-3 rounded-lg bg-warning/5 border border-warning/20">
            <div class="flex items-center gap-2">
              <Zap class="h-4 w-4 text-warning" />
              <Label class="cursor-pointer">{{ t('bundles.autoConnectOnStartup') }}</Label>
            </div>
            <Switch v-model="formData.autoConnect" />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" @click="showModal = false">{{ t('common.cancel') }}</Button>
            <Button type="submit" :disabled="!formData.name || !formData.localPort">
              {{ editingBundle ? t('common.save') : t('common.create') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Delete Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-destructive">
            <Trash2 class="h-5 w-5" />
            {{ t('bundles.deleteBundle') }}
          </DialogTitle>
          <DialogDescription>{{ t('bundles.confirmDelete') }}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showDeleteDialog = false">{{ t('common.cancel') }}</Button>
          <Button variant="destructive" @click="deleteBundle">{{ t('common.delete') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
