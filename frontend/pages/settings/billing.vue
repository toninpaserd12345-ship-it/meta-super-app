<script setup lang="ts">
const fetchApi = (url: string, options?: any) => $fetch(url, options)

const plans = ref<any[]>([])
const subscription = ref<any>(null)
const loading = ref(true)

const checkoutModal = ref(false)
const selectedPlan = ref<any>(null)
const checkoutForm = ref({ reference: '' })
const checkoutLoading = ref(false)

const loadBilling = async () => {
  loading.value = true
  try {
    const [plansData, subData] = await Promise.all([
      fetchApi('/api/v1/billing/plans'),
      fetchApi('/api/v1/billing/subscription')
    ])
    plans.value = plansData as any[]
    subscription.value = subData
  } catch (err: any) {
    console.error(err)
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
  checkoutForm.value.reference = ''
  checkoutModal.value = true
}

const submitCheckout = async () => {
  if (!checkoutForm.value.reference) {
    alert('Please enter a transfer reference number')
    return
  }
  
  checkoutLoading.value = true
  try {
    await fetchApi('/api/v1/billing/checkout', {
      method: 'POST',
      body: {
        planId: selectedPlan.value.id,
        paymentMethod: 'bank_transfer',
        reference: checkoutForm.value.reference
      }
    })
    
    alert('Checkout successful! (Simulated for MVP)')
    checkoutModal.value = false
    await loadBilling()
  } catch (err: any) {
    alert(err.message || 'Checkout failed')
  } finally {
    checkoutLoading.value = false
  }
}

const formatDate = (dateString: string) => {
  if (!dateString) return 'N/A'
  return new Date(dateString).toLocaleDateString('en-US', {
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
        <h2>Billing & Plans</h2>
        <p class="subtitle">Manage your subscription and payment methods.</p>
      </div>
    </div>
    
    <div v-if="loading" class="loading">Loading billing details...</div>
    
    <template v-else>
      <div class="current-subscription" v-if="subscription">
        <div class="sub-status">
          <h3>Current Plan: <strong>{{ subscription.plan?.name }}</strong></h3>
          <div class="badge" :class="subscription.status">{{ subscription.status }}</div>
        </div>
        <p class="period">
          Renews / Expires on: <strong>{{ formatDate(subscription.currentPeriodEnd) }}</strong>
        </p>
      </div>

      <div class="plans-grid">
        <div v-for="plan in plans" :key="plan.id" class="plan-card" :class="{ active: subscription?.planId === plan.id }">
          <div class="plan-header">
            <h4>{{ plan.name }}</h4>
            <div class="price">
              <span class="amount">{{ plan.price > 0 ? (plan.price.toLocaleString() + ' LAK') : 'Free' }}</span>
              <span class="duration" v-if="plan.price > 0">/ month</span>
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
            {{ subscription?.planId === plan.id ? 'Current Plan' : 'Upgrade' }}
          </button>
        </div>
      </div>
    </template>

    <!-- Checkout Modal -->
    <div v-if="checkoutModal" class="modal-overlay" @click.self="checkoutModal = false">
      <div class="modal-content">
        <h3>Upgrade to {{ selectedPlan?.name }}</h3>
        
        <div class="payment-instructions">
          <p>Please transfer <strong>{{ selectedPlan?.price.toLocaleString() }} LAK</strong> to our bank account:</p>
          <div class="bank-details">
            <p>Bank: <strong>BCEL</strong></p>
            <p>Account Name: <strong>Meta Super App</strong></p>
            <p>Account No: <strong>1234567890</strong></p>
          </div>
        </div>

        <form @submit.prevent="submitCheckout" class="checkout-form">
          <div class="form-group">
            <label>Transfer Reference Number (Ref/Txn ID)</label>
            <input type="text" v-model="checkoutForm.reference" required placeholder="e.g. 100012398481" />
          </div>
          
          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="checkoutModal = false">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="checkoutLoading">
              {{ checkoutLoading ? 'Processing...' : 'Submit Payment' }}
            </button>
          </div>
        </form>
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
</style>
