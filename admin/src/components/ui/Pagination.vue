<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

interface Props {
  page: number
  total: number
  pageSize: number
  pageSizes?: number[]
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  pageSizes: () => [10, 20, 50, 100],
})

const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))

const rangeStart = computed(() => {
  if (props.total === 0) return 0
  return (props.page - 1) * props.pageSize + 1
})

const rangeEnd = computed(() => Math.min(props.page * props.pageSize, props.total))

const canPrev = computed(() => props.page > 1)
const canNext = computed(() => props.page < totalPages.value)

const visiblePages = computed(() => {
  const pages: (number | 'ellipsis')[] = []
  const total = totalPages.value
  const current = props.page

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
    return pages
  }

  pages.push(1)

  if (current > 3) {
    pages.push('ellipsis')
  }

  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }

  if (current < total - 2) {
    pages.push('ellipsis')
  }

  pages.push(total)

  return pages
})

function goToPage(page: number) {
  if (page < 1 || page > totalPages.value || page === props.page) return
  emit('update:page', page)
}

function changePageSize(event: Event) {
  const target = event.target as HTMLSelectElement
  const newSize = Number(target.value)
  emit('update:pageSize', newSize)
  emit('update:page', 1)
}
</script>

<template>
  <div
    :class="cn(
      'flex flex-col sm:flex-row items-center justify-between gap-4 text-sm',
      props.class,
    )"
  >
    <!-- Info -->
    <div class="text-muted-foreground">
      <template v-if="total > 0">
        Показано {{ rangeStart }}-{{ rangeEnd }} из {{ total }}
      </template>
      <template v-else>
        Нет записей
      </template>
    </div>

    <div class="flex items-center gap-4">
      <!-- Page size selector -->
      <div class="flex items-center gap-2">
        <span class="text-muted-foreground whitespace-nowrap">На странице:</span>
        <select
          :value="pageSize"
          class="h-8 rounded-lg border border-input bg-background px-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          @change="changePageSize"
        >
          <option v-for="size in pageSizes" :key="size" :value="size">
            {{ size }}
          </option>
        </select>
      </div>

      <!-- Page navigation -->
      <div class="flex items-center gap-1">
        <!-- Prev -->
        <button
          type="button"
          :disabled="!canPrev"
          class="inline-flex h-8 w-8 items-center justify-center rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed hover:bg-surface-elevated text-muted-foreground hover:text-foreground"
          @click="goToPage(page - 1)"
        >
          <ChevronLeft class="h-4 w-4" />
        </button>

        <!-- Page numbers -->
        <template v-for="(p, idx) in visiblePages" :key="idx">
          <span
            v-if="p === 'ellipsis'"
            class="inline-flex h-8 w-8 items-center justify-center text-muted-foreground"
          >
            ...
          </span>
          <button
            v-else
            type="button"
            class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-sm font-medium transition-colors"
            :class="[
              p === page
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-surface-elevated hover:text-foreground',
            ]"
            @click="goToPage(p as number)"
          >
            {{ p }}
          </button>
        </template>

        <!-- Next -->
        <button
          type="button"
          :disabled="!canNext"
          class="inline-flex h-8 w-8 items-center justify-center rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed hover:bg-surface-elevated text-muted-foreground hover:text-foreground"
          @click="goToPage(page + 1)"
        >
          <ChevronRight class="h-4 w-4" />
        </button>
      </div>
    </div>
  </div>
</template>
