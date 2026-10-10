<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const { can } = useAuth()
const { l } = useLocale()
if (!can('customers:read')) throw createError({ statusCode: 403 })
type Lead={id:string;pageId:string;senderId:string;sourceType:string;sourceId:string;status:string;assignedUserId:string;alertRead:boolean;createdAt:string}
type Step={DelayMinutes:number;Message:string;IsEnabled:boolean}
const {data,pending,error,refresh}=await useApi<{items:Lead[]}>('/proxy/api/v1/leads')
const {data:config,refresh:refreshConfig}=await useApi<any>('/proxy/api/v1/leads/settings')
const leads=computed(()=>data.value?.items||[])
const settings=reactive({assignmentMode:'round_robin',alertsEnabled:true,followUpEnabled:false,capiEnabled:false})
const steps=ref<Step[]>([]);const saving=ref(false);const notice=ref('')
watchEffect(()=>{if(config.value?.settings){const s=config.value.settings;settings.assignmentMode=s.assignmentMode||'round_robin';settings.alertsEnabled=s.alertsEnabled;settings.followUpEnabled=s.followUpEnabled;settings.capiEnabled=s.capiEnabled;steps.value=(config.value.steps||[]).map((x:any)=>({DelayMinutes:x.delayMinutes,Message:x.message,IsEnabled:x.isEnabled}))}})
function addStep(){steps.value.push({DelayMinutes:60,Message:'',IsEnabled:true})}
async function save(){saving.value=true;notice.value='';try{await $fetch('/api/proxy/api/v1/leads/settings',{method:'PUT',body:{settings,steps:steps.value}});notice.value=l({lo:'ບັນທຶກການຕັ້ງຄ່າແລ້ວ',th:'บันทึกการตั้งค่าแล้ว',en:'Settings saved.'});await refreshConfig()}finally{saving.value=false}}
async function read(lead:Lead){if(lead.alertRead)return;await $fetch(`/api/proxy/api/v1/leads/${lead.id}/read`,{method:'PATCH'});await refresh()}

