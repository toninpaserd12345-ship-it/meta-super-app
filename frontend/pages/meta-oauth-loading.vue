<script setup lang="ts">
definePageMeta({ layout: 'auth' })
</script>

<template>
  <main class="loading-card">
    <v-progress-circular color="primary" indeterminate size="42" />
    <h1>Opening Facebook</h1>
    <p>Please wait while the secure Facebook Login page loads.</p>
  </main>
</template>

<style scoped>
.loading-card{width:min(420px,calc(100vw - 32px));margin:auto;padding:34px;text-align:center;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
.loading-card h1{margin:18px 0 8px;font-size:var(--text-xl)}.loading-card p{margin:0;color:var(--color-text-secondary)}
</style>
