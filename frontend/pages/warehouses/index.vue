<script setup lang="ts">
import type { ItemsResponse, Warehouse } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

const { can } = useAuth()
const { l } = useLocale()
if (!can('pages:read') && !can('orders:read')) {
  throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Warehouses.' })
}

const { data, pending, error, refresh } = await useApi<ItemsResponse<Warehouse>>('/proxy/api/v1/warehouses')
const warehouses = computed(() => data.value?.items || [])

const typeFilter = ref<'all' | 'warehouse' | 'shop'>('all')
const searchQuery = ref('')
const dialog = ref(false)
const saving = ref(false)
const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')

const form = reactive<{
  id?: string
  code: string
  name: string
  type: 'warehouse' | 'shop'
  address: string
  phone: string
  isDefault: boolean
  isActive: boolean
}>({
  id: undefined,
  code: '',
  name: '',
  type: 'warehouse',
  address: '',
  phone: '',
  isDefault: false,
  isActive: true,
})

const deleteDialog = ref(false)
const deleting = ref(false)
const warehouseToDelete = ref<Warehouse | null>(null)

const filteredWarehouses = computed(() => {
  return warehouses.value.filter((w) => {
    if (typeFilter.value !== 'all' && w.type !== typeFilter.value) return false
    const q = searchQuery.value.trim().toLowerCase()
    if (!q) return true
    return (
      w.name.toLowerCase().includes(q) ||
      w.code.toLowerCase().includes(q) ||
      (w.address && w.address.toLowerCase().includes(q)) ||
      (w.phone && w.phone.includes(q))
    )
  })
})

const stats = computed(() => {
  const total = warehouses.value.length
  const totalWarehouses = warehouses.value.filter((w) => w.type === 'warehouse').length
  const totalShops = warehouses.value.filter((w) => w.type === 'shop').length
  const defaultWh = warehouses.value.find((w) => w.isDefault)
  return { total, totalWarehouses, totalShops, defaultName: defaultWh ? defaultWh.name : '—' }
})

function openCreate() {
  Object.assign(form, {
    id: undefined,
    code: `WH-0${warehouses.value.length + 1}`,
    name: '',
    type: 'warehouse',
    address: '',
    phone: '',
    isDefault: warehouses.value.length === 0,
    isActive: true,
  })
  dialog.value = true
}

function openEdit(warehouse: Warehouse) {
  Object.assign(form, {
    id: warehouse.id,
    code: warehouse.code,
    name: warehouse.name,
    type: warehouse.type,
    address: warehouse.address || '',
    phone: warehouse.phone || '',
    isDefault: warehouse.isDefault,
    isActive: warehouse.isActive,
  })
  dialog.value = true
}

function requestDelete(warehouse: Warehouse) {
  warehouseToDelete.value = warehouse
  deleteDialog.value = true
}

async function saveWarehouse() {
  if (!form.name.trim() || !form.code.trim()) return
  saving.value = true
  notice.value = ''
  try {
    await $fetch('/api/proxy/api/v1/warehouses', {
      method: 'POST',
      body: {
        id: form.id || undefined,
        code: form.code.trim().toUpperCase(),
        name: form.name.trim(),
        type: form.type,
        address: form.address.trim(),
        phone: form.phone.trim(),
        isDefault: form.isDefault,
        isActive: form.isActive,
      },
    })
    dialog.value = false
    noticeType.value = 'success'
    notice.value = form.id
      ? l({ lo: 'ອັບເດດຂໍ້ມູນສາງ/ຮ້ານຄ້າແລ້ວ', th: 'อัปเดตข้อมูลคลัง/หน้าร้านเรียบร้อย', en: 'Location updated.' })
      : l({ lo: 'ສ້າງສາງ/ຮ້ານຄ້າໃໝ່ສຳເລັດແລ້ວ', th: 'สร้างคลัง/หน้าร้านใหม่เรียบร้อย', en: 'Location created successfully.' })
    await refresh()
  } catch (err: any) {
    noticeType.value = 'error'
    notice.value = err?.data?.error || err?.message || l({ lo: 'ບັນທຶກບໍ່ສຳເລັດ ກະລຸນາລອງໃໝ່', th: 'บันทึกไม่สำเร็จ โปรดลองอีกครั้ง', en: 'Failed to save location.' })
  } finally {
    saving.value = false
  }
}

