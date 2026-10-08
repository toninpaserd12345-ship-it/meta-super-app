import re
content = open("frontend/pages/ads.vue").read()

content = content.replace("  <v-card class=\\"mb-4 pa-4\\" variant=\\"outlined\\">", "  <v-alert v-if=\\"notice && !bindDialog\\" type=\\"error\\" variant=\\"tonal\\" class=\\"mb-4\\" closable @click:close=\\"notice=''\\">{{ notice }}</v-alert>\\n  <v-card class=\\"mb-4 pa-4\\" variant=\\"outlined\\">")

open("frontend/pages/ads.vue", "w").write(content)
