<script setup lang="ts">
import type { AutomationFlow, ItemsResponse, Product, ProductVariant, Warehouse, WarehouseStock } from '~/types/automation'
import { useProductAdStrategy, type ProductAdStrategy } from '~/composables/useProductAdStrategy'

definePageMeta({ middleware: 'auth' })

const { can } = useAuth()
const { l } = useLocale()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Products.' })

const { data, pending, error, refresh } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: automationData } = await useApi<ItemsResponse<AutomationFlow>>('/proxy/api/v1/automations')
const { data: warehouseData } = await useApi<ItemsResponse<Warehouse>>('/proxy/api/v1/warehouses')

const products = computed(() => data.value?.items || [])
const automations = computed(() => automationData.value?.items || [])
const warehouses = computed(() => warehouseData.value?.items || [])

const dialog = ref(false)
const saving = ref(false)
const uploading = ref(false)
const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')

const form = reactive<Product & { code?: string; salePrice: number | null; cost: number | null; quantity: number | null; currency: string; packagingCost: number | null; shippingCostStore: number | null; targetProfit: number | null }>({
  id: '',
  code: '',
  name: '',
  price: '',
  salePrice: null,
  cost: null,
  packagingCost: null,
  shippingCostStore: null,
  targetProfit: null,
  quantity: null,
  currency: 'LAK',
  description: '',
  imageUrl: '',
  barcode: '',
  category: '',
  unit: 'ຊິ້ນ',
  weight: null,
  minStock: null,
  status: 'active',
  galleryUrls: '',
  colors: '',
  sizes: '',
  variants: '',
})

const variantsList = ref<ProductVariant[]>([])

const deleteDialog = ref(false)
const deleting = ref(false)
const productToDelete = ref<Product | null>(null)

// Stock per warehouse dialog
const stockDialog = ref(false)
const stockLoading = ref(false)
const stockSaving = ref(false)
const selectedProductForStock = ref<Product | null>(null)
const currentProductStocks = ref<WarehouseStock[]>([])
const stockForm = reactive<{
  warehouseId: string
  quantity: number
  alertThreshold: number
}>({
  warehouseId: '',
  quantity: 0,
  alertThreshold: 5,
})

// Product detail modal
const detailDialog = ref(false)
const selectedProductForDetail = ref<Product | null>(null)

// Product Ad Strategy modal
const { productStrategies, loadStrategies, getStrategy, setStrategy, autoCalculate } = useProductAdStrategy()
const strategyDialog = ref(false)
const strategySaving = ref(false)
const selectedProductForStrategy = ref<Product | null>(null)
const strategyForm = reactive<ProductAdStrategy>({
  productId: '',
  enabled: true,
  targetCost: null,
  grossMargin: null,
  scaleMinPurchases: 3,
  scaleMaxCPA: null,
  scaleMaxDailyBudgetCap: null,
  stopMinSpend: null,
})

const fileInput = ref<HTMLInputElement>()
const searchQuery = ref('')
const copiedCodeId = ref<string | null>(null)

const filteredProducts = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return products.value
  return products.value.filter((p) =>
    p.name.toLowerCase().includes(q) ||
    (p.code && p.code.toLowerCase().includes(q)) ||
    (p.barcode && p.barcode.toLowerCase().includes(q)) ||
    (p.category && p.category.toLowerCase().includes(q)) ||
    p.price.toLowerCase().includes(q) ||
    (p.description && p.description.toLowerCase().includes(q))
  )
})

function copyCode(code: string, id: string) {
  navigator.clipboard.writeText(`#${code}`)
  copiedCodeId.value = id
  noticeType.value = 'success'
  notice.value = l({
    lo: `ຄັດລອກໂຄ້ດ #${code} ແລ້ວ! ພິມໃນແຊັດໄດ້ທັນທີ`,
    th: `คัดลอกโค้ด #${code} แล้ว! พิมพ์ในแชทได้ทันที`,
    en: `Copied #${code}! Type this in chat to reply instantly.`,
  })
  setTimeout(() => {
    copiedCodeId.value = null
    notice.value = ''
  }, 3000)
}

function openCreate() {
  Object.assign(form, {
    id: '',
    code: '',
    name: '',
    price: '',
    salePrice: null,
    cost: null,
    packagingCost: null,
    shippingCostStore: null,
    targetProfit: null,
    quantity: null,
    currency: 'LAK',
    description: '',
    imageUrl: '',
    barcode: '',
    category: '',
    unit: 'ຊິ້ນ',
    weight: null,
    minStock: null,
    status: 'active',
    galleryUrls: '',
    colors: '',
    sizes: '',
    variants: '',
  })
  variantsList.value = []
  dialog.value = true
}

function openEdit(product: Product) {
  Object.assign(
    form,
    {
      salePrice: null,
      cost: null,
      packagingCost: null,
      shippingCostStore: null,
      targetProfit: null,
      quantity: null,
      currency: 'LAK',
      barcode: '',
      category: '',
      unit: 'ຊິ້ນ',
      weight: null,
      minStock: null,
      status: 'active',
      galleryUrls: '',
      colors: '',
      sizes: '',
      variants: '',
    },
    product
  )
  try {
    variantsList.value = product.variants ? JSON.parse(product.variants) : []
  } catch {
    variantsList.value = []
  }
  dialog.value = true
}

function generateVariants() {
  const colors = (form.colors || '').split(',').map((s) => s.trim()).filter(Boolean)
  const sizes = (form.sizes || '').split(',').map((s) => s.trim()).filter(Boolean)

  if (colors.length === 0 && sizes.length === 0) return

  const existingMap = new Map<string, ProductVariant>()
  for (const v of variantsList.value) {
    existingMap.set(`${v.color}___${v.size}`, v)
  }

  const nextList: ProductVariant[] = []
  if (colors.length > 0 && sizes.length > 0) {
    for (const c of colors) {
      for (const s of sizes) {
        const key = `${c}___${s}`
        const old = existingMap.get(key)
        nextList.push({
          id: old?.id,
          color: c,
          size: s,
          quantity: old ? old.quantity : 0,
          sku: old?.sku || (form.code ? `${form.code}-${c}-${s}` : ''),
          price: old?.price ?? (form.salePrice || undefined),
        })
      }
    }
  } else if (colors.length > 0) {
    for (const c of colors) {
      const key = `${c}___`
      const old = existingMap.get(key)
      nextList.push({
        id: old?.id,
        color: c,
        size: '-',
        quantity: old ? old.quantity : 0,
        sku: old?.sku || (form.code ? `${form.code}-${c}` : ''),
        price: old?.price ?? (form.salePrice || undefined),
      })
    }
  } else if (sizes.length > 0) {
    for (const s of sizes) {
      const key = `___${s}`
      const old = existingMap.get(key)
      nextList.push({
        id: old?.id,
        color: '-',
        size: s,
        quantity: old ? old.quantity : 0,
        sku: old?.sku || (form.code ? `${form.code}-${s}` : ''),
        price: old?.price ?? (form.salePrice || undefined),
      })
    }
  }
  variantsList.value = nextList
  syncVariants()
}

function addCustomVariant() {
  variantsList.value.push({
    color: '',
    size: '',
    quantity: 0,
    sku: '',
    price: form.salePrice || undefined,
  })
  syncVariants()
}

function removeVariant(index: number) {
  variantsList.value.splice(index, 1)
  syncVariants()
}

function syncVariants() {
  form.variants = variantsList.value.length ? JSON.stringify(variantsList.value) : ''
  if (variantsList.value.length > 0) {
    form.quantity = variantsList.value.reduce((sum, v) => sum + (Number(v.quantity) || 0), 0)
  }
}

function parseProductVariants(jsonStr?: string): ProductVariant[] {
  if (!jsonStr) return []
  try {
    return JSON.parse(jsonStr)
  } catch {
    return []
  }
}

const productAutomations = (productId: string) => automations.value.filter((item) => item.productId === productId)

function requestDelete(product: Product) {
  productToDelete.value = product
  deleteDialog.value = true
}

