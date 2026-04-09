<script setup lang="ts">
import { ref, onMounted, onUnmounted, type Component } from 'vue'
import { cn } from '@/lib/utils'

interface DropdownItem {
  key: string
  label: string
  icon?: Component
  destructive?: boolean
  disabled?: boolean
  divider?: boolean
}

interface Props {
  items: DropdownItem[]
  class?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  select: [key: string]
}>()

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)

function toggle() {
  isOpen.value = !isOpen.value
}

function selectItem(item: DropdownItem) {
  if (item.disabled) return
  emit('select', item.key)
  isOpen.value = false
}

function onClickOutside(e: MouseEvent) {
  if (containerRef.value && !containerRef.value.contains(e.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('mousedown', onClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', onClickOutside)
})
</script>

<template>
  <div ref="containerRef" :class="cn('relative inline-flex', props.class)">
    <!-- Trigger -->
    <div @click="toggle">
      <slot />
    </div>

    <!-- Menu -->
    <Transition name="dropdown">
      <div
        v-if="isOpen"
        class="absolute right-0 top-full z-50 mt-1 min-w-[160px] overflow-hidden rounded-lg border border-border bg-card shadow-xl py-1"
      >
        <template v-for="item in items" :key="item.key">
          <!-- Divider -->
          <div v-if="item.divider" class="my-1 h-px bg-border" />

          <!-- Item -->
          <button
            v-else
            type="button"
            :disabled="item.disabled"
            class="flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors duration-150 disabled:opacity-50 disabled:cursor-not-allowed"
            :class="[
              item.destructive
                ? 'text-destructive hover:bg-destructive/10'
                : 'text-foreground hover:bg-surface-elevated',
            ]"
            @click="selectItem(item)"
          >
            <component
              v-if="item.icon"
              :is="item.icon"
              class="h-4 w-4 flex-shrink-0"
            />
            <span>{{ item.label }}</span>
          </button>
        </template>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.dropdown-enter-active {
  transition: all 0.15s cubic-bezier(0.16, 1, 0.3, 1);
}

.dropdown-leave-active {
  transition: all 0.1s ease-in;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.95);
}
</style>
