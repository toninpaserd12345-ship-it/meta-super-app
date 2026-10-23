<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

const { l } = useAdminLocale()
const { adminFetch } = useAdminApi()

interface OverviewData {
  totalTenants: number
  activeTenants: number
  suspendedTenants: number
  proSubscribers: number
  trialSubscribers: number
  freeSubscribers: number
  pendingApprovals: number
  totalRevenue: number
}

interface TransactionItem {
  id: string
  accountId: string
  accountName?: string
  planId?: string
  amount: number
  currency: string
  status: string
  paymentMethod: string
  reference: string
  createdAt: string
}

const loading = ref(true)
const overview = ref<OverviewData | null>(null)
const pendingTxns = ref<TransactionItem[]>([])
const actionLoading = ref<Record<string, boolean>>({})

const loadDashboard = async () => {
  loading.value = true
  try {
    const [stats, txns] = await Promise.all([
      adminFetch<OverviewData>('/api/v1/admin/overview'),
      adminFetch<{ items: TransactionItem[] }>('/api/v1/admin/transactions?status=pending'),
    ])
    overview.value = stats
    pendingTxns.value = txns.items || []
  } catch (err) {
    console.error('Failed to load admin overview:', err)
  } finally {
    loading.value = false
  }
}

const approveTxn = async (id: string) => {
  actionLoading.value[id] = true
  try {
    await adminFetch(`/api/v1/admin/transactions/${id}/approve`, { method: 'POST' })
    await loadDashboard()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Approval failed')
  } finally {
    actionLoading.value[id] = false
  }
}

const rejectTxn = async (id: string) => {
  const reason = prompt(l({
    lo: 'ກະລຸນາລະບຸເຫດຜົນໃນການປະຕິເສດສະລິບ:',
    th: 'กรุณาระบุเหตุผลในการปฏิเสธสลิป:',
    en: 'Please specify reason for rejection:'
  }))
  if (reason === null) return

  actionLoading.value[id] = true
  try {
    await adminFetch(`/api/v1/admin/transactions/${id}/reject`, {
      method: 'POST',
      body: { reason: reason.trim() },
    })
    await loadDashboard()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Rejection failed')
  } finally {
    actionLoading.value[id] = false
  }
}

const formatCurrency = (amount?: number) => {
  return new Intl.NumberFormat().format(amount || 0) + ' LAK'
}

onMounted(loadDashboard)
</script>

