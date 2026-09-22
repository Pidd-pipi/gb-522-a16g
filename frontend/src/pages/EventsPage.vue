<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Filter, Radar } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EventTypeBadge from '@/components/common/EventTypeBadge.vue'
import ReviewDialog from '@/components/common/ReviewDialog.vue'
import { useEventStore } from '@/stores/events'
import { useRouteStore } from '@/stores/routes'
import { useAuth } from '@/hooks/useAuth'
import { EVENT_TYPES, eventLabel, type EventMarker, type EventType } from '@/types/event'

const events = useEventStore(); const routes = useRouteStore(); const auth = useAuth(); const dialog = ref(false); const current = ref<EventMarker | null>(null); const busy = ref(false)
const filters = reactive<{ route_id?: number; event_type?: EventType; reviewed?: boolean }>({})
async function search() { await events.fetch({ ...filters, page_size: 100 }) }
function openReview(item: EventMarker) { current.value = item; dialog.value = true }
async function review(value: Record<string, unknown>) { if (!current.value) return; busy.value = true; try { await events.review(current.value.id, value as {event_type:EventType;distance_m?:number;review_note:string}); dialog.value = false; ElMessage.success('事件复核结果已保存') } finally { busy.value = false } }
onMounted(async () => { await Promise.all([routes.fetch({page_size:100}), search()]) })
</script>

<template>
  <PageHeader title="事件复核" eyebrow="EVENT REVIEW" description="对照算法原值修订事件类型、距离和备注。" />
  <section class="content-band">
    <div class="filter-band"><div class="filter-label"><Filter :size="16" /><span>筛选条件</span></div><el-select v-model="filters.route_id" clearable placeholder="全部线路" @change="search"><el-option v-for="route in routes.items" :key="route.id" :label="route.route_code" :value="route.id" /></el-select><el-select v-model="filters.event_type" clearable placeholder="全部类型" @change="search"><el-option v-for="type in EVENT_TYPES" :key="type" :label="eventLabel[type]" :value="type" /></el-select><el-select v-model="filters.reviewed" clearable placeholder="全部状态" @change="search"><el-option label="未复核" :value="false" /><el-option label="已复核" :value="true" /></el-select><span class="toolbar-spacer subtle-count">{{ events.total }} 项事件</span></div>
    <div class="data-surface">
      <el-table v-loading="events.loading" :data="events.items" row-key="id">
        <el-table-column label="类型" width="125"><template #default="scope"><EventTypeBadge :type="scope.row.event_type" :reviewed="scope.row.reviewed" /></template></el-table-column>
        <el-table-column prop="trace_id" label="轨迹" width="85"><template #default="scope">#{{ scope.row.trace_id }}</template></el-table-column>
        <el-table-column prop="distance_m" label="定位距离" width="120"><template #default="scope"><strong>{{ scope.row.distance_m.toFixed(2) }}</strong> m</template></el-table-column>
        <el-table-column label="算法原值" min-width="180"><template #default="scope"><span class="algorithm-value">{{ eventLabel[scope.row.algorithm_event_type as EventType] }} · {{ scope.row.algorithm_distance_m.toFixed(2) }} m</span></template></el-table-column>
        <el-table-column prop="insertion_loss_db" label="损耗 dB" width="100" />
        <el-table-column label="置信度" width="120"><template #default="scope">{{ Math.round(scope.row.confidence*100) }}%</template></el-table-column>
        <el-table-column label="复核记录" min-width="190"><template #default="scope"><span v-if="scope.row.reviewed" class="review-note">{{ scope.row.review_note }}</span><span v-else class="unreviewed">待人工复核</span></template></el-table-column>
        <el-table-column width="94" fixed="right"><template #default="scope"><el-button v-if="auth.canReview()" link type="primary" @click="openReview(scope.row)">{{ scope.row.reviewed ? '重新复核' : '复核' }}</el-button><span v-else class="muted">只读</span></template></el-table-column>
        <template #empty><div class="empty-state"><div><Radar :size="34" /><strong>没有匹配事件</strong><span>请先在轨迹分析页执行检测。</span></div></div></template>
      </el-table>
    </div>
  </section>
  <ReviewDialog v-model="dialog" mode="event" :event="current" :loading="busy" @submit="review" />
</template>

<style scoped>
.filter-band{display:flex;align-items:center;flex-wrap:wrap;gap:9px;margin-bottom:14px;padding:11px;border:1px solid var(--line);background:var(--surface)}.filter-band .el-select{width:160px}.filter-label{display:flex;align-items:center;gap:7px;margin-right:5px;color:var(--text-muted);font-size:12px;font-weight:800}.algorithm-value{color:var(--text-muted);font-size:12px}.review-note{display:-webkit-box;overflow:hidden;-webkit-line-clamp:2;-webkit-box-orient:vertical}.unreviewed{color:var(--warning);font-size:12px;font-weight:700}.muted{color:var(--text-muted);font-size:12px}@media(max-width:620px){.filter-band .el-select{width:calc(50% - 5px)}.filter-label{width:100%}}
</style>
