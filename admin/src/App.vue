<template>
  <div class="min-h-screen bg-background text-foreground font-sans">
    <router-view />
    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import ToastContainer from '@/components/ToastContainer.vue'
</script>
