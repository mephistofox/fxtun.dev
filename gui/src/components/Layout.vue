<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useTunnelsStore } from '@/stores/tunnels'
import { useSyncStore } from '@/stores/sync'
import { Tooltip } from '@/components/ui'
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
  AlertCircle,
  Zap,
  ChevronLeft,
  ChevronRight
} from 'lucide-vue-next'
import { ref } from 'vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const tunnelsStore = useTunnelsStore()
const syncStore = useSyncStore()

const sidebarCollapsed = ref(false)
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
  <div class="flex h-screen bg-background overflow-hidden">
    <!-- Animated grid background -->
    <div class="fixed inset-0 grid-pattern opacity-30 pointer-events-none" />

    <!-- Sidebar -->
    <aside
      :class="[
        'relative z-20 flex flex-col border-r border-border/50 bg-card/80 backdrop-blur-xl transition-all duration-300',
        sidebarCollapsed ? 'w-16' : 'w-56'
      ]"
    >
      <!-- Logo & Title bar (draggable) -->
      <div class="wails-drag flex h-14 items-center gap-3 border-b border-border/50 px-4">
        <div class="relative flex h-9 w-9 items-center justify-center">
          <!-- Animated glow ring -->
          <div class="absolute inset-0 rounded-xl bg-gradient-to-br from-primary to-accent opacity-20 blur-md animate-pulse" />
          <div class="relative flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-accent">
            <Zap class="h-5 w-5 text-primary-foreground" />
          </div>
        </div>
        <Transition name="fade">
          <div v-if="!sidebarCollapsed" class="flex flex-col">
            <span class="font-display font-bold text-sm tracking-tight">fxTunnel</span>
            <span class="text-[10px] text-muted-foreground">v1.5.0</span>
          </div>
        </Transition>
      </div>

      <!-- Connection Status -->
      <div class="p-3 border-b border-border/50">
        <div
          :class="[
            'flex items-center gap-3 rounded-xl p-3 transition-all duration-300',
            connectionStatus === 'connected'
              ? 'bg-success/10 border border-success/20'
              : connectionStatus === 'connecting'
                ? 'bg-warning/10 border border-warning/20'
                : 'bg-muted/50 border border-border/50'
          ]"
        >
          <StatusIndicator :status="connectionStatus" size="md" />
          <Transition name="fade">
            <div v-if="!sidebarCollapsed" class="flex-1 min-w-0">
              <p class="text-xs font-medium truncate">{{ statusText }}</p>
              <p class="text-[10px] text-muted-foreground truncate">
                {{ tunnelsStore.activeTunnels.length }} {{ t('status.activeTunnels').toLowerCase() }}
              </p>
            </div>
          </Transition>
        </div>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 overflow-y-auto p-3 space-y-1">
        <Tooltip
          v-for="item in navItems"
          :key="item.name"
          :content="sidebarCollapsed ? t(item.labelKey) : ''"
          :delay-duration="0"
          side="right"
        >
          <button
            @click="router.push({ name: item.name })"
            :class="[
              'group relative flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-200',
              route.name === item.name
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
            ]"
          >
            <!-- Active indicator -->
            <div
              v-if="route.name === item.name"
              class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 rounded-r-full bg-primary"
            />

            <component
              :is="item.icon"
              :class="[
                'h-5 w-5 transition-transform duration-200',
                route.name === item.name && 'scale-110'
              ]"
            />
            <Transition name="fade">
              <span v-if="!sidebarCollapsed">{{ t(item.labelKey) }}</span>
            </Transition>
          </button>
        </Tooltip>
      </nav>

      <!-- Bottom section -->
      <div class="p-3 space-y-2 border-t border-border/50">
        <!-- Sync Status -->
        <div
          :class="[
            'flex items-center gap-3 rounded-xl px-3 py-2 text-xs',
            syncStore.lastError ? 'bg-destructive/10' : 'bg-muted/30'
          ]"
        >
          <RefreshCw
            v-if="syncStore.isSyncing"
            class="h-4 w-4 animate-spin text-primary"
          />
          <AlertCircle
            v-else-if="syncStore.lastError"
            class="h-4 w-4 text-destructive"
          />
          <Check
            v-else
            class="h-4 w-4 text-success"
          />
          <Transition name="fade">
            <span v-if="!sidebarCollapsed" class="text-muted-foreground">
              {{ syncStore.isSyncing ? t('sync.syncing') : syncStore.lastError || t('sync.synced') }}
            </span>
          </Transition>
        </div>

        <!-- Logout -->
        <Tooltip :content="sidebarCollapsed ? t('nav.logout') : ''" :delay-duration="0" side="right">
          <button
            @click="logout"
            class="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-muted-foreground transition-all hover:bg-destructive/10 hover:text-destructive"
          >
            <LogOut class="h-5 w-5" />
            <Transition name="fade">
              <span v-if="!sidebarCollapsed">{{ t('nav.logout') }}</span>
            </Transition>
          </button>
        </Tooltip>

        <!-- Collapse toggle -->
        <button
          @click="sidebarCollapsed = !sidebarCollapsed"
          class="flex w-full items-center justify-center gap-2 rounded-xl py-2 text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-all"
        >
          <component :is="sidebarCollapsed ? ChevronRight : ChevronLeft" class="h-4 w-4" />
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <div class="relative flex-1 flex flex-col overflow-hidden">
      <!-- Top bar -->
      <header class="relative z-10 flex h-14 items-center justify-between border-b border-border/50 bg-card/50 backdrop-blur-sm px-6">
        <div class="flex items-center gap-3">
          <h1 class="font-display font-semibold text-lg">
            {{ t(navItems.find(i => i.name === route.name)?.labelKey || 'nav.dashboard') }}
          </h1>
        </div>

        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <Wifi class="h-3.5 w-3.5" />
          <span>{{ authStore.serverAddress || t('status.noServer') }}</span>
        </div>
      </header>

      <!-- Page content -->
      <main class="flex-1 overflow-auto p-6">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.grid-pattern {
  background-image:
    linear-gradient(hsl(var(--border) / 0.3) 1px, transparent 1px),
    linear-gradient(90deg, hsl(var(--border) / 0.3) 1px, transparent 1px);
  background-size: 40px 40px;
}
</style>
