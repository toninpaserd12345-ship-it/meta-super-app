<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

const { l } = useAdminLocale()
const { adminFetch } = useAdminApi()

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
  rejectionReason?: string
  createdAt: string
}

const loading = ref(true)
const transactions = ref<TransactionItem[]>([])
const currentFilter = ref<'pending' | 'completed' | 'failed' | ''>('pending')
const actionLoading = ref<Record<string, boolean>>({})

// Image modal
const previewImage = ref<string | null>(null)

const loadTransactions = async () => {
  loading.value = true
  try {
    const statusQuery = currentFilter.value ? `?status=${currentFilter.value}` : ''
    const res = await adminFetch<{ items: TransactionItem[] }>(`/api/v1/admin/transactions${statusQuery}`)
    transactions.value = res.items || []
  } catch (err) {
    console.error('Failed to load transactions:', err)
  } finally {
    loading.value = false
  }
}

const approveTxn = async (id: string) => {
  const confirmMsg = l({
    lo: 'ຢືນຢັນການອະນຸມັດສະລິບນີ້? ລະບົບຈະເປີດ/ຕໍ່ອາຍຸ Pro Plan ໃຫ້ຮ້ານຄ້ານີ້ອີກ +30 ມື້ ທັນທີ.',
    th: 'ยืนยันการอนุมัติสลิปนี้? ระบบจะเปิด/ต่ออายุ Pro Plan ให้ร้านค้านี้อีก +30 วันทันที',
    en: 'Confirm approval? This will activate/extend Pro Plan for +30 days.'
  })

  if (!confirm(confirmMsg)) return

  actionLoading.value[id] = true
  try {
    await adminFetch(`/api/v1/admin/transactions/${id}/approve`, { method: 'POST' })
    await loadTransactions()
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
    await loadTransactions()
  } catch (err: any) {
    alert(err?.data?.message || err?.message || 'Rejection failed')
  } finally {
    actionLoading.value[id] = false
  }
}

const formatCurrency = (amount?: number) => {
  return new Intl.NumberFormat().format(amount || 0) + ' LAK'
}

watch(currentFilter, () => {
  loadTransactions()
})

onMounted(loadTransactions)
</script>