async function deleteWarehouse() {
  if (!warehouseToDelete.value) return
  deleting.value = true
  notice.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/warehouses/${warehouseToDelete.value.id}`, {
      method: 'DELETE',
    })
    noticeType.value = 'success'
    notice.value = l({
      lo: `ລຶບ “${warehouseToDelete.value.name}” ສຳເລັດແລ້ວ`,
      th: `ลบ “${warehouseToDelete.value.name}” เรียบร้อยแล้ว`,
      en: `Deleted “${warehouseToDelete.value.name}”.`,
    })
    deleteDialog.value = false
    await refresh()
  } catch (err: any) {
    noticeType.value = 'error'
    notice.value = err?.data?.error || err?.message || l({ lo: 'ລຶບບໍ່ສຳເລັດ', th: 'ลบไม่สำเร็จ', en: 'Failed to delete.' })
  } finally {
    deleting.value = false
    warehouseToDelete.value = null
  }
}
</script>

<template>
  <div class="warehouses-page">
    <section class="app-page-head">
      <div class="head-content">
        <span class="head-badge">
          <v-icon icon="mdi-warehouse" size="14" />
          Inventory & Locations
        </span>
        <h1>{{ l({ lo: 'ສາງ ແລະ ໜ້າຮ້ານ (Warehouses & Shops)', th: 'คลังสินค้าและหน้าร้าน', en: 'Warehouses & Locations' }) }}</h1>
        <p>{{ l({ lo: 'ຈັດການສາງເກັບສິນຄ້າ ແລະ ສາຂາໜ້າຮ້ານ, ຕັດສະຕ໋ອກແຍກຕາມສະຖານທີ່', th: 'จัดการคลังสินค้าและสาขาหน้าร้าน พร้อมตัดสต็อกแยกตามสถานที่', en: 'Manage inventory warehouses and branch shops with location-based stock deduction.' }) }}</p>
      </div>
      <div class="head-actions">
        <v-btn variant="outlined" prepend-icon="mdi-package-variant-closed" to="/products">
          {{ l({ lo: 'ຈັດການສິນຄ້າ', th: 'จัดการสินค้า', en: 'Products' }) }}
        </v-btn>
        <v-btn variant="outlined" prepend-icon="mdi-cart-outline" to="/orders">
          {{ l({ lo: 'ຈັດການອໍເດີ', th: 'จัดการออเดอร์', en: 'Orders' }) }}
        </v-btn>
        <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" @click="openCreate">
          {{ l({ lo: 'ເພີ່ມສາງ / ຮ້ານຄ້າ', th: 'เพิ่มคลัง / หน้าร้าน', en: 'Add Location' }) }}
        </v-btn>
      </div>
    </section>

    <!-- Metrics Stats Cards -->
    <div class="app-kpi-grid mb-4">
      <div class="app-kpi-card accent-primary">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ສະຖານທີ່ທັງໝົດ', th: 'สถานที่ทั้งหมด', en: 'Total Locations' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-map-marker-multiple" size="16"/></span>
        </div>
        <strong class="kpi-value">{{ stats.total }}</strong>
        <div class="kpi-sub"><span>ສາງ + ໜ້າຮ້ານທັງໝົດ</span></div>
      </div>

      <div class="app-kpi-card accent-info">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ສາງເກັບສິນຄ້າ', th: 'คลังสินค้า', en: 'Warehouses' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-warehouse" size="16"/></span>
        </div>
        <strong class="kpi-value text-info">{{ stats.totalWarehouses }}</strong>
        <div class="kpi-sub"><span>ສາງຫຼັກ & ສາງຍ່ອຍ</span></div>
      </div>

      <div class="app-kpi-card accent-purple">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ສາຂາໜ້າຮ້ານ', th: 'สาขาหน้าร้าน', en: 'Branch Shops' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-storefront" size="16"/></span>
        </div>
        <strong class="kpi-value text-purple">{{ stats.totalShops }}</strong>
        <div class="kpi-sub"><span>ຈຸດຂາຍໜ້າຮ້ານ</span></div>
      </div>

      <div class="app-kpi-card accent-warning">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ສາງຫຼັກເລີ່ມຕົ້ນ', th: 'คลังหลักเริ่มต้น', en: 'Default Location' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-star-circle" size="16"/></span>
        </div>
        <strong class="kpi-value truncate-text" :title="stats.defaultName">{{ stats.defaultName }}</strong>
        <div class="kpi-sub"><span>ສຳລັບຕັດສະຕ໋ອກເລີ່ມຕົ້ນ</span></div>
      </div>
    </div>

    <!-- Alert Notices -->
    <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">
      {{ notice }}
    </v-alert>
    <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
      {{ l({ lo: 'ໂຫຼດຂໍ້ມູນສາງບໍ່ສຳເລັດ', th: 'โหลดข้อมูลคลังไม่สำเร็จ', en: 'Locations could not be loaded.' }) }}
      <button class="inline-action" @click="refresh()">{{ l({ lo: 'ລອງໃໝ່', th: 'ลองอีกครั้ง', en: 'Try again' }) }}</button>
    </v-alert>

    <!-- Filter & Search Toolbar -->
    <div class="warehouse-toolbar mb-4">
      <div class="filter-pills">
        <button
          type="button"
          :class="['filter-btn', { active: typeFilter === 'all' }]"
          @click="typeFilter = 'all'"
        >
          {{ l({ lo: 'ທັງໝົດ', th: 'ทั้งหมด', en: 'All' }) }} ({{ warehouses.length }})
        </button>
        <button
          type="button"
          :class="['filter-btn', { active: typeFilter === 'warehouse' }]"
          @click="typeFilter = 'warehouse'"
        >
          <v-icon icon="mdi-warehouse" size="14" class="mr-1"/>
          {{ l({ lo: 'ສາງເກັບສິນຄ້າ', th: 'คลังสินค้า', en: 'Warehouses' }) }} ({{ stats.totalWarehouses }})
        </button>
        <button
          type="button"
          :class="['filter-btn', { active: typeFilter === 'shop' }]"
          @click="typeFilter = 'shop'"
        >
          <v-icon icon="mdi-storefront" size="14" class="mr-1"/>
          {{ l({ lo: 'ສາຂາໜ້າຮ້ານ', th: 'สาขาหน้าร้าน', en: 'Shops' }) }} ({{ stats.totalShops }})
        </button>
      </div>

      <div class="search-box">
        <v-icon icon="mdi-magnify" size="18" class="search-icon"/>
        <input
          v-model="searchQuery"
          type="text"
          :aria-label="l({ lo: 'ຄົ້ນຫາສາງ', th: 'ค้นหาคลัง', en: 'Search Locations' })"
          :placeholder="l({ lo: 'ຄົ້ນຫາຊື່ສາງ, ລະຫັດ, ທີ່ຢູ່...', th: 'ค้นหาชื่อคลัง, รหัส, ที่อยู่...', en: 'Search location name, code, address...' })"
        />
        <button v-if="searchQuery" class="clear-btn" @click="searchQuery = ''">
          <v-icon icon="mdi-close" size="14"/>
        </button>
      </div>
    </div>

    <!-- Cards Grid -->
    <section v-if="pending" class="warehouse-grid">
      <v-skeleton-loader v-for="i in 3" :key="i" type="card"/>
    </section>

    <section v-else-if="filteredWarehouses.length" class="warehouse-grid">
      <article v-for="item in filteredWarehouses" :key="item.id" class="warehouse-card">
        <div class="card-top">
          <div class="type-icon" :class="item.type === 'warehouse' ? 'is-wh' : 'is-shop'">
            <v-icon :icon="item.type === 'warehouse' ? 'mdi-warehouse' : 'mdi-storefront'" size="24"/>
          </div>
          <div class="card-badges">
            <span v-if="item.isDefault" class="badge-default">
              <v-icon icon="mdi-star" size="12" class="mr-1"/>
              {{ l({ lo: 'ສາງຫຼັກ', th: 'คลังหลัก', en: 'Default' }) }}
            </span>
            <span class="badge-type" :class="item.type === 'warehouse' ? 'badge-wh' : 'badge-shop'">
              {{ item.type === 'warehouse' ? l({ lo: 'ສາງເກັບສິນຄ້າ', th: 'คลังสินค้า', en: 'Warehouse' }) : l({ lo: 'ສາຂາໜ້າຮ້ານ', th: 'หน้าร้าน', en: 'Shop' }) }}
            </span>
            <span class="badge-code">#{{ item.code }}</span>
          </div>
        </div>

        <div class="card-body">
          <h3 class="location-name">{{ item.name }}</h3>
          <p class="location-address">
            <v-icon icon="mdi-map-marker-outline" size="14" class="mr-1 text-muted"/>
            {{ item.address || l({ lo: 'ບໍ່ໄດ້ລະບຸທີ່ຢູ່', th: 'ไม่ได้ระบุที่อยู่', en: 'No address specified' }) }}
          </p>
          <p v-if="item.phone" class="location-phone">
            <v-icon icon="mdi-phone-outline" size="14" class="mr-1 text-muted"/>
            <a :href="`tel:${item.phone}`">{{ item.phone }}</a>
          </p>
        </div>

        <div class="card-footer">
          <span class="status-indicator" :class="item.isActive ? 'active' : 'inactive'">
            <span class="status-dot"></span>
            {{ item.isActive ? l({ lo: 'ພ້ອມໃຊ້ງານ', th: 'พร้อมใช้งาน', en: 'Active' }) : l({ lo: 'ປິດໃຊ້ງານ', th: 'ปิดใช้งาน', en: 'Inactive' }) }}
          </span>

          <div class="card-actions">
            <v-btn icon="mdi-pencil-outline" variant="text" size="small" @click="openEdit(item)" :title="l({ lo: 'ແກ້ໄຂ', th: 'แก้ไข', en: 'Edit' })"/>
            <v-btn
              icon="mdi-delete-outline"
              variant="text"
              size="small"
              color="error"
              :disabled="item.isDefault"
              @click="requestDelete(item)"
              :title="item.isDefault ? l({ lo: 'ສາງຫຼັກບໍ່ສາມາດລຶບໄດ້', th: 'คลังหลักไม่สามารถลบได้', en: 'Cannot delete default location' }) : l({ lo: 'ລຶບ', th: 'ลบ', en: 'Delete' })"
            />
          </div>
        </div>
      </article>
    </section>

    <section v-else class="empty-state">
      <div class="empty-icon-circle">
        <v-icon icon="mdi-warehouse" size="32" color="primary"/>
      </div>
      <h2>{{ l({ lo: 'ບໍ່ພົບສາງ ຫຼື ຮ້ານຄ້າ', th: 'ไม่พบคลังหรือหน้าร้าน', en: 'No locations found' }) }}</h2>
      <p>{{ l({ lo: 'ສ້າງສາງເກັບສິນຄ້າ ຫຼື ສາຂາໜ້າຮ້ານ ເພື່ອຕັດສະຕ໋ອກສິນຄ້າອັດຕະໂນມັດເມື່ອມີການສັ່ງຊື້', th: 'สร้างคลังสินค้าหรือสาขาหน้าร้านเพื่อตัดสต็อกสินค้าอัตโนมัติเมื่อมีออเดอร์', en: 'Create warehouses or shops to manage your inventory and enable auto-stock deduction.' }) }}</p>
      <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreate">
        {{ l({ lo: 'ເພີ່ມສາງ / ຮ້ານຄ້າ', th: 'เพิ่มคลัง / หน้าร้าน', en: 'Add Location' }) }}
      </v-btn>
    </section>

    <!-- Create / Edit Dialog -->
    <v-dialog v-model="dialog" max-width="520">
      <v-card class="dialog-card">
        <v-card-title class="dialog-title">
          <v-icon :icon="form.id ? 'mdi-pencil' : 'mdi-plus-circle'" class="mr-2" color="primary"/>
          {{ form.id ? l({ lo: 'ແກ້ໄຂສາງ / ຮ້ານຄ້າ', th: 'แก้ไขคลัง / หน้าร้าน', en: 'Edit Location' }) : l({ lo: 'ເພີ່ມສາງ / ຮ້ານຄ້າໃໝ່', th: 'เพิ่มคลัง / หน้าร้านใหม่', en: 'Add New Location' }) }}
        </v-card-title>
        <v-card-text class="dialog-body">
          <!-- Type selection buttons -->
          <div class="type-selector mb-3">
            <button
              type="button"
              class="type-choice-btn"
              :class="{ selected: form.type === 'warehouse' }"
              @click="form.type = 'warehouse'"
            >
              <v-icon icon="mdi-warehouse" size="20" class="mb-1"/>
              <strong>{{ l({ lo: 'ສາງເກັບສິນຄ້າ', th: 'คลังสินค้า', en: 'Warehouse' }) }}</strong>
              <small>{{ l({ lo: 'ເກັບສິນຄ້າສະຕ໋ອກຫຼັກ', th: 'เก็บสต็อกหลัก/คลังกลาง', en: 'Main storage' }) }}</small>
            </button>
            <button
              type="button"
              class="type-choice-btn"
              :class="{ selected: form.type === 'shop' }"
              @click="form.type = 'shop'"
            >
              <v-icon icon="mdi-storefront" size="20" class="mb-1"/>
              <strong>{{ l({ lo: 'ສາຂາໜ້າຮ້ານ', th: 'สาขาหน้าร้าน', en: 'Shop' }) }}</strong>
              <small>{{ l({ lo: 'ຂາຍໜ້າຮ້ານ/ຈຸດສົ່ງ', th: 'ขายหน้าร้าน/จุดส่ง', en: 'Retail branch' }) }}</small>
            </button>
          </div>

          <v-text-field
            v-model="form.name"
            :label="l({ lo: 'ຊື່ສາງ ຫຼື ຮ້ານຄ້າ *', th: 'ชื่อคลัง หรือ หน้าร้าน *', en: 'Location Name *' })"
            :placeholder="l({ lo: 'ຕົວຢ່າງ: ສາງໃຫຍ່ວຽງຈັນ, ຮ້ານສາຂາດົງໂດກ', th: 'ตัวอย่าง: คลังใหญ่เวียงจันทน์, สาขาดงโดก', en: 'e.g. Main Vientiane WH, Branch 1' })"
            class="mb-2"
            autofocus
            required
          />

          <v-text-field
            v-model="form.code"
            :label="l({ lo: 'ລະຫັດອ້າງອີງ *', th: 'รหัสอ้างอิง *', en: 'Location Code *' })"
            placeholder="WH-01, SHOP-VTE"
            class="mb-2"
            maxlength="30"
            required
          />

          <v-text-field
            v-model="form.phone"
            :label="l({ lo: 'ເບີໂທຕິດຕໍ່', th: 'เบอร์โทรศัพท์ติดต่อ', en: 'Contact Phone' })"
            placeholder="020 xxxxxxxx"
            class="mb-2"
          />

          <v-textarea
            v-model="form.address"
            :label="l({ lo: 'ທີ່ຢູ່ຕັ້ງ / ລາຍລະອຽດ', th: 'ที่ตั้ง / รายละเอียด', en: 'Address / Location Details' })"
            placeholder="ບ້ານ, ເມືອງ, ແຂວງ..."
            rows="3"
            class="mb-2"
          />

          <div class="checkbox-options">
            <v-checkbox
              v-model="form.isDefault"
              :label="l({ lo: 'ຕັ້ງເປັນສາງຫຼັກເລີ່ມຕົ້ນ (Default)', th: 'ตั้งเป็นคลังหลักเริ่มต้น (Default)', en: 'Set as Default Location' })"
              color="primary"
              hide-details
              density="compact"
            />
            <v-checkbox
              v-model="form.isActive"
              :label="l({ lo: 'ເປີດໃຊ້ງານ (Active)', th: 'เปิดใช้งาน (Active)', en: 'Active' })"
              color="primary"
              hide-details
              density="compact"
            />
          </div>
        </v-card-text>
        <v-card-actions class="dialog-actions">
          <v-btn variant="text" @click="dialog = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </v-btn>
          <v-spacer/>
          <v-btn
            color="primary"
            :loading="saving"
            :disabled="!form.name.trim() || !form.code.trim()"
            @click="saveWarehouse"
          >
            {{ l({ lo: 'ບັນທຶກ', th: 'บันทึก', en: 'Save' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" max-width="440">
      <v-card class="dialog-card">
        <v-card-title>{{ l({ lo: 'ຢືນຢັນການລຶບ?', th: 'ยืนยันการลบ?', en: 'Confirm Deletion' }) }}</v-card-title>
        <v-card-text>
          <p>
            {{ l({
              lo: `ທ່ານຕ້ອງການລຶບ “${warehouseToDelete?.name || ''}” ແທ້ບໍ່? ຂໍ້ມູນສະຕ໋ອກທີ່ກ່ຽວຂ້ອງຈະຖືກລຶບອອກ.`,
              th: `คุณต้องการลบ “${warehouseToDelete?.name || ''}” หรือไม่? ข้อมูลสต็อกที่เกี่ยวข้องจะถูกลบไปด้วย`,
              en: `Are you sure you want to delete “${warehouseToDelete?.name || ''}”? Stock records for this location will be removed.`,
            }) }}
          </p>
        </v-card-text>
        <v-card-actions>
          <v-btn variant="text" @click="deleteDialog = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </v-btn>
          <v-spacer/>
          <v-btn color="error" :loading="deleting" @click="deleteWarehouse">
            {{ l({ lo: 'ລຶບອອກ', th: 'ลบออก', en: 'Delete' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.warehouses-page {
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
  font-size: 12px;
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
.bg-indigo-light { background: #e0e7ff; }
.bg-purple-light { background: #f3e8ff; }
.bg-amber-light { background: #fef3c7; }
.stat-text {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.stat-text small {
  font-size: 12px;
  color: var(--color-text-muted);
}
.stat-text strong {
  font-size: 18px;
  font-weight: 800;
  color: var(--color-text);
}
.truncate-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 130px;
  font-size: 14px !important;
}

.warehouse-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.filter-pills {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.filter-btn {
  display: inline-flex;
  align-items: center;
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
.filter-btn:hover {
  background: var(--color-surface-variant);
  color: var(--color-text);
}
.filter-btn.active {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  max-width: 360px;
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
  transition: border-color .15s ease;
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

.warehouse-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.warehouse-card {
  display: flex;
  flex-direction: column;
  padding: 20px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  transition: transform .15s ease, box-shadow .15s ease;
}
.warehouse-card:hover {
  border-color: color-mix(in srgb, var(--color-primary) 40%, var(--color-border));
}

.card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 14px;
}
.type-icon {
  width: 46px;
  height: 46px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.type-icon.is-wh {
  background: #eff6ff;
  color: #2563eb;
}
.type-icon.is-shop {
  background: #fdf4ff;
  color: #c026d3;
}

.card-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
}
.badge-default {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  background: #fef3c7;
  color: #b45309;
  border: 1px solid #fde68a;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 800;
}
.badge-type {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 750;
}
.badge-wh {
  background: #eff6ff;
  color: #1d4ed8;
  border: 1px solid #bfdbfe;
}
.badge-shop {
  background: #faf5ff;
  color: #7e22ce;
  border: 1px solid #e9d5ff;
}
.badge-code {
  padding: 3px 8px;
  background: var(--color-surface-variant);
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  font-size: 12px;
  font-weight: 700;
}

.card-body {
  flex: 1;
  margin-bottom: 16px;
}
.location-name {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 750;
  color: var(--color-text);
}
.location-address {
  margin: 0 0 6px;
  font-size: 12.5px;
  color: var(--color-text-secondary);
  line-height: 1.4;
  display: flex;
  align-items: center;
}
.location-phone {
  margin: 0;
  font-size: 12.5px;
  display: flex;
  align-items: center;
}
.location-phone a {
  color: var(--color-primary);
  text-decoration: none;
  font-weight: 600;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid var(--color-border);
  padding-top: 12px;
}
.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 650;
}
.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}
.status-indicator.active {
  color: #16a34a;
}
.status-indicator.active .status-dot {
  background: #16a34a;
}
.status-indicator.inactive {
  color: #94a3b8;
}
.status-indicator.inactive .status-dot {
  background: #94a3b8;
}

.card-actions {
  display: flex;
  gap: 2px;
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

.type-selector {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.type-choice-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px;
  border-radius: 12px;
  border: 1.5px solid var(--color-border);
  background: var(--color-surface);
  cursor: pointer;
  transition: all .15s ease;
}
.type-choice-btn.selected {
  border-color: var(--color-primary);
  background: var(--color-primary-soft);
  color: var(--color-primary);
}
.type-choice-btn strong {
  font-size: 13px;
}
.type-choice-btn small {
  font-size: 12px;
  color: var(--color-text-muted);
}
.type-choice-btn.selected small {
  color: var(--color-primary);
  opacity: .85;
}

.checkbox-options {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.inline-action {
  background: none;
  border: none;
  color: var(--color-primary);
  font-weight: 750;
  cursor: pointer;
  margin-left: 6px;
}

@media (max-width: 960px) {
  .warehouse-grid {
    grid-template-columns: repeat(2, 1fr);
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
  .warehouse-grid {
    grid-template-columns: 1fr;
  }
  .warehouse-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  .search-box {
    max-width: 100%;
  }
}
</style>
