<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { setLocale, getLocale } from '@/i18n'
import Button from '@/components/ui/Button.vue'

const authStore = useAuthStore()
const themeStore = useThemeStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const appVersion = ref('')

onMounted(async () => {
  try {
    const res = await fetch('/health')
    const data = await res.json()
    appVersion.value = data.version || ''
  } catch {
    // ignore
  }
})

const navigation = [
  { key: 'dashboard', path: '/', icon: 'layout-dashboard' },
  { key: 'domains', path: '/domains', icon: 'globe' },
  { key: 'tokens', path: '/tokens', icon: 'key' },
  { key: 'downloads', path: '/downloads', icon: 'download' },
  { key: 'profile', path: '/profile', icon: 'user' },
]

const adminNavigation = [
  { key: 'admin', path: '/admin', icon: 'shield' },
  { key: 'adminUsers', path: '/admin/users', icon: 'users' },
  { key: 'adminInvites', path: '/admin/invites', icon: 'ticket' },
  { key: 'adminTunnels', path: '/admin/tunnels', icon: 'network' },
  { key: 'adminCustomDomains', path: '/admin/custom-domains', icon: 'globe' },
  { key: 'adminPlans', path: '/admin/plans', icon: 'credit-card' },
  { key: 'adminSubscriptions', path: '/admin/subscriptions', icon: 'calendar' },
  { key: 'adminAudit', path: '/admin/audit', icon: 'file-text' },
]

const adminMenuOpen = ref(false)
const adminMenuRef = ref<HTMLDivElement | null>(null)

function toggleAdminMenu() {
  adminMenuOpen.value = !adminMenuOpen.value
}

function closeAdminMenu() {
  adminMenuOpen.value = false
}

function navigateAdmin(path: string) {
  router.push(path)
  closeAdminMenu()
}

