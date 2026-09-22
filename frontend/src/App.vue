<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Activity, Cable, ClipboardCheck, LogOut, Menu, Radar, ScrollText, X } from 'lucide-vue-next'
import { useAuth } from '@/hooks/useAuth'

const route = useRoute(); const router = useRouter(); const auth = useAuth(); const navOpen = ref(false)
const isLogin = computed(() => route.name === 'login')
const nav = computed(() => [
  { to: '/routes', label: '线路档案', icon: Cable, show: true },
  { to: '/traces', label: '轨迹分析', icon: Activity, show: true },
  { to: '/events', label: '事件复核', icon: Radar, show: true },
  { to: '/cases', label: '定位案例', icon: ClipboardCheck, show: true },
  { to: '/audit', label: '审计检索', icon: ScrollText, show: auth.canReview() },
].filter((item) => item.show))
function leave() { auth.logout(); router.push('/login') }
</script>

<template>
  <router-view v-if="isLogin" />
  <div v-else class="app-frame">
    <header class="mobile-bar"><button class="plain-icon" :aria-label="navOpen ? '关闭导航' : '打开导航'" @click="navOpen = !navOpen"><X v-if="navOpen" /><Menu v-else /></button><strong>FiberScope</strong><span class="live-dot">离线分析</span></header>
    <aside class="sidebar" :class="{ open: navOpen }">
      <div class="brand"><div class="brand-mark"><Activity :size="22" /></div><div><strong>FiberScope</strong><span>OTDR REVIEW DESK</span></div></div>
      <nav aria-label="主导航"><router-link v-for="item in nav" :key="item.to" :to="item.to" @click="navOpen = false"><component :is="item.icon" :size="18" /><span>{{ item.label }}</span></router-link></nav>
      <div class="system-boundary"><span class="boundary-light" />离线分析模式</div>
      <div class="account"><div class="avatar">{{ auth.user.value?.display_name.slice(0, 1) }}</div><div><strong>{{ auth.user.value?.display_name }}</strong><span>{{ auth.user.value?.role }}</span></div><el-tooltip content="退出登录"><button class="plain-icon" aria-label="退出登录" @click="leave"><LogOut :size="17" /></button></el-tooltip></div>
    </aside>
    <main class="main-content"><router-view /></main>
  </div>
</template>
