<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

const { l, locale } = useAdminLocale()
const { adminFetch } = useAdminApi()

interface AdminPlan {
  id: string
  name: string
  price: number
  maxUsers: number
  maxPages: number
  isActive: boolean
  subscriberCount: number
}

interface TenantItem {
  id: string
  name: string
  planId: string
  planName: string
  planStatus: string
}

const loading = ref(true)
const plans = ref<AdminPlan[]>([])
const actionLoading = ref<Record<string, boolean>>({})

// Create / Edit modal
const planModalOpen = ref(false)
const isEditing = ref(false)
const modalError = ref('')
const saving = ref(false)

const form = ref({
  id: '',
  name: '',
  price: 500000,
  maxUsers: 10,
  maxPages: 10,
  isActive: true,
})

// Assign plan to tenant modal
const assignModalOpen = ref(false)
const assigning = ref(false)
const assignError = ref('')
const tenantsList = ref<TenantItem[]>([])
const assignForm = ref({
  accountId: '',
  planId: 'pro',
  days: 30,
  status: 'active',
})

const loadPlans = async () => {
  loading.value = true
  try {
    const res = await adminFetch<{ items: AdminPlan[] }>('/api/v1/admin/plans')
    plans.value = res.items || []
  } catch (err: any) {
    console.error('Failed to load plans:', err)
  } finally {
    loading.value = false
  }
}

const loadTenantsForAssign = async () => {
  try {
    const res = await adminFetch<{ items: TenantItem[] }>('/api/v1/admin/tenants')
    tenantsList.value = res.items || []
  } catch (err) {
    console.error('Failed to load tenants for assign:', err)
  }
}

const totalPlans = computed(() => plans.value.length)
const activePlans = computed(() => plans.value.filter(p => p.isActive).length)
const totalSubscribers = computed(() => plans.value.reduce((acc, p) => acc + (p.subscriberCount || 0), 0))

const formatLAK = (val: number) => {
  if (val === 0) return l({ lo: 'ຟຣີ (0 ₭)', th: 'ฟรี (0 ₭)', en: 'Free (0 ₭)' })
  return new Intl.NumberFormat(locale.value === 'lo' ? 'lo-LA' : locale.value === 'th' ? 'th-TH' : 'en-US').format(val) + ' ₭'
}

const openCreateModal = () => {
  isEditing.value = false
  modalError.value = ''
  form.value = {
    id: '',
    name: '',
    price: 500000,
    maxUsers: 10,
    maxPages: 10,
    isActive: true,
  }
  planModalOpen.value = true
}

const openEditModal = (p: AdminPlan) => {
  isEditing.value = true
  modalError.value = ''
  form.value = {
    id: p.id,
    name: p.name,
    price: p.price,
    maxUsers: p.maxUsers,
    maxPages: p.maxPages,
    isActive: p.isActive,
  }
  planModalOpen.value = true
}

const applyPreset = (preset: { id: string; name: string; price: number; maxUsers: number; maxPages: number }) => {
  if (!isEditing.value) {
    form.value.id = preset.id
  }
  form.value.name = preset.name
  form.value.price = preset.price
  form.value.maxUsers = preset.maxUsers
  form.value.maxPages = preset.maxPages
}

const savePlan = async () => {
  modalError.value = ''
  if (!form.value.id.trim()) {
    modalError.value = l({ lo: 'ກະລຸນາປ້ອນລະຫັດ Plan (ID)', th: 'กรุณากรอกรหัส Plan (ID)', en: 'Plan ID is required' })
    return
  }
  if (!form.value.name.trim()) {
    modalError.value = l({ lo: 'ກະລຸນາປ້ອນຊື່ Plan', th: 'กรุณากรอกชื่อ Plan', en: 'Plan Name is required' })
    return
  }

  saving.value = true
  try {
    const payload = {
      id: form.value.id.trim().toLowerCase(),
      name: form.value.name.trim(),
      price: Number(form.value.price) || 0,
      maxUsers: Number(form.value.maxUsers) || 1,
      maxPages: Number(form.value.maxPages) || 1,
      isActive: Boolean(form.value.isActive),
    }

    if (isEditing.value) {
      await adminFetch(`/api/v1/admin/plans/${form.value.id}`, {
        method: 'PUT',
        body: payload,
      })
    } else {
      await adminFetch('/api/v1/admin/plans', {
        method: 'POST',
        body: payload,
      })
    }

    planModalOpen.value = false
    await loadPlans()
  } catch (err: any) {
    modalError.value = err?.data?.message || err?.message || 'Failed to save plan'
  } finally {
    saving.value = false
  }
}

