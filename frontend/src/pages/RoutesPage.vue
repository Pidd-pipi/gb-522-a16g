<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Cable, CheckCircle2, Plus, Search, Pencil, Ban, ListChecks, ChartLine } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import TraceChart from '@/components/common/TraceChart.vue'
import { useRouteStore } from '@/stores/routes'
import { routeApi, traceApi } from '@/api/domain'
import { useAuth } from '@/hooks/useAuth'
import { caseStatusLabel } from '@/types/case'
import type { FiberRoute, TraceCapture, TraceEnvelope, PendingCase, BaselineRejection, BaselineRejectReason } from '@/types/domain'

const store = useRouteStore(); const auth = useAuth(); const keyword = ref(''); const createOpen = ref(false); const detailOpen = ref(false); const busy = ref(false)
const selected = ref<FiberRoute | null>(null); const history = ref<TraceCapture[]>([]); const preview = ref<TraceEnvelope | null>(null); const editOpen = ref(false)
const pendingCases = ref<PendingCase[]>([]); const rejections = ref<BaselineRejection[]>([]); const activeTab = ref<'traces' | 'cases' | 'rejections'>('traces')
const form = reactive({ route_code: '', name: '', length_m: 12000, refractive_index: 1.468, launch_connector: 'SC/APC', route_status: 'active' })
const editForm = reactive({ name: '', length_m: 0, refractive_index: 1.468, launch_connector: '', route_status: 'active' })

const rejectReasonLabel: Record<BaselineRejectReason, string> = {
  open_cases: '存在未关闭案例引用当前基线',
  trace_not_on_route: '所选轨迹不属于该线路',
  same_baseline: '该轨迹已是当前基线，重复操作不生效',
  baseline_concurrent_change: '基线已被其他操作并发修改',
}

async function search() { await store.fetch({ keyword: keyword.value, page_size: 50 }) }
async function create() { busy.value = true; try { await store.create(form); createOpen.value = false; Object.assign(form, { route_code: '', name: '', length_m: 12000, refractive_index: 1.468, launch_connector: 'SC/APC', route_status: 'active' }); ElMessage.success('线路档案已建立') } finally { busy.value = false } }
async function loadDetail() {
  if (!selected.value) return
  const { data } = await routeApi.detail(selected.value.id)
  selected.value = data.data.route
  history.value = data.data.traces
  pendingCases.value = data.data.pending_cases
  rejections.value = data.data.rejections
  if (!history.value.some((trace) => trace.id === preview.value?.trace.id)) preview.value = null
  if (!preview.value && history.value[0]) await showTrace(history.value[0].id)
}
async function inspect(route: FiberRoute) { selected.value = route; preview.value = null; activeTab.value = 'traces'; detailOpen.value = true; await loadDetail() }
async function showTrace(id: number) { const { data } = await traceApi.detail(id); preview.value = data.data }
async function setBaseline(traceId: number) {
  if (!selected.value) return
  try {
    await store.setBaseline(selected.value.id, traceId)
    selected.value.baseline_trace_id = traceId
    ElMessage.success('基线轨迹已更新')
    await loadDetail()
  } catch {
    // The global interceptor shows the error code; reload the committed
    // rejection and surface it in the rejection tab.
    await loadDetail()
    activeTab.value = 'rejections'
  }
}
function openEdit() { if (!selected.value) return; Object.assign(editForm, { name: selected.value.name, length_m: selected.value.length_m, refractive_index: selected.value.refractive_index, launch_connector: selected.value.launch_connector, route_status: selected.value.route_status }); editOpen.value = true }
async function update() { if (!selected.value) return; busy.value = true; try { const updated = await store.update(selected.value.id, editForm); selected.value = updated; editOpen.value = false; ElMessage.success('线路档案已更新') } finally { busy.value = false } }
onMounted(search)
</script>

