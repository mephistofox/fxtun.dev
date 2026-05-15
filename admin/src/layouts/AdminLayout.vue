<template>
  <div class="flex h-screen bg-background">
    <!-- Sidebar -->
    <aside class="w-64 flex flex-col border-r border-border bg-card/80 backdrop-blur-xl">
      <!-- Logo -->
      <div class="p-4 border-b border-border">
        <h1 class="font-display font-bold text-primary text-lg">fxTunnel</h1>
        <p class="text-xs text-muted-foreground">Админ-панель</p>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 p-2 space-y-0.5 overflow-auto">
        <router-link
          v-for="item in menuItems"
          :key="item.route"
          :to="{ name: item.route }"
          :class="[
            'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors duration-150',
            isActive(item.route)
              ? 'bg-primary/10 text-primary border-l-2 border-primary'
              : 'text-muted-foreground hover:bg-surface-elevated/50 hover:text-foreground border-l-2 border-transparent',
          ]"
        >
          <component :is="item.icon" class="w-5 h-5 flex-shrink-0" />
          {{ item.label }}
        </router-link>
      </nav>

      <!-- User info + Logout -->
      <div class="p-3 border-t border-border space-y-2">
        <div v-if="authStore.user" class="flex items-center gap-3 px-3 py-1">
          <div class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
            <span class="text-xs font-display font-bold text-primary">
              {{ (authStore.user.display_name || authStore.user.email || authStore.user.phone || '?').slice(0, 2).toUpperCase() }}
            </span>
          </div>
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium text-foreground truncate">
              {{ authStore.user.display_name || authStore.user.email || authStore.user.phone }}
            </p>
            <p v-if="authStore.user.email" class="text-xs text-muted-foreground truncate">
              {{ authStore.user.email }}
            </p>
          </div>
        </div>
        <button
          class="flex items-center gap-3 w-full px-3 py-2 rounded-lg text-sm font-medium text-muted-foreground hover:bg-surface-elevated/50 hover:text-foreground transition-colors duration-150"
          @click="handleLogout"
        >
          <LogOut class="w-5 h-5 flex-shrink-0" />
          Выйти
        </button>
      </div>
    </aside>

    <!-- Content -->
    <main class="flex-1 overflow-auto">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import {
  LayoutDashboard,
  Users,
  Network,
  Server,
  CreditCard,
  Receipt,
  Wallet,
  Globe,
  ShieldCheck,
  FileText,
  Settings,
  LogOut,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const authStore = useAuthStore()

const menuItems = [
  { label: 'Панель управления', route: 'dashboard', icon: LayoutDashboard },
  { label: 'Пользователи', route: 'users', icon: Users },
  { label: 'Тоннели', route: 'tunnels', icon: Network },
  { label: 'Ноды', route: 'nodes', icon: Server },
  { label: 'Тарифы', route: 'plans', icon: CreditCard },
  { label: 'Подписки', route: 'subscriptions', icon: Receipt },
  { label: 'Платежи', route: 'payments', icon: Wallet },
  { label: 'Домены', route: 'domains', icon: Globe },
  { label: 'Сертификаты', route: 'certificates', icon: ShieldCheck },
  { label: 'Журнал', route: 'audit', icon: FileText },
  { label: 'Настройки', route: 'settings', icon: Settings },
]

function isActive(routeName: string): boolean {
  const currentName = route.name as string
  if (currentName === 'user-detail' && routeName === 'users') return true
  return currentName === routeName
}

function handleLogout() {
  authStore.logout()
}
</script>