function handleClickOutside(event: MouseEvent) {
  if (adminMenuRef.value && !adminMenuRef.value.contains(event.target as Node)) {
    closeAdminMenu()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

function toggleLocale() {
  const current = getLocale()
  setLocale(current === 'en' ? 'ru' : 'en')
}

function cycleTheme() {
  const modes: ThemeMode[] = ['light', 'dark', 'system']
  const currentIndex = modes.indexOf(themeStore.mode)
  const nextIndex = (currentIndex + 1) % modes.length
  themeStore.setMode(modes[nextIndex])
}
</script>

<template>
  <div class="min-h-screen bg-background flex flex-col">
    <!-- Header -->
    <header class="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-lg">
      <div class="container mx-auto px-4 h-16 flex items-center justify-between">
        <div class="flex items-center space-x-8">
          <!-- Logo -->
          <RouterLink to="/" class="flex items-center gap-2 group">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 group-hover:bg-primary/20 transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M5 12.55a11 11 0 0 1 14.08 0" />
                <path d="M1.42 9a16 16 0 0 1 21.16 0" />
                <path d="M8.53 16.11a6 6 0 0 1 6.95 0" />
                <line x1="12" y1="20" x2="12.01" y2="20" />
              </svg>
            </div>
            <span class="text-xl font-bold text-primary">fxTunnel</span>
          </RouterLink>

          <!-- Desktop Navigation -->
          <nav class="hidden md:flex items-center space-x-1">
            <RouterLink
              v-for="item in navigation"
              :key="item.path"
              :to="item.path"
              :class="[
                'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200',
                route.path === item.path
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted/50',
              ]"
            >
              <!-- Dashboard icon -->
              <svg v-if="item.icon === 'layout-dashboard'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="7" height="9" />
                <rect x="14" y="3" width="7" height="5" />
                <rect x="14" y="12" width="7" height="9" />
                <rect x="3" y="16" width="7" height="5" />
              </svg>
              <!-- Globe icon -->
              <svg v-else-if="item.icon === 'globe'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="2" y1="12" x2="22" y2="12" />
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
              </svg>
              <!-- Key icon -->
              <svg v-else-if="item.icon === 'key'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4" />
              </svg>
              <!-- Download icon -->
              <svg v-else-if="item.icon === 'download'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              <!-- User icon -->
              <svg v-else-if="item.icon === 'user'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
                <circle cx="12" cy="7" r="4" />
              </svg>
              <span class="hidden lg:inline">{{ t(`nav.${item.key}`) }}</span>
            </RouterLink>

            <!-- Admin Dropdown Menu -->
            <template v-if="authStore.isAdmin">
              <div class="w-px h-6 bg-border mx-2"></div>
              <div ref="adminMenuRef" class="relative">
                <button
                  @click.stop="toggleAdminMenu"
                  :class="[
                    'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200',
                    route.path.startsWith('/admin')
                      ? 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300'
                      : 'text-muted-foreground hover:text-foreground hover:bg-muted/50',
                  ]"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                  </svg>
                  <span class="hidden lg:inline">{{ t('nav.admin') }}</span>
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3 transition-transform" :class="{ 'rotate-180': adminMenuOpen }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="6 9 12 15 18 9" />
                  </svg>
                </button>

                <!-- Dropdown Menu -->
                <Transition
                  enter-active-class="transition ease-out duration-100"
                  enter-from-class="transform opacity-0 scale-95"
                  enter-to-class="transform opacity-100 scale-100"
                  leave-active-class="transition ease-in duration-75"
                  leave-from-class="transform opacity-100 scale-100"
                  leave-to-class="transform opacity-0 scale-95"
                >
                  <div
                    v-if="adminMenuOpen"
                    class="absolute left-0 top-full mt-1 w-48 rounded-lg border bg-background shadow-lg z-50 py-1"
                  >
                    <button
                      v-for="item in adminNavigation"
                      :key="item.path"
                      @click="navigateAdmin(item.path)"
                      :class="[
                        'w-full flex items-center gap-3 px-4 py-2 text-sm transition-colors',
                        route.path === item.path
                          ? 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300'
                          : 'text-foreground hover:bg-muted',
                      ]"
                    >
                      <!-- Shield icon -->
                      <svg v-if="item.icon === 'shield'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                      </svg>
                      <!-- Users icon -->
                      <svg v-else-if="item.icon === 'users'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                        <circle cx="9" cy="7" r="4" />
                        <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                        <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                      </svg>
                      <!-- Ticket icon -->
                      <svg v-else-if="item.icon === 'ticket'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z" />
                        <path d="M13 5v2" />
                        <path d="M13 17v2" />
                        <path d="M13 11v2" />
                      </svg>
                      <!-- Network icon -->
                      <svg v-else-if="item.icon === 'network'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="16" y="16" width="6" height="6" rx="1" />
                        <rect x="2" y="16" width="6" height="6" rx="1" />
                        <rect x="9" y="2" width="6" height="6" rx="1" />
                        <path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3" />
                        <path d="M12 12V8" />
                      </svg>
                      <!-- Globe icon (custom domains) -->
                      <svg v-else-if="item.icon === 'globe'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10" />
                        <line x1="2" y1="12" x2="22" y2="12" />
                        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
                      </svg>
                      <!-- Credit-card icon -->
                      <svg v-else-if="item.icon === 'credit-card'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="1" y="4" width="22" height="16" rx="2" ry="2" />
                        <line x1="1" y1="10" x2="23" y2="10" />
                      </svg>
                      <!-- Calendar icon -->
                      <svg v-else-if="item.icon === 'calendar'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                        <line x1="16" y1="2" x2="16" y2="6" />
                        <line x1="8" y1="2" x2="8" y2="6" />
                        <line x1="3" y1="10" x2="21" y2="10" />
                      </svg>
                      <!-- File-text icon -->
                      <svg v-else-if="item.icon === 'file-text'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                        <polyline points="14 2 14 8 20 8" />
                        <line x1="16" y1="13" x2="8" y2="13" />
                        <line x1="16" y1="17" x2="8" y2="17" />
                        <polyline points="10 9 9 9 8 9" />
                      </svg>
                      {{ t(`nav.${item.key}`) }}
                    </button>
                  </div>
                </Transition>
              </div>
            </template>
          </nav>
        </div>

        <div class="flex items-center space-x-2">
          <!-- Theme Switcher -->
          <button
            @click="cycleTheme"
            class="p-2 rounded-lg hover:bg-muted transition-colors"
            :title="t(`theme.${themeStore.mode}`)"
          >
            <!-- Sun icon for light mode -->
            <svg
              v-if="themeStore.mode === 'light'"
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <circle cx="12" cy="12" r="5" />
              <line x1="12" y1="1" x2="12" y2="3" />
              <line x1="12" y1="21" x2="12" y2="23" />
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
              <line x1="1" y1="12" x2="3" y2="12" />
              <line x1="21" y1="12" x2="23" y2="12" />
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
            </svg>
            <!-- Moon icon for dark mode -->
            <svg
              v-else-if="themeStore.mode === 'dark'"
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
            <!-- Monitor icon for system mode -->
            <svg
              v-else
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
              <line x1="8" y1="21" x2="16" y2="21" />
              <line x1="12" y1="17" x2="12" y2="21" />
            </svg>
          </button>

          <!-- Language Switcher -->
          <button
            @click="toggleLocale"
            class="px-3 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors"
          >
            {{ getLocale() === 'en' ? 'RU' : 'EN' }}
          </button>

          <!-- User Info & Logout -->
          <div class="hidden sm:flex items-center gap-3 pl-3 border-l border-border">
            <span class="text-sm text-muted-foreground">
              {{ authStore.user?.display_name || authStore.user?.phone }}
            </span>
            <Button variant="outline" size="sm" @click="authStore.logout">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 sm:mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                <polyline points="16 17 21 12 16 7" />
                <line x1="21" y1="12" x2="9" y2="12" />
              </svg>
              <span class="hidden sm:inline">{{ t('common.logout') }}</span>
            </Button>
          </div>
        </div>
      </div>
    </header>

    <!-- Mobile navigation -->
    <nav class="md:hidden border-b bg-background/50 backdrop-blur-sm p-2 sticky top-16 z-40">
      <div class="flex overflow-x-auto space-x-1 pb-1">
        <RouterLink
          v-for="item in navigation"
          :key="item.path"
          :to="item.path"
          :class="[
            'flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg whitespace-nowrap transition-all duration-200',
            route.path === item.path
              ? 'bg-primary text-primary-foreground shadow-md'
              : 'text-muted-foreground hover:bg-muted',
          ]"
        >
          <!-- Dashboard icon -->
          <svg v-if="item.icon === 'layout-dashboard'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="9" />
            <rect x="14" y="3" width="7" height="5" />
            <rect x="14" y="12" width="7" height="9" />
            <rect x="3" y="16" width="7" height="5" />
          </svg>
          <!-- Globe icon -->
          <svg v-else-if="item.icon === 'globe'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="2" y1="12" x2="22" y2="12" />
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
          </svg>
          <!-- Key icon -->
          <svg v-else-if="item.icon === 'key'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4" />
          </svg>
          <!-- Download icon -->
          <svg v-else-if="item.icon === 'download'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          <!-- User icon -->
          <svg v-else-if="item.icon === 'user'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
          </svg>
          {{ t(`nav.${item.key}`) }}
        </RouterLink>

        <!-- Admin mobile links -->
        <template v-if="authStore.isAdmin">
          <div class="w-px h-8 bg-border mx-1 self-center"></div>
          <RouterLink
            to="/admin"
            :class="[
              'flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg whitespace-nowrap transition-all duration-200',
              route.path.startsWith('/admin')
                ? 'bg-purple-600 text-white shadow-md'
                : 'text-muted-foreground hover:bg-muted',
            ]"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
            {{ t('nav.admin') }}
          </RouterLink>
        </template>
      </div>
    </nav>

    <!-- Main content -->
    <main class="container mx-auto px-4 py-6 flex-1">
      <slot />
    </main>

    <!-- Footer -->
    <footer class="border-t bg-muted/30 mt-auto">
      <div class="container mx-auto px-4 py-4 flex items-center justify-between text-sm text-muted-foreground">
        <span>fxTunnel</span>
        <a
          v-if="appVersion"
          :href="`https://github.com/mephistofox/fxtunnel/releases/tag/${appVersion}`"
          target="_blank"
          rel="noopener"
          class="hover:text-foreground transition-colors"
        >
          {{ appVersion }}
        </a>
      </div>
    </footer>
  </div>
</template>
