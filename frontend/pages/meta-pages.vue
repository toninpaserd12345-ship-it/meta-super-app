<script setup lang="ts">
import type { AutomationFlow, ItemsResponse, Product, ReplySet } from '~/types/automation'
definePageMeta({ middleware: 'auth' })
interface MetaPage { id:string;name:string;category:string;pictureUrl?:string;phoneNumber?:string;tokenReady:boolean;connected:boolean;webhookStatus:string;tokenExpiresAt?:number;dataAccessExpiresAt?:number;grantedPermissions?:string[] }
interface WhatsAppDiagnostics { state:string;message:string;requiredPermissions:string[];grantedPermissions?:string[];missingPermissions?:string[];issues?:string[];businessCount:number;whatsAppBusinessAccountCount:number;phoneNumberCount:number }
interface PageList { items:MetaPage[];mode:string;whatsappDiagnostics?:WhatsAppDiagnostics }
interface OAuthStart { authorizationUrl:string }
interface WhatsAppSignupConfig { appId:string;configId:string;version:string;enabled:boolean }
interface WhatsAppSignupSession { businessId?:string;wabaId?:string;phoneNumberId?:string }
const { can } = useAuth()
const route = useRoute()
if (!can('pages:read')) throw createError({statusCode:403,statusMessage:'You do not have permission to view Meta Pages.'})
const { data, pending, error, refresh } = await useApi<PageList>('/proxy/api/v1/meta/pages')
const { data: automationData } = await useApi<ItemsResponse<AutomationFlow>>('/proxy/api/v1/automations')
const { data: productData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: replySetData } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const loadErrorMessage = computed(() => {
  const value:any = error.value
  return value?.data?.error?.message || value?.data?.message || value?.statusMessage || 'Unable to load Pages from Meta. Reconnect Facebook and try again.'
})
const needsReconnect = computed(() => {
  const value:any = error.value
  return value?.data?.error?.code === 'meta_reconnect_required' || Number(value?.statusCode || value?.status) === 428
})
const toggling = ref<string|null>(null)
const authorizing = ref(false)
const activating = ref(false)
const whatsAppConnecting = ref(false)
const whatsAppWaitHint = ref(false)
const whatsAppConfig = ref<WhatsAppSignupConfig|null>(null)
const whatsAppSession = ref<WhatsAppSignupSession>({})
const whatsAppCode = ref('')
const whatsAppDetailsOpen = ref(false)

const pickerOpen = ref(false)
const activeView = ref('pending')
const expandedPages = ref<string[]>([])
const permissionsOpen = ref(false)
const selectedPermissions = ref<string[]>([])
const optionalPermissions=[
  {value:'pages_manage_posts',label:'Manage posts',description:'Create and manage Page posts'},
  {value:'read_insights',label:'Insights',description:'Read Page performance insights'},
]
const search = ref('')
const selected = ref<string[]>([])
const statusFilter = ref<'all'|'active'|'available'>('all')
const mainPlatformFilter = ref<'all'|'whatsapp'>('all')
const message = ref('')
const messageType = ref<'success'|'error'>('success')
const failedPictures = ref<Record<string, boolean>>({})
const pictureAttempts = ref<Record<string, number>>({})
const pages=computed(()=>data.value?.items||[])
const activePages=computed(()=>pages.value.filter(page=>page.connected))
const inactivePages=computed(()=>pages.value.filter(page=>!page.connected))
const whatsappPages=computed(()=>pages.value.filter(page=>page.category.toLowerCase().includes('whatsapp')))
const whatsappDiagnostics=computed(()=>data.value?.whatsappDiagnostics)

const availablePages=computed(()=>inactivePages.value.filter(page=>!search.value||page.name.toLowerCase().includes(search.value.toLowerCase())))
const productsById=computed(()=>new Map((productData.value?.items||[]).map(item=>[item.id,item])))
const replySetsById=computed(()=>new Map((replySetData.value||[]).map(item=>[item.id,item])))
const pageAutomations=(pageId:string)=>(automationData.value?.items||[]).filter(item=>item.pageId===pageId)
const pageRelationCounts=(pageId:string)=>{
  const flows=pageAutomations(pageId)
  return {automations:flows.length,active:flows.filter(item=>item.isActive).length,targets:flows.reduce((sum,item)=>sum+item.targets.length,0)}
}
const togglePageDetails=(pageId:string)=>{
  expandedPages.value=expandedPages.value.includes(pageId)?expandedPages.value.filter(id=>id!==pageId):[...expandedPages.value,pageId]
}

