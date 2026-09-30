<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
await navigateTo('/auto-replies', { replace: true, redirectCode: 301 })
</script>

<template>
  <div class="redirecting"><v-progress-circular indeterminate color="primary"/><span>Opening Auto Replies…</span></div>
</template>

<style scoped>
.redirecting{min-height:360px;display:grid;place-items:center;align-content:center;gap:14px;color:var(--color-text-secondary)}
</style>
