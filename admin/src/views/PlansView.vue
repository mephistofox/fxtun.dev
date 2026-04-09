<template>
  <n-space vertical :size="16">
    <!-- Toolbar -->
    <n-space justify="end">
      <n-button type="primary" @click="openCreateModal">Create Plan</n-button>
    </n-space>

    <!-- Table -->
    <n-data-table
      :columns="columns"
      :data="plans"
      :loading="loading"
      :row-key="(row: Plan) => row.id"
    />

    <!-- Create/Edit Modal -->
    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="editingPlan ? 'Edit Plan' : 'Create Plan'"
      style="width: 600px"
      :mask-closable="false"
    >
      <n-form
        ref="formRef"
        :model="formValue"
        :rules="formRules"
        label-placement="left"
        label-width="180"
      >
        <n-form-item label="Slug" path="slug">
          <n-input v-model:value="formValue.slug" placeholder="e.g. free, base, pro" :disabled="!!editingPlan" />
        </n-form-item>
        <n-form-item label="Name" path="name">
          <n-input v-model:value="formValue.name" placeholder="Plan name" />
        </n-form-item>
        <n-form-item label="Price (USD)" path="price">
          <n-input-number v-model:value="formValue.price" :min="0" :precision="2" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Price (RUB)" path="price_rub">
          <n-input-number v-model:value="formValue.price_rub" :min="0" :precision="2" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Max Tunnels" path="max_tunnels">
          <n-input-number v-model:value="formValue.max_tunnels" :min="-1" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Max Domains" path="max_domains">
          <n-input-number v-model:value="formValue.max_domains" :min="-1" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Max Custom Domains" path="max_custom_domains">
          <n-input-number v-model:value="formValue.max_custom_domains" :min="-1" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Max Tokens" path="max_tokens">
          <n-input-number v-model:value="formValue.max_tokens" :min="-1" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Max Tunnels/Token" path="max_tunnels_per_token">
          <n-input-number v-model:value="formValue.max_tunnels_per_token" :min="-1" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Rate Limit TCP" path="rate_limit_tcp">
          <n-input-number v-model:value="formValue.rate_limit_tcp" :min="0" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Rate Limit UDP" path="rate_limit_udp">
          <n-input-number v-model:value="formValue.rate_limit_udp" :min="0" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Rate Limit HTTP" path="rate_limit_http">
          <n-input-number v-model:value="formValue.rate_limit_http" :min="0" style="width: 100%" />
        </n-form-item>
        <n-form-item label="Creem Product ID" path="creem_product_id">
          <n-input v-model:value="formValue.creem_product_id" placeholder="Optional" />
        </n-form-item>
        <n-form-item label="Inspector Enabled">
          <n-switch v-model:value="formValue.inspector_enabled" />
        </n-form-item>
        <n-form-item label="Public">
          <n-switch v-model:value="formValue.is_public" />
        </n-form-item>
        <n-form-item label="Recommended">
          <n-switch v-model:value="formValue.is_recommended" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">Cancel</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">
            {{ editingPlan ? 'Update' : 'Create' }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useMessage, useDialog, NTag, NButton, NSpace } from 'naive-ui'
import type { DataTableColumns, FormRules, FormInst } from 'naive-ui'
import { adminApi } from '@/api/client'
import type { Plan } from '@/api/types'

const message = useMessage()
const dialog = useDialog()

const plans = ref<Plan[]>([])
const loading = ref(false)
const showModal = ref(false)
const saving = ref(false)
const editingPlan = ref<Plan | null>(null)
const formRef = ref<FormInst | null>(null)

function formatLimit(val: number): string {
  if (val === -1 || val === 0) return '\u221E'
  return String(val)
}

