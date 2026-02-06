import { ref, computed } from 'vue'
import { getLocale } from '@/i18n'

const FALLBACK_RATE = 75 // Fallback rate if API fails

const usdToRubRate = ref<number | null>(null)
const isLoading = ref(false)
const isRuLocale = computed(() => {
  return getLocale() === 'ru'
})

async function fetchRate(): Promise<number> {
  try {
    const response = await fetch('/api/exchange-rate')
    if (!response.ok) {
      throw new Error('Failed to fetch rate')
    }
    const data = await response.json()
    if (data.rate && typeof data.rate === 'number') {
      return data.rate
    }
    throw new Error('Invalid rate data')
  } catch {
    return FALLBACK_RATE
  }
}

export function useCurrencyRate() {
  const loadRate = async () => {
    if (usdToRubRate.value !== null || isLoading.value) {
      return
    }

    isLoading.value = true
    try {
      usdToRubRate.value = await fetchRate()
    } finally {
      isLoading.value = false
    }
  }

  const convertToRub = (usdPrice: number): number => {
    if (usdToRubRate.value === null) {
      return usdPrice * FALLBACK_RATE
    }
    return Math.round(usdPrice * usdToRubRate.value)
  }

  const formatPrice = (usdPrice: number): string => {
    if (usdPrice === 0) {
      return ''
    }

    if (isRuLocale.value) {
      const rubPrice = convertToRub(usdPrice)
      return `${rubPrice} ₽`
    }

    return `$${usdPrice}`
  }

  return {
    isRuLocale,
    usdToRubRate,
    isLoading,
    loadRate,
    convertToRub,
    formatPrice
  }
}
