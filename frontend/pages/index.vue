<script setup lang="ts">
import type { AutomationFlow, ItemsResponse, MetaPage, Product, ReplySet } from '~/types/automation'
definePageMeta({ middleware: 'auth' })
const route = useRoute()
const { locale, t } = useLocale()
const { data: pageData, pending: pagePending } = await useApi<ItemsResponse<MetaPage>>('/proxy/api/v1/meta/pages')
const { data: productData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: replyData } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const { data: automationData } = await useApi<ItemsResponse<AutomationFlow>>('/proxy/api/v1/automations')
const pages = computed(() => pageData.value?.items.filter(item => item.connected) || [])
const products = computed(() => productData.value?.items || [])
const replies = computed(() => replyData.value || [])
const automations = computed(() => automationData.value?.items || [])
const activeAutomations = computed(() => automations.value.filter(item => item.isActive))
const justConnected = computed(() => route.query.meta === 'connected' && pages.value.length > 0)
const readyCount = computed(() => [pages.value.length, products.value.length, replies.value.length, automations.value.length].filter(Boolean).length)
const steps = computed(() => [
  { done: pages.value.length > 0, icon:'mdi-facebook', title:locale.value==='lo'?'ເຊື່ອມ Facebook Page':'Connect Facebook Page', detail:pages.value.length ? `${pages.value.length} ${locale.value==='lo'?'Page ເຊື່ອມແລ້ວ':'Page(s) connected'}` : locale.value==='lo'?'ຈຳເປັນສຳລັບຮັບ Webhook':'Required to receive Webhooks', to:'/meta-pages' },
  { done: products.value.length > 0, icon:'mdi-package-variant-closed', title:locale.value==='lo'?'ສ້າງສິນຄ້າ':'Create a Product', detail:`${products.value.length} ${t('nav.products')}`, to:'/products' },
  { done: replies.value.length > 0, icon:'mdi-message-text-fast-outline', title:locale.value==='lo'?'ສ້າງຊຸດຂໍ້ຄວາມ':'Build a Reply Set', detail:`${replies.value.length} ${t('nav.replies')}`, to:'/replies' },
  { done: automations.value.length > 0, icon:'mdi-robot-happy-outline', title:locale.value==='lo'?'ສ້າງ Automation':'Create an Automation', detail:`${activeAutomations.value.length} ${locale.value==='lo'?'ເປີດໃຊ້ຢູ່':'active'}`, to:'/auto-replies' },
])
</script>

<template>
  <v-alert v-if="justConnected" type="success" variant="tonal" closable class="mb-4">{{ locale==='lo'?'ເຊື່ອມ Facebook Page ສຳເລັດ ແລະ Webhook ເປີດໃຊ້ແລ້ວ':'Facebook Page connected and Webhook is active.' }}</v-alert>
  <section class="onboarding-hero">
    <div><small>{{ locale==='lo'?'ສະຖານະ WORKSPACE':'WORKSPACE STATUS' }}</small><h2>{{ readyCount===4 ? (locale==='lo'?'ພ້ອມເຮັດວຽກແລ້ວ':'Ready to work') : (locale==='lo'?'ກຽມ Automation ໃຫ້ພ້ອມ':'Finish your Automation setup') }}</h2><p>{{ locale==='lo'?'ຕິດຕາມສະຖານະຈິງ ໂດຍບໍ່ໃຊ້ຕົວເລກຕົວຢ່າງ':'Live workspace status without placeholder business metrics.' }}</p></div>
    <div class="progress"><strong>{{ readyCount }}/4</strong><span>{{ locale==='lo'?'ຂັ້ນຕອນສຳເລັດ':'steps complete' }}</span><div><i :style="{width:`${readyCount*25}%`}"/></div></div>
  </section>

  <section class="overview-grid">
    <article class="setup-panel"><header><div><small>QUICK START</small><h3>{{ locale==='lo'?'ເລີ່ມໃຊ້ງານຕາມລຳດັບ':'Complete setup in order' }}</h3></div><v-btn variant="text" to="/automation-guide" append-icon="mdi-arrow-right">{{ t('auto.guide') }}</v-btn></header><div class="step-list"><NuxtLink v-for="(item,index) in steps" :key="item.to" :to="item.to"><span :class="{done:item.done}"><v-icon :icon="item.done?'mdi-check':item.icon"/></span><div><strong>{{ index+1 }}. {{ item.title }}</strong><small>{{ item.detail }}</small></div><v-icon icon="mdi-chevron-right"/></NuxtLink></div></article>
    <aside class="live-panel"><small>LIVE STATUS</small><h3>{{ locale==='lo'?'ລະບົບປັດຈຸບັນ':'Current system' }}</h3><div class="status-row"><span><v-icon icon="mdi-facebook"/></span><div><strong>{{ pagePending?'…':pages.length }}</strong><small>Connected Pages</small></div></div><div class="status-row"><span><v-icon icon="mdi-robot-happy-outline"/></span><div><strong>{{ activeAutomations.length }}</strong><small>Active Automations</small></div></div><div class="status-row"><span><v-icon icon="mdi-target"/></span><div><strong>{{ automations.reduce((sum,item)=>sum+item.targets.length,0) }}</strong><small>Connected Targets</small></div></div><v-btn block color="primary" to="/auto-replies" prepend-icon="mdi-robot-happy-outline">{{ t('auto.title') }}</v-btn></aside>
  </section>
