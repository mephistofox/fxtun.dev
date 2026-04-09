<template>
  <n-space vertical :size="16">
    <!-- Toolbar -->
    <n-space align="center" justify="space-between">
      <n-input
        v-model:value="searchText"
        placeholder="Search domains..."
        clearable
        style="width: 280px"
      />
      <n-pagination
        v-model:page="currentPage"
        :page-count="totalPages"
        :page-slot="7"
        @update:page="fetchDomains"
      />
    </n-space>

    <!-- Table -->
    <n-data-table
      :columns="columns"
      :data="filteredDomains"
      :loading="loading"
      :row-key="(row: CustomDomain) => row.id"
    />
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useMessage, useDialog, NTag, NButton } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { format, differenceInDays } from 'date-fns'
import { adminApi } from '@/api/client'
import type { CustomDomain } from '@/api/types'

const message = useMessage()
const dialog = useDialog()

const domains = ref<CustomDomain[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 20
const searchText = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const filteredDomains = computed(() => {
  if (!searchText.value) return domains.value
  const q = searchText.value.toLowerCase()
  return domains.value.filter(d =>
    d.domain.toLowerCase().includes(q) ||
    d.target_subdomain?.toLowerCase().includes(q) ||
    d.user_phone?.toLowerCase().includes(q),
  )
})

const columns: DataTableColumns<CustomDomain> = [
  {
    title: 'Domain',
    key: 'domain',
    width: 200,
    ellipsis: { tooltip: true },
  },
  {
    title: 'Target',
    key: 'target_subdomain',
    width: 180,
    ellipsis: { tooltip: true },
  },
  {
    title: 'User',
    key: 'user_phone',
    width: 140,
  },
  {
    title: 'Status',
    key: 'verified',
    width: 100,
    render(row) {
      return h(NTag, { type: row.verified ? 'success' : 'warning', size: 'small' }, {
        default: () => row.verified ? 'Verified' : 'Pending',
      })
    },
  },
  {
    title: 'TLS Expiry',
    key: 'tls_expiry',
    width: 150,
    render(row) {
      if (!row.tls_expiry) return '-'
      const expiry = new Date(row.tls_expiry)
      const daysLeft = differenceInDays(expiry, new Date())
      const text = format(expiry, 'yyyy-MM-dd')
      if (daysLeft < 7) {
        return h(NTag, { type: 'error', size: 'small' }, { default: () => `${text} (${daysLeft}d)` })
      }
      if (daysLeft < 30) {
        return h(NTag, { type: 'warning', size: 'small' }, { default: () => `${text} (${daysLeft}d)` })
      }
      return text
    },
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
        { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) },
        { default: () => 'Delete' },
      )
    },
  },
]

function handleDelete(domain: CustomDomain) {
  dialog.error({
    title: 'Delete Domain',
    content: `Permanently delete domain "${domain.domain}"? This cannot be undone.`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.deleteCustomDomain(domain.id)
        message.success('Domain deleted')
        await fetchDomains()
      } catch (err: unknown) {
        const error = err as { response?: { data?: { error?: string } }; message?: string }
        message.error(error.response?.data?.error || error.message || 'Failed to delete domain')
      }
    },
  })
}

async function fetchDomains() {
  loading.value = true
  try {
    const { data } = await adminApi.listCustomDomains(currentPage.value, pageSize)
    domains.value = data.domains || []
    total.value = data.total || 0
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load domains')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchDomains()
})
</script>
