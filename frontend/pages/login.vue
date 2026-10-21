<script setup lang="ts">
definePageMeta({ layout: 'auth' })
const { user, fetchUser } = useAuth()
const { locale, setLocale, l } = useLocale()
if (!user.value) await fetchUser()
if (user.value) await navigateTo('/')
const errorMessage = ref('')
const facebookLoading = ref(false)
const facebookResult = useRoute().query.facebook
if(facebookResult==='error')errorMessage.value=l({lo:'ບໍ່ສາມາດເຂົ້າດ້ວຍ Facebook ໄດ້ ກະລຸນາເລີ່ມໃໝ່',th:'ไม่สามารถเข้าสู่ระบบด้วย Facebook ได้ โปรดเริ่มใหม่',en:'Facebook sign-in could not be completed. Please start again.'})
if(facebookResult==='expired')errorMessage.value=l({lo:'ຄຳຮ້ອງຂໍ Facebook ໝົດອາຍຸ ກະລຸນາເລີ່ມໃໝ່',th:'คำขอ Facebook หมดอายุ โปรดเริ่มใหม่',en:'This Facebook connection request expired. Click Continue with Facebook to start a new request.'})
const emailLoading = ref(false)
const email = ref('')
const password = ref('')

async function loginWithFacebook(){facebookLoading.value=true;errorMessage.value='';try{const result=await $fetch<{authorizationUrl:string}>('/api/auth/facebook/start',{method:'POST'});window.location.assign(result.authorizationUrl)}catch{errorMessage.value=l({lo:'ບໍ່ສາມາດເລີ່ມ Facebook Login ໄດ້',th:'ไม่สามารถเริ่ม Facebook Login ได้',en:'Unable to start Facebook Login.'});facebookLoading.value=false}}

async function loginWithEmail(){
  if(!email.value || !password.value) { errorMessage.value = l({lo:'ກະລຸນາໃສ່ Email ແລະລະຫັດຜ່ານ',th:'กรุณากรอกอีเมลและรหัสผ่าน',en:'Email and password are required.'}); return; }
  emailLoading.value=true;errorMessage.value='';
  try {
    await $fetch('/api/auth/login',{method:'POST', body: { email: email.value, password: password.value }});
    navigateTo('/meta-pages')
  } catch(err: any) {
    errorMessage.value = err.statusMessage || err.message || l({lo:'ເຂົ້າລະບົບບໍ່ສຳເລັດ',th:'เข้าสู่ระบบไม่สำเร็จ',en:'Login failed.'});
    emailLoading.value=false;
  }
}
</script>

