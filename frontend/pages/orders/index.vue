<script setup lang="ts">
import type { ItemsResponse, MetaPage, Order, OrderItem, Product, Warehouse } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

const { can, activeAccount } = useAuth()
const { l } = useLocale()

if (!can('orders:read') && !can('pages:read')) {
  throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Orders.' })
}

const { data: ordersData, pending, error, refresh } = await useApi<ItemsResponse<Order>>('/proxy/api/v1/orders')
const { data: warehousesData } = await useApi<ItemsResponse<Warehouse>>('/proxy/api/v1/warehouses')
const { data: productsData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: pagesData } = await useApi<ItemsResponse<MetaPage>>('/proxy/api/v1/meta/pages')

const orders = computed(() => ordersData.value?.items || [])
const warehouses = computed(() => warehousesData.value?.items || [])
const products = computed(() => productsData.value?.items || [])
const pages = computed(() => pagesData.value?.items || [])

function getPageName(pageId?: string) {
  if (!pageId) return ''
  return pages.value.find((p) => p.id === pageId)?.name || ''
}

const activeTab = ref<'all' | 'prepaid' | 'cod' | 'unpaid' | 'paid' | 'shipping' | 'delivered' | 'cancelled'>('all')
const searchQuery = ref('')
const selectedWarehouseFilter = ref<string>('all')

const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')

// Create Order Dialog
const createDialog = ref(false)
const creating = ref(false)

interface OrderFormItem {
  productId: string
  productCode: string
  productName: string
  productImage: string
  color?: string
  size?: string
  quantity: number
  unitPrice: number
}

const orderForm = reactive<{
  pageId: string
  warehouseId: string
  customerName: string
  customerPhone: string
  customerAddress: string
  shippingFee: number
  discount: number
  currency: string
  paymentStatus: 'unpaid' | 'paid'
  paymentMethod: string
  prepaidAmount: number
  codAmount: number
  deliveryCourier: string
  deliveryTracking: string
  notes: string
  items: OrderFormItem[]
}>({
  pageId: '',
  warehouseId: '',
  customerName: '',
  customerPhone: '',
  customerAddress: '',
  shippingFee: 0,
  discount: 0,
  currency: 'LAK',
  paymentStatus: 'paid',
  paymentMethod: 'Bank Transfer',
  prepaidAmount: 0,
  codAmount: 0,
  deliveryCourier: 'Anousith Express',
  deliveryTracking: '',
  notes: '',
  items: [],
})

// Selected Product for adding to line items
const selectedProductToAdd = ref<Product | null>(null)
const selectedColorToAdd = ref<string>('')
const selectedSizeToAdd = ref<string>('')

const availableColorsForSelected = computed(() => {
  if (!selectedProductToAdd.value?.colors) return []
  return selectedProductToAdd.value.colors.split(',').map((s) => s.trim()).filter(Boolean)
})

const availableSizesForSelected = computed(() => {
  if (!selectedProductToAdd.value?.sizes) return []
  return selectedProductToAdd.value.sizes.split(',').map((s) => s.trim()).filter(Boolean)
})

watch(selectedProductToAdd, (p) => {
  if (p) {
    const cList = (p.colors || '').split(',').map((s) => s.trim()).filter(Boolean)
    const sList = (p.sizes || '').split(',').map((s) => s.trim()).filter(Boolean)
    selectedColorToAdd.value = cList[0] || ''
    selectedSizeToAdd.value = sList[0] || ''
  } else {
    selectedColorToAdd.value = ''
    selectedSizeToAdd.value = ''
  }
})

// Order Detail / Receipt Modal
const receiptDialog = ref(false)
const selectedOrder = ref<Order | null>(null)
const copiedReceipt = ref(false)

// Shop / Sender Information for Waybill & Receipt Printing
const senderShopName = ref('')
const senderShopPhone = ref('')
const senderShopAddress = ref('')
const senderSavedNotice = ref(false)

function initSenderInfo(order?: Order | null) {
  const savedPhone = typeof localStorage !== 'undefined' ? localStorage.getItem('meta_shop_phone') : ''
  const savedAddress = typeof localStorage !== 'undefined' ? localStorage.getItem('meta_shop_address') : ''
  const savedName = typeof localStorage !== 'undefined' ? localStorage.getItem('meta_shop_name') : ''

  // 1. "ຊື່ page ໃດ ອອກຊື່ນັ້ນ" (If order originated from a Facebook Page, prioritize that Page's exact Name!)
  if (order?.pageId) {
    const page = pages.value.find((p) => p.id === order.pageId)
    if (page?.name) {
      senderShopName.value = page.name
      senderShopPhone.value = order.warehouse?.phone || savedPhone || '020 5555 7777'
      senderShopAddress.value = order.warehouse?.address || savedAddress || 'ນະຄອນຫຼວງວຽງຈັນ (Vientiane)'
      return
    }
  }

  // 2. If order explicitly belongs to a Shop warehouse
  if (order?.warehouse?.type === 'shop' && order.warehouse.name) {
    senderShopName.value = order.warehouse.name
    senderShopPhone.value = order.warehouse.phone || savedPhone || '020 5555 7777'
    senderShopAddress.value = order.warehouse.address || savedAddress || 'ນະຄອນຫຼວງວຽງຈັນ (Vientiane)'
    return
  }

  // 3. If connected to Facebook Pages, default to the first connected Page's name!
  if (pages.value.length > 0 && pages.value[0]?.name && !savedName) {
    senderShopName.value = pages.value[0].name
    const firstShop = warehouses.value.find((w) => w.type === 'shop')
    senderShopPhone.value = firstShop?.phone || order?.warehouse?.phone || savedPhone || '020 5555 7777'
    senderShopAddress.value = firstShop?.address || order?.warehouse?.address || savedAddress || 'ນະຄອນຫຼວງວຽງຈັນ (Vientiane)'
    return
  }

  // 4. If saved in localStorage previously
  if (savedName) {
    senderShopName.value = savedName
    senderShopPhone.value = savedPhone || '020 5555 7777'
    senderShopAddress.value = savedAddress || 'ນະຄອນຫຼວງວຽງຈັນ (Vientiane)'
    return
  }

  // 5. Fallback to first shop in warehouses list, or activeAccount, or warehouse
  const firstShop = warehouses.value.find((w) => w.type === 'shop')
  senderShopName.value = firstShop?.name || activeAccount.value?.name || order?.warehouse?.name || 'ຮ້ານຄ້າ (My Shop)'
  senderShopPhone.value = firstShop?.phone || order?.warehouse?.phone || '020 5555 7777'
  senderShopAddress.value = firstShop?.address || order?.warehouse?.address || 'ນະຄອນຫຼວງວຽງຈັນ (Vientiane)'
}

function saveSenderAsDefault() {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('meta_shop_name', senderShopName.value.trim())
    localStorage.setItem('meta_shop_phone', senderShopPhone.value.trim())
    localStorage.setItem('meta_shop_address', senderShopAddress.value.trim())
  }
  senderSavedNotice.value = true
  setTimeout(() => {
    senderSavedNotice.value = false
  }, 2500)
}

function applyPageToSender(p: MetaPage) {
  senderShopName.value = p.name
}

function applyWarehouseToSender(w: Warehouse) {
  senderShopName.value = w.name
  if (w.phone) senderShopPhone.value = w.phone
  if (w.address) senderShopAddress.value = w.address
}

// Anousith Express Waybill Printing Modal
const waybillDialog = ref(false)
const waybillOrder = ref<Order | null>(null)
const waybillPrintSize = ref<'thermal' | 'a4'>('thermal')

function openWaybill(order: Order) {
  waybillOrder.value = order
  initSenderInfo(order)
  waybillDialog.value = true
}

function triggerPrint() {
  if (typeof window !== 'undefined') {
    window.print()
  }
}

function extractDestination(address?: string) {
  if (!address) return 'VTE / ປາຍທາງ'
  const provinces = [
    'ວຽງຈັນ', 'ນະຄອນຫຼວງ', 'ຫຼວງພະບາງ', 'ຈຳປາສັກ', 'ສະຫວັນນະເຂດ', 'ຄຳມ່ວນ',
    'ບໍລິຄຳໄຊ', 'ຊຽງຂວາງ', 'ຫົວພັນ', 'ອຸດົມໄຊ', 'ບໍ່ແກ້ວ', 'ຫຼວງນ້ຳທາ',
    'ຜົ້ງສາລີ', 'ໄຊຍະບູລີ', 'ສາລະວັນ', 'ເຊກອງ', 'ອັດຕະປື', 'ໄຊສົມບູນ'
  ]
  for (const prov of provinces) {
    if (address.includes(prov)) return prov
  }
  return address.split(',')[0] || 'VTE / ປາຍທາງ'
}

function generateBarcodePattern(text: string): { width: number; x: number }[] {
  const bars: { width: number; x: number }[] = []
  let currentX = 10
  bars.push({ width: 2, x: currentX }); currentX += 4
  bars.push({ width: 1, x: currentX }); currentX += 3
  bars.push({ width: 3, x: currentX }); currentX += 5

  for (let i = 0; i < text.length; i++) {
    const code = text.charCodeAt(i)
    const w1 = (code % 3) + 1
    const s1 = ((code >> 1) % 2) + 2
    const w2 = ((code >> 2) % 3) + 1
    const s2 = ((code >> 3) % 2) + 2
    bars.push({ width: w1, x: currentX }); currentX += w1 + s1
    bars.push({ width: w2, x: currentX }); currentX += w2 + s2
  }

  bars.push({ width: 3, x: currentX }); currentX += 5
  bars.push({ width: 1, x: currentX }); currentX += 3
  bars.push({ width: 2, x: currentX }); currentX += 4
  return bars
}

// Status Update Modal
const statusDialog = ref(false)
const updatingStatus = ref(false)
const statusForm = reactive<{
  orderId: string
  paymentStatus: 'unpaid' | 'paid' | 'refunded'
  fulfillmentStatus: 'pending' | 'confirmed' | 'shipping' | 'delivered' | 'cancelled'
  deliveryCourier: string
  deliveryTracking: string
  notes: string
}>({
  orderId: '',
  paymentStatus: 'paid',
  fulfillmentStatus: 'pending',
  deliveryCourier: '',
  deliveryTracking: '',
  notes: '',
})

// Couriers List
const couriers = [
  { value: 'Anousith Express', label: 'Anousith Express (ອານຸສິດ)' },
  { value: 'HAL Logistics', label: 'HAL Logistics (ຮຸ່ງອາລຸນ)' },
  { value: 'Houngaloun', label: 'Houngaloun (ຮຸ່ງອາລຸນ)' },
  { value: 'Flash Express', label: 'Flash Express' },
  { value: 'Kerry Express', label: 'Kerry Express' },
  { value: 'J&T Express', label: 'J&T Express' },
  { value: 'Lao Post', label: 'ໄປສະນີລາວ (Lao Post)' },
  { value: 'Store Pickup', label: 'ຮັບເອງທີ່ຮ້ານ (Store Pickup)' },
  { value: 'Other', label: 'ອື່ນໆ (Other)' },
]

const paymentMethods = [
  { value: 'Bank Transfer', label: 'ໂອນຜ່ານທະນາຄານ (Bank Transfer)' },
  { value: 'COD', label: 'ເກັບເງິນປາຍທາງ (Cash on Delivery)' },
  { value: 'Cash', label: 'ເງິນສົດ (Cash)' },
  { value: 'PromptPay', label: 'PromptPay (THB)' },
  { value: 'Other', label: 'ອື່ນໆ' },
]

// Calculated totals for order form
const formSubtotal = computed(() => {
  return orderForm.items.reduce((sum, item) => sum + item.quantity * item.unitPrice, 0)
})

const formTotal = computed(() => {
  const total = formSubtotal.value + Number(orderForm.shippingFee || 0) - Number(orderForm.discount || 0)
  return Math.max(0, total)
})

watch(() => orderForm.paymentMethod, (method) => {
  if (method === 'COD') {
    orderForm.paymentStatus = 'unpaid'
  } else if (method === 'Bank Transfer' || method === 'Cash' || method === 'PromptPay') {
    orderForm.paymentStatus = 'paid'
    orderForm.prepaidAmount = formTotal.value
  }
})