let evtSource: EventSource | null = null
onMounted(() => {
  evtSource = new EventSource('/api/proxy/api/v1/chat/stream')
  evtSource.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data)
      if (data.type === 'lead_alert') {
        notice.value = l({lo:'🎉 ມີ Lead ໃໝ່ເຂົ້າມາ!',th:'🎉 มี Lead ใหม่เข้ามา!',en:'🎉 New lead assigned!'})
        refresh()
      }
    } catch {}
  }
})
onUnmounted(() => { if (evtSource) evtSource.close() })
</script>
<template>
<section class="hero"><div><small>LEAD AUTOMATION</small><h2>{{l({lo:'ຈັດການ Lead',th:'จัดการ Lead',en:'Lead Management'})}}</h2><p>{{l({lo:'ຈັດຄົນຮັບ Lead, ແຈ້ງເຕືອນ, ຕິດຕາມ ແລະສົ່ງ Conversion ໃນບ່ອນດຽວ',th:'มอบหมาย Lead แจ้งเตือน ติดตาม และส่ง Conversion ในที่เดียว',en:'Assign, alert, follow up and report conversions in one place.'})}}</p></div><v-btn prepend-icon="mdi-refresh" variant="outlined" @click="refresh()">{{l({lo:'ໂຫຼດໃໝ່',th:'รีเฟรช',en:'Refresh'})}}</v-btn></section>
<v-alert v-if="notice" type="success" variant="tonal" closable class="mb-4">{{notice}}</v-alert>
<section class="grid"><article class="panel"><header><h3>{{l({lo:'Lead ລ່າສຸດ',th:'Lead ล่าสุด',en:'Recent leads'})}}</h3><v-chip color="primary">{{leads.length}}</v-chip></header><v-progress-linear v-if="pending" indeterminate/><v-alert v-else-if="error" type="error" variant="tonal">{{l({lo:'ໂຫຼດ Lead ບໍ່ສຳເລັດ',th:'โหลด Lead ไม่สำเร็จ',en:'Unable to load leads.'})}}</v-alert><div v-else-if="leads.length" class="lead-list"><button v-for="lead in leads" :key="lead.ID" :class="{unread:!lead.AlertRead}" @click="read(lead)"><v-icon icon="mdi-account-outline"/><span><strong>{{lead.SenderID}}</strong><small>{{lead.SourceType}} · {{lead.SourceID||lead.PageID}}</small></span><time>{{new Date(lead.CreatedAt).toLocaleString()}}</time></button></div><div v-else class="empty">{{l({lo:'ຍັງບໍ່ມີ Lead; ເມື່ອມີຂໍ້ຄວາມ ຫຼື comment ຈະສະແດງຢູ່ນີ້',th:'ยังไม่มี Lead เมื่อมีข้อความหรือคอมเมนต์จะแสดงที่นี่',en:'No leads yet. Messages and comments will appear here.'})}}</div></article>
<article class="panel settings"><header><h3>{{l({lo:'ຕັ້ງຄ່າ Automation',th:'ตั้งค่า Automation',en:'Automation settings'})}}</h3></header><v-select v-model="settings.assignmentMode" :items="[{title:l({lo:'ແບ່ງໃຫ້ທີມຕາມລຳດັບ',th:'แบ่งให้ทีมตามลำดับ',en:'Round-robin assignment'}),value:'round_robin'},{title:l({lo:'ບໍ່ຈັດຄົນອັດຕະໂນມັດ',th:'ไม่มอบหมายอัตโนมัติ',en:'No automatic assignment'}),value:'none'}]" item-title="title" item-value="value" :label="l({lo:'ການຈັດ Lead',th:'การมอบหมาย Lead',en:'Lead assignment'})"/><v-switch v-model="settings.alertsEnabled" color="primary" :label="l({lo:'ແຈ້ງເຕືອນ Lead ໃໝ່ທັນທີ',th:'แจ้งเตือน Lead ใหม่ทันที',en:'Instant new-lead alerts'})"/><v-switch v-model="settings.followUpEnabled" color="primary" :label="l({lo:'ເປີດ Follow-up sequence',th:'เปิด Follow-up sequence',en:'Enable follow-up sequence'})"/><v-switch v-model="settings.capiEnabled" color="primary" :label="l({lo:'ສົ່ງ Lead ໄປ Meta Conversions API',th:'ส่ง Lead ไป Meta Conversions API',en:'Send leads to Meta Conversions API'})"/><div v-if="settings.followUpEnabled" class="steps"><div v-for="(step,i) in steps" :key="i" class="step"><v-text-field v-model.number="step.DelayMinutes" type="number" min="0" :label="l({lo:'ລໍຖ້າ (ນາທີ)',th:'รอ (นาที)',en:'Wait (minutes)'})"/><v-textarea v-model="step.Message" rows="2" :label="l({lo:'ຂໍ້ຄວາມ',th:'ข้อความ',en:'Message'})"/><v-btn icon="mdi-delete-outline" color="error" variant="text" @click="steps.splice(i,1)"/></div><v-btn prepend-icon="mdi-plus" variant="tonal" @click="addStep">{{l({lo:'ເພີ່ມຂັ້ນຕອນ',th:'เพิ่มขั้นตอน',en:'Add step'})}}</v-btn></div><v-btn block color="primary" :loading="saving" @click="save">{{l({lo:'ບັນທຶກ',th:'บันทึก',en:'Save settings'})}}</v-btn></article></section>
</template>
<style scoped>.hero{display:flex;justify-content:space-between;gap:20px;padding:24px;margin-bottom:16px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.hero small{color:var(--color-primary);font-weight:800;letter-spacing:.12em}.hero h2{margin:6px 0}.hero p{margin:0;color:var(--color-text-secondary)}.grid{display:grid;grid-template-columns:minmax(0,1.2fr) minmax(340px,.8fr);gap:16px}.panel{padding:20px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.panel header{display:flex;align-items:center;justify-content:space-between;margin-bottom:16px}.panel h3{margin:0}.lead-list{display:grid;gap:8px}.lead-list button{display:grid;grid-template-columns:auto 1fr auto;gap:12px;align-items:center;width:100%;padding:13px;border:1px solid var(--color-border);border-radius:12px;background:transparent;text-align:left}.lead-list button.unread{border-color:var(--color-primary);background:var(--color-primary-soft)}.lead-list span{display:grid}.lead-list small,time{color:var(--color-text-secondary);font-size:11px}.settings{display:grid;gap:8px}.steps{display:grid;gap:8px}.step{display:grid;grid-template-columns:130px 1fr auto;gap:8px}.empty{padding:50px 15px;text-align:center;color:var(--color-text-secondary)}@media(max-width:900px){.grid{grid-template-columns:1fr}}@media(max-width:640px){.hero{display:grid}.step{grid-template-columns:1fr}.lead-list button{grid-template-columns:auto 1fr}.lead-list time{grid-column:2}}</style>
