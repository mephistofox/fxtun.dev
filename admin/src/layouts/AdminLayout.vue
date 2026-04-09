<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="240"
      :collapsed="appStore.sidebarCollapsed"
      show-trigger
      @collapse="appStore.sidebarCollapsed = true"
      @expand="appStore.sidebarCollapsed = false"
      :native-scrollbar="false"
      style="height: 100vh"
    >
      <div
        style="
          display: flex;
          align-items: center;
          justify-content: center;
          height: 48px;
          padding: 0 16px;
          font-size: 18px;
          font-weight: 700;
          color: rgba(255, 255, 255, 0.82);
          white-space: nowrap;
          overflow: hidden;
        "
      >
        <span v-if="!appStore.sidebarCollapsed">fxTunnel Admin</span>
        <span v-else>fx</span>
      </div>
      <n-menu
        :collapsed="appStore.sidebarCollapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenuUpdate"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-header
        bordered
        style="
          height: 48px;
          padding: 0 24px;
          display: flex;
          align-items: center;
          justify-content: space-between;
        "
      >
        <n-text strong style="font-size: 16px">{{ pageTitle }}</n-text>
        <n-dropdown
          :options="userDropdownOptions"
          @select="handleUserDropdown"
        >
          <n-button quaternary>
            <template #icon>
              <n-icon><PersonOutline /></n-icon>
            </template>
            {{ authStore.user?.display_name || authStore.user?.phone || 'Admin' }}
          </n-button>
        </n-dropdown>
      </n-layout-header>
      <n-layout-content
        content-style="padding: 24px;"
        :native-scrollbar="false"
        style="height: calc(100vh - 48px)"
      >
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NIcon } from 'naive-ui'
import type { MenuOption, DropdownOption } from 'naive-ui'
import {
  HomeOutline,
  PeopleOutline,
  GitNetworkOutline,
  ServerOutline,
  PricetagsOutline,
  CardOutline,
  WalletOutline,
  GlobeOutline,
  ListOutline,
  SettingsOutline,
  PersonOutline,
  LogOutOutline,
} from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import type { Component } from 'vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  {
    label: 'Dashboard',
    key: 'dashboard',
    icon: renderIcon(HomeOutline),
  },
  {
    label: 'Users',
    key: 'users',
    icon: renderIcon(PeopleOutline),
  },
  {
    label: 'Tunnels',
    key: 'tunnels',
    icon: renderIcon(GitNetworkOutline),
  },
  {
    label: 'Nodes',
    key: 'nodes',
    icon: renderIcon(ServerOutline),
  },
  {
    label: 'Plans',
    key: 'plans',
    icon: renderIcon(PricetagsOutline),
  },
  {
    label: 'Subscriptions',
    key: 'subscriptions',
    icon: renderIcon(CardOutline),
  },
  {
    label: 'Payments',
    key: 'payments',
    icon: renderIcon(WalletOutline),
  },
  {
    label: 'Domains',
    key: 'domains',
    icon: renderIcon(GlobeOutline),
  },
  {
    label: 'Audit Log',
    key: 'audit',
    icon: renderIcon(ListOutline),
  },
  {
    label: 'Settings',
    key: 'settings',
    icon: renderIcon(SettingsOutline),
  },
]

const pageTitles: Record<string, string> = {
  dashboard: 'Dashboard',
  users: 'Users',
  'user-detail': 'User Detail',
  tunnels: 'Tunnels',
  nodes: 'Edge Nodes',
  plans: 'Plans',
  subscriptions: 'Subscriptions',
  payments: 'Payments',
  domains: 'Custom Domains',
  audit: 'Audit Log',
  settings: 'Settings',
}

const activeKey = computed(() => {
  const name = route.name as string
  // For user-detail, highlight "users" menu item
  if (name === 'user-detail') return 'users'
  return name || 'dashboard'
})

const pageTitle = computed(() => {
  const name = route.name as string
  return pageTitles[name] || 'Admin'
})

function handleMenuUpdate(key: string) {
  router.push({ name: key })
}

const userDropdownOptions: DropdownOption[] = [
  {
    label: 'Logout',
    key: 'logout',
    icon: renderIcon(LogOutOutline),
  },
]

function handleUserDropdown(key: string) {
  if (key === 'logout') {
    authStore.logout()
  }
}
</script>
