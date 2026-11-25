<script setup lang="ts">
definePageMeta({ layout: 'auth', middleware: 'auth' })

await navigateTo('/meta-pages', { replace: true })
</script>

<template>
  <span class="sr-only">Redirecting to Facebook Pages</span>
</template>

<style scoped>
.sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
</style>
