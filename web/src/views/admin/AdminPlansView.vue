<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { adminApi, type Plan } from '@/api/client'

const { t } = useI18n()

const plans = ref<Plan[]>([])
const loading = ref(true)
const error = ref('')
const showCreateModal = ref(false)
const editingId = ref<number | null>(null)
const deletingId = ref<number | null>(null)
const saving = ref(false)

const defaultForm = {
  slug: '',
  name: '',
  price: 0,
  max_tunnels: -1,
  max_domains: -1,
  max_custom_domains: 0,
  max_tokens: -1,
  max_tunnels_per_token: -1,
  inspector_enabled: false,
  is_public: false,
  is_recommended: false,
}

const form = ref({ ...defaultForm })

function resetForm() {
  form.value = { ...defaultForm }
}

function displayLimit(val: number): string {
  return val < 0 ? '∞' : String(val)
}

type PlanTier = 'free' | 'paid' | 'expensive' | 'unlimited'

function getPlanTier(plan: Plan): PlanTier {
  const allUnlimited = plan.max_tunnels < 0 && plan.max_domains < 0 && plan.max_tokens < 0
  if (allUnlimited) return 'unlimited'
  if (plan.price === 0) return 'free'
  if (plan.price >= 50) return 'expensive'
  return 'paid'
}

const tierStyles: Record<PlanTier, { border: string; badge: string; accent: string }> = {
  free: {
    border: 'border-border',
    badge: 'bg-muted text-muted-foreground',
    accent: 'text-muted-foreground',
  },
  paid: {
    border: 'border-blue-500/30',
    badge: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
    accent: 'text-blue-600 dark:text-blue-400',
  },
  expensive: {
    border: 'border-amber-500/30',
    badge: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
    accent: 'text-amber-600 dark:text-amber-400',
  },
  unlimited: {
    border: 'border-purple-500/30',
    badge: 'bg-purple-500/10 text-purple-600 dark:text-purple-400',
    accent: 'text-purple-600 dark:text-purple-400',
  },
}

const sortedPlans = computed(() =>
  [...plans.value].sort((a, b) => a.price - b.price)
)