</template>

<style scoped>
.onboarding-hero{display:flex;justify-content:space-between;align-items:center;gap:24px;margin-bottom:16px;padding:28px;background:linear-gradient(135deg,var(--color-primary-deep),#4d43bf 65%,#6458ec);border-radius:var(--radius-lg);color:white;overflow:hidden}.onboarding-hero small,.setup-panel header small,.live-panel>small{font-size:10px;font-weight:850;letter-spacing:.13em}.onboarding-hero small{color:var(--color-brand-caption)}.onboarding-hero h2{margin:7px 0 5px;font-size:clamp(24px,3vw,34px)}.onboarding-hero p{margin:0;color:rgba(255,255,255,.7);font-size:13px}.progress{flex:0 0 190px;display:grid;gap:4px;padding:18px;background:rgba(255,255,255,.1);border:1px solid rgba(255,255,255,.12);border-radius:15px}.progress strong{font-size:25px}.progress span{color:rgba(255,255,255,.72);font-size:11px}.progress>div{height:6px;margin-top:6px;background:rgba(255,255,255,.18);border-radius:99px;overflow:hidden}.progress i{display:block;height:100%;background:white;border-radius:inherit}.overview-grid{display:grid;grid-template-columns:minmax(0,1.6fr) minmax(260px,.7fr);gap:16px}.setup-panel,.live-panel{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.setup-panel header{display:flex;justify-content:space-between;align-items:center;gap:16px;padding:20px;border-bottom:1px solid var(--color-border)}.setup-panel header small,.live-panel>small{color:var(--color-primary)}.setup-panel h3,.live-panel h3{margin:5px 0 0}.step-list{display:grid}.step-list a{min-height:76px;display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:14px;padding:14px 20px;border-bottom:1px solid var(--color-border-subtle);color:var(--color-text);text-decoration:none}.step-list a:last-child{border:0}.step-list a:hover{background:var(--color-surface-soft)}.step-list a>span{width:44px;height:44px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:13px}.step-list a>span.done{color:var(--color-success);background:var(--color-success-soft)}.step-list a>div{display:grid;gap:4px}.step-list a small{color:var(--color-text-secondary)}.live-panel{align-self:start;padding:20px}.status-row{display:flex;align-items:center;gap:12px;padding:16px 0;border-bottom:1px solid var(--color-border-subtle)}.status-row>span{width:40px;height:40px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:11px}.status-row>div{display:grid}.status-row strong{font-size:20px}.status-row small{color:var(--color-text-secondary)}.live-panel :deep(.v-btn){margin-top:18px}@media(max-width:900px){.overview-grid{grid-template-columns:1fr}.live-panel{width:100%}}@media(max-width:640px){.onboarding-hero{display:grid;padding:21px}.progress{width:100%;flex-basis:auto}.setup-panel header{align-items:flex-start;padding:16px}.setup-panel header :deep(.v-btn){min-width:auto}.step-list a{padding:12px 14px}.step-list a>span{width:40px;height:40px}}
</style>
