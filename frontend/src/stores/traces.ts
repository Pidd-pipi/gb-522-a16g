import { defineStore } from 'pinia'
import { ref } from 'vue'
import { traceApi } from '@/api/domain'
import type { TraceCapture, TraceEnvelope } from '@/types/domain'

export const useTraceStore = defineStore('traces', () => {
  const items = ref<TraceCapture[]>([]); const selected = ref<TraceEnvelope | null>(null); const loading = ref(false); const total = ref(0)
  async function fetch(params?: object) { loading.value = true; try { const { data } = await traceApi.list(params); items.value = data.data; total.value = data.meta?.total ?? data.data.length } finally { loading.value = false } }
  async function select(id: number) { const { data } = await traceApi.detail(id); selected.value = data.data }
  async function importTrace(body: object) { const { data } = await traceApi.import(body); items.value.unshift(data.data); await select(data.data.id); return data.data }
  async function detect(id: number, body: object) { const { data } = await traceApi.detect(id, body); await select(id); return data.data }
  return { items, selected, loading, total, fetch, select, importTrace, detect }
})