// Metrics
const stats = computed(() => {
  const list = orders.value
  const total = list.length
  const unpaid = list.filter((o) => o.paymentStatus === 'unpaid').length
  const inDelivery = list.filter((o) => o.fulfillmentStatus === 'shipping' || o.fulfillmentStatus === 'confirmed').length
  const delivered = list.filter((o) => o.fulfillmentStatus === 'delivered').length
  const totalRevenue = list
    .filter((o) => o.paymentStatus === 'paid')
    .reduce((sum, o) => sum + (o.totalAmount || 0), 0)
  const prepaidCount = list.filter((o) => !o.isCod && !o.paymentMethod?.toLowerCase().includes('cod')).length
  const codCount = list.filter((o) => o.isCod || o.paymentMethod?.toLowerCase().includes('cod')).length
  return { total, unpaid, inDelivery, delivered, totalRevenue, prepaidCount, codCount }
})

// Filtered Orders
const filteredOrders = computed(() => {
  return orders.value.filter((order) => {
    // Tab filter
    if (activeTab.value === 'prepaid' && (order.isCod || order.paymentMethod?.toLowerCase().includes('cod'))) return false
    if (activeTab.value === 'cod' && (!order.isCod && !order.paymentMethod?.toLowerCase().includes('cod'))) return false
    if (activeTab.value === 'unpaid' && order.paymentStatus !== 'unpaid') return false
    if (activeTab.value === 'paid' && order.paymentStatus !== 'paid') return false
    if (activeTab.value === 'shipping' && order.fulfillmentStatus !== 'shipping') return false
    if (activeTab.value === 'delivered' && order.fulfillmentStatus !== 'delivered') return false
    if (activeTab.value === 'cancelled' && order.fulfillmentStatus !== 'cancelled') return false

    // Warehouse filter
    if (selectedWarehouseFilter.value !== 'all' && order.warehouseId !== selectedWarehouseFilter.value) {
      return false
    }

    // Search query
    const q = searchQuery.value.trim().toLowerCase()
    if (!q) return true
    return (
      order.orderNumber.toLowerCase().includes(q) ||
      order.customerName.toLowerCase().includes(q) ||
      order.customerPhone.includes(q) ||
      (order.deliveryTracking && order.deliveryTracking.toLowerCase().includes(q))
    )
  })
})

function formatMoney(amount?: number | null, currency = 'LAK') {
  if (amount == null) return '0 ' + currency
  return new Intl.NumberFormat().format(amount) + ' ' + currency
}