const togglePlanStatus = async (p: AdminPlan) => {
  actionLoading.value[p.id] = true
  try {
    await adminFetch(`/api/v1/admin/plans/${p.id}`, {
      method: 'PUT',
      body: {
        ...p,
        isActive: !p.isActive,
      },
    })
    await loadPlans()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Failed to toggle status')
  } finally {
    actionLoading.value[p.id] = false
  }
}

const deletePlan = async (p: AdminPlan) => {
  const confirmMsg = l({
    lo: `ທ່ານແນ່ໃຈບໍ່ວ່າຕ້ອງການລຶບ ຫຼື ປິດການໃຊ້ງານແພັກເກດ "${p.name}"?`,
    th: `คุณแน่ใจหรือไม่ว่าต้องการลบหรือปิดใช้งานแพ็กเกจ "${p.name}"?`,
    en: `Are you sure you want to delete/archive plan "${p.name}"?`
  })
  if (!confirm(confirmMsg)) return

  actionLoading.value[p.id] = true
  try {
    await adminFetch(`/api/v1/admin/plans/${p.id}`, {
      method: 'DELETE',
    })
    await loadPlans()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Failed to delete plan')
  } finally {
    actionLoading.value[p.id] = false
  }
}

const openAssignModal = async () => {
  assignError.value = ''
  assignForm.value = {
    accountId: tenantsList.value[0]?.id || '',
    planId: 'pro',
    days: 30,
    status: 'active',
  }
  await loadTenantsForAssign()
  if (tenantsList.value.length > 0 && !assignForm.value.accountId && tenantsList.value[0]) {
    assignForm.value.accountId = tenantsList.value[0].id
  }
  assignModalOpen.value = true
}

const submitAssign = async () => {
  assignError.value = ''
  if (!assignForm.value.accountId) {
    assignError.value = l({ lo: 'ກະລຸນາເລືອກຮ້ານຄ້າ', th: 'กรุณาเลือกร้านค้า', en: 'Please select a store' })
    return
  }

  assigning.value = true
  try {
    await adminFetch(`/api/v1/admin/tenants/${assignForm.value.accountId}/plan`, {
      method: 'PUT',
      body: {
        planId: assignForm.value.planId,
        days: Number(assignForm.value.days) || 30,
        status: assignForm.value.status,
      },
    })
    assignModalOpen.value = false
    alert(l({
      lo: 'ອັບເດດແພັກເກດໃຫ້ຮ້ານຄ້າສຳເລັດແລ້ວ!',
      th: 'อัปเดตแพ็กเกจให้ร้านค้าสำเร็จเรียบร้อย!',
      en: 'Store subscription plan updated successfully!'
    }))
    await loadPlans()
  } catch (err: any) {
    assignError.value = err?.data?.message || err?.message || 'Failed to assign plan'
  } finally {
    assigning.value = false
  }
}

onMounted(() => {
  loadPlans()
  loadTenantsForAssign()
})
</script>

