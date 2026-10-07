content = open("frontend/pages/live-chat.vue").read()

# Remove useAuth token
content = content.replace("const { token, currentAccountID } = useAuth()", "")
# Remove useApi
content = content.replace("const api = useApi()", "")

# Replace api.post
content = content.replace("await api.post('/chat/send', {", "await $fetch('/api/proxy/api/v1/chat/send', {\n    method: 'POST',")
content = content.replace("const history = await api.get('/chat/history') as ChatMessage[]", "const history = await $fetch<{items: ChatMessage[]}>('/api/proxy/api/v1/chat/history').then(r => r.items)")

# msgs[0]?.platform
content = content.replace("const pageId = msgs[0]?.page_id", "const pageId = msgs?.[0]?.page_id")
content = content.replace("v-icon :icon=\"msgs[0].platform", "v-icon :icon=\"msgs?.[0]?.platform")
content = content.replace("{{ msgs[msgs.length - 1].message || 'Media message' }}", "{{ msgs?.[msgs.length - 1]?.message || 'Media message' }}")

open("frontend/pages/live-chat.vue", "w").write(content)