const columns: DataTableColumns<Plan> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: 'Slug', key: 'slug', width: 100 },
  { title: 'Name', key: 'name', width: 120 },
  {
    title: 'Price',
    key: 'price',
    width: 100,
    render(row) {
      return `$${row.price.toFixed(2)}`
    },
  },
  {
    title: 'Max Tunnels',
    key: 'max_tunnels',
    width: 110,
    render(row) { return formatLimit(row.max_tunnels) },
  },
  {
    title: 'Max Domains',
    key: 'max_domains',
    width: 110,
    render(row) { return formatLimit(row.max_domains) },
  },
  {
    title: 'Custom Domains',
    key: 'max_custom_domains',
    width: 130,
    render(row) { return formatLimit(row.max_custom_domains) },
  },
  {
    title: 'Max Tokens',
    key: 'max_tokens',
    width: 100,
    render(row) { return formatLimit(row.max_tokens) },
  },
  {
    title: 'Inspector',
    key: 'inspector_enabled',
    width: 90,
    render(row) {
      return h(NTag, { type: row.inspector_enabled ? 'success' : 'default', size: 'small' }, {
        default: () => row.inspector_enabled ? 'Yes' : 'No',
      })
    },
  },
  {
    title: 'Public',
    key: 'is_public',
    width: 80,
    render(row) {
      return h(NTag, { type: row.is_public ? 'success' : 'default', size: 'small' }, {
        default: () => row.is_public ? 'Yes' : 'No',
      })
    },
  },
  {
    title: 'Recommended',
    key: 'is_recommended',
    width: 110,
    render(row) {
      return h(NTag, { type: row.is_recommended ? 'info' : 'default', size: 'small' }, {
        default: () => row.is_recommended ? 'Yes' : 'No',
      })
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 140,
    render(row) {
      return h(NSpace, { size: 4 }, {
        default: () => [
          h(NButton, { size: 'small', type: 'info', quaternary: true, onClick: () => openEditModal(row) }, { default: () => 'Edit' }),
          h(NButton, { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) }, { default: () => 'Delete' }),
        ],
      })
    },
  },
]

const defaultFormValue = () => ({
  slug: '',
  name: '',
  price: 0,
  price_rub: 0,
  max_tunnels: 3,
  max_domains: 0,
  max_custom_domains: 0,
  max_tokens: 1,
  max_tunnels_per_token: 3,
  inspector_enabled: false,
  is_public: true,
  is_recommended: false,
  rate_limit_tcp: 0,
  rate_limit_udp: 0,
  rate_limit_http: 0,
  creem_product_id: '',
})

const formValue = ref(defaultFormValue())

const formRules: FormRules = {
  slug: [{ required: true, message: 'Slug is required', trigger: 'blur' }],
  name: [{ required: true, message: 'Name is required', trigger: 'blur' }],
  price: [{ required: true, type: 'number', min: 0, message: 'Price must be >= 0', trigger: 'blur' }],
  price_rub: [{ type: 'number', min: 0, message: 'Price must be >= 0', trigger: 'blur' }],
}

function openCreateModal() {
  editingPlan.value = null
  formValue.value = defaultFormValue()
  showModal.value = true
}

function openEditModal(plan: Plan) {
  editingPlan.value = plan
  formValue.value = {
    slug: plan.slug,
    name: plan.name,
    price: plan.price,
    price_rub: plan.price_rub ?? 0,
    max_tunnels: plan.max_tunnels,
    max_domains: plan.max_domains,
    max_custom_domains: plan.max_custom_domains,
    max_tokens: plan.max_tokens,
    max_tunnels_per_token: plan.max_tunnels_per_token,
    inspector_enabled: plan.inspector_enabled,
    is_public: plan.is_public,
    is_recommended: plan.is_recommended,
    rate_limit_tcp: plan.rate_limit_tcp,
    rate_limit_udp: plan.rate_limit_udp,
    rate_limit_http: plan.rate_limit_http,
    creem_product_id: plan.creem_product_id,
  }
  showModal.value = true
}

async function handleSave() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    if (editingPlan.value) {
      await adminApi.updatePlan(editingPlan.value.id, formValue.value)
      message.success('Plan updated')
    } else {
      await adminApi.createPlan(formValue.value as Omit<Plan, 'id'>)
      message.success('Plan created')
    }
    showModal.value = false
    await fetchPlans()
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to save plan')
  } finally {
    saving.value = false
  }
}

function handleDelete(plan: Plan) {
  dialog.error({
    title: 'Delete Plan',
    content: `Permanently delete plan "${plan.name}" (${plan.slug})? This cannot be undone.`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await adminApi.deletePlan(plan.id)
        message.success('Plan deleted')
        await fetchPlans()
      } catch (err: unknown) {
        const error = err as { response?: { data?: { error?: string } }; message?: string }
        message.error(error.response?.data?.error || error.message || 'Failed to delete plan')
      }
    },
  })
}

async function fetchPlans() {
  loading.value = true
  try {
    const { data } = await adminApi.listPlans()
    plans.value = data.plans || []
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } }; message?: string }
    message.error(error.response?.data?.error || error.message || 'Failed to load plans')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchPlans()
})
</script>
