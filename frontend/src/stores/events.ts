import { defineStore } from 'pinia'
import { ref } from 'vue'
import { eventApi } from '@/api/domain'
import type { EventMarker, EventType } from '@/types/event'

export const useEventStore = defineStore('events', () => {
  const items = ref<EventMarker[]>([]); const loading = ref(false); const total = ref(0)
  async function fetch(params?: object) { loading.value = true; try { const { data } = await eventApi.list(params); items.value = data.data; total.value = data.meta?.total ?? data.data.length } finally { loading.value = false } }
  async function review(id: number, body: { event_type: EventType; distance_m?: number; review_note: string }) { const { data } = await eventApi.review(id, body); const index = items.value.findIndex((item) => item.id === id); if (index >= 0) items.value[index] = data.data }
  return { items, loading, total, fetch, review }
})
