import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as CustomDomainService from '@/wailsjs/wailsjs/go/gui/CustomDomainService'

export interface CustomDomain {
  id: number
  domain: string
  targetSubdomain: string
  verified: boolean
  verifiedAt?: string
  createdAt: string
}

export const useCustomDomainsStore = defineStore('customDomains', () => {
  const domains = ref<CustomDomain[]>([])
  const maxDomains = ref(5)
  const baseDomain = ref('')
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function loadDomains(): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      const result = await CustomDomainService.List()
      domains.value = (result.domains || []).map((d) => ({
        id: d.id,
        domain: d.domain,
        targetSubdomain: d.target_subdomain,
        verified: d.verified,
        verifiedAt: d.verified_at,
        createdAt: d.created_at,
      }))
      maxDomains.value = result.max_domains || 5
      baseDomain.value = result.base_domain || ''
    } catch (e) {
      console.error('Failed to load custom domains:', e)
      error.value = e instanceof Error ? e.message : 'Failed to load custom domains'
    } finally {
      isLoading.value = false
    }
  }

  async function addDomain(domain: string, targetSubdomain: string): Promise<CustomDomain | null> {
    isLoading.value = true
    error.value = null
    try {
      const result = await CustomDomainService.Add(domain, targetSubdomain)
      const cd: CustomDomain = {
        id: result.id,
        domain: result.domain,
        targetSubdomain: result.target_subdomain,
        verified: result.verified,
        verifiedAt: result.verified_at,
        createdAt: result.created_at,
      }
      domains.value.push(cd)
      return cd
    } catch (e) {
      console.error('Failed to add custom domain:', e)
      error.value = e instanceof Error ? e.message : 'Failed to add custom domain'
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function deleteDomain(id: number): Promise<boolean> {
    try {
      await CustomDomainService.Delete(id)
      domains.value = domains.value.filter(d => d.id !== id)
      return true
    } catch (e) {
      console.error('Failed to delete custom domain:', e)
      error.value = e instanceof Error ? e.message : 'Failed to delete custom domain'
      return false
    }
  }

  async function verifyDomain(id: number): Promise<{ verified: boolean; error?: string } | null> {
    try {
      const result = await CustomDomainService.Verify(id)
      if (result.verified) {
        const domain = domains.value.find(d => d.id === id)
        if (domain) {
          domain.verified = true
          domain.verifiedAt = new Date().toISOString()
        }
      }
      return { verified: result.verified, error: result.error }
    } catch (e) {
      console.error('Failed to verify custom domain:', e)
      return null
    }
  }

  return {
    domains,
    maxDomains,
    baseDomain,
    isLoading,
    error,
    loadDomains,
    addDomain,
    deleteDomain,
    verifyDomain,
  }
})
