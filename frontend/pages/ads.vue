<script setup lang="ts">
import type { MetaAdAccount, MetaCampaign, MetaAd, AutomationRule } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { can } = useAuth()

const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const campaignAds = ref<Record<string, MetaAd[]>>({})
const rules = ref<AutomationRule[]>([])

const adAccountId = ref('')
const loading = ref(false)
const expandedCampaigns = ref<string[]>([])

const notice = ref('')

async function loadData() {
  loading.value = true
  try {
    const accRes = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts').catch(err => { notice.value = err.data?.message || err.message; return { items: [] } })
    const ruleRes = await $fetch<{ items: AutomationRule[] }>('/api/proxy/api/v1/automation/rules').catch(() => ({ items: [] }))

    adAccounts.value = accRes.items || []
    rules.value = ruleRes.items || []

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

function isBound(adId: string) {
  return rules.value.some((r: any) => r.triggerType === 'ad' && r.triggerValue === adId)
}

watch(adAccountId, loadCampaigns)
onMounted(loadData)
</script>

<template>
  <section class="page-head">
    <div>
      <p>MARKETING</p>
      <h2>Campaign & Ads Manager</h2>
      <span>Inspect Facebook delivery here. Create and manage all reply behavior in the Automation Center.</span>
    </div>
  </section>

  <v-alert v-if="notice" type="error" variant="tonal" class="mb-4">{{ notice }}</v-alert>
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
              size="small"
              color="primary"
              variant="tonal"
              prepend-icon="mdi-robot-outline"
              to="/auto-replies"
            >
              Automation Center
            </v-btn>
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
              <v-chip v-if="isBound(ad.id)" color="success" size="small" prepend-icon="mdi-check">Legacy binding</v-chip>
              <span v-else class="text-caption text-disabled">Managed by Campaign</span>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </v-card>

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
