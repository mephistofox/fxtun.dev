<script setup lang="ts">
import { computed } from 'vue'
import { Inbox } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

export interface Column {
  key: string
  title: string
  width?: string
  align?: 'left' | 'center' | 'right'
}

interface Props {
  columns: Column[]
  data: any[]
  loading?: boolean
  selectable?: boolean
  selectedKeys?: (string | number)[]
  rowKey?: string
  emptyText?: string
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  selectable: false,
  selectedKeys: () => [],
  rowKey: 'id',
  emptyText: 'Нет данных',
})

const emit = defineEmits<{
  'update:selectedKeys': [keys: (string | number)[]]
}>()

const allSelected = computed(() => {
  if (props.data.length === 0) return false
  return props.data.every((row) => props.selectedKeys.includes(row[props.rowKey]))
})

const someSelected = computed(() => {
  if (props.data.length === 0) return false
  return (
    props.selectedKeys.length > 0 &&
    props.data.some((row) => props.selectedKeys.includes(row[props.rowKey]))
  )
})

function toggleAll() {
  if (allSelected.value) {
    emit('update:selectedKeys', [])
  } else {
    emit(
      'update:selectedKeys',
      props.data.map((row) => row[props.rowKey]),
    )
  }
}

function toggleRow(row: any) {
  const key = row[props.rowKey]
  const index = props.selectedKeys.indexOf(key)
  const newKeys = [...props.selectedKeys]
  if (index === -1) {
    newKeys.push(key)
  } else {
    newKeys.splice(index, 1)
  }
  emit('update:selectedKeys', newKeys)
}

function isSelected(row: any): boolean {
  return props.selectedKeys.includes(row[props.rowKey])
}

function alignClass(align?: string): string {
  switch (align) {
    case 'center':
      return 'text-center'
    case 'right':
      return 'text-right'
    default:
      return 'text-left'
  }
}
</script>

<template>
  <div
    :class="cn(
      'relative overflow-visible rounded-xl border border-border bg-card backdrop-blur-xl',
      props.class,
    )"
  >
    <!-- Loading overlay -->
    <div
      v-if="loading"
      class="absolute inset-0 z-10 flex items-center justify-center bg-background/60 backdrop-blur-sm"
    >
      <svg
        class="h-6 w-6 animate-spin text-primary"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        />
      </svg>
    </div>

    <!-- Table -->
    <div class="overflow-x-auto">
      <table class="w-full">
        <!-- Header -->
        <thead>
          <tr class="bg-surface-elevated">
            <th
              v-if="selectable"
              class="w-12 px-4 py-3"
            >
              <label class="flex items-center justify-center cursor-pointer">
                <input
                  type="checkbox"
                  :checked="allSelected"
                  :indeterminate="someSelected && !allSelected"
                  class="h-4 w-4 rounded border-input accent-primary cursor-pointer"
                  @change="toggleAll"
                />
              </label>
            </th>
            <th
              v-for="col in columns"
              :key="col.key"
              :style="col.width ? { width: col.width } : undefined"
              :class="[
                'px-4 py-3 text-xs font-medium uppercase tracking-wider text-muted-foreground',
                alignClass(col.align),
              ]"
            >
              {{ col.title }}
            </th>
          </tr>
        </thead>

        <!-- Body -->
        <tbody>
          <tr
            v-for="(row, idx) in data"
            :key="row[rowKey] ?? idx"
            class="border-b border-border transition-colors duration-150 hover:bg-surface-elevated/50"
            :class="{ 'bg-primary/5': selectable && isSelected(row) }"
          >
            <td
              v-if="selectable"
              class="w-12 px-4 py-3"
            >
              <label class="flex items-center justify-center cursor-pointer">
                <input
                  type="checkbox"
                  :checked="isSelected(row)"
                  class="h-4 w-4 rounded border-input accent-primary cursor-pointer"
                  @change="toggleRow(row)"
                />
              </label>
            </td>
            <td
              v-for="col in columns"
              :key="col.key"
              :class="[
                'px-4 py-3 text-sm text-foreground',
                alignClass(col.align),
              ]"
            >
              <slot :name="col.key" :row="row" :value="row[col.key]">
                {{ row[col.key] }}
              </slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Empty state -->
    <div
      v-if="!loading && data.length === 0"
      class="flex flex-col items-center justify-center py-16 text-muted-foreground"
    >
      <Inbox class="h-10 w-10 mb-3 opacity-50" />
      <p class="text-sm">{{ emptyText }}</p>
    </div>
  </div>
</template>
