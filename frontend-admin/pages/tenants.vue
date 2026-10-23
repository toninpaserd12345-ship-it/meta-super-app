<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

const { l } = useAdminLocale()
const { adminFetch } = useAdminApi()

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
  currentPeriodEnd?: string
  createdAt: string
}

const loading = ref(true)
const tenants = ref<Tenant[]>([])
const search = ref('')
const filterStatus = ref<'all' | 'active' | 'suspended'>('all')
const actionLoading = ref<Record<string, boolean>>({})

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

const filteredTenants = computed(() => {
  if (filterStatus.value === 'all') return tenants.value
  return tenants.value.filter(t => t.status === filterStatus.value)
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

const formatDate = (val?: string) => {
  if (!val) return '-'
  return new Date(val).toLocaleDateString()
}

onMounted(loadTenants)
</script>

<template>
  <div class="tenants-page">
    <div class="page-title-row">
      <div>
        <h1>{{ l({ lo: 'ຈັດການຮ້ານຄ້າ (Tenants & Workspaces)', th: 'จัดการร้านค้า (Tenants & Workspaces)', en: 'Tenant & Workspace Management' }) }}</h1>
        <p>{{ l({ lo: 'ກວດສອບລາຍຊື່ຮ້ານຄ້າ, ສະຖານະແພັກເກດ, ຈຳນວນ Pages ທີ່ເຊື່ອມຕໍ່ ແລະ ລະງັບ/ປົດລັອກບັນຊີ', th: 'ตรวจสอบรายชื่อร้านค้า, สถานะแพ็กเกจ, จำนวน Pages และระงับ/ปลดล็อกบัญชี', en: 'Monitor shops, plan statuses, connected pages, and lock/unlock workspaces' }) }}</p>
      </div>
      <button type="button" class="btn-secondary" @click="loadTenants">
        <v-icon icon="mdi-refresh" size="18" /> {{ l({ lo: 'ໂຫຼດໃໝ່', th: 'โหลดใหม่', en: 'Refresh' }) }}
      </button>
    </div>

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
        <v-icon icon="mdi-loading mdi-spin" size="32" color="primary" />
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
              <th>{{ l({ lo: 'FB Pages', th: 'FB Pages', en: 'Pages' }) }}</th>
              <th>{{ l({ lo: 'ແພັກເກດ (Plan)', th: 'แพ็กเกจ (Plan)', en: 'Plan' }) }}</th>
              <th>{{ l({ lo: 'ວັນໝົດອາຍຸ', th: 'วันหมดอายุ', en: 'Expires' }) }}</th>
              <th>{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</th>
              <th class="text-right">{{ l({ lo: 'ຈັດການ', th: 'จัดการ', en: 'Actions' }) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in filteredTenants" :key="t.id">
              <td>
                <div class="shop-cell">
                  <strong>{{ t.name }}</strong>
                  <small>{{ t.slug }}</small>
                </div>
              </td>
              <td>
                <div class="owner-cell">
                  <span>{{ t.ownerName || '-' }}</span>
                  <small>{{ t.ownerEmail || '-' }}</small>
                </div>
              </td>
              <td>
                <span class="pages-badge">
                  <v-icon icon="mdi-facebook" size="14" /> {{ t.connectedPagesCount }}
                </span>
              </td>
              <td>
                <span class="plan-chip" :class="t.planId">
                  {{ t.planName }}
                  <small v-if="t.planStatus === 'trialing'">(7D Trial)</small>
                </span>
              </td>
              <td><small>{{ formatDate(t.currentPeriodEnd) }}</small></td>
              <td>
                <span class="status-indicator" :class="t.status">
                  <span class="dot" />
                  {{ t.status === 'active' ? l({ lo: 'ເປີດໃຊ້', th: 'เปิดใช้', en: 'Active' }) : l({ lo: 'ລະງັບ', th: 'ระงับ', en: 'Suspended' }) }}
                </span>
              </td>
              <td class="text-right">
                <div class="action-buttons">
                  <!-- Lock / Unlock Toggle -->
                  <button
                    v-if="t.status === 'active'"
                    type="button"
                    class="btn-secondary lock-btn"
                    title="Suspend / Lock shop"
                    :disabled="actionLoading[t.id]"
                    @click="toggleStatus(t)"
                  >
                    <v-icon icon="mdi-lock-outline" size="16" />
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
                    <v-icon icon="mdi-lock-open-outline" size="16" />
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
                    <v-icon icon="mdi-delete-outline" size="16" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
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

.pages-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background-color: #EFF6FF;
  color: #1E40AF;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
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
</style>
