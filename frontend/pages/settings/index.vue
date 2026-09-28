<script setup lang="ts">
const { user } = useAuth()
</script>

<template>
  <div class="settings-section">
    <h2>Profile Settings</h2>
    <p class="subtitle">Manage your personal account details.</p>
    
    <div class="profile-card" v-if="user">
      <div class="avatar">
        {{ user.name.charAt(0).toUpperCase() }}
      </div>
      <div class="info">
        <h3>{{ user.name }}</h3>
        <p>{{ user.email }}</p>
        <span class="role-badge">User ID: {{ user.id }}</span>
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
.profile-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 24px;
  border-radius: 12px;
  border: 1px solid var(--color-border);
  background: var(--color-background);
}
.avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--color-primary);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 700;
}
.info h3 {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 4px;
}
.info p {
  color: var(--color-text-secondary);
  font-size: 14px;
  margin-bottom: 8px;
}
.role-badge {
  display: inline-block;
  padding: 4px 8px;
  background: var(--color-surface-hover);
  border-radius: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
  font-family: monospace;
}
</style>
