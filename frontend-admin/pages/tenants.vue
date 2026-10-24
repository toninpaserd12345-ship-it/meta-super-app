<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

const { l } = useAdminLocale()
const { adminFetch } = useAdminApi()

interface TenantUser {
  id: string
  name: string
  email: string
  role: string
  createdAt: string
}

interface TenantPage {
  pageId: string
  name: string
  category: string
  pictureUrl: string
  isConnected: boolean
  createdAt: string
}

interface Tenant {
  id: string
  name: string
  slug: string
  status: string
  ownerName: string
  ownerEmail: string
  userCount: number
  connectedPagesCount: number
  planId: string
  planName: string
  planStatus: string
  lockedPrice?: number
  currentPeriodEnd?: string
  users?: TenantUser[]
  pages?: TenantPage[]
  createdAt: string
}

interface GlobalPage {
  pageId: string
  pageName: string
  category: string
  pictureUrl: string
  isConnected: boolean
  accountId: string
  storeName: string
  createdAt: string
}

interface GlobalUser {
  userId: string
  userName: string
  email: string
  isSuperAdmin: boolean
  accountId: string
  storeName: string
  role: string
  createdAt: string
}

const currentTab = ref<'tenants' | 'pages' | 'users'>('tenants')
const loading = ref(true)
const tenants = ref<Tenant[]>([])
const search = ref('')
const filterStatus = ref<'all' | 'active' | 'suspended'>('all')
const actionLoading = ref<Record<string, boolean>>({})

// Pages & Users global data
const allPages = ref<GlobalPage[]>([])
const allUsers = ref<GlobalUser[]>([])
const pagesLoading = ref(false)
const usersLoading = ref(false)
const pageSearch = ref('')
const userSearch = ref('')

// Dialog detail state
const selectedTenant = ref<Tenant | null>(null)
const detailDialog = ref(false)

const loadTenants = async () => {
  loading.value = true
  try {
    const query = search.value.trim() ? `?search=${encodeURIComponent(search.value.trim())}` : ''
    const res = await adminFetch<{ items: Tenant[] }>(`/api/v1/admin/tenants${query}`)
    tenants.value = res.items || []
  } catch (err) {
    console.error('Failed to load tenants:', err)
  } finally {
    loading.value = false
  }
}

const loadPages = async () => {
  pagesLoading.value = true
  try {
    const res = await adminFetch<{ items: GlobalPage[] }>('/api/v1/admin/pages')
    allPages.value = res.items || []
  } catch (err) {
    console.error('Failed to load pages:', err)
  } finally {
    pagesLoading.value = false
  }
}

const loadUsers = async () => {
  usersLoading.value = true
  try {
    const res = await adminFetch<{ items: GlobalUser[] }>('/api/v1/admin/users')
    allUsers.value = res.items || []
  } catch (err) {
    console.error('Failed to load users:', err)
  } finally {
    usersLoading.value = false
  }
}

const switchTab = (tab: 'tenants' | 'pages' | 'users') => {
  currentTab.value = tab
  if (tab === 'pages') loadPages()
  if (tab === 'users') loadUsers()
  if (tab === 'tenants') loadTenants()
}

const filteredTenants = computed(() => {
  if (filterStatus.value === 'all') return tenants.value
  return tenants.value.filter(t => t.status === filterStatus.value)
})

const filteredPages = computed(() => {
  const q = pageSearch.value.trim().toLowerCase()
  if (!q) return allPages.value
  return allPages.value.filter(p =>
    (p.pageName || '').toLowerCase().includes(q) ||
    (p.pageId || '').toLowerCase().includes(q) ||
    (p.storeName || '').toLowerCase().includes(q) ||
    (p.category || '').toLowerCase().includes(q)
  )
})

const filteredUsers = computed(() => {
  const q = userSearch.value.trim().toLowerCase()
  if (!q) return allUsers.value
  return allUsers.value.filter(u =>
    (u.userName || '').toLowerCase().includes(q) ||
    (u.email || '').toLowerCase().includes(q) ||
    (u.storeName || '').toLowerCase().includes(q) ||
    (u.role || '').toLowerCase().includes(q)
  )
})

const toggleStatus = async (tenant: Tenant) => {
  const newStatus = tenant.status === 'active' ? 'suspended' : 'active'
  const confirmMsg = newStatus === 'suspended'
    ? l({
        lo: `ທ່ານແນ່ໃຈບໍ່ວ່າຕ້ອງການລະງັບ (Lock) ຮ້ານຄ້າ "${tenant.name}"? ລູກຄ້າຈະບໍ່ສາມາດເຂົ້າໃຊ້ງານລະບົບໄດ້.`,
        th: `คุณแน่ใจหรือไม่ว่าต้องการระงับ (Lock) ร้านค้า "${tenant.name}"? ลูกค้าจะไม่สามารถเข้าใช้งานระบบได้`,
        en: `Are you sure you want to suspend "${tenant.name}"? Users of this workspace will be immediately blocked.`
      })
    : l({
        lo: `ປົດລັອກ ແລະ ເປີດໃຊ້ງານຮ້ານຄ້າ "${tenant.name}" ຄືນໃໝ່?`,
        th: `ปลดล็อกและเปิดใช้งานร้านค้า "${tenant.name}" อีกครั้ง?`,
        en: `Reactivate workspace "${tenant.name}"?`
      })

  if (!confirm(confirmMsg)) return

  actionLoading.value[tenant.id] = true
  try {
    await adminFetch(`/api/v1/admin/tenants/${tenant.id}/status`, {
      method: 'PUT',
      body: { status: newStatus },
    })
    await loadTenants()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Failed to update tenant status')
  } finally {
    actionLoading.value[tenant.id] = false
  }
}

