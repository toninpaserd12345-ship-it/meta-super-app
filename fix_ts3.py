content = open("frontend/pages/ads.vue").read()
content = content.replace("adAccounts.value[0].id", "adAccounts.value[0]?.id")
content = content.replace("pages.value[0].id", "pages.value[0]?.id")
open("frontend/pages/ads.vue", "w").write(content)

content = open("frontend/pages/live-chat.vue").read()
content = content.replace("conversations.value[key].sort", "conversations.value[key]?.sort")
open("frontend/pages/live-chat.vue", "w").write(content)
