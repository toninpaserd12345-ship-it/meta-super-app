<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
interface MetaPage { id:string;name:string;category:string;pictureUrl?:string;tokenReady:boolean;connected:boolean;webhookStatus:string;tokenExpiresAt?:number;dataAccessExpiresAt?:number;grantedPermissions?:string[] }
interface PageList { items:MetaPage[];mode:string }
interface OAuthStart { authorizationUrl:string }
const { can } = useAuth()
const route = useRoute()
if (!can('pages:read')) throw createError({statusCode:403,statusMessage:'You do not have permission to view Meta Pages.'})
const { data, pending, error, refresh } = await useApi<PageList>('/proxy/api/v1/meta/pages')
const toggling = ref<string|null>(null)
const authorizing = ref(false)
const activating = ref(false)
const pickerOpen = ref(false)
const permissionsOpen = ref(false)
const selectedPermissions = ref<string[]>([])
const optionalPermissions=[
  {value:'pages_manage_posts',label:'Manage posts',description:'Create and manage Page posts'},
  {value:'read_insights',label:'Insights',description:'Read Page performance insights'},
]
const search = ref('')
const selected = ref<string[]>([])
const statusFilter = ref<'all'|'active'|'available'>('all')
const message = ref('')
const messageType = ref<'success'|'error'>('success')
const pages=computed(()=>data.value?.items||[])
const activePages=computed(()=>pages.value.filter(page=>page.connected))
const inactivePages=computed(()=>pages.value.filter(page=>!page.connected))
const availablePages=computed(()=>inactivePages.value.filter(page=>page.name.toLowerCase().includes(search.value.toLowerCase())))
const visiblePages=computed(()=>pages.value.filter(page=>statusFilter.value==='all'||(statusFilter.value==='active'?page.connected:!page.connected)))
if(route.query.meta==='connected'){messageType.value='success';message.value='Facebook connected successfully. Choose the Pages you want to activate.';pickerOpen.value=availablePages.value.length>0;await navigateTo('/meta-pages',{replace:true})}
if(route.query.meta==='error'){messageType.value='error';message.value=String(route.query.reason||'Facebook authorization failed.');await navigateTo('/meta-pages',{replace:true})}
function tokenExpiry(page:MetaPage){if(!page.tokenExpiresAt)return 'No fixed expiry reported';return `Expires ${new Intl.DateTimeFormat(undefined,{dateStyle:'medium'}).format(new Date(page.tokenExpiresAt*1000))}`}
async function connectFacebook(){
  authorizing.value=true
  try{
    const result=await $fetch<OAuthStart>('/api/proxy/api/v1/meta/oauth/start',{method:'POST',body:{permissions:selectedPermissions.value},timeout:10000})
    window.location.assign(result.authorizationUrl)
  }
  catch{messageType.value='error';message.value='Unable to start Facebook Login. Please sign in to this app again and retry.';authorizing.value=false}
}
function apiMessage(error:any,fallback:string){return error?.data?.message||error?.data?.statusMessage||error?.statusMessage||fallback}
async function setPageEnabled(page:MetaPage,enabled:boolean){
  if(!enabled&&!confirm(`Pause Webhook for “${page.name}”? New events from this Page will stop until it is enabled again.`))return
  toggling.value=page.id;message.value=''
  try{await $fetch(`/api/proxy/api/v1/meta/pages/${enabled?'connect':'disconnect'}`,{method:'POST',body:{pageId:page.id}});await refresh();messageType.value='success';message.value=`${page.name}: Webhook ${enabled?'enabled':'paused'}.`}
  catch(error:any){messageType.value='error';message.value=apiMessage(error,`Unable to ${enabled?'enable':'pause'} ${page.name}. Check the Page permission and try again.`);await refresh()}
  finally{toggling.value=null}
}
async function activateSelected(){
  if(!selected.value.length)return
  activating.value=true;message.value=''
  const chosen=[...selected.value]
  const results=await Promise.allSettled(chosen.map(pageId=>$fetch('/api/proxy/api/v1/meta/pages/connect',{method:'POST',body:{pageId}})))
  const successIds=chosen.filter((_,index)=>results[index]?.status==='fulfilled')
  const failed=chosen.length-successIds.length
  selected.value=selected.value.filter(id=>!successIds.includes(id));await refresh();activating.value=false
  if(!failed){pickerOpen.value=false;messageType.value='success';message.value=`${successIds.length} Page${successIds.length===1?'':'s'} activated. Webhooks are ready.`;return}
  messageType.value='error';message.value=`Activated ${successIds.length} of ${chosen.length} Pages. ${failed} failed because Meta rejected the Page permission.`
}
</script>

