<script setup lang="ts">
import { appNavigation } from '~/config/navigation'
const route = useRoute(); const { can } = useAuth()
const items = computed(() => appNavigation.filter(item => !item.claim || can(item.claim)).slice(0, 4))
</script>
<template><nav class="mobile-nav" aria-label="Mobile navigation"><NuxtLink v-for="item in items" :key="item.to" :to="item.to" :class="{ active: route.path === item.to }"><v-icon :icon="item.icon" size="21"/><span>{{ item.label }}</span></NuxtLink></nav></template>
<style scoped>
.mobile-nav{display:none}@media(max-width:640px){.mobile-nav{height:68px;display:flex;position:fixed;z-index:30;inset:auto 10px max(10px,env(safe-area-inset-bottom)) 10px;padding:6px;background:rgba(255,255,255,.94);border:1px solid var(--color-border);border-radius:19px;box-shadow:0 12px 34px rgba(31,35,75,.18);backdrop-filter:blur(16px)}.mobile-nav a{flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:2px;border-radius:14px;color:var(--color-text-secondary);text-decoration:none;font-size:9px;font-weight:650}.mobile-nav a.active{color:var(--color-primary);background:var(--color-primary-soft)}}
</style>
