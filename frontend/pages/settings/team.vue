<script setup lang="ts">
const fetchApi = (url: string, options?: any) => $fetch(url, options)
const { can } = useAuth()
const { l } = useLocale()

const members = ref<any[]>([])
const loading = ref(true)
const loadError = ref('')
const updatingMember = ref('')

const inviteModal = ref(false)
const inviteForm = ref({ email: '', name: '', role: 'employee' })
const inviteError = ref('')
const inviteLoading = ref(false)
const inviteResult = ref<any>(null)

const loadMembers = async () => {
  loading.value = true
  loadError.value = ''
  try {
    members.value = await fetchApi('/api/proxy/api/v1/team') as any[]
  } catch (err: any) {
    loadError.value = err?.data?.message || err?.statusMessage || err?.message || l({lo:'ບໍ່ສາມາດໂຫຼດສະມາຊິກທີມໄດ້',th:'ไม่สามารถโหลดสมาชิกทีมได้',en:'Team members could not be loaded.'})
  } finally {
    loading.value = false
  }
}

const updateRole = async (member: any, role: string) => {
  if (!role || role === member.role) return
  const previous = member.role
  member.role = role
  updatingMember.value = member.user.id
  try {
    await fetchApi(`/api/proxy/api/v1/team/${member.user.id}`, { method: 'PUT', body: { role } })
  } catch (err: any) {
    member.role = previous
    loadError.value = err?.data?.message || err?.statusMessage || err?.message || l({lo:'ບໍ່ສາມາດອັບເດດບົດບາດໄດ້',th:'ไม่สามารถอัปเดตบทบาทสมาชิกได้',en:'The member role could not be updated.'})
  } finally {
    updatingMember.value = ''
  }
}

const submitInvite = async () => {
  inviteLoading.value = true
  inviteError.value = ''
  try {
    const res = await fetchApi('/api/proxy/api/v1/team/invite', {
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
  if (!confirm(l({lo:'ຢືນຢັນການລຶບສະມາຊິກນີ້?',th:'ยืนยันการลบสมาชิกคนนี้?',en:'Are you sure you want to remove this member?'}))) return
  try {
    await fetchApi(`/api/proxy/api/v1/team/${id}`, { method: 'DELETE' })
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
        <h2>{{ l({lo:'ສະມາຊິກທີມ',th:'สมาชิกทีม',en:'Team Members'}) }}</h2>
        <p class="subtitle">{{ l({lo:'ຈັດການຜູ້ທີ່ເຂົ້າໃຊ້ Workspace ໄດ້',th:'จัดการผู้ที่สามารถเข้าถึงพื้นที่ทำงาน',en:'Manage who has access to your workspace.'}) }}</p>
      </div>
      <button class="btn-primary" :disabled="!can('users:invite')" @click="inviteModal = true">
        <v-icon icon="mdi-account-plus-outline" /> {{ l({lo:'ເຊີນສະມາຊິກ',th:'เชิญสมาชิก',en:'Invite Member'}) }}
      </button>
    </div>
    
    <div v-if="loading" class="loading">{{ l({lo:'ກຳລັງໂຫຼດສະມາຊິກ...',th:'กำลังโหลดสมาชิก...',en:'Loading members...'}) }}</div>
    <v-alert v-else-if="loadError" type="error" variant="tonal" closable class="mb-4" @click:close="loadError=''">
      {{ loadError }} <button class="inline-action" @click="loadMembers">{{ l({lo:'ລອງໃໝ່',th:'ลองอีกครั้ง',en:'Retry'}) }}</button>
    </v-alert>
    
    <div v-else class="members-table">
      <table>
        <thead>
          <tr>
            <th>{{ l({lo:'ຊື່',th:'ชื่อ',en:'Name'}) }}</th>
            <th>{{ l({lo:'ອີເມວ',th:'อีเมล',en:'Email'}) }}</th>
            <th>{{ l({lo:'ບົດບາດ',th:'บทบาท',en:'Role'}) }}</th>
            <th>{{ l({lo:'ການຈັດການ',th:'การจัดการ',en:'Actions'}) }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="member in members" :key="member.user.id">
            <td>{{ member.user.name }}</td>
            <td>{{ member.user.email }}</td>
            <td>
              <select class="role-select" :value="member.role" :disabled="!can('users:update') || updatingMember===member.user.id" @change="updateRole(member, ($event.target as HTMLSelectElement).value)">
                <option value="admin">Admin</option>
                <option value="manager">Manager</option>
                <option value="employee">Employee</option>
              </select>
            </td>
            <td>
              <button class="btn-icon text-danger" :disabled="!can('users:remove')" @click="removeMember(member.user.id)" title="Remove">
                <v-icon icon="mdi-delete-outline" />
              </button>
            </td>
          </tr>
          <tr v-if="members.length === 0">
            <td colspan="4" class="empty-state">{{ l({lo:'ບໍ່ພົບສະມາຊິກ',th:'ไม่พบสมาชิก',en:'No members found'}) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Invite Modal -->
    <div v-if="inviteModal" class="modal-overlay" @click.self="closeInviteModal">
      <div class="modal-content">
        <h3>{{ l({lo:'ເຊີນສະມາຊິກທີມ',th:'เชิญสมาชิกทีม',en:'Invite Team Member'}) }}</h3>
        
        <div v-if="inviteResult" class="success-box">
          <v-icon icon="mdi-check-circle" color="green" />
          <p>{{ l({lo:'ເຊີນຜູ້ໃຊ້ສຳເລັດ!',th:'เชิญผู้ใช้สำเร็จ!',en:'User successfully invited!'}) }}</p>
          <div v-if="inviteResult.password" class="password-box">
            <small>This user is new. Please share this temporary password with them:</small>
            <strong>{{ inviteResult.password }}</strong>
          </div>
          <button class="btn-primary full-width" @click="closeInviteModal" style="margin-top: 16px;">Done</button>
        </div>

        <form v-else @submit.prevent="submitInvite" class="invite-form">
          <div v-if="inviteError" class="error-msg">{{ inviteError }}</div>
          
          <div class="form-group">
            <label>{{ l({lo:'ຊື່',th:'ชื่อ',en:'Name'}) }}</label>
            <input type="text" v-model="inviteForm.name" required placeholder="Full name" />
          </div>
          
          <div class="form-group">
            <label>{{ l({lo:'ອີເມວ',th:'อีเมล',en:'Email Address'}) }}</label>
            <input type="email" v-model="inviteForm.email" required placeholder="Email address" />
          </div>
          
          <div class="form-group">
            <label>{{ l({lo:'ບົດບາດ',th:'บทบาท',en:'Role'}) }}</label>
            <select v-model="inviteForm.role">
              <option value="admin">Admin</option>
              <option value="manager">Manager</option>
              <option value="employee">Employee</option>
            </select>
          </div>
          
          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeInviteModal">{{ l({lo:'ຍົກເລີກ',th:'ยกเลิก',en:'Cancel'}) }}</button>
            <button type="submit" class="btn-primary" :disabled="inviteLoading">
              {{ inviteLoading ? l({lo:'ກຳລັງເຊີນ...',th:'กำลังเชิญ...',en:'Inviting...'}) : l({lo:'ສົ່ງຄຳເຊີນ',th:'ส่งคำเชิญ',en:'Send Invite'}) }}
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
.role-select{min-height:40px;padding:6px 10px;border:1px solid var(--color-border);border-radius:9px;color:var(--color-text);background:var(--color-surface)}
.inline-action{margin-left:8px;border:0;color:var(--color-primary);background:none;font-weight:700;cursor:pointer}

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
