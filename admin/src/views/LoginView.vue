<template>
  <div
    style="
      height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background-color: #101014;
    "
  >
    <n-card style="width: 400px" :bordered="true">
      <template #header>
        <div style="text-align: center">
          <n-h2 style="margin: 0">fxTunnel Admin</n-h2>
        </div>
      </template>
      <n-form ref="formRef" :model="formValue" :rules="rules" @submit.prevent="handleLogin">
        <n-form-item path="phone" label="Phone or Email">
          <n-input
            v-model:value="formValue.phone"
            placeholder="Phone or Email"
            :disabled="loading"
            @keyup.enter="handleLogin"
          />
        </n-form-item>
        <n-form-item path="password" label="Password">
          <n-input
            v-model:value="formValue.password"
            type="password"
            show-password-on="click"
            placeholder="Password"
            :disabled="loading"
            @keyup.enter="handleLogin"
          />
        </n-form-item>
        <n-form-item v-if="showTotp" path="totp_code" label="TOTP Code">
          <n-input
            v-model:value="formValue.totp_code"
            placeholder="6-digit code"
            :disabled="loading"
            :maxlength="6"
            @keyup.enter="handleLogin"
          />
        </n-form-item>
        <n-button
          type="primary"
          block
          :loading="loading"
          :disabled="loading"
          @click="handleLogin"
        >
          Login
        </n-button>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import type { FormRules } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const message = useMessage()
const authStore = useAuthStore()

const loading = ref(false)
const showTotp = ref(false)
const formValue = ref({
  phone: '',
  password: '',
  totp_code: '',
})

const rules: FormRules = {
  phone: [{ required: true, message: 'Phone or email is required', trigger: 'blur' }],
  password: [{ required: true, message: 'Password is required', trigger: 'blur' }],
}

async function handleLogin() {
  if (!formValue.value.phone || !formValue.value.password) {
    message.error('Please fill in all fields')
    return
  }

  loading.value = true
  try {
    await authStore.login(formValue.value.phone, formValue.value.password, formValue.value.totp_code || undefined)
    message.success('Login successful')
    router.push('/')
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string; code?: string } }; message?: string }
    const errorMessage =
      error.response?.data?.error || error.message || 'Login failed'
    if (error.response?.data?.code === 'totp_required') {
      showTotp.value = true
      message.info('Enter your TOTP code')
    } else {
      message.error(errorMessage)
    }
  } finally {
    loading.value = false
  }
}
</script>
