<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import * as AppService from '@/wailsjs/wailsjs/go/gui/App'
import { useSettingsStore, type Theme, type Locale } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useHistoryStore } from '@/stores/history'
import { toast } from '@/composables/useToast'
import {
  Button, Input, Label, Switch, Separator,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter
} from '@/components/ui'
import {
  Moon, Sun, Monitor, Trash2, Palette, Globe, Server, Bell, Shield, Languages, Settings, Zap
} from 'lucide-vue-next'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const historyStore = useHistoryStore()

const serverAddress = ref('')
const platform = typeof window !== 'undefined' ? window.navigator.platform : 'Unknown'
const appVersion = ref('...')
const buildDate = ref('...')

// Dialogs
const showClearCredentialsDialog = ref(false)
const showClearHistoryDialog = ref(false)

// Sync serverAddress with store when it loads
onMounted(async () => {
  serverAddress.value = settingsStore.serverAddress || ''
  try {
    appVersion.value = await AppService.GetVersion()
    buildDate.value = await (AppService as any).GetBuildDate()
  } catch {
    // GetBuildDate may not exist in bindings yet
  }
})

// Watch for store changes
watch(() => settingsStore.serverAddress, (newValue) => {
  if (newValue !== serverAddress.value) {
    serverAddress.value = newValue
  }
})

const themes = computed(() => [
  { value: 'light' as Theme, labelKey: 'settings.themeLight', icon: Sun },
  { value: 'dark' as Theme, labelKey: 'settings.themeDark', icon: Moon },
  { value: 'system' as Theme, labelKey: 'settings.themeSystem', icon: Monitor },
])

const languages = computed(() => [
  { value: 'en' as Locale, label: 'English', flag: 'EN' },
  { value: 'ru' as Locale, label: 'Русский', flag: 'RU' },
])

function selectTheme(theme: Theme) {
  settingsStore.saveTheme(theme)
  toast({ title: t('settings.saved'), variant: 'success' })
}

function selectLanguage(locale: Locale) {
  settingsStore.saveLocale(locale)
  toast({ title: t('settings.saved'), variant: 'success' })
}

function saveServerAddress() {
  settingsStore.saveServerAddress(serverAddress.value)
  toast({ title: t('settings.saved'), variant: 'success' })
}

async function clearCredentials() {
  await authStore.logout()
  showClearCredentialsDialog.value = false
  toast({ title: t('settings.credentialsCleared'), variant: 'success' })
}

async function clearHistory() {
  await historyStore.clearHistory()
  showClearHistoryDialog.value = false
  toast({ title: t('settings.historyCleared'), variant: 'success' })
}
</script>

