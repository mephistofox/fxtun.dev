<script setup lang="ts">
import { toasts, dismissToast } from '@/composables/useToast'
import Toast from './Toast.vue'
</script>

<template>
  <div class="fixed bottom-4 right-4 z-[100] flex max-h-screen w-full flex-col-reverse gap-2 sm:max-w-[420px]">
    <TransitionGroup name="list">
      <Toast
        v-for="t in toasts"
        :key="t.id"
        :id="t.id"
        :title="t.title"
        :description="t.description"
        :variant="t.variant"
        @dismiss="dismissToast"
      />
    </TransitionGroup>
  </div>
</template>
