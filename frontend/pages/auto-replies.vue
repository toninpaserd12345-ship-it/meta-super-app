<script setup lang="ts">
import type { AutomationRule, ItemsResponse, MetaAd, MetaAdAccount, MetaCampaign, MetaPage, MetaPost, Product, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { can } = useAuth()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Auto Replies.' })

const { data: pageData } = await useApi<ItemsResponse<MetaPage>>('/proxy/api/v1/meta/pages')
const { data: productData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: setData } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const { data: ruleData, refresh: refreshRules } = await useApi<ItemsResponse<AutomationRule>>('/proxy/api/v1/automation/rules')

const pages = computed(() => pageData.value?.items.filter(item => item.connected) || [])
const products = computed(() => productData.value?.items || [])
const replySets = computed(() => setData.value || [])
const rules = computed(() => ruleData.value?.items || [])
const usableSets = computed(() => replySets.value.filter(set => set.items?.some(item => item.isEnabled && item.content.trim())))

const view = ref<'dashboard' | 'wizard'>('dashboard')
const step = ref(1)
const productId = ref('')
const replySetId = ref('')
const pageId = ref('')
const targetType = ref<'post' | 'ad'>('post')
const selectedPosts = ref<string[]>([])
const selectedAds = ref<string[]>([])
const posts = ref<MetaPost[]>([])
const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const ads = ref<MetaAd[]>([])
const adAccountId = ref('')
const selectedCampaignIds = ref<string[]>([])
const loadingTargets = ref(false)
const saving = ref(false)
const deletingId = ref('')
const notice = ref('')
const noticeType = ref<'success' | 'error' | 'warning'>('success')
const search = ref('')

const selectedProduct = computed(() => products.value.find(item => item.id === productId.value))
const selectedSet = computed(() => replySets.value.find(item => item.id === replySetId.value))
const selectedPage = computed(() => pages.value.find(item => item.id === pageId.value))
const targetIds = computed(() => targetType.value === 'post' ? selectedPosts.value : selectedAds.value)
const stepReady = computed(() => step.value === 1 ? !!productId.value : step.value === 2 ? !!replySetId.value : step.value === 3 ? !!pageId.value && targetIds.value.length > 0 : true)
const filteredRules = computed(() => {
  const value = search.value.trim().toLowerCase()
  if (!value) return rules.value
  return rules.value.filter(rule => [rule.triggerType, rule.triggerValue, productName(rule.productId), setName(rule.replySetId), pageName(rule.pageId)].join(' ').toLowerCase().includes(value))
})

function productName(id: string) { return products.value.find(item => item.id === id)?.name || 'Missing product' }
function setName(id: string) { return replySets.value.find(item => item.id === id)?.name || 'Missing Reply Set' }
function pageName(id: string) { return pages.value.find(item => item.id === id)?.name || id }
function isBound(id: string) { return rules.value.some(rule => rule.pageId === pageId.value && rule.triggerType === targetType.value && rule.triggerValue === id) }
function apiError(error: unknown, fallback: string) {
  const value = error as { data?: { error?: string | { message?: string }; message?: string }; statusMessage?: string }
  const dataError = value.data?.error
  return (typeof dataError === 'string' ? dataError : dataError?.message) || value.data?.message || value.statusMessage || fallback
}
function startWizard() {
  step.value = 1; productId.value = products.value[0]?.id || ''; replySetId.value = usableSets.value[0]?.id || ''; pageId.value = pages.value[0]?.id || ''
  targetType.value = 'post'; selectedPosts.value = []; selectedAds.value = []; notice.value = ''; view.value = 'wizard'
}
function next() { if (stepReady.value && step.value < 4) step.value++ }
function back() { if (step.value > 1) step.value--; else view.value = 'dashboard' }

async function loadPosts() {
  if (!pageId.value) return
  loadingTargets.value = true; selectedPosts.value = []
  try { posts.value = (await $fetch<ItemsResponse<MetaPost>>(`/api/proxy/api/v1/meta/pages/${pageId.value}/posts`)).items }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load Facebook posts.') }
  finally { loadingTargets.value = false }
}
async function loadAdAccounts() {
  if (adAccounts.value.length) return
  loadingTargets.value = true
  try { adAccounts.value = (await $fetch<ItemsResponse<MetaAdAccount>>('/api/proxy/api/v1/meta/ad-accounts')).items; adAccountId.value = adAccounts.value[0]?.id || '' }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load Ad Accounts. Reconnect Facebook with ads_read permission.') }
  finally { loadingTargets.value = false }
}
async function loadCampaigns() {
  campaigns.value = []; selectedCampaignIds.value = []; ads.value = []; selectedAds.value = []
  if (!adAccountId.value) return
  loadingTargets.value = true
  try { campaigns.value = (await $fetch<ItemsResponse<MetaCampaign>>(`/api/proxy/api/v1/meta/ad-accounts/${adAccountId.value}/campaigns`)).items }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load campaigns.') }
  finally { loadingTargets.value = false }
}
async function toggleCampaign(id: string) {
  selectedCampaignIds.value = selectedCampaignIds.value.includes(id) ? selectedCampaignIds.value.filter(item => item !== id) : [...selectedCampaignIds.value, id]
  loadingTargets.value = true
  try {
    const results = await Promise.all(selectedCampaignIds.value.map(campaignId => $fetch<ItemsResponse<MetaAd>>(`/api/proxy/api/v1/meta/campaigns/${campaignId}/ads`)))
    const unique = new Map<string, MetaAd>(); results.flatMap(result => result.items).forEach(ad => unique.set(ad.id, ad)); ads.value = [...unique.values()]
    selectedAds.value = ads.value.filter(ad => !isBound(ad.id)).map(ad => ad.id)
  } catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load Ads in the selected campaigns.') }
  finally { loadingTargets.value = false }
}

async function createRules() {
  if (!selectedProduct.value || !selectedSet.value || !selectedPage.value || !targetIds.value.length) return
  saving.value = true; notice.value = ''
  const failures: string[] = []; let created = 0
  for (const id of targetIds.value) {
    try {
      await $fetch('/api/proxy/api/v1/automation/rules', { method: 'POST', body: { pageId: pageId.value, triggerType: targetType.value, triggerValue: id, productId: productId.value, replySetId: replySetId.value } })
      created++
    } catch (error) { failures.push(`${id}: ${apiError(error, 'failed')}`) }
  }
  await refreshRules(); saving.value = false
  if (failures.length) { noticeType.value = created ? 'warning' : 'error'; notice.value = `${created} created; ${failures.length} failed. ${failures[0]}`; return }
  noticeType.value = 'success'; notice.value = `Created ${created} Auto ${created === 1 ? 'Reply' : 'Replies'}.`; view.value = 'dashboard'
}
async function toggleRule(rule: AutomationRule, active: boolean) {
  try { await $fetch(`/api/proxy/api/v1/automation/rules/${rule.id}`, { method: 'PUT', body: { triggerType: rule.triggerType, triggerValue: rule.triggerValue, productId: rule.productId, replySetId: rule.replySetId, isActive: active } }); await refreshRules() }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not update this Auto Reply.') }
}
async function deleteRule(rule: AutomationRule) {
  if (!confirm(`Delete the Auto Reply for ${rule.triggerValue}?`)) return
  deletingId.value = rule.id
  try { await $fetch(`/api/proxy/api/v1/automation/rules/${rule.id}`, { method: 'DELETE' }); await refreshRules(); noticeType.value = 'success'; notice.value = 'Auto Reply deleted.' }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not delete this Auto Reply.') }
  finally { deletingId.value = '' }
}

watch(pageId, () => { if (targetType.value === 'post') loadPosts() })
watch(targetType, value => { selectedPosts.value = []; selectedAds.value = []; if (value === 'post') loadPosts(); else loadAdAccounts() })
watch(adAccountId, loadCampaigns)
</script>

<template>
  <section class="page-head">
    <div><p>AUTO REPLY</p><h2>{{ view === 'dashboard' ? 'Auto Reply Manager' : 'Create Auto Reply' }}</h2><span>{{ view === 'dashboard' ? 'Connect a Product and Reply Set to Facebook Posts or Ads.' : 'Follow four clear steps. Nothing is published until Review.' }}</span></div>
    <div class="head-actions"><v-btn variant="outlined" to="/products" prepend-icon="mdi-package-variant-closed">Products</v-btn><v-btn variant="outlined" to="/replies" prepend-icon="mdi-message-text-fast-outline">Reply Sets</v-btn><v-btn v-if="view==='dashboard'" color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="startWizard">Create Auto Reply</v-btn></div>
  </section>
  <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>

  <template v-if="view==='dashboard'">
    <section class="summary-grid"><article><v-icon icon="mdi-package-variant-closed"/><div><strong>{{ products.length }}</strong><span>Products</span></div></article><article><v-icon icon="mdi-message-text-fast-outline"/><div><strong>{{ usableSets.length }}</strong><span>Ready Reply Sets</span></div></article><article><v-icon icon="mdi-robot-happy-outline"/><div><strong>{{ rules.filter(item=>item.isActive).length }}</strong><span>Active Auto Replies</span></div></article></section>
    <v-alert v-if="!products.length || !usableSets.length || !pages.length" type="info" variant="tonal" class="mb-4"><strong>Setup needed:</strong> <NuxtLink v-if="!products.length" to="/products">add a Product</NuxtLink><span v-if="!products.length && (!usableSets.length || !pages.length)"> · </span><NuxtLink v-if="!usableSets.length" to="/replies">build a Reply Set</NuxtLink><span v-if="!usableSets.length && !pages.length"> · </span><NuxtLink v-if="!pages.length" to="/meta-pages">connect a Facebook Page</NuxtLink>.</v-alert>
    <section class="rules-panel"><div class="panel-head"><div><small>LIVE CONFIGURATION</small><h3>Auto Replies</h3></div><v-text-field v-model="search" density="compact" variant="outlined" prepend-inner-icon="mdi-magnify" placeholder="Search" hide-details/></div>
      <div v-if="filteredRules.length" class="rule-list"><article v-for="rule in filteredRules" :key="rule.id" class="rule-row"><div class="rule-type"><v-icon :icon="rule.triggerType==='post'?'mdi-post-outline':'mdi-bullhorn-outline'"/><span>{{ rule.triggerType }}</span></div><div class="rule-main"><strong>{{ productName(rule.productId) }}</strong><span>{{ setName(rule.replySetId) }} · {{ pageName(rule.pageId) }}</span><small>{{ rule.triggerValue }}</small></div><v-switch :model-value="rule.isActive" color="success" hide-details density="compact" :disabled="!can('pages:connect')" @update:model-value="toggleRule(rule, Boolean($event))"/><v-btn icon="mdi-delete-outline" variant="text" color="error" :loading="deletingId===rule.id" :disabled="!can('pages:connect')" @click="deleteRule(rule)"/></article></div>
      <div v-else class="empty"><v-icon icon="mdi-robot-happy-outline" size="34"/><h3>No Auto Replies yet</h3><p>Create one to connect product answers to a Post or Ad.</p><v-btn color="primary" :disabled="!products.length || !usableSets.length || !pages.length" @click="startWizard">Create Auto Reply</v-btn></div>
    </section>
  </template>

  <template v-else>
    <nav class="steps"><button v-for="item in [{n:1,label:'Product'},{n:2,label:'Reply Set'},{n:3,label:'Post or Ad'},{n:4,label:'Review'}]" :key="item.n" :class="{active:step===item.n,done:step>item.n}" :disabled="item.n>step" @click="step=item.n"><span>{{ step>item.n?'✓':item.n }}</span><b>{{ item.label }}</b></button></nav>
    <section class="wizard-card">
      <template v-if="step===1"><div class="section-title"><small>STEP 1</small><h3>Which product is this reply about?</h3><p>The chosen product supplies name, price and description variables.</p></div><div class="choice-grid"><button v-for="product in products" :key="product.id" :class="{selected:productId===product.id}" @click="productId=product.id"><v-icon icon="mdi-package-variant-closed"/><div><strong>{{ product.name }}</strong><span>{{ product.price }}</span><small>{{ product.description || 'No description' }}</small></div><v-icon :icon="productId===product.id?'mdi-check-circle':'mdi-circle-outline'"/></button></div><div v-if="!products.length" class="empty compact"><p>Create a product before continuing.</p><v-btn to="/products" color="primary">Manage Products</v-btn></div></template>
      <template v-else-if="step===2"><div class="section-title"><small>STEP 2</small><h3>Choose the message sequence</h3><p>Only enabled messages will be sent, from top to bottom.</p></div><div class="choice-grid"><button v-for="set in usableSets" :key="set.id" :class="{selected:replySetId===set.id}" @click="replySetId=set.id"><v-icon icon="mdi-message-text-fast-outline"/><div><strong>{{ set.name }}</strong><span>{{ set.items.filter(item=>item.isEnabled).length }} enabled messages</span><small>Text, image, video and audio supported</small></div><v-icon :icon="replySetId===set.id?'mdi-check-circle':'mdi-circle-outline'"/></button></div><div v-if="!usableSets.length" class="empty compact"><p>Create a Reply Set with at least one enabled message.</p><v-btn to="/replies" color="primary">Manage Reply Sets</v-btn></div></template>
      <template v-else-if="step===3"><div class="section-title"><small>STEP 3</small><h3>Choose the Facebook source</h3><p>Select one Page, then one or more Posts or Ads.</p></div><div class="target-toolbar"><v-select v-model="pageId" :items="pages" item-title="name" item-value="id" label="Facebook Page" variant="outlined" hide-details/><v-btn-toggle v-model="targetType" mandatory color="primary"><v-btn value="post" prepend-icon="mdi-post-outline">Posts</v-btn><v-btn value="ad" prepend-icon="mdi-bullhorn-outline">Ads</v-btn></v-btn-toggle></div><div v-if="loadingTargets" class="loading"><v-progress-circular indeterminate color="primary"/></div>
        <div v-else-if="targetType==='post'" class="target-list"><label v-for="post in posts" :key="post.id" :class="{disabled:isBound(post.id)}"><v-checkbox-btn v-model="selectedPosts" :value="post.id" :disabled="isBound(post.id)"/><div><strong>{{ post.message || 'Facebook post' }}</strong><span>{{ post.id }}</span><small v-if="isBound(post.id)">Already has an Auto Reply</small></div></label><div v-if="!posts.length" class="empty compact"><p>No posts found for this Page.</p></div></div>
        <template v-else><v-select v-model="adAccountId" :items="adAccounts.map(item=>({ ...item, label:`${item.accountType==='business'?'Business':'Personal'} · ${item.name}` }))" item-title="label" item-value="id" label="Ad Account" variant="outlined" class="mb-4"/><div class="campaign-grid"><button v-for="campaign in campaigns" :key="campaign.id" :class="{selected:selectedCampaignIds.includes(campaign.id)}" @click="toggleCampaign(campaign.id)"><v-checkbox-btn :model-value="selectedCampaignIds.includes(campaign.id)"/><div><strong>{{ campaign.name }}</strong><span>{{ campaign.adSetCount }} ad sets · {{ campaign.adCount }} ads</span><small>{{ campaign.effectiveStatus }}</small></div></button></div><div v-if="ads.length" class="target-list mt-4"><label v-for="ad in ads" :key="ad.id" :class="{disabled:isBound(ad.id)}"><v-checkbox-btn v-model="selectedAds" :value="ad.id" :disabled="isBound(ad.id)"/><div><strong>{{ ad.name }}</strong><span>{{ ad.id }}</span><small>{{ isBound(ad.id) ? 'Already has an Auto Reply' : ad.effectiveStatus }}</small></div></label></div></template>
      </template>
      <template v-else><div class="section-title"><small>STEP 4</small><h3>Review before creating</h3><p>One Auto Reply will be created for each selected target.</p></div><div class="review"><article><span>Product</span><strong>{{ selectedProduct?.name }}</strong><small>{{ selectedProduct?.price }}</small></article><article><span>Reply Set</span><strong>{{ selectedSet?.name }}</strong><small>{{ selectedSet?.items.filter(item=>item.isEnabled).length }} enabled messages</small></article><article><span>Facebook Page</span><strong>{{ selectedPage?.name }}</strong><small>{{ targetType.toUpperCase() }} · {{ targetIds.length }} selected</small></article></div><v-alert type="info" variant="tonal">Product variables in the Reply Set will be filled from <strong>{{ selectedProduct?.name }}</strong> when the webhook receives the first matching event.</v-alert></template>
    </section>
    <footer class="wizard-actions"><v-btn variant="text" @click="back">{{ step===1?'Cancel':'Back' }}</v-btn><v-spacer/><v-btn v-if="step<4" color="primary" append-icon="mdi-arrow-right" :disabled="!stepReady" @click="next">Continue</v-btn><v-btn v-else color="primary" prepend-icon="mdi-robot-happy-outline" :loading="saving" @click="createRules">Create {{ targetIds.length }} Auto {{ targetIds.length===1?'Reply':'Replies' }}</v-btn></footer>
  </template>
</template>

<style scoped>
.page-head{display:flex;justify-content:space-between;gap:22px;margin-bottom:18px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head p,.section-title small,.panel-head small{margin:0 0 6px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 6px;font-size:24px}.page-head span,.section-title p{color:var(--color-text-secondary);font-size:13px}.head-actions{display:flex;gap:8px;flex-wrap:wrap;align-items:flex-start}.summary-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin-bottom:16px}.summary-grid article{display:flex;align-items:center;gap:14px;padding:18px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);color:var(--color-primary)}.summary-grid article>div{display:grid}.summary-grid strong{color:var(--color-text);font-size:22px}.summary-grid span{color:var(--color-text-secondary);font-size:11px}.rules-panel,.wizard-card{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.panel-head{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:18px 20px;border-bottom:1px solid var(--color-border)}.panel-head h3{margin:0}.panel-head :deep(.v-input){max-width:300px}.rule-row{display:grid;grid-template-columns:100px 1fr auto auto;align-items:center;gap:14px;padding:15px 20px;border-bottom:1px solid var(--color-border-subtle)}.rule-row:last-child{border:0}.rule-type{display:flex;align-items:center;gap:6px;color:var(--color-primary);text-transform:capitalize}.rule-main{display:grid;min-width:0}.rule-main span,.rule-main small{color:var(--color-text-secondary);font-size:11px}.rule-main small{overflow:hidden;text-overflow:ellipsis}.empty{min-height:300px;display:grid;place-items:center;align-content:center;text-align:center;padding:30px;color:var(--color-text-muted)}.empty h3{margin:12px 0 3px;color:var(--color-text)}.empty p{margin:0 0 15px}.empty.compact{min-height:170px}.steps{display:grid;grid-template-columns:repeat(4,1fr);margin-bottom:14px;padding:9px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.steps button{display:flex;align-items:center;justify-content:center;gap:8px;padding:10px;border:0;background:none;color:var(--color-text-muted)}.steps button span{width:27px;height:27px;display:grid;place-items:center;border-radius:50%;background:var(--color-surface-soft);font-size:11px}.steps button.active,.steps button.done{color:var(--color-primary)}.steps button.active span,.steps button.done span{color:white;background:var(--color-primary)}.wizard-card{min-height:430px;padding:24px}.section-title h3{margin:0 0 6px;font-size:21px}.section-title p{margin:0 0 20px}.choice-grid,.campaign-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:11px}.choice-grid>button,.campaign-grid>button{display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:12px;padding:16px;border:1px solid var(--color-border);border-radius:14px;background:white;text-align:left;color:var(--color-text);cursor:pointer}.choice-grid>button.selected,.campaign-grid>button.selected{border-color:var(--color-primary);background:var(--color-primary-soft)}.choice-grid>button>div,.campaign-grid>button>div{display:grid}.choice-grid span,.choice-grid small,.campaign-grid span,.campaign-grid small{color:var(--color-text-secondary);font-size:11px}.target-toolbar{display:grid;grid-template-columns:1fr auto;gap:12px;margin-bottom:16px}.target-list{display:grid;gap:8px}.target-list label{display:grid;grid-template-columns:auto 1fr;align-items:center;padding:10px 14px;border:1px solid var(--color-border);border-radius:12px}.target-list label.disabled{opacity:.55}.target-list label>div{display:grid}.target-list span,.target-list small{color:var(--color-text-secondary);font-size:10px}.campaign-grid>button{grid-template-columns:auto 1fr}.loading{min-height:220px;display:grid;place-items:center}.review{display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin-bottom:18px}.review article{display:grid;gap:5px;padding:18px;background:var(--color-surface-soft);border-radius:14px}.review span,.review small{color:var(--color-text-secondary);font-size:11px}.wizard-actions{display:flex;align-items:center;margin-top:14px;padding:14px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
@media(max-width:850px){.summary-grid,.review{grid-template-columns:1fr}.choice-grid,.campaign-grid{grid-template-columns:1fr}.rule-row{grid-template-columns:80px 1fr auto}.rule-row>:last-child{grid-column:3}.target-toolbar{grid-template-columns:1fr}.steps b{display:none}}
@media(max-width:640px){.page-head{display:grid;padding:20px}.head-actions{display:grid;grid-template-columns:1fr 1fr}.head-actions>:last-child{grid-column:1/-1}.wizard-card{padding:16px}.steps{grid-template-columns:repeat(4,1fr)}.rule-row{grid-template-columns:1fr auto}.rule-type{grid-column:1}.rule-main{grid-column:1}.rule-row>:last-child{grid-column:2;grid-row:2}.panel-head{display:grid}.panel-head :deep(.v-input){max-width:none}}
</style>
