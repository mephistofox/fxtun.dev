<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useTunnelsStore } from '@/stores/tunnels'
import { useSyncStore } from '@/stores/sync'
import { Button, Tooltip } from '@/components/ui'
import StatusIndicator from '@/components/StatusIndicator.vue'
import {
  LayoutDashboard,
  Boxes,
  Globe,
  History,
  Settings,
  FileText,
  LogOut,
  Wifi,
  RefreshCw,
  Check,
  AlertCircle
} from 'lucide-vue-next'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const tunnelsStore = useTunnelsStore()
const syncStore = useSyncStore()

let stopPolling: (() => void) | null = null

onMounted(() => {
  syncStore.getStatus()
  stopPolling = syncStore.startPolling()
})

onUnmounted(() => {
  if (stopPolling) {
    stopPolling()
  }
})

const navItems = computed(() => [
  { name: 'dashboard', labelKey: 'nav.dashboard', icon: LayoutDashboard },
  { name: 'bundles', labelKey: 'nav.bundles', icon: Boxes },
  { name: 'domains', labelKey: 'nav.domains', icon: Globe },
  { name: 'history', labelKey: 'nav.history', icon: History },
  { name: 'settings', labelKey: 'nav.settings', icon: Settings },
  { name: 'logs', labelKey: 'nav.logs', icon: FileText },
])

const connectionStatus = computed(() => {
  switch (tunnelsStore.status) {
    case 'connected': return 'connected'
    case 'connecting': return 'connecting'
    default: return 'disconnected'
  }
})

const statusText = computed(() => {
  switch (tunnelsStore.status) {
    case 'connected': return t('status.connected')
    case 'connecting': return t('status.connecting')
    default: return t('status.disconnected')
  }
})

async function logout() {
  await authStore.logout()
  router.push('/auth')
}
</script>

<template>
  <div class="flex h-screen flex-col">
    <!-- Title bar (draggable) -->
    <div class="wails-drag flex h-10 items-center justify-between border-b bg-gradient-to-r from-muted/80 to-muted/50 px-4">
      <div class="flex items-center gap-2">
        <div class="flex h-6 w-6 items-center justify-center rounded-md bg-primary/10">
          <Wifi class="h-3.5 w-3.5 text-primary" />
        </div>
        <span class="text-sm font-semibold tracking-tight">fxTunnel</span>
      </div>
      <div class="flex items-center gap-3">
        <!-- Sync indicator -->
        <Tooltip :content="syncStore.lastError || (syncStore.isSyncing ? t('sync.syncing') : t('sync.synced'))" :delay-duration="300">
          <div class="flex items-center gap-1.5 rounded-full bg-background/50 px-2 py-1">
            <RefreshCw
              v-if="syncStore.isSyncing"
              class="h-3 w-3 animate-spin text-primary"
            />
            <AlertCircle
              v-else-if="syncStore.lastError"
              class="h-3 w-3 text-destructive"
            />
            <Check
              v-else
              class="h-3 w-3 text-green-500"
            />
          </div>
        </Tooltip>
        <!-- Connection status -->
        <div class="flex items-center gap-2 rounded-full bg-background/50 px-3 py-1">
          <StatusIndicator :status="connectionStatus" size="sm" />
          <span class="text-xs font-medium">{{ statusText }}</span>
        </div>
      </div>
    </div>

    <!-- Navigation -->
    <nav class="flex items-center justify-between border-b bg-background/50 px-4 py-2">
      <div class="flex gap-1">
        <Tooltip
          v-for="item in navItems"
          :key="item.name"
          :content="t(item.labelKey)"
          :delay-duration="500"
        >
          <Button
            :variant="route.name === item.name ? 'secondary' : 'ghost'"
            size="sm"
            :class="['transition-all duration-200', route.name === item.name && 'shadow-sm']"
            @click="router.push({ name: item.name })"
          >
            <component :is="item.icon" class="h-4 w-4 sm:mr-2" />
            <span class="hidden sm:inline">{{ t(item.labelKey) }}</span>
          </Button>
        </Tooltip>
      </div>
      <Tooltip :content="t('nav.logout')" :delay-duration="500">
        <Button variant="ghost" size="sm" @click="logout">
          <LogOut class="h-4 w-4 sm:mr-2" />
          <span class="hidden sm:inline">{{ t('nav.logout') }}</span>
        </Button>
      </Tooltip>
    </nav>

    <!-- Main content -->
    <main class="flex-1 overflow-auto p-4">
      <slot />
    </main>

    <!-- Status bar -->
    <footer class="flex h-8 items-center justify-between border-t bg-muted/30 px-4 text-xs text-muted-foreground">
      <div class="flex items-center gap-4">
        <span class="flex items-center gap-1.5">
          <span class="opacity-60">{{ t('status.server') }}:</span>
          <span class="font-medium">{{ authStore.serverAddress || t('status.noServer') }}</span>
        </span>
        <span v-if="tunnelsStore.activeTunnels.length" class="flex items-center gap-1.5">
          <span class="opacity-60">{{ t('status.activeTunnels') }}:</span>
          <span class="font-medium text-primary">{{ tunnelsStore.activeTunnels.length }}</span>
        </span>
      </div>
      <span class="opacity-60">v1.0.0</span>
    </footer>
  </div>
</template>
