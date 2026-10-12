export interface ItemsResponse<T> {
  items: T[]
}

export interface Product {
  id: string
  code?: string
  name: string
  price: string
  salePrice?: number | null
  cost?: number | null
  quantity?: number | null
  currency?: string
  description: string
  imageUrl?: string
  barcode?: string
  category?: string
  unit?: string
  weight?: number | null
  galleryUrls?: string
  minStock?: number | null
  status?: string
}

export interface Warehouse {
  id: string
  accountId: string
  code: string
  name: string
  type: 'warehouse' | 'shop'
  address?: string
  phone?: string
  isDefault: boolean
  isActive: boolean
  createdAt?: string
  updatedAt?: string
}

export interface WarehouseStock {
  id: string
  accountId: string
  productId: string
  warehouseId: string
  warehouse?: Warehouse
  quantity: number
  alertThreshold: number
  updatedAt?: string
}

export interface OrderItem {
  id: string
  orderId: string
  productId: string
  productCode?: string
  productName: string
  productImage?: string
  quantity: number
  unitPrice: number
  totalPrice: number
}

export interface Order {
  id: string
  accountId: string
  orderNumber: string
  leadId?: string
  senderId?: string
  pageId?: string
  warehouseId: string
  warehouse?: Warehouse
  customerName: string
  customerPhone: string
  customerAddress: string
  subtotal: number
  discount: number
  shippingFee: number
  totalAmount: number
  currency: string
  paymentStatus: 'unpaid' | 'paid' | 'refunded'
  paymentMethod: string
  fulfillmentStatus: 'pending' | 'confirmed' | 'shipping' | 'delivered' | 'cancelled'
  deliveryCourier?: string
  deliveryTracking?: string
  notes?: string
  facebookLeadSent: boolean
  items: OrderItem[]
  createdAt: string
  updatedAt: string
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

export interface MetaInsightResult {
  type: string
  value?: number
  cost?: number
}

export interface MetaInsightItem {
  entityType: 'account' | 'campaign' | 'adset' | 'ad'
  entityId: string
  entityName: string
  status?: string
  effectiveStatus?: string
  campaignId?: string
  campaignName?: string
  adSetId?: string
  adSetName?: string
  adId?: string
  adName?: string
  spend?: number
  impressions?: number
  reach?: number
  frequency?: number
  linkClicks?: number
  linkCtr?: number
  linkCpc?: number
  cpm?: number
  conversionValue?: number
  roas?: number
  results: MetaInsightResult
  qualityRanking?: string
  engagementRateRanking?: string
  conversionRateRanking?: string
}

export interface MetaInsightPoint {
  date: string
  spend?: number
  results?: number
  costPerResult?: number
  linkCtr?: number
  frequency?: number
}

export interface MetaInsightsReport {
  currency: string
  timezone: string
  level: 'campaign' | 'adset' | 'ad'
  dateStart: string
  dateStop: string
  generatedAt: string
  summary?: MetaInsightItem
  items: MetaInsightItem[]
  trend: MetaInsightPoint[]
  warnings: string[]
}

export interface MetaAudienceTargeting {
  ageMin?: number
  ageMax?: number
  genders?: string[]
  countries?: string[]
  regions?: string[]
  cities?: string[]
  interests?: string[]
  customAudiences?: string[]
  excludedCustomAudiences?: string[]
  publisherPlatforms?: string[]
  devicePlatforms?: string[]
}

export interface MetaAudienceAnalysis {
  adSetId: string
  adSetName: string
  optimizationGoal?: string
  targeting: MetaAudienceTargeting
  estimatedDailyUsers?: number
  estimatedMonthlyUsers?: number
  estimateReady: boolean
  estimateSource?: string
  warnings?: string[]
}
