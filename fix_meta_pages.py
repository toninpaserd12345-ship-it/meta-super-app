import re

content = open("frontend/pages/meta-pages.vue").read()

# 1. Fix availablePages filter
new_available = """  if (activePlatform.value !== 'all') {
     if (activePlatform.value === 'instagram') {
        list = list.filter(p => p.category.toLowerCase().includes('instagram'))
     } else if (activePlatform.value === 'whatsapp') {
        list = list.filter(p => p.category.toLowerCase().includes('whatsapp'))
     } else if (activePlatform.value === 'facebook') {
        list = list.filter(p => !p.category.toLowerCase().includes('instagram') && !p.category.toLowerCase().includes('whatsapp'))
     } else {
        list = []
     }
  }"""
content = re.sub(r"  if \(activePlatform\.value !== 'all'\) \{.*?(?=  return list)", new_available + "\n", content, flags=re.DOTALL)

# 2. Fix the hardcoded v-if in the template for not connected
content = content.replace("activePlatform !== 'facebook' && activePlatform !== 'instagram'", "activePlatform !== 'facebook' && activePlatform !== 'instagram' && activePlatform !== 'whatsapp'")

# 3. Change "Facebook Pages" to "Facebook & WhatsApp"
content = content.replace("<h2>Facebook Pages</h2>", "<h2>Facebook & WhatsApp</h2>")
content = content.replace("<strong>No Facebook Pages found.</strong><br>Connect Facebook and allow the required Page permissions", "<strong>No Facebook or WhatsApp accounts found.</strong><br>Connect Facebook and allow the required permissions")
content = content.replace("Choose the Pages you want to activate", "Choose the accounts you want to activate")
content = content.replace("Select Pages", "Select Accounts")

# 4. In the page-grid, show the WhatsApp icon if it's a WhatsApp page
content = content.replace("""<div class="page-avatar"><v-icon icon="mdi-facebook" size="28"/></div>""", """<div class="page-avatar">
        <v-img v-if="page.pictureUrl" :src="page.pictureUrl" cover></v-img>
        <v-icon v-else-if="page.category.toLowerCase().includes('whatsapp')" icon="mdi-whatsapp" color="success" size="28"/>
        <v-icon v-else icon="mdi-facebook" color="blue" size="28"/>
      </div>""")

# 5. Same in the modal picker (pc-card-avatar already handles pictureUrl, but fallback to whatsapp icon)
content = content.replace("""<v-icon v-else icon="mdi-account" color="grey" size="32"></v-icon>""", """<v-icon v-else-if="page.category.toLowerCase().includes('whatsapp')" icon="mdi-whatsapp" color="success" size="32"></v-icon>
                  <v-icon v-else icon="mdi-facebook" color="blue" size="32"></v-icon>""")

open("frontend/pages/meta-pages.vue", "w").write(content)
