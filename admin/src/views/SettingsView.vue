<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold">Настройки</h1>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Server config -->
      <Card variant="glass" class="p-6">
        <h2 class="text-lg font-display font-semibold mb-4">Конфигурация сервера</h2>
        <div v-if="settingsLoading" class="flex items-center justify-center py-8">
          <svg class="h-6 w-6 animate-spin text-primary" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </div>
        <dl v-else-if="settings" class="space-y-3">
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">Порт управления</dt>
            <dd class="text-sm font-mono">{{ settings.server?.control_port }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">HTTP порт</dt>
            <dd class="text-sm font-mono">{{ settings.server?.http_port }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">Web порт</dt>
            <dd class="text-sm font-mono">{{ settings.web?.port }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">CORS домены</dt>
            <dd class="text-sm font-mono text-right max-w-[200px] truncate" :title="String(settings.web?.cors_origins)">{{ settings.web?.cors_origins }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">Базовый домен</dt>
            <dd class="text-sm font-mono">{{ settings.domain?.base }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">Регистрация</dt>
            <dd>
              <Badge :variant="settings.features?.registration_enabled ? 'success' : 'outline'">
                {{ settings.features?.registration_enabled ? 'Включена' : 'Выключена' }}
              </Badge>
            </dd>
          </div>
          <div class="flex justify-between py-2">
            <dt class="text-sm text-muted-foreground">TOTP</dt>
            <dd>
              <Badge :variant="settings.features?.totp_enabled ? 'success' : 'outline'">
                {{ settings.features?.totp_enabled ? 'Включён' : 'Выключен' }}
              </Badge>
            </dd>
          </div>
        </dl>
      </Card>

      <!-- System info -->
      <Card variant="glass" class="p-6">
        <h2 class="text-lg font-display font-semibold mb-4">Системная информация</h2>
        <div v-if="systemLoading" class="flex items-center justify-center py-8">
          <svg class="h-6 w-6 animate-spin text-primary" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </div>
        <dl v-else-if="systemInfo" class="space-y-3">
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">Версия</dt>
            <dd class="text-sm font-mono text-primary">{{ systemInfo.version }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">Go версия</dt>
            <dd class="text-sm font-mono">{{ systemInfo.go_version }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">ОС</dt>
            <dd class="text-sm">{{ systemInfo.os }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">Архитектура</dt>
            <dd class="text-sm">{{ systemInfo.arch }}</dd>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <dt class="text-sm text-muted-foreground">CPU</dt>
            <dd class="text-sm">{{ systemInfo.num_cpu }}</dd>
          </div>
          <div class="flex justify-between py-2">
            <dt class="text-sm text-muted-foreground">Горутины</dt>
            <dd class="text-sm font-mono">{{ systemInfo.goroutines }}</dd>
          </div>
        </dl>
      </Card>
    </div>

    <!-- Invite codes -->
    <Card variant="glass" class="p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-display font-semibold">Инвайт-коды</h2>
        <Button size="sm" @click="openCreateCode">
          <Plus class="h-4 w-4" />
          Создать код
        </Button>
      </div>

      <DataTable
        :columns="codeColumns"
        :data="inviteCodes"
        :loading="codesLoading"
        row-key="id"
        empty-text="Нет инвайт-кодов"
      >
        <template #id="{ value }">
          <span class="font-mono text-sm text-muted-foreground">{{ value }}</span>
        </template>

        <template #code="{ value }">
          <span class="font-mono text-sm">{{ value }}</span>
        </template>

        <template #max_uses="{ value }">
          <span class="text-sm">{{ value || '\u221E' }}</span>
        </template>

        <template #use_count="{ value }">
          <span class="text-sm">{{ value }}</span>
        </template>

        <template #created_at="{ value }">
          <span class="text-sm text-muted-foreground">{{ formatDate(value) }}</span>
        </template>

        <template #actions="{ row }">
          <Button variant="ghost" size="icon" @click="confirmDeleteCode(row)">
            <Trash2 class="h-4 w-4 text-destructive" />
          </Button>
        </template>
      </DataTable>
    </Card>

    <!-- Create code modal -->
    <Modal v-model:show="showCreateCodeModal" title="Создать инвайт-код" width="max-w-sm">
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-foreground mb-1.5">Код (необязательно)</label>
          <Input v-model="newCode" placeholder="Оставьте пустым для автогенерации" />
        </div>
        <div>
          <label class="block text-sm font-medium text-foreground mb-1.5">Макс. использований</label>
          <Input v-model="newCodeMaxUses" type="number" placeholder="1" />
        </div>
      </div>
      <template #footer>
        <Button variant="outline" @click="showCreateCodeModal = false">Отмена</Button>
        <Button :loading="creatingCode" @click="createCode">Создать</Button>
      </template>
    </Modal>

    <!-- Delete code confirm -->
    <ConfirmDialog
      v-model:show="showDeleteCodeConfirm"
      title="Удалить инвайт-код"
      :message="`Удалить инвайт-код «${deletingCode?.code || ''}»?`"
      confirm-text="Удалить"
      variant="destructive"
      @confirm="deleteCode"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { adminApi } from '@/api/client'
import type { ServerSettings, SystemInfo, InviteCode } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale'
import { Plus, Trash2 } from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Modal from '@/components/ui/Modal.vue'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'

const settings = ref<ServerSettings | null>(null)
const systemInfo = ref<SystemInfo | null>(null)
const inviteCodes = ref<InviteCode[]>([])
const settingsLoading = ref(false)
const systemLoading = ref(false)
const codesLoading = ref(false)

const showCreateCodeModal = ref(false)
const creatingCode = ref(false)
const newCode = ref<string | number>('')
const newCodeMaxUses = ref<string | number>(1)

const showDeleteCodeConfirm = ref(false)
const deletingCode = ref<InviteCode | null>(null)

let goroutineInterval: ReturnType<typeof setInterval> | null = null

const codeColumns: Column[] = [
  { key: 'id', title: 'ID', width: '60px' },
  { key: 'code', title: 'Код' },
  { key: 'max_uses', title: 'Макс. использований', width: '170px' },
  { key: 'use_count', title: 'Использован', width: '120px' },
  { key: 'created_at', title: 'Создан', width: '160px' },
  { key: 'actions', title: '', width: '60px', align: 'right' },
]

function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'dd.MM.yyyy HH:mm', { locale: ru })
}

function openCreateCode() {
  newCode.value = ''
  newCodeMaxUses.value = 1
  showCreateCodeModal.value = true
}

function confirmDeleteCode(code: InviteCode) {
  deletingCode.value = code
  showDeleteCodeConfirm.value = true
}

async function fetchSettings() {
  settingsLoading.value = true
  try {
    const { data } = await adminApi.getSettings()
    settings.value = data
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    settingsLoading.value = false
  }
}

async function fetchSystemInfo() {
  systemLoading.value = true
  try {
    const { data } = await adminApi.getSystemInfo()
    systemInfo.value = data
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    systemLoading.value = false
  }
}

async function refreshGoroutines() {
  try {
    const { data } = await adminApi.getSystemInfo()
    if (systemInfo.value) {
      systemInfo.value.goroutines = data.goroutines
    }
  } catch {
    // silently ignore refresh errors
  }
}

async function fetchInviteCodes() {
  codesLoading.value = true
  try {
    const { data } = await adminApi.listInviteCodes()
    inviteCodes.value = data.codes || []
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    codesLoading.value = false
  }
}

async function createCode() {
  creatingCode.value = true
  try {
    const code = String(newCode.value).trim() || undefined
    const maxUses = Number(newCodeMaxUses.value) || undefined
    await adminApi.createInviteCode(code, maxUses)
    showCreateCodeModal.value = false
    await fetchInviteCodes()
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    creatingCode.value = false
  }
}

async function deleteCode() {
  if (!deletingCode.value) return
  try {
    await adminApi.deleteInviteCode(deletingCode.value.id)
    await fetchInviteCodes()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

onMounted(() => {
  fetchSettings()
  fetchSystemInfo()
  fetchInviteCodes()
  goroutineInterval = setInterval(refreshGoroutines, 10000)
})

onUnmounted(() => {
  if (goroutineInterval) {
    clearInterval(goroutineInterval)
  }
})
</script>