<template>
  <div class="dashboard-page">
    <div class="page-title-row">
      <div>
        <h1>{{ l({ lo: 'ພາບລວມລະບົບ ແລະ ລູກຄ້າ', th: 'ภาพรวมระบบและลูกค้า', en: 'Platform Overview & Metrics' }) }}</h1>
        <p>{{ l({ lo: 'ຕິດຕາມສະຖານະຮ້ານຄ້າ, ສະມາຊິກ Pro, ຊ່ວງທົດລອງ 7 ມື້ ແລະ ສະລິບທີ່ລໍຖ້າການອະນຸມັດ', th: 'ติดตามสถานะร้านค้า, สมาชิก Pro, ทดลอง 7 วัน และสลิปที่รออนุมัติ', en: 'Track shops, active Pro plans, 7-day trials, and pending payment slips' }) }}</p>
      </div>
      <button type="button" class="btn-secondary" @click="loadDashboard">
        <v-icon icon="mdi-refresh" size="18" /> {{ l({ lo: 'ໂຫຼດຂໍ້ມູນໃໝ່', th: 'รีเฟรชข้อมูล', en: 'Refresh Data' }) }}
      </button>
    </div>

    <!-- KPI Metric Cards -->
    <div class="kpi-grid">
      <div class="kpi-card">
        <div class="kpi-icon blue"><v-icon icon="mdi-storefront" size="24" /></div>
        <div class="kpi-content">
          <span>{{ l({ lo: 'ຮ້ານຄ້າທັງໝົດ', th: 'ร้านค้าทั้งหมด', en: 'Total Tenants' }) }}</span>
          <strong>{{ overview?.totalTenants || 0 }}</strong>
          <small>{{ overview?.activeTenants || 0 }} {{ l({ lo: 'ເປີດໃຊ້', th: 'ใช้งาน', en: 'active' }) }} · {{ overview?.suspendedTenants || 0 }} {{ l({ lo: 'ລະງັບ', th: 'ระงับ', en: 'suspended' }) }}</small>
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-icon indigo"><v-icon icon="mdi-shield-star" size="24" /></div>
        <div class="kpi-content">
          <span>{{ l({ lo: 'ສະມາຊິກ Pro Plan', th: 'สมาชิก Pro Plan', en: 'Active Pro Subscribers' }) }}</span>
          <strong>{{ overview?.proSubscribers || 0 }}</strong>
          <small>{{ l({ lo: 'ຊຳລະລາຍເດືອນ 500,000 LAK', th: 'ชำระรายเดือน 500,000 LAK', en: '500,000 LAK / mo' }) }}</small>
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-icon amber"><v-icon icon="mdi-gift-outline" size="24" /></div>
        <div class="kpi-content">
          <span>{{ l({ lo: 'ທົດລອງ 7 ມື້ (Trial)', th: 'ทดลอง 7 วัน (Trial)', en: '7-Day Free Trial' }) }}</span>
          <strong>{{ overview?.trialSubscribers || 0 }}</strong>
          <small>{{ l({ lo: 'ລູກຄ້າໃໝ່ທີ່ກຳລັງທົດລອງ Pro', th: 'ลูกค้าใหม่ที่กำลังทดลอง Pro', en: 'New shops trying Pro' }) }}</small>
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-icon rose"><v-icon icon="mdi-clock-alert-outline" size="24" /></div>
        <div class="kpi-content">
          <span>{{ l({ lo: 'ລໍຖ້າກວດສອບສະລິບ', th: 'รอตรวจสอบสลิป', en: 'Pending Approvals' }) }}</span>
          <strong class="highlight-rose">{{ overview?.pendingApprovals || 0 }}</strong>
          <small>{{ l({ lo: 'ສະລິບ BCEL One ທີ່ລູກຄ້າໂອນ', th: 'สลิป BCEL One ที่ลูกค้าโอน', en: 'Slips submitted by shops' }) }}</small>
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-icon emerald"><v-icon icon="mdi-cash-multiple" size="24" /></div>
        <div class="kpi-content">
          <span>{{ l({ lo: 'ລາຍຮັບລວມທັງໝົດ', th: 'รายรับรวมทั้งหมด', en: 'Total Revenue' }) }}</span>
          <strong class="highlight-emerald">{{ formatCurrency(overview?.totalRevenue) }}</strong>
          <small>{{ l({ lo: 'ຈາກທຸລະກຳທີ່ອະນຸມັດແລ້ວ', th: 'จากรายการที่อนุมัติแล้ว', en: 'From completed transactions' }) }}</small>
        </div>
      </div>
    </div>

    <!-- Pending BCEL One Slips Section -->
    <div class="section-card">
      <div class="section-header">
        <div class="d-flex align-center gap-2">
          <v-icon icon="mdi-credit-card-clock-outline" color="primary" size="22" />
          <h2>{{ l({ lo: 'ຄິວລໍຖ້າກວດສອບສະລິບ BCEL One', th: 'คิวรอตรวจสอบสลิป BCEL One', en: 'Pending BCEL One Slip Queue' }) }}</h2>
        </div>
        <NuxtLink to="/approvals" class="view-all-link">
          {{ l({ lo: 'ເບິ່ງທັງໝົດ', th: 'ดูทั้งหมด', en: 'View All' }) }} <v-icon icon="mdi-arrow-right" size="16" />
        </NuxtLink>
      </div>

      <div v-if="loading" class="empty-state">
        <v-icon icon="mdi-loading mdi-spin" size="32" color="primary" />
        <p>{{ l({ lo: 'ກຳລັງໂຫຼດຂໍ້ມູນ...', th: 'กำลังโหลดข้อมูล...', en: 'Loading queue...' }) }}</p>
      </div>

      <div v-else-if="!pendingTxns.length" class="empty-state">
        <v-icon icon="mdi-check-decagram" size="36" color="success" />
        <p>{{ l({ lo: 'ບໍ່ມີສະລິບຄ້າງ! ກວດສອບຄົບທຸກລາຍການແລ້ວ', th: 'ไม่มีสลิปค้าง! ตรวจสอบครบทุกรายการแล้ว', en: 'All clear! No pending slips waiting for review.' }) }}</p>
      </div>

      <div v-else class="table-responsive">
        <table class="admin-table">
          <thead>
            <tr>
              <th>{{ l({ lo: 'ຮ້ານຄ້າ / Workspace', th: 'ร้านค้า / Workspace', en: 'Shop / Workspace' }) }}</th>
              <th>{{ l({ lo: 'ແພັກເກດ', th: 'แพ็กเกจ', en: 'Plan' }) }}</th>
              <th>{{ l({ lo: 'ຍອດເງິນ', th: 'ยอดเงิน', en: 'Amount' }) }}</th>
              <th>{{ l({ lo: 'ວິທີຊຳລະ', th: 'วิธีชำระ', en: 'Method' }) }}</th>
              <th>{{ l({ lo: 'ຫຼັກຖານ / ສະລິບ', th: 'หลักฐาน / สลิป', en: 'Slip Reference' }) }}</th>
              <th>{{ l({ lo: 'ເວລາສົ່ງ', th: 'เวลาส่ง', en: 'Submitted' }) }}</th>
              <th class="text-right">{{ l({ lo: 'ຈັດການ', th: 'จัดการ', en: 'Actions' }) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="txn in pendingTxns" :key="txn.id">
              <td>
                <div class="shop-name-cell">
                  <strong>{{ txn.accountName || txn.accountId }}</strong>
                  <small>{{ txn.accountId }}</small>
                </div>
              </td>
              <td><span class="plan-badge pro">Pro (30 Days)</span></td>
              <td><strong>{{ formatCurrency(txn.amount) }}</strong></td>
              <td><span class="method-badge">BCEL One</span></td>
              <td>
                <a
                  v-if="txn.reference.startsWith('http')"
                  :href="txn.reference"
                  target="_blank"
                  class="slip-link"
                >
                  <v-icon icon="mdi-image-outline" size="16" /> {{ l({ lo: 'ເບິ່ງຮູບສະລິບ', th: 'ดูรูปสลิป', en: 'View Slip' }) }}
                </a>
                <span v-else class="ref-text">{{ txn.reference || '-' }}</span>
              </td>
              <td><small>{{ new Date(txn.createdAt).toLocaleString() }}</small></td>
              <td class="text-right">
                <div class="action-buttons">
                  <button
                    type="button"
                    class="btn-success action-btn"
                    :disabled="actionLoading[txn.id]"
                    @click="approveTxn(txn.id)"
                  >
                    <v-icon icon="mdi-check-bold" size="16" />
                    {{ l({ lo: 'ອະນຸມັດ', th: 'อนุมัติ', en: 'Approve' }) }}
                  </button>
                  <button
                    type="button"
                    class="btn-danger action-btn"
                    :disabled="actionLoading[txn.id]"
                    @click="rejectTxn(txn.id)"
                  >
                    <v-icon icon="mdi-close-thick" size="16" />
                    {{ l({ lo: 'ປະຕິເສດ', th: 'ปฏิเสธ', en: 'Reject' }) }}
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
.dashboard-page {
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

.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
  gap: var(--space-3);
}

.kpi-card {
  background-color: #FFFFFF;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-3);
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  box-shadow: var(--shadow-xs);
}

.kpi-icon {
  width: 3rem;
  height: 3rem;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.kpi-icon.blue { background-color: #EFF6FF; color: #1E40AF; }
.kpi-icon.indigo { background-color: #EEF2FF; color: #4338CA; }
.kpi-icon.amber { background-color: #FEF3C7; color: #D97706; }
.kpi-icon.rose { background-color: #FFE4E6; color: #E11D48; }
.kpi-icon.emerald { background-color: #ECFDF5; color: #059669; }

.kpi-content {
  display: flex;
  flex-direction: column;
}

.kpi-content span {
  font-size: var(--text-xs);
  font-weight: 600;
  color: #64748B;
}

.kpi-content strong {
  font-size: var(--text-xl);
  font-weight: 800;
  color: #0F172A;
  margin: 0.125rem 0;
}

.kpi-content strong.highlight-rose { color: #E11D48; }
.kpi-content strong.highlight-emerald { color: #059669; }

.kpi-content small {
  font-size: 11px;
  color: #94A3B8;
}

.section-card {
  background-color: #FFFFFF;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  box-shadow: var(--shadow-xs);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}

.section-header h2 {
  margin: 0;
  font-size: var(--text-md);
  font-weight: 700;
}

.view-all-link {
  font-size: var(--text-xs);
  font-weight: 700;
  color: #1E40AF;
  display: flex;
  align-items: center;
  gap: 2px;
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

.shop-name-cell {
  display: flex;
  flex-direction: column;
}

.shop-name-cell strong {
  color: #0F172A;
}

.shop-name-cell small {
  color: #94A3B8;
  font-family: monospace;
}

.plan-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}

.plan-badge.pro {
  background-color: #EEF2FF;
  color: #4338CA;
}

.method-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-xs);
  background-color: #EFF6FF;
  color: #1E40AF;
  font-size: 11px;
  font-weight: 700;
}

.slip-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #2563EB;
  font-weight: 600;
}

.slip-link:hover {
  text-decoration: underline;
}

.ref-text {
  font-family: monospace;
  font-size: 11px;
  color: #64748B;
}

.action-buttons {
  display: inline-flex;
  gap: var(--space-1);
}

.action-btn {
  padding: 0.35rem 0.65rem;
  font-size: var(--text-xs);
  border-radius: var(--radius-xs);
}

.text-right {
  text-align: right;
}
</style>
