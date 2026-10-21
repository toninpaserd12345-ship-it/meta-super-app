<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

type NotificationType = 'new_lead' | 'order_updates' | 'ad_performance' | 'connection_health' | 'weekly_summary'
type NotificationFrequency = 'instant' | 'hourly' | 'daily' | 'weekly'
type NotificationPreference = { type: NotificationType; frequency: NotificationFrequency; enabled: boolean }
type NotificationResponse = { items: NotificationPreference[] }

const { l } = useLocale()
const { can } = useAuth()
const { data, pending, error, refresh } = useApi<NotificationResponse>('/proxy/api/v1/settings/notifications')

const preferences = ref<NotificationPreference[]>([])
const savedPreferences = ref<NotificationPreference[]>([])
const savingType = ref<NotificationType | null>(null)
const savedType = ref<NotificationType | null>(null)
const saveError = ref('')

watch(data, value => {
 if (!value?.items) return
 preferences.value = value.items.map(item => ({ ...item }))
 savedPreferences.value = value.items.map(item => ({ ...item }))
}, { immediate: true })

const notificationDefinitions = computed(() => [
 {
 type: 'new_lead' as const,
 icon: 'mdi-account-plus-outline',
 title: l({ lo: 'Lead ໃໝ່', th: 'ลีดใหม่', en: 'New leads' }),
 description: l({ lo: 'ແຈ້ງເມື່ອມີລູກຄ້າໃໝ່ຈາກ Facebook ຫຼື WhatsApp.', th: 'แจ้งเมื่อมีลูกค้าใหม่จาก Facebook หรือ WhatsApp', en: 'When a new customer arrives from Facebook or WhatsApp.' }),
 options: ['instant', 'hourly', 'daily'] as NotificationFrequency[],
 },
 {
 type: 'order_updates' as const,
 icon: 'mdi-package-variant-closed-check',
 title: l({ lo: 'ການປ່ຽນແປງອໍເດີ', th: 'การเปลี่ยนแปลงออเดอร์', en: 'Order updates' }),
 description: l({ lo: 'ການຊຳລະ, ການຈັດສົ່ງ ແລະສະຖານະອໍເດີທີ່ສຳຄັນ.', th: 'การชำระเงิน การจัดส่ง และสถานะออเดอร์ที่สำคัญ', en: 'Important payment, fulfillment, and order status changes.' }),
 options: ['instant', 'daily'] as NotificationFrequency[],
 },
 {
 type: 'ad_performance' as const,
 icon: 'mdi-chart-line',
 title: l({ lo: 'ຜົນງານໂຄສະນາ', th: 'ประสิทธิภาพโฆษณา', en: 'Ad performance' }),
 description: l({ lo: 'ແຈ້ງເມື່ອຄ່າໃຊ້ຈ່າຍ, CPA ຫຼືຜົນງານປ່ຽນແປງສຳຄັນ.', th: 'แจ้งเมื่อค่าใช้จ่าย CPA หรือประสิทธิภาพเปลี่ยนแปลงอย่างสำคัญ', en: 'Significant changes in spend, CPA, or delivery performance.' }),
 options: ['instant', 'daily', 'weekly'] as NotificationFrequency[],
 },
 {
 type: 'connection_health' as const,
 icon: 'mdi-access-point-network',
 title: l({ lo: 'ສະຖານະການເຊື່ອມຕໍ່', th: 'สถานะการเชื่อมต่อ', en: 'Connection health' }),
 description: l({ lo: 'Token, Webhook, Facebook Page ຫຼື WhatsApp ຕ້ອງການກວດສອບ.', th: 'โทเค็น Webhook เพจ Facebook หรือ WhatsApp ต้องได้รับการตรวจสอบ', en: 'Token, Webhook, Facebook Page, or WhatsApp connection needs attention.' }),
 options: ['instant', 'daily'] as NotificationFrequency[],
 },
 {
 type: 'weekly_summary' as const,
 icon: 'mdi-calendar-text-outline',
 title: l({ lo: 'ສະຫຼຸບປະຈຳອາທິດ', th: 'สรุปรายสัปดาห์', en: 'Weekly summary' }),
 description: l({ lo: 'ສະຫຼຸບ Lead, ອໍເດີ, ຍອດຂາຍ ແລະຜົນງານໂຄສະນາ.', th: 'สรุปลีด ออเดอร์ ยอดขาย และประสิทธิภาพโฆษณา', en: 'A weekly roll-up of leads, orders, sales, and ad performance.' }),
 options: ['weekly'] as NotificationFrequency[],
 },
])

