<script setup lang="ts">
definePageMeta({ layout: 'auth' })
const { user, fetchUser } = useAuth()
if (!user.value) await fetchUser()
if (user.value) await navigateTo('/')
const errorMessage = ref('')
const facebookLoading = ref(false)
const facebookResult = useRoute().query.facebook
if(facebookResult==='error')errorMessage.value='Facebook sign-in could not be completed. Please start again.'
if(facebookResult==='expired')errorMessage.value='This Facebook connection request expired. Click Continue with Facebook to start a new request.'
async function loginWithFacebook(){facebookLoading.value=true;errorMessage.value='';try{const result=await $fetch<{authorizationUrl:string}>('/api/auth/facebook/start',{method:'POST'});window.location.assign(result.authorizationUrl)}catch{errorMessage.value='Unable to start Facebook Login.';facebookLoading.value=false}}
</script>

<template>
  <main class="login-page">
    <div class="orb one"/><div class="orb two"/>
    <section class="login-shell">
      <aside class="brand-panel">
        <div>
          <div class="brand-mark">M</div><div class="eyebrow">META SUPER APP</div>
          <h1 class="text-balance">One calm place to run your business.</h1>
          <p class="brand-copy">Bring your teams, daily operations, and business insights together in a workspace built to stay simple.</p>
        </div>
        <div class="trust-card"><div class="trust-icon"><v-icon icon="mdi-shield-check-outline" size="22"/></div><div><strong>Secure by design</strong><p>Your session is protected with server-side authentication and encrypted connections.</p></div></div>
        <div class="panel-footer"><div class="avatar-stack"><span>KT</span><span>AN</span><span>PM</span></div><span>Built for modern teams</span></div>
      </aside>
      <section class="form-panel">
        <div class="mobile-brand"><span>M</span><strong>Meta Super App</strong></div>
        <div class="form-wrap">
          <div class="welcome-icon"><v-icon icon="mdi-hand-wave-outline" size="25"/></div>
          <p class="form-kicker">WELCOME</p><h2>Connect your Facebook</h2><p class="form-subtitle">Continue securely with Facebook to access and manage your Pages.</p>
          <v-alert v-if="errorMessage" type="error" variant="tonal" density="comfortable" closable class="error-alert mb-5" @click:close="errorMessage = ''">{{ errorMessage }}</v-alert>
          <v-btn color="#1877F2" size="x-large" block prepend-icon="mdi-facebook" :loading="facebookLoading" class="facebook-btn" @click="loginWithFacebook">Continue with Facebook</v-btn>
          <div class="facebook-note"><v-icon icon="mdi-shield-lock-outline" size="18"/><span>We never receive your Facebook password. Authentication is handled securely by Meta.</span></div>
        </div>
        <footer>© {{ new Date().getFullYear() }} Meta Super App · Privacy · Security · <strong>Version 1.0.1 (New UI)</strong></footer>
      </section>
    </section>
  </main>
</template>

