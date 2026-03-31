<script setup lang="ts">
import { DialogTrigger } from 'radix-vue'
</script>

<template>
  <DialogTrigger as-child>
    <slot />
  </DialogTrigger>
</template>
