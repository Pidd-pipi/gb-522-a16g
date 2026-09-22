<script setup lang="ts">
defineProps<{ title: string; eyebrow: string; description: string }>()
</script>

<template>
  <header class="page-header">
    <div><p>{{ eyebrow }}</p><h1>{{ title }}</h1><span>{{ description }}</span></div>
    <div class="header-actions"><slot /></div>
  </header>
</template>

<style scoped>
.page-header { display: flex; align-items: flex-end; justify-content: space-between; gap: 22px; padding: 26px 0 20px; border-bottom: 1px solid var(--line); }
.page-header p { margin: 0 0 5px; color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: 0; text-transform: uppercase; }
.page-header h1 { margin: 0; color: var(--text); font-size: clamp(24px, 3vw, 34px); line-height: 1.15; letter-spacing: 0; }
.page-header span { display: block; margin-top: 7px; color: var(--text-muted); font-size: 14px; }
.header-actions { flex: 0 0 auto; display: flex; gap: 8px; }
@media (max-width: 700px) { .page-header { align-items: stretch; flex-direction: column; } .header-actions :deep(.el-button) { flex: 1; } }
</style>