const visiblePages=computed(()=>pages.value.filter(page=>{
  const matchesStatus=statusFilter.value==='all'||(statusFilter.value==='active'?page.connected:!page.connected)
  const matchesPlatform=mainPlatformFilter.value==='all'||page.category.toLowerCase().includes('whatsapp')
  return matchesStatus&&matchesPlatform
}))
const pagePictureUrl=(page:MetaPage)=>`/api/meta/page-picture/${encodeURIComponent(page.id)}?attempt=${pictureAttempts.value[page.id]||0}`
const pictureAvailable=(page:MetaPage)=>page.tokenReady&&!failedPictures.value[page.id]
const handlePictureError=(pageId:string)=>{
  const attempt=pictureAttempts.value[pageId]||0
  if(attempt<2){pictureAttempts.value={...pictureAttempts.value,[pageId]:attempt+1};return}
  failedPictures.value={...failedPictures.value,[pageId]:true}
}
if(route.query.meta==='connected'){messageType.value='success';message.value='Facebook connected successfully. Choose the accounts you want to activate.';pickerOpen.value=availablePages.value.length>0;await navigateTo('/meta-pages',{replace:true})}
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
function apiMessage(error:any,fallback:string){return error?.data?.error?.message||error?.data?.message||error?.data?.statusMessage||error?.statusMessage||fallback}
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

declare global {
  interface Window {
    FB?: { init:(options:Record<string,unknown>)=>void;login:(callback:(response:any)=>void,options:Record<string,unknown>)=>void }
    fbAsyncInit?:()=>void
  }
}

let whatsAppMessageHandler:((event:MessageEvent)=>void)|null=null
let whatsAppHintTimer:ReturnType<typeof setTimeout>|null=null
let whatsAppTimeoutTimer:ReturnType<typeof setTimeout>|null=null
function clearWhatsAppTimers(){
  if(whatsAppHintTimer)clearTimeout(whatsAppHintTimer)
  if(whatsAppTimeoutTimer)clearTimeout(whatsAppTimeoutTimer)
  whatsAppHintTimer=null;whatsAppTimeoutTimer=null;whatsAppWaitHint.value=false
}
function startWhatsAppTimers(){
  clearWhatsAppTimers()
  whatsAppHintTimer=setTimeout(()=>{whatsAppWaitHint.value=true},8000)
  whatsAppTimeoutTimer=setTimeout(()=>{
    whatsAppConnecting.value=false;whatsAppCode.value='';whatsAppSession.value={};whatsAppWaitHint.value=false
    messageType.value='error'
    message.value='Meta did not finish the connection. Add meta-super-app-frontend.pages.dev to Allowed Domains for the JavaScript SDK in Meta, then try again.'
  },120000)
}
async function loadFacebookSDK(config:WhatsAppSignupConfig){
  if(window.FB)return
  await new Promise<void>((resolve,reject)=>{
    const existing=document.getElementById('facebook-jssdk') as HTMLScriptElement|null
    window.fbAsyncInit=()=>{window.FB?.init({appId:config.appId,cookie:true,xfbml:false,version:config.version});resolve()}
    if(existing){if(window.FB)resolve();return}
    const script=document.createElement('script');script.id='facebook-jssdk';script.async=true;script.defer=true;script.crossOrigin='anonymous';script.src='https://connect.facebook.net/en_US/sdk.js';script.onerror=()=>reject(new Error('Facebook SDK could not be loaded'));document.head.appendChild(script)
  })
}
async function finishWhatsAppSignup(){
  if(!whatsAppCode.value||!whatsAppSession.value.wabaId)return
  clearWhatsAppTimers()
  const code=whatsAppCode.value;const session={...whatsAppSession.value};whatsAppCode.value='';whatsAppSession.value={}
  try{
    await $fetch('/api/proxy/api/v1/meta/whatsapp/signup/complete',{method:'POST',body:{code,businessId:session.businessId||'',wabaId:session.wabaId,phoneNumberId:session.phoneNumberId||''},timeout:30000})
    await refresh();messageType.value='success';message.value='WhatsApp connected. The webhook is subscribed and ready to receive messages.'
  }catch(error:any){messageType.value='error';message.value=apiMessage(error,'WhatsApp setup could not be completed. Check the selected Business Account and phone number.')}
  finally{whatsAppConnecting.value=false}
}
async function connectWhatsApp(){
  if(!can('pages:connect'))return
  whatsAppConnecting.value=true;message.value='';whatsAppCode.value='';whatsAppSession.value={}
  try{
    const config=await $fetch<WhatsAppSignupConfig>('/api/proxy/api/v1/meta/whatsapp/signup/config')
    whatsAppConfig.value=config
    if(!config.enabled)throw new Error('WhatsApp Embedded Signup is not configured on the API server.')
    await loadFacebookSDK(config)
    if(!window.FB)throw new Error('Facebook SDK is unavailable.')
    startWhatsAppTimers()
    window.FB.login((response:any)=>{
      const code=response?.authResponse?.code
      if(!code){clearWhatsAppTimers();whatsAppConnecting.value=false;messageType.value='error';message.value='WhatsApp connection was cancelled or Meta did not return an authorization code.';return}
      whatsAppCode.value=String(code);void finishWhatsAppSignup()
    },{config_id:config.configId,response_type:'code',override_default_response_type:true})
  }catch(error:any){clearWhatsAppTimers();whatsAppConnecting.value=false;messageType.value='error';message.value=error?.message||'Unable to start WhatsApp connection.'}
}
function cancelWhatsAppConnection(){
  clearWhatsAppTimers()
  whatsAppConnecting.value=false
  whatsAppCode.value=''
  whatsAppSession.value={}
  message.value=''
}
onMounted(()=>{
  void $fetch<WhatsAppSignupConfig>('/api/proxy/api/v1/meta/whatsapp/signup/config').then(config=>{whatsAppConfig.value=config}).catch(()=>{})
  whatsAppMessageHandler=(event:MessageEvent)=>{
    if(!['https://www.facebook.com','https://web.facebook.com'].includes(event.origin))return
    let payload:any=event.data
    if(typeof payload==='string'){try{payload=JSON.parse(payload)}catch{return}}
    if(payload?.type!=='WA_EMBEDDED_SIGNUP')return
    if(payload.event==='CANCEL'){clearWhatsAppTimers();whatsAppConnecting.value=false;messageType.value='error';message.value='WhatsApp connection was cancelled.';return}
    if(payload.event!=='FINISH')return
    const info=payload.data||{}
    whatsAppSession.value={businessId:String(info.business_id||''),wabaId:String(info.waba_id||''),phoneNumberId:String(info.phone_number_id||'')}
    void finishWhatsAppSignup()
  }
  window.addEventListener('message',whatsAppMessageHandler)
})
onBeforeUnmount(()=>{clearWhatsAppTimers();if(whatsAppMessageHandler)window.removeEventListener('message',whatsAppMessageHandler)})
</script>

