<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
interface MetaPage { id:string;name:string;category:string;connected:boolean;webhookStatus:string }
interface PageList { items:MetaPage[];mode:string }
const route = useRoute()
const { data: metaPages, pending: metaPending, error: metaError } = await useApi<PageList>('/proxy/api/v1/meta/pages')
const connectedPages = computed(() => metaPages.value?.items.filter(page => page.connected) ?? [])
const justConnected = computed(() => route.query.meta === 'connected' && connectedPages.value.length > 0)
const stats = [
  { label:'Total revenue', value:'$24,680', change:'+12.5%', icon:'mdi-wallet-outline', color:'var(--color-primary)', tint:'var(--color-primary-softer)' },
  { label:'Active customers', value:'1,429', change:'+8.2%', icon:'mdi-account-group-outline', color:'var(--color-success)', tint:'var(--color-success-soft)' },
  { label:'New orders', value:'368', change:'+5.7%', icon:'mdi-shopping-outline', color:'var(--color-warning)', tint:'var(--color-warning-soft)' },
]
const activities = [
  ['New order received','Order #2048 · 5 min ago','mdi-cart-outline'],
  ['New customer','Somchai joined · 42 min ago','mdi-account-plus-outline'],
  ['Order shipped','Order #2039 · 2 hrs ago','mdi-package-variant'],
]
</script>

