<script setup lang="ts">
withDefaults(defineProps<{ as?: string; padded?: boolean }>(), { as: 'section', padded: true })
</script>
<template><component :is="as" class="ui-panel" :class="{ 'ui-panel--padded': padded }"><slot/></component></template>
<style scoped>.ui-panel--padded{padding:var(--space-6)}@media(max-width:640px){.ui-panel--padded{padding:var(--space-5)}}</style>