<template>
  <PageHeader title="线路档案" eyebrow="FIBER ROUTES" description="维护线路光学参数、历史轨迹与复核基线。">
    <el-button v-if="auth.canAnalyze()" type="primary" @click="createOpen = true"><Plus :size="16" />新建线路</el-button>
  </PageHeader>
  <section class="content-band">
    <div class="toolbar"><el-input v-model="keyword" clearable placeholder="线路编码或名称" style="width: min(320px, 100%)" @keyup.enter="search"><template #prefix><Search :size="15" /></template></el-input><el-button @click="search">检索</el-button><span class="toolbar-spacer subtle-count">共 {{ store.total }} 条线路</span></div>
    <div class="data-surface">
      <el-table v-loading="store.loading" :data="store.items" row-key="id" @row-dblclick="inspect">
        <el-table-column prop="route_code" label="线路编码" min-width="130"><template #default="scope"><strong>{{ scope.row.route_code }}</strong></template></el-table-column>
        <el-table-column prop="name" label="线路名称" min-width="180" />
        <el-table-column prop="length_m" label="长度" width="120"><template #default="scope">{{ scope.row.length_m.toLocaleString() }} m</template></el-table-column>
        <el-table-column prop="refractive_index" label="折射率" width="100" />
        <el-table-column prop="launch_connector" label="发射端" width="110" />
        <el-table-column label="基线" width="105"><template #default="scope"><span v-if="scope.row.baseline_trace_id" class="baseline"><CheckCircle2 :size="14" />#{{ scope.row.baseline_trace_id }}</span><span v-else class="muted">未设置</span></template></el-table-column>
        <el-table-column label="状态" width="105"><template #default="scope"><span class="status-pill">{{ scope.row.route_status }}</span></template></el-table-column>
        <el-table-column width="92" fixed="right"><template #default="scope"><el-button link type="primary" @click="inspect(scope.row)">查看</el-button></template></el-table-column>
        <template #empty><div class="empty-state"><div><Cable :size="32" /><strong>尚无线路档案</strong><span>建立线路后才能导入 OTDR 轨迹。</span></div></div></template>
      </el-table>
    </div>
  </section>

  <el-dialog v-model="createOpen" title="建立线路档案" width="min(620px, calc(100vw - 28px))">
    <el-form label-position="top"><div class="form-grid"><el-form-item label="线路编码"><el-input v-model="form.route_code" placeholder="METRO-A01" /></el-form-item><el-form-item label="线路名称"><el-input v-model="form.name" placeholder="城北环线 A 段" /></el-form-item><el-form-item label="线路长度（m）"><el-input-number v-model="form.length_m" :min="1" :max="500000" style="width:100%" /></el-form-item><el-form-item label="折射率"><el-input-number v-model="form.refractive_index" :min="1.3" :max="1.7" :precision="4" :step="0.001" style="width:100%" /></el-form-item><el-form-item label="发射端连接器"><el-input v-model="form.launch_connector" /></el-form-item><el-form-item label="线路状态"><el-select v-model="form.route_status" style="width:100%"><el-option label="在线" value="active" /><el-option label="维护" value="maintenance" /><el-option label="退役" value="retired" /></el-select></el-form-item></div></el-form>
    <template #footer><el-button @click="createOpen=false">取消</el-button><el-button type="primary" :loading="busy" :disabled="!form.route_code || !form.name" @click="create">建立档案</el-button></template>
  </el-dialog>

  <el-drawer v-model="detailOpen" :title="selected ? `${selected.route_code} / ${selected.name}` : '线路详情'" size="min(860px, 94vw)">
    <template #header><div class="drawer-title"><span>{{ selected ? `${selected.route_code} / ${selected.name}` : '线路详情' }}</span><el-button v-if="selected && auth.canAnalyze()" size="small" @click="openEdit"><Pencil :size="14" />编辑档案</el-button></div></template>
    <div v-if="selected" class="route-facts"><span><small>长度</small><strong>{{ selected.length_m.toLocaleString() }} m</strong></span><span><small>折射率</small><strong>{{ selected.refractive_index }}</strong></span><span><small>基线轨迹</small><strong>{{ selected.baseline_trace_id ? `#${selected.baseline_trace_id}` : '未设置' }}</strong></span><span><small>未关闭案例</small><strong>{{ pendingCases.length }} 件</strong></span></div>
    <el-tabs v-model="activeTab" class="detail-tabs">
      <el-tab-pane name="traces">
        <template #label><span class="tab-label"><ChartLine :size="14" />可切换轨迹（{{ history.length }}）</span></template>
        <TraceChart v-if="preview && selected" :points="preview.trace.processed_points.length ? preview.trace.processed_points : preview.trace.points" :sample-interval-ns="preview.trace.sample_interval_ns" :refractive-index="selected.refractive_index" :events="preview.events" :noise-floor="preview.trace.noise_floor_db" :height="300" />
        <h2 class="history-title">历史轨迹</h2>
        <div class="history-list"><button v-for="trace in history" :key="trace.id" :class="{ active: preview?.trace.id === trace.id }" @click="showTrace(trace.id)"><span><strong>#{{ trace.id }} · {{ trace.wavelength_nm }} nm</strong><small>{{ new Date(trace.captured_at).toLocaleString() }}</small></span><span class="history-actions"><i v-if="selected?.baseline_trace_id === trace.id">当前基线</i><el-button v-else-if="auth.canReview()" size="small" @click.stop="setBaseline(trace.id)">设为基线</el-button></span></button><div v-if="!history.length" class="empty-state"><span>该线路还没有轨迹记录。</span></div></div>
      </el-tab-pane>
      <el-tab-pane name="cases">
        <template #label><span class="tab-label"><ListChecks :size="14" />待处理案例（{{ pendingCases.length }}）</span></template>
        <el-table :data="pendingCases" row-key="id" size="small">
          <el-table-column prop="id" label="案例" width="90"><template #default="scope">#{{ scope.row.id }}</template></el-table-column>
          <el-table-column label="状态" width="110"><template #default="scope"><el-tag size="small" type="warning">{{ caseStatusLabel[scope.row.case_status as keyof typeof caseStatusLabel] ?? scope.row.case_status }}</el-tag></template></el-table-column>
          <el-table-column label="引用基线" width="110"><template #default="scope">#{{ scope.row.baseline_trace_id }}</template></el-table-column>
          <el-table-column label="当前轨迹" width="110"><template #default="scope">#{{ scope.row.current_trace_id }}</template></el-table-column>
          <el-table-column prop="conclusion" label="结论" min-width="200"><template #default="scope"><span>{{ scope.row.conclusion || '待分析/确认' }}</span></template></el-table-column>
          <el-table-column label="创建时间" width="170"><template #default="scope">{{ new Date(scope.row.created_at).toLocaleString() }}</template></el-table-column>
          <template #empty><div class="empty-state"><span>没有未关闭案例，可以更换基线。</span></div></template>
        </el-table>
      </el-tab-pane>
      <el-tab-pane name="rejections">
        <template #label><span class="tab-label"><Ban :size="14" />基线拒绝原因（{{ rejections.length }}）</span></template>
        <div class="rejection-list">
          <div v-for="item in rejections" :key="item.id" :class="['rejection-card', item.result]">
            <div class="rejection-head">
              <el-tag size="small" :type="item.result === 'succeeded' ? 'success' : 'danger'">{{ item.result === 'succeeded' ? '已生效' : '已拒绝' }}</el-tag>
              <strong v-if="item.result === 'rejected'">{{ rejectReasonLabel[item.reason as BaselineRejectReason] ?? item.reason }}</strong>
              <strong v-else>基线更换成功</strong>
              <span class="muted">{{ new Date(item.created_at).toLocaleString() }} · {{ item.actor_name }}</span>
            </div>
            <div class="rejection-body">
              <span>请求轨迹：#{{ item.requested_trace_id }}</span>
              <span v-if="item.current_trace_id">当时基线：#{{ item.current_trace_id }}</span>
              <span v-else>当时基线：未设置</span>
            </div>
            <div v-if="item.blocking_case_ids.length" class="blocking-cases">
              <Ban :size="13" />阻塞案例：
              <el-tag v-for="caseId in item.blocking_case_ids" :key="caseId" size="small" type="danger" effect="plain">#{{ caseId }}</el-tag>
              <span>请先在定位案例页确认并关闭后再更换基线。</span>
            </div>
          </div>
          <div v-if="!rejections.length" class="empty-state"><span>暂无基线更换记录。</span></div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </el-drawer>

  <el-dialog v-model="editOpen" title="编辑线路档案" width="min(620px, calc(100vw - 28px))">
    <el-form label-position="top"><div class="form-grid"><el-form-item label="线路名称"><el-input v-model="editForm.name" /></el-form-item><el-form-item label="线路长度（m）"><el-input-number v-model="editForm.length_m" :min="1" :max="500000" style="width:100%" /></el-form-item><el-form-item label="折射率"><el-input-number v-model="editForm.refractive_index" :min="1.3" :max="1.7" :precision="4" :step="0.001" style="width:100%" /></el-form-item><el-form-item label="发射端连接器"><el-input v-model="editForm.launch_connector" /></el-form-item><el-form-item label="线路状态"><el-select v-model="editForm.route_status" style="width:100%"><el-option label="在线" value="active" /><el-option label="维护" value="maintenance" /><el-option label="退役" value="retired" /></el-select></el-form-item></div></el-form>
    <template #footer><el-button @click="editOpen=false">取消</el-button><el-button type="primary" :loading="busy" :disabled="!editForm.name || !editForm.launch_connector" @click="update">保存变更</el-button></template>
  </el-dialog>
