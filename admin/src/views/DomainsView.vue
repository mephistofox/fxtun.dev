<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold">Домены</h1>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-3">
      <Input
        v-model="search"
        placeholder="Поиск по домену, субдомену, пользователю..."
        class="w-80"
      />
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredDomains"
      :loading="loading"
      row-key="id"
      empty-text="Нет доменов"
    >
      <template #domain="{ value }">
        <span class="font-mono font-bold text-sm">{{ value }}</span>
      </template>

      <template #target_subdomain="{ value }">
        <span class="text-sm">{{ value }}</span>
      </template>

      <template #user_phone="{ value }">
        <span class="text-sm">{{ value || '-' }}</span>
      </template>

      <template #verified="{ value }">
        <Badge :variant="value ? 'success' : 'warning'">
          {{ value ? 'Подтверждён' : 'Ожидает' }}
        </Badge>
      </template>

      <template #tls_expiry="{ value }">
        <template v-if="value">
          <Badge :variant="tlsExpiryBadge(value)">
            {{ formatDate(value) }}
          </Badge>
        </template>
        <span v-else class="text-sm text-muted-foreground">-</span>
      </template>

      <template #created_at="{ value }">
        <span class="text-sm text-muted-foreground">{{ formatDate(value) }}</span>
      </template>

      <template #actions="{ row }">
        <Dropdown :items="rowActions" @select="(key) => handleAction(key, row)">
          <Button variant="ghost" size="icon">
            <MoreHorizontal class="h-4 w-4" />
          </Button>
        </Dropdown>
      </template>
    </DataTable>

    <!-- Pagination -->
    <Pagination
      v-if="total > pageSize"
      :page="page"
      :total="total"
      :page-size="pageSize"
      @update:page="(p) => { page = p; fetchDomains() }"
      @update:page-size="(s) => { pageSize = s; page = 1; fetchDomains() }"
    />

    <!-- Delete confirm -->
    <ConfirmDialog
      v-model:show="showDeleteConfirm"
      title="Удалить домен"
      :message="`Удалить домен «${deletingDomain?.domain || ''}»? Это действие необратимо.`"
      confirm-text="Удалить"
      variant="destructive"
      @confirm="deleteDomain"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { adminApi } from '@/api/client'
import type { CustomDomain } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { format, differenceInDays } from 'date-fns'
import { ru } from 'date-fns/locale'
import { MoreHorizontal } from 'lucide-vue-next'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import Pagination from '@/components/ui/Pagination.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'

const domains = ref<CustomDomain[]>([])
const loading = ref(false)
const search = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const showDeleteConfirm = ref(false)
const deletingDomain = ref<CustomDomain | null>(null)

const columns: Column[] = [
  { key: 'domain', title: 'Домен' },
  { key: 'target_subdomain', title: 'Целевой субдомен' },
  { key: 'user_phone', title: 'Пользователь' },
  { key: 'verified', title: 'Статус', width: '130px' },
  { key: 'tls_expiry', title: 'TLS истекает', width: '150px' },
  { key: 'created_at', title: 'Создан', width: '160px' },
  { key: 'actions', title: '', width: '60px', align: 'right' },
]

const rowActions = [
  { key: 'delete', label: 'Удалить', destructive: true },
]

const filteredDomains = computed(() => {
  if (!search.value) return domains.value
  const q = search.value.toLowerCase()
  return domains.value.filter(d =>
    d.domain.toLowerCase().includes(q) ||
    d.target_subdomain.toLowerCase().includes(q) ||
    (d.user_phone?.toLowerCase().includes(q))
  )
})

function tlsExpiryBadge(dateStr: string): 'destructive' | 'warning' | 'outline' {
  const days = differenceInDays(new Date(dateStr), new Date())
  if (days < 7) return 'destructive'
  if (days < 30) return 'warning'
  return 'outline'
}

function formatDate(dateStr: string): string {
  return format(new Date(dateStr), 'dd.MM.yyyy HH:mm', { locale: ru })
}

function handleAction(key: string, row: CustomDomain) {
  if (key === 'delete') {
    deletingDomain.value = row
    showDeleteConfirm.value = true
  }
}

async function fetchDomains() {
  loading.value = true
  try {
    const { data } = await adminApi.listCustomDomains(page.value, pageSize.value)
    domains.value = data.domains || []
    total.value = data.total
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function deleteDomain() {
  if (!deletingDomain.value) return
  try {
    await adminApi.deleteCustomDomain(deletingDomain.value.id)
    await fetchDomains()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

onMounted(() => {
  fetchDomains()
})
</script>
