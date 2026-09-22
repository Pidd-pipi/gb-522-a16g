import { defineStore } from 'pinia'
import { ref } from 'vue'
import { routeApi } from '@/api/domain'
import type { FiberRoute } from '@/types/domain'

export const useRouteStore = defineStore('routes', () => {
  const items = ref<FiberRoute[]>([]); const loading = ref(false); const total = ref(0)
  async function fetch(params?: object) { loading.value = true; try { const { data } = await routeApi.list(params); items.value = data.data; total.value = data.meta?.total ?? data.data.length } finally { loading.value = false } }
  async function create(body: object) { const { data } = await routeApi.create(body); items.value.unshift(data.data); total.value++ }
  async function update(id: number, body: object) { const { data } = await routeApi.update(id, body); const index = items.value.findIndex((item) => item.id === id); if (index >= 0) items.value[index] = data.data; return data.data }
  async function setBaseline(routeId: number, traceId: number) { const { data } = await routeApi.setBaseline(routeId, traceId); const index = items.value.findIndex((item) => item.id === routeId); if (index >= 0) items.value[index] = data.data }
  return { items, loading, total, fetch, create, update, setBaseline }
})
