<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-display font-bold">Тарифы</h1>
      <Button @click="openCreate">
        <Plus class="h-4 w-4" />
        Создать тариф
      </Button>
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="plans"
      :loading="loading"
      row-key="id"
      empty-text="Нет тарифов"
    >
      <template #id="{ value }">
        <span class="font-mono text-sm text-muted-foreground">{{ value }}</span>
      </template>

      <template #slug="{ value }">
        <span class="font-mono text-sm">{{ value }}</span>
      </template>

      <template #name="{ value }">
        <span class="font-medium">{{ value }}</span>
      </template>

      <template #price="{ value }">
        <span class="text-sm">${{ value }}</span>
      </template>

      <template #max_tunnels="{ value }">
        <span class="text-sm">{{ value <= 0 ? '\u221E' : value }}</span>
      </template>

      <template #max_domains="{ value }">
        <span class="text-sm">{{ value <= 0 ? '\u221E' : value }}</span>
      </template>

      <template #max_custom_domains="{ value }">
        <span class="text-sm">{{ value <= 0 ? '\u221E' : value }}</span>
      </template>

      <template #inspector_enabled="{ value }">
        <Badge :variant="value ? 'success' : 'outline'">{{ value ? 'Да' : 'Нет' }}</Badge>
      </template>

      <template #is_public="{ value }">
        <Badge :variant="value ? 'success' : 'outline'">{{ value ? 'Да' : 'Нет' }}</Badge>
      </template>

      <template #actions="{ row }">
        <Dropdown :items="rowActions" @select="(key) => handleAction(key, row)">
          <Button variant="ghost" size="icon">
            <MoreHorizontal class="h-4 w-4" />
          </Button>
        </Dropdown>
      </template>
    </DataTable>

    <!-- Create/Edit modal -->
    <Modal v-model:show="showModal" :title="editingPlan ? 'Редактировать тариф' : 'Создать тариф'" width="max-w-xl">
      <form class="space-y-4" @submit.prevent="savePlan">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Slug</label>
            <Input v-model="form.slug" placeholder="free" :disabled="!!editingPlan" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Название</label>
            <Input v-model="form.name" placeholder="Free" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Цена USD</label>
            <Input v-model="form.price" type="number" placeholder="0" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Макс. тоннелей</label>
            <Input v-model="form.max_tunnels" type="number" placeholder="3" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Макс. доменов</label>
            <Input v-model="form.max_domains" type="number" placeholder="0" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Макс. кастомных доменов</label>
            <Input v-model="form.max_custom_domains" type="number" placeholder="0" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Макс. токенов</label>
            <Input v-model="form.max_tokens" type="number" placeholder="1" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">Макс. тоннелей на токен</label>
            <Input v-model="form.max_tunnels_per_token" type="number" placeholder="3" />
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-6 pt-2">
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="form.inspector_enabled" class="h-4 w-4 rounded border-input accent-primary" />
            <span class="text-sm">Инспектор</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="form.is_public" class="h-4 w-4 rounded border-input accent-primary" />
            <span class="text-sm">Публичный</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="form.is_recommended" class="h-4 w-4 rounded border-input accent-primary" />
            <span class="text-sm">Рекомендуемый</span>
          </label>
        </div>
      </form>

      <template #footer>
        <Button variant="outline" @click="showModal = false">Отмена</Button>
        <Button :loading="saving" @click="savePlan">
          {{ editingPlan ? 'Сохранить' : 'Создать' }}
        </Button>
      </template>
    </Modal>

    <!-- Delete confirm -->
    <ConfirmDialog
      v-model:show="showDeleteConfirm"
      title="Удалить тариф"
      :message="`Удалить тариф «${deletingPlan?.name || ''}»? Это действие необратимо.`"
      confirm-text="Удалить"
      variant="destructive"
      @confirm="deletePlan"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApi } from '@/api/client'
