content = open("frontend/pages/ads.vue").read()

new_load = """async function loadData() {
  loading.value = true
  try {
    const accRes = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts').catch(() => ({ items: [] }))
    const ruleRes = await $fetch<{ items: AutomationRule[] }>('/api/proxy/api/v1/automation/rules').catch(() => ({ items: [] }))
    const prodRes = await $fetch<{ items: Product[] }>('/api/proxy/api/v1/products').catch(() => ({ items: [] }))
    const repRes = await $fetch<{ items: ReplySet[] }>('/api/proxy/api/v1/replies').catch(() => ({ items: [] }))
    const pageRes = await $fetch<{ items: MetaPage[] }>('/api/proxy/api/v1/meta/pages').catch(() => ({ items: [] }))

    adAccounts.value = accRes.items || []
    rules.value = ruleRes.items || []
    products.value = prodRes.items || []
    replySets.value = repRes.items || []
    pages.value = pageRes.items || []

    if (adAccounts.value.length > 0) {
      adAccountId.value = (adAccounts.value[0]?.id || '')
    }
  } finally {
    loading.value = false
  }
}"""

import re
content = re.sub(r"async function loadData\(\) \{.*?(?=\nasync function loadCampaigns)", new_load + "\n", content, flags=re.DOTALL)
open("frontend/pages/ads.vue", "w").write(content)