async function loadPlans() {
  loading.value = true
  error.value = ''
  try {
    const response = await adminApi.listPlans()
    plans.value = response.data.plans || []
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.plans.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function createPlan() {
  saving.value = true
  error.value = ''
  try {
    await adminApi.createPlan(form.value)
    showCreateModal.value = false
    resetForm()
    await loadPlans()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.plans.failedToCreate')
  } finally {
    saving.value = false
  }
}

function startEdit(plan: Plan) {
  editingId.value = plan.id
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
}

function cancelEdit() {
  editingId.value = null
  resetForm()
}

async function saveEdit(id: number) {
  saving.value = true
  error.value = ''
  try {
    const { slug: _slug, ...data } = form.value
    await adminApi.updatePlan(id, data)
    editingId.value = null
    resetForm()
    await loadPlans()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.plans.failedToUpdate')
  } finally {
    saving.value = false
  }
}

function requestDelete(plan: Plan) {
  deletingId.value = plan.id
}

async function confirmDelete(plan: Plan) {
  error.value = ''
  try {
    await adminApi.deletePlan(plan.id)
    deletingId.value = null
    await loadPlans()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || t('admin.plans.failedToDelete')
  }
}

onMounted(loadPlans)
</script>

<template>
  <Layout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-foreground">{{ t('admin.plans.title') }}</h1>
          <p class="text-muted-foreground text-sm mt-1">{{ t('admin.plans.subtitle') }}</p>
        </div>
        <div class="flex gap-2">
          <Button variant="outline" :loading="loading" @click="loadPlans">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 2v6h-6"/><path d="M3 12a9 9 0 0 1 15-6.7L21 8"/><path d="M3 22v-6h6"/><path d="M21 12a9 9 0 0 1-15 6.7L3 16"/></svg>
            <span class="hidden sm:inline ml-1">{{ t('common.refresh') }}</span>
          </Button>
          <Button @click="showCreateModal = true; resetForm()">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            <span class="ml-1">{{ t('admin.plans.create') }}</span>
          </Button>
        </div>
      </div>

      <!-- Error -->
      <div v-if="error" class="bg-destructive/10 text-destructive p-3 rounded-lg text-sm">
        {{ error }}
      </div>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-16 text-muted-foreground">
        <svg class="h-8 w-8 animate-spin mx-auto mb-3 text-muted-foreground/50" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/></svg>
        {{ t('common.loading') }}
      </div>

      <!-- Empty -->
      <div v-else-if="plans.length === 0" class="text-center py-16">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-4 text-muted-foreground/30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>
        <p class="text-muted-foreground">{{ t('admin.plans.noPlans') }}</p>
      </div>

      <!-- Plan Cards Grid -->
      <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        <Card
          v-for="plan in sortedPlans"
          :key="plan.id"
          :class="[
            'p-5 transition-all duration-200 hover:shadow-md',
            tierStyles[getPlanTier(plan)].border,
            editingId === plan.id ? 'ring-2 ring-primary/50' : '',
          ]"
        >
          <!-- View Mode -->
          <template v-if="editingId !== plan.id">
            <!-- Header: name + slug + price -->
            <div class="flex items-start justify-between mb-4">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <h3 class="text-lg font-semibold text-foreground truncate">{{ plan.name }}</h3>
                  <span v-if="plan.is_public" class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                    {{ t('admin.plans.public') }}
                  </span>
                  <span v-if="plan.is_recommended" class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium bg-primary/10 text-primary">
                    {{ t('admin.plans.recommended') }}
                  </span>
                </div>
                <p class="text-xs font-mono text-muted-foreground mt-0.5">{{ plan.slug }}</p>
              </div>
              <span
                :class="[
                  'shrink-0 ml-3 inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold',
                  tierStyles[getPlanTier(plan)].badge,
                ]"
              >
                {{ plan.price === 0 ? t('admin.plans.free', 'Free') : `$${plan.price}` }}
              </span>
            </div>

            <!-- Stats Grid -->
            <div class="grid grid-cols-2 gap-x-4 gap-y-2.5 mb-4">
              <!-- Tunnels -->
              <div class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-muted-foreground shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="16" y="16" width="6" height="6" rx="1"/><rect x="2" y="16" width="6" height="6" rx="1"/><rect x="9" y="2" width="6" height="6" rx="1"/><path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3"/><path d="M12 12V8"/></svg>
                <span class="text-xs text-muted-foreground truncate">{{ t('admin.plans.maxTunnels') }}</span>
                <span class="text-sm font-medium text-foreground ml-auto">{{ displayLimit(plan.max_tunnels) }}</span>
              </div>
              <!-- Domains -->
              <div class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-muted-foreground shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
                <span class="text-xs text-muted-foreground truncate">{{ t('admin.plans.maxDomains') }}</span>
                <span class="text-sm font-medium text-foreground ml-auto">{{ displayLimit(plan.max_domains) }}</span>
              </div>
              <!-- Custom Domains -->
              <div class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-muted-foreground shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                <span class="text-xs text-muted-foreground truncate">{{ t('admin.plans.maxCustomDomains') }}</span>
                <span class="text-sm font-medium text-foreground ml-auto">{{ displayLimit(plan.max_custom_domains) }}</span>
              </div>
              <!-- Tokens -->
              <div class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-muted-foreground shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
                <span class="text-xs text-muted-foreground truncate">{{ t('admin.plans.maxTokens') }}</span>
                <span class="text-sm font-medium text-foreground ml-auto">{{ displayLimit(plan.max_tokens) }}</span>
              </div>
              <!-- Tunnels per Token -->
              <div class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-muted-foreground shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><line x1="19" y1="8" x2="19" y2="14"/><line x1="22" y1="11" x2="16" y2="11"/></svg>
                <span class="text-xs text-muted-foreground truncate">{{ t('admin.plans.tunnelsPerToken') }}</span>
                <span class="text-sm font-medium text-foreground ml-auto">{{ displayLimit(plan.max_tunnels_per_token) }}</span>
              </div>
              <!-- Inspector -->
              <div class="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-muted-foreground shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
                <span class="text-xs text-muted-foreground truncate">{{ t('admin.plans.inspector') }}</span>
                <span
                  :class="[
                    'ml-auto inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                    plan.inspector_enabled
                      ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                      : 'bg-muted text-muted-foreground',
                  ]"
                >
                  {{ plan.inspector_enabled ? t('admin.plans.yes') : t('admin.plans.no') }}
                </span>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex items-center gap-2 pt-3 border-t border-border">
              <Button variant="outline" size="xs" @click="startEdit(plan)" class="flex-1">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                <span class="ml-1">{{ t('admin.plans.editTitle') }}</span>
              </Button>
              <template v-if="deletingId === plan.id">
                <Button variant="destructive" size="xs" @click="confirmDelete(plan)">
                  {{ t('common.delete') }}
                </Button>
                <Button variant="ghost" size="xs" @click="deletingId = null">
                  {{ t('common.cancel') }}
                </Button>
              </template>
              <Button v-else variant="ghost" size="xs" @click="requestDelete(plan)">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 text-destructive" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </Button>
            </div>
          </template>

          <!-- Edit Mode (inline) -->
          <template v-else>
            <div class="space-y-3">
              <h3 class="text-sm font-semibold text-foreground mb-3">{{ t('admin.plans.editTitle') }}</h3>

              <div class="space-y-1">
                <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.slug') }}</label>
                <p class="text-sm font-mono text-muted-foreground bg-muted/50 rounded px-2 py-1.5">{{ form.slug }}</p>
              </div>

              <div class="space-y-1">
                <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.name') }}</label>
                <Input v-model="form.name" :placeholder="t('admin.plans.namePlaceholder')" />
              </div>

              <div class="space-y-1">
                <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.price') }}</label>
                <Input v-model="form.price" type="number" />
              </div>

              <div class="grid grid-cols-2 gap-2">
                <div class="space-y-1">
                  <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxTunnels') }}</label>
                  <Input v-model="form.max_tunnels" type="number" />
                </div>
                <div class="space-y-1">
                  <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxDomains') }}</label>
                  <Input v-model="form.max_domains" type="number" />
                </div>
                <div class="space-y-1">
                  <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxCustomDomains') }}</label>
                  <Input v-model="form.max_custom_domains" type="number" />
                </div>
                <div class="space-y-1">
                  <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxTokens') }}</label>
                  <Input v-model="form.max_tokens" type="number" />
                </div>
                <div class="space-y-1">
                  <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.tunnelsPerToken') }}</label>
                  <Input v-model="form.max_tunnels_per_token" type="number" />
                </div>
                <div class="space-y-1">
                  <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.inspector') }}</label>
                  <label class="flex items-center gap-2 h-10 px-3 rounded-lg border border-input bg-background cursor-pointer">
                    <input v-model="form.inspector_enabled" type="checkbox" class="rounded" />
                    <span class="text-sm">{{ t('admin.plans.inspectorEnabled') }}</span>
                  </label>
                </div>
              </div>

              <!-- Visibility settings -->
              <div class="grid grid-cols-2 gap-2">
                <label class="flex items-center gap-2 h-10 px-3 rounded-lg border border-input bg-background cursor-pointer">
                  <input v-model="form.is_public" type="checkbox" class="rounded" />
                  <span class="text-sm">{{ t('admin.plans.isPublic') }}</span>
                </label>
                <label class="flex items-center gap-2 h-10 px-3 rounded-lg border border-input bg-background cursor-pointer">
                  <input v-model="form.is_recommended" type="checkbox" class="rounded" />
                  <span class="text-sm">{{ t('admin.plans.isRecommended') }}</span>
                </label>
              </div>

              <p class="text-xs text-muted-foreground">{{ t('admin.plans.unlimitedHint') }}</p>

              <div class="flex gap-2 pt-2">
                <Button size="sm" :loading="saving" @click="saveEdit(plan.id)" class="flex-1">
                  {{ t('common.save') }}
                </Button>
                <Button variant="outline" size="sm" @click="cancelEdit" class="flex-1">
                  {{ t('common.cancel') }}
                </Button>
              </div>
            </div>
          </template>
        </Card>
      </div>

      <!-- Create Modal -->
      <Teleport to="body">
        <Transition
          enter-active-class="transition ease-out duration-200"
          enter-from-class="opacity-0"
          enter-to-class="opacity-100"
          leave-active-class="transition ease-in duration-150"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0"
        >
          <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div class="fixed inset-0 bg-black/50" @click="showCreateModal = false" />
            <Card class="relative z-10 w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
              <h2 class="text-lg font-semibold text-foreground mb-4">{{ t('admin.plans.createTitle') }}</h2>

              <form @submit.prevent="createPlan" class="space-y-3">
                <div class="grid grid-cols-2 gap-3">
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.slug') }}</label>
                    <Input v-model="form.slug" required :placeholder="t('admin.plans.slugPlaceholder')" />
                  </div>
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.name') }}</label>
                    <Input v-model="form.name" required :placeholder="t('admin.plans.namePlaceholder')" />
                  </div>
                </div>

                <div class="space-y-1">
                  <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.price') }}</label>
                  <Input v-model="form.price" type="number" />
                </div>

                <div class="grid grid-cols-2 gap-3">
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxTunnels') }}</label>
                    <Input v-model="form.max_tunnels" type="number" />
                  </div>
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxDomains') }}</label>
                    <Input v-model="form.max_domains" type="number" />
                  </div>
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxCustomDomains') }}</label>
                    <Input v-model="form.max_custom_domains" type="number" />
                  </div>
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.maxTokens') }}</label>
                    <Input v-model="form.max_tokens" type="number" />
                  </div>
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.tunnelsPerToken') }}</label>
                    <Input v-model="form.max_tunnels_per_token" type="number" />
                  </div>
                  <div class="space-y-1">
                    <label class="text-xs font-medium text-muted-foreground">{{ t('admin.plans.inspector') }}</label>
                    <label class="flex items-center gap-2 h-10 px-3 rounded-lg border border-input bg-background cursor-pointer">
                      <input v-model="form.inspector_enabled" type="checkbox" class="rounded" />
                      <span class="text-sm">{{ t('admin.plans.inspectorEnabled') }}</span>
                    </label>
                  </div>
                </div>

                <!-- Visibility settings -->
                <div class="grid grid-cols-2 gap-3">
                  <label class="flex items-center gap-2 h-10 px-3 rounded-lg border border-input bg-background cursor-pointer">
                    <input v-model="form.is_public" type="checkbox" class="rounded" />
                    <span class="text-sm">{{ t('admin.plans.isPublic') }}</span>
                  </label>
                  <label class="flex items-center gap-2 h-10 px-3 rounded-lg border border-input bg-background cursor-pointer">
                    <input v-model="form.is_recommended" type="checkbox" class="rounded" />
                    <span class="text-sm">{{ t('admin.plans.isRecommended') }}</span>
                  </label>
                </div>

                <p class="text-xs text-muted-foreground">{{ t('admin.plans.unlimitedHint') }}</p>

                <div class="flex gap-2 pt-2">
                  <Button type="submit" :loading="saving" class="flex-1">
                    {{ t('admin.plans.create') }}
                  </Button>
                  <Button variant="outline" type="button" @click="showCreateModal = false; resetForm()" class="flex-1">
                    {{ t('common.cancel') }}
                  </Button>
                </div>
              </form>
            </Card>
          </div>
        </Transition>
      </Teleport>
    </div>
  </Layout>
</template>
