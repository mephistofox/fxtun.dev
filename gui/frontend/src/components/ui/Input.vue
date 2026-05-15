<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Props {
  modelValue?: string | number
  type?: string
  placeholder?: string
  disabled?: boolean
  class?: string
  id?: string
  maxlength?: string | number
  phone?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  modelValue: '',
  disabled: false,
  phone: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function formatPhone(value: string): string {
  const digits = value.replace(/\D/g, '')

  let normalized = digits
  if (normalized.startsWith('8') && normalized.length >= 1) {
    normalized = '7' + normalized.slice(1)
  } else if (!normalized.startsWith('7') && normalized.length > 0) {
    normalized = '7' + normalized
  }

  if (normalized.length > 11) {
    normalized = normalized.slice(0, 11)
  }

  let result = ''
  if (normalized.length >= 1) result = '+' + normalized[0]
  if (normalized.length >= 2) result += ' (' + normalized.slice(1, Math.min(4, normalized.length))
  if (normalized.length >= 4) result += ')'
  if (normalized.length >= 5) result += ' ' + normalized.slice(4, Math.min(7, normalized.length))
  if (normalized.length >= 8) result += ' ' + normalized.slice(7, Math.min(9, normalized.length))
  if (normalized.length >= 10) result += ' ' + normalized.slice(9, 11)

  return result
}

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  let value = target.value

  if (props.phone) {
    value = formatPhone(value)
    target.value = value
  }

  emit('update:modelValue', value)
}

const classes = computed(() =>
  cn(
    'flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm transition-colors',
    'placeholder:text-muted-foreground',
    'hover:border-muted-foreground/50',
    'focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-background focus:border-primary',
    'disabled:cursor-not-allowed disabled:opacity-50',
    props.class
  )
)
</script>

<template>
  <input
    :id="id"
    :class="classes"
    :type="phone ? 'tel' : type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :maxlength="maxlength"
    @input="handleInput"
  />
</template>