<template>
  <v-alert v-if="justConnected" type="success" variant="tonal" closable class="mb-4">Facebook Page connected successfully. The shared Webhook is active.</v-alert>
  <section class="facebook-status" :class="{ connected: connectedPages.length }">
    <div class="facebook-status-icon"><v-icon :icon="connectedPages.length ? 'mdi-check-circle' : 'mdi-facebook'" size="27"/></div>
    <div class="facebook-status-copy">
      <small>FACEBOOK CONNECTION</small>
      <strong v-if="metaPending">Checking Page connection…</strong>
      <strong v-else-if="metaError">Unable to verify the connection</strong>
      <strong v-else-if="connectedPages.length">{{ connectedPages.length }} Page{{ connectedPages.length > 1 ? 's' : '' }} connected</strong>
      <strong v-else>No Facebook Page connected</strong>
      <p v-if="connectedPages.length">{{ connectedPages.map(page => page.name).join(', ') }} · Webhook active</p>
      <p v-else-if="!metaPending">Connect a Page to start receiving Webhook events.</p>
    </div>
    <v-btn color="primary" :variant="connectedPages.length ? 'tonal' : 'flat'" to="/meta-pages" prepend-icon="mdi-cog-outline">{{ connectedPages.length ? 'Manage Pages' : 'Connect Page' }}</v-btn>
  </section>
  <section class="stats" aria-label="Business summary">
    <article v-for="s in stats" :key="s.label"><div :style="{color:s.color,background:s.tint}"><v-icon :icon="s.icon"/></div><small>{{s.label}}</small><strong>{{s.value}}</strong><p><b>{{s.change}}</b> from last month</p></article>
  </section>
  <section class="content-grid">
    <article class="panel chart-panel"><div class="head"><div><h2>Revenue overview</h2><p>Your earnings over the last 6 months</p></div><button>Last 6 months <v-icon icon="mdi-chevron-down" size="17"/></button></div><div class="plot"><div class="lines"><i/><i/><i/><i/></div><svg viewBox="0 0 620 220" preserveAspectRatio="none" aria-label="Revenue trend"><defs><linearGradient id="fill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="var(--color-primary)" stop-opacity=".25"/><stop offset="1" stop-color="var(--color-primary)" stop-opacity="0"/></linearGradient></defs><path d="M0 185C80 170 90 135 155 148S260 82 315 105 405 42 470 75 545 20 620 38V220H0Z" fill="url(#fill)"/><path d="M0 185C80 170 90 135 155 148S260 82 315 105 405 42 470 75 545 20 620 38" fill="none" stroke="var(--color-primary)" stroke-width="4" stroke-linecap="round"/></svg><div class="months"><span>Apr</span><span>May</span><span>Jun</span><span>Jul</span><span>Aug</span><span>Sep</span></div></div></article>
    <article class="panel activity"><div class="head"><div><h2>Recent activity</h2><p>Latest updates</p></div></div><div v-for="(a,i) in activities" :key="a[0]" class="activity-row"><span :class="`color-${i}`"><v-icon :icon="a[2]" size="18"/></span><p><strong>{{a[0]}}</strong><small>{{a[1]}}</small></p></div><button class="view-all">View all activity <v-icon icon="mdi-arrow-right" size="17"/></button></article>
  </section>
</template>

<style scoped>
.facebook-status{display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:15px;margin-bottom:18px;padding:18px 20px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:18px}.facebook-status.connected{border-color:color-mix(in srgb,var(--color-success) 32%,var(--color-border))}.facebook-status-icon{width:48px;height:48px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-softer);border-radius:14px}.facebook-status.connected .facebook-status-icon{color:var(--color-success);background:var(--color-success-soft)}.facebook-status-copy{display:grid;gap:2px;min-width:0}.facebook-status-copy small{color:var(--color-text-muted);font-size:9px;font-weight:800;letter-spacing:.12em}.facebook-status-copy strong{font-size:14px}.facebook-status-copy p{margin:0;color:var(--color-text-secondary);font-size:11px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.stats{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:18px}.stats article,.panel{background:var(--color-surface);border:1px solid var(--color-border);border-radius:18px}.stats article{padding:21px;display:grid;grid-template-columns:48px 1fr;column-gap:14px;min-width:0}.stats article>div{width:46px;height:46px;grid-row:span 3;display:grid;place-items:center;border-radius:14px}.stats small{color:var(--color-text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.stats strong{font-size:24px}.stats p{margin:0;color:var(--color-text-faint);font-size:9px}.stats b{color:var(--color-success)}.content-grid{display:grid;grid-template-columns:minmax(0,1.7fr) minmax(280px,.8fr);gap:18px;margin-top:18px}.panel{padding:25px;min-width:0}.head{display:flex;justify-content:space-between;gap:16px}.head h2{margin:0 0 5px;font-size:15px}.head p{margin:0;color:var(--color-text-muted);font-size:10px}.head button,.view-all{border:0;background:none;cursor:pointer}.head button{border:1px solid var(--color-border);border-radius:9px;padding:7px 10px;color:var(--color-text-secondary)}.plot{height:265px;position:relative;margin-top:25px}.plot svg{width:100%;height:220px;position:relative}.lines{position:absolute;inset:0 0 45px;display:flex;flex-direction:column;justify-content:space-between}.lines i{border-top:1px dashed var(--color-border)}.months{display:flex;justify-content:space-between;color:var(--color-text-faint);font-size:9px}.activity-row{display:flex;align-items:center;gap:11px;padding:16px 0;border-bottom:1px solid var(--color-border-subtle)}.activity-row>span{flex:0 0 36px;width:36px;height:36px;display:grid;place-items:center;border-radius:11px}.color-0{color:var(--color-primary);background:var(--color-primary-softer)}.color-1{color:var(--color-success);background:var(--color-success-soft)}.color-2{color:var(--color-warning);background:var(--color-warning-soft)}.activity-row p{margin:0;display:grid}.activity-row strong{font-size:11px}.activity-row small{color:var(--color-text-faint);font-size:9px}.view-all{display:flex;align-items:center;gap:6px;margin:18px auto 0;color:var(--color-primary);font-size:10px;font-weight:800}
@media(max-width:1100px){.content-grid{grid-template-columns:1fr}}
@media(max-width:800px){.stats{grid-template-columns:1fr}}
@media(max-width:640px){.facebook-status{grid-template-columns:auto 1fr}.facebook-status :deep(.v-btn){grid-column:1/-1;width:100%}.stats{gap:12px}.content-grid{gap:12px;margin-top:12px}.panel{padding:20px}.chart-panel{overflow-x:auto}.plot{min-width:520px}.stats article{padding:18px}}
</style>