<template>
  <div class="approvals-page">
    <div class="page-title-row">
      <div>
        <h1>{{ l({ lo: 'ກວດສອບສະລິບໂອນເງິນ (Payment Approvals)', th: 'ตรวจสอบสลิปโอนเงิน (Payment Approvals)', en: 'BCEL One Slip Approvals' }) }}</h1>
        <p>{{ l({ lo: 'ກວດສອບຮູບສະລິບ BCEL One / OnePay ທີ່ຮ້ານຄ້າໂອນມາ ແລະ ກົດອະນຸມັດເພື່ອເປີດໃຊ້ Pro Plan (+30 ມື້)', th: 'ตรวจสอบรูปสลิป BCEL One / OnePay ที่ร้านค้าโอนมา และกดอนุมัติเพื่อเปิดใช้งาน Pro Plan (+30 วัน)', en: 'Review BCEL One slips submitted by shops and approve to grant 30-day Pro access' }) }}</p>
      </div>
      <button type="button" class="btn-secondary" @click="loadTransactions">
        <v-icon icon="mdi-refresh" size="18" /> {{ l({ lo: 'ໂຫຼດໃໝ່', th: 'โหลดใหม่', en: 'Refresh' }) }}
      </button>
    </div>

    <!-- Filter Tabs -->
    <div class="filter-bar">
      <div class="tabs">
        <button
          type="button"
          :class="{ active: currentFilter === 'pending' }"
          @click="currentFilter = 'pending'"
        >
          <v-icon icon="mdi-clock-outline" size="16" />
          {{ l({ lo: 'ລໍຖ້າກວດສອບ (Pending)', th: 'รอตรวจสอบ (Pending)', en: 'Pending' }) }}
        </button>
        <button
          type="button"
          :class="{ active: currentFilter === 'completed' }"
          @click="currentFilter = 'completed'"
        >
          <v-icon icon="mdi-check-circle-outline" size="16" />
          {{ l({ lo: 'ອະນຸມັດແລ້ວ (Approved)', th: 'อนุมัติแล้ว (Approved)', en: 'Approved' }) }}
        </button>
        <button
          type="button"
          :class="{ active: currentFilter === 'failed' }"
          @click="currentFilter = 'failed'"
        >
          <v-icon icon="mdi-close-circle-outline" size="16" />
          {{ l({ lo: 'ປະຕິເສດແລ້ວ (Rejected)', th: 'ปฏิเสธแล้ว (Rejected)', en: 'Rejected' }) }}
        </button>
        <button
          type="button"
          :class="{ active: currentFilter === '' }"
          @click="currentFilter = ''"
        >
          {{ l({ lo: 'ທຸກລາຍການ (All)', th: 'ทุกรายการ (All)', en: 'All' }) }}
        </button>
      </div>
    </div>

    <!-- Table Card -->
    <div class="table-card">
      <div v-if="loading" class="empty-state">
        <v-icon icon="mdi-loading mdi-spin" size="32" color="primary" />
        <p>{{ l({ lo: 'ກຳລັງໂຫຼດລາຍການສະລິບ...', th: 'กำลังโหลดรายการสลิป...', en: 'Loading transactions...' }) }}</p>
      </div>

      <div v-else-if="!transactions.length" class="empty-state">
        <v-icon icon="mdi-receipt-text-outline" size="36" color="grey" />
        <p>{{ l({ lo: 'ບໍ່ມີລາຍການສະລິບໃນໝວດນີ້', th: 'ไม่มีรายการสลิปในหมวดนี้', en: 'No transactions found in this view.' }) }}</p>
      </div>

      <div v-else class="table-responsive">
        <table class="admin-table">
          <thead>
            <tr>
              <th>{{ l({ lo: 'ຮ້ານຄ້າ / Workspace', th: 'ร้านค้า / Workspace', en: 'Workspace' }) }}</th>
              <th>{{ l({ lo: 'ແພັກເກດ', th: 'แพ็กเกจ', en: 'Plan' }) }}</th>
              <th>{{ l({ lo: 'ຍອດເງິນ', th: 'ยอดเงิน', en: 'Amount' }) }}</th>
              <th>{{ l({ lo: 'ຮູບສະລິບ / ຫຼັກຖານ', th: 'รูปสลิป / หลักฐาน', en: 'Slip Image / Ref' }) }}</th>
              <th>{{ l({ lo: 'ວັນທີສົ່ງ', th: 'วันที่ส่ง', en: 'Date' }) }}</th>
              <th>{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</th>
              <th class="text-right">{{ l({ lo: 'ຈັດການ', th: 'จัดการ', en: 'Actions' }) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="txn in transactions" :key="txn.id">
              <td>
                <div class="shop-cell">
                  <strong>{{ txn.accountName || txn.accountId }}</strong>
                  <small>{{ txn.accountId }}</small>
                </div>
              </td>
              <td>
                <span class="plan-chip pro">Pro (30 Days)</span>
              </td>
              <td><strong>{{ formatCurrency(txn.amount) }}</strong></td>
              <td>
                <div v-if="txn.reference.startsWith('http')" class="slip-preview-cell">
                  <img
                    :src="txn.reference"
                    alt="Payment Slip"
                    class="slip-thumb"
                    @click="previewImage = txn.reference"
                  />
                  <button
                    type="button"
                    class="preview-link-btn"
                    @click="previewImage = txn.reference"
                  >
                    <v-icon icon="mdi-magnify-plus-outline" size="14" /> {{ l({ lo: 'ຊູມເບິ່ງ', th: 'ขยายดู', en: 'Zoom' }) }}
                  </button>
                </div>
                <div v-else class="ref-box">
                  <span class="ref-code">{{ txn.reference || '-' }}</span>
                </div>
              </td>
              <td><small>{{ new Date(txn.createdAt).toLocaleString() }}</small></td>
              <td>
                <span class="status-badge" :class="txn.status">
                  {{ txn.status === 'completed' ? l({ lo: 'ອະນຸມັດແລ້ວ', th: 'อนุมัติแล้ว', en: 'Approved' }) : txn.status === 'failed' ? l({ lo: 'ປະຕິເສດ', th: 'ปฏิเสธ', en: 'Rejected' }) : l({ lo: 'ລໍຖ້າກວດສອບ', th: 'รอตรวจสอบ', en: 'Pending' }) }}
                </span>
                <div v-if="txn.rejectionReason" class="rejection-note">
                  <small>💬 {{ txn.rejectionReason }}</small>
                </div>
              </td>
              <td class="text-right">
                <div v-if="txn.status === 'pending'" class="action-buttons">
                  <button
                    type="button"
                    class="btn-success action-btn"
                    :disabled="actionLoading[txn.id]"
                    @click="approveTxn(txn.id)"
                  >
                    <v-icon icon="mdi-check-bold" size="14" />
                    {{ l({ lo: 'ອະນຸມັດ (+30 ມື້)', th: 'อนุมัติ (+30 วัน)', en: 'Approve (+30d)' }) }}
                  </button>
                  <button
                    type="button"
                    class="btn-danger action-btn"
                    :disabled="actionLoading[txn.id]"
                    @click="rejectTxn(txn.id)"
                  >
                    <v-icon icon="mdi-close-thick" size="14" />
                    {{ l({ lo: 'ປະຕິເສດ', th: 'ปฏิเสธ', en: 'Reject' }) }}
                  </button>
                </div>
                <span v-else class="text-muted"><small>{{ l({ lo: 'ດຳເນີນການແລ້ວ', th: 'ดำเนินการแล้ว', en: 'Handled' }) }}</small></span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Image Preview Modal -->
    <div v-if="previewImage" class="modal-backdrop" @click="previewImage = null">
      <div class="modal-box" @click.stop>
        <div class="modal-header">
          <h3>{{ l({ lo: 'ຫຼັກຖານສະລິບການໂອນເງິນ', th: 'หลักฐานสลิปการโอนเงิน', en: 'Payment Transfer Slip' }) }}</h3>
          <button type="button" class="close-btn" @click="previewImage = null">
            <v-icon icon="mdi-close" size="20" />
          </button>
        </div>
        <div class="modal-body">
          <img :src="previewImage" alt="Slip Full View" class="full-slip-img" />
        </div>
        <div class="modal-footer">
          <a :href="previewImage" target="_blank" class="btn-secondary">
            <v-icon icon="mdi-open-in-new" size="16" /> {{ l({ lo: 'ເປີດໃນແທັບໃໝ່', th: 'เปิดในแท็บใหม่', en: 'Open Full Size' }) }}
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.approvals-page {
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

.filter-bar {
  display: flex;
  align-items: center;
}

.tabs {
  display: flex;
  background-color: #F1F5F9;
  border-radius: var(--radius-sm);
  padding: 3px;
  gap: 2px;
}

.tabs button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0.5rem 1rem;
  border: 0;
  background: transparent;
  border-radius: var(--radius-xs);
  font-size: var(--text-xs);
  font-weight: 700;
  color: #64748B;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tabs button.active {
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

.shop-cell strong {
  color: #0F172A;
}

.shop-cell small {
  color: #94A3B8;
  font-family: monospace;
}

.plan-chip.pro {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  background-color: #EEF2FF;
  color: #4338CA;
}

.slip-preview-cell {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.slip-thumb {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: var(--radius-xs);
  border: 1px solid var(--color-border);
  cursor: pointer;
  transition: transform 0.15s ease;
}

.slip-thumb:hover {
  transform: scale(1.08);
}

.preview-link-btn {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: transparent;
  border: 0;
  color: #2563EB;
  font-size: var(--text-xs);
  font-weight: 600;
  cursor: pointer;
}

.ref-code {
  font-family: monospace;
  font-size: 11px;
  color: #475569;
  background-color: #F1F5F9;
  padding: 2px 6px;
  border-radius: 4px;
}

.status-badge {
  display: inline-block;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}

.status-badge.pending { background-color: #FEF3C7; color: #B45309; }
.status-badge.completed { background-color: #ECFDF5; color: #047857; }
.status-badge.failed { background-color: #FEE2E2; color: #B91C1C; }

.rejection-note {
  margin-top: 4px;
  color: #DC2626;
  font-size: 11px;
}

.action-buttons {
  display: inline-flex;
  gap: var(--space-1);
}

.action-btn {
  padding: 0.4rem 0.75rem;
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

.text-muted {
  color: #94A3B8;
}

.text-right {
  text-align: right;
}

/* Modal */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: var(--space-3);
}

.modal-box {
  background-color: #FFFFFF;
  border-radius: var(--radius-lg);
  max-width: 32rem;
  width: 100%;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.35);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.modal-header h3 {
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
  padding: var(--space-3);
  overflow-y: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #F8FAFC;
}

.full-slip-img {
  max-width: 100%;
  max-height: 60vh;
  object-fit: contain;
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-sm);
}

.modal-footer {
  padding: var(--space-2) var(--space-3);
  border-top: 1px solid var(--color-border);
  display: flex;
  justify-content: flex-end;
}
</style>
