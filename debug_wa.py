import re
content = open("frontend/pages/meta-pages.vue").read()

content = content.replace("whatsAppMessageHandler=(event:MessageEvent)=>{", "whatsAppMessageHandler=(event:MessageEvent)=>{ console.log('Message received!', event.origin, event.data);")
content = content.replace("async function finishWhatsAppSignup(){", "async function finishWhatsAppSignup(){ console.log('finishWhatsAppSignup called! Code:', whatsAppCode.value, 'Session:', whatsAppSession.value);")

open("frontend/pages/meta-pages.vue", "w").write(content)
