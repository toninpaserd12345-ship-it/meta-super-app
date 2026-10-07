import sys
content = open("frontend/pages/auto-replies.vue").read()

old_cond = "if (!selectedProduct.value || !selectedSet.value || !selectedPage.value || !targetIds.value.length) return"
new_cond = "if (!selectedSet.value || !selectedPage.value || !targetIds.value.length) return"
content = content.replace(old_cond, new_cond)

# Wait, there's another place where 'stepReady' might require selectedProduct!
# Let's see how 'stepReady' is defined in auto-replies.vue.
