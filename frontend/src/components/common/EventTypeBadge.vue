<script setup lang="ts">
import { computed } from 'vue'
import { eventLabel, type EventType } from '@/types/event'

const props = defineProps<{ type: EventType; reviewed?: boolean }>()
const tone = computed(() => ({ connector: 'teal', splice: 'blue', bend: 'amber', break: 'red', end: 'charcoal', unknown: 'gray' }[props.type]))
</script>

<template>
  <span class="event-badge" :class="[`tone-${tone}`, { reviewed }]">
    <span class="event-dot" aria-hidden="true" />
    {{ eventLabel[type] }}
  </span>
</template>

<style scoped>
.event-badge { display: inline-flex; align-items: center; gap: 6px; min-height: 24px; padding: 2px 8px; border: 1px solid var(--badge-border); border-radius: 3px; color: var(--badge-text); background: var(--badge-bg); font-size: 12px; font-weight: 700; white-space: nowrap; }
.event-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.tone-teal { --badge-bg: #e8f5f1; --badge-border: #a7d9ca; --badge-text: #146451; }
.tone-blue { --badge-bg: #eaf2f7; --badge-border: #aec9d9; --badge-text: #275d78; }
.tone-amber { --badge-bg: #fff4d8; --badge-border: #e8c66e; --badge-text: #755310; }
.tone-red { --badge-bg: #fbeceb; --badge-border: #e6afaa; --badge-text: #9c3028; }
.tone-charcoal { --badge-bg: #e9eeec; --badge-border: #b7c1bd; --badge-text: #34413d; }
.tone-gray { --badge-bg: #f2f4f3; --badge-border: #ced4d1; --badge-text: #5c6662; }
.reviewed { box-shadow: inset 0 -2px currentColor; }
</style>