<template>
  <div class="plans-page">
    <!-- Top Header -->
    <div class="page-header">
      <div class="header-left">
        <div class="icon-wrap">
          <v-icon icon="mdi-tag-multiple" size="24" color="#0284C7" />
        </div>
        <div>
          <h1>{{ l({ lo: 'ຈັດການແພັກເກດລາຄາ (Plans)', th: 'จัดการแพ็กเกจราคา (Plans)', en: 'Subscription Plans Management' }) }}</h1>
          <p>{{ l({ lo: 'ກຳນົດລາຄາ, ໂຄຕ້າເພຈ/ທີມງານ ແລະ ແພັກເກດສຳລັບຮ້ານຄ້າໃນລະບົບ', th: 'กำหนดราคา โควตาเพจ/ทีมงาน และแพ็กเกจสำหรับร้านค้าในระบบ', en: 'Configure pricing tiers, page/user quotas, and store subscriptions' }) }}</p>
        </div>
      </div>
      <div class="header-actions">
        <button type="button" class="btn-secondary" @click="openAssignModal">
          <v-icon icon="mdi-account-arrow-right" size="18" />
          <span>{{ l({ lo: 'ມອບສິດແພັກເກດໃຫ້ຮ້ານຄ້າ', th: 'มอบสิทธิ์แพ็กเกจให้ร้าน', en: 'Assign Plan to Store' }) }}</span>
        </button>
        <button type="button" class="btn-primary" @click="openCreateModal">
          <v-icon icon="mdi-plus-circle" size="18" />
          <span>{{ l({ lo: 'ສ້າງແພັກເກດໃໝ່', th: 'สร้างแพ็กเกจใหม่', en: 'New Plan' }) }}</span>
        </button>
      </div>
    </div>

    <!-- Overview Stats -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon bg-blue">
          <v-icon icon="mdi-layers-outline" size="22" color="#0284C7" />
        </div>
        <div>
          <span class="stat-label">{{ l({ lo: 'ແພັກເກດທັງໝົດ', th: 'แพ็กเกจทั้งหมด', en: 'Total Plans' }) }}</span>
          <h3 class="stat-val">{{ totalPlans }}</h3>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-emerald">
          <v-icon icon="mdi-check-decagram" size="22" color="#059669" />
        </div>
        <div>
          <span class="stat-label">{{ l({ lo: 'ເປີດໃຫ້ບໍລິການ (Active)', th: 'เปิดให้บริการ (Active)', en: 'Active Plans' }) }}</span>
          <h3 class="stat-val text-emerald">{{ activePlans }}</h3>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-purple">
          <v-icon icon="mdi-storefront" size="22" color="#7C3AED" />
        </div>
        <div>
          <span class="stat-label">{{ l({ lo: 'ຮ້ານຄ້າທີ່ໃຊ້ງານຢູ່', th: 'ร้านค้าที่ใช้งานอยู่', en: 'Subscribed Stores' }) }}</span>
          <h3 class="stat-val text-purple">{{ totalSubscribers }}</h3>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon bg-amber">
          <v-icon icon="mdi-currency-usd" size="22" color="#D97706" />
        </div>
        <div>
          <span class="stat-label">{{ l({ lo: 'ລາຄາ Pro ມາດຕະຖານ', th: 'ราคา Pro มาตรฐาน', en: 'Standard Pro Price' }) }}</span>
          <h3 class="stat-val text-amber">500,000 ₭</h3>
        </div>
      </div>
    </div>

    <!-- Plans Visual Grid -->
    <div class="section-title">
      <h3>{{ l({ lo: 'ແພັກເກດທີ່ມີໃນລະບົບ (Available Plans)', th: 'แพ็กเกจที่มีในระบบ (Available Plans)', en: 'Available Plans' }) }}</h3>
      <span class="badge-count">{{ plans.length }}</span>
    </div>

    <div v-if="loading" class="loading-box">
      <v-progress-circular indeterminate color="#0284C7" size="40" />
      <span>{{ l({ lo: 'ກຳລັງໂຫຼດຂໍ້ມູນແພັກເກດ...', th: 'กำลังโหลดข้อมูลแพ็กเกจ...', en: 'Loading plans...' }) }}</span>
    </div>

    <div v-else class="plans-grid">
      <div
        v-for="p in plans"
        :key="p.id"
        class="plan-card"
        :class="{ inactive: !p.isActive, featured: p.id === 'pro' }"
      >
        <div v-if="p.id === 'pro'" class="featured-ribbon">
          {{ l({ lo: 'ແນະນຳ (POPULAR)', th: 'ยอดนิยม (POPULAR)', en: 'MOST POPULAR' }) }}
        </div>

        <div class="card-head">
          <div class="head-top">
            <span class="plan-slug">{{ p.id }}</span>
            <span class="status-pill" :class="p.isActive ? 'active' : 'inactive'">
              <span class="dot" />
              {{ p.isActive ? l({ lo: 'ເປີດໃຊ້', th: 'เปิดใช้', en: 'Active' }) : l({ lo: 'ປິດໃຊ້', th: 'ปิดใช้', en: 'Inactive' }) }}
            </span>
          </div>
          <h4 class="plan-name">{{ p.name }}</h4>
          <div class="price-wrap">
            <span class="price-number">{{ formatLAK(p.price) }}</span>
            <span v-if="p.price > 0" class="price-period">/ {{ l({ lo: 'ເດືອນ', th: 'เดือน', en: 'month' }) }}</span>
          </div>
        </div>

        <div class="card-body">
          <div class="subscriber-badge">
            <v-icon icon="mdi-storefront-outline" size="16" />
            <span>{{ p.subscriberCount || 0 }} {{ l({ lo: 'ຮ້ານຄ້າໃຊ້ງານ', th: 'ร้านค้าใช้งาน', en: 'stores subscribed' }) }}</span>
          </div>

          <ul class="features-list">
            <li>
              <v-icon icon="mdi-account-group-outline" size="16" color="#0284C7" />
              <span><strong>{{ p.maxUsers }}</strong> {{ l({ lo: 'ບັນຊີທີມງານ', th: 'บัญชีทีมงาน', en: 'team users' }) }}</span>
            </li>
            <li>
              <v-icon icon="mdi-facebook" size="16" color="#1877F2" />
              <span><strong>{{ p.maxPages }}</strong> {{ l({ lo: 'Facebook Pages ເຊື່ອມຕໍ່', th: 'Facebook Pages เชื่อมต่อ', en: 'connected Pages' }) }}</span>
            </li>
            <li>
              <v-icon icon="mdi-robot-outline" size="16" color="#059669" />
              <span>{{ l({ lo: 'ລະບົບ Auto-Reply & AI Chat', th: 'ระบบ Auto-Reply & AI Chat', en: 'Auto-Reply & AI Chat' }) }}</span>
            </li>
            <li>
              <v-icon icon="mdi-qrcode-scan" size="16" color="#D97706" />
              <span>{{ l({ lo: 'QR BCEL One & ອອກໃບບິນ', th: 'QR BCEL One & ออกใบส่งของ', en: 'BCEL One QR & Waybills' }) }}</span>
            </li>
          </ul>
        </div>

        <div class="card-footer">
          <button
            type="button"
            class="btn-icon-text"
            :disabled="actionLoading[p.id]"
            @click="openEditModal(p)"
          >
            <v-icon icon="mdi-pencil-outline" size="16" />
            <span>{{ l({ lo: 'ແກ້ໄຂ', th: 'แก้ไข', en: 'Edit' }) }}</span>
          </button>

          <button
            type="button"
            class="btn-icon-text"
            :class="p.isActive ? 'btn-warn' : 'btn-activate'"
            :disabled="actionLoading[p.id]"
            @click="togglePlanStatus(p)"
          >
            <v-icon :icon="p.isActive ? 'mdi-eye-off-outline' : 'mdi-eye-outline'" size="16" />
            <span>{{ p.isActive ? l({ lo: 'ປິດໃຊ້', th: 'ปิด', en: 'Disable' }) : l({ lo: 'ເປີດໃຊ້', th: 'เปิด', en: 'Enable' }) }}</span>
          </button>

          <button
            v-if="p.id !== 'free' && p.id !== 'pro'"
            type="button"
            class="btn-icon-text btn-danger"
            :disabled="actionLoading[p.id]"
            @click="deletePlan(p)"
          >
            <v-icon icon="mdi-delete-outline" size="16" />
          </button>
        </div>
      </div>
    </div>

    <!-- Comprehensive Plans Table -->
    <div class="table-container">
      <div class="table-header">
        <h4>{{ l({ lo: 'ຕາຕະລາງລາຍລະອຽດແພັກເກດ', th: 'ตารางรายละเอียดแพ็กเกจ', en: 'Plan Specifications Table' }) }}</h4>
      </div>
      <table class="data-table">
        <thead>
          <tr>
            <th>{{ l({ lo: 'ລະຫັດ (Slug)', th: 'รหัส (Slug)', en: 'Plan ID' }) }}</th>
            <th>{{ l({ lo: 'ຊື່ແພັກເກດ', th: 'ชื่อแพ็กเกจ', en: 'Plan Name' }) }}</th>
            <th>{{ l({ lo: 'ລາຄາ (ກີບ)', th: 'ราคา (กีบ)', en: 'Price (LAK)' }) }}</th>
            <th>{{ l({ lo: 'ໂຄຕ້າທີມງານ', th: 'โควตาทีมงาน', en: 'Max Users' }) }}</th>
            <th>{{ l({ lo: 'ໂຄຕ້າເພຈ', th: 'โควตาเพจ', en: 'Max Pages' }) }}</th>
            <th>{{ l({ lo: 'ຮ້ານຄ້າທີ່ໃຊ້', th: 'ร้านค้าที่ใช้', en: 'Subscribers' }) }}</th>
            <th>{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</th>
            <th class="text-right">{{ l({ lo: 'ຈັດການ', th: 'จัดการ', en: 'Actions' }) }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in plans" :key="'tbl-' + p.id">
            <td>
              <code class="plan-code">{{ p.id }}</code>
            </td>
            <td>
              <strong>{{ p.name }}</strong>
            </td>
            <td>
              <span class="price-tag">{{ formatLAK(p.price) }}</span>
            </td>
            <td>
              <span class="quota-pill">{{ p.maxUsers }} {{ l({ lo: 'ຄົນ', th: 'คน', en: 'users' }) }}</span>
            </td>
            <td>
              <span class="quota-pill">{{ p.maxPages }} {{ l({ lo: 'ເພຈ', th: 'เพจ', en: 'pages' }) }}</span>
            </td>
            <td>
              <span class="subscriber-count">{{ p.subscriberCount || 0 }}</span>
            </td>
            <td>
              <span class="status-pill" :class="p.isActive ? 'active' : 'inactive'">
                <span class="dot" />
                {{ p.isActive ? l({ lo: 'ເປີດໃຊ້', th: 'เปิดใช้', en: 'Active' }) : l({ lo: 'ປິດໃຊ້', th: 'ปิด', en: 'Inactive' }) }}
              </span>
            </td>
            <td class="text-right">
              <div class="table-actions">
                <button
                  type="button"
                  class="btn-tbl"
                  title="Edit"
                  @click="openEditModal(p)"
                >
                  <v-icon icon="mdi-pencil-outline" size="16" />
                </button>
                <button
                  type="button"
                  class="btn-tbl"
                  :title="p.isActive ? 'Deactivate' : 'Activate'"
                  @click="togglePlanStatus(p)"
                >
                  <v-icon :icon="p.isActive ? 'mdi-pause-circle-outline' : 'mdi-play-circle-outline'" size="16" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create / Edit Plan Modal -->
    <v-dialog v-model="planModalOpen" max-width="540">
      <div class="modal-card">
        <div class="modal-header">
          <div class="modal-title-wrap">
            <v-icon :icon="isEditing ? 'mdi-pencil' : 'mdi-plus-box'" size="22" color="#0284C7" />
            <h3>{{ isEditing ? l({ lo: 'ແກ້ໄຂແພັກເກດ', th: 'แก้ไขแพ็กเกจ', en: 'Edit Subscription Plan' }) : l({ lo: 'ສ້າງແພັກເກດໃໝ່', th: 'สร้างแพ็กเกจใหม่', en: 'Create New Plan' }) }}</h3>
          </div>
          <button type="button" class="close-btn" @click="planModalOpen = false">
            <v-icon icon="mdi-close" size="20" />
          </button>
        </div>

        <div class="modal-body">
          <div v-if="modalError" class="error-banner">
            <v-icon icon="mdi-alert-circle" size="18" />
            <span>{{ modalError }}</span>
          </div>

          <!-- Quick Presets for New Plans -->
          <div v-if="!isEditing" class="presets-box">
            <label class="field-label">{{ l({ lo: 'ເລືອກແມ່ແບບສຳເລັດຮູບ (Quick Presets):', th: 'เลือกเทมเพลตสำเร็จรูป (Quick Presets):', en: 'Quick Presets:' }) }}</label>
            <div class="presets-row">
              <button
                type="button"
                class="preset-chip"
                @click="applyPreset({ id: 'pro-quarterly', name: 'Pro 3 ເດືອນ (ຫຼຸດ 10%)', price: 1350000, maxUsers: 10, maxPages: 10 })"
              >
                Pro 3 ເດືອນ (1,350,000₭)
              </button>
              <button
                type="button"
                class="preset-chip"
                @click="applyPreset({ id: 'pro-yearly', name: 'Pro 1 ປີ (ຫຼຸດ 20%)', price: 4800000, maxUsers: 15, maxPages: 15 })"
              >
                Pro 1 ປີ (4,800,000₭)
              </button>
              <button
                type="button"
                class="preset-chip"
                @click="applyPreset({ id: 'vip-enterprise', name: 'VIP Enterprise', price: 1500000, maxUsers: 50, maxPages: 50 })"
              >
                VIP (1,500,000₭)
              </button>
            </div>
          </div>

          <div class="form-grid">
            <div class="form-group">
              <label class="field-label">
                {{ l({ lo: 'ລະຫັດແພັກເກດ (Plan ID)', th: 'รหัสแพ็กเกจ (Plan ID)', en: 'Plan ID' }) }}
                <small class="required">*</small>
              </label>
              <input
                v-model="form.id"
                type="text"
                class="form-input"
                :placeholder="l({ lo: 'ເຊັ່ນ: pro-yearly, starter', th: 'เช่น: pro-yearly, starter', en: 'e.g. pro-yearly, starter' })"
                :disabled="isEditing"
              >
              <small class="help-text">{{ l({ lo: 'ໃຊ້ຕົວອັກສອນພາສາອັງກິດ ແລະ ເຄື່ອງໝາຍຂີດກາງ (-)', th: 'ใช้ภาษาอังกฤษพิมพ์เล็กและขีด (-)', en: 'Lowercase letters and hyphens only' }) }}</small>
            </div>

            <div class="form-group">
              <label class="field-label">
                {{ l({ lo: 'ຊື່ແພັກເກດ (Display Name)', th: 'ชื่อแพ็กเกจ (Display Name)', en: 'Display Name' }) }}
                <small class="required">*</small>
              </label>
              <input
                v-model="form.name"
                type="text"
                class="form-input"
                :placeholder="l({ lo: 'ເຊັ່ນ: Pro Plan 1 ປີ', th: 'เช่น: Pro Plan 1 ปี', en: 'e.g. Pro Plan 1 Year' })"
              >
            </div>

            <div class="form-group">
              <label class="field-label">
                {{ l({ lo: 'ລາຄາຕໍ່ຮອບບິນ (LAK)', th: 'ราคาต่อรอบบิล (LAK)', en: 'Price (LAK)' }) }}
                <small class="required">*</small>
              </label>
              <input
                v-model.number="form.price"
                type="number"
                min="0"
                step="10000"
                class="form-input"
              >
              <span class="preview-price">{{ formatLAK(form.price) }}</span>
            </div>

            <div class="form-row-2">
              <div class="form-group">
                <label class="field-label">{{ l({ lo: 'ໂຄຕ້າທີມງານ (Max Users)', th: 'โควตาทีมงาน (Max Users)', en: 'Max Users' }) }}</label>
                <input
                  v-model.number="form.maxUsers"
                  type="number"
                  min="1"
                  class="form-input"
                >
              </div>

              <div class="form-group">
                <label class="field-label">{{ l({ lo: 'ໂຄຕ້າเพจ (Max Pages)', th: 'โควตาเพจ (Max Pages)', en: 'Max Pages' }) }}</label>
                <input
                  v-model.number="form.maxPages"
                  type="number"
                  min="1"
                  class="form-input"
                >
              </div>
            </div>

            <div class="form-group toggle-group">
              <label class="toggle-label">
                <input v-model="form.isActive" type="checkbox" class="toggle-checkbox">
                <span>{{ l({ lo: 'ເປີດໃຫ້ຮ້ານຄ້າເລືອກໃຊ້ງານ (Is Active)', th: 'เปิดให้ร้านค้าเลือกใช้งาน (Is Active)', en: 'Active & Available for Merchants' }) }}</span>
              </label>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" class="btn-secondary" @click="planModalOpen = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </button>
          <button type="button" class="btn-primary" :disabled="saving" @click="savePlan">
            <v-progress-circular v-if="saving" indeterminate size="18" color="white" />
            <span v-else>{{ l({ lo: 'ບັນທຶກແພັກເກດ', th: 'บันทึกแพ็กเกจ', en: 'Save Plan' }) }}</span>
          </button>
        </div>
      </div>
    </v-dialog>

    <!-- Assign Plan Directly to Store Modal -->
    <v-dialog v-model="assignModalOpen" max-width="500">
      <div class="modal-card">
        <div class="modal-header">
          <div class="modal-title-wrap">
            <v-icon icon="mdi-account-arrow-right" size="22" color="#7C3AED" />
            <h3>{{ l({ lo: 'ມອບສິດແພັກເກດໃຫ້ຮ້ານຄ້າ', th: 'มอบสิทธิ์แพ็กเกจให้ร้านค้า', en: 'Assign Plan to Store' }) }}</h3>
          </div>
          <button type="button" class="close-btn" @click="assignModalOpen = false">
            <v-icon icon="mdi-close" size="20" />
          </button>
        </div>

        <div class="modal-body">
          <div v-if="assignError" class="error-banner">
            <v-icon icon="mdi-alert-circle" size="18" />
            <span>{{ assignError }}</span>
          </div>

          <p class="modal-intro">
            {{ l({ lo: 'Admin ສາມາດປັບແພັກເກດ ແລະ ເພີ່ມວັນໃຊ້ງານໃຫ້ຮ້ານຄ້າໃດກໍ່ໄດ້ໂດຍກົງ ໂດຍບໍ່ຕ້ອງຜ່ານການກວດສະລິບ.', th: 'Admin สามารถปรับแพ็กเกจและเพิ่มวันใช้งานให้ร้านค้าใดก็ได้โดยตรง', en: 'Super Admin can manually grant or adjust subscriptions for any store.' }) }}
          </p>

          <div class="form-grid">
            <div class="form-group">
              <label class="field-label">{{ l({ lo: 'ເລືອກຮ້ານຄ້າ (Store / Workspace)', th: 'เลือกร้านค้า (Store / Workspace)', en: 'Select Store' }) }}</label>
              <select v-model="assignForm.accountId" class="form-select">
                <option v-for="t in tenantsList" :key="t.id" :value="t.id">
                  {{ t.name }} (ປັດຈຸບັນ: {{ t.planName }})
                </option>
              </select>
            </div>

            <div class="form-group">
              <label class="field-label">{{ l({ lo: 'ເລືອກແພັກເກດ (Plan)', th: 'เลือกแพ็กเกจ (Plan)', en: 'Select Plan' }) }}</label>
              <select v-model="assignForm.planId" class="form-select">
                <option v-for="p in plans" :key="'assign-plan-' + p.id" :value="p.id">
                  {{ p.name }} ({{ formatLAK(p.price) }})
                </option>
              </select>
            </div>

            <div class="form-group">
              <label class="field-label">{{ l({ lo: 'ໄລຍະເວລາຕໍ່ອາຍຸ (Duration)', th: 'ระยะเวลาต่ออายุ (Duration)', en: 'Duration' }) }}</label>
              <select v-model.number="assignForm.days" class="form-select">
                <option :value="7">{{ l({ lo: '7 ວັນ (Trial)', th: '7 วัน (Trial)', en: '7 Days (Trial)' }) }}</option>
                <option :value="30">{{ l({ lo: '30 ວັນ (1 ເດືອນ)', th: '30 วัน (1 เดือน)', en: '30 Days (1 Month)' }) }}</option>
                <option :value="90">{{ l({ lo: '90 ວັນ (3 ເດືອນ)', th: '90 วัน (3 เดือน)', en: '90 Days (3 Months)' }) }}</option>
                <option :value="180">{{ l({ lo: '180 ວັນ (6 ເດືອນ)', th: '180 วัน (6 เดือน)', en: '180 Days (6 Months)' }) }}</option>
                <option :value="365">{{ l({ lo: '365 ວັນ (1 ປີ)', th: '365 วัน (1 ปี)', en: '365 Days (1 Year)' }) }}</option>
                <option :value="0">{{ l({ lo: 'ຕະຫຼອດຊີບ / ບໍ່ມີກຳນົດ (Lifetime / Free)', th: 'ตลอดชีพ / ไม่มีกำหนด (Lifetime / Free)', en: 'Lifetime / No Expiry' }) }}</option>
              </select>
            </div>

            <div class="form-group">
              <label class="field-label">{{ l({ lo: 'ສະຖານະ (Status)', th: 'สถานะ (Status)', en: 'Subscription Status' }) }}</label>
              <select v-model="assignForm.status" class="form-select">
                <option value="active">{{ l({ lo: 'ໃຊ້ງານຢູ່ (Active)', th: 'ใช้งานอยู่ (Active)', en: 'Active' }) }}</option>
                <option value="trialing">{{ l({ lo: 'ທົດລອງໃຊ້ (Trialing)', th: 'ทดลองใช้ (Trialing)', en: 'Trialing' }) }}</option>
              </select>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" class="btn-secondary" @click="assignModalOpen = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </button>
          <button type="button" class="btn-primary" :disabled="assigning" @click="submitAssign">
            <v-progress-circular v-if="assigning" indeterminate size="18" color="white" />
            <span v-else>{{ l({ lo: 'ຢືນຢັນການມອບສິດ', th: 'ยืนยันการมอบสิทธิ์', en: 'Confirm Assignment' }) }}</span>
          </button>
        </div>
      </div>
    </v-dialog>
  </div>
</template>

<style scoped>
.plans-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.icon-wrap {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background-color: #E0F2FE;
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-header h1 {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: 800;
  color: var(--color-text);
}

.page-header p {
  margin: 4px 0 0 0;
  font-size: var(--text-sm);
  color: #64748B;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

/* Stats Row */
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-3);
}

.stat-card {
  background-color: #FFFFFF;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  padding: var(--space-3);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  box-shadow: var(--shadow-sm);
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.bg-blue { background-color: #E0F2FE; }
.bg-emerald { background-color: #D1FAE5; }
.bg-purple { background-color: #EDE9FE; }
.bg-amber { background-color: #FEF3C7; }

.stat-label {
  font-size: var(--text-xs);
  color: #64748B;
  font-weight: 600;
  display: block;
}

.stat-val {
  margin: 2px 0 0 0;
  font-size: var(--text-xl);
  font-weight: 800;
}

.text-emerald { color: #059669; }
.text-purple { color: #7C3AED; }
.text-amber { color: #D97706; }

/* Section Title */
.section-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.section-title h3 {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
}

.badge-count {
  background-color: #F1F5F9;
  color: #475569;
  font-size: var(--text-xs);
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 9999px;
}

.loading-box {
  padding: var(--space-6);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  color: #64748B;
  background-color: #FFFFFF;
  border-radius: var(--radius-md);
}

/* Plans Grid */
.plans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-3);
}

.plan-card {
  background-color: #FFFFFF;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
  transition: transform 0.2s, box-shadow 0.2s;
}

.plan-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.plan-card.featured {
  border: 2px solid #0284C7;
}

.featured-ribbon {
  background: linear-gradient(135deg, #0284C7, #2563EB);
  color: #FFFFFF;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.5px;
  text-align: center;
  padding: 4px;
}

.plan-card.inactive {
  opacity: 0.65;
  background-color: #F8FAFC;
}

.card-head {
  padding: var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.head-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-1);
}

.plan-slug {
  font-family: monospace;
  font-size: 11px;
  background-color: #F1F5F9;
  color: #334155;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 600;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 8px;
  border-radius: 9999px;
  font-size: 11px;
  font-weight: 600;
}

.status-pill.active {
  background-color: #D1FAE5;
  color: #065F46;
}

.status-pill.inactive {
  background-color: #F1F5F9;
  color: #64748B;
}

.status-pill .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: currentColor;
}

.plan-name {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 800;
  color: var(--color-text);
}

.price-wrap {
  margin-top: 6px;
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.price-number {
  font-size: 22px;
  font-weight: 800;
  color: #0F172A;
}

.price-period {
  font-size: var(--text-xs);
  color: #64748B;
}

.card-body {
  padding: var(--space-3);
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.subscriber-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: #F8FAFC;
  border: 1px solid #E2E8F0;
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 600;
  color: #475569;
}

.features-list {
  list-style: none;
  padding: 0;
  margin: 8px 0 0 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--text-sm);
  color: #334155;
}

.features-list li {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-footer {
  padding: var(--space-2) var(--space-3);
  background-color: #F8FAFC;
  border-top: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.btn-icon-text {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background-color: #FFFFFF;
  font-size: var(--text-xs);
  font-weight: 600;
  color: #334155;
  cursor: pointer;
}

.btn-icon-text:hover {
  background-color: #F1F5F9;
}

.btn-warn {
  color: #B45309;
}

.btn-activate {
  color: #059669;
}

.btn-danger {
  color: #DC2626;
  margin-left: auto;
}

/* Table */
.table-container {
  background-color: #FFFFFF;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
  margin-top: var(--space-2);
}

.table-header {
  padding: var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.table-header h4 {
  margin: 0;
  font-size: var(--text-md);
  font-weight: 700;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: var(--text-sm);
}

.data-table th {
  background-color: #F8FAFC;
  padding: 10px 14px;
  font-weight: 600;
  color: #475569;
  border-bottom: 1px solid var(--color-border);
}

.data-table td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--color-border);
  color: #334155;
}

.data-table tr:last-child td {
  border-bottom: 0;
}

.plan-code {
  background-color: #F1F5F9;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
}

.price-tag {
  font-weight: 700;
  color: #0F172A;
}

.quota-pill {
  background-color: #F1F5F9;
  padding: 3px 8px;
  border-radius: 6px;
  font-weight: 600;
  font-size: 12px;
}

.subscriber-count {
  font-weight: 700;
  color: #7C3AED;
}

.table-actions {
  display: inline-flex;
  gap: 6px;
}

.btn-tbl {
  background-color: #F8FAFC;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #475569;
}

.btn-tbl:hover {
  background-color: #F1F5F9;
  color: #0284C7;
}

/* Modals */
.modal-card {
  background-color: #FFFFFF;
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.modal-title-wrap {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.modal-title-wrap h3 {
  margin: 0;
  font-size: var(--text-md);
  font-weight: 700;
}

.close-btn {
  background: transparent;
  border: 0;
  cursor: pointer;
  color: #64748B;
}

.modal-body {
  padding: var(--space-4);
  max-height: 75vh;
  overflow-y: auto;
}

.modal-intro {
  margin: 0 0 var(--space-3) 0;
  font-size: var(--text-xs);
  color: #64748B;
  line-height: 1.5;
}

.error-banner {
  background-color: #FEE2E2;
  color: #B91C1C;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: var(--space-3);
}

.presets-box {
  margin-bottom: var(--space-3);
  padding: var(--space-2);
  background-color: #F8FAFC;
  border-radius: var(--radius-sm);
  border: 1px solid #E2E8F0;
}

.presets-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
}

.preset-chip {
  background-color: #FFFFFF;
  border: 1px solid #CBD5E1;
  padding: 4px 8px;
  border-radius: 9999px;
  font-size: 11px;
  font-weight: 600;
  color: #334155;
  cursor: pointer;
}

.preset-chip:hover {
  background-color: #E0F2FE;
  border-color: #0284C7;
  color: #0284C7;
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.form-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-label {
  font-size: var(--text-xs);
  font-weight: 600;
  color: #334155;
}

.required {
  color: #EF4444;
}

.help-text {
  font-size: 11px;
  color: #94A3B8;
}

.form-input, .form-select {
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  outline: none;
  background-color: #FFFFFF;
}

.form-input:focus, .form-select:focus {
  border-color: #0284C7;
  box-shadow: 0 0 0 2px rgba(2, 132, 199, 0.15);
}

.preview-price {
  font-size: 12px;
  font-weight: 700;
  color: #0284C7;
}

.toggle-group {
  margin-top: 4px;
}

.toggle-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: var(--text-sm);
  font-weight: 600;
  color: #334155;
  cursor: pointer;
}

.toggle-checkbox {
  width: 18px;
  height: 18px;
  accent-color: #0284C7;
}

.modal-footer {
  padding: var(--space-3) var(--space-4);
  background-color: #F8FAFC;
  border-top: 1px solid var(--color-border);
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

/* Global Buttons */
.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: #0284C7;
  color: #FFFFFF;
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  border: 0;
  font-size: var(--text-sm);
  font-weight: 600;
  cursor: pointer;
}

.btn-primary:hover {
  background-color: #0369A1;
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: #FFFFFF;
  color: #334155;
  border: 1px solid var(--color-border);
  padding: 8px 14px;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 600;
  cursor: pointer;
}

.btn-secondary:hover {
  background-color: #F8FAFC;
}

.text-right {
  text-align: right;
}
</style>