<template>
  <main class="login-page">
 <div class="login-locale" :aria-label="l({lo:'ເລືອກພາສາ',th:'เลือกภาษา',en:'Choose language'})"><button :class="{active:locale==='lo'}" @click="setLocale('lo')">ລາວ</button><button :class="{active:locale==='th'}" @click="setLocale('th')">ไทย</button><button :class="{active:locale==='en'}" @click="setLocale('en')">EN</button></div>
    <section class="login-shell">
      <aside class="brand-panel">
          <div class="brand-header">
 <div class="brand-mark" aria-hidden="true">M</div>
            <span class="brand-title">Meta Super App</span>
          </div>
 <div class="brand-message">
 <h1>{{l({lo:'ຮ້ານ, ໂຄສະນາ ແລະລູກຄ້າ ຢູ່ໃນ workspace ດຽວ',th:'ร้าน โฆษณา และลูกค้า อยู่ในเวิร์กสเปซเดียว',en:'Your store, advertising and customers in one workspace.'})}}</h1>
 <p>{{l({lo:'ຕິດຕາມເສັ້ນທາງຈາກ Campaign ໄປຫາ Automation, Order ແລະລາຍຮັບ ໂດຍບໍ່ຕ້ອງສະຫຼັບຫຼາຍລະບົບ.',th:'ติดตามเส้นทางจากแคมเปญไปยังระบบอัตโนมัติ ออเดอร์ และรายได้ โดยไม่ต้องสลับหลายระบบ',en:'Follow the path from campaign to automation, order and revenue without switching tools.'})}}</p>
        </div>
 <ol class="workflow-list" :aria-label="l({lo:'ຂັ້ນຕອນການເຮັດວຽກ',th:'ขั้นตอนการทำงาน',en:'Workspace flow'})">
 <li><span>1</span><div><strong>Facebook &amp; WhatsApp</strong><small>{{l({lo:'ຮັບຂໍ້ຄວາມ ແລະ Lead',th:'รับข้อความและลีด',en:'Capture messages and leads'})}}</small></div></li>
 <li><span>2</span><div><strong>Automation</strong><small>{{l({lo:'ຈັດການ reply set ແລະການມອບໝາຍ',th:'จัดการชุดตอบกลับและการมอบหมาย',en:'Route replies and assignments'})}}</small></div></li>
 <li><span>3</span><div><strong>Orders &amp; Revenue</strong><small>{{l({lo:'ປິດການຂາຍ ແລະວັດຜົນ',th:'ปิดการขายและวัดผล',en:'Close sales and measure outcomes'})}}</small></div></li>
 </ol>
 <p class="panel-footer"><v-icon icon="mdi-shield-lock-outline" size="18"/><span>{{l({lo:'Facebook Login ຈັດການໂດຍ Meta; ລະບົບບໍ່ເກັບລະຫັດຜ່ານ Facebook.',th:'Facebook Login จัดการโดย Meta ระบบไม่จัดเก็บรหัสผ่าน Facebook',en:'Facebook Login is handled by Meta; this app never stores your Facebook password.'})}}</span></p>
      </aside>
      <section class="form-panel">
 <div class="mobile-brand"><span aria-hidden="true">M</span><strong>Meta Super App</strong></div>
        <div class="form-wrap">
 <h2>{{l({lo:'ເຂົ້າສູ່ Workspace',th:'เข้าสู่เวิร์กสเปซ',en:'Sign in to your workspace'})}}</h2><p class="form-subtitle">{{l({lo:'ສະມາຊິກທີມໃຊ້ email; ເຈົ້າຂອງຮ້ານສາມາດສືບຕໍ່ດ້ວຍ Facebook.',th:'สมาชิกทีมใช้อีเมล เจ้าของร้านสามารถดำเนินการต่อด้วย Facebook',en:'Team members use email; store owners can continue with Facebook.'})}}</p>
          <v-alert v-if="errorMessage" type="error" variant="tonal" density="comfortable" closable class="error-alert mb-5" @click:close="errorMessage = ''">{{ errorMessage }}</v-alert>
          
          <form class="login-form mb-5" @submit.prevent="loginWithEmail">
            <div class="form-group">
              <label for="login-email">{{l({lo:'ອີເມວ',th:'อีเมล',en:'Email Address'})}}</label>
              <input 
                id="login-email"
                type="email" 
                v-model="email" 
                required 
                placeholder="name@example.com" 
                class="login-input" 
                :class="{ 'input-error': !!errorMessage }"
                :aria-invalid="!!errorMessage"
              />
            </div>
            <div class="form-group mt-3">
              <label for="login-password">{{l({lo:'ລະຫັດຜ່ານ',th:'รหัสผ่าน',en:'Password'})}}</label>
              <input 
                id="login-password"
                type="password" 
                v-model="password" 
                required 
                :placeholder="l({lo:'ໃສ່ລະຫັດຜ່ານ',th:'กรอกรหัสผ่าน',en:'Enter password'})" 
                class="login-input" 
                :class="{ 'input-error': !!errorMessage }"
                :aria-invalid="!!errorMessage"
              />
            </div>
            <button type="submit" class="btn-primary sign-in-btn mt-4 full-width" :disabled="emailLoading">{{ emailLoading ? l({lo:'ກຳລັງເຂົ້າ...',th:'กำลังเข้าสู่ระบบ...',en:'Signing in...'}) : l({lo:'ເຂົ້າລະບົບ',th:'เข้าสู่ระบบ',en:'Sign In'}) }}</button>
          </form>
          
          <div class="divider"><span>{{l({lo:'ຫຼື',th:'หรือ',en:'OR'})}}</span></div>

          <v-btn size="x-large" block prepend-icon="mdi-facebook" :loading="facebookLoading" class="facebook-btn" @click="loginWithFacebook">{{l({lo:'ສືບຕໍ່ດ້ວຍ Facebook',th:'ดำเนินการต่อด้วย Facebook',en:'Continue with Facebook'})}}</v-btn>
 <p class="facebook-note"><v-icon icon="mdi-information-outline" size="18"/><span>{{l({lo:'Facebook ໃຊ້ສຳລັບ owner ເພື່ອເຊື່ອມ Page, Ads ແລະ WhatsApp.',th:'Facebook ใช้สำหรับเจ้าของเพื่อเชื่อมต่อเพจ โฆษณา และ WhatsApp',en:'Owners use Facebook to connect Pages, Ads and WhatsApp.'})}}</span></p>
        </div>
 <footer> {{ new Date().getFullYear() }} Meta Super App</footer>
      </section>
    </section>
  </main>
</template>

