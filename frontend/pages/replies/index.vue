<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-6">
      <div>
        <h1 class="text-h4 font-weight-bold">Quick Replies</h1>
        <p class="text-subtitle-1 text-medium-emphasis">Manage your canned responses and message sequences</p>
      </div>
      <v-btn
        color="primary"
        prepend-icon="mdi-plus"
        @click="createDialog = true"
      >
        Create Set
      </v-btn>
    </div>

    <v-row v-if="loading">
      <v-col cols="12" sm="6" md="4" v-for="i in 3" :key="i">
        <v-skeleton-loader type="card"></v-skeleton-loader>
      </v-col>
    </v-row>

    <v-row v-else-if="sets.length > 0">
      <v-col cols="12" sm="6" md="4" v-for="set in sets" :key="set.ID">
        <v-card variant="outlined" class="h-100 d-flex flex-column transition-swing hover-card" @click="editSet(set.ID)">
          <v-card-title class="d-flex align-center">
            <v-icon icon="mdi-message-flash-outline" color="primary" class="mr-2"></v-icon>
            {{ set.Name }}
            <v-spacer></v-spacer>
            <v-btn icon="mdi-delete-outline" variant="text" size="small" color="error" @click.stop="confirmDelete(set)"></v-btn>
          </v-card-title>
          <v-card-text class="flex-grow-1">
            <div class="text-body-2 text-medium-emphasis mb-2">
              Contains {{ set.Items?.length || 0 }} items
            </div>
            <div class="d-flex flex-wrap gap-1">
              <v-chip size="x-small" v-for="(count, type) in getStats(set.Items)" :key="type" color="secondary" variant="flat">
                {{ count }} {{ type }}
              </v-chip>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-empty-state
      v-else
      icon="mdi-message-text-outline"
      title="No quick replies yet"
      text="Create your first message sequence to reply to customers faster."
      action-text="Create Set"
      @click:action="createDialog = true"
    ></v-empty-state>

    <!-- Create Dialog -->
    <v-dialog v-model="createDialog" max-width="500">
      <v-card>
        <v-card-title>Create Quick Reply Set</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="newSetName"
            label="Set Name"
            placeholder="e.g. Promotion A, Welcome Message"
            variant="outlined"
            @keyup.enter="createSet"
            autofocus
          ></v-text-field>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="createDialog = false">Cancel</v-btn>
          <v-btn color="primary" @click="createSet" :loading="creating" :disabled="!newSetName.trim()">Create</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirm -->
    <v-dialog v-model="deleteDialog" max-width="400">
      <v-card>
        <v-card-title>Delete Set?</v-card-title>
        <v-card-text>
          Are you sure you want to delete "{{ setToDelete?.Name }}"? This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" @click="deleteSet" :loading="deleting">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useApi } from '~/composables/useApi'

const router = useRouter()
const api = useApi()

const sets = ref<any[]>([])
const loading = ref(true)

// Create
const createDialog = ref(false)
const newSetName = ref('')
const creating = ref(false)

// Delete
const deleteDialog = ref(false)
const setToDelete = ref<any>(null)
const deleting = ref(false)

const loadSets = async () => {
  loading.value = true
  try {
    const data = await api.get('/api/v1/replies')
    sets.value = data || []
  } catch (error) {
    console.error('Failed to load sets', error)
  } finally {
    loading.value = false
  }
}

const createSet = async () => {
  if (!newSetName.value.trim()) return
  
  creating.value = true
  try {
    const data = await api.post('/api/v1/replies', { name: newSetName.value.trim() })
    createDialog.value = false
    newSetName.value = ''
    router.push(`/replies/${data.ID}`)
  } catch (error) {
    console.error('Failed to create set', error)
  } finally {
    creating.value = false
  }
}

const editSet = (id: string) => {
  router.push(`/replies/${id}`)
}

const confirmDelete = (set: any) => {
  setToDelete.value = set
  deleteDialog.value = true
}

const deleteSet = async () => {
  if (!setToDelete.value) return
  
  deleting.value = true
  try {
    await api.delete(`/api/v1/replies/${setToDelete.value.ID}`)
    sets.value = sets.value.filter(s => s.ID !== setToDelete.value.ID)
    deleteDialog.value = false
  } catch (error) {
    console.error('Failed to delete set', error)
  } finally {
    deleting.value = false
    setToDelete.value = null
  }
}

const getStats = (items: any[]) => {
  if (!items || items.length === 0) return {}
  return items.reduce((acc: any, item: any) => {
    acc[item.Type] = (acc[item.Type] || 0) + 1
    return acc
  }, {})
}

onMounted(() => {
  loadSets()
})
</script>

<style scoped>
.hover-card:hover {
  border-color: rgb(var(--v-theme-primary));
  cursor: pointer;
}
.gap-1 {
  gap: 4px;
}
</style>
