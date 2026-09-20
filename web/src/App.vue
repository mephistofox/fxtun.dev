<script setup lang="ts">
import { RouterView } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useThemeStore } from './stores/theme'
import { onMounted } from 'vue'

const authStore = useAuthStore()
const themeStore = useThemeStore()

onMounted(() => {
  authStore.init()
  themeStore.init()
})
</script>

<template>
  <!--
    Landings share one component and read `meta.ns` once at setup, so without a
    per-path key a hop between them changes the URL and keeps the old page.
  -->
  <RouterView v-slot="{ Component, route }">
    <component :is="Component" :key="route.meta.ns ? route.path : undefined" />
  </RouterView>
</template>
