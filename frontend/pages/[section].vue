<script setup lang="ts">
import { appNavigation } from '~/config/navigation'
definePageMeta({ middleware: 'auth' })
const route = useRoute(); const { can } = useAuth()
const { locale, t } = useLocale()
const item = computed(() => appNavigation.find(entry => entry.to === `/${route.params.section}`))
if (!item.value) throw createError({ statusCode: 404, statusMessage: 'Page not found' })
if (item.value.claim && !can(item.value.claim)) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view this page.' })
</script>
<template><section class="empty-panel"><div><v-icon :icon="item?.icon" size="28"/></div><p>MODULE</p><h2>{{ item ? t(item.labelKey) : '' }}</h2><span>{{ locale==='lo'?'ສ່ວນນີ້ກຽມພ້ອມສຳລັບ feature ແລະຂໍ້ມູນຂອງ workspace ແລ້ວ':'This module is ready for feature components and workspace data.' }}</span></section></template>
<style scoped>.empty-panel{min-height:420px;display:grid;place-items:center;align-content:center;text-align:center;padding:32px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:18px}.empty-panel>div{width:54px;height:54px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-softer);border-radius:16px}.empty-panel p{margin:20px 0 7px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.14em}.empty-panel h2{margin:0;font-size:26px}.empty-panel span{max-width:440px;margin-top:10px;color:var(--color-text-secondary);font-size:13px;line-height:1.6}</style>
