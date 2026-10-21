<template>
  <div class="app-layout">
    <AppSidebar/>
    <main class="app-main"><div class="content-container"><AppHeader/><slot/></div></main>
    <AppMobileNav/>
  </div>
</template>
<style scoped>
.app-layout{min-height:100dvh;background:var(--color-background)}.app-main{min-width:0;margin-left:var(--sidebar-width);min-height:100dvh}.content-container{width:100%;max-width:var(--content-max-width);margin:0 auto;padding:20px clamp(20px,2.5vw,40px) 52px}
@media(max-width:1023px) and (min-width:641px){.app-main{margin-left:var(--sidebar-compact-width)}.content-container{padding:18px 22px 44px}}
@media(max-width:640px){.app-main{margin-left:0}.content-container{padding:12px 12px calc(82px + env(safe-area-inset-bottom))}}
</style>
