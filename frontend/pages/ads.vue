<script setup lang="ts">
import type { MetaAdAccount, MetaCampaign, MetaAd, AutomationFlow, ItemsResponse, Product, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { can } = useAuth()
const { l } = useLocale()
const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const campaignAds = ref<Record<string, MetaAd[]>>({})
const flows = ref<AutomationFlow[]>([])
const products = ref<Product[]>([])
const replySets = ref<ReplySet[]>([])

const adAccountId = ref('')
const loading = ref(false)
const expandedCampaigns = ref<string[]>([])

const notice = ref('')

async function loadData() {
  loading.value = true
  try {
    const accRes = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts').catch(err => { notice.value = err.data?.message || err.message; return { items: [] } })
    const [flowRes, productRes, replyRes] = await Promise.all([
      $fetch<ItemsResponse<AutomationFlow>>('/api/proxy/api/v1/automations').catch(() => ({ items: [] })),
      $fetch<ItemsResponse<Product>>('/api/proxy/api/v1/products').catch(() => ({ items: [] })),
      $fetch<ReplySet[]>('/api/proxy/api/v1/replies').catch(() => []),
    ])

    adAccounts.value = accRes.items || []
    flows.value = flowRes.items || []
    products.value = productRes.items || []
    replySets.value = replyRes || []

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

const campaignFlows=(campaignId:string)=>flows.value.filter(flow=>flow.targets.some(target=>target.type==='campaign'&&target.value===campaignId))
const adFlows=(adId:string)=>flows.value.filter(flow=>flow.targets.some(target=>target.type==='ad'&&target.value===adId))
const productName=(id:string)=>products.value.find(item=>item.id===id)?.name||l({lo:'ບໍ່ພົບສິນຄ້າ',th:'ไม่พบสินค้า',en:'Product unavailable'})
const replySetName=(id:string)=>replySets.value.find(item=>item.id===id)?.name||l({lo:'ບໍ່ພົບຊຸດຂໍ້ຄວາມ',th:'ไม่พบชุดข้อความ',en:'Reply Set unavailable'})

watch(adAccountId, loadCampaigns)
onMounted(loadData)
</script>

<template>
  <section class="page-head">
    <div>
      <p>MARKETING</p>
      <h2>{{ l({lo:'Campaign ແລະ Ads',th:'แคมเปญและโฆษณา',en:'Campaigns and Ads'}) }}</h2>
      <span>{{ l({lo:'ກວດສອບ Campaign, Ads ແລະ Automation ທີ່ເຊື່ອມກັນດ້ວຍຂໍ້ມູນຈາກ Meta',th:'ตรวจสอบแคมเปญ โฆษณา และ Automation ที่เชื่อมต่อด้วยข้อมูลจาก Meta',en:'Review campaigns, ads, and connected Automations using Meta data.'}) }}</span>
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
        :label="l({lo:'ເລືອກບັນຊີໂຄສະນາ',th:'เลือกบัญชีโฆษณา',en:'Select Ad Account'})"
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
          <th>{{ l({lo:'ຊື່ Campaign',th:'ชื่อแคมเปญ',en:'Campaign Name'}) }}</th>
          <th>{{ l({lo:'ສະຖານະ',th:'สถานะ',en:'Status'}) }}</th>
          <th>{{ l({lo:'ເປົ້າໝາຍ',th:'วัตถุประสงค์',en:'Objective'}) }}</th>
          <th>{{ l({lo:'Ad Sets / Ads',th:'ชุดโฆษณา / โฆษณา',en:'Ad Sets / Ads'}) }}</th>
          <th>{{ l({lo:'ຕອບກັບອັດຕະໂນມັດ',th:'ตอบกลับอัตโนมัติ',en:'Auto Reply'}) }}</th>
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
            <div v-if="campaignFlows(campaign.id).length" class="automation-links"><NuxtLink v-for="flow in campaignFlows(campaign.id)" :key="flow.id" to="/auto-replies"><span :class="{active:flow.isActive}"/><div><strong>{{flow.name}}</strong><small>{{productName(flow.productId)}} → {{replySetName(flow.replySetId)}}</small></div></NuxtLink></div>
            <v-btn v-else size="small" color="primary" variant="tonal" prepend-icon="mdi-robot-outline" to="/auto-replies">{{ l({lo:'ສ້າງ Automation',th:'สร้าง Automation',en:'Create Automation'}) }}</v-btn>
          </td>
        </tr>
        <template v-if="expandedCampaigns.includes(campaign.id)">
          <tr v-if="!campaignAds[campaign.id]" class="ad-row">
            <td colspan="5" class="text-center text-caption py-3"><v-progress-circular indeterminate size="20" /> {{ l({lo:'ກຳລັງໂຫຼດ Ads...',th:'กำลังโหลดโฆษณา...',en:'Loading ads...'}) }}</td>
          </tr>
          <tr v-else-if="(campaignAds[campaign.id]?.length || 0) === 0" class="ad-row">
            <td colspan="5" class="text-center text-caption py-3">{{ l({lo:'ບໍ່ພົບ Ads ໃນ Campaign ນີ້',th:'ไม่พบโฆษณาในแคมเปญนี้',en:'No ads found in this campaign.'}) }}</td>
          </tr>
          <tr v-else v-for="ad in campaignAds[campaign.id]" :key="ad.id" class="ad-row">
            <td class="pl-8">
              <div class="ad-identity">
                <img v-if="ad.thumbnailUrl || ad.imageUrl" :src="ad.thumbnailUrl || ad.imageUrl" :alt="ad.name" loading="lazy">
                <span v-else class="ad-placeholder"><v-icon icon="mdi-bullhorn-outline" size="small" /></span>
                <span>{{ ad.name }}</span>
              </div>
            </td>
            <td><v-chip size="small" variant="text" :color="ad.effectiveStatus === 'ACTIVE' ? 'success' : 'default'">{{ ad.effectiveStatus }}</v-chip></td>
            <td><small class="text-disabled">{{ ad.id }}</small></td>
            <td></td>
            <td>
              <div v-if="adFlows(ad.id).length" class="automation-links"><NuxtLink v-for="flow in adFlows(ad.id)" :key="flow.id" to="/auto-replies"><span :class="{active:flow.isActive}"/><div><strong>{{flow.name}}</strong><small>{{productName(flow.productId)}} → {{replySetName(flow.replySetId)}}</small></div></NuxtLink></div>
              <span v-else-if="campaignFlows(campaign.id).length" class="text-caption text-success">{{ l({lo:'ຄວບຄຸມໂດຍ Campaign Automation',th:'ควบคุมโดย Campaign Automation',en:'Controlled by Campaign Automation'}) }}</span>
              <span v-else class="text-caption text-disabled">{{ l({lo:'ຍັງບໍ່ມີ Automation',th:'ยังไม่มี Automation',en:'No Automation yet'}) }}</span>
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
.ad-identity{display:flex;align-items:center;gap:10px;min-width:220px}.ad-identity img,.ad-placeholder{width:48px;height:48px;flex:0 0 48px;border-radius:9px;background:var(--color-surface);object-fit:cover}.ad-placeholder{display:grid;place-items:center;color:var(--color-text-muted)}
.automation-links{display:grid;gap:5px;min-width:190px}.automation-links a{display:flex;align-items:center;gap:7px;color:var(--color-text);text-decoration:none}.automation-links a>span{width:7px;height:7px;background:var(--color-text-muted);border-radius:50%}.automation-links a>span.active{background:var(--color-success)}.automation-links a>div{display:grid;min-width:0}.automation-links strong,.automation-links small{max-width:230px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.automation-links strong{font-size:11px}.automation-links small{color:var(--color-text-secondary);font-size:9px}
</style>
