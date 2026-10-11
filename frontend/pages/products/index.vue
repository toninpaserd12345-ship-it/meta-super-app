<script setup lang="ts">
import type { AutomationFlow, ItemsResponse, Product } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

const { can } = useAuth()
const { locale, l } = useLocale()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Products.' })

const { data, pending, error, refresh } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: automationData } = await useApi<ItemsResponse<AutomationFlow>>('/proxy/api/v1/automations')
const products = computed(() => data.value?.items || [])
const automations = computed(() => automationData.value?.items || [])
const dialog = ref(false)
const saving = ref(false)
const uploading = ref(false)
const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')
const form = reactive<Product & { code?: string }>({ id: '', code: '', name: '', price: '', description: '', imageUrl: '' })
const deleteDialog = ref(false)
const deleting = ref(false)
const productToDelete = ref<Product | null>(null)

const fileInput = ref<HTMLInputElement>()
const searchQuery = ref('')
const copiedCodeId = ref<string | null>(null)

const filteredProducts = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return products.value
  return products.value.filter(p => 
    p.name.toLowerCase().includes(q) ||
    (p.code && p.code.toLowerCase().includes(q)) ||
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
  Object.assign(form, { id: '', code: '', name: '', price: '', description: '', imageUrl: '' })
  dialog.value = true
}

function openEdit(product: Product) {
  Object.assign(form, product)
  dialog.value = true
}

const productAutomations = (productId: string) => automations.value.filter(item => item.productId === productId)

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
    input.value = '' // reset
  }
}

