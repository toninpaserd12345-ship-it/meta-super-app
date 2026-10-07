import re
content = open("frontend/pages/ads.vue").read()

content = content.replace("const accRes = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts').catch(() => ({ items: [] }))", "const accRes = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts').catch(err => { noticeType.value = 'error'; notice.value = err.data?.message || err.message; return { items: [] } })")

open("frontend/pages/ads.vue", "w").write(content)
