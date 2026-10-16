export interface ProductAdStrategy {
  productId: string
  enabled?: boolean
  targetCost?: number | null // CPA ສູງສຸດ / ຈຸດຄຸ້ມທຶນ (Target CPA)
  grossMargin?: number | null // Gross profit margin %
  scaleMinPurchases?: number | null // ອໍເດີຂັ້ນຕ່ຳເພື່ອ Scale (default 3)
  scaleMaxCPA?: number | null // CPA ສູງສຸດທີ່ອະນຸຍາດໃຫ້ Scale
  scaleMaxDailyBudgetCap?: number | null // ເພດານງົບສູງສຸດຕໍ່ມື້ (Daily Budget Cap)
  stopMinSpend?: number | null // ໃຊ້ງົບຮອດເທົ່າໃດແລ້ວບໍ່ມີອໍເດີໃຫ້ຢຸດ (Stop-loss spend)
  updatedAt?: string
}

export function useProductAdStrategy() {
  const productStrategies = useState<Record<string, ProductAdStrategy>>('ads-product-strategies', () => ({}))

  function loadStrategies(adAccountId?: string) {
    if (!import.meta.client) return
    // Load global first
    const globalSaved = localStorage.getItem('ads-product-strategies:global')
    if (globalSaved) {
      try {
        const parsed = JSON.parse(globalSaved)
        productStrategies.value = { ...productStrategies.value, ...parsed }
      } catch {}
    }
    // Then load account-specific if available
    if (adAccountId) {
      const acctSaved = localStorage.getItem(`ads-product-strategies:${adAccountId}`)
      if (acctSaved) {
        try {
          const parsed = JSON.parse(acctSaved)
          productStrategies.value = { ...productStrategies.value, ...parsed }
        } catch {}
      }
    }
  }

  function saveStrategies(adAccountId?: string) {
    if (!import.meta.client) return
    const json = JSON.stringify(productStrategies.value)
    localStorage.setItem('ads-product-strategies:global', json)
    if (adAccountId) {
      localStorage.setItem(`ads-product-strategies:${adAccountId}`, json)
    }
  }

  function getStrategy(productId: string): ProductAdStrategy | undefined {
    return productStrategies.value[productId]
  }

  function setStrategy(productId: string, strategy: Partial<ProductAdStrategy>, adAccountId?: string) {
    productStrategies.value[productId] = {
      ...(productStrategies.value[productId] || { productId }),
      ...strategy,
      productId,
      updatedAt: new Date().toISOString(),
    }
    saveStrategies(adAccountId)
  }

  function autoCalculate(
    product: { id: string; price?: string; salePrice?: number | null; cost?: number | null; currency?: string },
    adCurrency = 'USD'
  ): ProductAdStrategy {
    let rawPrice = product.salePrice ?? 0
    if (!rawPrice && product.price) {
      const cleaned = String(product.price).replace(/[^0-9.]/g, '')
      const num = parseFloat(cleaned)
      if (!isNaN(num) && num > 0) rawPrice = num
    }
    if (!rawPrice) rawPrice = adCurrency === 'LAK' ? 250000 : 10

    let priceInAdCurr = rawPrice
    if (adCurrency === 'USD' && (product.currency === 'LAK' || rawPrice > 2000)) {
      priceInAdCurr = Math.round((rawPrice / 22000) * 100) / 100
    } else if (adCurrency === 'THB' && (product.currency === 'LAK' || rawPrice > 2000)) {
      priceInAdCurr = Math.round((rawPrice / 620) * 100) / 100
    } else if (adCurrency === 'LAK' && rawPrice < 200) {
      priceInAdCurr = Math.round(rawPrice * 22000)
    }

    const isLak = adCurrency === 'LAK' || priceInAdCurr > 2000
    const margin = 50
    const targetCpa = isLak ? Math.round(priceInAdCurr * 0.20 / 1000) * 1000 : Math.round(priceInAdCurr * 0.20 * 100) / 100
    const scaleCpa = isLak ? Math.round(priceInAdCurr * 0.25 / 1000) * 1000 : Math.round(priceInAdCurr * 0.25 * 100) / 100
    const stopSpend = isLak ? Math.round(priceInAdCurr * 0.80 / 1000) * 1000 : Math.round(priceInAdCurr * 0.80 * 100) / 100
    const budgetCap = isLak ? Math.round(priceInAdCurr * 3 / 1000) * 1000 : Math.round(priceInAdCurr * 3 * 100) / 100

    const calculated: ProductAdStrategy = {
      productId: product.id,
      enabled: true,
      targetCost: targetCpa,
      grossMargin: margin,
      scaleMinPurchases: 3,
      scaleMaxCPA: scaleCpa,
      scaleMaxDailyBudgetCap: budgetCap,
      stopMinSpend: stopSpend,
      updatedAt: new Date().toISOString(),
    }
    productStrategies.value[product.id] = calculated
    saveStrategies()
    return calculated
  }

  return {
    productStrategies,
    loadStrategies,
    saveStrategies,
    getStrategy,
    setStrategy,
    autoCalculate,
  }
}
