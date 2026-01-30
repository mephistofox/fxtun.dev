<template>
  <div class="bg-gray-900 rounded border border-gray-800 overflow-hidden">
    <table class="w-full text-sm">
      <tbody>
        <tr v-for="(values, name) in headers" :key="name" class="border-b border-gray-800/50 last:border-0">
          <td class="px-3 py-1.5 text-gray-400 font-mono whitespace-nowrap align-top">{{ name }}</td>
          <td class="px-3 py-1.5 text-gray-200 font-mono break-all">{{ values?.join(', ') }}</td>
        </tr>
        <tr v-if="!headers || Object.keys(headers).length === 0">
          <td class="px-3 py-2 text-gray-600 text-center" colspan="2">No headers</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  headers: Record<string, string[]> | null
}>()
</script>
