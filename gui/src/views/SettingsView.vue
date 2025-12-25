<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore, type Theme, type Locale } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useHistoryStore } from '@/stores/history'
import { toast } from '@/composables/useToast'
import {
  Button, Card, CardHeader, CardTitle, CardContent, Input, Label, Switch, Separator,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter
} from '@/components/ui'
import {
  Moon, Sun, Monitor, Trash2, Palette, Globe, Server, Bell, Shield, Info, Languages
} from 'lucide-vue-next'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const historyStore = useHistoryStore()

const serverAddress = ref('')
const platform = typeof window !== 'undefined' ? window.navigator.platform : 'Unknown'

// Dialogs
const showClearCredentialsDialog = ref(false)
const showClearHistoryDialog = ref(false)

// Sync serverAddress with store when it loads
onMounted(() => {
  serverAddress.value = settingsStore.serverAddress || ''
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
    <h1 class="text-2xl font-bold">{{ t('settings.title') }}</h1>

    <!-- Appearance -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="flex items-center gap-2 text-base">
          <Palette class="h-4 w-4 text-muted-foreground" />
          {{ t('settings.appearance') }}
        </CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <!-- Theme -->
        <div>
          <Label class="mb-2 block">{{ t('settings.theme') }}</Label>
          <div class="flex flex-wrap gap-2">
            <Button
              v-for="theme in themes"
              :key="theme.value"
              :variant="settingsStore.theme === theme.value ? 'default' : 'outline'"
              size="sm"
              @click="selectTheme(theme.value)"
            >
              <component :is="theme.icon" class="mr-2 h-4 w-4" />
              {{ t(theme.labelKey) }}
            </Button>
          </div>
        </div>

        <Separator />

        <!-- Language -->
        <div>
          <Label class="mb-2 flex items-center gap-2">
            <Languages class="h-4 w-4 text-muted-foreground" />
            {{ t('settings.language') }}
          </Label>
          <div class="flex flex-wrap gap-2">
            <Button
              v-for="lang in languages"
              :key="lang.value"
              :variant="settingsStore.locale === lang.value ? 'default' : 'outline'"
              size="sm"
              @click="selectLanguage(lang.value)"
            >
              <span class="mr-2 text-xs font-bold opacity-60">{{ lang.flag }}</span>
              {{ lang.label }}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Connection -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="flex items-center gap-2 text-base">
          <Globe class="h-4 w-4 text-muted-foreground" />
          {{ t('settings.connection') }}
        </CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="space-y-2">
          <Label class="flex items-center gap-2">
            <Server class="h-4 w-4 text-muted-foreground" />
            {{ t('settings.defaultServer') }}
          </Label>
          <p class="text-xs text-muted-foreground">{{ t('settings.defaultServerHint') }}</p>
          <div class="flex gap-2">
            <Input
              v-model="serverAddress"
              :placeholder="t('settings.defaultServerPlaceholder')"
              class="flex-1"
            />
            <Button @click="saveServerAddress">{{ t('common.save') }}</Button>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- System -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="flex items-center gap-2 text-base">
          <Bell class="h-4 w-4 text-muted-foreground" />
          {{ t('settings.system') }}
        </CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="space-y-0.5">
            <Label>{{ t('settings.minimizeToTray') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.minimizeToTrayHint') }}
            </p>
          </div>
          <Switch
            :model-value="settingsStore.minimizeToTray"
            @update:model-value="settingsStore.saveMinimizeToTray($event)"
          />
        </div>

        <Separator />

        <div class="flex items-center justify-between">
          <div class="space-y-0.5">
            <Label>{{ t('settings.notifications') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.notificationsHint') }}
            </p>
          </div>
          <Switch
            :model-value="settingsStore.notifications"
            @update:model-value="settingsStore.saveNotifications($event)"
          />
        </div>
      </CardContent>
    </Card>

    <!-- Data Management -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="flex items-center gap-2 text-base">
          <Shield class="h-4 w-4 text-muted-foreground" />
          {{ t('settings.data') }}
        </CardTitle>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="space-y-0.5">
            <Label>{{ t('settings.clearCredentials') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.clearCredentialsHint') }}
            </p>
          </div>
          <Button variant="destructive" size="sm" @click="showClearCredentialsDialog = true">
            <Trash2 class="mr-2 h-4 w-4" />
            {{ t('common.clear') }}
          </Button>
        </div>

        <Separator />

        <div class="flex items-center justify-between">
          <div class="space-y-0.5">
            <Label>{{ t('settings.clearHistory') }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ t('settings.clearHistoryHint') }}
            </p>
          </div>
          <Button variant="destructive" size="sm" @click="showClearHistoryDialog = true">
            <Trash2 class="mr-2 h-4 w-4" />
            {{ t('common.clear') }}
          </Button>
        </div>
      </CardContent>
    </Card>

    <!-- About -->
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="flex items-center gap-2 text-base">
          <Info class="h-4 w-4 text-muted-foreground" />
          {{ t('settings.about') }}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div class="space-y-2 text-sm">
          <div class="flex justify-between">
            <span class="text-muted-foreground">{{ t('settings.version') }}</span>
            <span class="font-medium">1.0.0</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">{{ t('settings.build') }}</span>
            <span class="font-mono text-xs">2024.01.01</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">{{ t('settings.platform') }}</span>
            <span class="font-mono text-xs">{{ platform }}</span>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Clear Credentials Dialog -->
    <Dialog v-model:open="showClearCredentialsDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t('settings.clearCredentials') }}</DialogTitle>
          <DialogDescription>
            {{ t('settings.confirmClearCredentials') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showClearCredentialsDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" @click="clearCredentials">
            {{ t('common.confirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Clear History Dialog -->
    <Dialog v-model:open="showClearHistoryDialog">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t('settings.clearHistory') }}</DialogTitle>
          <DialogDescription>
            {{ t('settings.confirmClearHistory') }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="showClearHistoryDialog = false">
            {{ t('common.cancel') }}
          </Button>
          <Button variant="destructive" @click="clearHistory">
            {{ t('common.confirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
