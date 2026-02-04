<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Layout from '@/components/Layout.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
const { t } = useI18n()
</script>

<template>
  <Layout>
    <div class="max-w-md mx-auto mt-12">
      <Card class="p-8 text-center">
        <div class="w-16 h-16 mx-auto mb-6 rounded-full bg-green-900/30 flex items-center justify-center">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-green-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
            <polyline points="22 4 12 14.01 9 11.01" />
          </svg>
        </div>

        <h1 class="text-2xl font-bold mb-2">{{ t('payment.successTitle') }}</h1>
        <p class="text-muted-foreground mb-6">{{ t('payment.successMessage') }}</p>

        <div class="space-y-3">
          <Button class="w-full" @click="router.push('/profile')">
            {{ t('payment.goToProfile') }}
          </Button>
          <Button variant="outline" class="w-full" @click="router.push('/dashboard')">
            {{ t('payment.goToDashboard') }}
          </Button>
        </div>
      </Card>
    </div>
  </Layout>
</template>
