export interface ItemsResponse<T> {
  items: T[]
}

export interface Product {
  id: string
  code?: string
  name: string
  price: string
  description: string
  imageUrl?: string
}

export type ReplyItemType = 'text' | 'image' | 'video' | 'audio'

export interface ReplyItem {
  id?: string
  replySetId?: string
  type: ReplyItemType
  content: string
  orderIndex: number
  isEnabled: boolean
}

export interface ReplySet {
  id: string
  code?: string
  accountId?: string
  name: string
  items: ReplyItem[]
  createdAt?: string
  updatedAt?: string
}

export type AutomationTriggerType = 'post' | 'ad' | 'campaign' | 'adset' | 'keyword'

export interface AutomationRule {
  id: string
  code?: string
  accountId?: string
  pageId: string
  triggerType: AutomationTriggerType
  triggerValue: string
  triggerName?: string
  productId: string
  replySetId: string
  isActive: boolean
  createdAt?: string
  updatedAt?: string
}

export interface AutomationTarget {
  id?: string
  type: AutomationTriggerType
  value: string
  name: string
}

export interface AutomationFlow {
  id: string
  code?: string
  accountId?: string
  name: string
  pageId: string
  productId: string
  replySetId: string
  firstMessageOnly: boolean
  cooldownSeconds: number
  isActive: boolean
  targets: AutomationTarget[]
  createdAt?: string
  updatedAt?: string
}

export interface MetaPage {
  id: string
  code?: string
  name: string
  connected: boolean
}

export interface MetaPost {
  id: string
  code?: string
  message: string
  createdTime: string
  permalinkUrl: string
  pictureUrl?: string
}

export interface MetaAdAccount {
  id: string
  code?: string
  name: string
  accountStatus: number
  accountType: 'personal' | 'business'
  businessId?: string
  businessName?: string
}

export interface MetaCampaign {
  id: string
  code?: string
  name: string
  status: string
  effectiveStatus: string
  objective: string
  adSetCount: number
  adCount: number
}

export interface MetaAd {
  id: string
  code?: string
  name: string
  status: string
  effectiveStatus: string
  thumbnailUrl?: string
  imageUrl?: string
}
