import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as UpdateService from '@/wailsjs/wailsjs/go/gui/UpdateService'

export const useUpdateStore = defineStore('updates', () => {
  const available = ref(false)
  const clientVersion = ref('')
  const serverVersion = ref('')
  const downloadURL = ref('')
  const checking = ref(false)
  const downloading = ref(false)
  const error = ref<string | null>(null)

  async function checkUpdate() {
    checking.value = true
    error.value = null
    try {
      const info = await UpdateService.CheckUpdate()
      available.value = info.available
      clientVersion.value = info.client_version
      serverVersion.value = info.server_version
      downloadURL.value = info.download_url
    } catch (e: any) {
      error.value = e?.message || String(e)
    } finally {
      checking.value = false
    }
  }

  async function downloadUpdate() {
    if (!downloadURL.value) return
    downloading.value = true
    error.value = null
    try {
      await UpdateService.DownloadUpdate(downloadURL.value)
    } catch (e: any) {
      error.value = e?.message || String(e)
    } finally {
      downloading.value = false
    }
  }

  return {
    available, clientVersion, serverVersion, downloadURL,
    checking, downloading, error,
    checkUpdate, downloadUpdate,
  }
})