function formatDate(iso?: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString('lo-LA', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function openCreateOrder() {
  const defaultWh = warehouses.value.find((w) => w.isDefault) || warehouses.value[0]
  orderForm.warehouseId = defaultWh ? defaultWh.id : ''
  orderForm.pageId = pages.value[0]?.id || ''
  orderForm.customerName = ''
  orderForm.customerPhone = ''
  orderForm.customerAddress = ''
  orderForm.shippingFee = 0
  orderForm.discount = 0
  orderForm.currency = 'LAK'
  orderForm.paymentStatus = 'paid'
  orderForm.paymentMethod = 'Bank Transfer'
  orderForm.prepaidAmount = 0
  orderForm.codAmount = 0
  orderForm.deliveryCourier = 'Anousith Express'
  orderForm.deliveryTracking = ''
  orderForm.notes = ''
  orderForm.items = []
  selectedProductToAdd.value = null
  createDialog.value = true
}

function addProductToForm() {
  if (!selectedProductToAdd.value) return
  const p = selectedProductToAdd.value
  const c = selectedColorToAdd.value.trim()
  const s = selectedSizeToAdd.value.trim()
  const existing = orderForm.items.find(
    (i) => i.productId === p.id && (i.color || '') === (c || '') && (i.size || '') === (s || '')
  )
  if (existing) {
    existing.quantity++
  } else {
    orderForm.items.push({
      productId: p.id,
      productCode: p.code || '',
      productName: p.name,
      productImage: p.imageUrl || '',
      color: c || undefined,
      size: s || undefined,
      quantity: 1,
      unitPrice: p.salePrice ?? (parseFloat(p.price) || 0),
    })
  }
  selectedProductToAdd.value = null
  selectedColorToAdd.value = ''
  selectedSizeToAdd.value = ''
}

function removeFormItem(index: number) {
  orderForm.items.splice(index, 1)
}

async function submitCreateOrder() {
  if (!orderForm.customerName.trim() || !orderForm.customerPhone.trim()) {
    noticeType.value = 'error'
    notice.value = l({
      lo: 'ກະລຸນາປ້ອນຊື່ ແລະ ເບີໂທລູກຄ້າ',
      th: 'โปรดกรอกชื่อและเบอร์โทรศัพท์ลูกค้า',
      en: 'Please enter customer name and phone.',
    })
    return
  }
  if (!orderForm.warehouseId) {
    noticeType.value = 'error'
    notice.value = l({
      lo: 'ກະລຸນາເລືອກສາງ ຫຼື ໜ້າຮ້ານ',
      th: 'โปรดเลือกคลังหรือหน้าร้าน',
      en: 'Please select warehouse or shop.',
    })
    return
  }
  if (orderForm.items.length === 0) {
    noticeType.value = 'error'
    notice.value = l({
      lo: 'ກະລຸນາເພີ່ມສິນຄ້າຢ່າງໜ້ອຍ 1 ລາຍການ',
      th: 'โปรดเพิ่มสินค้าอย่างน้อย 1 รายการ',
      en: 'Please add at least 1 product item.',
    })
    return
  }

  creating.value = true
  notice.value = ''
  try {
    const payload = {
      warehouseId: orderForm.warehouseId,
      pageId: orderForm.pageId || undefined,
      customerName: orderForm.customerName.trim(),
      customerPhone: orderForm.customerPhone.trim(),
      customerAddress: orderForm.customerAddress.trim(),
      subtotal: formSubtotal.value,
      shippingFee: Number(orderForm.shippingFee || 0),
      discount: Number(orderForm.discount || 0),
      totalAmount: formTotal.value,
      currency: orderForm.currency,
      paymentStatus: orderForm.paymentStatus,
      paymentMethod: orderForm.paymentMethod,
      isCod: orderForm.paymentMethod === 'COD',
      prepaidAmount: orderForm.paymentMethod === 'COD' ? Number(orderForm.prepaidAmount || 0) : formTotal.value,
      codAmount: orderForm.paymentMethod === 'COD' ? Math.max(0, formTotal.value - Number(orderForm.prepaidAmount || 0)) : 0,
      fulfillmentStatus: orderForm.paymentStatus === 'paid' ? 'confirmed' : 'pending',
      deliveryCourier: orderForm.deliveryCourier,
      deliveryTracking: orderForm.deliveryTracking.trim(),
      notes: orderForm.notes.trim(),
      items: orderForm.items.map((it) => ({
        productId: it.productId,
        productCode: it.productCode,
        productName: it.productName,
        productImage: it.productImage,
        color: it.color || '',
        size: it.size || '',
        quantity: it.quantity,
        unitPrice: it.unitPrice,
      })),
    }

    const created = await $fetch<Order>('/api/proxy/api/v1/orders', {
      method: 'POST',
      body: payload,
    })

    createDialog.value = false
    noticeType.value = 'success'
    notice.value = l({
      lo: `ສ້າງອໍເດີ #${created.orderNumber} ສຳເລັດແລ້ວ! ຕັດສະຕ໋ອກ ແລະ ສົ່ງ Meta CAPI ຮຽບຮ້ອຍ`,
      th: `สร้างออเดอร์ #${created.orderNumber} สำเร็จแล้ว! ตัดสต็อกและส่ง Meta CAPI เรียบร้อย`,
      en: `Order #${created.orderNumber} created! Stock deducted and Meta CAPI Purchase sent.`,
    })
    await refresh()
  } catch (err: any) {
    noticeType.value = 'error'
    notice.value = err?.data?.error || err?.message || l({
      lo: 'ສ້າງອໍເດີບໍ່ສຳເລັດ ກະລຸນາກວດສອບສະຕ໋ອກສິນຄ້າ',
      th: 'สร้างออเดอร์ไม่สำเร็จ โปรดตรวจสอบสต็อกสินค้า',
      en: 'Failed to create order. Please check stock.',
    })
  } finally {
    creating.value = false
  }
}

function openReceipt(order: Order) {
  selectedOrder.value = order
  initSenderInfo(order)
  receiptDialog.value = true
  copiedReceipt.value = false
}

function copyReceiptText() {
  if (!selectedOrder.value) return
  const o = selectedOrder.value
  initSenderInfo(o)
  const itemsText = o.items
    ?.map((i) => {
      const variantStr = [i.color ? `ສີ: ${i.color}` : '', i.size ? `ເບີ: ${i.size}` : ''].filter(Boolean).join(', ')
      return `• ${i.productName}${variantStr ? ` (${variantStr})` : ''} x${i.quantity} = ${formatMoney(i.totalPrice, o.currency)}`
    })
    .join('\n')

  const text = `
==============================
🧾 ໃບບິນຄຳສັ່ງຊື້ / ORDER RECEIPT
🏪 ຮ້ານ: ${senderShopName.value}
📞 ເບີໂທຮ້ານ: ${senderShopPhone.value}
${senderShopAddress.value ? `📍 ທີ່ຢູ່ຮ້ານ: ${senderShopAddress.value}\n` : ''}------------------------------
ເລກທີ: ${o.orderNumber}
ວັນທີ: ${formatDate(o.createdAt)}
------------------------------
👤 ລູກຄ້າ: ${o.customerName}
📞 ເບີໂທ: ${o.customerPhone}
📍 ທີ່ຢູ່: ${o.customerAddress || '—'}
------------------------------
📦 ລາຍການສິນຄ້າ:
${itemsText}
------------------------------
ລວມຄ່າສິນຄ້າ: ${formatMoney(o.subtotal, o.currency)}
ຄ່າຈັດສົ່ງ: ${formatMoney(o.shippingFee, o.currency)}
ສ່ວນຫຼຸດ: ${formatMoney(o.discount, o.currency)}
💰 ຍອດຊຳລະທັງໝົດ: ${formatMoney(o.totalAmount, o.currency)}
------------------------------
💳 ສະຖານະຊຳລະ: ${o.paymentStatus === 'paid' ? 'ຊຳລະແລ້ວ (PAID)' : 'ຍັງບໍ່ຊຳລະ'} (${o.paymentMethod || '—'})
🚚 ຂົນສົ່ງ: ${o.deliveryCourier || '—'} ${o.deliveryTracking ? `(Track: ${o.deliveryTracking})` : ''}
${o.notes ? `ໝາຍເຫດ: ${o.notes}\n` : ''}==============================
ຂອບໃຈທີ່ໃຊ້ບໍລິການ! 🙏
`.trim()

  navigator.clipboard.writeText(text)
  copiedReceipt.value = true
  setTimeout(() => {
    copiedReceipt.value = false
  }, 2500)
}

function openStatusModal(order: Order) {
  statusForm.orderId = order.id
  statusForm.paymentStatus = order.paymentStatus
  statusForm.fulfillmentStatus = order.fulfillmentStatus
  statusForm.deliveryCourier = order.deliveryCourier || 'Anousith Express'
  statusForm.deliveryTracking = order.deliveryTracking || ''
  statusForm.notes = order.notes || ''
  statusDialog.value = true
}

async function saveStatusUpdate() {
  if (!statusForm.orderId) return
  updatingStatus.value = true
  notice.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/orders/${statusForm.orderId}/status`, {
      method: 'PATCH',
      body: {
        paymentStatus: statusForm.paymentStatus,
        fulfillmentStatus: statusForm.fulfillmentStatus,
        deliveryCourier: statusForm.deliveryCourier,
        deliveryTracking: statusForm.deliveryTracking.trim(),
        notes: statusForm.notes.trim(),
      },
    })
    statusDialog.value = false
    noticeType.value = 'success'
    notice.value = l({
      lo: 'ອັບເດດສະຖານະອໍເດີສຳເລັດແລ້ວ',
      th: 'อัปเดตสถานะออเดอร์เรียบร้อยแล้ว',
      en: 'Order status updated successfully.',
    })
    await refresh()
  } catch (err: any) {
    noticeType.value = 'error'
    notice.value = err?.data?.error || err?.message || l({
      lo: 'ອັບເດດບໍ່ສຳເລັດ',
      th: 'อัปเดตไม่สำเร็จ',
      en: 'Failed to update order status.',
    })
  } finally {
    updatingStatus.value = false
  }
}
</script>

<template>
  <div class="orders-page">
    <section class="app-page-head">
      <div class="head-content">
        <span class="head-badge">
          <v-icon icon="mdi-clipboard-list-outline" size="14" />
          Commerce & Fulfillment
        </span>
        <h1>{{ l({ lo: 'ຈັດການຄຳສັ່ງຊື້ (Orders)', th: 'จัดการคำสั่งซื้อ (Orders)', en: 'Orders & Fulfillment' }) }}</h1>
        <p>{{ l({ lo: 'ຕິດຕາມອໍເດີ, ຕັດສະຕ໋ອກອັດຕະໂນມັດ, ຈັດສົ່ງສິນຄ້າ ແລະ ສົ່ງ Lead ໄປຍັງ Facebook Meta CAPI', th: 'ติดตามออเดอร์ ตัดสต็อกอัตโนมัติ ขนส่งสินค้า และส่ง Lead ไปยัง Facebook Meta CAPI', en: 'Track orders, auto-deduct warehouse stock, manage delivery and dispatch Meta CAPI Purchase events.' }) }}</p>
      </div>
      <div class="head-actions">
        <v-btn variant="outlined" prepend-icon="mdi-warehouse" to="/warehouses">
          {{ l({ lo: 'ສາງ ແລະ ໜ້າຮ້ານ', th: 'คลังและหน้าร้าน', en: 'Warehouses' }) }}
        </v-btn>
        <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" @click="openCreateOrder">
          {{ l({ lo: 'ສ້າງອໍເດີໃໝ່', th: 'สร้างออเดอร์ใหม่', en: 'New Order' }) }}
        </v-btn>
      </div>
    </section>

    <!-- Metrics Stats Cards -->
    <div class="app-kpi-grid mb-4">
      <div class="app-kpi-card accent-primary">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ອໍເດີທັງໝົດ', th: 'ออเดอร์ทั้งหมด', en: 'Total Orders' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-cart-check" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ stats.total }}</strong>
        <div class="kpi-sub">
          <span>ໂອນ: {{ stats.prepaidCount }} | COD: {{ stats.codCount }}</span>
        </div>
      </div>

      <div class="app-kpi-card accent-warning">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ລໍຖ້າຊຳລະເງິນ', th: 'รอชำระเงิน', en: 'Unpaid' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-clock-outline" size="16" /></span>
        </div>
        <strong class="kpi-value text-warning">{{ stats.unpaid }}</strong>
        <div class="kpi-sub">
          <span>ລໍຖ້າລູກຄ້າໂອນ/ຢືນຢັນ</span>
        </div>
      </div>

      <div class="app-kpi-card accent-info">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ກຳລັງຈັດສົ່ງ', th: 'กำลังจัดส่ง', en: 'In Delivery' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-truck-fast-outline" size="16" /></span>
        </div>
        <strong class="kpi-value text-info">{{ stats.inDelivery }}</strong>
        <div class="kpi-sub">
          <span>ກຳລັງເດີນທາງຫາລູກຄ້າ</span>
        </div>
      </div>

      <div class="app-kpi-card accent-success">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ຍອດຂາຍທີ່ຮັບເງິນແລ້ວ', th: 'ยอดขายที่รับเงินแล้ว', en: 'Paid Sales' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-cash-multiple" size="16" /></span>
        </div>
        <strong class="kpi-value text-success">{{ formatMoney(stats.totalRevenue) }}</strong>
        <div class="kpi-sub">
          <span>ສຳເລັດແລ້ວ: {{ stats.delivered }} ອໍເດີ</span>
        </div>
      </div>
    </div>

    <!-- Alert Notices -->
    <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">
      {{ notice }}
    </v-alert>
    <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
      {{ l({ lo: 'ໂຫຼດຂໍ້ມູນອໍເດີບໍ່ສຳເລັດ', th: 'โหลดข้อมูลออเดอร์ไม่สำเร็จ', en: 'Orders could not be loaded.' }) }}
      <button class="inline-action" @click="refresh()">{{ l({ lo: 'ລອງໃໝ່', th: 'ลองอีกครั้ง', en: 'Try again' }) }}</button>
    </v-alert>

    <!-- Toolbar Filters -->
    <div class="orders-toolbar mb-4">
      <div class="filter-tabs">
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'all' }]"
          @click="activeTab = 'all'"
        >
          {{ l({ lo: 'ທັງໝົດ', th: 'ทั้งหมด', en: 'All' }) }} ({{ orders.length }})
        </button>
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'prepaid' }]"
          @click="activeTab = 'prepaid'"
        >
          💳 {{ l({ lo: 'ໂອນກ່ອນ', th: 'โอนก่อน', en: 'Prepaid' }) }} ({{ stats.prepaidCount }})
        </button>
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'cod' }]"
          @click="activeTab = 'cod'"
        >
          🚚 {{ l({ lo: 'ເກັບປາຍທາງ (COD)', th: 'เก็บปลายทาง (COD)', en: 'COD' }) }} ({{ stats.codCount }})
        </button>
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'unpaid' }]"
          @click="activeTab = 'unpaid'"
        >
          {{ l({ lo: 'ລໍຖ້າຈ່າຍ', th: 'รอจ่าย', en: 'Unpaid' }) }} ({{ stats.unpaid }})
        </button>
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'paid' }]"
          @click="activeTab = 'paid'"
        >
          {{ l({ lo: 'ຈ່າຍແລ້ວ', th: 'จ่ายแล้ว', en: 'Paid' }) }}
        </button>
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'shipping' }]"
          @click="activeTab = 'shipping'"
        >
          {{ l({ lo: 'ກຳລັງສົ່ງ', th: 'กำลังส่ง', en: 'Shipping' }) }} ({{ stats.inDelivery }})
        </button>
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'delivered' }]"
          @click="activeTab = 'delivered'"
        >
          {{ l({ lo: 'ສຳເລັດ', th: 'สำเร็จ', en: 'Delivered' }) }}
        </button>
        <button
          type="button"
          :class="['tab-pill', { active: activeTab === 'cancelled' }]"
          @click="activeTab = 'cancelled'"
        >
          {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancelled' }) }}
        </button>
      </div>

      <div class="search-filters">
        <select v-model="selectedWarehouseFilter" class="warehouse-select">
          <option value="all">{{ l({ lo: 'ທຸກສາງ / ໜ້າຮ້ານ', th: 'ทุกคลัง / หน้าร้าน', en: 'All Locations' }) }}</option>
          <option v-for="w in warehouses" :key="w.id" :value="w.id">
            {{ w.name }} ({{ w.code }})
          </option>
        </select>

        <div class="search-box">
          <v-icon icon="mdi-magnify" size="18" class="search-icon"/>
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="l({ lo: 'ຄົ້ນຫາເລກອໍເດີ, ຊື່, ເບີໂທ, tracking...', th: 'ค้นหารหัสออเดอร์, ชื่อ, เบอร์โทร, tracking...', en: 'Search order #, customer, phone, tracking...' })"
          />
          <button v-if="searchQuery" class="clear-btn" @click="searchQuery = ''">
            <v-icon icon="mdi-close" size="14"/>
          </button>
        </div>
      </div>
    </div>

    <!-- Orders Table / List -->
    <section v-if="pending" class="order-list">
      <v-skeleton-loader v-for="i in 4" :key="i" type="list-item-two-line"/>
    </section>

    <section v-else-if="filteredOrders.length" class="order-cards-container">
      <div v-for="order in filteredOrders" :key="order.id" class="order-row-card">
        <div class="order-header-row">
          <div class="order-id-group">
            <strong class="order-number">#{{ order.orderNumber }}</strong>
            <span class="order-date">{{ formatDate(order.createdAt) }}</span>
            <span v-if="order.facebookLeadSent" class="badge-capi" title="Meta Conversions API & Lead synced">
              <v-icon icon="mdi-facebook" size="13" color="#1877f2" class="mr-1"/>
              Meta CAPI
            </span>
            <span v-if="order.pageId && getPageName(order.pageId)" class="badge-page-tag" :title="`Page: ${getPageName(order.pageId)}`">
              <v-icon icon="mdi-facebook" size="13" color="#1877f2" class="mr-1"/>
              {{ getPageName(order.pageId) }}
            </span>
            <span v-if="order.warehouse" class="badge-wh-tag">
              <v-icon :icon="order.warehouse.type === 'shop' ? 'mdi-storefront' : 'mdi-warehouse'" size="12" class="mr-1"/>
              {{ order.warehouse.name }}
            </span>
          </div>

          <div class="order-status-group">
            <span
              class="badge-status"
              :class="{
                'status-paid': order.paymentStatus === 'paid',
                'status-unpaid': order.paymentStatus === 'unpaid',
                'status-refunded': order.paymentStatus === 'refunded',
              }"
            >
              {{ order.paymentStatus === 'paid' ? l({ lo: 'ຊຳລະແລ້ວ', th: 'ชำระแล้ว', en: 'Paid' }) : order.paymentStatus === 'unpaid' ? l({ lo: 'ຍັງບໍ່ຊຳລະ', th: 'ยังไม่ชำระ', en: 'Unpaid' }) : l({ lo: 'ຄືນເງິນແລ້ວ', th: 'คืนเงินแล้ว', en: 'Refunded' }) }}
            </span>

            <span
              class="badge-fulfillment"
              :class="{
                'fulfill-pending': order.fulfillmentStatus === 'pending',
                'fulfill-shipping': order.fulfillmentStatus === 'shipping' || order.fulfillmentStatus === 'confirmed',
                'fulfill-delivered': order.fulfillmentStatus === 'delivered',
                'fulfill-cancelled': order.fulfillmentStatus === 'cancelled',
              }"
            >
              {{
                order.fulfillmentStatus === 'delivered'
                  ? l({ lo: 'ຈັດສົ່ງສຳເລັດ', th: 'จัดส่งสำเร็จ', en: 'Delivered' })
                  : order.fulfillmentStatus === 'shipping'
                  ? l({ lo: 'ກຳລັງຈັດສົ່ງ', th: 'กำลังจัดส่ง', en: 'Shipping' })
                  : order.fulfillmentStatus === 'cancelled'
                  ? l({ lo: 'ຍົກເລີກແລ້ວ', th: 'ยกเลิกแล้ว', en: 'Cancelled' })
                  : l({ lo: 'ລໍຖ້າຈັດສົ່ງ', th: 'รอจัดส่ง', en: 'Pending' })
              }}
            </span>
          </div>
        </div>

        <div class="order-content-grid">
          <!-- Customer info -->
          <div class="order-col customer-col">
            <div class="customer-name-row">
              <v-icon icon="mdi-account-outline" size="16" class="mr-1 text-muted"/>
              <strong>{{ order.customerName }}</strong>
            </div>
            <div class="customer-contact-row">
              <v-icon icon="mdi-phone-outline" size="14" class="mr-1 text-muted"/>
              <a :href="`tel:${order.customerPhone}`">{{ order.customerPhone }}</a>
            </div>
            <div v-if="order.customerAddress" class="customer-address-row">
              <v-icon icon="mdi-map-marker-outline" size="14" class="mr-1 text-muted"/>
              <span>{{ order.customerAddress }}</span>
            </div>
          </div>

          <!-- Items snippet -->
          <div class="order-col items-col">
            <div class="items-summary-list">
              <div v-for="item in (order.items || []).slice(0, 3)" :key="item.id" class="item-summary-line">
                <span class="item-qty">{{ item.quantity }}x</span>
                <span class="item-name">
                  {{ item.productName }}
                  <small v-if="item.color || item.size" class="text-primary font-weight-bold">
                    ({{ [item.color ? `ສີ:${item.color}` : '', item.size ? `ເບີ:${item.size}` : ''].filter(Boolean).join(' ') }})
                  </small>
                </span>
                <span class="item-price">{{ formatMoney(item.totalPrice, order.currency) }}</span>
              </div>
              <small v-if="(order.items || []).length > 3" class="text-muted">
                + ອີກ {{ (order.items || []).length - 3 }} ລາຍການ...
              </small>
            </div>
          </div>

          <!-- Courier & Tracking -->
          <div class="order-col delivery-col">
            <div class="courier-badge">
              <v-icon icon="mdi-truck-outline" size="15" class="mr-1"/>
              <span>{{ order.deliveryCourier || '—' }}</span>
            </div>
            <div v-if="order.deliveryTracking" class="tracking-pill">
              <small>{{ l({ lo: 'Tracking:', th: 'Tracking:', en: 'Track:' }) }}</small>
              <strong>{{ order.deliveryTracking }}</strong>
            </div>
          </div>

          <!-- Total & Actions -->
          <div class="order-col total-col">
            <div class="total-display">
              <small>{{ l({ lo: 'ຍອດລວມ', th: 'ยอดรวม', en: 'Total' }) }}</small>
              <strong>{{ formatMoney(order.totalAmount, order.currency) }}</strong>
            </div>
            <div class="action-buttons">
              <button type="button" class="action-btn waybill-btn" @click="openWaybill(order)" :title="l({ lo: 'ພິມບິນ ອານຸສິດ (Waybill)', th: 'พิมพ์บิล อานุสิต (Waybill)', en: 'Print Anousith Bill' })">
                <v-icon icon="mdi-printer" size="15" class="mr-1"/>
                {{ l({ lo: 'ພິມບິນອານຸສິດ', th: 'พิมพ์บิลอานุสิต', en: 'Print Bill' }) }}
              </button>
              <button type="button" class="action-btn receipt-btn" @click="openReceipt(order)" :title="l({ lo: 'ເບິ່ງໃບບິນ', th: 'ดูใบเสร็จ', en: 'View Receipt' })">
                <v-icon icon="mdi-receipt-text-outline" size="15" class="mr-1"/>
                {{ l({ lo: 'ໃບບິນ', th: 'ใบเสร็จ', en: 'Receipt' }) }}
              </button>
              <button type="button" class="action-btn status-btn" @click="openStatusModal(order)" :title="l({ lo: 'ປ່ຽນສະຖານະ', th: 'เปลี่ยนสถานะ', en: 'Change Status' })">
                <v-icon icon="mdi-cog-outline" size="15" class="mr-1"/>
                {{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section v-else class="empty-state">
      <div class="empty-icon-circle">
        <v-icon icon="mdi-cart-plus" size="32" color="primary"/>
      </div>
      <h2>{{ l({ lo: 'ຍັງບໍ່ມີລາຍການຄຳສັ່ງຊື້', th: 'ยังไม่มีรายการคำสั่งซื้อ', en: 'No orders yet' }) }}</h2>
      <p>{{ l({ lo: 'ສ້າງອໍເດີໃໝ່ເພື່ອຕັດສະຕ໋ອກສິນຄ້າອັດຕະໂນມັດ ແລະ ສົ່ງ Event Purchase ຫາ Meta CAPI', th: 'สร้างออเดอร์ใหม่เพื่อตัดสต็อกสินค้าอัตโนมัติและส่ง Purchase event ไปยัง Meta CAPI', en: 'Create orders to trigger automatic stock deductions and Meta Conversions API purchase tracking.' }) }}</p>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreateOrder">
        {{ l({ lo: 'ສ້າງອໍເດີໃໝ່', th: 'สร้างออเดอร์ใหม่', en: 'New Order' }) }}
      </v-btn>
    </section>

    <!-- Create Order Dialog -->
    <v-dialog v-model="createDialog" max-width="680" scrollable>
      <v-card class="dialog-card">
        <v-card-title class="dialog-header">
          <v-icon icon="mdi-cart-plus" class="mr-2" color="primary"/>
          {{ l({ lo: 'ສ້າງອໍເດີໃໝ່', th: 'สร้างออเดอร์ใหม่', en: 'Create New Order' }) }}
        </v-card-title>
        <v-card-text class="dialog-body">
          <!-- Step 1: Location & Customer -->
          <div class="form-section-title">
            <v-icon icon="mdi-map-marker-radius" size="16" class="mr-1"/>
            {{ l({ lo: '1. ສະຖານທີ່ ແລະ ຂໍ້ມູນລູກຄ້າ', th: '1. สถานที่และข้อมูลลูกค้า', en: '1. Location & Customer' }) }}
          </div>

          <v-select
            v-model="orderForm.warehouseId"
            :items="warehouses"
            item-title="name"
            item-value="id"
            :label="l({ lo: 'ຕັດສະຕ໋ອກຈາກສາງ / ໜ້າຮ້ານ *', th: 'ตัดสต็อกสินค้าจากคลัง / หน้าร้าน *', en: 'Deduct stock from warehouse/shop *' })"
            class="mb-3"
            density="compact"
            variant="outlined"
          >
            <template #item="{ props, item }">
              <v-list-item v-bind="props" :subtitle="`Code: ${(item as any).code || (item as any).raw?.code} · ${((item as any).type || (item as any).raw?.type) === 'shop' ? 'Shop' : 'Warehouse'}`"/>
            </template>
          </v-select>

          <v-select
            v-if="pages.length > 0"
            v-model="orderForm.pageId"
            :items="pages"
            item-title="name"
            item-value="id"
            :label="l({ lo: '📘 Facebook Page ທີ່ຮັບອໍເດີ (ຊື່ຮ້ານໃນບິນ)', th: '📘 Facebook Page ที่รับออเดอร์ (ชื่อร้านในบิล)', en: '📘 Order Facebook Page (Shop on Bill)' })"
            density="compact"
            variant="outlined"
            clearable
            class="mb-3"
          >
            <template #prepend-inner>
              <v-icon icon="mdi-facebook" color="#1877f2" size="20"/>
            </template>
          </v-select>

          <div class="fields-row-2 mb-2">
            <v-text-field
              v-model="orderForm.customerName"
              :label="l({ lo: 'ຊື່ລູກຄ້າ *', th: 'ชื่อลูกค้า *', en: 'Customer Name *' })"
              placeholder="ທ້າວ ສົມສັກ..."
              density="compact"
              variant="outlined"
              required
            />
            <v-text-field
              v-model="orderForm.customerPhone"
              :label="l({ lo: 'ເບີໂທລະສັບ *', th: 'เบอร์โทรศัพท์ *', en: 'Phone *' })"
              placeholder="020 xxxxxxxx"
              density="compact"
              variant="outlined"
              required
            />
          </div>

          <v-textarea
            v-model="orderForm.customerAddress"
            :label="l({ lo: 'ທີ່ຢູ່ຈັດສົ່ງສິນຄ້າ', th: 'ที่อยู่จัดส่งสินค้า', en: 'Delivery Address' })"
            placeholder="ບ້ານ, ເມືອງ, ແຂວງ (ຫຼື ຈຸດສົ່ງ Anousith/HAL)"
            rows="2"
            density="compact"
            variant="outlined"
            class="mb-3"
          />

          <!-- Step 2: Line Items -->
          <div class="form-section-title">
            <v-icon icon="mdi-package-variant" size="16" class="mr-1"/>
            {{ l({ lo: '2. ເລືອກສິນຄ້າທີ່ສັ່ງຊື້', th: '2. เลือกสินค้าที่สั่งซื้อ', en: '2. Order Line Items' }) }}
          </div>

          <div class="product-picker-row mb-2">
            <v-select
              v-model="selectedProductToAdd"
              :items="products"
              item-title="name"
              return-object
              :label="l({ lo: 'ຄົ້ນຫາ ແລະ ເລືອກສິນຄ້າ', th: 'ค้นหาและเลือกสินค้า', en: 'Select product to add' })"
              placeholder="ເລືອກສິນຄ້າ..."
              density="compact"
              variant="outlined"
              hide-details
              class="flex-1"
            >
              <template #item="{ props, item }">
                <v-list-item
                  v-bind="props"
                  :subtitle="`ລາຄາ: ${formatMoney((item as any).salePrice ?? parseFloat((item as any).price), (item as any).currency || 'LAK')} · ${(item as any).colors ? `ສີ: ${(item as any).colors} · ` : ''}${(item as any).sizes ? `ເບີ: ${(item as any).sizes} · ` : ''}ສະຕ໋ອກ: ${(item as any).quantity ?? '—'}`"
                />
              </template>
            </v-select>
          </div>

          <!-- Color and Size Picker for Selected Product -->
          <div v-if="selectedProductToAdd && (availableColorsForSelected.length || availableSizesForSelected.length)" class="fields-row-2 mb-2 p-2 variant-select-box">
            <v-select
              v-if="availableColorsForSelected.length"
              v-model="selectedColorToAdd"
              :items="availableColorsForSelected"
              :label="l({ lo: 'ເລືອກສີ (Color)', th: 'เลือกสี (Color)', en: 'Select Color' })"
              density="compact"
              variant="outlined"
              hide-details
            />
            <v-select
              v-if="availableSizesForSelected.length"
              v-model="selectedSizeToAdd"
              :items="availableSizesForSelected"
              :label="l({ lo: 'ເລືອກເບີ / ໄຊສ໌ (Size)', th: 'เลือกเบอร์/ไซส์ (Size)', en: 'Select Size' })"
              density="compact"
              variant="outlined"
              hide-details
            />
          </div>

          <div class="d-flex justify-end mb-3">
            <v-btn
              color="primary"
              variant="tonal"
              :disabled="!selectedProductToAdd"
              @click="addProductToForm"
            >
              + {{ l({ lo: 'ເພີ່ມເຂົ້າອໍເດີ', th: 'เพิ่มเข้าออเดอร์', en: 'Add to Order' }) }}
            </v-btn>
          </div>

          <!-- Items Table -->
          <div v-if="orderForm.items.length" class="order-items-table mb-3">
            <div v-for="(it, idx) in orderForm.items" :key="`${it.productId}-${it.color || ''}-${it.size || ''}-${idx}`" class="order-item-row">
              <div class="item-pic" :style="it.productImage ? `background-image: url(${it.productImage})` : ''">
                <v-icon v-if="!it.productImage" icon="mdi-package-variant-closed" size="16"/>
              </div>
              <div class="item-info">
                <strong>{{ it.productName }}</strong>
                <div class="d-flex gap-1 flex-wrap align-center mt-1">
                  <small v-if="it.productCode" class="text-muted">#{{ it.productCode }}</small>
                  <span v-if="it.color" class="variant-tag-pill tag-color">
                    {{ l({ lo: 'ສີ', th: 'สี', en: 'Color' }) }}: {{ it.color }}
                  </span>
                  <span v-if="it.size" class="variant-tag-pill tag-size">
                    {{ l({ lo: 'ເບີ', th: 'เบอร์', en: 'Size' }) }}: {{ it.size }}
                  </span>
                </div>
              </div>
              <div class="item-qty-input">
                <button type="button" class="qty-btn" @click="it.quantity > 1 ? it.quantity-- : removeFormItem(idx)">-</button>
                <input v-model.number="it.quantity" type="number" min="1" class="qty-num"/>
                <button type="button" class="qty-btn" @click="it.quantity++">+</button>
              </div>
              <div class="item-unit-price">
                <input v-model.number="it.unitPrice" type="number" class="price-input" min="0"/>
                <small>{{ orderForm.currency }}</small>
              </div>
              <div class="item-total">
                <strong>{{ formatMoney(it.quantity * it.unitPrice, orderForm.currency) }}</strong>
              </div>
              <button type="button" class="del-btn" @click="removeFormItem(idx)" title="Remove">
                <v-icon icon="mdi-close" size="16"/>
              </button>
            </div>
          </div>

          <!-- Step 3: Payment, Delivery & Calculation -->
          <div class="form-section-title">
            <v-icon icon="mdi-calculator" size="16" class="mr-1"/>
            {{ l({ lo: '3. ຄ່າຈັດສົ່ງ, ສ່ວນຫຼຸດ ແລະ ການຊຳລະ', th: '3. ค่าจัดส่ง ส่วนลด และการชำระเงิน', en: '3. Delivery, Discount & Payment' }) }}
          </div>

          <div class="fields-row-3 mb-2">
            <v-text-field
              v-model.number="orderForm.shippingFee"
              type="number"
              min="0"
              :label="l({ lo: 'ຄ່າຈັດສົ່ງ', th: 'ค่าจัดส่ง', en: 'Shipping Fee' })"
              density="compact"
              variant="outlined"
            />
            <v-text-field
              v-model.number="orderForm.discount"
              type="number"
              min="0"
              :label="l({ lo: 'ສ່ວນຫຼຸດ', th: 'ส่วนลด', en: 'Discount' })"
              density="compact"
              variant="outlined"
            />
            <v-text-field
              v-model="orderForm.currency"
              :label="l({ lo: 'ສະກຸນເງິນ', th: 'สกุลเงิน', en: 'Currency' })"
              density="compact"
              variant="outlined"
              maxlength="6"
            />
          </div>

          <div class="fields-row-2 mb-2">
            <v-select
              v-model="orderForm.deliveryCourier"
              :items="couriers"
              item-title="label"
              item-value="value"
              :label="l({ lo: 'ບໍລິສັດຂົນສົ່ງ', th: 'บริษัทขนส่ง', en: 'Courier' })"
              density="compact"
              variant="outlined"
            />
            <v-text-field
              v-model="orderForm.deliveryTracking"
              :label="l({ lo: 'ເລກຕິດຕາມ (Tracking)', th: 'เลขพัสดุ (Tracking)', en: 'Tracking #' })"
              placeholder="AN123456LA"
              density="compact"
              variant="outlined"
            />
          </div>

          <div class="fields-row-2 mb-2">
            <v-select
              v-model="orderForm.paymentStatus"
              :items="[
                { value: 'paid', label: l({ lo: 'ຊຳລະເງິນແລ້ວ', th: 'ชำระเงินแล้ว', en: 'Paid' }) },
                { value: 'unpaid', label: l({ lo: 'ຍັງບໍ່ຊຳລະ (ລໍຖ້າ)', th: 'ยังไม่ชำระ (รอชำระ)', en: 'Unpaid' }) },
              ]"
              item-title="label"
              item-value="value"
              :label="l({ lo: 'ສະຖານະຊຳລະເງິນ', th: 'สถานะการชำระ', en: 'Payment Status' })"
              density="compact"
              variant="outlined"
            />
            <v-select
              v-model="orderForm.paymentMethod"
              :items="paymentMethods"
              item-title="label"
              item-value="value"
              :label="l({ lo: 'ວິທີຊຳລະເງິນ', th: 'วิธีชำระเงิน', en: 'Payment Method' })"
              density="compact"
              variant="outlined"
            />
          </div>

          <!-- COD vs Prepaid Amount Breakdown -->
          <div v-if="orderForm.paymentMethod === 'COD'" class="fields-2col mb-2">
            <v-text-field
              v-model.number="orderForm.prepaidAmount"
              type="number"
              min="0"
              :label="l({ lo: 'ລູກຄ້າໂອນມາກ່ອນ (ມັດຈຳ/ຄ່າສົ່ງ)', th: 'ลูกค้าโอนมาก่อน (มัดจำ/ค่าส่ง)', en: 'Prepaid amount' })"
              :suffix="orderForm.currency"
              density="compact"
              variant="outlined"
              hide-details
            />
            <v-text-field
              :model-value="formatMoney(Math.max(0, formTotal - (orderForm.prepaidAmount || 0)), orderForm.currency)"
              readonly
              :label="l({ lo: 'ຍອດເກັບປາຍທາງຕົວຈິງ (COD)', th: 'ยอดเก็บปลายทางจริง (COD)', en: 'Actual COD Amount' })"
              density="compact"
              variant="outlined"
              hide-details
            />
          </div>

          <v-textarea
            v-model="orderForm.notes"
            :label="l({ lo: 'ໝາຍເຫດເພີ່ມເຕີມ', th: 'หมายเหตุเพิ่มเติม', en: 'Notes' })"
            placeholder="ລາຍລະອຽດ ຫຼື ຂໍ້ຄວາມພິເສດ..."
            rows="2"
            density="compact"
            variant="outlined"
            class="mb-3"
          />

          <!-- Summary Box -->
          <div class="order-summary-box">
            <div class="summary-line">
              <span>{{ l({ lo: 'ມູນຄ່າສິນຄ້າ (Subtotal):', th: 'มูลค่าสินค้า (Subtotal):', en: 'Subtotal:' }) }}</span>
              <span>{{ formatMoney(formSubtotal, orderForm.currency) }}</span>
            </div>
            <div class="summary-line">
              <span>{{ l({ lo: 'ຄ່າຈັດສົ່ງ (Shipping):', th: 'ค่าจัดส่ง (Shipping):', en: 'Shipping:' }) }}</span>
              <span>+ {{ formatMoney(orderForm.shippingFee, orderForm.currency) }}</span>
            </div>
            <div class="summary-line text-red">
              <span>{{ l({ lo: 'ສ່ວນຫຼຸດ (Discount):', th: 'ส่วนลด (Discount):', en: 'Discount:' }) }}</span>
              <span>- {{ formatMoney(orderForm.discount, orderForm.currency) }}</span>
            </div>
            <div class="summary-line total-highlight">
              <strong>{{ l({ lo: 'ຍອດລວມທັງໝົດ (Total Amount):', th: 'ยอดรวมทั้งหมด (Total Amount):', en: 'Total Amount:' }) }}</strong>
              <strong class="total-price">{{ formatMoney(formTotal, orderForm.currency) }}</strong>
            </div>
          </div>
        </v-card-text>

        <v-card-actions class="dialog-actions">
          <v-btn variant="text" @click="createDialog = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </v-btn>
          <v-spacer/>
          <v-btn
            color="primary"
            :loading="creating"
            :disabled="!orderForm.customerName.trim() || !orderForm.customerPhone.trim() || orderForm.items.length === 0"
            @click="submitCreateOrder"
          >
            <v-icon icon="mdi-check" class="mr-1"/>
            {{ l({ lo: 'ຢືນຢັນສ້າງອໍເດີ (ຕັດສະຕ໋ອກ)', th: 'ยืนยันสร้างออเดอร์ (ตัดสต็อก)', en: 'Confirm Order (Deduct Stock)' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Receipt / Order Detail Modal -->
    <v-dialog v-model="receiptDialog" max-width="500">
      <v-card v-if="selectedOrder" class="receipt-card">
        <v-card-title class="receipt-header">
          <div class="receipt-title-box">
            <v-icon icon="mdi-receipt" color="primary" class="mr-2"/>
            <div>
              <h3 class="mb-0">{{ l({ lo: 'ໃບບິນຄຳສັ່ງຊື້', th: 'ใบเสร็จรับเงิน', en: 'Order Receipt' }) }}</h3>
              <div class="text-caption font-weight-bold text-primary">🏪 {{ senderShopName }} · 📞 {{ senderShopPhone }}</div>
            </div>
          </div>
          <span class="receipt-order-no">#{{ selectedOrder.orderNumber }}</span>
        </v-card-title>

        <v-card-text class="receipt-body">
          <div class="receipt-meta">
            <div>
              <small>{{ l({ lo: 'ຮ້ານຄ້າ / Shop:', th: 'ร้านค้า / Shop:', en: 'Shop:' }) }}</small>
              <strong class="text-primary">{{ senderShopName }}</strong>
            </div>
            <div>
              <small>{{ l({ lo: 'ເບີໂທຮ້ານ / Tel:', th: 'เบอร์โทรร้าน / Tel:', en: 'Shop Tel:' }) }}</small>
              <span>{{ senderShopPhone }}</span>
            </div>
            <div>
              <small>{{ l({ lo: 'ວັນທີ:', th: 'วันที่:', en: 'Date:' }) }}</small>
              <span>{{ formatDate(selectedOrder.createdAt) }}</span>
            </div>
          </div>

          <div class="receipt-customer">
            <p><strong>{{ selectedOrder.customerName }}</strong></p>
            <p>📞 {{ selectedOrder.customerPhone }}</p>
            <p v-if="selectedOrder.customerAddress">📍 {{ selectedOrder.customerAddress }}</p>
          </div>

          <div class="receipt-items-list">
            <div v-for="item in selectedOrder.items" :key="item.id" class="receipt-item-row">
              <span class="ri-name">
                {{ item.productName }}
                <small v-if="item.color || item.size" class="text-primary font-weight-bold">
                  ({{ [item.color ? `ສີ: ${item.color}` : '', item.size ? `ເບີ: ${item.size}` : ''].filter(Boolean).join(', ') }})
                </small>
                <small class="text-muted">x{{ item.quantity }}</small>
              </span>
              <span class="ri-price">{{ formatMoney(item.totalPrice, selectedOrder.currency) }}</span>
            </div>
          </div>

          <div class="receipt-calculations">
            <div class="rc-line">
              <span>{{ l({ lo: 'ລວມສິນຄ້າ:', th: 'รวมสินค้า:', en: 'Subtotal:' }) }}</span>
              <span>{{ formatMoney(selectedOrder.subtotal, selectedOrder.currency) }}</span>
            </div>
            <div class="rc-line">
              <span>{{ l({ lo: 'ຄ່າຈັດສົ່ງ:', th: 'ค่าจัดส่ง:', en: 'Shipping:' }) }}</span>
              <span>{{ formatMoney(selectedOrder.shippingFee, selectedOrder.currency) }}</span>
            </div>
            <div v-if="selectedOrder.discount" class="rc-line text-red">
              <span>{{ l({ lo: 'ສ່ວນຫຼຸດ:', th: 'ส่วนลด:', en: 'Discount:' }) }}</span>
              <span>-{{ formatMoney(selectedOrder.discount, selectedOrder.currency) }}</span>
            </div>
            <div class="rc-line rc-total">
              <strong>{{ l({ lo: 'ຍອດລວມ:', th: 'ยอดรวม:', en: 'Total:' }) }}</strong>
              <strong>{{ formatMoney(selectedOrder.totalAmount, selectedOrder.currency) }}</strong>
            </div>
          </div>

          <div class="receipt-badges">
            <span class="badge-status" :class="selectedOrder.paymentStatus === 'paid' ? 'status-paid' : 'status-unpaid'">
              {{ selectedOrder.paymentStatus === 'paid' ? 'ຊຳລະແລ້ວ' : 'ຍັງບໍ່ຊຳລະ' }} ({{ selectedOrder.paymentMethod }})
            </span>
            <span class="courier-badge">
              🚚 {{ selectedOrder.deliveryCourier }}
              <strong v-if="selectedOrder.deliveryTracking">· {{ selectedOrder.deliveryTracking }}</strong>
            </span>
          </div>
        </v-card-text>

        <v-card-actions class="receipt-actions">
          <v-btn variant="outlined" prepend-icon="mdi-content-copy" @click="copyReceiptText">
            {{ copiedReceipt ? l({ lo: 'ຄັດລອກແລ້ວ! ✓', th: 'คัดลอกแล้ว! ✓', en: 'Copied! ✓' }) : l({ lo: 'ຄັດລອກຂໍ້ຄວາມໃບບິນ', th: 'คัดลอกข้อความใบเสร็จ', en: 'Copy Receipt Text' }) }}
          </v-btn>
          <v-btn color="error" variant="tonal" prepend-icon="mdi-printer" @click="openWaybill(selectedOrder)">
            {{ l({ lo: 'ພິມບິນ ອານຸສິດ', th: 'พิมพ์บิล อานุสิต', en: 'Print Anousith Bill' }) }}
          </v-btn>
          <v-spacer/>
          <v-btn color="primary" @click="receiptDialog = false">
            {{ l({ lo: 'ປິດ', th: 'ปิด', en: 'Close' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Change Status Dialog -->
    <v-dialog v-model="statusDialog" max-width="480">
      <v-card class="dialog-card">
        <v-card-title class="dialog-header">
          <v-icon icon="mdi-cog" class="mr-2" color="primary"/>
          {{ l({ lo: 'ປ່ຽນສະຖານະອໍເດີ', th: 'เปลี่ยนสถานะออเดอร์', en: 'Change Order Status' }) }}
        </v-card-title>
        <v-card-text class="dialog-body">
          <v-select
            v-model="statusForm.paymentStatus"
            :items="[
              { value: 'unpaid', label: l({ lo: 'ຍັງບໍ່ຊຳລະ (Unpaid)', th: 'ยังไม่ชำระ (Unpaid)', en: 'Unpaid' }) },
              { value: 'paid', label: l({ lo: 'ຊຳລະແລ້ວ (Paid)', th: 'ชำระแล้ว (Paid)', en: 'Paid' }) },
              { value: 'refunded', label: l({ lo: 'ຄືນເງິນ (Refunded)', th: 'คืนเงิน (Refunded)', en: 'Refunded' }) },
            ]"
            item-title="label"
            item-value="value"
            :label="l({ lo: 'ສະຖານະການຊຳລະເງິນ', th: 'สถานะการชำระเงิน', en: 'Payment Status' })"
            class="mb-3"
            density="compact"
            variant="outlined"
          />

          <v-select
            v-model="statusForm.fulfillmentStatus"
            :items="[
              { value: 'pending', label: l({ lo: 'ລໍຖ້າດຳເນີນການ (Pending)', th: 'รอดำเนินการ (Pending)', en: 'Pending' }) },
              { value: 'confirmed', label: l({ lo: 'ຢືນຢັນແລ້ວ / ຕຽມແພັກ (Confirmed)', th: 'ยืนยันแล้ว / เตรียมแพ็ค (Confirmed)', en: 'Confirmed' }) },
              { value: 'shipping', label: l({ lo: 'ກຳລັງຈັດສົ່ງ (Shipping)', th: 'กำลังจัดส่ง (Shipping)', en: 'Shipping' }) },
              { value: 'delivered', label: l({ lo: 'ຈັດສົ່ງສຳເລັດ (Delivered)', th: 'จัดส่งสำเร็จ (Delivered)', en: 'Delivered' }) },
              { value: 'cancelled', label: l({ lo: 'ຍົກເລີກອໍເດີ (Cancelled - ຄືນສະຕ໋ອກ)', th: 'ยกเลิกออเดอร์ (Cancelled - คืนสต็อก)', en: 'Cancelled (Auto Stock Restore)' }) },
            ]"
            item-title="label"
            item-value="value"
            :label="l({ lo: 'ສະຖານະການຈັດສົ່ງ', th: 'สถานะการจัดส่ง', en: 'Fulfillment Status' })"
            class="mb-3"
            density="compact"
            variant="outlined"
          />

          <v-alert
            v-if="statusForm.fulfillmentStatus === 'cancelled'"
            type="warning"
            variant="tonal"
            class="mb-3 text-caption"
          >
            {{ l({
              lo: '⚠️ ໝາຍເຫດ: ເມື່ອປ່ຽນເປັນ "ຍົກເລີກອໍເດີ", ລະບົບຈະຄືນຈຳນວນສິນຄ້າກັບເຂົ້າສາງອັດຕະໂນມັດ.',
              th: '⚠️ หมายเหตุ: เมื่อเปลี่ยนเป็น "ยกเลิกออเดอร์" ระบบจะคืนจำนวนสินค้ากลับเข้าคลังอัตโนมัติ',
              en: '⚠️ Note: Cancelling will automatically restore deducted stock back to the warehouse.',
            }) }}
          </v-alert>

          <v-select
            v-model="statusForm.deliveryCourier"
            :items="couriers"
            item-title="label"
            item-value="value"
            :label="l({ lo: 'ບໍລິສັດຂົນສົ່ງ', th: 'บริษัทขนส่ง', en: 'Courier' })"
            class="mb-3"
            density="compact"
            variant="outlined"
          />

          <v-text-field
            v-model="statusForm.deliveryTracking"
            :label="l({ lo: 'ເລກພັດສະດຸ / Tracking Code', th: 'เลขพัสดุ / Tracking Code', en: 'Tracking Number' })"
            placeholder="AN123456..."
            class="mb-3"
            density="compact"
            variant="outlined"
          />

          <v-textarea
            v-model="statusForm.notes"
            :label="l({ lo: 'ໝາຍເຫດ', th: 'หมายเหตุ', en: 'Notes' })"
            rows="2"
            density="compact"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="dialog-actions">
          <v-btn variant="text" @click="statusDialog = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </v-btn>
          <v-spacer/>
          <v-btn
            color="primary"
            :loading="updatingStatus"
            @click="saveStatusUpdate"
          >
            {{ l({ lo: 'ບັນທຶກ', th: 'บันทึก', en: 'Save' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Anousith Express Waybill Print Modal -->
    <v-dialog v-model="waybillDialog" max-width="580" scrollable>
      <v-card v-if="waybillOrder" class="dialog-card waybill-modal-card">
        <v-card-title class="dialog-header no-print">
          <div class="d-flex align-center gap-2">
            <span class="anousith-badge-icon">AN</span>
            <div>
              <h3 class="mb-0 text-subtitle-1 font-weight-bold">
                {{ l({ lo: 'ໃບປະໜ້າພັດສະດຸ ອານຸສິດ (Waybill)', th: 'ใบปะหน้าพัสดุ อานุสิต (Waybill)', en: 'Anousith Express Waybill' }) }}
              </h3>
              <small class="text-muted">#{{ waybillOrder.orderNumber }} · {{ waybillOrder.deliveryCourier || 'Anousith Express' }}</small>
            </div>
          </div>
        </v-card-title>

        <!-- Waybill Size Toolbar (no-print) -->
        <div class="waybill-toolbar no-print">
          <div class="size-btn-group">
            <button
              type="button"
              class="size-toggle-btn"
              :class="{ active: waybillPrintSize === 'thermal' }"
              @click="waybillPrintSize = 'thermal'"
            >
              <v-icon icon="mdi-printer-pos" size="14" class="mr-1"/>
              100x150 mm (Thermal)
            </button>
            <button
              type="button"
              class="size-toggle-btn"
              :class="{ active: waybillPrintSize === 'a4' }"
              @click="waybillPrintSize = 'a4'"
            >
              <v-icon icon="mdi-file-document-outline" size="14" class="mr-1"/>
              A4 / A5 (Standard)
            </button>
          </div>
          <v-btn
            color="error"
            prepend-icon="mdi-printer"
            class="print-action-btn"
            @click="triggerPrint"
          >
            {{ l({ lo: 'ສັ່ງພິມບິນດຽວນີ້ (Print)', th: 'สั่งพิมพ์บิลตอนนี้ (Print)', en: 'Print Now' }) }}
          </v-btn>
        </div>

        <!-- Sender Shop Customizer Toolbar (no-print) -->
        <div class="sender-customizer-toolbar no-print">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="d-inline-flex align-center gap-1 text-caption font-weight-bold text-grey-darken-3">
              <v-icon icon="mdi-storefront" size="16" color="primary"/>
              {{ l({ lo: 'ຂໍ້ມູນຮ້ານຄ້າຜູ້ສົ່ງ (ປັບປ່ຽນໄດ້ທັນທີກ່ອນພິມ):', th: 'ข้อมูลร้านค้าผู้ส่ง (ปรับเปลี่ยนได้ทันทีก่อนพิมพ์):', en: 'Sender Shop Information (Editable):' }) }}
            </span>
            <div class="d-flex align-center gap-1">
              <v-chip v-if="senderSavedNotice" size="x-small" color="success" variant="flat" prepend-icon="mdi-check">
                {{ l({ lo: 'ບັນທຶກແລ້ວ', th: 'บันทึกแล้ว', en: 'Saved' }) }}
              </v-chip>
              <v-btn size="x-small" variant="tonal" color="primary" prepend-icon="mdi-content-save-outline" @click="saveSenderAsDefault">
                {{ l({ lo: 'ຈື່ເປັນຄ່າເລີ່ມຕົ້ນ', th: 'จำค่านี้ไว้', en: 'Save as Default' }) }}
              </v-btn>
            </div>
          </div>
          <div class="sender-customizer-inputs">
            <v-text-field
              v-model="senderShopName"
              :label="l({ lo: 'ຊື່ຮ້ານຄ້າ (Shop Name) *', th: 'ชื่อร้านค้า (Shop Name) *', en: 'Shop Name *' })"
              density="compact"
              variant="outlined"
              hide-details
              prepend-inner-icon="mdi-store"
              class="sender-input"
            />
            <v-text-field
              v-model="senderShopPhone"
              :label="l({ lo: 'ເບີໂທຮ້ານ (Shop Tel) *', th: 'เบอร์โทรร้าน (Shop Tel) *', en: 'Shop Phone *' })"
              density="compact"
              variant="outlined"
              hide-details
              prepend-inner-icon="mdi-phone"
              class="sender-input"
            />
            <v-text-field
              v-model="senderShopAddress"
              :label="l({ lo: 'ທີ່ຢູ່ຮ້ານ (Address)', th: 'ที่อยู่ร้าน (Address)', en: 'Shop Address' })"
              density="compact"
              variant="outlined"
              hide-details
              prepend-inner-icon="mdi-map-marker"
              class="sender-input"
            />
          </div>
          <!-- Facebook Page 1-Click Quick Picks ("ຊື່ page ໃດອອກຊື່ນັ້ນ") -->
          <div v-if="pages.length > 0" class="sender-quick-picks mt-2 d-flex align-center gap-1 flex-wrap">
            <span class="text-caption text-medium-emphasis mr-1 font-weight-bold text-primary">📘 {{ l({ lo: 'ຊື່ຕາມ Page:', th: 'ชื่อตาม Page:', en: 'Facebook Page:' }) }}</span>
            <v-chip
              v-for="p in pages"
              :key="p.id"
              size="x-small"
              :variant="senderShopName === p.name ? 'flat' : 'outlined'"
              color="primary"
              class="font-weight-medium"
              @click="applyPageToSender(p)"
            >
              <v-icon icon="mdi-facebook" size="12" class="mr-1"/>
              {{ p.name }}
            </v-chip>
          </div>

          <div v-if="warehouses.length > 0" class="sender-quick-picks mt-2 d-flex align-center gap-1 flex-wrap">
            <span class="text-caption text-medium-emphasis mr-1">{{ l({ lo: 'ເລືອກຈາກສາງ/ຮ້ານ:', th: 'เลือกจากคลัง/ร้าน:', en: 'Pick from:' }) }}</span>
            <v-chip
              v-for="w in warehouses"
              :key="w.id"
              size="x-small"
              variant="outlined"
              :color="w.type === 'shop' ? 'purple' : 'default'"
              @click="applyWarehouseToSender(w)"
            >
              <v-icon :icon="w.type === 'shop' ? 'mdi-storefront' : 'mdi-warehouse'" size="12" class="mr-1"/>
              {{ w.name }}
            </v-chip>
          </div>
        </div>

        <v-card-text class="waybill-dialog-body">
          <!-- The Waybill Print Container (Styled like real Anousith Waybill) -->
          <div
            id="anousith-print-area"
            class="anousith-waybill"
            :class="{ 'size-a4': waybillPrintSize === 'a4' }"
          >
            <!-- 1. Header Banner -->
            <div class="an-header">
              <div class="an-logo-group">
                <div class="an-logo-box">
                  <span class="an-logo-main">ANOUSITH</span>
                  <span class="an-logo-sub">EXPRESS</span>
                </div>
                <div class="an-company-info">
                  <strong>ອານຸສິດ ເອັກສ໌ເພຣັສ (ANOUSITH EXPRESS)</strong>
                  <small>ບໍລິສັດ ອານຸສິດ ຂົນສົ່ງດ່ວນ ຈຳກັດຜູ້ດຽວ</small>
                  <small class="hotline">🏪 ຮ້ານ: <strong>{{ senderShopName }}</strong> (📞 {{ senderShopPhone }})</small>
                </div>
              </div>
              <div class="an-route-box">
                <div class="an-express-pill">EXPRESS ດ່ວນ</div>
                <div class="an-route-code">
                  <span>{{ waybillOrder.warehouse?.code || 'VTE' }}</span>
                  <v-icon icon="mdi-arrow-right" size="14"/>
                  <strong>{{ extractDestination(waybillOrder.customerAddress) }}</strong>
                </div>
              </div>
            </div>

            <!-- 2. Barcode & Tracking -->
            <div class="an-barcode-section">
              <div class="an-barcode-wrap">
                <svg class="an-barcode-svg" viewBox="0 0 320 50" preserveAspectRatio="none">
                  <rect
                    v-for="(b, bIdx) in generateBarcodePattern(waybillOrder.deliveryTracking || waybillOrder.orderNumber)"
                    :key="bIdx"
                    :x="b.x"
                    y="0"
                    :width="b.width"
                    height="50"
                    fill="#000"
                  />
                </svg>
                <div class="an-barcode-text">
                  *{{ waybillOrder.deliveryTracking || waybillOrder.orderNumber }}*
                </div>
              </div>
              <div class="an-track-meta">
                <div>
                  <small>Tracking No / ເລກພັດສະດຸ:</small>
                  <strong>{{ waybillOrder.deliveryTracking || `AN-${waybillOrder.orderNumber}` }}</strong>
                </div>
                <div>
                  <small>ວັນທີສົ່ງ / Date:</small>
                  <span>{{ formatDate(waybillOrder.createdAt) }}</span>
                </div>
              </div>
            </div>

            <!-- 3. Sender & Receiver Section -->
            <div class="an-address-grid">
              <!-- Sender (From) -->
              <div class="an-address-box an-from">
                <div class="an-box-title">
                  <span class="title-tag">FROM</span>
                  <span>ຜູ້ຝາກສົ່ງ (Sender) · ຮ້ານຄ້າ</span>
                </div>
                <div class="an-address-content">
                  <p class="an-name"><strong class="shop-title-text">{{ senderShopName }}</strong></p>
                  <p class="an-phone">📞 <strong>ເບີໂທ: {{ senderShopPhone }}</strong></p>
                  <p class="an-addr">{{ senderShopAddress }}</p>
                </div>
              </div>

              <!-- Receiver (To) - BIG & PROMINENT -->
              <div class="an-address-box an-to">
                <div class="an-box-title to-title">
                  <span class="title-tag to-tag">TO</span>
                  <span>ຜູ້ຮັບພັດສະດຸ (Receiver)</span>
                </div>
                <div class="an-address-content">
                  <div class="an-receiver-name">
                    <strong>{{ waybillOrder.customerName }}</strong>
                  </div>
                  <div class="an-receiver-phone-frame">
                    <span class="phone-label">TEL / ເບີໂທ:</span>
                    <strong class="phone-number">{{ waybillOrder.customerPhone }}</strong>
                  </div>
                  <div class="an-receiver-addr">
                    <small>ທີ່ຢູ່ຈັດສົ່ງ / ສາຂາປາຍທາງ:</small>
                    <p>{{ waybillOrder.customerAddress || '—' }}</p>
                  </div>
                </div>
              </div>
            </div>

            <!-- 4. COD / Payment Highlight Banner -->
            <div class="an-cod-banner" :class="waybillOrder.paymentStatus === 'paid' ? 'is-paid' : 'is-cod'">
              <div class="an-cod-type">
                <span v-if="waybillOrder.paymentStatus === 'paid'" class="cod-tag paid-tag">
                  ✓ ຊຳລະເງິນແລ້ວ (PAID) · ຫ້າມເກັບເງິນປາຍທາງ
                </span>
                <span v-else class="cod-tag cod-alert-tag">
                  ⚠️ ເກັບເງິນປາຍທາງ (COD)
                </span>
                <small class="pay-method">({{ waybillOrder.paymentMethod || 'Bank Transfer' }})</small>
              </div>
              <div class="an-cod-amount">
                <span class="cod-amt-label">ຍອດເກັບປາຍທາງ:</span>
                <strong class="cod-amt-val">
                  {{ waybillOrder.paymentStatus === 'paid' ? '0 LAK' : formatMoney(waybillOrder.totalAmount, waybillOrder.currency) }}
                </strong>
              </div>
            </div>

            <!-- 5. Package Items Matrix -->
            <div class="an-items-section">
              <div class="an-table-header">
                <span class="col-no">#</span>
                <span class="col-name">ລາຍການສິນຄ້າ (Item Name)</span>
                <span class="col-var">ສີ / ເບີ</span>
                <span class="col-qty">ຈຳນວນ</span>
                <span class="col-price">ລາຄາ</span>
              </div>
              <div v-for="(item, idx) in waybillOrder.items" :key="item.id" class="an-table-row">
                <span class="col-no">{{ idx + 1 }}</span>
                <span class="col-name">{{ item.productName }}</span>
                <span class="col-var">{{ [item.color ? `ສີ: ${item.color}` : '', item.size ? `ເບີ: ${item.size}` : ''].filter(Boolean).join(' ') || '-' }}</span>
                <span class="col-qty">{{ item.quantity }}</span>
                <span class="col-price">{{ formatMoney(item.totalPrice, waybillOrder.currency) }}</span>
              </div>
              <div class="an-table-footer">
                <div class="an-footer-left">
                  <span>ລວມຈຳນວນ: <strong>{{ (waybillOrder.items || []).reduce((s, it) => s + it.quantity, 0) }}</strong> ຊິ້ນ</span>
                  <span> | ຄ່າຈັດສົ່ງ: <strong>{{ formatMoney(waybillOrder.shippingFee, waybillOrder.currency) }}</strong></span>
                </div>
                <div class="an-footer-right">
                  <span>ຍອດລວມທັງໝົດ: <strong>{{ formatMoney(waybillOrder.totalAmount, waybillOrder.currency) }}</strong></span>
                </div>
              </div>
            </div>

            <!-- 6. Remarks & Signatures -->
            <div class="an-bottom-section">
              <div class="an-remarks-box">
                <small class="remark-lbl">ໝາຍເຫດ (Remarks):</small>
                <p class="remark-text">{{ waybillOrder.notes || 'ກະລຸນາໂທຫາລູກຄ້າກ່ອນສົ່ງທຸກຄັ້ງ, ຫ້າມໂຍນພັດສະດຸ' }}</p>
                <div class="an-courier-sign">
                  <small>ຂົນສົ່ງ: {{ waybillOrder.deliveryCourier || 'Anousith Express' }}</small>
                </div>
              </div>
              <div class="an-signatures-box">
                <div class="sign-col">
                  <small>ລາຍເຊັນຜູ້ຝາກສົ່ງ (Sender):</small>
                  <div class="sign-line"></div>
                </div>
                <div class="sign-col">
                  <small>ລາຍເຊັນຜູ້ຮັບ (Receiver):</small>
                  <div class="sign-line"></div>
                  <small class="date-lbl">ວັນທີ: ____/____/________</small>
                </div>
              </div>
            </div>

            <!-- 7. Footer Tagline -->
            <div class="an-footer-tagline">
              <small>ANOUSITH EXPRESS · ວ່ອງໄວ ປອດໄພ ທົ່ວປະເທດ · Powered by Meta Super App</small>
            </div>
          </div>
        </v-card-text>

        <v-card-actions class="dialog-actions no-print">
          <v-btn variant="text" @click="waybillDialog = false">
            {{ l({ lo: 'ປິດ', th: 'ปิด', en: 'Close' }) }}
          </v-btn>
          <v-spacer/>
          <v-btn color="error" prepend-icon="mdi-printer" @click="triggerPrint">
            {{ l({ lo: 'ສັ່ງພິມບິນດຽວນີ້ (Print)', th: 'สั่งพิมพ์บิลตอนนี้ (Print)', en: 'Print Waybill' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.orders-page {
  padding-bottom: 40px;
}
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 18px;
  padding: 24px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.page-head p {
  margin: 0 0 7px;
  color: var(--color-primary);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: .13em;
}
.page-head h2 {
  margin: 0 0 7px;
  font-size: 24px;
  font-weight: 750;
}
.page-head span {
  color: var(--color-text-secondary);
  font-size: 13px;
}
.head-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.stat-icon-wrapper {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.bg-blue-light { background: #eff6ff; }
.bg-amber-light { background: #fef3c7; }
.bg-purple-light { background: #f3e8ff; }
.bg-green-light { background: #f0fdf4; }
.stat-text {
  display: flex;
  flex-direction: column;
}
.stat-text small {
  font-size: 11px;
  color: var(--color-text-muted);
}
.stat-text strong {
  font-size: 18px;
  font-weight: 800;
  color: var(--color-text);
}
.text-amber { color: #d97706 !important; }
.text-green { color: #16a34a !important; }

.orders-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.filter-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.tab-pill {
  padding: 7px 14px;
  border-radius: 20px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  font-size: 12.5px;
  font-weight: 700;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all .15s ease;
}
.tab-pill:hover {
  background: var(--color-surface-variant);
  color: var(--color-text);
}
.tab-pill.active {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}

.search-filters {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.warehouse-select {
  height: 38px;
  border: 1px solid var(--color-border);
  border-radius: 20px;
  padding: 0 14px;
  font-size: 13px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
  cursor: pointer;
}
.search-box {
  position: relative;
  display: flex;
  align-items: center;
  max-width: 320px;
  width: 100%;
}
.search-box input {
  width: 100%;
  height: 38px;
  border: 1px solid var(--color-border);
  border-radius: 20px;
  padding: 0 32px 0 36px;
  font-size: 13px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
}
.search-box input:focus {
  border-color: var(--color-primary);
}
.search-box .search-icon {
  position: absolute;
  left: 12px;
  color: var(--color-text-muted);
  pointer-events: none;
}
.search-box .clear-btn {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
}

.order-cards-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.order-row-card {
  padding: 18px 20px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: border-color .15s ease;
}
.order-row-card:hover {
  border-color: color-mix(in srgb, var(--color-primary) 35%, var(--color-border));
}

.order-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 10px;
}
.order-id-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.order-number {
  font-size: 15px;
  font-weight: 800;
  color: var(--color-text);
}
.order-date {
  font-size: 12px;
  color: var(--color-text-muted);
}
.badge-capi {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  background: #eff6ff;
  color: #1877f2;
  border: 1px solid #bfdbfe;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 750;
}
.badge-page-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  background: #eff6ff;
  color: #1d4ed8;
  border: 1px solid #bfdbfe;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 700;
}
.badge-wh-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  background: var(--color-surface-variant);
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  font-size: 11px;
  font-weight: 650;
}

.order-status-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.badge-status {
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 11.5px;
  font-weight: 750;
}
.status-paid { background: #f0fdf4; color: #16a34a; border: 1px solid #bbf7d0; }
.status-unpaid { background: #fef3c7; color: #d97706; border: 1px solid #fde68a; }
.status-refunded { background: #f3e8ff; color: #7e22ce; border: 1px solid #e9d5ff; }

.badge-fulfillment {
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 11.5px;
  font-weight: 750;
}
.fulfill-pending { background: #f1f5f9; color: #475569; border: 1px solid #cbd5e1; }
.fulfill-shipping { background: #eff6ff; color: #2563eb; border: 1px solid #bfdbfe; }
.fulfill-delivered { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
.fulfill-cancelled { background: #fef2f2; color: #dc2626; border: 1px solid #fecaca; }

.order-content-grid {
  display: grid;
  grid-template-columns: 2fr 2.5fr 1.5fr 1.8fr;
  gap: 16px;
  align-items: center;
}
.order-col {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.customer-name-row, .customer-contact-row, .customer-address-row {
  display: flex;
  align-items: center;
  font-size: 12.5px;
}
.customer-name-row strong {
  font-size: 14px;
  color: var(--color-text);
}
.customer-contact-row a {
  color: var(--color-primary);
  text-decoration: none;
  font-weight: 600;
}
.customer-address-row {
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.3;
}

.items-summary-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.item-summary-line {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
}
.item-qty {
  font-weight: 800;
  color: var(--color-primary);
  font-size: 11px;
  background: var(--color-primary-soft);
  padding: 1px 5px;
  border-radius: 6px;
}
.item-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--color-text);
}
.item-price {
  font-weight: 600;
  color: var(--color-text-secondary);
  font-size: 11.5px;
}

.courier-badge {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  font-weight: 650;
  color: var(--color-text-secondary);
}
.tracking-pill {
  font-size: 11px;
  background: var(--color-background);
  padding: 2px 8px;
  border-radius: 8px;
  border: 1px solid var(--color-border);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
}

.total-col {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}
.total-display {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}
.total-display small {
  font-size: 10px;
  color: var(--color-text-muted);
}
.total-display strong {
  font-size: 16px;
  font-weight: 850;
  color: var(--color-text);
}

.action-buttons {
  display: flex;
  gap: 6px;
}
.action-btn {
  display: inline-flex;
  align-items: center;
  padding: 5px 10px;
  border-radius: 8px;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all .15s ease;
}
.receipt-btn:hover {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  border-color: var(--color-primary);
}
.status-btn:hover {
  background: var(--color-surface-variant);
  color: var(--color-text);
}

.form-section-title {
  display: flex;
  align-items: center;
  font-size: 13px;
  font-weight: 800;
  color: var(--color-primary);
  margin: 12px 0 8px;
}
.fields-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.fields-row-3 {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 10px;
}

.product-picker-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.flex-1 { flex: 1; }

.order-items-table {
  border: 1px solid var(--color-border);
  border-radius: 12px;
  overflow: hidden;
  background: var(--color-surface-variant);
}
.order-item-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}
.order-item-row:last-child {
  border-bottom: none;
}
.item-pic {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background-color: var(--color-surface-variant);
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-border);
  flex-shrink: 0;
}
.item-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.item-info strong {
  font-size: 12.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.item-qty-input {
  display: flex;
  align-items: center;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
}
.qty-btn {
  width: 26px;
  height: 26px;
  border: none;
  background: var(--color-surface-variant);
  cursor: pointer;
  font-weight: 800;
}
.qty-num {
  width: 34px;
  text-align: center;
  border: none;
  outline: none;
  font-size: 12px;
  font-weight: 750;
}
.item-unit-price {
  display: flex;
  align-items: center;
  gap: 4px;
}
.price-input {
  width: 80px;
  height: 28px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 0 6px;
  font-size: 12px;
  outline: none;
}
.item-total {
  min-width: 80px;
  text-align: right;
  font-size: 12.5px;
}
.del-btn {
  background: none;
  border: none;
  color: #ef4444;
  cursor: pointer;
  padding: 4px;
}

.order-summary-box {
  padding: 14px;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.summary-line {
  display: flex;
  justify-content: space-between;
  font-size: 12.5px;
}
.total-highlight {
  border-top: 1px dashed var(--color-border);
  padding-top: 8px;
  margin-top: 4px;
  font-size: 14px;
}
.total-price {
  color: var(--color-primary);
  font-size: 16px;
  font-weight: 850;
}

.receipt-card {
  padding: 16px;
}
.receipt-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 12px;
}
.receipt-title-box {
  display: flex;
  align-items: center;
}
.receipt-order-no {
  font-size: 14px;
  font-weight: 800;
  color: var(--color-primary);
}
.receipt-meta {
  display: flex;
  justify-content: space-between;
  margin: 12px 0;
  font-size: 12px;
  color: var(--color-text-secondary);
}
.receipt-customer {
  padding: 10px;
  background: var(--color-surface-variant);
  border-radius: 10px;
  font-size: 12.5px;
  margin-bottom: 12px;
}
.receipt-customer p {
  margin: 2px 0;
}
.receipt-items-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  border-top: 1px dashed var(--color-border);
  border-bottom: 1px dashed var(--color-border);
  padding: 10px 0;
  margin-bottom: 12px;
}
.receipt-item-row {
  display: flex;
  justify-content: space-between;
  font-size: 12.5px;
}
.receipt-calculations {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12.5px;
  margin-bottom: 14px;
}
.rc-line {
  display: flex;
  justify-content: space-between;
}
.rc-total {
  border-top: 1px solid var(--color-border);
  padding-top: 6px;
  margin-top: 4px;
  font-size: 14px;
}
.receipt-badges {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.empty-icon-circle {
  width: 64px;
  height: 64px;
  border-radius: 20px;
  background: var(--color-primary-soft);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
}
.empty-state h2 {
  font-size: 18px;
  font-weight: 750;
  margin-bottom: 8px;
}
.empty-state p {
  color: var(--color-text-secondary);
  font-size: 13px;
  max-width: 440px;
  margin-bottom: 20px;
}

@media (max-width: 960px) {
  .order-content-grid {
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .page-head {
    display: flex;
    flex-direction: column;
    padding: 18px;
    gap: 16px;
  }
  .head-actions {
    width: 100%;
    display: grid;
    grid-template-columns: 1fr;
  }
  .stats-row {
    grid-template-columns: 1fr;
    gap: 8px;
  }
  .order-content-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }
  .total-col {
    align-items: stretch;
  }
  .total-display {
    align-items: flex-start;
  }
  .action-buttons {
    width: 100%;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }
  .action-buttons .waybill-btn {
    grid-column: 1 / -1;
    min-height: 44px;
    font-size: 13.5px;
    font-weight: 750;
    justify-content: center;
    background: #fef2f2;
    color: #d9251d;
    border-color: #fecaca;
  }
  .action-buttons .receipt-btn,
  .action-buttons .status-btn {
    min-height: 40px;
    font-size: 12.5px;
    justify-content: center;
  }
  .orders-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  .search-filters {
    flex-direction: column;
    align-items: stretch;
  }
  .search-box {
    max-width: 100%;
  }
  .fields-row-2, .fields-row-3 {
    grid-template-columns: 1fr;
  }
}

.variant-select-box {
  background: var(--color-background);
  border: 1px dashed var(--color-border);
  border-radius: 8px;
  padding: 8px;
}
.variant-tag-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
}
.tag-color {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fee2e2;
}
.tag-size {
  background: #f0fdf4;
  color: #16a34a;
  border: 1px solid #dcfce7;
}

/* ===================================================
   ANOUSITH EXPRESS WAYBILL STYLES & PRINT LAYOUT
   =================================================== */

.anousith-badge-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: #ffd200;
  color: #d9251d;
  font-weight: 900;
  font-size: 14px;
  border-radius: 8px;
  border: 2px solid #d9251d;
  letter-spacing: -0.5px;
}

.waybill-modal-card {
  background: #f8fafc;
}

.waybill-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  padding: 10px 16px;
  background: #f1f5f9;
  border-bottom: 1px solid #e2e8f0;
}

.size-btn-group {
  display: inline-flex;
  background: #e2e8f0;
  border-radius: 8px;
  padding: 2px;
  gap: 2px;
}

.size-toggle-btn {
  display: inline-flex;
  align-items: center;
  padding: 6px 12px;
  border: none;
  background: transparent;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.size-toggle-btn.active {
  background: #ffffff;
  color: #0f172a;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
}

.sender-customizer-toolbar {
  padding: 12px 16px;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
}

.sender-customizer-inputs {
  display: grid;
  grid-template-columns: 1fr 1fr 1.2fr;
  gap: 8px;
}

@media (max-width: 600px) {
  .sender-customizer-inputs {
    grid-template-columns: 1fr;
  }
}

.shop-title-text {
  font-size: 13px;
  font-weight: 900;
  color: #000000;
  letter-spacing: 0.3px;
}

.waybill-dialog-body {
  padding: 16px;
  display: flex;
  justify-content: center;
  background: #e2e8f0;
}

/* Base Waybill Container (Standard 100x150 mm Thermal Sticker format) */
.anousith-waybill {
  width: 100%;
  max-width: 420px;
  min-height: 590px;
  background: #ffffff;
  color: #000000;
  border: 2px solid #000000;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  font-family: 'Noto Sans Lao', system-ui, -apple-system, sans-serif;
  font-size: 11px;
  line-height: 1.25;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.anousith-waybill.size-a4 {
  max-width: 520px;
  font-size: 12px;
}

/* 1. Header */
.an-header {
  display: flex;
  align-items: stretch;
  justify-content: space-between;
  background: #ffd200;
  border-bottom: 2px solid #000000;
  padding: 8px 10px;
  color: #000000;
}

.an-logo-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.an-logo-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #d9251d;
  color: #ffffff;
  padding: 4px 6px;
  border-radius: 4px;
  line-height: 1;
}

.an-logo-main {
  font-size: 13px;
  font-weight: 900;
  letter-spacing: 0.5px;
}

.an-logo-sub {
  font-size: 8px;
  font-weight: 800;
  letter-spacing: 1.5px;
}

.an-company-info {
  display: flex;
  flex-direction: column;
}

.an-company-info strong {
  font-size: 11px;
  font-weight: 800;
  color: #d9251d;
}

.an-company-info small {
  font-size: 9px;
  color: #333333;
}

.an-company-info .hotline {
  font-weight: 700;
  color: #000000;
}

.an-route-box {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: center;
  gap: 2px;
}

.an-express-pill {
  background: #d9251d;
  color: #ffffff;
  font-size: 9px;
  font-weight: 800;
  padding: 2px 6px;
  border-radius: 3px;
}

.an-route-code {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 900;
  color: #000000;
}

/* 2. Barcode & Tracking */
.an-barcode-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 2px solid #000000;
  padding: 8px 10px;
  background: #ffffff;
  gap: 12px;
}

.an-barcode-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.an-barcode-svg {
  width: 100%;
  max-width: 240px;
  height: 44px;
}

.an-barcode-text {
  font-family: monospace;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 1px;
  margin-top: 2px;
}

.an-track-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-size: 10px;
  text-align: right;
  gap: 2px;
}

.an-track-meta strong {
  font-size: 12px;
  font-weight: 900;
  color: #d9251d;
}

/* 3. Address Grid (Sender & Receiver) */
.an-address-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  border-bottom: 2px solid #000000;
}

.an-address-box {
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.an-address-box.an-from {
  border-right: 1px solid #000000;
  background: #fafafa;
}

.an-address-box.an-to {
  background: #ffffff;
}

.an-box-title {
  display: flex;
  align-items: center;
  gap: 4px;
  font-weight: 800;
  font-size: 10px;
  border-bottom: 1px dashed #cccccc;
  padding-bottom: 3px;
  margin-bottom: 2px;
}

.title-tag {
  background: #000000;
  color: #ffffff;
  font-size: 8px;
  font-weight: 900;
  padding: 1px 4px;
  border-radius: 2px;
}

.an-box-title.dest-title .title-tag {
  background: #d9251d;
}

.dest-route-pill {
  margin-left: auto;
  background: #ffd200;
  color: #000000;
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 9px;
  font-weight: 800;
}

.an-receiver-phone-frame {
  margin: 4px 0;
  padding: 4px 6px;
  background: #fef08a;
  border: 2px solid #000000;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.phone-label {
  font-size: 9px;
  font-weight: 800;
  color: #000000;
}

.phone-number {
  font-size: 14px;
  font-weight: 900;
  color: #d9251d;
  letter-spacing: 0.5px;
}

.an-receiver-addr {
  font-size: 10px;
  line-height: 1.3;
}

.an-receiver-addr p {
  margin: 2px 0 0;
  font-weight: 700;
  color: #000000;
}

.an-name {
  font-size: 11px;
  margin: 0;
}

.an-phone {
  font-size: 10px;
  margin: 2px 0;
  color: #333333;
}

.an-addr {
  font-size: 9px;
  color: #555555;
  margin: 0;
}

/* 4. COD Highlight Banner */
.an-cod-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-bottom: 2px solid #000000;
  color: #ffffff;
}

.an-cod-banner.is-cod {
  background: #d9251d;
}

.an-cod-banner.is-paid {
  background: #047857;
}

.an-cod-type {
  display: flex;
  align-items: center;
  gap: 6px;
}

.cod-tag {
  font-size: 11px;
  font-weight: 900;
  letter-spacing: 0.3px;
}

.pay-method {
  font-size: 9px;
  opacity: 0.9;
}

.an-cod-amount {
  display: flex;
  align-items: center;
  gap: 6px;
}

.cod-amt-label {
  font-size: 10px;
  font-weight: 700;
}

.cod-amt-val {
  font-size: 15px;
  font-weight: 900;
  background: rgba(0, 0, 0, 0.25);
  padding: 2px 8px;
  border-radius: 4px;
  letter-spacing: 0.5px;
}

/* 5. Package Items Matrix */
.an-items-section {
  border-bottom: 2px solid #000000;
  display: flex;
  flex-direction: column;
}

.an-table-header {
  display: flex;
  align-items: center;
  background: #f1f5f9;
  border-bottom: 1px solid #000000;
  font-weight: 800;
  font-size: 9px;
  padding: 4px 8px;
  text-transform: uppercase;
}

.an-table-row {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-bottom: 1px dashed #e2e8f0;
  font-size: 10px;
}

.col-no {
  width: 20px;
  font-weight: 700;
}

.col-name {
  flex: 1;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.col-var {
  width: 70px;
  font-size: 9px;
  color: #475569;
}

.col-qty {
  width: 35px;
  text-align: center;
  font-weight: 700;
}

.col-price {
  width: 65px;
  text-align: right;
  font-weight: 700;
}

.an-table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  background: #fafafa;
  font-size: 9px;
  border-top: 1px solid #000000;
}

/* 6. Remarks & Signatures */
.an-bottom-section {
  display: grid;
  grid-template-columns: 1.2fr 1fr;
  border-bottom: 1px solid #000000;
  flex: 1;
}

.an-remarks-box {
  padding: 6px 8px;
  border-right: 1px solid #000000;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.remark-lbl {
  font-weight: 800;
  font-size: 9px;
  color: #475569;
}

.remark-text {
  margin: 2px 0 6px;
  font-size: 9px;
  color: #000000;
  line-height: 1.25;
}

.an-courier-sign small {
  font-size: 8px;
  color: #64748b;
  font-weight: 600;
}

.an-signatures-box {
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 8px;
}

.sign-col small {
  font-size: 8px;
  font-weight: 700;
  color: #475569;
}

.sign-line {
  border-bottom: 1px dotted #94a3b8;
  height: 14px;
}

.date-lbl {
  font-size: 7px;
  color: #94a3b8;
}

/* 7. Footer Tagline */
.an-footer-tagline {
  padding: 3px 8px;
  background: #ffd200;
  text-align: center;
  font-weight: 800;
  font-size: 8px;
  color: #000000;
  letter-spacing: 0.3px;
}

/* ===================================================
   PRINT STYLES FOR WAYBILL
   =================================================== */

@media print {
  @page {
    size: 100mm 150mm;
    margin: 0;
  }

  body, html {
    margin: 0 !important;
    padding: 0 !important;
    background: #ffffff !important;
  }

  body * {
    visibility: hidden;
  }

  #anousith-print-area,
  #anousith-print-area * {
    visibility: visible;
  }

  #anousith-print-area {
    position: fixed !important;
    left: 0 !important;
    top: 0 !important;
    width: 100% !important;
    max-width: 100% !important;
    min-height: 100% !important;
    margin: 0 !important;
    padding: 4mm !important;
    box-shadow: none !important;
    border: none !important;
    background: #ffffff !important;
    -webkit-print-color-adjust: exact !important;
    print-color-adjust: exact !important;
    box-sizing: border-box;
  }

  .no-print,
  .v-overlay__scrim,
  .v-overlay-container,
  header,
  nav {
    display: none !important;
  }
}
</style>
