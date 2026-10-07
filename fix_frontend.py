import sys
content = open("frontend/pages/auto-replies.vue").read()

# We need to map `id` (which is targetId) to its name.
# It's an ad or post.
# In `createRules()`, we have `targetType.value` and `targetIds.value`.
# For each `id`, we can find its name.
new_createRules = """async function createRules() {
  if (!selectedSet.value || !selectedPage.value || !targetIds.value.length) return
  saving.value = true; notice.value = ''
  const failures: string[] = []; let created = 0
  for (const id of targetIds.value) {
    let tName = ''
    if (targetType.value === 'post') {
      const p = posts.value.find(x => x.id === id)
      tName = p ? (p.message || 'Facebook Post') : ''
    } else {
      const a = ads.value.find(x => x.id === id)
      tName = a ? a.name : ''
    }
    
    try {
      await $fetch('/api/proxy/api/v1/automation/rules', { method: 'POST', body: { pageId: pageId.value, triggerType: targetType.value, triggerValue: id, triggerName: tName, productId: productId.value, replySetId: replySetId.value } })
      created++
    } catch (error) { failures.push(`${id}: ${apiError(error, 'failed')}`) }
  }"""

content = content.replace("""async function createRules() {
  if (!selectedSet.value || !selectedPage.value || !targetIds.value.length) return
  saving.value = true; notice.value = ''
  const failures: string[] = []; let created = 0
  for (const id of targetIds.value) {
    try {
      await $fetch('/api/proxy/api/v1/automation/rules', { method: 'POST', body: { pageId: pageId.value, triggerType: targetType.value, triggerValue: id, productId: productId.value, replySetId: replySetId.value } })
      created++
    } catch (error) { failures.push(`${id}: ${apiError(error, 'failed')}`) }
  }""", new_createRules)

# Also update toggleRule (PUT request)
new_toggleRule = """async function toggleRule(rule: AutomationRule, active: boolean) {
  try { await $fetch(`/api/proxy/api/v1/automation/rules/${rule.id}`, { method: 'PUT', body: { triggerType: rule.triggerType, triggerValue: rule.triggerValue, triggerName: rule.triggerName, productId: rule.productId, replySetId: rule.replySetId, isActive: active } }); await refreshRules() }"""

content = content.replace("""async function toggleRule(rule: AutomationRule, active: boolean) {
  try { await $fetch(`/api/proxy/api/v1/automation/rules/${rule.id}`, { method: 'PUT', body: { triggerType: rule.triggerType, triggerValue: rule.triggerValue, productId: rule.productId, replySetId: rule.replySetId, isActive: active } }); await refreshRules() }""", new_toggleRule)

# And in the dashboard, display the name!
# `<small>{{ rule.triggerValue }}</small>` -> `<small>{{ rule.triggerName || rule.triggerValue }}</small>`
content = content.replace("<small>{{ rule.triggerValue }}</small>", "<small>{{ rule.triggerName || rule.triggerValue }}</small>")

open("frontend/pages/auto-replies.vue", "w").write(content)
