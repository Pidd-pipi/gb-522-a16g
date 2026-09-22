<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { Search, ScrollText } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import { useAuditStore } from '@/stores/audit'
import { useRouteStore } from '@/stores/routes'

const audit = useAuditStore(); const routes = useRouteStore(); const filters = reactive({route_id:undefined as number|undefined,actor:'',action:''})
const actionLabels:Record<string,string>={'route.created':'新建线路','route.updated':'线路变更','route.baseline_changed':'基线变更','trace.imported':'轨迹导入','trace.events_detected':'事件检测','event.reviewed':'事件复核','case.created':'新建案例','case.analysis_started':'分析启动','case.analysis_completed':'分析完成','case.analysis_failed':'分析失败','case.confirmed':'案例确认','case.closed':'案例关闭'}
async function search(){await audit.fetch({...filters,page_size:100})}
function summary(value:string){try{const parsed=JSON.parse(value);const text=JSON.stringify(parsed);return text.length>110?text.slice(0,110)+'…':text}catch{return value}}
onMounted(async()=>{await Promise.all([routes.fetch({page_size:100}),search()])})
</script>

<template>
  <PageHeader title="审计检索" eyebrow="IMMUTABLE AUDIT" description="按线路、操作者与时间回溯关键变更，关联每个 request ID。" />
  <section class="content-band"><div class="audit-filters"><el-select v-model="filters.route_id" clearable placeholder="全部线路"><el-option v-for="route in routes.items" :key="route.id" :label="route.route_code" :value="route.id" /></el-select><el-input v-model="filters.actor" clearable placeholder="操作者" /><el-select v-model="filters.action" clearable placeholder="全部动作"><el-option v-for="(label,value) in actionLabels" :key="value" :label="label" :value="value" /></el-select><el-button type="primary" @click="search"><Search :size="15" />检索</el-button><span class="toolbar-spacer subtle-count">{{ audit.total }} 条记录</span></div>
    <div class="data-surface"><el-table v-loading="audit.loading" :data="audit.items" row-key="id"><el-table-column prop="created_at" label="时间" width="170"><template #default="scope">{{ new Date(scope.row.created_at).toLocaleString() }}</template></el-table-column><el-table-column prop="actor_name" label="操作者" width="110"><template #default="scope"><strong>{{ scope.row.actor_name }}</strong></template></el-table-column><el-table-column prop="action" label="动作" width="140"><template #default="scope"><span class="audit-action">{{ actionLabels[scope.row.action] ?? scope.row.action }}</span></template></el-table-column><el-table-column label="资源" width="150"><template #default="scope">{{ scope.row.resource_type }} #{{ scope.row.resource_id }}</template></el-table-column><el-table-column label="前值摘要" min-width="190"><template #default="scope"><code>{{ summary(scope.row.before) }}</code></template></el-table-column><el-table-column label="后值摘要" min-width="190"><template #default="scope"><code>{{ summary(scope.row.after) }}</code></template></el-table-column><el-table-column prop="request_id" label="Request ID" width="205"><template #default="scope"><span class="request-id">{{ scope.row.request_id }}</span></template></el-table-column><template #empty><div class="empty-state"><div><ScrollText :size="34" /><strong>没有审计记录</strong><span>放宽检索条件后重试。</span></div></div></template></el-table></div>
  </section>
</template>

<style scoped>
.audit-filters{display:flex;align-items:center;flex-wrap:wrap;gap:9px;margin-bottom:14px}.audit-filters .el-select,.audit-filters .el-input{width:170px}.audit-action{color:var(--accent);font-size:12px;font-weight:800}.request-id{color:var(--text-muted);font-family:"SFMono-Regular",Consolas,monospace;font-size:11px}code{display:block;overflow:hidden;color:#596660;font-family:"SFMono-Regular",Consolas,monospace;font-size:11px;white-space:nowrap;text-overflow:ellipsis}@media(max-width:620px){.audit-filters .el-select,.audit-filters .el-input{width:calc(50% - 5px)}}
</style>