<template>
  <section class="intro"><div><p>META CHANNELS</p><h2>Facebook</h2><span>ຈັດການບັນຊີ, Webhook ແລະຄວາມສຳພັນກັບ Automation ໃນບ່ອນດຽວ</span></div><div class="intro-actions"><v-chip :color="error?'error':'success'" variant="tonal" :prepend-icon="error?'mdi-alert-circle-outline':'mdi-access-point'">{{error?'ການເຊື່ອມຕໍ່ຕ້ອງກວດສອບ':'ພ້ອມໃຊ້ງານ'}}</v-chip><v-btn v-if="inactivePages.length" variant="outlined" prepend-icon="mdi-checkbox-multiple-marked-outline" @click="search='';pickerOpen=true">ເລືອກບັນຊີ</v-btn><v-btn v-if="false" color="success" :prepend-icon="whatsAppConnecting?'mdi-loading':'mdi-whatsapp'" :disabled="authorizing||whatsAppConnecting" @click="connectWhatsApp">{{whatsAppConnecting?'ກຳລັງລໍຖ້າ Meta…':'ເຊື່ອມຕໍ່ WhatsApp'}}</v-btn><v-btn color="primary" prepend-icon="mdi-facebook" :loading="authorizing" :disabled="whatsAppConnecting" @click="permissionsOpen=true">{{needsReconnect?'ເຊື່ອມ Facebook ໃໝ່':'ເຊື່ອມຕໍ່ Facebook'}}</v-btn></div></section>
  <v-alert v-if="message" :type="messageType" variant="tonal" closable class="mb-4" @click:close="message=''">{{message}}</v-alert>
  <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
    <strong>{{loadErrorMessage}}</strong><br>
    <span v-if="needsReconnect">The saved Facebook authorization is no longer usable. Authorize once more, then the system will renew it automatically while it remains valid.</span>
    <span v-else>The API could not load the Meta connection. Retry after a moment; if it continues, reconnect Facebook.</span>
    <div class="mt-3 d-flex ga-2 flex-wrap"><v-btn color="primary" prepend-icon="mdi-facebook" :loading="authorizing" @click="permissionsOpen=true">Reconnect Facebook</v-btn><v-btn variant="outlined" prepend-icon="mdi-refresh" @click="refresh()">Retry</v-btn></div>
  </v-alert>
  <section v-if="!error" class="connection-summary">
    <button :class="{active:statusFilter==='all'&&mainPlatformFilter==='all'}" @click="statusFilter='all';mainPlatformFilter='all'"><v-icon icon="mdi-facebook"/><span><small>ALL ACCOUNTS</small><strong>{{pages.length}}</strong></span></button>
    <button v-if="false" :class="{active:mainPlatformFilter==='whatsapp'}" @click="statusFilter='all';mainPlatformFilter='whatsapp'"><v-icon icon="mdi-whatsapp"/><span><small>WHATSAPP NUMBERS</small><strong>{{whatsappPages.length}}</strong></span></button>
    <button :class="{active:statusFilter==='active'&&mainPlatformFilter==='all'}" @click="statusFilter='active';mainPlatformFilter='all'"><v-icon icon="mdi-webhook"/><span><small>WEBHOOK ACTIVE</small><strong>{{activePages.length}}</strong></span></button>
    <button :class="{active:statusFilter==='available'&&mainPlatformFilter==='all'}" @click="statusFilter='available';mainPlatformFilter='all'"><v-icon icon="mdi-power-plug-off-outline"/><span><small>NOT ACTIVE</small><strong>{{inactivePages.length}}</strong></span></button>
  </section>
  <section v-if="false" class="whatsapp-setup">
    <div class="whatsapp-setup-icon"><v-icon icon="mdi-whatsapp" size="34"/></div>
    <div class="whatsapp-setup-copy">
      <small>WHATSAPP SETUP</small>
      <h3>Connect a WhatsApp number</h3>
      <p>No WhatsApp number is connected to this workspace yet. Meta will guide you through choosing the Business Account and phone number.</p>
      <ol class="setup-steps">
        <li><span>1</span>Continue with Meta</li>
        <li><span>2</span>Choose your WhatsApp account</li>
        <li><span>3</span>Webhook turns on automatically</li>
      </ol>
      <div class="setup-actions">
        <v-btn color="success" :prepend-icon="whatsAppConnecting?'mdi-loading':'mdi-whatsapp'" :disabled="whatsAppConnecting" @click="connectWhatsApp">
          {{whatsAppConnecting?'Waiting for Meta…':'Connect WhatsApp'}}
        </v-btn>
        <v-btn v-if="whatsAppConnecting" variant="text" @click="cancelWhatsAppConnection">Cancel</v-btn>
        <v-btn v-if="whatsappDiagnostics" variant="text" append-icon="mdi-chevron-down" @click="whatsAppDetailsOpen=!whatsAppDetailsOpen">{{whatsAppDetailsOpen?'Hide technical details':'Technical details'}}</v-btn>
      </div>
      <v-alert v-if="whatsAppConnecting&&whatsAppWaitHint" type="warning" variant="tonal" density="compact" class="signup-wait-hint">
        Complete the Meta popup. If it shows “Unknown JSSDK host domain”, add <code>meta-super-app-frontend.pages.dev</code> to Meta’s Allowed Domains for the JavaScript SDK, then cancel and retry.
      </v-alert>
      <v-expand-transition>
        <div v-if="whatsAppDetailsOpen&&whatsappDiagnostics" class="diagnostic-details">
          <div class="diagnostic-counts">
            <span>Business portfolios <strong>{{whatsappDiagnostics.businessCount}}</strong></span>
            <span>WhatsApp accounts <strong>{{whatsappDiagnostics.whatsAppBusinessAccountCount}}</strong></span>
            <span>Phone numbers <strong>{{whatsappDiagnostics.phoneNumberCount}}</strong></span>
          </div>
          <p>{{whatsappDiagnostics.message}}</p>
          <p v-if="whatsappDiagnostics.missingPermissions?.length">Missing permissions: <code>{{whatsappDiagnostics.missingPermissions.join(', ')}}</code></p>
          <ul v-if="whatsappDiagnostics.issues?.length"><li v-for="issue in whatsappDiagnostics.issues" :key="issue"><code>{{issue}}</code></li></ul>
        </div>
      </v-expand-transition>
    </div>
  </section>
  <section v-if="!error" class="page-grid" :aria-busy="pending">
    <v-skeleton-loader v-if="pending" v-for="i in 3" :key="i" type="card"/>
    <article v-for="page in visiblePages" v-else :key="page.id" class="page-card">
      <div class="page-avatar">
        <v-img v-if="pictureAvailable(page)" :src="pagePictureUrl(page)" :alt="`${page.name} profile picture`" cover @error="handlePictureError(page.id)"></v-img>
        <v-icon v-else-if="page.category.toLowerCase().includes('whatsapp')" icon="mdi-whatsapp" color="success" size="28"/>
        <v-icon v-else icon="mdi-facebook" color="blue" size="28"/>
      </div><div class="page-info"><h3>{{page.name}}</h3><p>{{page.category}}</p><p v-if="page.phoneNumber" class="page-phone"><v-icon icon="mdi-phone-outline" size="14"/>{{page.phoneNumber}}</p><small>{{page.category.toLowerCase().includes('whatsapp')?'Phone number ID':'Page ID'}} · {{page.id}}</small><div class="token-row"><v-chip class="token-chip" :color="page.tokenReady?'success':'warning'" variant="tonal" size="x-small" :prepend-icon="page.tokenReady?'mdi-key-check':'mdi-key-alert'">{{page.tokenReady?'Token ພ້ອມໃຊ້':'Token ບໍ່ພ້ອມ'}}</v-chip><v-chip v-if="page.tokenReady" class="token-chip" color="info" variant="tonal" size="x-small" prepend-icon="mdi-clock-outline">{{tokenExpiry(page)}}</v-chip></div></div>
      <div class="page-action"><div v-if="page.connected" class="page-toggle"><span><strong>Webhook active</strong><small>Receiving events</small></span><v-switch :model-value="true" color="success" hide-details density="compact" :loading="toggling===page.id" :disabled="toggling!==null||!can('pages:connect')" :aria-label="`Pause Webhook for ${page.name}`" @update:model-value="value=>setPageEnabled(page,Boolean(value))"/></div><v-btn v-else color="primary" variant="flat" prepend-icon="mdi-play-circle-outline" :loading="toggling===page.id" :disabled="toggling!==null||!can('pages:connect')||!page.tokenReady" @click="setPageEnabled(page,true)">Enable Webhook</v-btn></div>
      <div class="relation-summary">
        <button @click="togglePageDetails(page.id)"><span><v-icon icon="mdi-robot-happy-outline"/>{{pageRelationCounts(page.id).automations}} Automation</span><span><v-icon icon="mdi-check-decagram-outline"/>{{pageRelationCounts(page.id).active}} ເປີດໃຊ້</span><span><v-icon icon="mdi-target"/>{{pageRelationCounts(page.id).targets}} ເປົ້າໝາຍ</span><v-icon :icon="expandedPages.includes(page.id)?'mdi-chevron-up':'mdi-chevron-down'"/></button>
        <div v-if="expandedPages.includes(page.id)" class="relation-details">
          <div v-if="pageAutomations(page.id).length" class="relation-list">
            <NuxtLink v-for="flow in pageAutomations(page.id)" :key="flow.id" to="/auto-replies">
              <span class="relation-state" :class="{active:flow.isActive}"/><div><strong>{{flow.name}}</strong><small>{{productsById.get(flow.productId)?.name||'ສິນຄ້າຖືກລຶບ'}} → {{replySetsById.get(flow.replySetId)?.name||'ຊຸດຂໍ້ຄວາມຖືກລຶບ'}}</small><small>{{flow.targets.length}} ເປົ້າໝາຍ · {{flow.firstMessageOnly?'ຕອບສະເພາະຂໍ້ຄວາມທຳອິດ':`ພັກ ${flow.cooldownSeconds} ວິນາທີ`}}</small></div><v-icon icon="mdi-arrow-right"/>
            </NuxtLink>
          </div>
          <div v-else class="relation-empty"><span>Page ນີ້ຍັງບໍ່ມີ Automation</span><v-btn size="small" variant="tonal" color="primary" to="/auto-replies">ສ້າງ Automation</v-btn></div>
        </div>
      </div>
    </article>
    <v-alert v-if="!pending&&data?.mode==='live'&&!pages.length" type="info" variant="tonal" icon="mdi-facebook">
      <strong>No Facebook or WhatsApp accounts found.</strong><br>Connect Facebook and allow the required permissions. Your Facebook account must have Page access.
    </v-alert>
    <v-alert v-else-if="!pending&&!visiblePages.length" type="info" variant="tonal">{{mainPlatformFilter==='whatsapp'?'No WhatsApp phone numbers were returned for this Meta account. Confirm the number belongs to an accessible Business Portfolio, then reconnect Facebook.':'No Pages match this filter.'}}</v-alert>
  </section>

  <v-dialog v-model="permissionsOpen" max-width="620">
    <v-card class="picker-card"><v-card-title class="picker-head"><div><small>FACEBOOK PERMISSIONS</small><h2>Choose features</h2></div><v-btn icon="mdi-close" variant="text" @click="permissionsOpen=false"/></v-card-title><v-card-text class="permission-body">
      <v-alert type="info" variant="tonal" density="compact">Page access, Webhooks and Messenger permissions are included automatically. Select only the additional features you need.</v-alert>
      <v-checkbox v-for="item in optionalPermissions" :key="item.value" v-model="selectedPermissions" :value="item.value" color="primary" hide-details><template #label><span class="permission-label"><strong>{{item.label}}</strong><small>{{item.description}}</small></span></template></v-checkbox>
      <p class="permission-note">Meta may require App Review and Advanced Access before these features work for public users.</p>
    </v-card-text><v-card-actions class="picker-actions"><v-btn variant="text" @click="selectedPermissions=[]">Required only</v-btn><v-spacer/><v-btn color="primary" prepend-icon="mdi-facebook" :loading="authorizing" @click="permissionsOpen=false;connectFacebook()">Continue with Facebook</v-btn></v-card-actions></v-card>
  </v-dialog>

    <v-dialog v-model="pickerOpen" max-width="1000" persistent scrollable>
    <v-card class="picker-card pc-wrapper">
      <div class="pc-header">
        <h3 class="pc-title">ເພີ່ມການເຊື່ອມຕໍ່</h3>
        <v-btn icon="mdi-close" variant="text" size="small" :disabled="activating" @click="pickerOpen=false"/>
      </div>
      
      <div class="pc-body">
        
        <div class="pc-sidebar">
          <div class="pc-nav-item active">
            <v-icon color="warning" size="20">mdi-clock-outline</v-icon>
            <span>ລໍຖ້າການເປີດໃຊ້ງານ</span>
          </div>
          <div class="pc-nav-item">
            <v-icon size="20" color="primary">mdi-facebook</v-icon>
            <span>Facebook</span>
          </div>
          <div class="pc-channel-note">WhatsApp ເຊື່ອມຕໍ່ຈາກປຸ່ມ “ເຊື່ອມຕໍ່ WhatsApp” ຢູ່ດ້ານເທິງ.</div>
        </div>
        
        <div class="pc-main">
          <div class="pc-main-head">
            <div class="pc-main-title">ເລືອກເພຈເພື່ອເປີດໃຊ້ງານ</div>
            <div class="pc-search-box">
              <v-icon size="18" color="grey">mdi-magnify</v-icon>
              <input type="text" v-model="search" placeholder="ຄົ້ນຫາໜ້າ" />
            </div>
          </div>
          
          <div class="pc-tabs-container">
            <div class="pc-tabs">
              <div class="pc-tab" :class="{active: activeView==='pending'}" @click="activeView='pending'">
                ລໍຖ້າການເປີດໃຊ້ງານ <span class="pc-tab-badge" v-if="availablePages.length">{{availablePages.length}}</span>
              </div>
            </div>
          </div>
          
          <div class="pc-content">
            <div v-if="availablePages.length" class="pc-grid">
              <div v-for="page in availablePages" :key="page.id" class="pc-card" :class="{'selected':selected.includes(page.id)}" @click="selected.includes(page.id)?selected=selected.filter(id=>id!==page.id):selected.push(page.id)">
                <div class="pc-card-avatar">
                  <v-img v-if="pictureAvailable(page)" :src="pagePictureUrl(page)" :alt="`${page.name} profile picture`" cover @error="handlePictureError(page.id)"></v-img>
                  <v-icon v-else-if="page.category.toLowerCase().includes('whatsapp')" icon="mdi-whatsapp" color="success" size="32"></v-icon>
                  <v-icon v-else icon="mdi-facebook" color="blue" size="32"></v-icon>
                </div>
                <div class="pc-card-info">
                  <div class="pc-card-name">{{page.name}}</div>
                  <div class="pc-card-sub">
                    <v-icon :icon="page.category.toLowerCase().includes('whatsapp')?'mdi-whatsapp':'mdi-facebook'" :color="page.category.toLowerCase().includes('whatsapp')?'success':'blue'" size="14"></v-icon>
                    <span>{{page.phoneNumber||page.id}}</span>
                  </div>
                </div>
                <v-checkbox-btn :model-value="selected.includes(page.id)" color="primary" class="pc-card-check" @click.stop/>
              </div>
            </div>
            <div v-else class="pc-empty">
              <v-alert type="info" variant="tonal" class="w-100">ບໍ່ມີເພຈທີ່ລໍຖ້າການເປີດໃຊ້ງານ (All available Pages are active).</v-alert>
            </div>
          </div>

          <div class="pc-footer">
            <div class="pc-footer-left" @click="selected.length === availablePages.length ? selected=[] : selected=availablePages.map(p=>p.id)">
              <v-icon size="18" class="mr-2">mdi-checkbox-multiple-marked-outline</v-icon>
              <span>ເລືອກທັງໝົດ / ບໍ່ເລືອກທັງໝົດ</span>
            </div>
            <v-btn color="primary" prepend-icon="mdi-lightning-bolt" size="large" :disabled="!selected.length" :loading="activating" @click="activateSelected" class="text-none px-6" style="border-radius: 8px;">ເປີດໃຊ້ງານ</v-btn>
          </div>
        </div>
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.intro{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;margin-bottom:16px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.intro-actions{display:flex;align-items:center;gap:10px;flex-wrap:wrap;justify-content:flex-end}.intro p{margin:0 0 8px;color:var(--color-primary);font-size:var(--text-xs);font-weight:800;letter-spacing:.13em}.intro h2{margin:0 0 8px;font-size:var(--text-xl)}.intro span{color:var(--color-text-secondary);font-size:var(--text-md)}.connection-summary{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-bottom:16px}.connection-summary button{min-height:76px;display:flex;align-items:center;gap:13px;padding:14px 17px;border:1px solid var(--color-border);border-radius:var(--radius-lg);color:var(--color-text-secondary);background:var(--color-surface);text-align:left;cursor:pointer;transition:var(--transition-fast)}.connection-summary button:hover,.connection-summary button.active{color:var(--color-primary);border-color:color-mix(in srgb,var(--color-primary) 35%,var(--color-border));background:var(--color-primary-soft)}.connection-summary button>i{padding:10px;border-radius:11px;background:var(--color-background)}.connection-summary span{display:grid}.connection-summary small{font-size:9px;font-weight:800;letter-spacing:.08em}.connection-summary strong{color:var(--color-text);font-size:20px}.page-grid{display:grid;gap:12px}.page-card{display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:16px;padding:20px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-avatar{width:52px;height:52px;display:grid;place-items:center;overflow:hidden;color:var(--color-surface);background:var(--color-primary);border-radius:16px}.page-info h3{margin:0;font-size:15px}.page-info p{margin:3px 0;color:var(--color-text-secondary);font-size:12px}.page-info .page-phone{display:flex;align-items:center;gap:5px;color:var(--color-success);font-weight:700}.page-info small{color:var(--color-text-muted);font-size:10px}.token-row{display:flex;gap:6px;flex-wrap:wrap}.token-chip{display:flex!important;width:max-content;margin-top:7px}.page-toggle{min-width:196px;display:flex;align-items:center;justify-content:flex-end;gap:12px;padding:8px 10px 8px 14px;color:var(--color-success);background:var(--color-success-soft);border-radius:13px}.page-toggle>span{display:grid}.page-toggle strong{font-size:11px}.page-toggle small{color:var(--color-text-secondary);font-size:9px}.relation-summary{grid-column:1/-1;border-top:1px solid var(--color-border-subtle)}.relation-summary>button{width:100%;display:flex;align-items:center;gap:18px;padding:14px 0 0;border:0;color:var(--color-text-secondary);background:none;cursor:pointer}.relation-summary>button span{display:flex;align-items:center;gap:6px;font-size:11px;font-weight:700}.relation-summary>button>i:last-child{margin-left:auto}.relation-details{padding-top:13px}.relation-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}.relation-list a{display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:10px;padding:12px;color:var(--color-text);background:var(--color-background);border:1px solid var(--color-border);border-radius:12px;text-decoration:none}.relation-list a>div{display:grid;gap:3px;min-width:0}.relation-list strong,.relation-list small{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.relation-list strong{font-size:12px}.relation-list small{color:var(--color-text-secondary);font-size:10px}.relation-state{width:8px;height:8px;background:var(--color-text-muted);border-radius:50%}.relation-state.active{background:var(--color-success)}.relation-empty{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px;color:var(--color-text-secondary);background:var(--color-background);border-radius:12px;font-size:12px}.permission-body{display:grid;gap:8px;padding:22px 24px}.permission-label{display:grid;margin-left:6px}.permission-label small,.permission-note{color:var(--color-text-muted);font-size:11px}.permission-note{margin:8px 0 0}
.whatsapp-setup{display:grid;grid-template-columns:auto 1fr;gap:18px;margin-bottom:16px;padding:24px;background:linear-gradient(135deg,color-mix(in srgb,var(--color-success) 8%,var(--color-surface)),var(--color-surface) 62%);border:1px solid color-mix(in srgb,var(--color-success) 24%,var(--color-border));border-radius:var(--radius-lg)}.whatsapp-setup-icon{width:58px;height:58px;display:grid;place-items:center;color:var(--color-success);background:var(--color-surface);border:1px solid color-mix(in srgb,var(--color-success) 22%,var(--color-border));border-radius:18px}.whatsapp-setup-copy>small{color:var(--color-success);font-size:10px;font-weight:800;letter-spacing:.13em}.whatsapp-setup-copy h3{margin:4px 0 5px;font-size:19px}.whatsapp-setup-copy>p{max-width:720px;margin:0;color:var(--color-text-secondary);font-size:13px}.setup-steps{display:flex;gap:10px;flex-wrap:wrap;margin:18px 0;padding:0;list-style:none}.setup-steps li{display:flex;align-items:center;gap:7px;padding:8px 11px;color:var(--color-text-secondary);font-size:11px;font-weight:700;background:var(--color-surface);border:1px solid var(--color-border);border-radius:999px}.setup-steps span{width:20px;height:20px;display:grid;place-items:center;color:white;background:var(--color-success);border-radius:50%;font-size:10px}.setup-actions{display:flex;align-items:center;gap:5px;flex-wrap:wrap}.signup-wait-hint{max-width:780px;margin-top:12px}.signup-wait-hint code{overflow-wrap:anywhere}.diagnostic-details{margin-top:14px;padding:14px;color:var(--color-text-secondary);background:var(--color-background);border:1px solid var(--color-border);border-radius:12px}.diagnostic-counts{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:9px}.diagnostic-counts span{padding:6px 9px;background:var(--color-surface);border-radius:8px;font-size:10px}.diagnostic-details p{margin:6px 0;font-size:11px}.diagnostic-details ul{max-height:130px;margin:8px 0 0;padding-left:18px;overflow:auto}.diagnostic-details li{margin:4px 0;font-size:10px;word-break:break-word}
@media(max-width:640px){.intro{display:grid;padding:20px}.intro-actions{display:grid;grid-template-columns:1fr;width:100%}.intro-actions :deep(.v-chip){width:max-content}.connection-summary{grid-template-columns:repeat(2,1fr);gap:7px}.connection-summary button{min-height:68px;justify-content:center;padding:9px 5px}.connection-summary button>i{display:none}.connection-summary small{font-size:7px}.connection-summary strong{font-size:18px}.whatsapp-setup{grid-template-columns:1fr;padding:19px}.whatsapp-setup-icon{width:48px;height:48px}.setup-steps{display:grid}.setup-actions{display:grid}.setup-actions :deep(.v-btn){width:100%}.page-card{grid-template-columns:auto 1fr;padding:17px}.page-action{grid-column:1/-1}.page-action :deep(.v-btn),.page-toggle{width:100%}.page-toggle{justify-content:space-between}.token-row{gap:3px}.token-chip{font-size:8px!important}.relation-summary>button{gap:9px;flex-wrap:wrap}.relation-list{grid-template-columns:1fr}.relation-empty{align-items:flex-start;flex-direction:column}}
.picker-card{border-radius:var(--radius-lg)!important}.picker-head{display:flex;align-items:center;justify-content:space-between;padding:22px 24px;border-bottom:1px solid var(--color-border)}.picker-head small{color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.14em}.picker-head h2{margin:2px 0 0;font-size:20px}.picker-body{padding:22px 24px!important}.picker-toolbar{display:grid;grid-template-columns:1fr minmax(240px,320px);align-items:center;gap:20px;margin-bottom:18px}.picker-toolbar div{display:grid;gap:3px}.picker-toolbar span{color:var(--color-text-muted);font-size:12px}.picker-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.picker-page{display:grid;grid-template-columns:auto auto 1fr;align-items:center;gap:12px;padding:14px;border:1px solid var(--color-border);border-radius:14px;cursor:pointer;transition:.18s ease}.picker-page:hover,.picker-page.is-selected{border-color:var(--color-primary);background:var(--color-primary-soft)}.picker-page>span{display:grid;min-width:0}.picker-page strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.picker-page small{overflow:hidden;color:var(--color-text-muted);font-size:11px;text-overflow:ellipsis;white-space:nowrap}.picker-actions{padding:16px 24px;border-top:1px solid var(--color-border)}
@media(max-width:700px){.picker-toolbar,.picker-grid{grid-template-columns:1fr}.picker-head,.picker-actions{padding-inline:16px}}

.pc-wrapper { display:flex; flex-direction:column; background:#ffffff!important; border-radius:12px!important; overflow:hidden; }
.pc-header { display:flex; align-items:center; justify-content:space-between; padding:16px 20px; border-bottom:1px solid #ebedf0; }
.pc-title { margin:0; font-size:16px; font-weight:600; color:#1a1a1a; }
.pc-body { display:flex; min-height:500px; max-height:75vh; }
.pc-sidebar { display:none; }
.pc-nav-item { display:flex; align-items:center; gap:12px; padding:10px 14px; border-radius:8px; cursor:pointer; font-size:14px; color:#4a4a4a; font-weight:500; transition:background 0.2s; margin-bottom:4px; }
.pc-nav-item:hover { background:#eff1f4; }
.pc-nav-item.active { background:#ffffff; color:#1a1a1a; box-shadow:0 1px 3px rgba(0,0,0,0.05); }
.pc-badge-beta { font-size:10px; background:#ffecb3; color:#f57c00; padding:2px 6px; border-radius:4px; margin-left:auto; font-weight:bold; }
.pc-main { flex:1; display:flex; flex-direction:column; background:#ffffff; }
.pc-main-head { display:flex; align-items:center; justify-content:space-between; padding:16px 24px 8px; }
.pc-main-title { font-size:14px; font-weight:600; color:#1a1a1a; }
.pc-search-box { display:flex; align-items:center; background:#f2f3f5; border-radius:8px; padding:6px 12px; gap:8px; width:220px; }
.pc-search-box input { border:none; background:transparent; outline:none; font-size:13px; flex:1; width:100%; }
.pc-tabs-container { display:none; }
.pc-tabs { display:flex; gap:24px; }
.pc-tab { padding:12px 0; font-size:14px; font-weight:500; color:#77798b; cursor:pointer; position:relative; display:flex; align-items:center; gap:6px; }
.pc-tab.active { color:#1877f2; }
.pc-tab.active::after { content:''; position:absolute; bottom:-1px; left:0; right:0; height:2px; background:#1877f2; border-radius:2px 2px 0 0; }
.pc-tab-badge { background:#ff4d4f; color:white; font-size:11px; padding:0 6px; border-radius:10px; font-weight:bold; line-height:16px; }
.pc-conn-code { display:flex; align-items:center; gap:4px; }
.pc-content { flex:1; padding:20px 24px; overflow-y:auto; background:#fbfcfd; }
.pc-grid { display:grid; grid-template-columns:repeat(2, 1fr); gap:16px; }
.pc-card { display:flex; align-items:center; padding:14px; background:#ffffff; border:1px solid #ebedf0; border-radius:8px; cursor:pointer; transition:all 0.2s; position:relative; }
.pc-card:hover { border-color:#d0d5dc; box-shadow:0 2px 8px rgba(0,0,0,0.04); }
.pc-card.selected { border-color:#1877f2; }
.pc-card-avatar { width:44px; height:44px; border-radius:50%; overflow:hidden; background:#f0f2f5; display:flex; align-items:center; justify-content:center; margin-right:12px; flex-shrink:0; border:1px solid #0000001a; }
.pc-card-info { flex:1; min-width:0; }
.pc-card-name { font-size:14px; font-weight:600; color:#1a1a1a; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; margin-bottom:2px; }
.pc-card-sub { display:flex; align-items:center; gap:4px; font-size:12px; color:#65676b; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.pc-card-check { position:absolute; right:12px; pointer-events:none; }
.pc-empty { display:flex; align-items:center; justify-content:center; height:100%; }
.pc-footer { padding:16px 24px; display:flex; align-items:center; justify-content:space-between; border-top:1px solid #ebedf0; background:#ffffff; }
.pc-footer-left { display:flex; align-items:center; font-size:13px; color:#65676b; cursor:pointer; user-select:none; }
.pc-hotkey { display:flex; align-items:center; gap:2px; background:#f2f3f5; border:1px solid #d0d5dc; padding:2px 6px; border-radius:4px; font-size:11px; margin-left:8px; font-weight:600; color:#1a1a1a; }
.pc-channel-note { margin:12px; padding:12px; color:var(--color-text-secondary); background:var(--color-surface); border:1px solid var(--color-border); border-radius:10px; font-size:12px; line-height:1.55; }

</style>
