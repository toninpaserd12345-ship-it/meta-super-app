export interface ItemsResponse<T> {
  items: T[]
}

export interface ProductVariant {
  id?: string
  color: string
  size: string
  quantity: number
  sku?: string
  price?: number
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
  colors?: string
  sizes?: string
  variants?: string
  packagingCost?: number | null
  shippingCostStore?: number | null
  targetProfit?: number | null
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
  color?: string
  size?: string
  quantity: number
  unitPrice: number
  totalPrice: number
  costPrice?: number
  totalCost?: number
}

export interface Order {
  id: string
  accountId: string
  orderNumber: string
  leadId?: string
  senderId?: string
  pageId?: string
  adId?: string
  campaignId?: string
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
  productCost?: number
  packagingCost?: number
  storeShippingCost?: number
  returnFee?: number
  totalVariableCost?: number
  netProfitBeforeAds?: number
  isCod?: boolean
  prepaidAmount?: number
  codAmount?: number
  codTransferStatus?: string
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
  commentReplyText?: string
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
  currency?: string
  businessCountryCode?: string
  taxId?: string
  taxIdStatus?: number
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
  messagesCount?: number
  costPerMessage?: number
  purchasesCount?: number
  costPerPurchase?: number
  qualityRanking?: string
  engagementRateRanking?: string
  conversionRateRanking?: string
  dailyBudget?: number
  lifetimeBudget?: number
  createdTime?: string
  updatedTime?: string
  trend?: MetaInsightPoint[]
}

export interface MetaInsightPoint {
  date: string
  spend?: number
  results?: number
  costPerResult?: number
  messagesCount?: number
  costPerMessage?: number
  purchasesCount?: number
  costPerPurchase?: number
  linkCtr?: number
  frequency?: number
  impressions?: number
  reach?: number
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

export interface AdStrategySettings {
  id: string
  accountId: string
  adAccountId: string
  effectiveFrom: string
  effectiveTo?: string | null
  targetProfitPerOrder: number
  avgPackagingCost: number
  avgShippingCostStore: number
  codReturnRatePercent: number
  returnFeePerOrder: number
  dailyBudgetCap: number
  maxTestLossBudget: number
  note?: string
  createdAt?: string
  updatedAt?: string
}

export interface AdPerformanceItem {
  entityType: 'campaign' | 'adset' | 'ad'
  entityId: string
  entityName: string
  campaignId?: string
  campaignName?: string
  adSetId?: string
  adSetName?: string
  status: string
  effectiveStatus: string

  spend: number
  impressions: number
  reach: number
  frequency: number
  linkClicks: number
  linkCtr: number
  linkCpc: number
  cpm: number
  reportedResults: number
  reportedCost?: number | null

  totalOrders: number
  prepaidOrders: number
  prepaidRevenue: number
  codTotalOrders: number
  codDeliveredOrders: number
  codDeliveredRevenue: number
  codPendingOrders: number
  codPendingRevenue: number
  codReturnedOrders: number
  completedOrders: number
  completedRevenue: number
  grossProfitBeforeAds: number
  netContribution: number

  realCpa?: number | null
  targetCpa: number
  breakEvenCpa: number
  closingRate?: number | null
  decisionCode: 'SCALE' | 'HOLD' | 'IMPROVE' | 'PAUSE' | 'PAUSE_TEST' | 'INSUFFICIENT_DATA'
  statusBadgeColor: 'success' | 'info' | 'warning' | 'error' | 'secondary'
  reason: string
  recommendedAction: string
  proposedBudgetChange?: string | null

  productId?: string
  productName?: string
  productImageUrl?: string
  productPrice?: string
  autoReplyFlowId?: string
  autoReplyFlowName?: string
  isLinkedToProduct?: boolean
}

export interface PerformanceAnalysisSummary {
  totalSpend: number
  totalImpressions: number
  totalChats: number
  totalPrepaidOrders: number
  totalPrepaidRevenue: number
  totalCODOrders: number
  totalCODDelivered: number
  totalCODDeliveredRev: number
  totalCODPending: number
  totalCODPendingRev: number
  totalCODReturned: number
  totalCompletedOrders: number
  totalCompletedRevenue: number
  totalGrossProfit: number
  totalNetProfit: number
  overallRealCpa?: number | null
  prepaidPercentage: number
}

export interface PerformanceActionPlan {
  toPause: string[]
  toScale: string[]
  toHold: string[]
  nextExperiments: string[]
  nextReviewDate: string
  dailyBudgetCap: number
}

export interface PerformanceAnalysisReport {
  currency: string
  timezone: string
  level: string
  dateStart: string
  dateStop: string
  generatedAt: string
  strategy: AdStrategySettings
  summary: PerformanceAnalysisSummary
  items: AdPerformanceItem[]
  actionPlan: PerformanceActionPlan
}

