<template>
  <div v-if="set">
    <div class="d-flex align-center mb-6">
      <v-btn icon="mdi-arrow-left" variant="text" class="mr-2" @click="router.push('/replies')"></v-btn>
      <h1 class="text-h4 font-weight-bold">{{ set.Name }}</h1>
      <v-spacer></v-spacer>
      <v-btn color="primary" @click="saveChanges" :loading="saving" prepend-icon="mdi-content-save">Save Changes</v-btn>
    </div>

    <v-row>
      <!-- Builder Area -->
      <v-col cols="12" md="8">
        <v-card variant="outlined" class="mb-4">
          <v-card-text class="bg-grey-lighten-4">
            <div v-if="items.length === 0" class="text-center py-8 text-medium-emphasis">
              No messages in this sequence yet. Add one from the right panel.
            </div>
            
            <div v-for="(item, index) in items" :key="index" class="d-flex mb-4">
              <!-- Reorder controls -->
              <div class="d-flex flex-column justify-center mr-3">
                <v-btn icon="mdi-chevron-up" variant="text" size="small" :disabled="index === 0" @click="moveItem(index, -1)"></v-btn>
                <div class="text-center text-caption text-medium-emphasis font-weight-bold">{{ index + 1 }}</div>
                <v-btn icon="mdi-chevron-down" variant="text" size="small" :disabled="index === items.length - 1" @click="moveItem(index, 1)"></v-btn>
              </div>
              
              <!-- Item Content -->
              <v-card class="flex-grow-1" elevation="1">
                <v-card-title class="d-flex align-center text-subtitle-2 bg-grey-lighten-3 py-2 px-3">
                  <v-icon :icon="getIconForType(item.Type)" size="small" class="mr-2"></v-icon>
                  {{ item.Type.toUpperCase() }}
                  <v-spacer></v-spacer>
                  <v-btn icon="mdi-delete-outline" variant="text" size="x-small" color="error" @click="removeItem(index)"></v-btn>
                </v-card-title>
                <v-card-text class="pa-3">
                  <!-- Text -->
                  <v-textarea
                    v-if="item.Type === 'text'"
                    v-model="item.Content"
                    variant="outlined"
                    density="compact"
                    hide-details
                    rows="3"
                    placeholder="Enter message text..."
                  ></v-textarea>
                  
                  <!-- Media (Image/Video/Audio) -->
                  <div v-else>
                    <div v-if="item.Content" class="mb-2">
                      <v-img v-if="item.Type === 'image'" :src="item.Content" max-height="200" contain class="bg-grey-lighten-2 rounded"></v-img>
                      <video v-else-if="item.Type === 'video'" :src="item.Content" controls class="w-100 rounded" style="max-height: 200px"></video>
                      <audio v-else-if="item.Type === 'audio'" :src="item.Content" controls class="w-100"></audio>
                    </div>
                    <v-file-input
                      v-if="!item.Content"
                      :accept="getAcceptForType(item.Type)"
                      label="Upload file"
                      variant="outlined"
                      density="compact"
                      hide-details
                      prepend-icon="mdi-cloud-upload"
                      @change="(e: Event) => uploadMedia(e, index)"
                      :loading="uploadingIndex === index"
                    ></v-file-input>
                    <v-btn v-else variant="text" size="small" color="error" class="mt-2" @click="item.Content = ''">Remove File</v-btn>
                  </div>
                </v-card-text>
              </v-card>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Toolbox -->
      <v-col cols="12" md="4">
        <v-card variant="outlined" class="position-sticky" style="top: 24px">
          <v-card-title>Add Message</v-card-title>
          <v-card-text>
            <v-list density="compact" nav>
              <v-list-item @click="addItem('text')" prepend-icon="mdi-text" title="Text Message" subtitle="Send a text message"></v-list-item>
              <v-list-item @click="addItem('image')" prepend-icon="mdi-image-outline" title="Image" subtitle="Send a picture"></v-list-item>
              <v-list-item @click="addItem('video')" prepend-icon="mdi-video-outline" title="Video" subtitle="Send a video clip"></v-list-item>
              <v-list-item @click="addItem('audio')" prepend-icon="mdi-microphone-outline" title="Audio" subtitle="Send a voice message"></v-list-item>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
  <div v-else-if="loading" class="d-flex justify-center mt-12">
    <v-progress-circular indeterminate color="primary"></v-progress-circular>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApi } from '~/composables/useApi'
import { useAuth } from '~/composables/useAuth'

const route = useRoute()
const router = useRouter()
const api = useApi()
const auth = useAuth()

const set = ref<any>(null)
const items = ref<any[]>([])
const loading = ref(true)
const saving = ref(false)
const uploadingIndex = ref(-1)

const loadSet = async () => {
  loading.value = true
  try {
    const data = await api.get(`/api/v1/replies/${route.params.id}`)
    set.value = data
    // Copy items to local state for editing
    items.value = [...(data.Items || [])].sort((a, b) => a.OrderIndex - b.OrderIndex)
  } catch (error) {
    console.error('Failed to load set', error)
  } finally {
    loading.value = false
  }
}

const addItem = (type: string) => {
  items.value.push({
    Type: type,
    Content: '',
    OrderIndex: items.value.length
  })
}

const removeItem = (index: number) => {
  items.value.splice(index, 1)
}

const moveItem = (index: number, direction: number) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= items.value.length) return
  
  const temp = items.value[index]
  items.value[index] = items.value[newIndex]
  items.value[newIndex] = temp
}

const getIconForType = (type: string) => {
  switch (type) {
    case 'text': return 'mdi-text'
    case 'image': return 'mdi-image-outline'
    case 'video': return 'mdi-video-outline'
    case 'audio': return 'mdi-microphone-outline'
    default: return 'mdi-file-outline'
  }
}

const getAcceptForType = (type: string) => {
  switch (type) {
    case 'image': return 'image/*'
    case 'video': return 'video/*'
    case 'audio': return 'audio/*'
    default: return '*/*'
  }
}

const uploadMedia = async (event: Event, index: number) => {
  const input = event.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return
  
  const file = input.files[0]
  uploadingIndex.value = index
  
  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('folder', 'replies')
    
    // Note: this uses native fetch since we need to send FormData
    const response = await fetch('/api/v1/storage/upload', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${auth.token.value}`
      },
      body: formData
    })
    
    if (!response.ok) throw new Error('Upload failed')
    
    const data = await response.json()
    items.value[index].Content = data.url
  } catch (error) {
    console.error('Failed to upload file', error)
    alert('Upload failed')
  } finally {
    uploadingIndex.value = -1
  }
}

const saveChanges = async () => {
  saving.value = true
  try {
    // Update OrderIndex before saving
    const formattedItems = items.value.map((item, index) => ({
      ...item,
      OrderIndex: index
    }))
    
    await api.put(`/api/v1/replies/${route.params.id}/items`, { items: formattedItems })
    // Reload set
    await loadSet()
  } catch (error) {
    console.error('Failed to save changes', error)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSet()
})
</script>