<template>
  <section class="intro"><div><p>META INTEGRATION</p><h2>Facebook Pages</h2><span>Connect an account once, then control each Page Webhook here.</span></div><div class="intro-actions"><v-chip :color="data?.mode==='live'?'success':'primary'" variant="tonal" :prepend-icon="data?.mode==='live'?'mdi-access-point':'mdi-flask-outline'">{{data?.mode==='live'?'Meta connected':'Mock mode'}}</v-chip><v-btn v-if="inactivePages.length" variant="outlined" prepend-icon="mdi-checkbox-multiple-marked-outline" @click="search='';pickerOpen=true">Select Pages</v-btn><v-btn v-if="data?.mode==='live'" color="primary" prepend-icon="mdi-facebook" :loading="authorizing" @click="permissionsOpen=true">Connect Facebook</v-btn></div></section>
  <v-alert v-if="message" :type="messageType" variant="tonal" closable class="mb-4" @click:close="message=''">{{message}}</v-alert>
  <v-alert v-if="error" type="error" variant="tonal">Unable to load Pages from the API.</v-alert>
  <section v-if="!error" class="connection-summary">
    <button :class="{active:statusFilter==='all'}" @click="statusFilter='all'"><v-icon icon="mdi-facebook"/><span><small>ALL PAGES</small><strong>{{pages.length}}</strong></span></button>
    <button :class="{active:statusFilter==='active'}" @click="statusFilter='active'"><v-icon icon="mdi-webhook"/><span><small>WEBHOOK ACTIVE</small><strong>{{activePages.length}}</strong></span></button>
    <button :class="{active:statusFilter==='available'}" @click="statusFilter='available'"><v-icon icon="mdi-power-plug-off-outline"/><span><small>NOT ACTIVE</small><strong>{{inactivePages.length}}</strong></span></button>
  </section>
  <section v-if="!error" class="page-grid" :aria-busy="pending">
    <v-skeleton-loader v-if="pending" v-for="i in 3" :key="i" type="card"/>
    <article v-for="page in visiblePages" v-else :key="page.id" class="page-card">
      <div class="page-avatar"><v-icon icon="mdi-facebook" size="28"/></div><div class="page-info"><h3>{{page.name}}</h3><p>{{page.category}}</p><small>Page ID · {{page.id}}</small><div class="token-row"><v-chip class="token-chip" :color="page.tokenReady?'success':'warning'" variant="tonal" size="x-small" :prepend-icon="page.tokenReady?'mdi-key-check':'mdi-key-alert'">{{page.tokenReady?'Access token ready':'Token unavailable'}}</v-chip><v-chip v-if="page.tokenReady" class="token-chip" color="info" variant="tonal" size="x-small" prepend-icon="mdi-clock-outline">{{tokenExpiry(page)}}</v-chip></div></div>
      <div class="page-action"><div v-if="page.connected" class="page-toggle"><span><strong>Webhook active</strong><small>Receiving events</small></span><v-switch :model-value="true" color="success" hide-details density="compact" :loading="toggling===page.id" :disabled="toggling!==null||!can('pages:connect')" :aria-label="`Pause Webhook for ${page.name}`" @update:model-value="value=>setPageEnabled(page,Boolean(value))"/></div><v-btn v-else color="primary" variant="flat" prepend-icon="mdi-play-circle-outline" :loading="toggling===page.id" :disabled="toggling!==null||!can('pages:connect')||!page.tokenReady" @click="setPageEnabled(page,true)">Enable Webhook</v-btn></div>
    </article>
    <v-alert v-if="!pending&&data?.mode==='live'&&!pages.length" type="info" variant="tonal" icon="mdi-facebook">
      <strong>No Facebook Pages found.</strong><br>Connect Facebook and allow the required Page permissions. Your Facebook account must have Page access.
    </v-alert>
    <v-alert v-else-if="!pending&&!visiblePages.length" type="info" variant="tonal">No Pages match this filter.</v-alert>
  </section>

  <v-dialog v-model="permissionsOpen" max-width="620">
    <v-card class="picker-card"><v-card-title class="picker-head"><div><small>FACEBOOK PERMISSIONS</small><h2>Choose features</h2></div><v-btn icon="mdi-close" variant="text" @click="permissionsOpen=false"/></v-card-title><v-card-text class="permission-body">
      <v-alert type="info" variant="tonal" density="compact">Page access, Webhooks and Messenger permissions are included automatically. Select only the additional features you need.</v-alert>
      <v-checkbox v-for="item in optionalPermissions" :key="item.value" v-model="selectedPermissions" :value="item.value" color="primary" hide-details><template #label><span class="permission-label"><strong>{{item.label}}</strong><small>{{item.description}}</small></span></template></v-checkbox>
      <p class="permission-note">Meta may require App Review and Advanced Access before these features work for public users.</p>
    </v-card-text><v-card-actions class="picker-actions"><v-btn variant="text" @click="selectedPermissions=[]">Required only</v-btn><v-spacer/><v-btn color="primary" prepend-icon="mdi-facebook" :loading="authorizing" @click="permissionsOpen=false;connectFacebook()">Continue with Facebook</v-btn></v-card-actions></v-card>
  </v-dialog>

  <v-dialog v-model="pickerOpen" max-width="920" persistent scrollable>
    <v-card class="picker-card">
      <v-card-title class="picker-head"><div><small>FACEBOOK</small><h2>Add connections</h2></div><v-btn icon="mdi-close" variant="text" :disabled="activating" @click="pickerOpen=false"/></v-card-title>
      <v-card-text class="picker-body">
        <div class="picker-toolbar"><div><strong>Select Pages to activate</strong><span>{{selected.length}} selected</span></div><v-text-field v-model="search" prepend-inner-icon="mdi-magnify" label="Search Pages" density="compact" hide-details clearable/></div>
        <div v-if="availablePages.length" class="picker-grid">
          <label v-for="page in availablePages" :key="page.id" class="picker-page" :class="{'is-selected':selected.includes(page.id)}">
            <v-checkbox-btn v-model="selected" :value="page.id" color="primary"/>
            <v-avatar size="52" color="primary"><v-img v-if="page.pictureUrl" :src="page.pictureUrl"/><v-icon v-else icon="mdi-facebook" color="white"/></v-avatar>
            <span><strong>{{page.name}}</strong><small>{{page.category}} · {{page.id}}</small></span>
          </label>
        </div>
        <v-alert v-else type="info" variant="tonal">All available Pages are already active.</v-alert>
      </v-card-text>
      <v-card-actions class="picker-actions"><v-btn variant="text" :disabled="activating" @click="selected=[]">Clear selection</v-btn><v-spacer/><v-btn color="primary" prepend-icon="mdi-lightning-bolt" :disabled="!selected.length" :loading="activating" @click="activateSelected">Activate {{selected.length||''}}</v-btn></v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.intro{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;margin-bottom:16px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.intro-actions{display:flex;align-items:center;gap:10px;flex-wrap:wrap;justify-content:flex-end}.intro p{margin:0 0 8px;color:var(--color-primary);font-size:var(--text-xs);font-weight:800;letter-spacing:.13em}.intro h2{margin:0 0 8px;font-size:var(--text-xl)}.intro span{color:var(--color-text-secondary);font-size:var(--text-md)}.connection-summary{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-bottom:16px}.connection-summary button{min-height:76px;display:flex;align-items:center;gap:13px;padding:14px 17px;border:1px solid var(--color-border);border-radius:var(--radius-lg);color:var(--color-text-secondary);background:var(--color-surface);text-align:left;cursor:pointer;transition:var(--transition-fast)}.connection-summary button:hover,.connection-summary button.active{color:var(--color-primary);border-color:color-mix(in srgb,var(--color-primary) 35%,var(--color-border));background:var(--color-primary-soft)}.connection-summary button>i{padding:10px;border-radius:11px;background:var(--color-background)}.connection-summary span{display:grid}.connection-summary small{font-size:9px;font-weight:800;letter-spacing:.08em}.connection-summary strong{color:var(--color-text);font-size:20px}.page-grid{display:grid;gap:12px}.page-card{display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:16px;padding:20px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-avatar{width:52px;height:52px;display:grid;place-items:center;color:var(--color-surface);background:var(--color-primary);border-radius:16px}.page-info h3{margin:0;font-size:15px}.page-info p{margin:3px 0;color:var(--color-text-secondary);font-size:12px}.page-info small{color:var(--color-text-muted);font-size:10px}.token-row{display:flex;gap:6px;flex-wrap:wrap}.token-chip{display:flex!important;width:max-content;margin-top:7px}.page-toggle{min-width:196px;display:flex;align-items:center;justify-content:flex-end;gap:12px;padding:8px 10px 8px 14px;color:var(--color-success);background:var(--color-success-soft);border-radius:13px}.page-toggle>span{display:grid}.page-toggle strong{font-size:11px}.page-toggle small{color:var(--color-text-secondary);font-size:9px}.permission-body{display:grid;gap:8px;padding:22px 24px}.permission-label{display:grid;margin-left:6px}.permission-label small,.permission-note{color:var(--color-text-muted);font-size:11px}.permission-note{margin:8px 0 0}
@media(max-width:640px){.intro{display:grid;padding:20px}.intro-actions{display:grid;grid-template-columns:1fr 1fr;width:100%}.intro-actions :deep(.v-chip){grid-column:1/-1;width:max-content}.intro-actions :deep(.v-btn:last-child){grid-column:1/-1}.connection-summary{grid-template-columns:repeat(3,1fr);gap:7px}.connection-summary button{min-height:68px;justify-content:center;padding:9px 5px}.connection-summary button>i{display:none}.connection-summary small{font-size:7px}.connection-summary strong{font-size:18px}.page-card{grid-template-columns:auto 1fr;padding:17px}.page-action{grid-column:1/-1}.page-action :deep(.v-btn),.page-toggle{width:100%}.page-toggle{justify-content:space-between}.token-row{gap:3px}.token-chip{font-size:8px!important}}
.picker-card{border-radius:var(--radius-lg)!important}.picker-head{display:flex;align-items:center;justify-content:space-between;padding:22px 24px;border-bottom:1px solid var(--color-border)}.picker-head small{color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.14em}.picker-head h2{margin:2px 0 0;font-size:20px}.picker-body{padding:22px 24px!important}.picker-toolbar{display:grid;grid-template-columns:1fr minmax(240px,320px);align-items:center;gap:20px;margin-bottom:18px}.picker-toolbar div{display:grid;gap:3px}.picker-toolbar span{color:var(--color-text-muted);font-size:12px}.picker-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.picker-page{display:grid;grid-template-columns:auto auto 1fr;align-items:center;gap:12px;padding:14px;border:1px solid var(--color-border);border-radius:14px;cursor:pointer;transition:.18s ease}.picker-page:hover,.picker-page.is-selected{border-color:var(--color-primary);background:var(--color-primary-soft)}.picker-page>span{display:grid;min-width:0}.picker-page strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.picker-page small{overflow:hidden;color:var(--color-text-muted);font-size:11px;text-overflow:ellipsis;white-space:nowrap}.picker-actions{padding:16px 24px;border-top:1px solid var(--color-border)}
@media(max-width:700px){.picker-toolbar,.picker-grid{grid-template-columns:1fr}.picker-head,.picker-actions{padding-inline:16px}}
</style>