import type { Plan } from '@/api/types'
import { getErrorMessage } from '@/utils/error'
import { MoreHorizontal, Plus } from 'lucide-vue-next'
import DataTable from '@/components/ui/DataTable.vue'
import type { Column } from '@/components/ui/DataTable.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Modal from '@/components/ui/Modal.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'

const plans = ref<Plan[]>([])
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const showDeleteConfirm = ref(false)
const editingPlan = ref<Plan | null>(null)
const deletingPlan = ref<Plan | null>(null)

const defaultForm = () => ({
  slug: '',
  name: '',
  price: 0,
  max_tunnels: 3,
  max_domains: 0,
  max_custom_domains: 0,
  max_tokens: 1,
  max_tunnels_per_token: 3,
  inspector_enabled: false,
  is_public: true,
  is_recommended: false,
})

const form = ref(defaultForm())

const columns: Column[] = [
  { key: 'id', title: 'ID', width: '60px' },
  { key: 'slug', title: 'Slug', width: '120px' },
  { key: 'name', title: 'Название' },
  { key: 'price', title: 'Цена USD', width: '100px' },
  { key: 'max_tunnels', title: 'Макс. тоннелей', width: '130px' },
  { key: 'max_domains', title: 'Макс. доменов', width: '130px' },
  { key: 'max_custom_domains', title: 'Макс. кастом.', width: '120px' },
  { key: 'inspector_enabled', title: 'Инспектор', width: '100px' },
  { key: 'is_public', title: 'Публичный', width: '100px' },
  { key: 'actions', title: '', width: '60px', align: 'right' },
]

const rowActions = [
  { key: 'edit', label: 'Редактировать' },
  { key: 'delete', label: 'Удалить', destructive: true },
]

function handleAction(key: string, row: Plan) {
  if (key === 'edit') {
    openEdit(row)
  } else if (key === 'delete') {
    deletingPlan.value = row
    showDeleteConfirm.value = true
  }
}

function openCreate() {
  editingPlan.value = null
  form.value = defaultForm()
  showModal.value = true
}

function openEdit(plan: Plan) {
  editingPlan.value = plan
  form.value = {
    slug: plan.slug,
    name: plan.name,
    price: plan.price,
    max_tunnels: plan.max_tunnels,
    max_domains: plan.max_domains,
    max_custom_domains: plan.max_custom_domains,
    max_tokens: plan.max_tokens,
    max_tunnels_per_token: plan.max_tunnels_per_token,
    inspector_enabled: plan.inspector_enabled,
    is_public: plan.is_public,
    is_recommended: plan.is_recommended,
  }
  showModal.value = true
}

async function fetchPlans() {
  loading.value = true
  try {
    const { data } = await adminApi.listPlans()
    plans.value = data.plans || []
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

async function savePlan() {
  saving.value = true
  try {
    const payload = {
      slug: form.value.slug,
      name: form.value.name,
      price: Number(form.value.price),
      max_tunnels: Number(form.value.max_tunnels),
      max_domains: Number(form.value.max_domains),
      max_custom_domains: Number(form.value.max_custom_domains),
      max_tokens: Number(form.value.max_tokens),
      max_tunnels_per_token: Number(form.value.max_tunnels_per_token),
      inspector_enabled: form.value.inspector_enabled,
      is_public: form.value.is_public,
      is_recommended: form.value.is_recommended,
      price_rub: 0,
      rate_limit_tcp: 0,
      rate_limit_udp: 0,
      rate_limit_http: 0,
      creem_product_id: '',
    }
    if (editingPlan.value) {
      await adminApi.updatePlan(editingPlan.value.id, payload)
    } else {
      await adminApi.createPlan(payload)
    }
    showModal.value = false
    await fetchPlans()
  } catch (err) {
    console.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function deletePlan() {
  if (!deletingPlan.value) return
  try {
    await adminApi.deletePlan(deletingPlan.value.id)
    await fetchPlans()
  } catch (err) {
    console.error(getErrorMessage(err))
  }
}

onMounted(() => {
  fetchPlans()
})
</script>
