<script setup lang="ts">
import type { MetaAdAccount, MetaCampaign, MetaAd, AutomationRule, Product, ReplySet, MetaPage } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { can } = useAuth()

const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const campaignAds = ref<Record<string, MetaAd[]>>({})
const rules = ref<AutomationRule[]>([])

const products = ref<Product[]>([])
const replySets = ref<ReplySet[]>([])
const pages = ref<MetaPage[]>([])

const adAccountId = ref('')
const loading = ref(false)
const expandedCampaigns = ref<string[]>([])

// Dialog state
const bindDialog = ref(false)
const bindTarget = ref<{ type: 'campaign' | 'ad', id: string, name: string, campaignId?: string } | null>(null)
const bindProductId = ref('')
const bindReplySetId = ref('')
const bindPageId = ref('')
const binding = ref(false)
const notice = ref('')

async function loadData() {
  loading.value = true
  try {
    const accRes = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts').catch(err => { notice.value = err.data?.message || err.message; return { items: [] } })
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
}

async function loadCampaigns() {
  if (!adAccountId.value) return
  loading.value = true
  try {
    const res = await $fetch<{ items: MetaCampaign[] }>(`/api/proxy/api/v1/meta/ad-accounts/${adAccountId.value}/campaigns`)
    campaigns.value = res.items || []
  } finally {
    loading.value = false
  }
}

async function toggleCampaign(campaignId: string) {
  if (expandedCampaigns.value.includes(campaignId)) {
    expandedCampaigns.value = expandedCampaigns.value.filter(id => id !== campaignId)
    return
  }
  expandedCampaigns.value.push(campaignId)
  if (!campaignAds.value[campaignId]) {
    try {
      const res = await $fetch<{ items: MetaAd[] }>(`/api/proxy/api/v1/meta/campaigns/${campaignId}/ads`)
      campaignAds.value[campaignId] = res.items || []
    } catch (e) {
      // ignore
    }
  }
}

function openBindDialog(type: 'campaign' | 'ad', id: string, name: string, campaignId?: string) {
  bindTarget.value = { type, id, name, campaignId }
  bindProductId.value = ''
  bindReplySetId.value = ''
  if (pages.value.length === 1) bindPageId.value = (pages.value[0]?.id || '')
  bindDialog.value = true
  notice.value = ''
}

async function confirmBind() {
  if (!bindTarget.value || !bindPageId.value || !bindReplySetId.value) return
  binding.value = true
  notice.value = ''
  
  let targetAds: MetaAd[] = []
  if (bindTarget.value.type === 'campaign') {
    targetAds = campaignAds.value[bindTarget.value.id] || []
  } else {
    targetAds = [{ id: bindTarget.value.id, name: bindTarget.value.name } as MetaAd]
  }

  if (targetAds.length === 0) {
    notice.value = "No ads found to bind."
    binding.value = false
    return
  }

  let created = 0
  for (const ad of targetAds) {
    if (isBound(ad.id)) continue
    try {
      await $fetch('/api/proxy/api/v1/automation/rules', {
        method: 'POST',
        body: {
          pageId: bindPageId.value,
          triggerType: 'ad',
          triggerValue: ad.id,
          triggerName: `${bindTarget.value.type === 'campaign' ? `[Campaign: ${bindTarget.value.name}] ` : ''}Ad: ${ad.name}`,
          productId: bindProductId.value, // Optional
          replySetId: bindReplySetId.value
        }
      })
      created++
    } catch (e) {
      console.error(e)
    }
  }

  if (created > 0) {
    const res = await $fetch<{ items: AutomationRule[] }>('/api/proxy/api/v1/automation/rules')
    rules.value = res.items || []
    bindDialog.value = false
  } else {
    notice.value = "No new rules created (maybe they were already bound)."
  }
  binding.value = false
}

function isBound(adId: string) {
  return rules.value.some((r: any) => r.triggerType === 'ad' && r.triggerValue === adId)
}

function campaignBoundCount(campaignId: string) {
  const ads = campaignAds.value[campaignId] || []
  if (ads.length === 0) return 0
  return ads.filter(ad => isBound(ad.id)).length
}

watch(adAccountId, loadCampaigns)
onMounted(loadData)
</script>

<template>
  <section class="page-head">
    <div>
      <p>MARKETING</p>
      <h2>Campaign & Ads Manager</h2>
      <span>View your Facebook Campaigns and quickly bind Auto Replies to them.</span>
    </div>
  </section>

  <v-alert v-if="notice && !bindDialog" type="error" variant="tonal" class="mb-4">{{ notice }}</v-alert>
  <v-card class="mb-4 pa-4" variant="outlined">
    <div style="max-width: 400px">
      <v-select
        v-model="adAccountId"
        :items="adAccounts.map((a: any) => ({ ...a, label: `${a.accountType === 'business' ? 'Business' : 'Personal'} · ${a.name}` }))"
        item-title="label"
        item-value="id"
        label="Select Ad Account"
        variant="outlined"
        hide-details
      />
    </div>
  </v-card>

  <v-progress-linear v-if="loading" indeterminate color="primary" class="mb-4" />

  <v-card variant="outlined" class="campaign-table-card">
    <table class="campaign-table">
      <thead>
        <tr>
          <th>Campaign Name</th>
          <th>Status</th>
          <th>Objective</th>
          <th>Ad Sets / Ads</th>
          <th>Auto Reply</th>
        </tr>
      </thead>
      <tbody v-for="campaign in campaigns" :key="campaign.id">
        <tr class="campaign-row" :class="{ expanded: expandedCampaigns.includes(campaign.id) }" @click="toggleCampaign(campaign.id)">
          <td>
            <v-icon :icon="expandedCampaigns.includes(campaign.id) ? 'mdi-chevron-down' : 'mdi-chevron-right'" class="mr-2" />
            <strong>{{ campaign.name }}</strong>
          </td>
          <td><v-chip size="small" :color="campaign.effectiveStatus === 'ACTIVE' ? 'success' : 'default'">{{ campaign.effectiveStatus }}</v-chip></td>
          <td><small>{{ campaign.objective }}</small></td>
          <td>{{ campaign.adSetCount }} Sets · {{ campaign.adCount }} Ads</td>
          <td @click.stop>
            <v-btn
              v-if="expandedCampaigns.includes(campaign.id) && (campaignAds[campaign.id]?.length || 0) > 0"
              size="small"
              color="primary"
              variant="tonal"
              prepend-icon="mdi-robot-outline"
              @click="openBindDialog('campaign', campaign.id, campaign.name)"
            >
              Bind Campaign
            </v-btn>
            <span v-else-if="campaignBoundCount(campaign.id) > 0" class="text-success text-caption ml-2">
              {{ campaignBoundCount(campaign.id) }} bound
            </span>
          </td>
        </tr>
        <template v-if="expandedCampaigns.includes(campaign.id)">
          <tr v-if="!campaignAds[campaign.id]" class="ad-row">
            <td colspan="5" class="text-center text-caption py-3"><v-progress-circular indeterminate size="20" /> Loading ads...</td>
          </tr>
          <tr v-else-if="(campaignAds[campaign.id]?.length || 0) === 0" class="ad-row">
            <td colspan="5" class="text-center text-caption py-3">No ads found in this campaign.</td>
          </tr>
          <tr v-else v-for="ad in campaignAds[campaign.id]" :key="ad.id" class="ad-row">
            <td class="pl-8">
              <v-icon icon="mdi-bullhorn-outline" size="small" class="mr-2 text-disabled" />
              {{ ad.name }}
            </td>
            <td><v-chip size="small" variant="text" :color="ad.effectiveStatus === 'ACTIVE' ? 'success' : 'default'">{{ ad.effectiveStatus }}</v-chip></td>
            <td><small class="text-disabled">{{ ad.id }}</small></td>
            <td></td>
            <td>
              <v-btn
                v-if="!isBound(ad.id)"
                size="small"
                variant="outlined"
                @click.stop="openBindDialog('ad', ad.id, ad.name, campaign.id)"
              >
                Bind Ad
              </v-btn>
              <v-chip v-else color="success" size="small" prepend-icon="mdi-check">Bound</v-chip>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </v-card>

  <!-- Bind Dialog -->
  <v-dialog v-model="bindDialog" max-width="500">
    <v-card class="pa-4 rounded-lg">
      <h3 class="mb-2">Bind Auto Reply</h3>
      <p class="text-caption mb-4 text-disabled">
        Binding to {{ bindTarget?.type === 'campaign' ? 'all ads in campaign' : 'ad' }}: <strong>{{ bindTarget?.name }}</strong>
      </p>

      <v-alert v-if="notice" type="error" variant="tonal" class="mb-4" density="compact">{{ notice }}</v-alert>

      <v-select
        v-model="bindPageId"
        :items="pages"
        item-title="name"
        item-value="id"
        label="Facebook Page (Required)"
        variant="outlined"
        class="mb-3"
      />
      <v-select
        v-model="bindReplySetId"
        :items="replySets.filter((s: any) => s.items.some((i: any) => i.isEnabled))"
        item-title="name"
        item-value="id"
        label="Reply Set (Required)"
        variant="outlined"
        class="mb-3"
      />
      <v-select
        v-model="bindProductId"
        :items="products"
        item-title="name"
        item-value="id"
        label="Product (Optional)"
        variant="outlined"
        class="mb-3"
        clearable
      />

      <div class="d-flex justify-end mt-2">
        <v-btn variant="text" @click="bindDialog = false">Cancel</v-btn>
        <v-btn color="primary" class="ml-2" :loading="binding" :disabled="!bindPageId || !bindReplySetId" @click="confirmBind">
          Create {{ bindTarget?.type === 'campaign' ? (campaignAds[bindTarget.id]?.length || 0) : 1 }} Auto Replies
        </v-btn>
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.page-head {
  padding: 24px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  margin-bottom: 16px;
}
.page-head p { margin: 0 0 6px; color: var(--color-primary); font-size: 10px; font-weight: 800; letter-spacing: 0.13em; }
.page-head h2 { margin: 0 0 6px; font-size: 24px; }
.page-head span { color: var(--color-text-secondary); font-size: 13px; }

.campaign-table-card {
  overflow-x: auto;
  border-radius: var(--radius-lg);
}
.campaign-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}
.campaign-table th {
  padding: 14px 16px;
  background: var(--color-surface-soft);
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  border-bottom: 1px solid var(--color-border);
}
.campaign-row td {
  padding: 16px;
  border-bottom: 1px solid var(--color-border-subtle);
  cursor: pointer;
}
.campaign-row:hover {
  background: var(--color-background);
}
.campaign-row.expanded {
  background: var(--color-primary-soft);
}
.ad-row td {
  padding: 12px 16px;
  background: var(--color-surface-soft);
  border-bottom: 1px solid var(--color-border-subtle);
  font-size: 13px;
}
.ad-row:last-child td {
  border-bottom: 1px solid var(--color-border);
}
</style>
