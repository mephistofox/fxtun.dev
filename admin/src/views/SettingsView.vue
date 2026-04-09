<template>
  <n-space vertical :size="24">
    <!-- Server Config -->
    <n-card title="Server Configuration">
      <n-spin :show="loadingSettings">
        <n-descriptions bordered :column="2" label-placement="left" v-if="settings">
          <n-descriptions-item label="Control Port">{{ settings.control_port }}</n-descriptions-item>
          <n-descriptions-item label="HTTP Port">{{ settings.http_port }}</n-descriptions-item>
          <n-descriptions-item label="Web Port">{{ settings.web_port }}</n-descriptions-item>
          <n-descriptions-item label="Base Domain">{{ settings.base_domain }}</n-descriptions-item>
          <n-descriptions-item label="CORS Origins">{{ settings.cors_origins }}</n-descriptions-item>
          <n-descriptions-item label="Registration Enabled">{{ settings.registration_enabled ? 'Yes' : 'No' }}</n-descriptions-item>
          <n-descriptions-item label="TOTP Enabled">{{ settings.totp_enabled ? 'Yes' : 'No' }}</n-descriptions-item>
        </n-descriptions>
        <n-empty v-else-if="!loadingSettings" description="Could not load settings" />
      </n-spin>
    </n-card>

    <!-- System Info -->
    <n-card title="System Information">
      <n-spin :show="loadingSystem">
        <n-descriptions bordered :column="2" label-placement="left" v-if="systemInfo">
          <n-descriptions-item label="Version">{{ systemInfo.version }}</n-descriptions-item>
          <n-descriptions-item label="Go Version">{{ systemInfo.go_version }}</n-descriptions-item>
          <n-descriptions-item label="OS">{{ systemInfo.os }}</n-descriptions-item>
          <n-descriptions-item label="Architecture">{{ systemInfo.arch }}</n-descriptions-item>
          <n-descriptions-item label="CPU Count">{{ systemInfo.num_cpu }}</n-descriptions-item>
          <n-descriptions-item label="Goroutines">{{ systemInfo.goroutines }}</n-descriptions-item>
        </n-descriptions>
        <n-empty v-else-if="!loadingSystem" description="Could not load system info" />
      </n-spin>
    </n-card>

    <!-- Invite Codes -->
    <n-card title="Invite Codes">
      <template #header-extra>
        <n-button type="primary" size="small" @click="openCreateCodeModal">
          Create Code
        </n-button>
      </template>
      <n-data-table
        :columns="codeColumns"
        :data="inviteCodes"
        :loading="loadingCodes"
        :row-key="(row: InviteCode) => row.id"
      />
    </n-card>

    <!-- Create Code Modal -->
    <n-modal
      v-model:show="showCodeModal"
      preset="card"
      title="Create Invite Code"
      style="width: 400px"
      :mask-closable="false"
    >
      <n-space vertical :size="12">
        <n-form-item label="Code (leave empty for random)">
          <n-input v-model:value="newCode" placeholder="e.g. WELCOME2026" clearable />
        </n-form-item>
        <n-form-item label="Max Uses">
          <n-input-number v-model:value="newMaxUses" :min="1" style="width: 100%" />
        </n-form-item>
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCodeModal = false">Cancel</n-button>
          <n-button type="primary" :loading="creatingCode" @click="handleCreateCode">Create</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, h } from 'vue'
import { useMessage, useDialog, NButton } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { format } from 'date-fns'
import { adminApi } from '@/api/client'
import type { ServerSettings, SystemInfo, InviteCode } from '@/api/types'

const message = useMessage()
const dialog = useDialog()

// Server Config
const settings = ref<ServerSettings | null>(null)
const loadingSettings = ref(false)

// System Info
const systemInfo = ref<SystemInfo | null>(null)
const loadingSystem = ref(false)
let systemRefreshInterval: ReturnType<typeof setInterval> | null = null

// Invite Codes
const inviteCodes = ref<InviteCode[]>([])
const loadingCodes = ref(false)
const showCodeModal = ref(false)
const newCode = ref('')
const newMaxUses = ref(10)
const creatingCode = ref(false)

const codeColumns: DataTableColumns<InviteCode> = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: 'Code',
    key: 'code',
    width: 200,
    render(row) {
      return h('code', { style: 'font-family: monospace; font-size: 13px' }, row.code)
    },
  },
  {
    title: 'Max Uses',
    key: 'max_uses',
    width: 90,
    render(row) {
      return row.max_uses != null ? String(row.max_uses) : 'Unlimited'
    },
  },
  {
    title: 'Used',
    key: 'use_count',
    width: 70,
  },
  {
    title: 'Created At',
    key: 'created_at',
    width: 160,
    render(row) {
      return row.created_at ? format(new Date(row.created_at), 'yyyy-MM-dd HH:mm') : '-'
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 80,
    render(row) {
      return h(
        NButton,
        { size: 'small', type: 'error', quaternary: true, onClick: () => handleDeleteCode(row) },
        { default: () => 'Delete' },
      )
    },
  },
]

async function fetchSettings() {
  loadingSettings.value = true
  try {
    const { data } = await adminApi.getSettings()
    settings.value = data
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load settings')
  } finally {
    loadingSettings.value = false
  }
}

async function fetchSystemInfo() {
  loadingSystem.value = true
  try {
    const { data } = await adminApi.getSystemInfo()
    systemInfo.value = data
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load system info')
  } finally {
    loadingSystem.value = false
  }
}

async function refreshGoroutines() {
  try {
    const { data } = await adminApi.getSystemInfo()
    if (systemInfo.value) {
      systemInfo.value.goroutines = data.goroutines
    }
  } catch {
    // Silently fail on auto-refresh
  }
}

async function fetchInviteCodes() {
  loadingCodes.value = true
  try {
    const { data } = await adminApi.listInviteCodes()
    inviteCodes.value = data.codes || []
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load invite codes')
  } finally {
    loadingCodes.value = false
  }
}

function openCreateCodeModal() {
  newCode.value = ''
  newMaxUses.value = 10
  showCodeModal.value = true
}

async function handleCreateCode() {
  creatingCode.value = true
  try {
    await adminApi.createInviteCode(newCode.value || undefined, newMaxUses.value)
    message.success('Invite code created')
    showCodeModal.value = false
    await fetchInviteCodes()
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to create invite code')
  } finally {
    creatingCode.value = false
  }
}

function handleDeleteCode(code: InviteCode) {
  dialog.error({
    title: 'Delete Invite Code',
    content: `Delete invite code "${code.code}"?`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.deleteInviteCode(code.id)
        message.success('Invite code deleted')
        await fetchInviteCodes()
      } catch (err: unknown) {
        const error = err as { response?: { data?: { error?: string } }; message?: string }
        message.error(error.response?.data?.error || error.message || 'Failed to delete invite code')
      }
    },
  })
}

onMounted(() => {
  fetchSettings()
  fetchSystemInfo()
  fetchInviteCodes()
  systemRefreshInterval = setInterval(refreshGoroutines, 10000)
})

onUnmounted(() => {
  if (systemRefreshInterval) {
    clearInterval(systemRefreshInterval)
    systemRefreshInterval = null
  }
})
</script>