<template>
  <div class="mx-auto max-w-2xl space-y-6">
    <!-- Header -->
    <div class="flex items-center gap-4 mb-8">
      <div class="relative">
        <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary to-accent opacity-20 blur-lg" />
        <div class="relative flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/20 to-accent/20 border border-primary/30">
          <Settings class="h-7 w-7 text-primary" />
        </div>
      </div>
      <div>
        <h1 class="font-display text-2xl font-bold tracking-tight">{{ t('settings.title') }}</h1>
        <p class="text-sm text-muted-foreground">Configure your preferences</p>
      </div>
    </div>

    <!-- Appearance -->
    <div class="cyber-card rounded-2xl overflow-hidden">
      <div class="p-5 border-b border-border/50">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/10 border border-accent/20">
            <Palette class="h-5 w-5 text-accent" />
          </div>
          <div>
            <h2 class="font-display font-semibold">{{ t('settings.appearance') }}</h2>
            <p class="text-xs text-muted-foreground">Customize how fxTunnel looks</p>
          </div>
        </div>
      </div>
      <div class="p-5 space-y-5">
        <!-- Theme -->
        <div>
          <Label class="mb-3 block text-xs uppercase tracking-wider text-muted-foreground">{{ t('settings.theme') }}</Label>
          <div class="flex flex-wrap gap-2">
            <Button
              v-for="theme in themes"
              :key="theme.value"
              :variant="settingsStore.theme === theme.value ? 'default' : 'outline'"
              :class="[
                'transition-all duration-300',
                settingsStore.theme === theme.value
                  ? 'bg-gradient-to-r from-primary to-primary shadow-lg shadow-primary/25'
                  : 'border-border/50 hover:border-primary/50 hover:bg-primary/5'
              ]"
              size="sm"
              @click="selectTheme(theme.value)"
            >
              <component :is="theme.icon" class="mr-2 h-4 w-4" />
              {{ t(theme.labelKey) }}
            </Button>
          </div>
        </div>

        <Separator class="bg-border/30" />

        <!-- Language -->
        <div>
          <Label class="mb-3 flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
            <Languages class="h-3.5 w-3.5" />
            {{ t('settings.language') }}
          </Label>
          <div class="flex flex-wrap gap-2">
            <Button
              v-for="lang in languages"
              :key="lang.value"
              :variant="settingsStore.locale === lang.value ? 'default' : 'outline'"
              :class="[
                'transition-all duration-300',
                settingsStore.locale === lang.value
                  ? 'bg-gradient-to-r from-primary to-primary shadow-lg shadow-primary/25'
                  : 'border-border/50 hover:border-primary/50 hover:bg-primary/5'
              ]"
              size="sm"
              @click="selectLanguage(lang.value)"
            >
              <span class="mr-2 text-xs font-bold opacity-60">{{ lang.flag }}</span>
              {{ lang.label }}
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- Connection -->
    <div class="cyber-card rounded-2xl overflow-hidden">
      <div class="p-5 border-b border-border/50">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-type-http/10 border border-type-http/20">
            <Globe class="h-5 w-5 text-type-http" />
          </div>
          <div>
            <h2 class="font-display font-semibold">{{ t('settings.connection') }}</h2>
            <p class="text-xs text-muted-foreground">Server connection settings</p>
          </div>
        </div>
      </div>
      <div class="p-5">
        <div class="space-y-3">
          <Label class="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
            <Server class="h-3.5 w-3.5" />
            {{ t('settings.defaultServer') }}
          </Label>
          <p class="text-xs text-muted-foreground">{{ t('settings.defaultServerHint') }}</p>
          <div class="flex gap-2">
            <Input
              v-model="serverAddress"
              :placeholder="t('settings.defaultServerPlaceholder')"
              class="flex-1 bg-muted/30 border-border/50 font-mono"
            />
            <Button
              @click="saveServerAddress"
              class="bg-gradient-to-r from-primary to-primary hover:to-accent shadow-lg shadow-primary/25 transition-all duration-300"
            >
              {{ t('common.save') }}
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- System -->
    <div class="cyber-card rounded-2xl overflow-hidden">
      <div class="p-5 border-b border-border/50">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-warning/10 border border-warning/20">
            <Bell class="h-5 w-5 text-warning" />
          </div>
          <div>
            <h2 class="font-display font-semibold">{{ t('settings.system') }}</h2>
            <p class="text-xs text-muted-foreground">System behavior settings</p>
          </div>
        </div>
      </div>
      <div class="p-5 space-y-4">
        <div class="flex items-center justify-between p-4 rounded-xl bg-muted/30 border border-border/30 transition-all hover:border-primary/30">
          <div class="space-y-0.5">
            <Label class="font-medium">{{ t('settings.minimizeToTray') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.minimizeToTrayHint') }}
            </p>
          </div>
          <Switch
            :model-value="settingsStore.minimizeToTray"
            @update:model-value="settingsStore.saveMinimizeToTray($event)"
          />
        </div>

        <div class="flex items-center justify-between p-4 rounded-xl bg-muted/30 border border-border/30 transition-all hover:border-primary/30">
          <div class="space-y-0.5">
            <Label class="font-medium">{{ t('settings.notifications') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.notificationsHint') }}
            </p>
          </div>
          <Switch
            :model-value="settingsStore.notifications"
            @update:model-value="settingsStore.saveNotifications($event)"
          />
        </div>
      </div>
    </div>

    <!-- Data Management -->
    <div class="cyber-card rounded-2xl overflow-hidden">
      <div class="p-5 border-b border-border/50">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-destructive/10 border border-destructive/20">
            <Shield class="h-5 w-5 text-destructive" />
          </div>
          <div>
            <h2 class="font-display font-semibold">{{ t('settings.data') }}</h2>
            <p class="text-xs text-muted-foreground">Manage your data and credentials</p>
          </div>
        </div>
      </div>
      <div class="p-5 space-y-4">
        <div class="flex items-center justify-between p-4 rounded-xl bg-destructive/5 border border-destructive/20 transition-all hover:border-destructive/40">
          <div class="space-y-0.5">
            <Label class="font-medium">{{ t('settings.clearCredentials') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.clearCredentialsHint') }}
            </p>
          </div>
          <Button variant="destructive" size="sm" class="shadow-lg shadow-destructive/25" @click="showClearCredentialsDialog = true">
            <Trash2 class="mr-2 h-4 w-4" />
            {{ t('common.clear') }}
          </Button>
        </div>

        <div class="flex items-center justify-between p-4 rounded-xl bg-destructive/5 border border-destructive/20 transition-all hover:border-destructive/40">
          <div class="space-y-0.5">
            <Label class="font-medium">{{ t('settings.clearHistory') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.clearHistoryHint') }}
            </p>
          </div>
          <Button variant="destructive" size="sm" class="shadow-lg shadow-destructive/25" @click="showClearHistoryDialog = true">
            <Trash2 class="mr-2 h-4 w-4" />
            {{ t('common.clear') }}
          </Button>
        </div>
      </div>
    </div>

    <!-- About -->
    <div class="cyber-card rounded-2xl overflow-hidden">
      <div class="p-5 border-b border-border/50">
        <div class="flex items-center gap-3">
          <div class="relative">
            <div class="absolute inset-0 rounded-xl bg-gradient-to-br from-primary to-accent opacity-30 blur-md" />
            <div class="relative flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-accent">
              <Zap class="h-5 w-5 text-primary-foreground" />
            </div>
          </div>
          <div>
            <h2 class="font-display font-semibold">{{ t('settings.about') }}</h2>
            <p class="text-xs text-muted-foreground">Application information</p>
          </div>
        </div>
      </div>
      <div class="p-5">
        <div class="space-y-3">
          <div class="flex justify-between items-center p-3 rounded-lg bg-muted/30">
            <span class="text-sm text-muted-foreground">{{ t('settings.version') }}</span>
            <span class="font-display font-semibold gradient-text">{{ appVersion }}</span>
          </div>
          <div class="flex justify-between items-center p-3 rounded-lg bg-muted/30">
            <span class="text-sm text-muted-foreground">{{ t('settings.build') }}</span>
            <span class="font-mono text-xs text-muted-foreground">{{ buildDate }}</span>
          </div>
          <div class="flex justify-between items-center p-3 rounded-lg bg-muted/30">
            <span class="text-sm text-muted-foreground">{{ t('settings.platform') }}</span>
            <span class="font-mono text-xs text-muted-foreground">{{ platform }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Clear Credentials Dialog -->
    <Dialog v-model:open="showClearCredentialsDialog">
      <DialogContent class="border-destructive/30 bg-card/95 backdrop-blur-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-3 font-display">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-destructive/20 border border-destructive/30">
              <Shield class="h-5 w-5 text-destructive" />
            </div>
            {{ t('settings.clearCredentials') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('settings.confirmClearCredentials') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" class="border-border/50" @click="showClearCredentialsDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" class="shadow-lg shadow-destructive/25" @click="clearCredentials">
            {{ t('common.confirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Clear History Dialog -->
    <Dialog v-model:open="showClearHistoryDialog">
      <DialogContent class="border-destructive/30 bg-card/95 backdrop-blur-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-3 font-display">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-destructive/20 border border-destructive/30">
              <Trash2 class="h-5 w-5 text-destructive" />
            </div>
            {{ t('settings.clearHistory') }}
          </DialogTitle>
          <DialogDescription>
            {{ t('settings.confirmClearHistory') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" class="border-border/50" @click="showClearHistoryDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" class="shadow-lg shadow-destructive/25" @click="clearHistory">
            {{ t('common.confirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<style scoped>
.gradient-text {
  background: linear-gradient(135deg, hsl(var(--primary)) 0%, hsl(var(--accent)) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
</style>
