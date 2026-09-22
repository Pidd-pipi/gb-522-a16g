import { computed, ref } from 'vue'

export function useTraceViewport(length = 100) {
  const start = ref(0); const end = ref(100)
  const visibleRange = computed(() => ({ startIndex: Math.floor((start.value / 100) * length), endIndex: Math.ceil((end.value / 100) * length) }))
  function update(nextStart: number, nextEnd: number) { start.value = Math.max(0, Math.min(nextStart, 99)); end.value = Math.max(start.value + 1, Math.min(nextEnd, 100)) }
  function reset() { start.value = 0; end.value = 100 }
  return { start, end, visibleRange, update, reset }
}
