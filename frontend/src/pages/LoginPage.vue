<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Activity, ArrowRight, ShieldCheck } from 'lucide-vue-next'
import { useAuth } from '@/hooks/useAuth'

const auth = useAuth(); const router = useRouter(); const route = useRoute(); const loading = ref(false)
const form = reactive({ username: 'analyst', password: 'DemoPass123!' })
async function submit() { loading.value = true; try { await auth.login(form.username, form.password); await router.push(String(route.query.redirect ?? '/routes')) } finally { loading.value = false } }
</script>

<template>
  <main class="login-page">
    <section class="login-instrument">
      <div class="instrument-rail"><Activity :size="24" /><span>FIBERSCOPE / OTDR</span><span class="rail-status"><i />READY</span></div>
      <div class="login-body">
        <div class="login-copy"><p>离线轨迹复核台</p><h1>光纤 OTDR<br />故障定位</h1><div class="trace-motif" aria-hidden="true"><span /><span /><span /><b /></div><small>定位结果仅供分析参考，必须由复核人员确认。</small></div>
        <form class="login-form" @submit.prevent="submit">
          <div class="form-head"><ShieldCheck :size="20" /><div><strong>分析席登录</strong><span>访问权限按角色动态加载</span></div></div>
          <label>用户名<el-input v-model="form.username" size="large" autocomplete="username" /></label>
          <label>密码<el-input v-model="form.password" size="large" type="password" show-password autocomplete="current-password" /></label>
          <button class="login-button" type="submit" :disabled="loading || !form.username || !form.password"><span>{{ loading ? '验证中…' : '进入复核台' }}</span><ArrowRight :size="18" /></button>
          <p class="demo-line">测试账号 analyst / reviewer / admin　密码 DemoPass123!</p>
        </form>
      </div>
    </section>
  </main>
</template>

<style scoped>
.login-page { min-height: 100vh; display: grid; place-items: center; padding: 24px; background: #e8eeeb; }
.login-instrument { width: min(1020px, 100%); min-height: 600px; border: 1px solid #aebcb6; background: #f9fbfa; box-shadow: 0 22px 64px rgba(31,51,43,.12); }
.instrument-rail { height: 54px; display: flex; align-items: center; gap: 10px; padding: 0 20px; color: #dcebe5; background: #22312c; font-size: 11px; font-weight: 800; }.rail-status { margin-left: auto; color: #a9c0b7; }.rail-status i { display: inline-block; width: 7px; height: 7px; margin-right: 7px; border-radius: 50%; background: #76c3aa; }
.login-body { min-height: 546px; display: grid; grid-template-columns: 1.1fr .9fr; }.login-copy { position: relative; overflow: hidden; padding: 68px clamp(34px, 7vw, 78px); color: #edf6f2; background: #2d433b; }.login-copy p { margin: 0 0 18px; color: #a9c6bb; font-size: 12px; font-weight: 800; }.login-copy h1 { margin: 0; font-size: clamp(38px, 6vw, 62px); line-height: 1.08; letter-spacing: 0; }.login-copy small { position: absolute; bottom: 42px; left: clamp(34px, 7vw, 78px); right: clamp(34px, 7vw, 78px); color: #b9ccc5; line-height: 1.7; }
.trace-motif { position: relative; height: 100px; margin-top: 58px; border-bottom: 1px solid #60756d; border-left: 1px solid #4c6259; border-right: 1px solid #4c6259; }.trace-motif span { position: absolute; left: 0; height: 2px; background: #79b9a3; }.trace-motif span:nth-child(1) { top: 22px; width: 22%; }.trace-motif span:nth-child(2) { top: 36px; left: 22%; width: 45%; transform: skewY(-3deg); }.trace-motif span:nth-child(3) { top: 68px; left: 66%; width: 34%; }.trace-motif b { position: absolute; left: 65%; top: 24px; width: 2px; height: 56px; background: #d39a34; }
.login-form { align-self: center; display: grid; gap: 22px; padding: 54px clamp(30px, 5vw, 62px); }.form-head { display: flex; align-items: center; gap: 11px; margin-bottom: 10px; color: #176c58; }.form-head strong, .form-head span { display: block; }.form-head strong { color: #26332e; font-size: 18px; }.form-head span { margin-top: 3px; color: #718079; font-size: 12px; }.login-form label { display: grid; gap: 7px; color: #56645e; font-size: 12px; font-weight: 700; }.login-button { min-height: 46px; display: flex; align-items: center; justify-content: space-between; padding: 0 16px; color: #f3faf7; background: #176c58; border: 1px solid #105543; cursor: pointer; font-weight: 800; }.login-button:hover { background: #105b4a; }.login-button:disabled { opacity: .6; cursor: wait; }.demo-line { margin: 0; color: #78847f; font-size: 11px; line-height: 1.6; }
@media (max-width: 740px) { .login-page { padding: 0; }.login-instrument { min-height: 100vh; border: 0; }.login-body { grid-template-columns: 1fr; }.login-copy { min-height: 310px; padding: 38px 28px; }.login-copy h1 { font-size: 40px; }.trace-motif { height: 60px; margin-top: 26px; }.login-copy small { position: static; display: block; margin-top: 28px; }.login-form { padding: 38px 28px; } }
</style>