const deleteTenant = async (tenant: Tenant) => {
  const confirmMsg = l({
    lo: `⚠️ ຄຳເຕືອນ: ທ່ານແນ່ໃຈບໍ່ວ່າຕ້ອງການລຶບຮ້ານ "${tenant.name}"? ຂໍ້ມູນທັງໝົດຂອງຮ້ານນີ້ຈະຖືກລຶບອອກຈາກລະບົບຢ່າງຖາວອນ!`,
    th: `⚠️ คำเตือน: คุณแน่ใจหรือไม่ว่าต้องการลบร้าน "${tenant.name}"? ข้อมูลทั้งหมดจะถูกลบถาวร`,
    en: `⚠️ Caution: Are you sure you want to permanently delete "${tenant.name}"?`
  })

  if (!confirm(confirmMsg)) return

  actionLoading.value[tenant.id] = true
  try {
    await adminFetch(`/api/v1/admin/tenants/${tenant.id}`, { method: 'DELETE' })
    await loadTenants()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Failed to delete tenant')
  } finally {
    actionLoading.value[tenant.id] = false
  }
}

const openDetail = (tenant: Tenant) => {
  selectedTenant.value = tenant
  detailDialog.value = true
}

const formatDate = (val?: string) => {
  if (!val) return '-'
  return new Date(val).toLocaleDateString()
}

// Plan adjustment in tenant detail
const changePlanDialog = ref(false)
const changePlanLoading = ref(false)
const availablePlans = ref<{ id: string; name: string; price: number }[]>([])
const changePlanForm = ref({
  planId: 'pro',
  days: 30,
  lockedPrice: 500000,
  status: 'active',
})

const openTenantPlanChange = async () => {
  if (!selectedTenant.value) return
  changePlanForm.value = {
    planId: selectedTenant.value.planId || 'pro',
    days: 30,
    lockedPrice: selectedTenant.value.lockedPrice || 0,
    status: 'active',
  }
  try {
    const res = await adminFetch<{ items: { id: string; name: string; price: number }[] }>('/api/v1/admin/plans')
    availablePlans.value = res.items || []
    if (changePlanForm.value.lockedPrice <= 0) {
      const match = availablePlans.value.find(p => p.id === changePlanForm.value.planId)
      if (match) changePlanForm.value.lockedPrice = match.price
    }
  } catch (err) {
    console.error('Failed to load plans:', err)
  }
  changePlanDialog.value = true
}

const submitTenantPlanChange = async () => {
  if (!selectedTenant.value) return
  changePlanLoading.value = true
  try {
    await adminFetch(`/api/v1/admin/tenants/${selectedTenant.value.id}/plan`, {
      method: 'PUT',
      body: {
        planId: changePlanForm.value.planId,
        days: Number(changePlanForm.value.days) || 30,
        lockedPrice: Number(changePlanForm.value.lockedPrice) || 0,
        status: changePlanForm.value.status,
      },
    })
    changePlanDialog.value = false
    detailDialog.value = false
    alert(l({
      lo: 'ປ່ຽນແພັກເກດໃຫ້ຮ້ານຄ້າສຳເລັດແລ້ວ!',
      th: 'เปลี่ยนแพ็กเกจให้ร้านค้าสำเร็จแล้ว!',
      en: 'Store subscription plan updated!'
    }))
    await loadTenants()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Failed to update plan')
  } finally {
    changePlanLoading.value = false
  }
}

onMounted(loadTenants)
</script>