</template>

<style scoped>
.baseline { display:inline-flex; align-items:center; gap:5px; color:var(--accent); font-weight:700; }.muted { color:var(--text-muted); }.form-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:0 16px; }.route-facts { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); margin-bottom:16px; border:1px solid var(--line); }.route-facts span { padding:12px 14px; border-right:1px solid var(--line); }.route-facts span:last-child { border-right:0; }.route-facts small,.route-facts strong { display:block; }.route-facts small { color:var(--text-muted); font-size:11px; }.route-facts strong { margin-top:4px; font-size:15px; }.history-title { margin:24px 0 10px; font-size:15px; }.history-list { border-top:1px solid var(--line); }.history-list>button { width:100%; min-height:58px; display:flex; align-items:center; justify-content:space-between; gap:12px; padding:9px 12px; border:0; border-bottom:1px solid var(--line); background:transparent; color:var(--text); text-align:left; cursor:pointer; }.history-list>button:hover,.history-list>button.active { background:var(--surface-strong); }.history-list strong,.history-list small { display:block; }.history-list small { margin-top:3px; color:var(--text-muted); }.history-actions { display:flex; align-items:center; gap:8px; }.history-actions i { color:var(--accent); font-size:11px; font-style:normal; font-weight:800; }
.drawer-title { display:flex; align-items:center; justify-content:space-between; gap:12px; width:100%; }
.tab-label { display:inline-flex; align-items:center; gap:6px; }
.rejection-list { display:flex; flex-direction:column; gap:10px; }.rejection-card { border:1px solid var(--line); border-left:3px solid var(--danger,#e54848); padding:10px 12px; background:var(--surface); }.rejection-card.succeeded { border-left-color:var(--success,#67c23a); }.rejection-head { display:flex; align-items:center; gap:10px; flex-wrap:wrap; }.rejection-head .muted { margin-left:auto; font-size:11px; }.rejection-body { display:flex; gap:18px; margin-top:8px; font-size:12px; color:var(--text-muted); }.blocking-cases { display:flex; align-items:center; gap:6px; margin-top:8px; flex-wrap:wrap; font-size:12px; color:var(--danger,#e54848); }
@media(max-width:600px){.form-grid,.route-facts{grid-template-columns:1fr}.route-facts span{border-right:0;border-bottom:1px solid var(--line)}.route-facts span:last-child{border-bottom:0}}
</style>
