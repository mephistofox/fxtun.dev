<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'

export interface BreadcrumbItem {
  name: string
  path: string
}

defineProps<{ items: BreadcrumbItem[] }>()

const { t } = useI18n()
</script>

<template>
  <nav aria-label="Breadcrumb" class="container mx-auto px-4 pt-4 pb-2">
    <ol class="flex items-center gap-1.5 text-sm text-muted-foreground">
      <li>
        <RouterLink to="/" class="hover:text-foreground transition-colors">
          {{ t('common.breadcrumbHome') }}
        </RouterLink>
      </li>
      <li v-for="(item, index) in items" :key="item.path" class="flex items-center gap-1.5">
        <span class="text-muted-foreground/50" aria-hidden="true">›</span>
        <RouterLink
          v-if="index < items.length - 1"
          :to="item.path"
          class="hover:text-foreground transition-colors"
        >
          {{ item.name }}
        </RouterLink>
        <span v-else class="text-foreground/70">{{ item.name }}</span>
      </li>
    </ol>
  </nav>
</template>
