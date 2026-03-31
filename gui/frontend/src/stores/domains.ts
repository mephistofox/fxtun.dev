import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as DomainService from '@/wailsjs/wailsjs/go/gui/DomainService'
import { gui } from '@/wailsjs/wailsjs/go/models'

export interface Domain {
  id: number
  subdomain: string
  url: string
  createdAt: string
}

export const useDomainsStore = defineStore('domains', () => {
  const domains = ref<Domain[]>([])
  const maxDomains = ref(3)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function loadDomains(): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      const result = await DomainService.List()
      domains.value = (result.domains || []).map((d: gui.Domain) => ({
        id: d.id,
        subdomain: d.subdomain,
        url: d.url,
        createdAt: d.created_at,
      }))
      maxDomains.value = result.max_domains || 3
    } catch (e) {
      console.error('Failed to load domains:', e)
      error.value = e instanceof Error ? e.message : 'Failed to load domains'
    } finally {
      isLoading.value = false
    }
  }

  async function checkAvailability(subdomain: string): Promise<{ available: boolean; reason?: string } | null> {
    try {
      const result = await DomainService.Check(subdomain)
      return {
        available: result.available,
        reason: result.reason,
      }
    } catch (e) {
      console.error('Failed to check domain:', e)
      return null
    }
  }

  async function reserveDomain(subdomain: string): Promise<Domain | null> {
    isLoading.value = true
    error.value = null
    try {
      const result = await DomainService.Reserve(subdomain)
      const domain: Domain = {
        id: result.id,
        subdomain: result.subdomain,
        url: result.url,
        createdAt: result.created_at,
      }
      domains.value.push(domain)
      return domain
    } catch (e) {
      console.error('Failed to reserve domain:', e)
      error.value = e instanceof Error ? e.message : 'Failed to reserve domain'
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function releaseDomain(id: number): Promise<boolean> {
    try {
      await DomainService.Release(id)
      domains.value = domains.value.filter(d => d.id !== id)
      return true
    } catch (e) {
      console.error('Failed to release domain:', e)
      error.value = e instanceof Error ? e.message : 'Failed to release domain'
      return false
    }
  }

  return {
    domains,
    maxDomains,
    isLoading,
    error,
    loadDomains,
    checkAvailability,
    reserveDomain,
    releaseDomain,
  }
})