<style scoped>
.login-page{min-height:100vh;display:grid;place-items:center;padding:32px;position:relative;overflow:hidden;background:var(--color-background)}.login-shell{width:min(1120px,100%);min-height:690px;display:grid;grid-template-columns:.92fr 1.08fr;background:var(--color-surface);border-radius:28px;overflow:hidden;box-shadow:0 28px 80px rgba(31,35,75,.12);position:relative;z-index:1}.brand-panel{padding:54px;color:var(--color-surface);background:var(--gradient-brand);display:flex;flex-direction:column;justify-content:space-between;position:relative;overflow:hidden}.brand-panel:after{content:'';position:absolute;width:370px;height:370px;border:1px solid rgba(255,255,255,.12);border-radius:50%;right:-180px;top:-80px;box-shadow:0 0 0 55px rgba(255,255,255,.035),0 0 0 110px rgba(255,255,255,.025)}.brand-mark,.mobile-brand>span{width:48px;height:48px;border-radius:15px;display:grid;place-items:center;font-size:23px;font-weight:800;background:rgba(255,255,255,.16);border:1px solid rgba(255,255,255,.25)}.eyebrow{margin-top:64px;font-size:12px;font-weight:800;letter-spacing:.18em;color:var(--color-brand-caption)}.brand-panel h1{font-size:clamp(40px,4vw,56px);line-height:1.06;letter-spacing:-.045em;margin:18px 0 22px;max-width:460px}.brand-copy{max-width:425px;color:rgba(255,255,255,.72);font-size:16px;line-height:1.75}.trust-card{display:flex;gap:15px;padding:18px;border:1px solid rgba(255,255,255,.14);background:rgba(255,255,255,.08);border-radius:18px;backdrop-filter:blur(12px);z-index:1}.trust-icon{min-width:42px;height:42px;display:grid;place-items:center;border-radius:12px;background:rgba(255,255,255,.13)}.trust-card strong{font-size:14px}.trust-card p{margin:4px 0 0;color:rgba(255,255,255,.65);font-size:12px;line-height:1.55}.panel-footer{display:flex;align-items:center;gap:13px;color:rgba(255,255,255,.7);font-size:12px;font-weight:600}.avatar-stack{display:flex}.avatar-stack span{width:29px;height:29px;margin-right:-8px;display:grid;place-items:center;border-radius:50%;border:2px solid var(--color-brand-avatar-border);background:var(--color-brand-avatar-bg);color:var(--color-brand-avatar-text);font-size:9px;font-weight:800}.form-panel{padding:50px 70px 28px;display:flex;flex-direction:column;justify-content:space-between}.form-wrap{width:min(420px,100%);margin:auto}.mobile-brand{display:none}.welcome-icon{width:49px;height:49px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-softer);border-radius:15px;margin-bottom:22px}.form-kicker{color:var(--color-primary);letter-spacing:.14em;font-size:11px;font-weight:800;margin:0 0 9px}.form-wrap h2{margin:0;color:var(--color-text);font-size:31px;letter-spacing:-.035em;line-height:1.2}.form-subtitle{margin:10px 0 30px;color:var(--color-text-secondary);font-size:14px}.login-form{display:grid;gap:12px}.login-form label{font-size:13px;font-weight:700;color:var(--color-text)}.label-row{display:flex;justify-content:space-between;align-items:center;margin-top:7px}.login-form :deep(.v-field){border-radius:13px;background:var(--color-surface-soft)}.login-form :deep(.v-field__outline){--v-field-border-opacity:.12}.login-form :deep(.v-field--focused .v-field__outline){--v-field-border-opacity:1}.login-form :deep(.v-checkbox){margin:2px 0 3px}.login-form :deep(.v-label){font-size:13px;font-weight:500;opacity:.85}.text-link{border:0;padding:0;background:none;color:var(--color-primary);font:inherit;font-size:12px;font-weight:700;cursor:pointer}.text-link:hover{text-decoration:underline}.sign-in-btn{height:54px!important;text-transform:none;letter-spacing:0;font-weight:700;box-shadow:0 10px 24px rgba(91,80,230,.25)!important}.help-copy{text-align:center;color:var(--color-text-secondary);margin:27px 0 0;font-size:12px}.form-panel footer{color:var(--color-text-faint);font-size:10px;text-align:center;padding-top:25px}.orb{position:absolute;border-radius:50%}.orb.one{width:280px;height:280px;right:-100px;top:-110px;background:var(--color-orb-primary)}.orb.two{width:220px;height:220px;left:-90px;bottom:-100px;background:var(--color-orb-secondary)}.error-alert{border-radius:13px;font-size:13px}
@media(max-width:900px){.login-page{padding:20px}.login-shell{grid-template-columns:1fr;min-height:0;max-width:580px}.brand-panel{display:none}.form-panel{min-height:calc(100vh - 40px);padding:32px 34px 24px}.mobile-brand{display:flex;align-items:center;gap:12px;color:var(--color-text)}.mobile-brand>span{width:39px;height:39px;border-radius:12px;color:var(--color-surface);background:var(--color-primary);font-size:18px}.form-wrap{margin:55px auto auto}}
@media(max-width:520px){.login-page{padding:0}.login-shell{min-height:100vh;border-radius:0}.form-panel{min-height:100vh;padding:24px}.form-wrap h2{font-size:27px}.form-wrap{margin-top:45px}}
.local-badge{display:flex;align-items:center;gap:8px;margin:-17px 0 22px;padding:10px 12px;color:var(--color-primary);background:var(--color-primary-soft);border:1px solid var(--color-primary-softer);border-radius:var(--radius-md);font-size:11px}
.facebook-btn{height:54px!important;color:#fff!important;text-transform:none;letter-spacing:0;font-weight:750;box-shadow:0 10px 24px rgba(24,119,242,.22)!important}
.facebook-note{display:flex;align-items:flex-start;gap:9px;margin-top:18px;padding:13px;color:var(--color-text-secondary);background:var(--color-surface-soft);border-radius:12px;font-size:11px;line-height:1.55}
</style>
