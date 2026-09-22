<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Activity, Play, Plus, Upload } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import TraceChart from '@/components/common/TraceChart.vue'
import EventTypeBadge from '@/components/common/EventTypeBadge.vue'
import { useTraceStore } from '@/stores/traces'
import { useRouteStore } from '@/stores/routes'
import { useAuth } from '@/hooks/useAuth'

const traces = useTraceStore(); const routes = useRouteStore(); const auth = useAuth(); const routeFilter = ref<number>(); const importOpen = ref(false); const busy = ref(false)
const form = reactive({ route_id: 0, wavelength_nm: 1550, pulse_width_ns: 100, sample_interval_ns: 100, captured_at: new Date().toISOString(), denoise_window: 5, peak_threshold_db: 0.8, merge_window: 3, points_text: '' })
const selectedRoute = computed(() => routes.items.find((item) => item.id === traces.selected?.trace.route_id))
function samplePoints() { const points = Array.from({ length: 240 }, (_, i) => Number((28 - i * .035 - (i >= 62 ? 2.8 : 0) - (i >= 154 ? 4.6 : 0) + Math.sin(i*.37)*.08).toFixed(3))); form.points_text = points.join(', '); form.captured_at = new Date().toISOString() }
async function filter() { await traces.fetch({ route_id: routeFilter.value, page_size: 100 }); if (traces.items[0]) await traces.select(traces.items[0].id) }
async function importTrace() { busy.value = true; try { const points = form.points_text.split(/[\s,;]+/).filter(Boolean).map(Number); await traces.importTrace({ ...form, points, captured_at: new Date(form.captured_at).toISOString() }); importOpen.value = false; ElMessage.success(`已导入 ${points.length} 个采样点`) } finally { busy.value = false } }
async function detect() { if (!traces.selected) return; busy.value = true; try { const result = await traces.detect(traces.selected.trace.id, { denoise_window: form.denoise_window, peak_threshold_db: form.peak_threshold_db, merge_window: form.merge_window }); ElMessage.success(`检出 ${result.detected_count} 个有效事件`) } finally { busy.value = false } }
onMounted(async () => { await routes.fetch({ page_size:100 }); if (routes.items[0]) form.route_id = routes.items[0].id; await filter() })
</script>

<template>
  <PageHeader title="轨迹分析" eyebrow="TRACE ANALYSIS" description="重放原始采样，调整阈值并检视事件证据。"><el-button v-if="auth.canAnalyze()" type="primary" @click="importOpen=true; samplePoints()"><Upload :size="16" />导入轨迹</el-button></PageHeader>
  <section class="content-band trace-workspace">
    <aside class="trace-index data-surface">
      <div class="index-head"><strong>轨迹列表</strong><el-select v-model="routeFilter" clearable placeholder="全部线路" size="small" @change="filter"><el-option v-for="route in routes.items" :key="route.id" :label="route.route_code" :value="route.id" /></el-select></div>
      <button v-for="trace in traces.items" :key="trace.id" :class="{ active: traces.selected?.trace.id === trace.id }" @click="traces.select(trace.id)"><span>#{{ trace.id }} · {{ trace.wavelength_nm }} nm</span><small>{{ routes.items.find(r=>r.id===trace.route_id)?.route_code ?? `Route ${trace.route_id}` }}<br>{{ new Date(trace.captured_at).toLocaleString() }}</small></button>
      <div v-if="!traces.items.length" class="empty-state"><div><Activity :size="28" /><strong>无轨迹</strong><span>请导入离线采样点。</span></div></div>
    </aside>
    <div class="analysis-stage">
      <template v-if="traces.selected">
        <div class="analysis-toolbar"><div><p>{{ selectedRoute?.route_code }} / TRACE #{{ traces.selected.trace.id }}</p><h2>{{ traces.selected.trace.wavelength_nm }} nm · {{ traces.selected.trace.points.length }} 采样点</h2></div><div class="analysis-actions"><span>噪声底 {{ traces.selected.trace.noise_floor_db.toFixed(2) }} dB</span><el-button v-if="auth.canAnalyze()" type="primary" :loading="busy" @click="detect"><Play :size="15" />执行事件检测</el-button></div></div>
        <TraceChart :points="traces.selected.trace.processed_points.length ? traces.selected.trace.processed_points : traces.selected.trace.points" :sample-interval-ns="traces.selected.trace.sample_interval_ns" :refractive-index="selectedRoute?.refractive_index" :events="traces.selected.events" :noise-floor="traces.selected.trace.noise_floor_db" />
        <div class="evidence data-surface"><div class="evidence-head"><div><p class="section-kicker">DETECTED EVIDENCE</p><h3 class="section-title">事件证据</h3></div><span>{{ traces.selected.events.length }} 项</span></div><el-table :data="traces.selected.events" size="small"><el-table-column label="类型" width="120"><template #default="scope"><EventTypeBadge :type="scope.row.event_type" :reviewed="scope.row.reviewed" /></template></el-table-column><el-table-column prop="distance_m" label="距离 m" width="110" /><el-table-column prop="insertion_loss_db" label="插入损耗 dB" width="130" /><el-table-column prop="reflectance_db" label="反射 dB" width="110" /><el-table-column label="置信度" min-width="140"><template #default="scope"><el-progress :percentage="Math.round(scope.row.confidence*100)" :stroke-width="6" /></template></el-table-column></el-table></div>
      </template>
      <div v-else class="empty-state data-surface"><div><Activity :size="36" /><strong>选择一条轨迹</strong><span>曲线、阈值与事件证据将在此展开。</span></div></div>
    </div>
  </section>
  <el-dialog v-model="importOpen" title="导入离线 OTDR 采样" width="min(720px, calc(100vw - 28px))">
    <el-form label-position="top"><div class="import-grid"><el-form-item label="线路"><el-select v-model="form.route_id" style="width:100%"><el-option v-for="route in routes.items" :key="route.id" :label="`${route.route_code} / ${route.name}`" :value="route.id" /></el-select></el-form-item><el-form-item label="波长"><el-select v-model="form.wavelength_nm" style="width:100%"><el-option v-for="value in [1310,1490,1550,1625]" :key="value" :label="`${value} nm`" :value="value" /></el-select></el-form-item><el-form-item label="脉宽 ns"><el-input-number v-model="form.pulse_width_ns" :min="1" style="width:100%" /></el-form-item><el-form-item label="采样间隔 ns"><el-input-number v-model="form.sample_interval_ns" :min="1" style="width:100%" /></el-form-item><el-form-item label="平滑窗口"><el-input-number v-model="form.denoise_window" :min="1" :max="31" style="width:100%" /></el-form-item><el-form-item label="峰值阈值 dB"><el-input-number v-model="form.peak_threshold_db" :min="0.1" :max="20" :step="0.1" style="width:100%" /></el-form-item></div><el-form-item label="采样点（逗号、空格或换行分隔）"><el-input v-model="form.points_text" type="textarea" :rows="7" /></el-form-item><div class="sample-row"><span>当前 {{ form.points_text.split(/[\s,;]+/).filter(Boolean).length }} 点</span><el-button text @click="samplePoints"><Plus :size="14" />重新生成可复现样例</el-button></div></el-form>
    <template #footer><el-button @click="importOpen=false">取消</el-button><el-button type="primary" :loading="busy" :disabled="!form.route_id || form.points_text.length < 30" @click="importTrace">导入并保存参数</el-button></template>
  </el-dialog>
