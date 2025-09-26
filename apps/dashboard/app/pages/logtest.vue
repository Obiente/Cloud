<script setup lang="ts">
definePageMeta({
  layout: false,
})

interface LogEntry {
  type: 'connected' | 'disconnected' | 'error' | 'stdout' | 'stderr';
  data: string;
  timestamp: string;
}

interface ContainerListItem {
  id: string;
  names: string[];
  image: string;
  state: string;
  status: string;
  created: number;
}

const selectedContainer = ref<ContainerListItem | null>(null)
const logs = ref<LogEntry[]>([])
const isConnected = ref(false)
const isLoading = ref(false)
const error = ref('')
const eventSource = ref<EventSource | null>(null)
const logContainer = ref<HTMLElement | null>(null)
const autoScroll = ref(true)
const availableContainers = ref<ContainerListItem[]>([])
const isLoadingContainers = ref(false)

// Load available containers on mount
onMounted(async () => {
  await loadContainers()
})

const loadContainers = async () => {
  isLoadingContainers.value = true
  try {
    const response = await $fetch('/api/docker/list-containers?all=true')
    if (response.success) {
      availableContainers.value = response.containers.map((container: any) => ({
        id: container.id || '',
        names: container.names || [],
        image: container.image || '',
        state: container.state || '',
        status: container.status || '',
        created: container.created || 0
      }))
    }
  } catch (err) {
    console.error('Failed to load containers:', err)
    error.value = 'Failed to load containers: ' + (err as Error).message
  } finally {
    isLoadingContainers.value = false
  }
}

const selectContainer = (container: ContainerListItem) => {
  selectedContainer.value = container
  error.value = ''
}

const connectToContainer = () => {
  if (!selectedContainer.value) {
    error.value = 'Please select a container first'
    return
  }

  if (isConnected.value) {
    disconnect()
  }

  isLoading.value = true
  error.value = ''
  logs.value = []
  
  startLogStream()
}

const startLogStream = () => {
  if (!selectedContainer.value) return

  const url = `/api/docker/attach-stream?id=${encodeURIComponent(selectedContainer.value.id)}&follow=true&timestamps=true&tail=100`
  
  eventSource.value = new EventSource(url)
  
  eventSource.value.onmessage = (event) => {
    try {
      const data: LogEntry = JSON.parse(event.data)
      logs.value.push(data)
      
      if (data.type === 'connected') {
        isConnected.value = true
        isLoading.value = false
      } else if (data.type === 'disconnected') {
        isConnected.value = false
        isLoading.value = false
      } else if (data.type === 'error') {
        error.value = data.data || 'Unknown error occurred'
        disconnect()
      }

      scrollToBottom()
    } catch (err) {
      console.error('Error parsing SSE message:', err)
    }
  }

  eventSource.value.onerror = (err) => {
    console.error('EventSource error:', err)
    error.value = 'Connection lost'
    disconnect()
  }
}

const disconnect = () => {
  if (eventSource.value) {
    eventSource.value.close()
    eventSource.value = null
  }
  isConnected.value = false
  isLoading.value = false
  
  if (logs.value.length > 0) {
    logs.value.push({
      type: 'disconnected',
      data: 'Disconnected from container',
      timestamp: new Date().toISOString()
    })
  }
}

const scrollToBottom = () => {
  if (logContainer.value && autoScroll.value) {
    nextTick(() => {
      logContainer.value!.scrollTop = logContainer.value!.scrollHeight
    })
  }
}

const clearLogs = () => {
  logs.value = []
}

const formatTimestamp = (timestamp: string) => {
  return new Date(timestamp).toLocaleTimeString()
}

const getLogTypeClass = (type: string) => {
  switch (type) {
    case 'connected':
      return 'text-green-400'
    case 'disconnected':
      return 'text-yellow-400'
    case 'error':
      return 'text-red-400'
    case 'stdout':
      return 'text-green-400'
    case 'stderr':
      return 'text-red-400'
    default:
      return 'text-green-400'
  }
}

// Cleanup on unmount
onUnmounted(() => {
  disconnect()
})
</script>

