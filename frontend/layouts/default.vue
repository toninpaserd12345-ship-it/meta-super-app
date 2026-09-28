<template>
  <div class="app-layout">
    <AppSidebar/>
    <main class="app-main"><div class="content-container"><AppHeader/><slot/></div></main>
    <AppMobileNav/>
  </div>
</template>
<style scoped>
.app-layout{min-height:100dvh;background:var(--color-background)}.app-main{min-width:0;margin-left:236px;min-height:100dvh}.content-container{width:100%;max-width:1680px;margin:0 auto;padding:40px clamp(24px,4vw,64px) 56px}
@media(min-width:1800px){.content-container{max-width:none}}
@media(max-width:1023px) and (min-width:641px){.app-main{margin-left:78px}.content-container{padding:32px 28px 48px}}
@media(max-width:640px){.app-main{margin-left:0}.content-container{padding:24px 16px calc(100px + env(safe-area-inset-bottom))}}
</style>
