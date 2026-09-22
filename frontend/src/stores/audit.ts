import { defineStore } from 'pinia'
import { ref } from 'vue'
import { auditApi } from '@/api/domain'
import type { AuditLog } from '@/types/domain'

export const useAuditStore = defineStore('audit', () => {
  const items = ref<AuditLog[]>([]); const loading = ref(false); const total = ref(0)
  async function fetch(params?: object) { loading.value = true; try { const { data } = await auditApi.list(params); items.value = data.data; total.value = data.meta?.total ?? data.data.length } finally { loading.value = false } }
  return { items, loading, total, fetch }
})