async function saveProduct() {
  if (!form.name.trim() || !form.price.trim()) return
  saving.value = true
  notice.value = ''
  try {
    await $fetch('/api/proxy/api/v1/products', {
      method: 'POST',
      body: {
        id: form.id || undefined,
        code: form.code?.trim() || undefined,
        name: form.name.trim(),
        price: form.price.trim(),
        description: form.description.trim(),
        imageUrl: form.imageUrl?.trim() || undefined,
      },
    })
    dialog.value = false
    noticeType.value = 'success'
    notice.value = form.id
      ? l({ lo: 'ອັບເດດສິນຄ້າແລ້ວ', th: 'อัปเดตสินค้าแล้ว', en: 'Product updated.' })
      : l({ lo: 'ສ້າງສິນຄ້າແລ້ວ ແລະພ້ອມໃຊ້ກັບ Auto Reply', th: 'สร้างสินค้าแล้วและพร้อมใช้กับ Auto Reply', en: 'Product created and ready for Auto Replies.' })
    await refresh()
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, l({ lo: 'ບັນທຶກສິນຄ້າບໍ່ສຳເລັດ', th: 'บันทึกสินค้าไม่สำเร็จ', en: 'Unable to save the product.' }))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="page-head">
    <div><p>PRODUCT CATALOG</p><h2>{{ l({lo:'ຈັດການສິນຄ້າ',th:'จัดการสินค้า',en:'Products'}) }}</h2><span>{{ l({lo:'ເກັບຊື່, ລາຄາ, ຮູບ ແລະລາຍລະອຽດເພື່ອໃຊ້ຮ່ວມກັບຊຸດຂໍ້ຄວາມ',th:'เก็บชื่อ ราคา รูปภาพ และรายละเอียดไว้ใช้กับชุดข้อความอัตโนมัติ',en:'Keep product details in one place and reuse them in automatic replies.'}) }}</span></div>
    <div class="head-actions"><v-btn variant="outlined" prepend-icon="mdi-robot-happy-outline" to="/auto-replies">{{ l({lo:'ຈັດການ Automation',th:'จัดการ Automation',en:'Manage Automations'}) }}</v-btn><v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreate">{{ l({lo:'ເພີ່ມສິນຄ້າ',th:'เพิ่มสินค้า',en:'Add product'}) }}</v-btn></div>
  </section>

  <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>
  <v-alert v-if="error" type="error" variant="tonal" class="mb-4">{{ l({lo:'ໂຫຼດສິນຄ້າບໍ່ສຳເລັດ',th:'โหลดสินค้าไม่สำเร็จ',en:'Products could not be loaded.'}) }} <button class="inline-action" @click="refresh()">{{ l({lo:'ລອງໃໝ່',th:'ลองอีกครั้ง',en:'Try again'}) }}</button></v-alert>

  <!-- Product Search & Filter Toolbar -->
  <div class="product-toolbar mb-4">
    <div class="product-search-box">
      <v-icon icon="mdi-magnify" size="18" class="search-icon"/>
      <input 
        v-model="searchQuery" 
        type="text" 
        :placeholder="l({lo:'ຄົ້ນຫາຊື່ສິນຄ້າ, ໂຄ້ດ (#P01), ລາຄາ...',th:'ค้นหาชื่อสินค้า, โค้ด (#P01), ราคา...',en:'Search product name, code (#P01), price...'})" 
      />
      <button v-if="searchQuery" class="clear-btn" @click="searchQuery = ''"><v-icon icon="mdi-close" size="14"/></button>
    </div>
    <div class="product-count-chip">
      {{ filteredProducts.length }} {{ l({lo:'ລາຍການ',th:'รายการ',en:'items'}) }}
    </div>
  </div>

  <section v-if="pending" class="product-grid"><v-skeleton-loader v-for="index in 3" :key="index" type="card"/></section>
  <section v-else-if="filteredProducts.length" class="product-grid">
    <article v-for="product in filteredProducts" :key="product.id" class="product-card">
      <div class="product-icon" :style="product.imageUrl ? `background-image: url(${product.imageUrl}); background-size: cover; background-position: center;` : ''">
        <v-icon v-if="!product.imageUrl" icon="mdi-package-variant-closed"/>
      </div>
      <div class="product-main-details">
        <div class="product-title-row">
          <h3>{{ product.name }}</h3>
          <span class="price-tag">{{ product.price }}</span>
        </div>
        <p v-if="product.description">{{ product.description }}</p>
        <div class="product-meta-row">
          <button 
            v-if="product.code" 
            type="button" 
            class="code-copy-pill" 
            @click="copyCode(product.code, product.id)" 
            :title="l({lo:'ຄລິກເພື່ອຄັດລອກໂຄ້ດ',th:'คลิกเพื่อคัดลอกโค้ด',en:'Click to copy code'})"
          >
            <v-icon icon="mdi-lightning-bolt" size="13"/>
            <strong>#{{ product.code }}</strong>
            <v-icon :icon="copiedCodeId === product.id ? 'mdi-check' : 'mdi-content-copy'" size="12" class="ml-1"/>
          </button>
          <small v-else class="text-muted">ID · {{ product.id.slice(0, 8) }}</small>
        </div>
      </div>
      <div class="product-actions">
        <v-btn 
          icon="mdi-robot-happy-outline" 
          variant="text" 
          size="small" 
          color="primary"
          to="/auto-replies"
          :title="l({lo:'ໄປທີ່ Auto-Reply Flow',th:'ไปที่ Auto-Reply Flow',en:'Go to Auto-Reply Flow'})"
        />
        <v-btn icon="mdi-pencil-outline" variant="text" size="small" :disabled="!can('pages:connect')" :aria-label="`${l({lo:'ແກ້ໄຂ',th:'แก้ไข',en:'Edit'})} ${product.name}`" @click="openEdit(product)"/>
        <v-btn icon="mdi-delete-outline" variant="text" size="small" color="error" :disabled="!can('pages:connect')" :aria-label="`${l({lo:'ລຶບ',th:'ลบ',en:'Delete'})} ${product.name}`" @click="requestDelete(product)"/>
      </div>
    </article>
  </section>
  <section v-else-if="!error" class="empty-state"><div><v-icon icon="mdi-package-variant-plus" size="30"/></div><h2>{{ l({lo:'ເພີ່ມສິນຄ້າທຳອິດ',th:'เพิ่มสินค้าชิ้นแรก',en:'Add the first product'}) }}</h2><p>{{ l({lo:'ສິນຄ້າຈະເກັບຊື່, ລາຄາ, ຮູບ ແລະລາຍລະອຽດທີ່ໃຊ້ໃນຊຸດຂໍ້ຄວາມ',th:'สินค้าจะเก็บชื่อ ราคา รูปภาพ และรายละเอียดที่ใช้ในชุดข้อความ',en:'A product provides the name, price, image and details inserted into your Reply Sets.'}) }}</p><v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreate">{{ l({lo:'ເພີ່ມສິນຄ້າ',th:'เพิ่มสินค้า',en:'Add product'}) }}</v-btn></section>

  <v-dialog v-model="dialog" max-width="580">
    <v-card class="dialog-card">
      <v-card-title>{{ form.id ? l({lo:'ແກ້ໄຂສິນຄ້າ',th:'แก้ไขสินค้า',en:'Edit product'}) : l({lo:'ເພີ່ມສິນຄ້າ',th:'เพิ่มสินค้า',en:'Add product'}) }}</v-card-title>
      <v-card-text class="form">
        <div class="image-upload-wrapper mb-3">
          <div class="image-preview" :style="form.imageUrl ? `background-image: url(${form.imageUrl})` : ''">
            <v-icon v-if="!form.imageUrl" icon="mdi-image-outline" size="32" color="grey"/>
            <v-progress-circular v-if="uploading" indeterminate color="primary" class="upload-loader"/>
          </div>
          <div class="upload-actions">
            <v-btn variant="outlined" size="small" prepend-icon="mdi-upload" :loading="uploading" @click="fileInput?.click()">{{ l({lo:'ອັບໂຫຼດຮູບ',th:'อัปโหลดรูป',en:'Upload image'}) }}</v-btn>
            <input ref="fileInput" type="file" accept="image/*" hidden @change="uploadProductImage">
            <div class="text-caption text-grey mt-1">{{ l({lo:'ຫຼືວາງ URL ດ້ານລຸ່ມ',th:'หรือวาง URL ด้านล่าง',en:'Or paste a URL below:'}) }}</div>
          </div>
        </div>
        <v-text-field v-model="form.imageUrl" :label="l({lo:'URL ຮູບພາບ',th:'URL รูปภาพ',en:'Image URL'})" placeholder="https://..." hide-details class="mb-2"/>
        <v-text-field v-model="form.code" :label="l({lo:'ລະຫັດສິນຄ້າ',th:'รหัสสินค้า',en:'Product Code'})" placeholder="P01" maxlength="50" hide-details class="mb-2"/>
        <v-text-field v-model="form.name" :label="l({lo:'ຊື່ສິນຄ້າ',th:'ชื่อสินค้า',en:'Product name'})" autofocus maxlength="120" counter/>
        <v-text-field v-model="form.price" :label="l({lo:'ລາຄາ',th:'ราคา',en:'Price'})" :placeholder="l({lo:'ລາຄາ ແລະສະກຸນເງິນ',th:'ราคาและสกุลเงิน',en:'Price and currency'})" maxlength="60" counter/>
        <v-textarea v-model="form.description" :label="l({lo:'ລາຍລະອຽດສິນຄ້າ',th:'รายละเอียดสินค้า',en:'Product details'})" rows="4" maxlength="1000" counter/>
      </v-card-text>
      <v-card-actions><v-btn variant="text" @click="dialog=false">{{ l({lo:'ຍົກເລີກ',th:'ยกเลิก',en:'Cancel'}) }}</v-btn><v-spacer/><v-btn color="primary" :loading="saving" :disabled="!form.name.trim() || !form.price.trim()" @click="saveProduct">{{ l({lo:'ບັນທຶກສິນຄ້າ',th:'บันทึกสินค้า',en:'Save product'}) }}</v-btn></v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="deleteDialog" max-width="500">
    <v-card class="dialog-card">
      <v-card-title>{{ l({lo:'ລຶບສິນຄ້າ?',th:'ลบสินค้า?',en:'Delete product?'}) }}</v-card-title>
      <v-card-text>
        <template v-if="productToDelete && productAutomations(productToDelete.id).length">
          <v-alert type="warning" variant="tonal" class="mb-3">{{ l({lo:'ຍັງລຶບບໍ່ໄດ້ ເພາະສິນຄ້ານີ້ຖືກໃຊ້ໃນ Automation',th:'ยังลบไม่ได้ เพราะสินค้านี้ถูกใช้ใน Automation',en:'This product cannot be deleted while an Automation uses it.'}) }}</v-alert>
          <div class="dependency-list"><NuxtLink v-for="flow in productAutomations(productToDelete.id)" :key="flow.id" to="/auto-replies"><v-icon icon="mdi-robot-outline"/><span><strong>{{flow.name}}</strong><small>{{flow.targets.length}} {{ l({lo:'ເປົ້າໝາຍ',th:'เป้าหมาย',en:'targets'}) }}</small></span><v-icon icon="mdi-arrow-right"/></NuxtLink></div>
        </template>
        <p v-else>{{ l({lo:`ຢືນຢັນການລຶບ “${productToDelete?.name||''}”. ການກະທຳນີ້ຍ້ອນກັບບໍ່ໄດ້.`,th:`ยืนยันการลบ “${productToDelete?.name||''}” การดำเนินการนี้ย้อนกลับไม่ได้`,en:`Delete “${productToDelete?.name||''}”? This action cannot be undone.`}) }}</p>
      </v-card-text>
      <v-card-actions><v-btn variant="text" @click="deleteDialog=false">{{ l({lo:'ຍົກເລີກ',th:'ยกเลิก',en:'Cancel'}) }}</v-btn><v-spacer/><v-btn v-if="productToDelete&&!productAutomations(productToDelete.id).length" color="error" :loading="deleting" @click="deleteProduct">{{ l({lo:'ລຶບສິນຄ້າ',th:'ลบสินค้า',en:'Delete product'}) }}</v-btn><v-btn v-else color="primary" to="/auto-replies">{{ l({lo:'ໄປຈັດການ Automation',th:'ไปจัดการ Automation',en:'Manage Automations'}) }}</v-btn></v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.product-toolbar{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}
.product-search-box{position:relative;display:flex;align-items:center;max-width:420px;width:100%}
.product-search-box input{width:100%;height:40px;border:1px solid var(--color-border);border-radius:20px;padding:0 32px 0 36px;font-size:13px;background:var(--color-surface);color:var(--color-text);outline:none;transition:border-color .15s ease}
.product-search-box input:focus{border-color:var(--color-primary)}
.product-search-box .search-icon{position:absolute;left:12px;color:var(--color-text-muted);pointer-events:none}
.product-search-box .clear-btn{position:absolute;right:10px;background:none;border:none;color:var(--color-text-muted);cursor:pointer}
.product-count-chip{padding:6px 14px;border-radius:20px;background:var(--color-surface);border:1px solid var(--color-border);font-size:12px;font-weight:750;color:var(--color-text-secondary)}
.product-title-row{display:flex;align-items:center;justify-content:space-between;gap:8px}
.price-tag{display:inline-block;padding:2px 8px;border-radius:10px;background:#f0fdf4;color:#16a34a;border:1px solid #bbf7d0;font-size:12.5px;font-weight:800}
.code-copy-pill{display:inline-flex;align-items:center;gap:3px;padding:3px 9px;border-radius:12px;background:var(--color-primary-soft);color:var(--color-primary);border:1px solid color-mix(in srgb,var(--color-primary) 30%,transparent);font-size:11.5px;cursor:pointer;transition:all .15s ease}
.code-copy-pill:hover{background:var(--color-primary);color:#fff}
.image-upload-wrapper{display:flex;gap:16px;align-items:center}.image-preview{width:80px;height:80px;border-radius:12px;background-color:var(--color-surface-variant);background-size:cover;background-position:center;display:grid;place-items:center;position:relative;border:1px solid var(--color-border)}.upload-loader{position:absolute}
.page-head{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;margin-bottom:18px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head p{margin:0 0 7px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 7px;font-size:24px}.page-head span{color:var(--color-text-secondary);font-size:13px}.head-actions{display:flex;gap:10px;flex-wrap:wrap;justify-content:flex-end}.product-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.product-card{display:grid;grid-template-columns:auto 1fr auto;gap:15px;align-items:start;padding:20px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.product-icon{width:56px;height:56px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:14px;flex-shrink:0}.product-actions{display:flex;gap:4px}.product-card h3{margin:0;font-size:15px;font-weight:750}.product-card p{min-height:34px;margin:6px 0;color:var(--color-text-secondary);font-size:12px;line-height:1.5}.product-card small{color:var(--color-text-muted);font-size:10px}.dependency-list{display:grid;gap:8px}.dependency-list a{display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:10px;padding:11px;color:var(--color-text);background:var(--color-background);border:1px solid var(--color-border);border-radius:10px;text-decoration:none}.dependency-list a span{display:grid}.dependency-list small{color:var(--color-text-secondary)}.empty-state{min-height:390px;display:grid;place-items:center;align-content:center;text-align:center;padding:36px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.empty-state>div{width:58px;height:58px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:17px}.empty-state h2{margin:18px 0 6px}.empty-state p{max-width:480px;margin:0 0 20px;color:var(--color-text-secondary)}.dialog-card{border-radius:var(--radius-lg)!important}.form{display:grid;gap:4px;padding-top:18px!important}.inline-action{border:0;color:var(--color-primary);background:none;font-weight:800;cursor:pointer}
@media(max-width:760px){
  .product-grid{grid-template-columns:1fr}
}
@media(max-width:640px){
  .page-head{display:grid;padding:16px;border-radius:18px}
  .head-actions{display:grid;grid-template-columns:1fr 1fr;width:100%;gap:8px}
  .product-card{padding:14px 16px;border-radius:16px;gap:12px}
  .product-icon{width:48px;height:48px;border-radius:12px}
  .product-card p{min-height:auto;margin:4px 0 6px}
  .image-upload-wrapper{flex-direction:column;align-items:flex-start}
}
</style>
