<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ChevronDown, Check } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

interface SelectOption {
  value: string | number
  label: string
}

interface Props {
  modelValue: string | number | null
  options: SelectOption[]
  placeholder?: string
  disabled?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: 'Выберите...',
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const isOpen = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)

const selectedLabel = computed(() => {
  const option = props.options.find((o) => o.value === props.modelValue)
  return option?.label ?? null
})

function toggle() {
  if (props.disabled) return
  isOpen.value = !isOpen.value
}

function select(option: SelectOption) {
  emit('update:modelValue', option.value)
  isOpen.value = false
}

function onClickOutside(e: MouseEvent) {
  const target = e.target as Node
  if (
    triggerRef.value &&
    !triggerRef.value.contains(target) &&
    dropdownRef.value &&
    !dropdownRef.value.contains(target)
  ) {
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
  <div :class="cn('relative', props.class)">
    <!-- Trigger -->
    <button
      ref="triggerRef"
      type="button"
      :disabled="disabled"
      class="flex h-10 w-full items-center justify-between rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"
      :class="{ 'border-primary/50': isOpen }"
      @click="toggle"
    >
      <span :class="{ 'text-muted-foreground': !selectedLabel }">
        {{ selectedLabel || placeholder }}
      </span>
      <ChevronDown
        class="h-4 w-4 text-muted-foreground transition-transform duration-200"
        :class="{ 'rotate-180': isOpen }"
      />
    </button>

    <!-- Dropdown -->
    <Transition name="select-dropdown">
      <div
        v-if="isOpen"
        ref="dropdownRef"
        class="absolute z-50 mt-1 w-full max-h-60 overflow-auto rounded-lg border border-border bg-card shadow-xl py-1"
      >
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          class="flex w-full items-center justify-between px-3 py-2 text-sm text-foreground transition-colors duration-150 hover:bg-primary/10 hover:text-primary"
          :class="{
            'bg-primary/10 text-primary': option.value === modelValue,
          }"
          @click="select(option)"
        >
          <span>{{ option.label }}</span>
          <Check
            v-if="option.value === modelValue"
            class="h-3.5 w-3.5 text-primary"
          />
        </button>
        <div
          v-if="options.length === 0"
          class="px-3 py-2 text-sm text-muted-foreground text-center"
        >
          Нет вариантов
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.select-dropdown-enter-active {
  transition: all 0.15s cubic-bezier(0.16, 1, 0.3, 1);
}

.select-dropdown-leave-active {
  transition: all 0.1s ease-in;
}

.select-dropdown-enter-from,
.select-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