<template>
  <div class="min-h-screen bg-gray-100 p-6">
    <div class="max-w-6xl mx-auto">
      <div class="bg-white rounded-lg shadow-lg">
        <!-- Header -->
        <div class="border-b border-gray-200 p-6">
          <h1 class="text-2xl font-bold text-gray-900 mb-6">Docker Container Log Viewer</h1>
          
          <!-- Container Selection -->
          <div v-if="!selectedContainer" class="space-y-4">
            <div class="flex items-center justify-between">
              <h2 class="text-lg font-medium text-gray-900">Select a Container</h2>
              <button
                @click="loadContainers"
                :disabled="isLoadingContainers"
                class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
              >
                {{ isLoadingContainers ? 'Loading...' : 'Refresh' }}
              </button>
            </div>
            
            <div v-if="isLoadingContainers" class="text-center py-8 text-gray-500">
              Loading containers...
            </div>
            
            <div v-else-if="availableContainers.length === 0" class="text-center py-8 text-gray-500">
              No containers found
            </div>
            
            <div v-else class="grid gap-4">
              <div
                v-for="container in availableContainers"
                :key="container.id"
                @click="selectContainer(container)"
                class="p-4 border border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 cursor-pointer transition-colors"
              >
                <div class="flex items-center justify-between">
                  <div class="flex-1">
                    <div class="font-medium text-gray-900">
                      {{ container.names[0]?.replace(/^\//, '') || 'Unnamed Container' }}
                    </div>
                    <div class="text-sm text-gray-500 mt-1">
                      ID: {{ container.id.substring(0, 12) }} • Image: {{ container.image }}
                    </div>
                  </div>
                  <div class="text-right">
                    <span 
                      class="inline-flex px-3 py-1 text-xs font-semibold rounded-full"
                      :class="container.state === 'running' ? 'bg-green-100 text-green-800' : 
                              container.state === 'exited' ? 'bg-red-100 text-red-800' : 
                              'bg-gray-100 text-gray-800'"
                    >
                      {{ container.state }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <!-- Selected Container Info -->
          <div v-else class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-lg font-medium text-gray-900">
                  {{ selectedContainer.names[0]?.replace(/^\//, '') || 'Unnamed Container' }}
                </h2>
                <div class="text-sm text-gray-500">
                  {{ selectedContainer.id.substring(0, 12) }} • {{ selectedContainer.image }}
                </div>
              </div>
              <div class="flex items-center gap-4">
                <span 
                  class="inline-flex px-3 py-1 text-xs font-semibold rounded-full"
                  :class="selectedContainer.state === 'running' ? 'bg-green-100 text-green-800' : 
                          selectedContainer.state === 'exited' ? 'bg-red-100 text-red-800' : 
                          'bg-gray-100 text-gray-800'"
                >
                  {{ selectedContainer.state }}
                </span>
                <button
                  @click="selectedContainer = null; disconnect()"
                  class="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700"
                >
                  Change Container
                </button>
              </div>
            </div>
            
            <div class="flex gap-4">
              <button
                @click="connectToContainer"
                :disabled="isLoading || isConnected"
                class="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <span v-if="isLoading">Connecting...</span>
                <span v-else-if="isConnected">Connected</span>
                <span v-else>Connect to Logs</span>
              </button>
              
              <button
                v-if="isConnected"
                @click="disconnect"
                class="px-6 py-2 bg-red-600 text-white rounded-md hover:bg-red-700"
              >
                Disconnect
              </button>
            </div>
          </div>

          <!-- Error Display -->
          <div v-if="error" class="bg-red-50 border border-red-200 rounded-md p-4 mt-4">
            <div class="flex">
              <div class="text-red-800">
                <strong>Error:</strong> {{ error }}
              </div>
            </div>
          </div>
        </div>

        <!-- Log Controls -->
        <div v-if="selectedContainer" class="border-b border-gray-200 p-4 bg-gray-50">
          <div class="flex justify-between items-center">
            <div class="flex items-center gap-4">
              <div class="flex items-center">
                <span class="text-sm text-gray-600">Status:</span>
                <span class="ml-2 inline-flex items-center">
                  <div 
                    class="w-2 h-2 rounded-full mr-2"
                    :class="isConnected ? 'bg-green-500' : 'bg-red-500'"
                  ></div>
                  {{ isConnected ? 'Connected' : 'Disconnected' }}
                </span>
              </div>
              
              <div class="text-sm text-gray-600">
                Messages: {{ logs.length }}
              </div>
            </div>
            
            <div class="flex gap-2">
              <label class="flex items-center text-sm text-gray-600">
                <input
                  type="checkbox"
                  v-model="autoScroll"
                  class="mr-2"
                />
                Auto-scroll
              </label>
              
              <button
                @click="clearLogs"
                class="px-3 py-1 bg-gray-600 text-white text-sm rounded hover:bg-gray-700"
              >
                Clear Logs
              </button>
            </div>
          </div>
        </div>

        <!-- Log Display -->
        <div v-if="selectedContainer" class="relative">
          <div
            ref="logContainer"
            class="h-96 overflow-auto bg-black text-green-400 font-mono text-sm p-4"
          >
            <div v-if="logs.length === 0 && !isLoading" class="text-gray-500 text-center py-8">
              No logs to display. Click "Connect to Logs" to start streaming.
            </div>
            
            <div v-if="isLoading" class="text-yellow-400 text-center py-8">
              Connecting to container...
            </div>

            <div
              v-for="(log, index) in logs"
              :key="index"
              class="mb-1 leading-relaxed"
              :class="getLogTypeClass(log.type)"
            >
              <span class="text-gray-500 mr-2">
                [{{ formatTimestamp(log.timestamp) }}]
              </span>
              
              <span class="text-blue-400 mr-2">
                [{{ log.type }}]
              </span>
              
              <span class="whitespace-pre-wrap">{{ log.data }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Custom scrollbar for dark log container */
.overflow-auto::-webkit-scrollbar {
  width: 8px;
}

.overflow-auto::-webkit-scrollbar-track {
  background: #1f2937;
}

.overflow-auto::-webkit-scrollbar-thumb {
  background: #4b5563;
  border-radius: 4px;
}

.overflow-auto::-webkit-scrollbar-thumb:hover {
  background: #6b7280;
}
</style>