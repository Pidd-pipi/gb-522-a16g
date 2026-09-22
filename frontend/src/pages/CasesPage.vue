<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ClipboardCheck, GitCompareArrows, Plus, Play, LockKeyhole } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import ReviewDialog from '@/components/common/ReviewDialog.vue'
import { useCaseStore } from '@/stores/cases'
import { useRouteStore } from '@/stores/routes'
import { useTraceStore } from '@/stores/traces'
import { useAuth } from '@/hooks/useAuth'
import { CASE_STATUSES, caseStatusLabel, type LocalizationCase } from '@/types/case'

const cases = useCaseStore(); const routes = useRouteStore(); const traces = useTraceStore(); const auth = useAuth(); const status = ref(''); const createOpen = ref(false); const reviewOpen = ref(false); const current = ref<LocalizationCase | null>(null); const busy = ref(false)
const form = reactive({ route_id: 0, baseline_trace_id: 0, current_trace_id: 0, distance_tolerance_m: 25, loss_increase_db: 0.5 })
const routeTraces = computed(() => traces.items.filter((item) => item.route_id === form.route_id))
async function search() { await cases.fetch({ status: status.value || undefined, page_size: 100 }) }
function chooseRoute() { const route = routes.items.find((item)=>item.id===form.route_id); form.baseline_trace_id = route?.baseline_trace_id ?? 0; form.current_trace_id = routeTraces.value.find((trace)=>trace.id !== form.baseline_trace_id)?.id ?? 0 }
async function create() { busy.value=true; try { await cases.create(form); createOpen.value=false; ElMessage.success('定位案例已建立') } finally { busy.value=false } }
async function analyze(item: LocalizationCase) { busy.value=true; try { await cases.analyze(item.id, {}); ElMessage.success('基线差异分析已完成') } finally { busy.value=false } }
function confirm(item: LocalizationCase) { current.value=item; reviewOpen.value=true }
async function submitReview(value: Record<string, unknown>) { if(!current.value)return; busy.value=true; try { await cases.confirm(current.value.id,value); reviewOpen.value=false; ElMessage.success('人工结论已确认') } finally { busy.value=false } }
async function close(item: LocalizationCase) { await ElMessageBox.confirm('关闭后案例不可再修改，确认继续？','关闭案例',{type:'warning',confirmButtonText:'确认关闭',cancelButtonText:'取消'}); await cases.close(item.id,item.version); ElMessage.success('案例已关闭') }
onMounted(async()=>{ await Promise.all([routes.fetch({page_size:100}),traces.fetch({page_size:100}),search()]); if(routes.items[0]){form.route_id=routes.items[0].id;chooseRoute()} })
</script>

