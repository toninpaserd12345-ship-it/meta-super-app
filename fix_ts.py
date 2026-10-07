import re

# 1. Fix types/automation.ts
content = open("frontend/types/automation.ts").read()
content = content.replace("triggerValue: string", "triggerValue: string\n  triggerName?: string")
open("frontend/types/automation.ts", "w").write(content)

# 2. Fix pages/ads.vue
content = open("frontend/pages/ads.vue").read()
content = content.replace("import type { MetaAdAccount, MetaCampaign, MetaAd, AutomationRule, Product, ReplySet, MetaPage } from '~/types'", "import type { MetaAdAccount, MetaCampaign, MetaAd, AutomationRule, Product, ReplySet, MetaPage } from '~/types/automation'")
content = content.replace("r => r.triggerType", "(r: any) => r.triggerType")
content = content.replace("a => ({ ...a, label", "(a: any) => ({ ...a, label")
content = content.replace("s => s.items.some(i => i.isEnabled)", "(s: any) => s.items.some((i: any) => i.isEnabled)")
content = content.replace("v-if=\"expandedCampaigns.includes(campaign.id) && campaignAds[campaign.id]?.length > 0\"", "v-if=\"expandedCampaigns.includes(campaign.id) && (campaignAds[campaign.id]?.length || 0) > 0\"")
content = content.replace("campaignAds[campaign.id].length", "(campaignAds[campaign.id]?.length || 0)")
open("frontend/pages/ads.vue", "w").write(content)
