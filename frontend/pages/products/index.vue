<script setup lang="ts">
import type { ItemsResponse, Product } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

const { can } = useAuth()
const { locale } = useLocale()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Products.' })

const { data, pending, error, refresh } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const products = computed(() => data.value?.items || [])
const dialog = ref(false)
const saving = ref(false)
const uploading = ref(false)
const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')
const form = reactive<Product>({ id: '', name: '', price: '', description: '', imageUrl: '' })

const fileInput = ref<HTMLInputElement>()

function openCreate() {
  Object.assign(form, { id: '', name: '', price: '', description: '', imageUrl: '' })
  dialog.value = true
}

function openEdit(product: Product) {
  Object.assign(form, product)
  dialog.value = true
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
    notice.value = errorMessage(err, 'Failed to upload image. Please try again.')
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
        name: form.name.trim(),
        price: form.price.trim(),
        description: form.description.trim(),
        imageUrl: form.imageUrl?.trim() || undefined,
      },
    })
    dialog.value = false
    noticeType.value = 'success'
    notice.value = form.id ? 'Product updated.' : 'Product created and ready for Auto Replies.'
    await refresh()
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, 'Unable to save the product.')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="page-head">
    <div><p>PRODUCT CATALOG</p><h2>{{locale==='lo'?'ຈັດການສິນຄ້າ':'Products'}}</h2><span>{{locale==='lo'?'ເກັບຊື່, ລາຄາ, ຮູບ ແລະລາຍລະອຽດເພື່ອໃຊ້ຮ່ວມກັບຊຸດຂໍ້ຄວາມ':'Keep product details in one place and reuse them in automatic replies.'}}</span></div>
    <div class="head-actions"><v-btn variant="outlined" prepend-icon="mdi-robot-happy-outline" to="/auto-replies">{{locale==='lo'?'ຈັດການ Automation':'Manage Automations'}}</v-btn><v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreate">{{locale==='lo'?'ເພີ່ມສິນຄ້າ':'Add product'}}</v-btn></div>
  </section>

  <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>
  <v-alert v-if="error" type="error" variant="tonal" class="mb-4">Products could not be loaded. <button class="inline-action" @click="refresh()">Try again</button></v-alert>

  <section v-if="pending" class="product-grid"><v-skeleton-loader v-for="index in 3" :key="index" type="card"/></section>
  <section v-else-if="products.length" class="product-grid">
    <article v-for="product in products" :key="product.id" class="product-card">
      <div class="product-icon" :style="product.imageUrl ? `background-image: url(${product.imageUrl}); background-size: cover; background-position: center;` : ''">
        <v-icon v-if="!product.imageUrl" icon="mdi-package-variant-closed"/>
      </div>
      <div><h3>{{ product.name }}</h3><strong>{{ product.price }}</strong><p v-if="product.description">{{ product.description }}</p><small>Product ID · {{ product.id }}</small></div>
      <v-btn icon="mdi-pencil-outline" variant="text" size="small" :disabled="!can('pages:connect')" :aria-label="`Edit ${product.name}`" @click="openEdit(product)"/>
    </article>
  </section>
  <section v-else-if="!error" class="empty-state"><div><v-icon icon="mdi-package-variant-plus" size="30"/></div><h2>Add the first product</h2><p>A product provides the name, price, image and details inserted into your Reply Sets.</p><v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreate">Add product</v-btn></section>

  <v-dialog v-model="dialog" max-width="580">
    <v-card class="dialog-card">
      <v-card-title>{{ form.id ? 'Edit product' : 'Add product' }}</v-card-title>
      <v-card-text class="form">
        <div class="image-upload-wrapper mb-3">
          <div class="image-preview" :style="form.imageUrl ? `background-image: url(${form.imageUrl})` : ''">
            <v-icon v-if="!form.imageUrl" icon="mdi-image-outline" size="32" color="grey"/>
            <v-progress-circular v-if="uploading" indeterminate color="primary" class="upload-loader"/>
          </div>
          <div class="upload-actions">
            <v-btn variant="outlined" size="small" prepend-icon="mdi-upload" :loading="uploading" @click="fileInput?.click()">Upload image</v-btn>
            <input ref="fileInput" type="file" accept="image/*" hidden @change="uploadProductImage">
            <div class="text-caption text-grey mt-1">Or paste a URL below:</div>
          </div>
        </div>
        <v-text-field v-model="form.imageUrl" label="Image URL" placeholder="https://..." hide-details class="mb-2"/>
        <v-text-field v-model="form.name" label="Product name" autofocus maxlength="120" counter/>
        <v-text-field v-model="form.price" label="Price" placeholder="Price and currency" maxlength="60" counter/>
        <v-textarea v-model="form.description" label="Product details" rows="4" maxlength="1000" counter/>
      </v-card-text>
      <v-card-actions><v-btn variant="text" @click="dialog=false">Cancel</v-btn><v-spacer/><v-btn color="primary" :loading="saving" :disabled="!form.name.trim() || !form.price.trim()" @click="saveProduct">Save product</v-btn></v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.image-upload-wrapper{display:flex;gap:16px;align-items:center}.image-preview{width:80px;height:80px;border-radius:12px;background-color:var(--color-surface-variant);background-size:cover;background-position:center;display:grid;place-items:center;position:relative;border:1px solid var(--color-border)}.upload-loader{position:absolute}
.page-head{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;margin-bottom:18px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head p{margin:0 0 7px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 7px;font-size:24px}.page-head span{color:var(--color-text-secondary);font-size:13px}.head-actions{display:flex;gap:10px;flex-wrap:wrap;justify-content:flex-end}.product-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.product-card{display:grid;grid-template-columns:auto 1fr auto;gap:15px;align-items:start;padding:20px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.product-icon{width:48px;height:48px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:14px}.product-card h3{margin:1px 0 3px;font-size:15px}.product-card strong{color:var(--color-primary);font-size:13px}.product-card p{min-height:38px;margin:8px 0;color:var(--color-text-secondary);font-size:12px;line-height:1.55}.product-card small{color:var(--color-text-muted);font-size:10px}.empty-state{min-height:390px;display:grid;place-items:center;align-content:center;text-align:center;padding:36px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.empty-state>div{width:58px;height:58px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:17px}.empty-state h2{margin:18px 0 6px}.empty-state p{max-width:480px;margin:0 0 20px;color:var(--color-text-secondary)}.dialog-card{border-radius:var(--radius-lg)!important}.form{display:grid;gap:4px;padding-top:18px!important}.inline-action{border:0;color:var(--color-primary);background:none;font-weight:800;cursor:pointer}
@media(max-width:900px){.product-grid{grid-template-columns:1fr}}
@media(max-width:640px){.page-head{display:grid;padding:20px}.head-actions{display:grid;grid-template-columns:1fr 1fr;width:100%}}
</style>