<template>
  <div class="tenants-page">
    <!-- Page Title & Header Actions -->
    <div class="page-title-row">
      <div>
        <h1>{{ l({ lo: 'ຈັດການຮ້ານຄ້າ, Pages ແລະ Users', th: 'จัดการร้านค้า, Pages และ Users', en: 'Tenants, Pages & Users Management' }) }}</h1>
        <p>{{ l({ lo: 'ກວດສອບຮ້ານຄ້າທັງໝົດ, Facebook Pages ທີ່ເຊື່ອມຕໍ່ ແລະ ລາຍຊື່ Users ຈັດກຸ່ມຕາມແຕ່ລະຮ້ານ', th: 'ตรวจสอบร้านค้าทั้งหมด, Facebook Pages ที่เชื่อมต่อ และรายชื่อ Users จัดกลุ่มตามร้าน', en: 'Monitor stores, connected Facebook pages, and users grouped by store' }) }}</p>
      </div>
      <button type="button" class="btn-secondary" @click="currentTab === 'pages' ? loadPages() : currentTab === 'users' ? loadUsers() : loadTenants()">
        <v-icon icon="mdi-refresh" size="18" /> {{ l({ lo: 'ໂຫຼດຂໍ້ມູນໃໝ່', th: 'โหลดข้อมูลใหม่', en: 'Refresh Data' }) }}
      </button>
    </div>

    <!-- Navigation Tabs (Stores / Pages / Users by Store) -->
    <div class="main-tabs-bar">
      <button
        type="button"
        class="tab-btn"
        :class="{ active: currentTab === 'tenants' }"
        @click="switchTab('tenants')"
      >
        <v-icon icon="mdi-storefront-outline" size="18" />
        <span>{{ l({ lo: 'ຮ້ານຄ້າທັງໝົດ (Stores)', th: 'ร้านค้าทั้งหมด (Stores)', en: 'All Stores' }) }}</span>
        <span class="count-pill">{{ tenants.length }}</span>
      </button>

      <button
        type="button"
        class="tab-btn"
        :class="{ active: currentTab === 'pages' }"
        @click="switchTab('pages')"
      >
        <v-icon icon="mdi-facebook" size="18" color="#1877f2" />
        <span>{{ l({ lo: 'Facebook Pages ທັງໝົດ', th: 'Facebook Pages ทั้งหมด', en: 'All Facebook Pages' }) }}</span>
        <span v-if="allPages.length" class="count-pill">{{ allPages.length }}</span>
      </button>

      <button
        type="button"
        class="tab-btn"
        :class="{ active: currentTab === 'users' }"
        @click="switchTab('users')"
      >
        <v-icon icon="mdi-account-group-outline" size="18" />
        <span>{{ l({ lo: 'Users ຈັດຕາມຮ້ານຄ້າ (Users by Store)', th: 'Users จัดตามร้านค้า', en: 'Users Grouped by Store' }) }}</span>
        <span v-if="allUsers.length" class="count-pill">{{ allUsers.length }}</span>
      </button>
    </div>

    <!-- TAB 1: TENANTS LIST -->
    <div v-if="currentTab === 'tenants'" class="tab-content">
      <!-- Filter & Search Toolbar -->
      <div class="toolbar-card">
        <div class="search-box">
          <v-icon icon="mdi-magnify" size="20" color="grey" />
          <input
            v-model="search"
            type="text"
            :placeholder="l({ lo: 'ຄົ້ນຫາຊື່ຮ້ານ, Slug...', th: 'ค้นหาชื่อร้าน, Slug...', en: 'Search shop name, slug...' })"
            @keyup.enter="loadTenants"
          />
          <button v-if="search" type="button" class="clear-search" @click="search = ''; loadTenants()">
            <v-icon icon="mdi-close" size="16" />
          </button>
        </div>

        <div class="filter-tabs">
          <button
            type="button"
            :class="{ active: filterStatus === 'all' }"
            @click="filterStatus = 'all'"
          >
            {{ l({ lo: 'ທັງໝົດ', th: 'ทั้งหมด', en: 'All' }) }} ({{ tenants.length }})
          </button>
          <button
            type="button"
            :class="{ active: filterStatus === 'active' }"
            @click="filterStatus = 'active'"
          >
            {{ l({ lo: 'ເປີດໃຊ້ງານ', th: 'เปิดใช้งาน', en: 'Active' }) }} ({{ tenants.filter(t => t.status === 'active').length }})
          </button>
          <button
            type="button"
            :class="{ active: filterStatus === 'suspended' }"
            @click="filterStatus = 'suspended'"
          >
            {{ l({ lo: 'ຖືກລະງັບ (Locked)', th: 'ถูกระงับ (Locked)', en: 'Suspended' }) }} ({{ tenants.filter(t => t.status === 'suspended').length }})
          </button>
        </div>
      </div>

      <!-- Tenants Table -->
      <div class="table-card">
        <div v-if="loading" class="empty-state">
          <v-progress-circular indeterminate color="primary" size="32" />
          <p>{{ l({ lo: 'ກຳລັງໂຫຼດລາຍຊື່ຮ້ານຄ້າ...', th: 'กำลังโหลดรายชื่อร้านค้า...', en: 'Loading tenants...' }) }}</p>
        </div>

        <div v-else-if="!filteredTenants.length" class="empty-state">
          <v-icon icon="mdi-storefront-outline" size="36" color="grey" />
          <p>{{ l({ lo: 'ບໍ່ພົບຮ້ານຄ້າທີ່ຄົ້ນຫາ', th: 'ไม่พบร้านค้าที่ค้นหา', en: 'No matching tenants found' }) }}</p>
        </div>

        <div v-else class="table-responsive">
          <table class="admin-table">
            <thead>
              <tr>
                <th>{{ l({ lo: 'ຮ້ານຄ້າ (Shop)', th: 'ร้านค้า (Shop)', en: 'Shop' }) }}</th>
                <th>{{ l({ lo: 'ເຈົ້າຂອງຮ້ານ (Owner)', th: 'เจ้าของร้าน (Owner)', en: 'Owner' }) }}</th>
                <th>{{ l({ lo: 'Users ໃນຮ້ານ', th: 'Users ในร้าน', en: 'Users' }) }}</th>
                <th>{{ l({ lo: 'FB Pages', th: 'FB Pages', en: 'Pages' }) }}</th>
                <th>{{ l({ lo: 'ແພັກເກດ (Plan)', th: 'แพ็กเกจ (Plan)', en: 'Plan' }) }}</th>
                <th>{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</th>
                <th class="text-right">{{ l({ lo: 'ຈັດການ / ລາຍລະອຽດ', th: 'จัดการ / รายละเอียด', en: 'Actions' }) }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="t in filteredTenants" :key="t.id">
                <td>
                  <div class="shop-cell">
                    <strong>{{ t.name }}</strong>
                    <small>{{ t.id }}</small>
                  </div>
                </td>
                <td>
                  <div class="owner-cell">
                    <span>{{ t.ownerName || '-' }}</span>
                    <small>{{ t.ownerEmail || '-' }}</small>
                  </div>
                </td>
                <td>
                  <button type="button" class="user-count-chip" @click="openDetail(t)">
                    <v-icon icon="mdi-account-multiple" size="14" />
                    <span>{{ t.users?.length || t.userCount || 0 }} {{ l({ lo: 'ຄົນ', th: 'คน', en: 'users' }) }}</span>
                  </button>
                </td>
                <td>
                  <button type="button" class="pages-badge-btn" @click="openDetail(t)">
                    <v-icon icon="mdi-facebook" size="14" />
                    <span>{{ t.pages?.length || t.connectedPagesCount || 0 }} {{ l({ lo: 'Pages', th: 'Pages', en: 'Pages' }) }}</span>
                  </button>
                </td>
                <td>
                  <span class="plan-chip" :class="t.planId">
                    {{ t.planName }}
                    <small v-if="t.planStatus === 'trialing'">(7D Trial)</small>
                  </span>
                </td>
                <td>
                  <span class="status-indicator" :class="t.status">
                    <span class="dot" />
                    {{ t.status === 'active' ? l({ lo: 'ເປີດໃຊ້', th: 'เปิดใช้', en: 'Active' }) : l({ lo: 'ລະງັບ', th: 'ระงับ', en: 'Suspended' }) }}
                  </span>
                </td>
                <td class="text-right">
                  <div class="action-buttons">
                    <!-- View Users & Pages Details -->
                    <button
                      type="button"
                      class="btn-detail"
                      :title="l({ lo: 'ເບິ່ງລາຍລະອຽດ, ສະມາຊິກ ແລະ Pages', th: 'ดูรายละเอียด สมาชิก และ Pages', en: 'View Users & Pages' })"
                      @click="openDetail(t)"
                    >
                      <v-icon icon="mdi-eye-outline" size="16" />
                      {{ l({ lo: 'ລາຍລະອຽດ', th: 'รายละเอียด', en: 'Details' }) }}
                    </button>

                    <!-- Lock / Unlock Toggle -->
                    <button
                      v-if="t.status === 'active'"
                      type="button"
                      class="btn-secondary lock-btn"
                      title="Suspend / Lock shop"
                      :disabled="actionLoading[t.id]"
                      @click="toggleStatus(t)"
                    >
                      <v-icon icon="mdi-lock-outline" size="15" />
                      {{ l({ lo: 'ລະງັບ', th: 'ระงับ', en: 'Lock' }) }}
                    </button>
                    <button
                      v-else
                      type="button"
                      class="btn-success unlock-btn"
                      title="Unlock shop"
                      :disabled="actionLoading[t.id]"
                      @click="toggleStatus(t)"
                    >
                      <v-icon icon="mdi-lock-open-outline" size="15" />
                      {{ l({ lo: 'ປົດລັອກ', th: 'ปลดล็อก', en: 'Unlock' }) }}
                    </button>

                    <!-- Delete -->
                    <button
                      type="button"
                      class="btn-danger delete-btn"
                      title="Delete Workspace"
                      :disabled="actionLoading[t.id]"
                      @click="deleteTenant(t)"
                    >
                      <v-icon icon="mdi-delete-outline" size="15" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 2: ALL FACEBOOK PAGES -->
    <div v-if="currentTab === 'pages'" class="tab-content">
      <div class="toolbar-card">
        <div class="search-box">
          <v-icon icon="mdi-magnify" size="20" color="grey" />
          <input
            v-model="pageSearch"
            type="text"
            :placeholder="l({ lo: 'ຄົ້ນຫາຊື່ Page, Page ID, ຊື່ຮ້ານຄ້າ...', th: 'ค้นหาชื่อ Page, Page ID, ชื่อร้าน...', en: 'Search page name, ID, store...' })"
          />
        </div>
        <div class="info-tag">
          {{ l({ lo: 'ລວມທັງໝົດ', th: 'รวมทั้งหมด', en: 'Total' }) }}: <strong>{{ allPages.length }}</strong> Pages
        </div>
      </div>

      <div class="table-card">
        <div v-if="pagesLoading" class="empty-state">
          <v-progress-circular indeterminate color="primary" size="32" />
          <p>{{ l({ lo: 'ກຳລັງດຶງຂໍ້ມູນ Facebook Pages...', th: 'กำลังดึงข้อมูล Facebook Pages...', en: 'Fetching Facebook Pages...' }) }}</p>
        </div>

        <div v-else-if="!filteredPages.length" class="empty-state">
          <v-icon icon="mdi-facebook" size="36" color="#94a3b8" />
          <p>{{ l({ lo: 'ຍັງບໍ່ມີ Facebook Page ໃນລະບົບ', th: 'ยังไม่มี Facebook Page ในระบบ', en: 'No Facebook pages found' }) }}</p>
        </div>

        <div v-else class="table-responsive">
          <table class="admin-table">
            <thead>
              <tr>
                <th>{{ l({ lo: 'Facebook Page', th: 'Facebook Page', en: 'Facebook Page' }) }}</th>
                <th>{{ l({ lo: 'Page ID', th: 'Page ID', en: 'Page ID' }) }}</th>
                <th>{{ l({ lo: 'ໝວດໝູ່ (Category)', th: 'หมวดหมู่ (Category)', en: 'Category' }) }}</th>
                <th>{{ l({ lo: 'ຮ້ານຄ້າທີ່ຄອບຄອງ (Belongs to Shop)', th: 'ร้านค้าที่ครอบครอง', en: 'Assigned Shop' }) }}</th>
                <th>{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</th>
                <th>{{ l({ lo: 'ວັນທີເຊື່ອມຕໍ່', th: 'วันที่เชื่อมต่อ', en: 'Connected At' }) }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in filteredPages" :key="p.pageId">
                <td>
                  <div class="page-title-cell">
                    <img v-if="p.pictureUrl" :src="p.pictureUrl" class="page-avatar" alt="Page avatar" />
                    <div v-else class="page-avatar-fallback">
                      <v-icon icon="mdi-facebook" size="18" color="white" />
                    </div>
                    <strong>{{ p.pageName || 'Unnamed Page' }}</strong>
                  </div>
                </td>
                <td>
                  <code>{{ p.pageId }}</code>
                </td>
                <td>
                  <span class="category-chip">{{ p.category || 'General' }}</span>
                </td>
                <td>
                  <div class="shop-link-cell">
                    <v-icon icon="mdi-storefront" size="15" color="#2563eb" />
                    <strong>{{ p.storeName }}</strong>
                  </div>
                </td>
                <td>
                  <span class="status-indicator" :class="p.isConnected ? 'active' : 'suspended'">
                    <span class="dot" />
                    {{ p.isConnected ? l({ lo: 'ເຊື່ອມຕໍ່ແລ້ວ', th: 'เชื่อมต่อแล้ว', en: 'Connected' }) : l({ lo: 'ຕັດການເຊື່ອມ', th: 'ตัดการเชื่อม', en: 'Disconnected' }) }}
                  </span>
                </td>
                <td>
                  <small>{{ formatDate(p.createdAt) }}</small>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 3: USERS GROUPED BY STORE -->
    <div v-if="currentTab === 'users'" class="tab-content">
      <div class="toolbar-card">
        <div class="search-box">
          <v-icon icon="mdi-magnify" size="20" color="grey" />
          <input
            v-model="userSearch"
            type="text"
            :placeholder="l({ lo: 'ຄົ້ນຫາຊື່ຜູ້ໃຊ້, ອີເມວ, ຊື່ຮ້ານຄ້າ...', th: 'ค้นหาชื่อผู้ใช้, อีเมล, ชื่อร้าน...', en: 'Search user name, email, store...' })"
          />
        </div>
        <div class="info-tag">
          {{ l({ lo: 'ລວມທັງໝົດ', th: 'รวมทั้งหมด', en: 'Total' }) }}: <strong>{{ allUsers.length }}</strong> Users
        </div>
      </div>

      <div class="table-card">
        <div v-if="usersLoading" class="empty-state">
          <v-progress-circular indeterminate color="primary" size="32" />
          <p>{{ l({ lo: 'ກຳລັງໂຫຼດລາຍຊື່ Users ຕາມຮ້ານຄ້າ...', th: 'กำลังโหลดรายชื่อ Users ตามร้านค้า...', en: 'Loading users by store...' }) }}</p>
        </div>

        <div v-else-if="!filteredUsers.length" class="empty-state">
          <v-icon icon="mdi-account-group-outline" size="36" color="#94a3b8" />
          <p>{{ l({ lo: 'ບໍ່ພົບຜູ້ໃຊ້ທີ່ຄົ້ນຫາ', th: 'ไม่พบผู้ใช้ที่ค้นหา', en: 'No users found' }) }}</p>
        </div>

        <div v-else class="table-responsive">
          <table class="admin-table">
            <thead>
              <tr>
                <th>{{ l({ lo: 'ຜູ້ໃຊ້ (User)', th: 'ผู้ใช้ (User)', en: 'User' }) }}</th>
                <th>{{ l({ lo: 'ອີເມວ (Email)', th: 'อีเมล (Email)', en: 'Email' }) }}</th>
                <th>{{ l({ lo: 'ສັງກັດຮ້ານຄ້າໃດ? (Belongs to Shop)', th: 'สังกัดร้านค้าใด? (Belongs to Shop)', en: 'Belongs to Shop' }) }}</th>
                <th>{{ l({ lo: 'ບົດບາດໃນຮ້ານ (Role)', th: 'บทบาทในร้าน (Role)', en: 'Role' }) }}</th>
                <th>{{ l({ lo: 'ສິດລະບົບ', th: 'สิทธิ์ระบบ', en: 'System Role' }) }}</th>
                <th>{{ l({ lo: 'ວັນທີເຂົ້າຮ່ວມ', th: 'วันที่เข้าร่วม', en: 'Joined Date' }) }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in filteredUsers" :key="u.userId + '-' + u.accountId">
                <td>
                  <div class="user-profile-cell">
                    <div class="user-avatar-circle">
                      {{ (u.userName || 'U').charAt(0).toUpperCase() }}
                    </div>
                    <strong>{{ u.userName }}</strong>
                  </div>
                </td>
                <td>
                  <code>{{ u.email }}</code>
                </td>
                <td>
                  <div class="shop-link-cell">
                    <v-icon icon="mdi-storefront" size="16" color="#2563eb" />
                    <strong>{{ u.storeName }}</strong>
                  </div>
                </td>
                <td>
                  <span class="role-badge" :class="u.role">
                    <v-icon v-if="u.role === 'owner'" icon="mdi-crown" size="13" color="#f59e0b" />
                    {{ u.role }}
                  </span>
                </td>
                <td>
                  <span v-if="u.isSuperAdmin" class="superadmin-chip">
                    <v-icon icon="mdi-shield-crown" size="13" color="#dc2626" />
                    Super Admin
                  </span>
                  <span v-else class="normal-user-chip">User</span>
                </td>
                <td>
                  <small>{{ formatDate(u.createdAt) }}</small>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- STORE DETAIL MODAL (Users & Pages inside this store) -->
    <v-dialog v-model="detailDialog" max-width="760">
      <div v-if="selectedTenant" class="modal-card">
        <div class="modal-header">
          <div class="modal-title-left">
            <div class="modal-store-icon">
              <v-icon icon="mdi-storefront" size="24" color="#2563eb" />
            </div>
            <div>
              <h3>{{ selectedTenant.name }}</h3>
              <small>{{ selectedTenant.id }}</small>
            </div>
          </div>
          <button type="button" class="btn-modal-close" @click="detailDialog = false">
            <v-icon icon="mdi-close" size="20" />
          </button>
        </div>

        <div class="modal-body">
          <!-- Overview row -->
          <div class="store-meta-grid">
            <div>
              <span class="meta-lbl">{{ l({ lo: 'ເຈົ້າຂອງຮ້ານ (Owner)', th: 'เจ้าของร้าน (Owner)', en: 'Store Owner' }) }}</span>
              <strong>{{ selectedTenant.ownerName || '-' }}</strong>
              <small>{{ selectedTenant.ownerEmail || '-' }}</small>
            </div>
            <div>
              <span class="meta-lbl">{{ l({ lo: 'ແພັກເກດ (Plan)', th: 'แพ็กเกจ (Plan)', en: 'Plan' }) }}</span>
              <div class="plan-change-wrap">
                <span class="plan-chip" :class="selectedTenant.planId">{{ selectedTenant.planName }}</span>
                <span v-if="selectedTenant.lockedPrice" class="locked-price-pill" title="Locked renewal price for this store">
                  🔒 {{ new Intl.NumberFormat().format(selectedTenant.lockedPrice) }} ₭
                </span>
                <button type="button" class="btn-change-plan" @click="openTenantPlanChange">
                  <v-icon icon="mdi-pencil" size="13" />
                  <span>{{ l({ lo: 'ປ່ຽນ', th: 'เปลี่ยน', en: 'Change' }) }}</span>
                </button>
              </div>
            </div>
            <div>
              <span class="meta-lbl">{{ l({ lo: 'ສະຖານະ (Status)', th: 'สถานะ (Status)', en: 'Status' }) }}</span>
              <span class="status-indicator" :class="selectedTenant.status">
                <span class="dot" />
                {{ selectedTenant.status === 'active' ? l({ lo: 'ເປີດໃຊ້', th: 'เปิดใช้', en: 'Active' }) : l({ lo: 'ຖືກລະງັບ', th: 'ถูกระงับ', en: 'Suspended' }) }}
              </span>
            </div>
          </div>

          <!-- Section 1: Users in this store -->
          <div class="modal-section">
            <div class="section-title-row">
              <div class="sec-title">
                <v-icon icon="mdi-account-group" size="20" color="#2563eb" />
                <h4>{{ l({ lo: 'ສະມາຊິກ / Users ໃນຮ້ານນີ້', th: 'สมาชิก / Users ในร้านนี้', en: 'Users in this Store' }) }}</h4>
              </div>
              <span class="count-badge">{{ selectedTenant.users?.length || selectedTenant.userCount || 0 }} {{ l({ lo: 'ຄົນ', th: 'คน', en: 'users' }) }}</span>
            </div>

            <div v-if="selectedTenant.users && selectedTenant.users.length" class="sub-table-wrap">
              <table class="sub-table">
                <thead>
                  <tr>
                    <th>{{ l({ lo: 'ຊື່ຜູ້ໃຊ້ (User Name)', th: 'ชื่อผู้ใช้', en: 'Name' }) }}</th>
                    <th>{{ l({ lo: 'ອີເມວ (Email)', th: 'อีเมล', en: 'Email' }) }}</th>
                    <th>{{ l({ lo: 'ບົດບາດ (Role)', th: 'บทบาท', en: 'Role' }) }}</th>
                    <th>{{ l({ lo: 'ວັນທີເຂົ້າຮ່ວມ', th: 'วันที่เข้าร่วม', en: 'Joined' }) }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="u in selectedTenant.users" :key="u.id">
                    <td>
                      <strong>{{ u.name }}</strong>
                    </td>
                    <td><code>{{ u.email }}</code></td>
                    <td>
                      <span class="role-badge" :class="u.role">
                        <v-icon v-if="u.role === 'owner'" icon="mdi-crown" size="13" color="#f59e0b" />
                        {{ u.role }}
                      </span>
                    </td>
                    <td><small>{{ formatDate(u.createdAt) }}</small></td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-else class="no-items-text">
              {{ l({ lo: 'ຍັງບໍ່ມີ Users ໃນຮ້ານນີ້', th: 'ยังไม่มี Users ในร้านนี้', en: 'No users found in this store' }) }}
            </p>
          </div>

          <!-- Section 2: Connected Facebook Pages -->
          <div class="modal-section">
            <div class="section-title-row">
              <div class="sec-title">
                <v-icon icon="mdi-facebook" size="20" color="#1877f2" />
                <h4>{{ l({ lo: 'Facebook Pages ທີ່ເຊື່ອມຕໍ່ໃນຮ້ານນີ້', th: 'Facebook Pages ที่เชื่อมต่อในร้านนี้', en: 'Facebook Pages Connected' }) }}</h4>
              </div>
              <span class="count-badge">{{ selectedTenant.pages?.length || selectedTenant.connectedPagesCount || 0 }} Pages</span>
            </div>

            <div v-if="selectedTenant.pages && selectedTenant.pages.length" class="sub-table-wrap">
              <table class="sub-table">
                <thead>
                  <tr>
                    <th>{{ l({ lo: 'ຊື່ Page', th: 'ชื่อ Page', en: 'Page Name' }) }}</th>
                    <th>{{ l({ lo: 'Page ID', th: 'Page ID', en: 'Page ID' }) }}</th>
                    <th>{{ l({ lo: 'ໝວດໝູ່ (Category)', th: 'หมวดหมู่', en: 'Category' }) }}</th>
                    <th>{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="p in selectedTenant.pages" :key="p.pageId">
                    <td>
                      <div class="page-title-cell">
                        <img v-if="p.pictureUrl" :src="p.pictureUrl" class="page-avatar" alt="Avatar" />
                        <v-icon v-else icon="mdi-facebook" size="18" color="#1877f2" />
                        <strong>{{ p.name }}</strong>
                      </div>
                    </td>
                    <td><code>{{ p.pageId }}</code></td>
                    <td><small>{{ p.category || '-' }}</small></td>
                    <td>
                      <span class="status-indicator" :class="p.isConnected ? 'active' : 'suspended'">
                        <span class="dot" />
                        {{ p.isConnected ? l({ lo: 'ເຊື່ອມຕໍ່ແລ້ວ', th: 'เชื่อมต่อแล้ว', en: 'Connected' }) : l({ lo: 'ຕັດການເຊື່ອມ', th: 'ตัดการเชื่อม', en: 'Disconnected' }) }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-else class="no-items-text">
              {{ l({ lo: 'ຮ້ານນີ້ຍັງບໍ່ທັນເຊື່ອມຕໍ່ Facebook Page', th: 'ร้านนี้ยังไม่ได้เชื่อมต่อ Facebook Page', en: 'No Facebook pages connected to this store yet' }) }}
            </p>
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" class="btn-secondary" @click="detailDialog = false">
            {{ l({ lo: 'ປິດໜ້າຕ່າງ', th: 'ปิดหน้าต่าง', en: 'Close' }) }}
          </button>
        </div>
      </div>
    </v-dialog>

    <!-- Change Tenant Plan Modal -->
    <v-dialog v-model="changePlanDialog" max-width="480">
      <div v-if="selectedTenant" class="tenant-modal-card">
        <div class="modal-header">
          <div class="modal-title-left">
            <v-icon icon="mdi-tag-edit" size="22" color="#7C3AED" />
            <div>
              <h3>{{ l({ lo: 'ປ່ຽນແພັກເກດໃຫ້ຮ້ານຄ້າ', th: 'เปลี่ยนแพ็กเกจให้ร้านค้า', en: 'Change Store Plan' }) }}</h3>
              <p class="sub-txt">{{ selectedTenant.name }} ({{ selectedTenant.slug }})</p>
            </div>
          </div>
          <button type="button" class="close-btn" @click="changePlanDialog = false">
            <v-icon icon="mdi-close" size="20" />
          </button>
        </div>

        <div class="modal-body">
          <div class="plan-form-grid">
            <div class="form-field">
              <label>{{ l({ lo: 'ເລືອກແພັກເກດ (Plan)', th: 'เลือกแพ็กเกจ (Plan)', en: 'Select Plan' }) }}</label>
              <select v-model="changePlanForm.planId" class="modal-select">
                <option v-for="p in availablePlans" :key="'plan-opt-' + p.id" :value="p.id">
                  {{ p.name }} ({{ p.price > 0 ? (new Intl.NumberFormat().format(p.price) + ' ₭') : 'ຟຣີ' }})
                </option>
              </select>
            </div>

            <div class="form-field">
              <label>{{ l({ lo: 'ໄລຍະເວລາຕໍ່ອາຍຸ', th: 'ระยะเวลาต่ออายุ', en: 'Duration' }) }}</label>
              <select v-model.number="changePlanForm.days" class="modal-select">
                <option :value="7">{{ l({ lo: '7 ວັນ (Trial)', th: '7 วัน (Trial)', en: '7 Days (Trial)' }) }}</option>
                <option :value="30">{{ l({ lo: '30 ວັນ (1 ເດືອນ)', th: '30 วัน (1 เดือน)', en: '30 Days (1 Month)' }) }}</option>
                <option :value="90">{{ l({ lo: '90 ວັນ (3 ເດືອນ)', th: '90 วัน (3 เดือน)', en: '90 Days (3 Months)' }) }}</option>
                <option :value="180">{{ l({ lo: '180 ວັນ (6 ເດືອນ)', th: '180 วัน (6 เดือน)', en: '180 Days (6 Months)' }) }}</option>
                <option :value="365">{{ l({ lo: '365 ວັນ (1 ປີ)', th: '365 วัน (1 ปี)', en: '365 Days (1 Year)' }) }}</option>
                <option :value="0">{{ l({ lo: 'ຕະຫຼອດຊີບ / ບໍ່ມີກຳນົດ (Lifetime / Free)', th: 'ตลอดชีพ / ไม่มีกำหนด (Lifetime / Free)', en: 'Lifetime / No Expiry' }) }}</option>
              </select>
            </div>

            <div class="form-field">
              <label>{{ l({ lo: 'ລາຄາລັອກໄວ້ສຳລັບຮ້ານນີ້ (Locked Price LAK)', th: 'ราคาที่ล็อกไว้สำหรับร้านนี้ (Locked Price LAK)', en: 'Locked Price (LAK)' }) }}</label>
              <input v-model.number="changePlanForm.lockedPrice" type="number" min="0" step="10000" class="modal-select">
              <small class="sub-txt">{{ l({ lo: 'ຮ້ານຄ້າຈະສຳລະລາຄານີ້ໃນຮອບຕໍ່ອາຍຸ ເຖິງແມ່ນວ່າ Admin ຈະປັບຂຶ້ນລາຄາແພັກເກດກໍ່ຕາມ', th: 'ร้านค้าจะจ่ายราคานี้ในการต่ออายุ แม้ว่าจะมีการปรับราคาขึ้นก็ตาม', en: 'Grandfathered price for this tenant, immune to future price increases' }) }}</small>
            </div>

            <div class="form-field">
              <label>{{ l({ lo: 'ສະຖານະ (Status)', th: 'สถานะ (Status)', en: 'Status' }) }}</label>
              <select v-model="changePlanForm.status" class="modal-select">
                <option value="active">{{ l({ lo: 'ໃຊ້ງານຢູ່ (Active)', th: 'ใช้งานอยู่ (Active)', en: 'Active' }) }}</option>
                <option value="trialing">{{ l({ lo: 'ທົດລອງໃຊ້ (Trialing)', th: 'ทดลองใช้ (Trialing)', en: 'Trialing' }) }}</option>
              </select>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" class="btn-secondary" @click="changePlanDialog = false">
            {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
          </button>
          <button type="button" class="btn-primary" :disabled="changePlanLoading" @click="submitTenantPlanChange">
            <v-progress-circular v-if="changePlanLoading" indeterminate size="18" color="white" />
            <span v-else>{{ l({ lo: 'ບັນທຶກການປ່ຽນແປງ', th: 'บันทึกการเปลี่ยนแปลง', en: 'Save Changes' }) }}</span>
          </button>
        </div>
      </div>
    </v-dialog>
  </div>
</template>

<style scoped>
.tenants-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.page-title-row h1 {
  margin: 0;
  font-size: var(--text-2xl);
  font-weight: 800;
  color: #0F172A;
}

.page-title-row p {
  margin: var(--space-0-5) 0 0;
  font-size: var(--text-sm);
  color: #64748B;
}

/* Main Tab Navigation Bar */
.main-tabs-bar {
  display: flex;
  gap: 8px;
  background: #ffffff;
  padding: 6px;
  border-radius: 14px;
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-xs);
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 10px;
  border: none;
  background: transparent;
  color: #64748b;
  font-size: 13.5px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-btn:hover {
  background: #f8fafc;
  color: #0f172a;
}

.tab-btn.active {
  background: #2563eb;
  color: #ffffff;
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.25);
}

.tab-btn.active :deep(.v-icon) {
  color: #ffffff !important;
}

.count-pill {
  font-size: 11px;
  padding: 2px 7px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.08);
  font-weight: 800;
}

.tab-btn.active .count-pill {
  background: rgba(255, 255, 255, 0.25);
  color: #ffffff;
}

.tab-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.toolbar-card {
  background-color: #FFFFFF;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-2) var(--space-3);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.search-box {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  background-color: #F8FAFC;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: 0 0.75rem;
  width: 22rem;
  height: 2.5rem;
}

.search-box input {
  border: 0;
  background: transparent;
  flex: 1;
  font-size: var(--text-sm);
  outline: none;
}

.clear-search {
  background: transparent;
  border: 0;
  cursor: pointer;
  color: #94A3B8;
}

.info-tag {
  font-size: 13px;
  color: #64748b;
}

.filter-tabs {
  display: flex;
  background-color: #F1F5F9;
  border-radius: var(--radius-xs);
  padding: 2px;
}

.filter-tabs button {
  padding: 0.4rem 0.85rem;
  border: 0;
  background: transparent;
  border-radius: var(--radius-xs);
  font-size: var(--text-xs);
  font-weight: 700;
  color: #64748B;
  cursor: pointer;
}

.filter-tabs button.active {
  background-color: #FFFFFF;
  color: #0F172A;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.table-card {
  background-color: #FFFFFF;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow-xs);
}

.table-responsive {
  overflow-x: auto;
}

.admin-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.admin-table th {
  text-align: left;
  padding: 0.75rem 1rem;
  background-color: #F8FAFC;
  color: #475569;
  font-weight: 700;
  font-size: var(--text-xs);
  border-bottom: 1px solid var(--color-border);
}

.admin-table td {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--color-border-subtle);
  vertical-align: middle;
}

.shop-cell, .owner-cell {
  display: flex;
  flex-direction: column;
}

.shop-cell strong {
  color: #0F172A;
}

.shop-cell small, .owner-cell small {
  color: #94A3B8;
  font-size: 11px;
}

.user-count-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  padding: 3px 8px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
  color: #334155;
  cursor: pointer;
}

.user-count-chip:hover {
  background: #e2e8f0;
  color: #0f172a;
}

.pages-badge-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background-color: #EFF6FF;
  color: #1E40AF;
  border: 1px solid #dbeafe;
  padding: 3px 8px;
  border-radius: 8px;
  font-size: 11.5px;
  font-weight: 700;
  cursor: pointer;
}

.pages-badge-btn:hover {
  background-color: #dbeafe;
}

.page-title-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.page-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
}

.page-avatar-fallback {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #1877f2;
  display: grid;
  place-items: center;
}

.category-chip {
  font-size: 11px;
  padding: 2px 7px;
  border-radius: 6px;
  background: #f1f5f9;
  color: #475569;
}

.shop-link-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #0f172a;
}

.user-profile-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar-circle {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #2563eb;
  color: #ffffff;
  font-size: 12px;
  font-weight: 800;
  display: grid;
  place-items: center;
}

.role-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 700;
  text-transform: capitalize;
}

.role-badge.owner { background: #fef3c7; color: #b45309; }
.role-badge.admin { background: #dbeafe; color: #1e40af; }
.role-badge.member { background: #f1f5f9; color: #475569; }

.superadmin-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 700;
  background: #fee2e2;
  color: #b91c1c;
}

.normal-user-chip {
  font-size: 11px;
  color: #64748b;
}

.plan-chip {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}

.plan-chip.pro {
  background-color: #EEF2FF;
  color: #4338CA;
}

.plan-chip.free {
  background-color: #F1F5F9;
  color: #475569;
}

.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: var(--text-xs);
  font-weight: 700;
}

.status-indicator .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-indicator.active { color: #059669; }
.status-indicator.active .dot { background-color: #10B981; }

.status-indicator.suspended { color: #DC2626; }
.status-indicator.suspended .dot { background-color: #EF4444; }

.action-buttons {
  display: inline-flex;
  gap: var(--space-1);
}

.btn-detail {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0.35rem 0.65rem;
  font-size: var(--text-xs);
  font-weight: 700;
  border-radius: var(--radius-xs);
  border: 1px solid #e2e8f0;
  background: #f8fafc;
  color: #0f172a;
  cursor: pointer;
}

.btn-detail:hover {
  background: #e2e8f0;
}

.lock-btn, .unlock-btn, .delete-btn {
  padding: 0.35rem 0.65rem;
  font-size: var(--text-xs);
  border-radius: var(--radius-xs);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-6) 0;
  gap: var(--space-2);
  color: #64748B;
}

.empty-state p {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: 600;
}

.text-right {
  text-align: right;
}

/* Modal styles */
.modal-card {
  background: #ffffff;
  border-radius: 20px;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid #e2e8f0;
  background: #f8fafc;
}

.modal-title-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.modal-store-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: #eff6ff;
  display: grid;
  place-items: center;
}

.modal-title-left h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 800;
  color: #0f172a;
}

.modal-title-left small {
  font-size: 11px;
  color: #94a3b8;
}

.btn-modal-close {
  background: transparent;
  border: none;
  color: #64748b;
  cursor: pointer;
  padding: 4px;
}

.modal-body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

.store-meta-grid {
  display: grid;
  grid-template-columns: 1.5fr 1fr 1fr;
  gap: 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 14px 18px;
}

.meta-lbl {
  display: block;
  font-size: 11px;
  font-weight: 700;
  color: #64748b;
  text-transform: uppercase;
  margin-bottom: 2px;
}

.modal-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.section-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sec-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sec-title h4 {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}

.count-badge {
  font-size: 11.5px;
  font-weight: 700;
  background: #f1f5f9;
  color: #475569;
  padding: 2px 8px;
  border-radius: 999px;
}

.sub-table-wrap {
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  overflow: hidden;
}

.sub-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}

.sub-table th {
  background: #f8fafc;
  padding: 8px 12px;
  text-align: left;
  font-size: 11px;
  font-weight: 700;
  color: #64748b;
  border-bottom: 1px solid #e2e8f0;
}

.sub-table td {
  padding: 10px 12px;
  border-bottom: 1px solid #f1f5f9;
}

.no-items-text {
  font-size: 13px;
  color: #94a3b8;
  font-style: italic;
  margin: 4px 0 0;
}

.modal-footer {
  padding: 16px 24px;
  border-top: 1px solid #e2e8f0;
  display: flex;
  justify-content: flex-end;
  background: #f8fafc;
}

.plan-change-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.locked-price-pill {
  font-size: 11px;
  font-weight: 700;
  background-color: #FEF3C7;
  color: #B45309;
  border: 1px solid #FDE68A;
  padding: 2px 7px;
  border-radius: 4px;
}

.btn-change-plan {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background-color: #EDE9FE;
  color: #6D28D9;
  border: 1px solid #DDD6FE;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}

.btn-change-plan:hover {
  background-color: #DDD6FE;
}

.tenant-modal-card {
  background-color: #FFFFFF;
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
}

.modal-title-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.modal-title-left h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
}

.sub-txt {
  margin: 0;
  font-size: 12px;
  color: #64748B;
}

.plan-form-grid {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-field label {
  font-size: 12px;
  font-weight: 600;
  color: #334155;
}

.modal-select {
  padding: 8px 12px;
  border: 1px solid #CBD5E1;
  border-radius: 6px;
  font-size: 13px;
  background-color: #FFFFFF;
}

.modal-select:focus {
  border-color: #7C3AED;
  outline: none;
}
</style>
