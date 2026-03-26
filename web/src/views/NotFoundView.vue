<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useSeoMeta } from '@unhead/vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

useSeoMeta({
  title: () => t('notFound.title'),
  robots: 'noindex, nofollow',
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background text-foreground p-4">
    <div class="text-center max-w-md">
      <p class="text-8xl font-display font-bold text-primary/20 mb-4">404</p>
      <h1 class="text-2xl font-semibold mb-2">{{ t('notFound.heading') }}</h1>
      <p class="text-muted-foreground mb-8">{{ t('notFound.description') }}</p>
      <RouterLink
        to="/"
        class="inline-flex items-center gap-2 px-6 py-3 text-sm font-medium rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
      >
        <svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        {{ t('notFound.backHome') }}
      </RouterLink>
    </div>
  </div>
</template>