function preferenceFor(type: NotificationType) {
 return preferences.value.find(item => item.type === type)
}

function frequencyLabel(value: NotificationFrequency) {
 const labels = {
 instant: l({ lo: 'ທັນທີ', th: 'ทันที', en: 'Instantly' }),
 hourly: l({ lo: 'ສະຫຼຸບທຸກຊົ່ວໂມງ', th: 'สรุปทุกชั่วโมง', en: 'Hourly digest' }),
 daily: l({ lo: 'ສະຫຼຸບທຸກມື້', th: 'สรุปรายวัน', en: 'Daily digest' }),
 weekly: l({ lo: 'ສະຫຼຸບທຸກອາທິດ', th: 'สรุปรายสัปดาห์', en: 'Weekly digest' }),
 }
 return labels[value]
}

async function updatePreference(type: NotificationType, patch: Partial<NotificationPreference>) {
 if (!can('settings:update') || savingType.value) return
 const target = preferenceFor(type)
 if (!target) return
 Object.assign(target, patch)
 savingType.value = type
 savedType.value = null
 saveError.value = ''
 try {
 const result = await $fetch<NotificationResponse>('/api/proxy/api/v1/settings/notifications', {
 method: 'PUT',
 body: { items: preferences.value },
 })
 preferences.value = result.items.map(item => ({ ...item }))
 savedPreferences.value = result.items.map(item => ({ ...item }))
 savedType.value = type
 window.setTimeout(() => { if (savedType.value === type) savedType.value = null }, 1800)
 } catch {
 preferences.value = savedPreferences.value.map(item => ({ ...item }))
 saveError.value = l({ lo: 'ບັນທຶກບໍ່ສຳເລັດ. ກະລຸນາລອງໃໝ່.', th: 'บันทึกไม่สำเร็จ โปรดลองอีกครั้ง', en: 'Changes could not be saved. Please try again.' })
 } finally {
 savingType.value = null
 }
}
</script>

