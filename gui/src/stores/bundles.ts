import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as BundleService from '@/wailsjs/wailsjs/go/gui/BundleService'
import { storage } from '@/wailsjs/wailsjs/go/models'
import type { TunnelType } from '@/types'

export interface Bundle {
  id: number
  name: string
  type: TunnelType
  localPort: number
  subdomain?: string
  remotePort?: number
  autoConnect: boolean
  createdAt?: string
  updatedAt?: string
}

export interface CreateBundleInput {
  name: string
  type: TunnelType
  localPort: number
  subdomain?: string
  remotePort?: number
  autoConnect: boolean
}

export const useBundlesStore = defineStore('bundles', () => {
  const bundles = ref<Bundle[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function loadBundles(): Promise<void> {
    isLoading.value = true
    try {
      const result = await BundleService.List()
      bundles.value = result.map((b: storage.Bundle) => ({
        id: b.id,
        name: b.name,
        type: b.type as TunnelType,
        localPort: b.local_port,
        subdomain: b.subdomain,
        remotePort: b.remote_port,
        autoConnect: b.auto_connect,
        createdAt: b.created_at,
        updatedAt: b.updated_at,
      }))
    } catch (e) {
      console.error('Failed to load bundles:', e)
    } finally {
      isLoading.value = false
    }
  }

  async function createBundle(input: CreateBundleInput): Promise<Bundle | null> {
    isLoading.value = true
    error.value = null

    try {
      const bundle = new storage.Bundle({
        id: 0,
        name: input.name,
        type: input.type,
        local_port: input.localPort,
        subdomain: input.subdomain,
        remote_port: input.remotePort,
        auto_connect: input.autoConnect,
      })

      const result = await BundleService.Create(bundle)

      const newBundle: Bundle = {
        id: result.id,
        name: result.name,
        type: result.type as TunnelType,
        localPort: result.local_port,
        subdomain: result.subdomain,
        remotePort: result.remote_port,
        autoConnect: result.auto_connect,
        createdAt: result.created_at,
        updatedAt: result.updated_at,
      }

      bundles.value.push(newBundle)
      return newBundle
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create bundle'
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function updateBundle(bundle: Bundle): Promise<boolean> {
    try {
      const storageBundle = new storage.Bundle({
        id: bundle.id,
        name: bundle.name,
        type: bundle.type,
        local_port: bundle.localPort,
        subdomain: bundle.subdomain,
        remote_port: bundle.remotePort,
        auto_connect: bundle.autoConnect,
      })

      await BundleService.Update(storageBundle)

      const index = bundles.value.findIndex(b => b.id === bundle.id)
      if (index !== -1) {
        bundles.value[index] = { ...bundle, updatedAt: new Date().toISOString() }
      }
      return true
    } catch (e) {
      console.error('Failed to update bundle:', e)
      return false
    }
  }

  async function deleteBundle(id: number): Promise<boolean> {
    try {
      await BundleService.Delete(id)
      bundles.value = bundles.value.filter(b => b.id !== id)
      return true
    } catch (e) {
      console.error('Failed to delete bundle:', e)
      return false
    }
  }

  async function connectBundle(id: number): Promise<boolean> {
    try {
      await BundleService.Connect(id)
      return true
    } catch (e) {
      console.error('Failed to connect bundle:', e)
      error.value = e instanceof Error ? e.message : 'Failed to connect'
      return false
    }
  }

  async function exportBundles(): Promise<string> {
    try {
      return await BundleService.Export()
    } catch (e) {
      console.error('Failed to export bundles:', e)
      return ''
    }
  }

  async function importBundles(data: string): Promise<boolean> {
    try {
      await BundleService.Import(data)
      await loadBundles() // Reload after import
      return true
    } catch (e) {
      console.error('Failed to import bundles:', e)
      return false
    }
  }

  return {
    bundles,
    isLoading,
    error,
    loadBundles,
    createBundle,
    updateBundle,
    deleteBundle,
    connectBundle,
    exportBundles,
    importBundles,
  }
})
