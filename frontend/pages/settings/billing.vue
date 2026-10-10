<script setup lang="ts">
const fetchApi = (url: string, options?: any) => $fetch(url, options)
const { locale, l } = useLocale()

const plans = ref<any[]>([])
const subscription = ref<any>(null)
const loading = ref(true)
const loadError = ref('')

const checkoutModal = ref(false)
const selectedPlan = ref<any>(null)

const loadBilling = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const [plansData, subData] = await Promise.all([
      fetchApi('/api/proxy/api/v1/billing/plans'),
      fetchApi('/api/proxy/api/v1/billing/subscription')
    ])
    plans.value = plansData as any[]
    subscription.value = subData
  } catch (err: any) {
    loadError.value = err?.data?.message || err?.statusMessage || err?.message || 'Billing details could not be loaded.'
  } finally {
    loading.value = false
  }
}

const openCheckout = (plan: any) => {
  if (subscription.value?.planId === plan.id) {
    alert('You are already on this plan')
    return
  }
  selectedPlan.value = plan
  checkoutModal.value = true
}

const formatDate = (dateString: string) => {
  if (!dateString) return 'N/A'
  return new Date(dateString).toLocaleDateString(locale.value === 'lo' ? 'lo-LA' : locale.value === 'th' ? 'th-TH' : 'en-US', {
    year: 'numeric', month: 'long', day: 'numeric'
  })
}

onMounted(() => {
  loadBilling()
})
</script>

<template>
  <div class="settings-section">
    <div class="header-row">
      <div>
        <h2>{{ l({lo:'ການຊຳລະ ແລະແພັກເກດ',th:'การเรียกเก็บเงินและแพ็กเกจ',en:'Billing & Plans'}) }}</h2>
        <p class="subtitle">{{ l({lo:'ຈັດການ subscription ແລະວິທີຊຳລະເງິນ',th:'จัดการการสมัครสมาชิกและวิธีชำระเงิน',en:'Manage your subscription and payment methods.'}) }}</p>
      </div>
    </div>
    
    <div v-if="loading" class="loading">{{ l({lo:'ກຳລັງໂຫຼດຂໍ້ມູນ...',th:'กำลังโหลดข้อมูลการเรียกเก็บเงิน...',en:'Loading billing details...'}) }}</div>
    <v-alert v-else-if="loadError" type="error" variant="tonal" class="mb-4">
      {{ loadError }} <button class="retry-button" @click="loadBilling">Retry</button>
    </v-alert>
    
    <template v-else>
      <div class="current-subscription" v-if="subscription">
        <div class="sub-status">
          <h3>{{ l({lo:'ແພັກເກດປັດຈຸບັນ:',th:'แพ็กเกจปัจจุบัน:',en:'Current Plan:'}) }} <strong>{{ subscription.plan?.name }}</strong></h3>
          <div class="badge" :class="subscription.status">{{ subscription.status }}</div>
        </div>
        <p class="period">
          {{ l({lo:'ຕໍ່ອາຍຸ / ໝົດອາຍຸ:',th:'ต่ออายุ / หมดอายุ:',en:'Renews / Expires on:'}) }} <strong>{{ formatDate(subscription.currentPeriodEnd) }}</strong>
        </p>
      </div>

      <div class="plans-grid">
        <div v-for="plan in plans" :key="plan.id" class="plan-card" :class="{ active: subscription?.planId === plan.id }">
          <div class="plan-header">
            <h4>{{ plan.name }}</h4>
            <div class="price">
              <span class="amount">{{ plan.price > 0 ? (plan.price.toLocaleString() + ' LAK') : 'Free' }}</span>
              <span class="duration" v-if="plan.price > 0">/ {{ l({lo:'ເດືອນ',th:'เดือน',en:'month'}) }}</span>
            </div>
          </div>
          <ul class="features">
            <li><v-icon icon="mdi-check" color="green" /> Up to {{ plan.maxUsers }} Users</li>
            <li><v-icon icon="mdi-check" color="green" /> Up to {{ plan.maxPages }} Facebook Pages</li>
          </ul>
          <button 
            class="btn-primary full-width" 
            :disabled="subscription?.planId === plan.id"
            @click="openCheckout(plan)"
          >
            {{ subscription?.planId === plan.id ? l({lo:'ແພັກເກດປັດຈຸບັນ',th:'แพ็กเกจปัจจุบัน',en:'Current Plan'}) : l({lo:'ອັບເກຣດ',th:'อัปเกรด',en:'Upgrade'}) }}
          </button>
        </div>
      </div>
    </template>

    <!-- Checkout Modal -->
    <div v-if="checkoutModal" class="modal-overlay" @click.self="checkoutModal = false">
      <div class="modal-content">
        <h3>Upgrade to {{ selectedPlan?.name }}</h3>
        
        <div class="payment-instructions">
          <p>Online payment for this plan is not available yet. No payment has been created and no money will be charged.</p>
          <p>Contact the workspace administrator to activate <strong>{{ selectedPlan?.name }}</strong>.</p>
        </div>
        <div class="modal-actions"><button type="button" class="btn-primary" @click="checkoutModal = false">{{ l({lo:'ປິດ',th:'ปิด',en:'Close'}) }}</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-section h2 {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 4px;
}
.subtitle {
  color: var(--color-text-secondary);
  font-size: 14px;
  margin-bottom: 24px;
}

.current-subscription {
  background: var(--color-primary-soft);
  border: 1px solid var(--color-primary);
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 32px;
}
.sub-status {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}
.sub-status h3 {
  font-size: 18px;
  color: var(--color-text);
}
.badge {
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.badge.active { background: #dcfce7; color: #166534; }
.period {
  color: var(--color-text-secondary);
  font-size: 14px;
}

.plans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 24px;
}
.plan-card {
  border: 2px solid var(--color-border);
  border-radius: 16px;
  padding: 24px;
  background: var(--color-background);
  display: flex;
  flex-direction: column;
  transition: transform 0.2s;
}
.plan-card:hover {
  transform: translateY(-4px);
}
.plan-card.active {
  border-color: var(--color-primary);
  box-shadow: 0 4px 20px var(--color-primary-soft);
}
.plan-header {
  margin-bottom: 24px;
  text-align: center;
}
.plan-header h4 {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 8px;
}
.price .amount {
  font-size: 28px;
  font-weight: 800;
  color: var(--color-text);
}
.price .duration {
  color: var(--color-text-secondary);
}
.features {
  list-style: none;
  padding: 0;
  margin: 0 0 32px 0;
  flex: 1;
}
.features li {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal-content {
  background: var(--color-surface);
  width: 100%;
  max-width: 480px;
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 10px 30px rgba(0,0,0,0.2);
}
.payment-instructions {
  background: var(--color-background);
  padding: 16px;
  border-radius: 8px;
  margin: 20px 0;
}
.bank-details {
  margin-top: 12px;
  padding: 12px;
  background: var(--color-surface);
  border: 1px dashed var(--color-border);
  border-radius: 6px;
}
.bank-details p {
  margin-bottom: 4px;
}
.checkout-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-group input {
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-background);
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}
.retry-button{margin-left:8px;border:0;color:var(--color-primary);background:none;font-weight:700;cursor:pointer}
</style>
