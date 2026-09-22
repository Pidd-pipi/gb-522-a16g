<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { RotateCcw, ZoomIn, ZoomOut } from 'lucide-vue-next'
import { eventLabel, type EventMarker } from '@/types/event'
import { useTraceViewport } from '@/hooks/useTraceViewport'

const props = withDefaults(defineProps<{
  points: number[]
  sampleIntervalNs: number
  refractiveIndex?: number
  events?: EventMarker[]
  noiseFloor?: number
  height?: number
}>(), { refractiveIndex: 1.468, events: () => [], noiseFloor: 0, height: 360 })

const target = ref<HTMLDivElement>()
const viewport = useTraceViewport(props.points.length)
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | undefined

function distanceAt(index: number) {
  return Number((299792458 * index * props.sampleIntervalNs * 1e-9 / (2 * props.refractiveIndex)).toFixed(2))
}

function chartData(): [number, number][] {
  return props.points.map((value, index) => [distanceAt(index), value])
}

function eventMarks(data: [number, number][]) {
  return props.events.map((event) => {
    let closest = data[0] ?? [0, 0]
    for (const point of data) {
      if (Math.abs(point[0] - event.distance_m) < Math.abs(closest[0] - event.distance_m)) closest = point
    }
    return {
      coord: closest,
      value: event.distance_m,
      shortLabel: event.event_type === 'break' ? '断' : '事',
      name: `${eventLabel[event.event_type]} ${event.distance_m.toFixed(1)}m`,
      itemStyle: { color: event.event_type === 'break' ? '#a43a32' : '#a66d0b' },
    }
  })
}

function makeOption(): echarts.EChartsOption {
  const data = chartData()
  return {
    animationDuration: 420,
    grid: { left: 62, right: 28, top: 26, bottom: 72 },
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    xAxis: { type: 'value', name: '距离 / m', nameLocation: 'middle', nameGap: 34, axisLine: { lineStyle: { color: '#899690' } }, splitLine: { lineStyle: { color: '#e5eae7' } } },
    yAxis: { type: 'value', name: '回波 / dB', nameGap: 18, axisLine: { show: true, lineStyle: { color: '#899690' } }, splitLine: { lineStyle: { color: '#e5eae7' } } },
    dataZoom: [
      { type: 'inside', start: viewport.start.value, end: viewport.end.value },
      { type: 'slider', height: 22, bottom: 12, borderColor: '#b9c4bf', fillerColor: 'rgba(27,109,89,.18)', handleStyle: { color: '#176c58' } },
    ],
    series: [{
      name: 'OTDR 轨迹', type: 'line', data, showSymbol: false, sampling: 'lttb',
      lineStyle: { color: '#176c58', width: 2 }, itemStyle: { color: '#176c58' },
      markLine: { silent: true, symbol: 'none', label: { formatter: '噪声底 {c} dB', color: '#68746f' }, lineStyle: { color: '#8a9691', type: 'dashed' }, data: [{ yAxis: props.noiseFloor }] },
      markPoint: { symbol: 'pin', symbolSize: 42, label: { formatter: (params: any) => params.data.shortLabel, color: '#f8fbf9', fontSize: 10 }, data: eventMarks(data) },
    }],
  }
}

function render() {
  if (!target.value) return
  if (!chart) {
    chart = echarts.init(target.value)
    chart.on('datazoom', () => {
      const state = (chart?.getOption().dataZoom as any[])?.[0]
      viewport.update(Number(state?.start ?? 0), Number(state?.end ?? 100))
    })
  }
  chart.setOption(makeOption(), true)
}

function zoom(delta: number) {
  const center = (viewport.start.value + viewport.end.value) / 2
  const span = Math.max(5, Math.min(100, viewport.end.value - viewport.start.value + delta))
  viewport.update(center - span / 2, center + span / 2)
  chart?.dispatchAction({ type: 'dataZoom', start: viewport.start.value, end: viewport.end.value })
}
function reset() { viewport.reset(); chart?.dispatchAction({ type: 'dataZoom', start: 0, end: 100 }) }

onMounted(async () => {
  await nextTick(); render()
  observer = new ResizeObserver(() => chart?.resize())
  if (target.value) observer.observe(target.value)
})
onBeforeUnmount(() => { observer?.disconnect(); chart?.dispose() })
watch(() => [props.points, props.events, props.noiseFloor], render, { deep: true })
</script>

<template>
  <section class="trace-panel" aria-label="OTDR 轨迹图">
    <div class="chart-toolbar">
      <div><span class="signal-key" />OTDR 采样 <strong>{{ points.length }}</strong> 点</div>
      <div class="chart-actions">
        <el-tooltip content="放大轨迹"><button class="icon-button" aria-label="放大轨迹" @click="zoom(-18)"><ZoomIn :size="17" /></button></el-tooltip>
        <el-tooltip content="缩小轨迹"><button class="icon-button" aria-label="缩小轨迹" @click="zoom(18)"><ZoomOut :size="17" /></button></el-tooltip>
        <el-tooltip content="重置视图"><button class="icon-button" aria-label="重置视图" @click="reset"><RotateCcw :size="17" /></button></el-tooltip>
      </div>
    </div>
    <div ref="target" class="chart" :style="{ height: `${height}px` }" />
  </section>
</template>

<style scoped>
.trace-panel { min-width: 0; border: 1px solid var(--line); background: var(--surface); }
.chart-toolbar { height: 46px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 0 14px; border-bottom: 1px solid var(--line); color: var(--text-muted); font-size: 13px; }
.signal-key { display: inline-block; width: 18px; height: 2px; margin: 0 8px 3px 0; background: #176c58; }
.chart-actions { display: flex; gap: 3px; }
.icon-button { width: 32px; height: 32px; display: grid; place-items: center; border: 1px solid transparent; background: transparent; color: var(--text-muted); cursor: pointer; }
.icon-button:hover, .icon-button:focus-visible { border-color: var(--line-strong); color: var(--accent); background: var(--surface-strong); outline: none; }
.chart { width: 100%; min-height: 280px; }
</style>