async function deleteProduct() {
  if (!productToDelete.value || productAutomations(productToDelete.value.id).length) return
  deleting.value = true
  notice.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/products/${productToDelete.value.id}`, { method: 'DELETE' })
    noticeType.value = 'success'
    notice.value = l({ lo: `ລຶບ “${productToDelete.value.name}” ແລ້ວ`, th: `ลบ “${productToDelete.value.name}” แล้ว`, en: `Deleted “${productToDelete.value.name}”.` })
    deleteDialog.value = false
    await refresh()
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, l({ lo: 'ລຶບສິນຄ້າບໍ່ສຳເລັດ', th: 'ไม่สามารถลบสินค้าได้', en: 'Unable to delete the product.' }))
  } finally {
    deleting.value = false
    productToDelete.value = null
  }
}

function errorMessage(error: unknown, fallback: string) {
  const value = error as { data?: { error?: string | { message?: string }; message?: string }; statusMessage?: string }
  const upstream = value.data?.error
  return (typeof upstream === 'string' ? upstream : upstream?.message) || value.data?.message || value.statusMessage || fallback
}

async function uploadProductImage(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files?.length) return
  const file = input.files[0]
  if (!file) return
  uploading.value = true
  notice.value = ''
  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('folder', 'products')
    const result = await $fetch<{ url: string }>('/api/proxy/api/v1/storage/upload', { method: 'POST', body: formData })
    form.imageUrl = result.url
  } catch (err) {
    noticeType.value = 'error'
    notice.value = errorMessage(err, l({ lo: 'ອັບໂຫຼດຮູບບໍ່ສຳເລັດ ກະລຸນາລອງໃໝ່', th: 'อัปโหลดรูปไม่สำเร็จ โปรดลองอีกครั้ง', en: 'Failed to upload image. Please try again.' }))
  } finally {
    uploading.value = false
    input.value = ''
  }
}

async function saveProduct() {
  if (!form.name.trim() || (form.salePrice == null && !form.price.trim())) return
  saving.value = true
  notice.value = ''
  try {
    await $fetch('/api/proxy/api/v1/products', {
      method: 'POST',
      body: {
        id: form.id || undefined,
        code: form.code?.trim() || undefined,
        name: form.name.trim(),
        price: form.price.trim() || `${form.salePrice} ${form.currency}`,
        salePrice: form.salePrice,
        cost: form.cost,
        packagingCost: form.packagingCost,
        shippingCostStore: form.shippingCostStore,
        targetProfit: form.targetProfit,
        quantity: form.quantity,
        currency: form.currency.trim().toUpperCase() || 'LAK',
        description: form.description.trim(),
        imageUrl: form.imageUrl?.trim() || undefined,
        barcode: form.barcode?.trim() || undefined,
        category: form.category?.trim() || undefined,
        unit: form.unit?.trim() || undefined,
        weight: form.weight,
        minStock: form.minStock,
        status: form.status || 'active',
        galleryUrls: form.galleryUrls?.trim() || undefined,
        colors: form.colors?.trim() || undefined,
        sizes: form.sizes?.trim() || undefined,
        variants: variantsList.value.length ? JSON.stringify(variantsList.value) : undefined,
      },
    })
    dialog.value = false
    noticeType.value = 'success'
    notice.value = form.id
      ? l({ lo: 'ອັບເດດສິນຄ້າແລ້ວ', th: 'อัปเดตสินค้าแล้ว', en: 'Product updated.' })
      : l({ lo: 'ສ້າງສິນຄ້າແລ້ວ ແລະ ພ້ອມໃຊ້ກັບ Auto Reply', th: 'สร้างสินค้าแล้วและพร้อมใช้กับ Auto Reply', en: 'Product created and ready for Auto Replies.' })
    await refresh()
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, l({ lo: 'ບັນທຶກສິນຄ້າບໍ່ສຳເລັດ', th: 'บันทึกสินค้าไม่สำเร็จ', en: 'Unable to save the product.' }))
  } finally {
    saving.value = false
  }
}

// Warehouse Stocks modal functions
async function openWarehouseStock(product: Product) {
  selectedProductForStock.value = product
  stockDialog.value = true
  stockLoading.value = true
  try {
    const res = await $fetch<ItemsResponse<WarehouseStock>>(`/api/proxy/api/v1/products/${product.id}/stock`)
    currentProductStocks.value = res.items || []
    const defaultWh = warehouses.value.find((w) => w.isDefault) || warehouses.value[0]
    stockForm.warehouseId = defaultWh ? defaultWh.id : ''
    const current = currentProductStocks.value.find((s) => s.warehouseId === stockForm.warehouseId)
    stockForm.quantity = current ? current.quantity : 0
    stockForm.alertThreshold = current ? current.alertThreshold : 5
  } catch (err) {
    noticeType.value = 'error'
    notice.value = errorMessage(err, l({ lo: 'ໂຫຼດສະຕ໋ອກສິນຄ້າບໍ່ສຳເລັດ', th: 'โหลดสต็อกสินค้าไม่สำเร็จ', en: 'Failed to load product stock.' }))
  } finally {
    stockLoading.value = false
  }
}

function onWarehouseSelectChanged() {
  const current = currentProductStocks.value.find((s) => s.warehouseId === stockForm.warehouseId)
  stockForm.quantity = current ? current.quantity : 0
  stockForm.alertThreshold = current ? current.alertThreshold : 5
}

async function saveWarehouseStock() {
  if (!selectedProductForStock.value || !stockForm.warehouseId) return
  stockSaving.value = true
  try {
    await $fetch(`/api/proxy/api/v1/products/${selectedProductForStock.value.id}/stock`, {
      method: 'POST',
      body: {
        warehouseId: stockForm.warehouseId,
        quantity: Number(stockForm.quantity),
        alertThreshold: Number(stockForm.alertThreshold),
      },
    })
    noticeType.value = 'success'
    notice.value = l({ lo: 'ອັບເດດສະຕ໋ອກສາງຮຽບຮ້ອຍແລ້ວ', th: 'อัปเดตสต็อกคลังเรียบร้อยแล้ว', en: 'Warehouse stock updated.' })
    // Reload stocks
    const res = await $fetch<ItemsResponse<WarehouseStock>>(`/api/proxy/api/v1/products/${selectedProductForStock.value.id}/stock`)
    currentProductStocks.value = res.items || []
    await refresh()
  } catch (err) {
    noticeType.value = 'error'
    notice.value = errorMessage(err, l({ lo: 'ອັບເດດສະຕ໋ອກບໍ່ສຳເລັດ', th: 'อัปเดตสต็อกไม่สำเร็จ', en: 'Failed to update stock.' }))
  } finally {
    stockSaving.value = false
  }
}

// Product Detail modal
async function openProductDetail(product: Product) {
  selectedProductForDetail.value = product
  detailDialog.value = true
  stockLoading.value = true
  try {
    const res = await $fetch<ItemsResponse<WarehouseStock>>(`/api/proxy/api/v1/products/${product.id}/stock`)
    currentProductStocks.value = res.items || []
  } catch {
    currentProductStocks.value = []
  } finally {
    stockLoading.value = false
  }
}

const formatAmount = (value?: number | null, currency = 'LAK') =>
  value == null
    ? '—'
    : new Intl.NumberFormat(undefined, { style: 'currency', currency, maximumFractionDigits: 2 }).format(value)

const unitProfit = (product: Product) =>
  product.salePrice != null && product.cost != null ? product.salePrice - product.cost : undefined

onMounted(() => {
  loadStrategies()
})

function getProductAdCount(productId: string): number {
  const flows = productAutomations(productId)
  return flows.reduce((sum, f) => sum + (f.targets?.length || 0), 0)
}

function openAdStrategy(product: Product) {
  selectedProductForStrategy.value = product
  loadStrategies()
  const existing = getStrategy(product.id)
  if (existing) {
    Object.assign(strategyForm, {
      productId: product.id,
      enabled: existing.enabled ?? true,
      targetCost: existing.targetCost ?? null,
      grossMargin: existing.grossMargin ?? null,
      scaleMinPurchases: existing.scaleMinPurchases ?? 3,
      scaleMaxCPA: existing.scaleMaxCPA ?? null,
      scaleMaxDailyBudgetCap: existing.scaleMaxDailyBudgetCap ?? null,
      stopMinSpend: existing.stopMinSpend ?? null,
    })
  } else {
    const calculated = autoCalculate(product, product.currency || 'LAK')
    Object.assign(strategyForm, calculated)
  }
  strategyDialog.value = true
}

function autoCalcCurrentStrategy() {
  if (!selectedProductForStrategy.value) return
  const calc = autoCalculate(selectedProductForStrategy.value, selectedProductForStrategy.value.currency || 'LAK')
  Object.assign(strategyForm, calc)
  noticeType.value = 'success'
  notice.value = l({
    lo: 'ຄິດໄລ່ຍຸດທະສາດໂຄສະນາອັດຕະໂນມັດຈາກລາຄາສິນຄ້າແລ້ວ!',
    th: 'คำนวณยุทธศาสตร์โฆษณาอัตโนมัติจากราคาสินค้าแล้ว!',
    en: 'Auto-calculated ad strategy from product price!',
  })
}

function saveAdStrategy() {
  if (!selectedProductForStrategy.value) return
  strategySaving.value = true
  try {
    setStrategy(selectedProductForStrategy.value.id, {
      enabled: strategyForm.enabled,
      targetCost: strategyForm.targetCost != null && !isNaN(Number(strategyForm.targetCost)) ? Number(strategyForm.targetCost) : null,
      grossMargin: strategyForm.grossMargin != null && !isNaN(Number(strategyForm.grossMargin)) ? Number(strategyForm.grossMargin) : null,
      scaleMinPurchases: strategyForm.scaleMinPurchases != null && !isNaN(Number(strategyForm.scaleMinPurchases)) ? Number(strategyForm.scaleMinPurchases) : 3,
      scaleMaxCPA: strategyForm.scaleMaxCPA != null && !isNaN(Number(strategyForm.scaleMaxCPA)) ? Number(strategyForm.scaleMaxCPA) : null,
      scaleMaxDailyBudgetCap: strategyForm.scaleMaxDailyBudgetCap != null && !isNaN(Number(strategyForm.scaleMaxDailyBudgetCap)) ? Number(strategyForm.scaleMaxDailyBudgetCap) : null,
      stopMinSpend: strategyForm.stopMinSpend != null && !isNaN(Number(strategyForm.stopMinSpend)) ? Number(strategyForm.stopMinSpend) : null,
    })
    noticeType.value = 'success'
    notice.value = l({
      lo: `ບັນທຶກຍຸດທະສາດໂຄສະນາຂອງ “${selectedProductForStrategy.value.name}” ສຳເລັດແລ້ວ!`,
      th: `บันทึกยุทธศาสตร์โฆษณาของ “${selectedProductForStrategy.value.name}” สำเร็จแล้ว!`,
      en: `Saved ad strategy for “${selectedProductForStrategy.value.name}”!`,
    })
    strategyDialog.value = false
  } catch (err) {
    noticeType.value = 'error'
    notice.value = l({
      lo: 'ບັນທຶກຍຸດທະສາດບໍ່ສຳເລັດ',
      th: 'บันทึกยุทธศาสตร์ไม่สำเร็จ',
      en: 'Failed to save strategy',
    })
  } finally {
    strategySaving.value = false
  }
}

function resetAdStrategy() {
  if (!selectedProductForStrategy.value) return
  setStrategy(selectedProductForStrategy.value.id, { enabled: false })
  strategyDialog.value = false
  noticeType.value = 'success'
  notice.value = l({
    lo: `ລ້າງຍຸດທະສາດສະເພາະຂອງ “${selectedProductForStrategy.value.name}” ແລ້ວ (ກັບໄປໃຊ້ຄ່າກາງ)`,
    th: `ล้างยุทธศาสตร์เฉพาะของ “${selectedProductForStrategy.value.name}” แล้ว (กลับไปใช้ค่ากลาง)`,
    en: `Reset custom strategy for “${selectedProductForStrategy.value.name}” to account default.`,
  })
}

function goToProductAds(productId: string) {
  strategyDialog.value = false
  if (detailDialog.value) detailDialog.value = false
  navigateTo({ path: '/ads', query: { product: productId } })
}
</script>

<template>
  <div class="products-page">
    <section class="app-page-head">
      <div class="head-content">
        <span class="head-badge">
          <v-icon icon="mdi-package-variant-closed" size="14" />
          Product Catalog & Inventory
        </span>
        <h1>{{ l({ lo: 'ຈັດການສິນຄ້າ (Products)', th: 'จัดการสินค้า (Products)', en: 'Product Catalog' }) }}</h1>
        <p>{{ l({ lo: 'ເກັບຊື່, ລາຄາ, ຮູບ, ບາໂຄ້ດ ແລະ ລາຍລະອຽດສະຕ໋ອກແຍກສາງ', th: 'เก็บชื่อ ราคา รูปภาพ บาร์โค้ด และรายละเอียดสต็อกแยกคลัง', en: 'Manage product details, barcodes, categories and warehouse stock balances.' }) }}</p>
      </div>
      <div class="head-actions">
        <v-btn variant="outlined" prepend-icon="mdi-warehouse" to="/warehouses">
          {{ l({ lo: 'ສາງ ແລະ ໜ້າຮ້ານ', th: 'คลังสินค้าและหน้าร้าน', en: 'Warehouses' }) }}
        </v-btn>
        <v-btn variant="outlined" prepend-icon="mdi-cart-outline" to="/orders">
          {{ l({ lo: 'ຈັດການອໍເດີ', th: 'จัดการออเดอร์', en: 'Orders' }) }}
        </v-btn>
        <v-btn variant="outlined" prepend-icon="mdi-robot-happy-outline" to="/auto-replies">
          {{ l({ lo: 'ຈັດການ Automation', th: 'จัดการ Automation', en: 'Automations' }) }}
        </v-btn>
        <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreate">
          {{ l({ lo: 'ເພີ່ມສິນຄ້າ', th: 'เพิ่มสินค้า', en: 'Add product' }) }}
        </v-btn>
      </div>
    </section>

    <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice = ''">
      {{ notice }}
    </v-alert>
    <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
      {{ l({ lo: 'ໂຫຼດສິນຄ້າບໍ່ສຳເລັດ', th: 'โหลดสินค้าไม่สำเร็จ', en: 'Products could not be loaded.' }) }}
      <button class="inline-action" @click="refresh()">{{ l({ lo: 'ລອງໃໝ່', th: 'ลองอีกครั้ง', en: 'Try again' }) }}</button>
    </v-alert>

    <!-- Product Search & Filter Toolbar -->
    <div class="product-toolbar mb-4">
      <div class="product-search-box">
        <v-icon icon="mdi-magnify" size="18" class="search-icon"/>
        <input
          v-model="searchQuery"
          type="text"
          :aria-label="l({ lo: 'ຄົ້ນຫາສິນຄ້າ', th: 'ค้นหาสินค้า', en: 'Search Products' })"
          :placeholder="l({ lo: 'ຄົ້ນຫາຊື່ສິນຄ້າ, ໂຄ້ດ (#P01), ບາໂຄ້ດ, ໝວດໝູ່...', th: 'ค้นหาชื่อสินค้า, โค้ด (#P01), บาร์โค้ด, หมวดหมู่...', en: 'Search product name, code (#P01), barcode, category...' })"
        />
        <button v-if="searchQuery" class="clear-btn" @click="searchQuery = ''">
          <v-icon icon="mdi-close" size="14"/>
        </button>
      </div>
      <div class="product-count-chip">
        {{ filteredProducts.length }} {{ l({ lo: 'ລາຍການ', th: 'รายการ', en: 'items' }) }}
      </div>
    </div>

    <!-- Product Grid -->
    <section v-if="pending" class="product-grid">
      <v-skeleton-loader v-for="index in 3" :key="index" type="card"/>
    </section>

    <section v-else-if="filteredProducts.length" class="product-grid">
      <article v-for="product in filteredProducts" :key="product.id" class="product-card">
        <div
          class="product-icon"
          :style="product.imageUrl ? `background-image: url(${product.imageUrl}); background-size: cover; background-position: center;` : ''"
          @click="openProductDetail(product)"
        >
          <v-icon v-if="!product.imageUrl" icon="mdi-package-variant-closed"/>
        </div>
        <div class="product-main-details">
          <div class="product-title-row">
            <h3 class="product-title-link" @click="openProductDetail(product)">{{ product.name }}</h3>
            <span class="price-tag">{{ product.salePrice != null ? formatAmount(product.salePrice, product.currency) : product.price }}</span>
          </div>

          <!-- Category, Barcode, Unit, Colors, Sizes pills -->
          <div class="product-tags-row">
            <span v-if="product.category" class="tag-pill tag-category">{{ product.category }}</span>
            <span v-if="product.barcode" class="tag-pill tag-barcode">
              <v-icon icon="mdi-barcode" size="11" class="mr-1"/>{{ product.barcode }}
            </span>
            <span v-if="product.unit" class="tag-pill tag-unit">{{ product.unit }}</span>
            <span v-if="product.colors" class="tag-pill tag-color" :title="product.colors">
              <v-icon icon="mdi-palette" size="11" class="mr-1"/>{{ l({ lo: 'ສີ', th: 'สี', en: 'Colors' }) }}: {{ product.colors }}
            </span>
            <span v-if="product.sizes" class="tag-pill tag-size" :title="product.sizes">
              <v-icon icon="mdi-ruler" size="11" class="mr-1"/>{{ l({ lo: 'ເບີ', th: 'เบอร์', en: 'Sizes' }) }}: {{ product.sizes }}
            </span>
            <span v-if="parseProductVariants(product.variants).length" class="tag-pill tag-variants">
              <v-icon icon="mdi-layers-outline" size="11" class="mr-1"/>{{ parseProductVariants(product.variants).length }} {{ l({ lo: 'ຮູບແບບ', th: 'แบบ', en: 'variants' }) }}
            </span>

            <!-- Auto-Reply Flow pill -->
            <span
              v-if="productAutomations(product.id).length"
              class="tag-pill tag-autoreply"
              :title="productAutomations(product.id).map(a => a.name).join(', ')"
              @click.stop="openAdStrategy(product)"
              style="cursor: pointer;"
            >
              <v-icon icon="mdi-robot-outline" size="11" class="mr-1"/>
              {{ productAutomations(product.id).length }} Flow
              <span class="ad-count-sub">({{ getProductAdCount(product.id) }} ແອດ)</span>
            </span>

            <!-- Custom Ad Strategy pill -->
            <span
              v-if="productStrategies[product.id]?.enabled"
              class="tag-pill tag-strategy"
              :title="`Target CPA: ${formatAmount(productStrategies[product.id]?.targetCost, product.currency)}`"
              @click.stop="openAdStrategy(product)"
              style="cursor: pointer;"
            >
              <v-icon icon="mdi-target-variant" size="11" class="mr-1"/>
              {{ l({ lo: 'Strategy ພ້ອມ', th: 'Strategy พร้อม', en: 'Custom Strategy' }) }}
            </span>
          </div>

          <p v-if="product.description" class="product-desc">{{ product.description }}</p>

          <div class="product-meta-row">
            <button
              v-if="product.code"
              type="button"
              class="code-copy-pill"
              @click="copyCode(product.code, product.id)"
              :title="l({ lo: 'ຄລິກເພື່ອຄັດລອກໂຄ້ດ', th: 'คลิกเพื่อคัดลอกโค้ด', en: 'Click to copy code' })"
            >
              <v-icon icon="mdi-lightning-bolt" size="13"/>
              <strong>#{{ product.code }}</strong>
              <v-icon :icon="copiedCodeId === product.id ? 'mdi-check' : 'mdi-content-copy'" size="12" class="ml-1"/>
            </button>
            <small v-else class="text-muted">ID · {{ product.id.slice(0, 8) }}</small>
          </div>

          <div class="inventory-row">
            <span>
              <small>{{ l({ lo: 'ຕົ້ນທຶນ', th: 'ต้นทุน', en: 'Cost' }) }}</small>
              <strong>{{ formatAmount(product.cost, product.currency) }}</strong>
            </span>
            <span>
              <small>{{ l({ lo: 'ກຳໄລ/ໜ່ວຍ', th: 'กำไร/หน่วย', en: 'Profit/unit' }) }}</small>
              <strong>{{ formatAmount(unitProfit(product), product.currency) }}</strong>
            </span>
            <span>
              <small>{{ l({ lo: 'ຄົງເຫຼືອທັງໝົດ', th: 'คงเหลือทั้งหมด', en: 'Total Stock' }) }}</small>
              <strong :class="{ 'text-red': (product.quantity || 0) <= (product.minStock || 0) }">{{ product.quantity ?? '—' }}</strong>
            </span>
          </div>
        </div>

        <div class="product-actions">
          <v-btn
            icon="mdi-target-variant"
            variant="text"
            size="small"
            color="deep-purple"
            @click="openAdStrategy(product)"
            :title="l({ lo: 'ຍຸດທະສາດໂຄສະນາ (Ad Strategy)', th: 'ยุทธศาสตร์โฆษณา (Ad Strategy)', en: 'Ad Strategy & Auto-Pilot' })"
          />
          <v-btn
            icon="mdi-warehouse"
            variant="text"
            size="small"
            color="indigo"
            @click="openWarehouseStock(product)"
            :title="l({ lo: 'ຈັດການສະຕ໋ອກແຍກສາງ', th: 'จัดการสต็อกแยกคลัง', en: 'Warehouse Stocks' })"
          />
          <v-btn
            icon="mdi-information-outline"
            variant="text"
            size="small"
            color="teal"
            @click="openProductDetail(product)"
            :title="l({ lo: 'ເບິ່ງລາຍລະອຽດສິນຄ້າ', th: 'ดูรายละเอียดสินค้า', en: 'Product Details' })"
          />
          <v-btn
            icon="mdi-pencil-outline"
            variant="text"
            size="small"
            :disabled="!can('pages:connect')"
            :aria-label="`${l({ lo: 'ແກ້ໄຂ', th: 'แก้ไข', en: 'Edit' })} ${product.name}`"
            @click="openEdit(product)"
          />
          <v-btn
            icon="mdi-delete-outline"
            variant="text"
            size="small"
            color="error"
            :disabled="!can('pages:connect')"
            :aria-label="`${l({ lo: 'ລຶບ', th: 'ลบ', en: 'Delete' })} ${product.name}`"
            @click="requestDelete(product)"
          />
        </div>
      </article>
    </section>

    <section v-else-if="!error" class="empty-state">
      <div><v-icon icon="mdi-package-variant-plus" size="30"/></div>
      <h2>{{ l({ lo: 'ເພີ່ມສິນຄ້າທຳອິດ', th: 'เพิ่มสินค้าชิ้นแรก', en: 'Add the first product' }) }}</h2>
      <p>{{ l({ lo: 'ສິນຄ້າຈະເກັບຊື່, ລາຄາ, ຮູບ ແລະ ລາຍລະອຽດທີ່ໃຊ້ໃນຊຸດຂໍ້ຄວາມ ແລະ ການຂຶ້ນອໍເດີ', th: 'สินค้าจะเก็บชื่อ ราคา รูปภาพ และรายละเอียดที่ใช้ในชุดข้อความและการสร้างออเดอร์', en: 'A product provides the name, price, image and details used in replies and orders.' }) }}</p>
      <v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreate">
        {{ l({ lo: 'ເພີ່ມສິນຄ້າ', th: 'เพิ่มสินค้า', en: 'Add product' }) }}
      </v-btn>
    </section>

    <!-- Create / Edit Product Dialog -->
    <v-dialog v-model="dialog" max-width="640" scrollable>
      <v-card class="dialog-card">
        <v-card-title class="dialog-title">
          {{ form.id ? l({ lo: 'ແກ້ໄຂສິນຄ້າ', th: 'แก้ไขสินค้า', en: 'Edit product' }) : l({ lo: 'ເພີ່ມສິນຄ້າໃໝ່', th: 'เพิ่มสินค้าใหม่', en: 'Add product' }) }}
        </v-card-title>
        <v-card-text class="form">
          <div class="image-upload-wrapper mb-3">
            <div class="image-preview" :style="form.imageUrl ? `background-image: url(${form.imageUrl})` : ''">
              <v-icon v-if="!form.imageUrl" icon="mdi-image-outline" size="32" color="grey"/>
              <v-progress-circular v-if="uploading" indeterminate color="primary" class="upload-loader"/>
            </div>
            <div class="upload-actions">
              <v-btn variant="outlined" size="small" prepend-icon="mdi-upload" :loading="uploading" @click="fileInput?.click()">
                {{ l({ lo: 'ອັບໂຫຼດຮູບ', th: 'อัปโหลดรูป', en: 'Upload image' }) }}
              </v-btn>
              <input ref="fileInput" type="file" accept="image/*" hidden :aria-label="l({ lo: 'ອັບໂຫຼດຮູບ', th: 'อัปโหลดรูป', en: 'Upload image' })" @change="uploadProductImage"/>
              <div class="text-caption text-grey mt-1">{{ l({ lo: 'ຫຼືວາງ URL ດ້ານລຸ່ມ', th: 'หรือวาง URL ด้านล่าง', en: 'Or paste a URL below:' }) }}</div>
            </div>
          </div>
          <v-text-field v-model="form.imageUrl" :label="l({ lo: 'URL ຮູບພາບ', th: 'URL รูปภาพ', en: 'Image URL' })" placeholder="https://..." hide-details class="mb-2"/>

          <div class="fields-2col mb-2">
            <v-text-field v-model="form.code" :label="l({ lo: 'ລະຫັດສິນຄ້າ (Code)', th: 'รหัสสินค้า (Code)', en: 'Product Code' })" placeholder="P01" maxlength="50" hide-details/>
            <v-text-field v-model="form.barcode" :label="l({ lo: 'ບາໂຄ້ດ (Barcode)', th: 'บาร์โค้ด (Barcode)', en: 'Barcode' })" placeholder="885000000000" hide-details/>
          </div>

          <v-text-field v-model="form.name" :label="l({ lo: 'ຊື່ສິນຄ້າ *', th: 'ชื่อสินค้า *', en: 'Product name *' })" autofocus maxlength="120" counter class="mb-1"/>

          <div class="fields-2col mb-2">
            <v-text-field v-model="form.category" :label="l({ lo: 'ໝວດໝູ່ສິນຄ້າ', th: 'หมวดหมู่สินค้า', en: 'Category' })" placeholder="ເຄື່ອງສຳອາງ, ເສື້ອຜ້າ, ອາຫານເສີມ..." hide-details/>
            <v-text-field v-model="form.unit" :label="l({ lo: 'ຫົວໜ່ວຍ', th: 'หน่วยนับ', en: 'Unit' })" placeholder="ຊິ້ນ, ກ່ອງ, ຊຸດ, ແກ້ວ..." hide-details/>
          </div>

          <div class="money-grid mb-2">
            <v-text-field v-model.number="form.salePrice" type="number" min="0" :label="l({ lo: 'ລາຄາຂາຍ', th: 'ราคาขาย', en: 'Sale price' })" hide-details/>
            <v-text-field v-model="form.currency" :label="l({ lo: 'ສະກຸນເງິນ', th: 'สกุลเงิน', en: 'Currency' })" maxlength="8" hide-details/>
          </div>

          <div class="money-grid mb-2">
            <v-text-field v-model.number="form.cost" type="number" min="0" :label="l({ lo: 'ຕົ້ນທຶນຕໍ່ໜ່ວຍ', th: 'ต้นทุนต่อหน่วย', en: 'Cost per unit' })" hide-details/>
            <v-text-field v-model.number="form.quantity" type="number" min="0" step="1" :label="l({ lo: 'ຈຳນວນຄົງເຫຼືອທັງໝົດ', th: 'จำนวนคงเหลือทั้งหมด', en: 'Total Stock' })" hide-details/>
          </div>

          <div class="fields-3col mb-2">
            <v-text-field v-model.number="form.packagingCost" type="number" min="0" :label="l({ lo: 'ຄ່າແພັກ/ໜ່ວຍ', th: 'ค่าแพ็ค/หน่วย', en: 'Packaging cost' })" suffix="₭" hide-details/>
            <v-text-field v-model.number="form.shippingCostStore" type="number" min="0" :label="l({ lo: 'ຄ່າສົ່ງຮ້ານອອກ/ໜ່ວຍ', th: 'ค่าส่งร้านออก', en: 'Store shipping subsidy' })" suffix="₭" hide-details/>
            <v-text-field v-model.number="form.targetProfit" type="number" min="0" :label="l({ lo: 'ເປົ້າກຳໄລ/ໜ່ວຍ', th: 'เป้ากำไร/หน่วย', en: 'Target profit' })" suffix="₭" hide-details/>
          </div>

          <div class="fields-2col mb-2">
            <v-text-field v-model.number="form.weight" type="number" min="0" step="0.01" :label="l({ lo: 'ນ້ຳໜັກ (kg)', th: 'น้ำหนัก (kg)', en: 'Weight (kg)' })" hide-details/>
            <v-text-field v-model.number="form.minStock" type="number" min="0" step="1" :label="l({ lo: 'ແຈ້ງເຕືອນສະຕ໋ອກຂັ້ນຕ່ຳ', th: 'แจ้งเตือนสต็อกขั้นต่ำ', en: 'Min Stock Alert' })" hide-details/>
          </div>

          <v-text-field
            v-model="form.price"
            :label="l({ lo: 'ລາຄາສຳລັບຂໍ້ຄວາມ (ບໍ່ບັງຄັບ)', th: 'ราคาที่แสดงในข้อความ (ไม่บังคับ)', en: 'Reply display price (optional)' })"
            :placeholder="form.salePrice == null ? '' : `${form.salePrice} ${form.currency}`"
            maxlength="60"
            counter
            class="mb-2"
          />

          <!-- Variants Section (Colors & Sizes & Quantities) -->
          <div class="variant-builder-box mb-3">
            <div class="variant-builder-header mb-2">
              <strong><v-icon icon="mdi-palette-swatch-outline" size="16" class="mr-1 text-primary"/>{{ l({ lo: 'ຮູບແບບສິນຄ້າ: ສີ (Colors) & ເບີ (Sizes)', th: 'รูปแบบสินค้า: สี & เบอร์/ไซส์', en: 'Product Variants (Color & Size)' }) }}</strong>
              <p class="text-caption text-grey mb-0">{{ l({ lo: 'ໃສ່ຊື່ສີ ແລະ ເບີ ຂັ້ນດ້ວຍຈຸດ (ເຊັ່ນ: ດຳ, ຂາວ, ແດງ ແລະ 38, 39, 40)', th: 'ใส่ชื่อสีและเบอร์ คั่นด้วยจุลภาค (เช่น ดำ, ขาว, แดง และ 38, 39, 40)', en: 'Enter colors and sizes separated by commas (e.g. Black, White and 38, 39, 40)' }) }}</p>
            </div>

            <div class="fields-2col mb-2">
              <v-text-field
                v-model="form.colors"
                :label="l({ lo: 'ລາຍການສີ (Colors)', th: 'รายการสี', en: 'Colors list' })"
                placeholder="ດຳ, ຂາວ, ແດງ, ຟ້າ..."
                density="compact"
                hide-details
              />
              <v-text-field
                v-model="form.sizes"
                :label="l({ lo: 'ລາຍການເບີ / ໄຊສ໌ (Sizes)', th: 'รายการเบอร์ / ไซส์', en: 'Sizes list' })"
                placeholder="38, 39, 40, 41 ຫຼື S, M, L, XL..."
                density="compact"
                hide-details
              />
            </div>

            <div class="variant-actions-bar mb-2">
              <v-btn
                size="small"
                variant="tonal"
                color="primary"
                prepend-icon="mdi-auto-fix"
                :disabled="!form.colors && !form.sizes"
                @click="generateVariants"
              >
                {{ l({ lo: 'ສ້າງຕາຕະລາງ ສີ x ເບີ ອັດຕະໂນມັດ', th: 'สร้างตาราง สี x เบอร์ อัตโนมัติ', en: 'Generate Color x Size Table' }) }}
              </v-btn>
              <v-btn
                size="small"
                variant="outlined"
                prepend-icon="mdi-plus"
                @click="addCustomVariant"
              >
                {{ l({ lo: '+ ເພີ່ມແຖວເອງ', th: '+ เพิ่มแถวเอง', en: '+ Add Row' }) }}
              </v-btn>
            </div>

            <!-- Variant Matrix Table -->
            <div v-if="variantsList.length" class="variant-matrix-table">
              <div class="variant-table-header">
                <span>{{ l({ lo: 'ສີ', th: 'สี', en: 'Color' }) }}</span>
                <span>{{ l({ lo: 'ເບີ/ໄຊສ໌', th: 'เบอร์/ไซส์', en: 'Size' }) }}</span>
                <span>{{ l({ lo: 'ຈຳນວນ', th: 'จำนวน', en: 'Qty' }) }} *</span>
                <span>SKU</span>
                <span></span>
              </div>
              <div v-for="(v, idx) in variantsList" :key="idx" class="variant-table-row">
                <input v-model="v.color" type="text" class="variant-input" :aria-label="l({ lo: 'ສີ', th: 'สี', en: 'Color' })" :placeholder="l({ lo: 'ສີ', th: 'สี', en: 'Color' })" @input="syncVariants"/>
                <input v-model="v.size" type="text" class="variant-input" :aria-label="l({ lo: 'ເບີ', th: 'เบอร์', en: 'Size' })" :placeholder="l({ lo: 'ເບີ', th: 'เบอร์', en: 'Size' })" @input="syncVariants"/>
                <input v-model.number="v.quantity" type="number" min="0" class="variant-input qty-box" :aria-label="l({ lo: 'ຈຳນວນ', th: 'จำนวน', en: 'Qty' })" @input="syncVariants"/>
                <input v-model="v.sku" type="text" class="variant-input" aria-label="SKU" placeholder="SKU" @input="syncVariants"/>
                <button type="button" class="btn-del-variant" @click="removeVariant(idx)" :title="l({ lo: 'ລຶບແຖວ', th: 'ลบแถว', en: 'Remove' })">
                  <v-icon icon="mdi-trash-can-outline" size="14"/>
                </button>
              </div>
              <div class="variant-table-footer">
                <small class="text-grey">
                  {{ l({ lo: 'ລວມສະຕ໋ອກທັງໝົດ:', th: 'รวมสต็อกทั้งหมด:', en: 'Total variant stock:' }) }}
                  <strong>{{ variantsList.reduce((acc, it) => acc + (Number(it.quantity) || 0), 0) }}</strong>
                  ({{ l({ lo: 'ອັບເດດລົງຊ່ອງຈຳນວນລວມແລ້ວ', th: 'อัปเดตลงช่องจำนวนรวมแล้ว', en: 'synced with Total Stock' }) }})
                </small>
              </div>
            </div>
          </div>

          <v-textarea v-model="form.description" :label="l({ lo: 'ລາຍລະອຽດສິນຄ້າ', th: 'รายละเอียดสินค้า', en: 'Product details' })" rows="3" maxlength="1000" counter/>
        </v-card-text>
        <v-card-actions>
          <v-btn variant="text" @click="dialog = false">{{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}</v-btn>
          <v-spacer/>
          <v-btn
            color="primary"
            :loading="saving"
            :disabled="!form.name.trim() || (form.salePrice == null && !form.price.trim())"
            @click="saveProduct"
          >
            {{ l({ lo: 'ບັນທຶກສິນຄ້າ', th: 'บันทึกสินค้า', en: 'Save product' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Warehouse Stock Modal -->
    <v-dialog v-model="stockDialog" max-width="540">
      <v-card class="dialog-card">
        <v-card-title class="dialog-title">
          <v-icon icon="mdi-warehouse" class="mr-2" color="indigo"/>
          {{ l({ lo: 'ຈັດການສະຕ໋ອກແຍກສາງ / ໜ້າຮ້ານ', th: 'จัดการสต็อกแยกคลัง / หน้าร้าน', en: 'Stock per Warehouse/Shop' }) }}
        </v-card-title>
        <v-card-text v-if="selectedProductForStock">
          <div class="product-mini-bar mb-3">
            <strong class="text-primary">{{ selectedProductForStock.name }}</strong>
            <small v-if="selectedProductForStock.code">#{{ selectedProductForStock.code }}</small>
          </div>

          <div v-if="stockLoading" class="py-4 text-center">
            <v-progress-circular indeterminate color="primary"/>
          </div>

          <div v-else class="warehouse-stocks-list mb-4">
            <div v-for="w in warehouses" :key="w.id" class="stock-location-item">
              <div class="location-info">
                <v-icon :icon="w.type === 'shop' ? 'mdi-storefront' : 'mdi-warehouse'" size="16" class="mr-1"/>
                <strong>{{ w.name }}</strong>
                <span v-if="w.isDefault" class="badge-mini-def">Default</span>
              </div>
              <div class="location-qty">
                <span class="stock-number">
                  {{ (currentProductStocks.find((s) => s.warehouseId === w.id)?.quantity) ?? 0 }}
                </span>
                <small class="text-muted">{{ selectedProductForStock.unit || 'ຊິ້ນ' }}</small>
              </div>
            </div>
          </div>

          <!-- Adjust Stock Section -->
          <div class="adjust-box">
            <h4 class="mb-2">{{ l({ lo: 'ປັບຍອດສະຕ໋ອກໃນສາງ', th: 'ปรับยอดสต็อกในคลัง', en: 'Adjust Location Stock' }) }}</h4>
            <v-select
              v-model="stockForm.warehouseId"
              :items="warehouses"
              item-title="name"
              item-value="id"
              :label="l({ lo: 'ເລືອກສາງ/ໜ້າຮ້ານ', th: 'เลือกคลัง/หน้าร้าน', en: 'Select Location' })"
              density="compact"
              variant="outlined"
              class="mb-2"
              @update:model-value="onWarehouseSelectChanged"
            />
            <div class="fields-2col mb-2">
              <v-text-field
                v-model.number="stockForm.quantity"
                type="number"
                min="0"
                :label="l({ lo: 'ຈຳນວນຄົງເຫຼືອ', th: 'จำนวนคงเหลือ', en: 'Stock Quantity' })"
                density="compact"
                variant="outlined"
              />
              <v-text-field
                v-model.number="stockForm.alertThreshold"
                type="number"
                min="0"
                :label="l({ lo: 'ແຈ້ງເຕືອນຂັ້ນຕ່ຳ', th: 'แจ้งเตือนขั้นต่ำ', en: 'Alert Threshold' })"
                density="compact"
                variant="outlined"
              />
            </div>
            <v-btn
              color="indigo"
              block
              :loading="stockSaving"
              @click="saveWarehouseStock"
            >
              {{ l({ lo: 'ບັນທຶກສະຕ໋ອກສາງນີ້', th: 'บันทึกสต็อกคลังนี้', en: 'Update This Location Stock' }) }}
            </v-btn>
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer/>
          <v-btn color="primary" @click="stockDialog = false">{{ l({ lo: 'ປິດ', th: 'ปิด', en: 'Close' }) }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Product Detail Modal -->
    <v-dialog v-model="detailDialog" max-width="560">
      <v-card v-if="selectedProductForDetail" class="dialog-card">
        <v-card-title class="dialog-title">
          <v-icon icon="mdi-information-outline" class="mr-2" color="teal"/>
          {{ l({ lo: 'ລາຍລະອຽດສິນຄ້າ', th: 'รายละเอียดสินค้า', en: 'Product Details' }) }}
        </v-card-title>
        <v-card-text class="detail-body">
          <div class="detail-top-card">
            <div
              class="detail-img"
              :style="selectedProductForDetail.imageUrl ? `background-image: url(${selectedProductForDetail.imageUrl})` : ''"
            >
              <v-icon v-if="!selectedProductForDetail.imageUrl" icon="mdi-package-variant-closed" size="36"/>
            </div>
            <div class="detail-headings">
              <h3>{{ selectedProductForDetail.name }}</h3>
              <span class="detail-price-badge">
                {{ selectedProductForDetail.salePrice != null ? formatAmount(selectedProductForDetail.salePrice, selectedProductForDetail.currency) : selectedProductForDetail.price }}
              </span>
              <p v-if="selectedProductForDetail.code" class="text-caption text-muted">Code: #{{ selectedProductForDetail.code }}</p>
            </div>
          </div>

          <div class="detail-specs-grid mb-3">
            <div class="spec-cell">
              <small>{{ l({ lo: 'ບາໂຄ້ດ (Barcode)', th: 'บาร์โค้ด', en: 'Barcode' }) }}</small>
              <strong>{{ selectedProductForDetail.barcode || '—' }}</strong>
            </div>
            <div class="spec-cell">
              <small>{{ l({ lo: 'ໝວດໝູ່ (Category)', th: 'หมวดหมู่', en: 'Category' }) }}</small>
              <strong>{{ selectedProductForDetail.category || '—' }}</strong>
            </div>
            <div class="spec-cell">
              <small>{{ l({ lo: 'ຫົວໜ່ວຍ (Unit)', th: 'หน่วยนับ', en: 'Unit' }) }}</small>
              <strong>{{ selectedProductForDetail.unit || 'ຊິ້ນ' }}</strong>
            </div>
            <div class="spec-cell">
              <small>{{ l({ lo: 'ນ້ຳໜັກ (Weight)', th: 'น้ำหนัก', en: 'Weight' }) }}</small>
              <strong>{{ selectedProductForDetail.weight ? `${selectedProductForDetail.weight} kg` : '—' }}</strong>
            </div>
            <div class="spec-cell">
              <small>{{ l({ lo: 'ຕົ້ນທຶນ (Cost)', th: 'ต้นทุน', en: 'Cost' }) }}</small>
              <strong>{{ formatAmount(selectedProductForDetail.cost, selectedProductForDetail.currency) }}</strong>
            </div>
            <div class="spec-cell">
              <small>{{ l({ lo: 'ກຳໄລຕໍ່ໜ່ວຍ (Profit)', th: 'กำไรต่อหน่วย', en: 'Profit' }) }}</small>
              <strong>{{ formatAmount(unitProfit(selectedProductForDetail), selectedProductForDetail.currency) }}</strong>
            </div>
            <div v-if="selectedProductForDetail.colors" class="spec-cell">
              <small>{{ l({ lo: 'ສີທີ່ມີ (Colors)', th: 'สีที่มี', en: 'Colors' }) }}</small>
              <strong class="text-primary">{{ selectedProductForDetail.colors }}</strong>
            </div>
            <div v-if="selectedProductForDetail.sizes" class="spec-cell">
              <small>{{ l({ lo: 'ເບີ/ໄຊສ໌ (Sizes)', th: 'เบอร์/ไซส์', en: 'Sizes' }) }}</small>
              <strong class="text-primary">{{ selectedProductForDetail.sizes }}</strong>
            </div>
          </div>

          <!-- Variants matrix breakdown table -->
          <div v-if="parseProductVariants(selectedProductForDetail.variants).length" class="detail-variants-section mb-3">
            <h4 class="mb-2">
              <v-icon icon="mdi-layers-outline" size="16" class="mr-1 text-primary"/>
              {{ l({ lo: 'ລາຍລະອຽດສະຕ໋ອກຕາມ ສີ & ເບີ (Variants Stock)', th: 'รายละเอียดสต็อกตาม สี & เบอร์', en: 'Stock by Color & Size (Variants)' }) }}
            </h4>
            <div class="detail-variants-table">
              <div class="dvt-header">
                <span>{{ l({ lo: 'ສີ', th: 'สี', en: 'Color' }) }}</span>
                <span>{{ l({ lo: 'ເບີ/ໄຊສ໌', th: 'เบอร์/ไซส์', en: 'Size' }) }}</span>
                <span>{{ l({ lo: 'ຈຳນວນສະຕ໋ອກ', th: 'จำนวนสต็อก', en: 'Stock' }) }}</span>
                <span>SKU</span>
              </div>
              <div v-for="(v, vIdx) in parseProductVariants(selectedProductForDetail.variants)" :key="vIdx" class="dvt-row">
                <span class="dvt-color">
                  <span class="dot-swatch"></span>
                  {{ v.color || '—' }}
                </span>
                <span class="dvt-size">
                  <strong class="size-pill">{{ v.size || '—' }}</strong>
                </span>
                <span class="dvt-qty">
                  <span class="qty-badge" :class="{ 'zero-stock': (v.quantity || 0) <= 0 }">
                    {{ v.quantity || 0 }} {{ selectedProductForDetail.unit || 'ຊິ້ນ' }}
                  </span>
                </span>
                <span class="dvt-sku text-muted text-caption">{{ v.sku || '—' }}</span>
              </div>
            </div>
          </div>

          <!-- Stock per warehouse list -->
          <div class="detail-stocks-section mb-3">
            <h4 class="mb-2">{{ l({ lo: 'ສະຕ໋ອກສິນຄ້າແຍກຕາມສາງ / ໜ້າຮ້ານ', th: 'สต็อกสินค้าแยกตามคลัง / หน้าร้าน', en: 'Stock Breakdown by Location' }) }}</h4>
            <div class="stocks-table-mini">
              <div v-for="w in warehouses" :key="w.id" class="stock-mini-row">
                <span>
                  <v-icon :icon="w.type === 'shop' ? 'mdi-storefront' : 'mdi-warehouse'" size="14" class="mr-1"/>
                  {{ w.name }}
                </span>
                <strong>
                  {{ (currentProductStocks.find((s) => s.warehouseId === w.id)?.quantity) ?? 0 }}
                  {{ selectedProductForDetail.unit || 'ຊິ້ນ' }}
                </strong>
              </div>
            </div>
          </div>

          <!-- Ad Strategy & Meta Ads section -->
          <div class="detail-strategy-section mb-3">
            <div class="d-flex align-center justify-space-between mb-2">
              <h4 class="mb-0">
                <v-icon icon="mdi-target-variant" size="16" class="mr-1 text-deep-purple"/>
                {{ l({ lo: 'ຍຸດທະສາດໂຄສະນາ & Meta Ads', th: 'ยุทธศาสตร์โฆษณา & Meta Ads', en: 'Ad Strategy & Meta Ads' }) }}
              </h4>
              <v-btn
                size="x-small"
                variant="tonal"
                color="deep-purple"
                prepend-icon="mdi-cog-outline"
                @click="openAdStrategy(selectedProductForDetail)"
              >
                {{ l({ lo: 'ຕັ້ງຄ່າຍຸດທະສາດ', th: 'ตั้งค่ายุทธศาสตร์', en: 'Configure Strategy' }) }}
              </v-btn>
            </div>

            <!-- Auto-Reply Flow integration info -->
            <div class="strategy-summary-card">
              <div class="ssc-flow-row">
                <div class="d-flex align-center">
                  <v-icon icon="mdi-robot-outline" size="18" class="mr-2 text-indigo" />
                  <div>
                    <div class="text-caption font-weight-bold">
                      {{ l({ lo: 'Auto-Reply Flow ທີ່ຜູກໄວ້:', th: 'Auto-Reply Flow ที่ผูกไว้:', en: 'Linked Auto-Reply Flows:' }) }}
                    </div>
                    <div v-if="productAutomations(selectedProductForDetail.id).length" class="text-caption text-indigo">
                      <span v-for="f in productAutomations(selectedProductForDetail.id)" :key="f.id" class="mr-2">
                        🤖 {{ f.name }} ({{ f.targets?.length || 0 }} ແອດ)
                      </span>
                    </div>
                    <div v-else class="text-caption text-muted">
                      {{ l({ lo: 'ຍັງບໍ່ທັນມີ Auto-Reply flow ຜູກກັບສິນຄ້ານີ້', th: 'ยังไม่มี Auto-Reply flow ผูกกับสินค้านี้', en: 'No auto-reply flow linked yet' }) }}
                    </div>
                  </div>
                </div>
                <v-btn
                  v-if="getProductAdCount(selectedProductForDetail.id) > 0"
                  size="x-small"
                  variant="outlined"
                  color="primary"
                  prepend-icon="mdi-bullhorn-outline"
                  @click="goToProductAds(selectedProductForDetail.id)"
                >
                  {{ l({ lo: 'ເບິ່ງແອດທັງໝົດ', th: 'ดูแอดทั้งหมด', en: 'View Linked Ads' }) }}
                </v-btn>
              </div>

              <!-- Metrics Breakdown -->
              <div class="ssc-metrics-grid mt-2">
                <div class="ssc-metric">
                  <span class="label">{{ l({ lo: 'ສະຖານະ Strategy', th: 'สถานะ Strategy', en: 'Strategy Status' }) }}</span>
                  <span class="value" :class="productStrategies[selectedProductForDetail.id]?.enabled ? 'text-success' : 'text-muted'">
                    {{ productStrategies[selectedProductForDetail.id]?.enabled ? l({ lo: 'ເປີດໃຊ້ສະເພາະສິນຄ້າ', th: 'เปิดใช้เฉพาะสินค้า', en: 'Custom Enabled' }) : l({ lo: 'ໃຊ້ຄ່າກາງບັນຊີ', th: 'ใช้ค่ากลางบัญชี', en: 'Account Default' }) }}
                  </span>
                </div>
                <div class="ssc-metric">
                  <span class="label">{{ l({ lo: 'Target CPA (ຈຸດຄຸ້ມທຶນ)', th: 'Target CPA (จุดคุ้มทุน)', en: 'Target CPA' }) }}</span>
                  <span class="value font-weight-bold">
                    {{ formatAmount(productStrategies[selectedProductForDetail.id]?.targetCost, selectedProductForDetail.currency) }}
                  </span>
                </div>
                <div class="ssc-metric">
                  <span class="label">{{ l({ lo: 'Scale Min Orders', th: 'Scale Min Orders', en: 'Scale Min Orders' }) }}</span>
                  <span class="value">
                    {{ productStrategies[selectedProductForDetail.id]?.scaleMinPurchases || 3 }} {{ l({ lo: 'ອໍເດີ', th: 'ออเดอร์', en: 'orders' }) }}
                  </span>
                </div>
                <div class="ssc-metric">
                  <span class="label">{{ l({ lo: 'Scale Max CPA', th: 'Scale Max CPA', en: 'Scale Max CPA' }) }}</span>
                  <span class="value">
                    {{ formatAmount(productStrategies[selectedProductForDetail.id]?.scaleMaxCPA, selectedProductForDetail.currency) }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div v-if="selectedProductForDetail.description" class="detail-desc-box">
            <small class="text-muted">{{ l({ lo: 'ລາຍລະອຽດ:', th: 'รายละเอียด:', en: 'Description:' }) }}</small>
            <p>{{ selectedProductForDetail.description }}</p>
          </div>
        </v-card-text>
        <v-card-actions>
          <v-btn variant="outlined" prepend-icon="mdi-cart-plus" to="/orders">
            {{ l({ lo: 'ໄປຂຶ້ນອໍເດີ', th: 'ไปสร้างออเดอร์', en: 'Go to Orders' }) }}
          </v-btn>
          <v-spacer/>
          <v-btn color="primary" @click="detailDialog = false">{{ l({ lo: 'ປິດ', th: 'ปิด', en: 'Close' }) }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Product Ad Strategy Dialog -->
    <v-dialog v-model="strategyDialog" max-width="680" scrollable>
      <v-card v-if="selectedProductForStrategy" class="dialog-card">
        <v-card-title class="dialog-title d-flex align-center justify-space-between">
          <div class="d-flex align-center">
            <v-icon icon="mdi-target-variant" color="deep-purple" class="mr-2" />
            <div>
              <div class="font-weight-bold text-subtitle-1">
                {{ l({ lo: 'ຍຸດທະສາດໂຄສະນາ & Auto-Pilot ແຍກຕາມສິນຄ້າ', th: 'ยุทธศาสตร์โฆษณา & Auto-Pilot แยกตามสินค้า', en: 'Per-Product Ad Strategy & Auto-Pilot' }) }}
              </div>
              <div class="text-caption text-muted font-weight-normal">
                {{ l({ lo: 'ຕັ້ງເກນ CPA, ເພດານງົບ ແລະ ເງື່ອນໄຂ Scale/Stop ໃຫ້ແທດເໝາະກັບກຳໄລຂອງສິນຄ້ານີ້', th: 'ตั้งเกณฑ์ CPA, เพดานงบ และเงื่อนไข Scale/Stop ให้ตรงกับกำไรของสินค้านี้', en: 'Tune CPA target, budget cap, and scale/stop rules tailored to this product’s margin.' }) }}
              </div>
            </div>
          </div>
          <v-btn icon="mdi-close" variant="text" size="small" @click="strategyDialog = false" />
        </v-card-title>

        <v-divider />

        <v-card-text style="max-height: 75vh; overflow-y: auto;" class="pa-4">
          <!-- Product Context Banner -->
          <div class="product-strategy-banner mb-4">
            <div
              class="banner-img"
              :style="selectedProductForStrategy.imageUrl ? `background-image: url(${selectedProductForStrategy.imageUrl}); background-size: cover; background-position: center;` : ''"
            >
              <v-icon v-if="!selectedProductForStrategy.imageUrl" icon="mdi-package-variant-closed" size="24" />
            </div>
            <div class="banner-info">
              <div class="d-flex align-center justify-space-between">
                <strong class="text-body-1">{{ selectedProductForStrategy.name }}</strong>
                <span v-if="selectedProductForStrategy.code" class="tag-pill tag-code">#{{ selectedProductForStrategy.code }}</span>
              </div>
              <div class="banner-numbers mt-1">
                <span>
                  <small>{{ l({ lo: 'ລາຄາຂາຍ:', th: 'ราคาขาย:', en: 'Price:' }) }}</small>
                  <strong>{{ selectedProductForStrategy.salePrice != null ? formatAmount(selectedProductForStrategy.salePrice, selectedProductForStrategy.currency) : selectedProductForStrategy.price }}</strong>
                </span>
                <span>
                  <small>{{ l({ lo: 'ຕົ້ນທຶນ:', th: 'ต้นทุน:', en: 'Cost:' }) }}</small>
                  <strong>{{ formatAmount(selectedProductForStrategy.cost, selectedProductForStrategy.currency) }}</strong>
                </span>
                <span>
                  <small>{{ l({ lo: 'ກຳໄລ/ໜ່ວຍ:', th: 'กำไร/หน่วย:', en: 'Profit/unit:' }) }}</small>
                  <strong class="text-success">{{ formatAmount(unitProfit(selectedProductForStrategy), selectedProductForStrategy.currency) }}</strong>
                </span>
              </div>
            </div>
          </div>

          <!-- Linked Auto-Reply Flows & Ads Banner -->
          <div class="flow-sync-banner mb-4">
            <div class="d-flex align-center justify-space-between">
              <div class="d-flex align-center">
                <v-icon icon="mdi-robot-outline" color="indigo" class="mr-2" size="20" />
                <div>
                  <div class="text-caption font-weight-bold">
                    {{ l({ lo: 'ການຜູກກັບ Auto-Reply & Meta Ads', th: 'การผูกกับ Auto-Reply & Meta Ads', en: 'Linked Auto-Reply & Meta Ads' }) }}
                  </div>
                  <div class="text-caption text-muted">
                    <template v-if="productAutomations(selectedProductForStrategy.id).length">
                      {{ l({ lo: 'ສິນຄ້ານີ້ຜູກກັບ', th: 'สินค้านี้ผูกกับ', en: 'Linked to' }) }}
                      <strong>{{ productAutomations(selectedProductForStrategy.id).length }} Flow</strong>
                      ({{ getProductAdCount(selectedProductForStrategy.id) }} {{ l({ lo: 'ແອດໂຄສະນາ', th: 'แอดโฆษณา', en: 'linked ads' }) }})
                    </template>
                    <template v-else>
                      {{ l({ lo: 'ຍັງບໍ່ມີ Auto-Reply flow ຜູກກັບສິນຄ້ານີ້. ສາມາດຜູກໄດ້ທີ່ເມນູ Auto Replies.', th: 'ยังไม่มี Auto-Reply flow ผูกกับสินค้านี้ ผูกได้ที่เมนู Auto Replies', en: 'No auto-reply flow linked yet. Connect in Auto Replies.' }) }}
                    </template>
                  </div>
                </div>
              </div>
              <v-btn
                v-if="getProductAdCount(selectedProductForStrategy.id) > 0"
                variant="tonal"
                color="primary"
                size="small"
                prepend-icon="mdi-bullhorn-outline"
                @click="goToProductAds(selectedProductForStrategy.id)"
              >
                {{ l({ lo: 'ເບິ່ງແອດຂອງສິນຄ້ານີ້', th: 'ดูแอดของสินค้านี้', en: 'View Product Ads' }) }}
              </v-btn>
              <v-btn
                v-else
                variant="text"
                color="indigo"
                size="small"
                prepend-icon="mdi-plus"
                to="/auto-replies"
              >
                {{ l({ lo: 'ຜູກ Auto-Reply', th: 'ผูก Auto-Reply', en: 'Link Auto-Reply' }) }}
              </v-btn>
            </div>
          </div>

          <!-- Strategy Enable & Auto-Calculate Bar -->
          <div class="d-flex align-center justify-space-between mb-4 pa-3 bg-surface-variant rounded-lg">
            <div>
              <v-switch
                v-model="strategyForm.enabled"
                color="deep-purple"
                hide-details
                density="compact"
                :label="l({ lo: 'ເປີດໃຊ້ຍຸດທະສາດສະເພາະສິນຄ້ານີ້ (Enable Custom Strategy)', th: 'เปิดใช้ยุทธศาสตร์เฉพาะสินค้านี้', en: 'Enable Custom Strategy for this Product' })"
              />
              <div class="text-caption text-muted ml-8">
                {{ strategyForm.enabled ? l({ lo: 'ລະບົບຈະໃຊ້ເກນເຫຼົ່ານີ້ແທນຄ່າກາງບັນຊີ ເມື່ອປະເມີນແອດຂອງສິນຄ້ານີ້', th: 'ระบบจะใช้เกณฑ์เหล่านี้แทนค่ากลางบัญชี เมื่อประเมินแอดของสินค้านี้', en: 'Rules will override account defaults when evaluating ads for this product.' }) : l({ lo: 'ປິດຢູ່ (ແອດຂອງສິນຄ້ານີ້ຈະໃຊ້ຄ່າກາງບັນຊີ Ads)', th: 'ปิดอยู่ (แอดของสินค้านี้จะใช้ค่ากลางบัญชี Ads)', en: 'Disabled (Will fallback to account default settings).' }) }}
              </div>
            </div>
            <v-btn
              variant="flat"
              color="deep-purple"
              size="small"
              prepend-icon="mdi-calculator-variant"
              @click="autoCalcCurrentStrategy"
            >
              {{ l({ lo: 'ຄິດໄລ່ອັດຕະໂນມັດ', th: 'คำนวณอัตโนมัติ', en: 'Auto-Calculate' }) }}
            </v-btn>
          </div>

          <!-- Form Fields Grid -->
          <div class="strategy-form-grid" :class="{ 'opacity-50 pointer-events-none': !strategyForm.enabled }">
            <!-- 1. Target CPA -->
            <div class="form-group-item">
              <label class="form-label">
                <v-icon icon="mdi-bullseye-arrow" size="14" class="mr-1 text-primary" />
                {{ l({ lo: 'Target CPA (ຕົ້ນທຶນຕໍ່ອໍເດີທີ່ຕ້ອງການ)', th: 'Target CPA (ต้นทุนต่อออเดอร์ที่ต้องการ)', en: 'Target CPA (Cost/Purchase)' }) }}
              </label>
              <v-text-field
                v-model.number="strategyForm.targetCost"
                type="number"
                variant="outlined"
                density="compact"
                hide-details
                :suffix="selectedProductForStrategy.currency || 'LAK'"
                placeholder="20% ຂອງລາຄາຂາຍ"
              />
              <span class="field-hint">
                {{ l({ lo: 'ເພດານຕົ້ນທຶນໂຄສະນາຕໍ່ 1 ອໍເດີ ທີ່ຖືວ່າຄຸ້ມຄ່າກຳໄລ', th: 'เพดานต้นทุนโฆษณาต่อ 1 ออเดอร์ที่ถือว่าคุ้มค่ากำไร', en: 'Target acquisition cost per purchase for healthy profit' }) }}
              </span>
            </div>

            <!-- 2. Gross Margin % -->
            <div class="form-group-item">
              <label class="form-label">
                <v-icon icon="mdi-percent" size="14" class="mr-1 text-teal" />
                {{ l({ lo: 'Gross Profit Margin % (ອັດຕາກຳໄລຂັ້ນຕົ້ນ)', th: 'Gross Profit Margin % (อัตรากำไรขั้นต้น)', en: 'Gross Profit Margin %' }) }}
              </label>
              <v-text-field
                v-model.number="strategyForm.grossMargin"
                type="number"
                variant="outlined"
                density="compact"
                hide-details
                suffix="%"
                placeholder="50"
              />
              <span class="field-hint">
                {{ l({ lo: 'ໃຊ້ຄຳນວນຈຸດຄຸ້ມທຶນ Break-Even ROAS ຕົວຈິງ', th: 'ใช้คำนวณจุดคุ้มทุน Break-Even ROAS ตัวจริง', en: 'Used to calculate Break-Even ROAS' }) }}
              </span>
            </div>

            <!-- 3. Scale Min Purchases -->
            <div class="form-group-item">
              <label class="form-label">
                <v-icon icon="mdi-cart-check" size="14" class="mr-1 text-success" />
                {{ l({ lo: 'Scale Min Purchases (ອໍເດີຂັ້ນຕ່ຳເພື່ອ Scale)', th: 'Scale Min Purchases (ออเดอร์ขั้นต่ำเพื่อ Scale)', en: 'Scale Min Purchases' }) }}
              </label>
              <v-text-field
                v-model.number="strategyForm.scaleMinPurchases"
                type="number"
                variant="outlined"
                density="compact"
                hide-details
                :suffix="l({ lo: 'ອໍເດີ', th: 'ออเดอร์', en: 'orders' })"
                placeholder="3"
              />
              <span class="field-hint">
                {{ l({ lo: 'ແອດຕ້ອງມີອໍເດີຂັ້ນຕ່ຳເທົ່ານີ້ຈຶ່ງອະນຸຍາດໃຫ້ເພີ່ມງົບ (Scale)', th: 'แอดต้องมีออเดอร์ขั้นต่ำเท่านี้จึงอนุญาตให้เพิ่มงบ (Scale)', en: 'Minimum verified purchases before Auto-Pilot scales budget' }) }}
              </span>
            </div>

            <!-- 4. Scale Max CPA -->
            <div class="form-group-item">
              <label class="form-label">
                <v-icon icon="mdi-currency-usd" size="14" class="mr-1 text-warning" />
                {{ l({ lo: 'Scale Max CPA (CPA ສູງສຸດທີ່ອະນຸຍາດ Scale)', th: 'Scale Max CPA (CPA สูงสุดที่อนุญาต Scale)', en: 'Scale Max CPA' }) }}
              </label>
              <v-text-field
                v-model.number="strategyForm.scaleMaxCPA"
                type="number"
                variant="outlined"
                density="compact"
                hide-details
                :suffix="selectedProductForStrategy.currency || 'LAK'"
                placeholder="25% ຂອງລາຄາຂາຍ"
              />
              <span class="field-hint">
                {{ l({ lo: 'ຖ້າ CPA ສູງເກີນຄ່ານີ້ ລະບົບຈະບໍ່ເພີ່ມງົບໃຫ້ ເພື່ອປ້ອງກັນຂາດທຶນ', th: 'ถ้า CPA สูงเกินค่านี้ ระบบจะไม่เพิ่มงบให้ เพื่อป้องกันขาดทุน', en: 'Do not scale if CPA exceeds this threshold' }) }}
              </span>
            </div>

            <!-- 5. Max Daily Budget Cap -->
            <div class="form-group-item">
              <label class="form-label">
                <v-icon icon="mdi-shield-lock-outline" size="14" class="mr-1 text-info" />
                {{ l({ lo: 'Max Daily Budget Cap (ເພດານງົບສູງສຸດຕໍ່ມື້)', th: 'Max Daily Budget Cap (เพดานงบสูงสุดต่อวัน)', en: 'Max Daily Budget Cap' }) }}
              </label>
              <v-text-field
                v-model.number="strategyForm.scaleMaxDailyBudgetCap"
                type="number"
                variant="outlined"
                density="compact"
                hide-details
                :suffix="selectedProductForStrategy.currency || 'LAK'"
                placeholder="ບໍ່ຈຳກັດ"
              />
              <span class="field-hint">
                {{ l({ lo: 'ເພດານງົບສູງສຸດຕໍ່ມື້ ປ້ອງກັນບໍ່ໃຫ້ແອດ Scale ງົບເກີນກຳນົດ', th: 'เพดานงบสูงสุดต่อวัน ป้องกันไม่ให้แอด Scale งบเกินกำหนด', en: 'Hard cap on daily budget during automatic scaling' }) }}
              </span>
            </div>

            <!-- 6. Stop-Loss Spend -->
            <div class="form-group-item">
              <label class="form-label">
                <v-icon icon="mdi-hand-back-left" size="14" class="mr-1 text-error" />
                {{ l({ lo: 'Stop-Loss Spend (ງົບຮອດນີ້ແລ້ວບໍ່ມີອໍເດີໃຫ້ຢຸດ)', th: 'Stop-Loss Spend (งบถึงนี้แล้วไม่มีออเดอร์ให้หยุด)', en: 'Stop-Loss Spend (Pause threshold)' }) }}
              </label>
              <v-text-field
                v-model.number="strategyForm.stopMinSpend"
                type="number"
                variant="outlined"
                density="compact"
                hide-details
                :suffix="selectedProductForStrategy.currency || 'LAK'"
                placeholder="80% ຂອງລາຄາຂາຍ"
              />
              <span class="field-hint">
                {{ l({ lo: 'ຖ້າແອດໃຊ້ງົບຮອດນີ້ແລ້ວຍັງບໍ່ມີອໍເດີ (0 purchases) ຈະແນະນຳໃຫ້ຢຸດ (Pause)', th: 'ถ้าแอดใช้งบถึงนี้แล้วยังไม่มีออเดอร์ (0 purchases) จะแนะนำให้หยุด (Pause)', en: 'Recommend pause if spent this amount with zero purchases' }) }}
              </span>
            </div>
          </div>
        </v-card-text>

        <v-divider />

        <v-card-actions class="pa-3">
          <v-btn
            v-if="selectedProductForStrategy"
            variant="text"
            color="error"
            size="small"
            prepend-icon="mdi-restore"
            @click="resetAdStrategy"
          >
            {{ l({ lo: 'ລ້າງຄ່າ (ໃຊ້ຄ່າກາງ)', th: 'ล้างค่า (ใช้ค่ากลาง)', en: 'Reset to Default' }) }}
          </v-btn>
          <v-spacer />
          <v-btn variant="text" @click="strategyDialog = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </v-btn>
          <v-btn
            color="deep-purple"
            variant="flat"
            :loading="strategySaving"
            prepend-icon="mdi-content-save"
            @click="saveAdStrategy"
          >
            {{ l({ lo: '💾 ບັນທຶກຍຸດທະສາດ', th: '💾 บันทึกยุทธศาสตร์', en: '💾 Save Strategy' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="500">
      <v-card class="dialog-card">
        <v-card-title>{{ l({ lo: 'ລຶບສິນຄ້າ?', th: 'ลบสินค้า?', en: 'Delete product?' }) }}</v-card-title>
        <v-card-text>
          <template v-if="productToDelete && productAutomations(productToDelete.id).length">
            <v-alert type="warning" variant="tonal" class="mb-3">
              {{ l({ lo: 'ຍັງລຶບບໍ່ໄດ້ ເພາະສິນຄ້ານີ້ຖືກໃຊ້ໃນ Automation', th: 'ยังลบไม่ได้ เพราะสินค้านี้ถูกใช้ใน Automation', en: 'This product cannot be deleted while an Automation uses it.' }) }}
            </v-alert>
            <div class="dependency-list">
              <NuxtLink v-for="flow in productAutomations(productToDelete.id)" :key="flow.id" to="/auto-replies">
                <v-icon icon="mdi-robot-outline"/>
                <span>
                  <strong>{{ flow.name }}</strong>
                  <small>{{ flow.targets.length }} {{ l({ lo: 'ເປົ້າໝາຍ', th: 'เป้าหมาย', en: 'targets' }) }}</small>
                </span>
                <v-icon icon="mdi-arrow-right"/>
              </NuxtLink>
            </div>
          </template>
          <p v-else>
            {{ l({ lo: `ຢືນຢັນການລຶບ “${productToDelete?.name || ''}”. ການກະທຳນີ້ຍ້ອນກັບບໍ່ໄດ້.`, th: `ยืนยันการลบ “${productToDelete?.name || ''}” การดำเนินการนี้ย้อนกลับไม่ได้`, en: `Delete “${productToDelete?.name || ''}”? This action cannot be undone.` }) }}
          </p>
        </v-card-text>
        <v-card-actions>
          <v-btn variant="text" @click="deleteDialog = false">{{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}</v-btn>
          <v-spacer/>
          <v-btn
            v-if="productToDelete && !productAutomations(productToDelete.id).length"
            color="error"
            :loading="deleting"
            @click="deleteProduct"
          >
            {{ l({ lo: 'ລຶບສິນຄ້າ', th: 'ลบสินค้า', en: 'Delete product' }) }}
          </v-btn>
          <v-btn v-else color="primary" to="/auto-replies">
            {{ l({ lo: 'ໄປຈັດການ Automation', th: 'ไปจัดการ Automation', en: 'Manage Automations' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.products-page {
  padding-bottom: 40px;
}
.product-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.product-search-box {
  position: relative;
  display: flex;
  align-items: center;
  max-width: 420px;
  width: 100%;
}
.product-search-box input {
  width: 100%;
  height: 40px;
  border: 1px solid var(--color-border);
  border-radius: 20px;
  padding: 0 32px 0 36px;
  font-size: 13px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
  transition: border-color .15s ease;
}
.product-search-box input:focus {
  border-color: var(--color-primary);
}
.product-search-box .search-icon {
  position: absolute;
  left: 12px;
  color: var(--color-text-muted);
  pointer-events: none;
}
.product-search-box .clear-btn {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
}
.product-count-chip {
  padding: 6px 14px;
  border-radius: 20px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  font-size: 12px;
  font-weight: 750;
  color: var(--color-text-secondary);
}
.product-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.product-title-link {
  cursor: pointer;
}
.product-title-link:hover {
  color: var(--color-primary);
}
.price-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  background: #f0fdf4;
  color: #16a34a;
  border: 1px solid #bbf7d0;
  font-size: 12.5px;
  font-weight: 800;
}
.product-tags-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin: 4px 0 6px;
}
.tag-pill {
  font-size: 12px;
  font-weight: 700;
  padding: 4px 8px;
  line-height: 1.4;
  border-radius: 8px;
}
.tag-category {
  background: #eff6ff;
  color: #2563eb;
  border: 1px solid #dbeafe;
}
.tag-barcode {
  background: #f8fafc;
  color: #64748b;
  border: 1px solid #e2e8f0;
  display: inline-flex;
  align-items: center;
}
.tag-unit {
  background: #fdf4ff;
  color: #a855f7;
  border: 1px solid #f3e8ff;
}
.tag-autoreply {
  background: #eef2ff;
  color: #4338ca;
  border: 1px solid #c7d2fe;
  display: inline-flex;
  align-items: center;
  transition: all .15s ease;
}
.tag-autoreply:hover {
  background: #e0e7ff;
}
.tag-strategy {
  background: #faf5ff;
  color: #6b21a8;
  border: 1px solid #e9d5ff;
  display: inline-flex;
  align-items: center;
  transition: all .15s ease;
}
.tag-strategy:hover {
  background: #f3e8ff;
}
.ad-count-sub {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 2px;
}
.code-copy-pill {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px 9px;
  border-radius: 12px;
  background: var(--color-primary-soft);
  color: var(--color-primary);
  border: 1px solid color-mix(in srgb, var(--color-primary) 30%, transparent);
  font-size: 12px;
  cursor: pointer;
  transition: all .15s ease;
}
.code-copy-pill:hover {
  background: var(--color-primary);
  color: #fff;
}
.inventory-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
  margin-top: 10px;
}
.inventory-row > span {
  display: grid;
  padding: 8px;
  border-radius: 9px;
  background: var(--color-background);
}
.inventory-row small {
  font-size: 12px;
}
.inventory-row strong {
  font-size: 12px;
  overflow-wrap: anywhere;
}
.money-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 10px;
}
.fields-2col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.fields-3col {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 10px;
}
.image-upload-wrapper {
  display: flex;
  gap: 16px;
  align-items: center;
}
.image-preview {
  width: 80px;
  height: 80px;
  border-radius: 12px;
  background-color: var(--color-surface-variant);
  background-size: cover;
  background-position: center;
  display: grid;
  place-items: center;
  position: relative;
  border: 1px solid var(--color-border);
}
.upload-loader {
  position: absolute;
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
  font-size: 12px;
  font-weight: 800;
  letter-spacing: .13em;
}
.page-head h2 {
  margin: 0 0 7px;
  font-size: 24px;
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
.product-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.product-card {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 15px;
  align-items: start;
  padding: 20px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.product-icon {
  width: 56px;
  height: 56px;
  display: grid;
  place-items: center;
  color: var(--color-primary);
  background: var(--color-primary-soft);
  border-radius: 14px;
  flex-shrink: 0;
  cursor: pointer;
}
.product-actions {
  display: flex;
  gap: 2px;
}
.product-card h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 750;
}
.product-desc {
  min-height: 28px;
  margin: 4px 0 6px;
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.4;
}
.product-card small {
  color: var(--color-text-muted);
  font-size: 12px;
}

.product-mini-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--color-surface-variant);
  border-radius: 8px;
}
.warehouse-stocks-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.stock-location-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}
.location-info {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}
.badge-mini-def {
  font-size: 12px;
  background: #fef3c7;
  color: #b45309;
  padding: 1px 6px;
  border-radius: 6px;
  font-weight: 700;
}
.location-qty {
  display: flex;
  align-items: center;
  gap: 4px;
}
.stock-number {
  font-size: 15px;
  font-weight: 800;
  color: var(--color-primary);
}
.adjust-box {
  padding: 14px;
  background: var(--color-surface-variant);
  border-radius: 10px;
}

.detail-top-card {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--color-border);
}
.detail-img {
  width: 72px;
  height: 72px;
  border-radius: 12px;
  background-color: var(--color-surface-variant);
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-border);
}
.detail-headings h3 {
  margin: 0 0 6px;
  font-size: 18px;
  font-weight: 800;
}
.detail-price-badge {
  display: inline-block;
  padding: 3px 10px;
  background: #f0fdf4;
  color: #16a34a;
  border: 1px solid #bbf7d0;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 800;
}
.detail-specs-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}
.spec-cell {
  display: flex;
  flex-direction: column;
  padding: 8px 10px;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}
.spec-cell small {
  font-size: 12px;
  color: var(--color-text-muted);
}
.spec-cell strong {
  font-size: 12.5px;
  color: var(--color-text);
}
.detail-stocks-section h4 {
  font-size: 13px;
  font-weight: 750;
}
.stocks-table-mini {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.stock-mini-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 10px;
  background: var(--color-surface-variant);
  border-radius: 6px;
  font-size: 12.5px;
}
.detail-desc-box {
  padding: 10px;
  background: var(--color-surface-variant);
  border-radius: 8px;
  font-size: 12.5px;
}
.detail-desc-box p {
  margin: 4px 0 0;
}

.dependency-list {
  display: grid;
  gap: 8px;
}
.dependency-list a {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 11px;
  color: var(--color-text);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  text-decoration: none;
}
.dependency-list a span {
  display: grid;
}
.dependency-list small {
  color: var(--color-text-secondary);
}
.empty-state {
  min-height: 390px;
  display: grid;
  place-items: center;
  align-content: center;
  text-align: center;
  padding: 36px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.empty-state > div {
  width: 58px;
  height: 58px;
  display: grid;
  place-items: center;
  color: var(--color-primary);
  background: var(--color-primary-soft);
  border-radius: 17px;
}
.empty-state h2 {
  margin: 18px 0 6px;
}
.empty-state p {
  max-width: 480px;
  margin: 0 0 20px;
  color: var(--color-text-secondary);
}
.dialog-card {
  border-radius: var(--radius-lg) !important;
}
.form {
  display: grid;
  gap: 4px;
  padding-top: 18px !important;
}
.inline-action {
  border: 0;
  color: var(--color-primary);
  background: none;
  font-weight: 800;
  cursor: pointer;
}
.text-red {
  color: #ef4444 !important;
}

@media (max-width: 760px) {
  .product-grid {
    grid-template-columns: 1fr;
  }
  .detail-specs-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
@media (max-width: 640px) {
  .page-head {
    display: grid;
    padding: 16px;
    border-radius: 18px;
  }
  .head-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    width: 100%;
    gap: 8px;
  }
  .product-card {
    padding: 14px 16px;
    border-radius: 16px;
    gap: 12px;
  }
  .product-icon {
    width: 48px;
    height: 48px;
    border-radius: 12px;
  }
  .image-upload-wrapper {
    flex-direction: column;
    align-items: flex-start;
  }
  .money-grid, .fields-2col {
    grid-template-columns: 1fr;
  }
}

/* Variant Builder & Table Styles */
.tag-color {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fee2e2;
  display: inline-flex;
  align-items: center;
}
.tag-size {
  background: #f0fdf4;
  color: #16a34a;
  border: 1px solid #dcfce7;
  display: inline-flex;
  align-items: center;
}
.tag-variants {
  background: #faf5ff;
  color: #7e22ce;
  border: 1px solid #f3e8ff;
  display: inline-flex;
  align-items: center;
}
.variant-builder-box {
  background: var(--color-background);
  border: 1px dashed var(--color-border);
  border-radius: 12px;
  padding: 12px 14px;
}
.variant-actions-bar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.variant-matrix-table {
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  background: var(--color-surface);
  margin-top: 8px;
}
.variant-table-header {
  display: grid;
  grid-template-columns: 1.2fr 1.2fr 1fr 1.2fr 36px;
  gap: 6px;
  padding: 8px 10px;
  background: var(--color-background);
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-secondary);
  border-bottom: 1px solid var(--color-border);
  min-width: 420px;
}
.variant-table-row {
  display: grid;
  grid-template-columns: 1.2fr 1.2fr 1fr 1.2fr 36px;
  gap: 6px;
  align-items: center;
  padding: 6px 10px;
  border-bottom: 1px solid var(--color-border);
  min-width: 420px;
}
.variant-table-row:last-child {
  border-bottom: none;
}
.variant-input {
  width: 100%;
  padding: 5px 8px;
  font-size: 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-background);
  color: var(--color-text);
  outline: none;
}
.variant-input:focus {
  border-color: var(--color-primary);
}
.qty-box {
  font-weight: 700;
  color: var(--color-primary);
}
.btn-del-variant {
  border: none;
  background: none;
  color: #ef4444;
  cursor: pointer;
  padding: 4px;
  display: grid;
  place-items: center;
  border-radius: 4px;
}
.btn-del-variant:hover {
  background: #fee2e2;
}
.variant-table-footer {
  padding: 8px 10px;
  background: var(--color-background);
  border-top: 1px dashed var(--color-border);
  text-align: right;
}

/* Detail Modal Variants Table */
.detail-variants-section {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 12px;
}
.detail-variants-table {
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-surface);
}
.dvt-header {
  display: grid;
  grid-template-columns: 1.2fr 1.2fr 1.2fr 1fr;
  padding: 8px 12px;
  background: var(--color-background);
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-secondary);
  border-bottom: 1px solid var(--color-border);
}
.dvt-row {
  display: grid;
  grid-template-columns: 1.2fr 1.2fr 1.2fr 1fr;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
  font-size: 12px;
}
.dvt-row:last-child {
  border-bottom: none;
}
.dvt-color {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}
.dot-swatch {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
}
.size-pill {
  display: inline-block;
  padding: 2px 7px;
  background: #f0fdf4;
  color: #16a34a;
  border-radius: 6px;
  font-size: 12px;
}
.qty-badge {
  display: inline-block;
  padding: 2px 8px;
  background: #eff6ff;
  color: #2563eb;
  border-radius: 12px;
  font-weight: 700;
  font-size: 12px;
}
.qty-badge.zero-stock {
  background: #fee2e2;
  color: #ef4444;
}

/* Product Ad Strategy styles */
.detail-strategy-section h4 {
  font-size: 13px;
  font-weight: 750;
}
.strategy-summary-card {
  padding: 10px 12px;
  background: var(--color-surface-variant);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}
.ssc-flow-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}
.ssc-metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}
@media (max-width: 600px) {
  .ssc-metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
.ssc-metric {
  display: flex;
  flex-direction: column;
  padding: 6px 8px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}
.ssc-metric .label {
  font-size: 12px;
  color: var(--color-text-muted);
}
.ssc-metric .value {
  font-size: 12px;
  color: var(--color-text);
}

.product-strategy-banner {
  display: flex;
  gap: 12px;
  padding: 12px;
  background: var(--color-surface-variant);
  border: 1px solid var(--color-border);
  border-radius: 10px;
}
.product-strategy-banner .banner-img {
  width: 64px;
  height: 64px;
  border-radius: 8px;
  background-color: var(--color-background);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.product-strategy-banner .banner-info {
  flex: 1;
  min-width: 0;
}
.banner-numbers {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}
.banner-numbers > span {
  display: flex;
  flex-direction: column;
  font-size: 12px;
}
.banner-numbers small {
  color: var(--color-text-muted);
  font-size: 12px;
}
.flow-sync-banner {
  padding: 10px 12px;
  background: rgba(99, 102, 241, 0.08);
  border: 1px solid rgba(99, 102, 241, 0.25);
  border-radius: 8px;
}
.strategy-form-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}
@media (max-width: 600px) {
  .strategy-form-grid {
    grid-template-columns: 1fr;
  }
}
.form-group-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.form-label {
  font-size: 12px;
  font-weight: 700;
  display: flex;
  align-items: center;
  color: var(--color-text);
}
.field-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  line-height: 1.3;
}
.tag-code {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  border: 1px solid color-mix(in srgb, var(--color-primary) 30%, transparent);
}
</style>