</template>

<style scoped>
.trace-workspace{display:grid;grid-template-columns:250px minmax(0,1fr);gap:16px}.trace-index{align-self:start;max-height:720px;overflow:auto}.index-head{position:sticky;top:0;z-index:2;display:grid;gap:10px;padding:13px;background:var(--surface);border-bottom:1px solid var(--line)}.trace-index>button{width:100%;display:block;padding:12px 13px;color:var(--text);background:transparent;border:0;border-bottom:1px solid var(--line);text-align:left;cursor:pointer}.trace-index>button:hover,.trace-index>button.active{background:var(--surface-strong);box-shadow:inset 3px 0 var(--accent)}.trace-index>button span,.trace-index>button small{display:block}.trace-index>button span{font-weight:800}.trace-index>button small{margin-top:5px;color:var(--text-muted);line-height:1.5}.analysis-stage{min-width:0}.analysis-toolbar{min-height:64px;display:flex;align-items:center;justify-content:space-between;gap:15px;padding:0 2px 12px}.analysis-toolbar p,.analysis-toolbar h2{margin:0}.analysis-toolbar p{color:var(--accent);font-size:11px;font-weight:800}.analysis-toolbar h2{margin-top:4px;font-size:17px}.analysis-actions{display:flex;align-items:center;gap:12px;color:var(--text-muted);font-size:12px}.evidence{margin-top:16px}.evidence-head{height:60px;display:flex;align-items:center;justify-content:space-between;padding:0 14px;border-bottom:1px solid var(--line)}.evidence-head p,.evidence-head h3{margin:0}.evidence-head>span{color:var(--text-muted);font-size:12px}.import-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 14px}.sample-row{display:flex;justify-content:space-between;align-items:center;color:var(--text-muted);font-size:12px}
@media(max-width:900px){.trace-workspace{grid-template-columns:1fr}.trace-index{display:flex;max-height:none;overflow-x:auto}.index-head{position:sticky;left:0;min-width:150px}.trace-index>button{min-width:190px;border-right:1px solid var(--line)}}@media(max-width:600px){.analysis-toolbar{align-items:flex-start;flex-direction:column}.analysis-actions{width:100%;justify-content:space-between}.import-grid{grid-template-columns:1fr}}
</style>