<template>
 <section class="notification-settings" aria-labelledby="notification-title">
 <header class="section-header">
 <div>
 <h2 id="notification-title">{{ l({ lo: 'ການແຈ້ງເຕືອນ', th: 'การแจ้งเตือน', en: 'Notification settings' }) }}</h2>
 <p>{{ l({ lo: 'ເລືອກສິ່ງທີ່ຕ້ອງການຮັບ ແລະຄວາມຖີ່ຂອງແຕ່ລະປະເພດ.', th: 'เลือกสิ่งที่ต้องการรับและความถี่ของการแจ้งเตือนแต่ละประเภท', en: 'Choose what you receive and set a separate delivery frequency for every notification type.' }) }}</p>
 </div>
 <div class="autosave-note"><v-icon icon="mdi-content-save-check-outline" size="18" />{{ l({ lo: 'ບັນທຶກອັດຕະໂນມັດ', th: 'บันทึกอัตโนมัติ', en: 'Saved automatically' }) }}</div>
 </header>

 <v-alert v-if="error" type="error" variant="tonal" class="mb-5">
 {{ l({ lo: 'ໂຫຼດການຕັ້ງຄ່າບໍ່ສຳເລັດ.', th: 'โหลดการตั้งค่าไม่สำเร็จ', en: 'Notification settings could not be loaded.' }) }}
 <button class="inline-action" type="button" @click="refresh()">{{ l({ lo: 'ລອງໃໝ່', th: 'ลองอีกครั้ง', en: 'Try again' }) }}</button>
 </v-alert>
 <v-alert v-if="saveError" type="error" variant="tonal" closable class="mb-5" @click:close="saveError = ''">{{ saveError }}</v-alert>

 <div v-if="pending" class="notification-list" aria-busy="true">
 <v-skeleton-loader v-for="index in 5" :key="index" type="list-item-avatar-three-line" />
 </div>

 <div v-else class="notification-list">
 <article v-for="definition in notificationDefinitions" :key="definition.type" class="notification-row" :class="{ muted: !preferenceFor(definition.type)?.enabled }">
 <div class="notification-icon" aria-hidden="true"><v-icon :icon="definition.icon" size="22" /></div>
 <div class="notification-copy">
 <h3>{{ definition.title }}</h3>
 <p>{{ definition.description }}</p>
 </div>

 <div class="notification-state">
 <span class="control-label">{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</span>
 <v-switch
 :model-value="preferenceFor(definition.type)?.enabled ?? false"
 color="primary"
 hide-details
 density="compact"
 :disabled="!can('settings:update') || savingType !== null"
 :aria-label="`${definition.title}: ${l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'status' })}`"
 @update:model-value="updatePreference(definition.type, { enabled: Boolean($event) })"
 />
 <strong>{{ preferenceFor(definition.type)?.enabled ? l({ lo: 'ເປີດ', th: 'เปิด', en: 'On' }) : l({ lo: 'ປິດ', th: 'ปิด', en: 'Off' }) }}</strong>
 </div>

 <label class="frequency-control">
 <span class="control-label">{{ l({ lo: 'ຄວາມຖີ່ການສົ່ງ', th: 'ความถี่ในการส่ง', en: 'Delivery frequency' }) }}</span>
 <span class="select-wrap">
 <select
 :value="preferenceFor(definition.type)?.frequency"
 :disabled="!preferenceFor(definition.type)?.enabled || !can('settings:update') || savingType !== null"
 @change="updatePreference(definition.type, { frequency: ($event.target as HTMLSelectElement).value as NotificationFrequency })"
 >
 <option v-for="option in definition.options" :key="option" :value="option">{{ frequencyLabel(option) }}</option>
 </select>
 <v-icon icon="mdi-chevron-down" size="18" aria-hidden="true" />
 </span>
 </label>

 <div class="save-state" role="status" aria-live="polite">
 <template v-if="savingType === definition.type"><v-progress-circular indeterminate size="16" width="2" /> {{ l({ lo: 'ກຳລັງບັນທຶກ', th: 'กำลังบันทึก', en: 'Saving' }) }}</template>
 <template v-else-if="savedType === definition.type"><v-icon icon="mdi-check" size="16" /> {{ l({ lo: 'ບັນທຶກແລ້ວ', th: 'บันทึกแล้ว', en: 'Saved' }) }}</template>
 </div>
 </article>
 </div>

 <p v-if="!can('settings:update')" class="permission-note"><v-icon icon="mdi-lock-outline" size="18" />{{ l({ lo: 'ບົດບາດຂອງທ່ານເບິ່ງໄດ້ ແຕ່ປ່ຽນການຕັ້ງຄ່າບໍ່ໄດ້.', th: 'บทบาทของคุณดูได้ แต่ไม่สามารถเปลี่ยนการตั้งค่าได้', en: 'Your role can view these preferences but cannot change them.' }) }}</p>
 </section>
</template>

<style scoped>
.notification-settings { display: grid; gap: 20px; }
.section-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; }
.section-header h2 { margin: 0 0 6px; color: var(--color-text); font-size: 22px; line-height: 1.25; }
.section-header p { max-width: 68ch; margin: 0; color: var(--color-text-secondary); font-size: 14px; line-height: 1.65; }
.autosave-note { display: inline-flex; align-items: center; flex: 0 0 auto; gap: 7px; min-height: 36px; padding: 6px 10px; border-radius: 9px; background: var(--color-surface-soft); color: var(--color-text-secondary); font-size: 12px; font-weight: 650; }
.notification-list { overflow: hidden; border: 1px solid var(--color-border); border-radius: 12px; background: var(--color-surface); }
.notification-row { display: grid; grid-template-columns: 24px minmax(220px, 1fr) 126px minmax(190px, 240px); grid-template-areas: "icon copy state frequency" ". . save save"; align-items: center; column-gap: 14px; row-gap: 4px; min-height: 102px; padding: 17px 18px; transition: background-color var(--transition-fast); }
.notification-row + .notification-row { border-top: 1px solid var(--color-border); }
.notification-row:hover { background: var(--color-background); }
.notification-icon { grid-area: icon; display: grid; place-items:center; width:24px; height:24px; color:var(--color-text-secondary); }
.notification-copy { grid-area: copy; min-width: 0; }
.notification-copy h3 { margin: 0 0 5px; color: var(--color-text); font-size: 15px; line-height: 1.35; }
.notification-copy p { max-width: 62ch; margin: 0; color: var(--color-text-secondary); font-size: 13px; line-height: 1.55; }
.control-label { color: var(--color-text-muted); font-size: 12px; font-weight: 650; }
.notification-state { grid-area: state; display: grid; grid-template-columns: auto auto; grid-template-areas: "label label" "switch value"; align-items: center; justify-content: start; column-gap: 4px; }
.notification-state .control-label { grid-area: label; margin-bottom: 3px; }
.notification-state :deep(.v-switch) { grid-area: switch; }
.notification-state strong { grid-area: value; color: var(--color-text); font-size: 13px; }
.frequency-control { grid-area: frequency; display: grid; gap: 7px; min-width: 0; }
.select-wrap { position: relative; display: block; }
.select-wrap select { width: 100%; min-height: 44px; appearance: none; border: 1px solid var(--color-border-strong); border-radius: 10px; padding: 0 40px 0 12px; background: var(--color-surface); color: var(--color-text); font: inherit; font-size: 13px; font-weight: 650; cursor: pointer; }
.select-wrap select:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-primary) 24%, transparent); outline-offset: 2px; border-color: var(--color-primary); }
.select-wrap select:disabled { cursor: not-allowed; background: var(--color-surface-soft); color: var(--color-text-muted); }
.select-wrap .v-icon { position: absolute; top: 50%; right: 12px; translate: 0 -50%; pointer-events: none; color: var(--color-text-muted); }
.save-state { grid-area: save; display: flex; justify-content: flex-end; align-items: center; gap: 6px; min-height: 20px; color: var(--color-success); font-size: 12px; }
.notification-row.muted .notification-icon { color: var(--color-text-muted); }
.permission-note { display: flex; align-items: flex-start; gap: 8px; margin: 0; padding: 12px 14px; border-radius: 10px; background: var(--color-warning-soft); color: var(--color-text-secondary); font-size: 13px; line-height: 1.5; }
.inline-action { min-height: 32px; margin-left: 8px; border: 0; background: transparent; color: var(--color-primary); font: inherit; font-weight: 700; text-decoration: underline; text-underline-offset: 3px; cursor: pointer; }

@media (max-width: 920px) {
 .notification-row { grid-template-columns: 24px minmax(0, 1fr) 126px; grid-template-areas: "icon copy state" ". frequency frequency" ". save save"; }
 .frequency-control { margin-top: 10px; }
}

@media (max-width: 640px) {
 .section-header { display: grid; gap: 12px; }
 .autosave-note { width: fit-content; }
 .notification-row { grid-template-columns: 24px minmax(0, 1fr); grid-template-areas: "icon copy" ". state" ". frequency" ". save"; align-items: start; min-height: 0; padding: 17px 15px; }
 .notification-state { margin-top: 10px; }
 .frequency-control { width: 100%; }
 .save-state { justify-content: flex-start; }
}
</style>
