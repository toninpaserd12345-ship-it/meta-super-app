<script setup lang="ts">
const { fetchApi } = useApi()

const members = ref<any[]>([])
const loading = ref(true)

const inviteModal = ref(false)
const inviteForm = ref({ email: '', name: '', role: 'employee' })
const inviteError = ref('')
const inviteLoading = ref(false)
const inviteResult = ref<any>(null)

const loadMembers = async () => {
  loading.value = true
  try {
    members.value = await fetchApi('/api/v1/team')
  } catch (err: any) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

const submitInvite = async () => {
  inviteLoading.value = true
  inviteError.value = ''
  try {
    const res = await fetchApi('/api/v1/team/invite', {
      method: 'POST',
      body: inviteForm.value
    })
    inviteResult.value = res
    await loadMembers()
  } catch (err: any) {
    inviteError.value = err.message || 'Failed to invite user'
  } finally {
    inviteLoading.value = false
  }
}

const closeInviteModal = () => {
  inviteModal.value = false
  inviteResult.value = null
  inviteForm.value = { email: '', name: '', role: 'employee' }
}

const removeMember = async (id: string) => {
  if (!confirm('Are you sure you want to remove this member?')) return
  try {
    await fetchApi(`/api/v1/team/${id}`, { method: 'DELETE' })
    await loadMembers()
  } catch (err: any) {
    alert(err.message || 'Failed to remove member')
  }
}

onMounted(() => {
  loadMembers()
})
</script>

<template>
  <div class="settings-section">
    <div class="header-row">
      <div>
        <h2>Team Members</h2>
        <p class="subtitle">Manage who has access to your workspace.</p>
      </div>
      <button class="btn-primary" @click="inviteModal = true">
        <v-icon icon="mdi-account-plus-outline" /> Invite Member
      </button>
    </div>
    
    <div v-if="loading" class="loading">Loading members...</div>
    
    <div v-else class="members-table">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Email</th>
            <th>Role</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="member in members" :key="member.user.id">
            <td>{{ member.user.name }}</td>
            <td>{{ member.user.email }}</td>
            <td>
              <span class="badge" :class="member.role">{{ member.role }}</span>
            </td>
            <td>
              <button class="btn-icon text-danger" @click="removeMember(member.user.id)" title="Remove">
                <v-icon icon="mdi-delete-outline" />
              </button>
            </td>
          </tr>
          <tr v-if="members.length === 0">
            <td colspan="4" class="empty-state">No members found</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Invite Modal -->
    <div v-if="inviteModal" class="modal-overlay" @click.self="closeInviteModal">
      <div class="modal-content">
        <h3>Invite Team Member</h3>
        
        <div v-if="inviteResult" class="success-box">
          <v-icon icon="mdi-check-circle" color="green" />
          <p>User successfully invited!</p>
          <div v-if="inviteResult.password" class="password-box">
            <small>This user is new. Please share this temporary password with them:</small>
            <strong>{{ inviteResult.password }}</strong>
          </div>
          <button class="btn-primary full-width" @click="closeInviteModal" style="margin-top: 16px;">Done</button>
        </div>

        <form v-else @submit.prevent="submitInvite" class="invite-form">
          <div v-if="inviteError" class="error-msg">{{ inviteError }}</div>
          
          <div class="form-group">
            <label>Name</label>
            <input type="text" v-model="inviteForm.name" required placeholder="John Doe" />
          </div>
          
          <div class="form-group">
            <label>Email Address</label>
            <input type="email" v-model="inviteForm.email" required placeholder="john@example.com" />
          </div>
          
          <div class="form-group">
            <label>Role</label>
            <select v-model="inviteForm.role">
              <option value="admin">Admin</option>
              <option value="manager">Manager</option>
              <option value="employee">Employee</option>
            </select>
          </div>
          
          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeInviteModal">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="inviteLoading">
              {{ inviteLoading ? 'Inviting...' : 'Send Invite' }}
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
}
.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.members-table table {
  width: 100%;
  border-collapse: collapse;
}
.members-table th {
  text-align: left;
  padding: 12px 16px;
  background: var(--color-background);
  color: var(--color-text-secondary);
  font-weight: 600;
  font-size: 13px;
  border-bottom: 1px solid var(--color-border);
}
.members-table td {
  padding: 16px;
  border-bottom: 1px solid var(--color-border);
  font-size: 14px;
}
.badge {
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  text-transform: capitalize;
}
.badge.admin { background: #fee2e2; color: #991b1b; }
.badge.manager { background: #e0e7ff; color: #3730a3; }
.badge.employee { background: #dcfce7; color: #166534; }

.text-danger { color: #ef4444; }

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
.modal-content h3 {
  margin-bottom: 24px;
}
.invite-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-group label {
  font-size: 13px;
  font-weight: 600;
}
.form-group input, .form-group select {
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-background);
  color: var(--color-text);
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}
.success-box {
  text-align: center;
  padding: 24px 0;
}
.password-box {
  margin-top: 16px;
  padding: 16px;
  background: var(--color-background);
  border-radius: 8px;
  border: 1px dashed var(--color-primary);
}
.password-box strong {
  display: block;
  font-size: 20px;
  margin-top: 8px;
  letter-spacing: 2px;
}
</style>