<template>
  <PageHeader title="定位案例" eyebrow="LOCALIZATION CASES" description="基线差异仅作分析参考，由复核人员形成最终结论。"><el-button v-if="auth.canAnalyze()" type="primary" @click="createOpen=true"><Plus :size="16" />新建案例</el-button></PageHeader>
  <section class="content-band">
    <div class="case-toolbar"><div class="state-track"><span v-for="(value,index) in CASE_STATUSES" :key="value"><i>{{ index+1 }}</i>{{ caseStatusLabel[value] }}</span></div><el-select v-model="status" clearable placeholder="全部状态" style="width:150px" @change="search"><el-option v-for="value in CASE_STATUSES" :key="value" :label="caseStatusLabel[value]" :value="value" /></el-select></div>
    <div class="data-surface">
      <el-table v-loading="cases.loading" :data="cases.items" row-key="id">
        <el-table-column prop="id" label="案例" width="80"><template #default="scope"><strong>#{{ scope.row.id }}</strong></template></el-table-column>
        <el-table-column label="线路" width="125"><template #default="scope">{{ routes.items.find(r=>r.id===scope.row.route_id)?.route_code ?? `#${scope.row.route_id}` }}</template></el-table-column>
        <el-table-column label="对比轨迹" min-width="150"><template #default="scope"><span class="trace-pair">#{{ scope.row.baseline_trace_id }} <GitCompareArrows :size="14" /> #{{ scope.row.current_trace_id }}</span></template></el-table-column>
        <el-table-column label="状态" width="115"><template #default="scope"><span class="status-pill" :class="scope.row.case_status">{{ caseStatusLabel[scope.row.case_status as LocalizationCase['case_status']] }}</span></template></el-table-column>
        <el-table-column label="估算位置" width="150"><template #default="scope"><template v-if="scope.row.estimated_distance_m != null"><strong>{{ scope.row.estimated_distance_m.toFixed(2) }} m</strong><small class="uncertainty">± {{ scope.row.uncertainty_m?.toFixed(2) }} m</small></template><span v-else class="muted">待分析</span></template></el-table-column>
        <el-table-column prop="conclusion" label="复核结论" min-width="230"><template #default="scope"><span v-if="scope.row.conclusion" class="conclusion">{{ scope.row.conclusion }}</span><span v-else class="muted">尚未确认</span></template></el-table-column>
        <el-table-column label="操作" width="190" fixed="right"><template #default="scope"><el-button v-if="scope.row.case_status==='draft' && auth.canAnalyze()" link type="primary" :loading="busy" @click="analyze(scope.row)"><Play :size="14" />分析</el-button><el-button v-if="scope.row.case_status==='pending_review' && auth.canReview()" link type="primary" @click="confirm(scope.row)"><ClipboardCheck :size="14" />确认</el-button><el-button v-if="scope.row.case_status==='confirmed' && auth.canReview()" link type="danger" @click="close(scope.row)"><LockKeyhole :size="14" />关闭</el-button><span v-if="scope.row.case_status==='closed'" class="muted">已归档</span></template></el-table-column>
        <template #empty><div class="empty-state"><div><ClipboardCheck :size="34" /><strong>尚无定位案例</strong><span>选择同线路基线与当前轨迹建立对比。</span></div></div></template>
      </el-table>
    </div>
  </section>
  <el-dialog v-model="createOpen" title="建立基线对比案例" width="min(620px, calc(100vw - 28px))"><el-form label-position="top"><el-form-item label="线路"><el-select v-model="form.route_id" style="width:100%" @change="chooseRoute"><el-option v-for="route in routes.items" :key="route.id" :label="`${route.route_code} / ${route.name}`" :value="route.id" /></el-select></el-form-item><div class="case-form-grid"><el-form-item label="基线轨迹"><el-select v-model="form.baseline_trace_id" style="width:100%"><el-option v-for="trace in routeTraces" :key="trace.id" :label="`#${trace.id} / ${trace.wavelength_nm}nm`" :value="trace.id" /></el-select></el-form-item><el-form-item label="当前轨迹"><el-select v-model="form.current_trace_id" style="width:100%"><el-option v-for="trace in routeTraces" :key="trace.id" :label="`#${trace.id} / ${trace.wavelength_nm}nm`" :value="trace.id" /></el-select></el-form-item><el-form-item label="距离容差 m"><el-input-number v-model="form.distance_tolerance_m" :min="0.1" :max="1000" style="width:100%" /></el-form-item><el-form-item label="损耗增量阈值 dB"><el-input-number v-model="form.loss_increase_db" :min="0.1" :max="20" :step="0.1" style="width:100%" /></el-form-item></div></el-form><template #footer><el-button @click="createOpen=false">取消</el-button><el-button type="primary" :loading="busy" :disabled="!form.route_id || !form.baseline_trace_id || !form.current_trace_id || form.baseline_trace_id===form.current_trace_id" @click="create">建立案例</el-button></template></el-dialog>
  <ReviewDialog v-model="reviewOpen" mode="case" :case-item="current" :loading="busy" @submit="submitReview" />
</template>

<style scoped>
.case-toolbar{display:flex;align-items:center;justify-content:space-between;gap:20px;margin-bottom:14px}.state-track{display:flex;align-items:center;overflow:auto}.state-track span{display:flex;align-items:center;gap:6px;color:var(--text-muted);font-size:11px;font-weight:700;white-space:nowrap}.state-track span:not(:last-child)::after{content:'';width:24px;height:1px;margin:0 7px;background:var(--line-strong)}.state-track i{width:20px;height:20px;display:grid;place-items:center;border:1px solid var(--line-strong);border-radius:50%;font-style:normal}.trace-pair{display:inline-flex;align-items:center;gap:6px}.uncertainty{display:block;margin-top:2px;color:var(--text-muted)}.conclusion{display:-webkit-box;overflow:hidden;-webkit-line-clamp:2;-webkit-box-orient:vertical}.muted{color:var(--text-muted);font-size:12px}.case-form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 14px}@media(max-width:700px){.case-toolbar{align-items:stretch;flex-direction:column}.case-form-grid{grid-template-columns:1fr}}
</style>
