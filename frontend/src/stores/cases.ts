import { defineStore } from 'pinia'
import { ref } from 'vue'
import { caseApi } from '@/api/domain'
import type { LocalizationCase } from '@/types/case'

export const useCaseStore = defineStore('cases', () => {
  const items = ref<LocalizationCase[]>([]); const loading = ref(false); const total = ref(0)
  async function fetch(params?: object) { loading.value = true; try { const { data } = await caseApi.list(params); items.value = data.data; total.value = data.meta?.total ?? data.data.length } finally { loading.value = false } }
  async function create(body: object) { const { data } = await caseApi.create(body); items.value.unshift(data.data) }
  async function analyze(id: number, body: object) { const { data } = await caseApi.analyze(id, body); replace(data.data) }
  async function confirm(id: number, body: object) { const { data } = await caseApi.confirm(id, body); replace(data.data) }
  async function close(id: number, version: number) { const { data } = await caseApi.close(id, version); replace(data.data) }
  function replace(item: LocalizationCase) { const index = items.value.findIndex((current) => current.id === item.id); if (index >= 0) items.value[index] = item }
  return { items, loading, total, fetch, create, analyze, confirm, close }
})