<style scoped>
.login-page{min-height:100vh;display:grid;place-items:center;padding:32px;background:var(--color-background);position:relative}.login-shell{width:min(1080px,100%);min-height:680px;display:grid;grid-template-columns:minmax(0,.92fr) minmax(0,1.08fr);overflow:hidden;background:var(--color-surface);border:1px solid var(--color-border);border-radius:16px;box-shadow:0 18px 55px -34px rgba(15,23,42,.45)}
.brand-panel{display:flex;flex-direction:column;padding:48px;color:var(--color-background);background:#172033}.brand-header{display:flex;align-items:center;gap:12px}.brand-mark,.mobile-brand>span{display:grid;place-items:center;width:40px;height:40px;border-radius:10px;background:#eef2ff;color:#3346a8;font-size:18px;font-weight:850}.brand-title{font-size:17px;font-weight:750}.brand-message{margin:auto 0 34px}.brand-message h1{max-width:12ch;margin:0 0 18px;font-size:clamp(34px,4vw,50px);line-height:1.08;letter-spacing:-.035em}.brand-message p{max-width:49ch;margin:0;color:#b9c2d3;font-size:15px;line-height:1.75}.workflow-list{display:grid;gap:0;margin:0 0 38px;padding:0;list-style:none;border-top:1px solid rgba(255,255,255,.14)}.workflow-list li{display:grid;grid-template-columns:30px 1fr;gap:14px;align-items:start;padding:15px 0;border-bottom:1px solid rgba(255,255,255,.14)}.workflow-list li>span{display:grid;place-items:center;width:24px;height:24px;border:1px solid #7482a0;border-radius:7px;color:#d9e0ed;font-size:12px;font-weight:750}.workflow-list strong,.workflow-list small{display:block}.workflow-list strong{font-size:13px}.workflow-list small{margin-top:3px;color:#9faac0;font-size:12px}.panel-footer{display:flex;gap:10px;align-items:flex-start;margin:0;color:#9faac0;font-size:12px;line-height:1.55}
.form-panel{display:flex;flex-direction:column;padding:48px 64px 26px}.form-wrap{width:min(410px,100%);margin:auto}.mobile-brand{display:none}.form-wrap h2{margin:0;color:var(--color-text);font-size:30px;line-height:1.2;letter-spacing:-.025em}.form-subtitle{margin:10px 0 30px;color:var(--color-text-secondary);font-size:14px;line-height:1.6}.login-form{display:grid;gap:14px}.login-form label{font-size:13px;font-weight:700;color:var(--color-text)}.login-input{width:100%;min-height:48px;margin-top:6px;padding:11px 13px;border:1px solid var(--color-border-strong);border-radius:10px;background:var(--color-surface);color:var(--color-text);font:inherit;font-size:14px}.login-input:focus-visible{outline:3px solid color-mix(in srgb,var(--color-primary) 22%,transparent);outline-offset:2px;border-color:var(--color-primary)}.sign-in-btn{min-height:48px!important}.facebook-btn{min-height:48px!important;background:var(--color-facebook)!important;color:var(--color-surface)!important;text-transform:none;letter-spacing:0;font-weight:700}.facebook-note{display:flex;align-items:flex-start;gap:8px;margin:16px 0 0;color:var(--color-text-secondary);font-size:12px;line-height:1.55}.divider{display:flex;align-items:center;gap:12px;margin:20px 0;color:var(--color-text-muted);font-size:12px;font-weight:700}.divider:before,.divider:after{content:'';height:1px;flex:1;background:var(--color-border)}.full-width{width:100%}.form-panel footer{padding-top:24px;color:var(--color-text-muted);font-size:12px;text-align:center}.error-alert{font-size:13px}
.login-locale{position:absolute;z-index:3;top:20px;right:20px;display:flex;gap:2px;padding:3px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:10px}.login-locale button{min-width:44px;min-height:36px;border:0;border-radius:7px;background:transparent;color:var(--color-text-secondary);font:inherit;font-size:12px;font-weight:700;cursor:pointer}.login-locale button:hover{background:var(--color-surface-soft);color:var(--color-text)}.login-locale button.active{color:var(--color-primary);background:var(--color-primary-soft)}
@media(max-width:860px){.login-page{padding:20px}.login-shell{grid-template-columns:1fr;max-width:590px;min-height:0}.brand-panel{display:none}.form-panel{min-height:calc(100vh - 40px);padding:32px}.mobile-brand{display:flex;align-items:center;gap:11px}.form-wrap{margin:54px auto auto}}
@media(max-width:520px){.login-page{padding:0}.login-shell{min-height:100vh;border:0;border-radius:0}.form-panel{min-height:100vh;padding:22px}.form-wrap{margin-top:44px}.form-wrap h2{font-size:26px}.login-locale{top:16px;right:16px}}
</style>
