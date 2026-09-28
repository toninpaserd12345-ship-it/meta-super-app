<script setup lang="ts">
definePageMeta({ layout: 'auth' })
const route = useRoute()
const success = route.query.meta === 'connected'
const message = success ? 'Facebook connected successfully.' : String(route.query.reason || 'Facebook authorization failed.')
function closeWindow(){ window.close() }

onMounted(() => {
  if (window.opener) {
    window.opener.postMessage({ type: 'meta-oauth-complete', success, message }, window.location.origin)
    window.close()
  }
})
</script>

<template>
  <main class="complete-card">
    <v-progress-circular v-if="success" color="success" indeterminate />
    <v-icon v-else icon="mdi-alert-circle-outline" color="error" size="42" />
    <h1>{{ success ? 'Facebook connected' : 'Connection failed' }}</h1>
    <p>{{ message }}</p>
    <v-btn v-if="!success" color="primary" variant="flat" @click="closeWindow">Close</v-btn>
  </main>
</template>

<style scoped>
.complete-card{width:min(420px,calc(100vw - 32px));margin:auto;padding:32px;text-align:center;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
.complete-card h1{margin:18px 0 8px;font-size:var(--text-xl)}.complete-card p{margin:0 0 20px;color:var(--color-text-secondary)}
</style>
