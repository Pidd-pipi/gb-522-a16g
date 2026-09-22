<script setup lang="ts">
import { reactive, watch } from 'vue'
import { EVENT_TYPES, eventLabel, type EventMarker, type EventType } from '@/types/event'
import type { LocalizationCase } from '@/types/case'

const props = defineProps<{ modelValue: boolean; mode: 'event' | 'case'; event?: EventMarker | null; caseItem?: LocalizationCase | null; loading?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; submit: [value: Record<string, unknown>] }>()
const form = reactive({ event_type: 'unknown' as EventType, distance_m: 0, review_note: '', conclusion: '', estimated_distance_m: 0, uncertainty_m: 10 })
watch(() => [props.modelValue, props.event, props.caseItem], () => {
  if (props.event) { form.event_type = props.event.event_type; form.distance_m = props.event.distance_m; form.review_note = props.event.review_note ?? '' }
  if (props.caseItem) { form.conclusion = props.caseItem.conclusion ?? ''; form.estimated_distance_m = props.caseItem.estimated_distance_m ?? 0; form.uncertainty_m = props.caseItem.uncertainty_m ?? 10 }
}, { immediate: true })
function submit() {
  if (props.mode === 'event') emit('submit', { event_type: form.event_type, distance_m: form.distance_m, review_note: form.review_note })
  else emit('submit', { conclusion: form.conclusion, estimated_distance_m: form.estimated_distance_m, uncertainty_m: form.uncertainty_m, version: props.caseItem?.version })
}
</script>

<template>
  <el-dialog :model-value="modelValue" :title="mode === 'event' ? '事件人工复核' : '定位结论确认'" width="min(560px, calc(100vw - 28px))" destroy-on-close @update:model-value="emit('update:modelValue', $event)">
    <el-alert v-if="mode === 'event' && event" type="info" :closable="false" show-icon>
      <template #title>算法原值：{{ eventLabel[event.algorithm_event_type] }} · {{ event.algorithm_distance_m.toFixed(2) }} m</template>
    </el-alert>
    <el-alert v-if="mode === 'case'" type="warning" :closable="false" show-icon title="确认后结论将进入不可直接编辑的已确认状态。" />
    <el-form label-position="top" class="review-form">
      <template v-if="mode === 'event'">
        <el-form-item label="修订类型"><el-select v-model="form.event_type" style="width: 100%"><el-option v-for="type in EVENT_TYPES" :key="type" :label="eventLabel[type]" :value="type" /></el-select></el-form-item>
        <el-form-item label="修订距离（m）"><el-input-number v-model="form.distance_m" :min="0" :precision="2" controls-position="right" style="width: 100%" /></el-form-item>
        <el-form-item label="复核备注"><el-input v-model="form.review_note" type="textarea" :rows="4" maxlength="1000" show-word-limit /></el-form-item>
      </template>
      <template v-else>
        <el-form-item label="人工结论"><el-input v-model="form.conclusion" type="textarea" :rows="5" minlength="10" maxlength="2000" show-word-limit /></el-form-item>
        <div class="two-columns"><el-form-item label="估算距离（m）"><el-input-number v-model="form.estimated_distance_m" :min="0" :precision="2" controls-position="right" /></el-form-item><el-form-item label="不确定度（m）"><el-input-number v-model="form.uncertainty_m" :min="0" :max="5000" :precision="2" controls-position="right" /></el-form-item></div>
      </template>
    </el-form>
    <template #footer><el-button @click="emit('update:modelValue', false)">取消</el-button><el-button type="primary" :loading="loading" :disabled="mode === 'event' ? form.review_note.trim().length < 3 : form.conclusion.trim().length < 10" @click="submit">提交复核</el-button></template>
  </el-dialog>
</template>

<style scoped>
.review-form { margin-top: 20px; }
.two-columns { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
@media (max-width: 520px) { .two-columns { grid-template-columns: 1fr; gap: 0; } }
</style>
