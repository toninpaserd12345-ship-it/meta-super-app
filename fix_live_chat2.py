content = open("frontend/pages/live-chat.vue").read()

content = content.replace("const url = `${baseURL}/chat/stream?token=${token.value}&account_id=${currentAccountID.value}`", "const url = '/api/proxy/api/v1/chat/stream'")

open("frontend/pages/live-chat.vue", "w").write(content)
