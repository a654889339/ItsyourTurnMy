<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-title">财务管理系统</div>
      <ul class="sidebar-menu">
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/' }"
          @click="router.push('/')"
        >
          首页概览
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/accounts' }"
          @click="router.push('/accounts')"
        >
          账户管理
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/transactions' }"
          @click="router.push('/transactions')"
        >
          收支记录
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/reports' }"
          @click="router.push('/reports')"
        >
          统计报表
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/dishes' }"
          @click="router.push('/dishes')"
        >
          菜品管理
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/menu' }"
          @click="router.push('/menu')"
        >
          点餐
        </li>
      </ul>
      <div style="padding: 20px; margin-top: auto; border-top: 1px solid rgba(255,255,255,0.1)">
        <div style="color: rgba(255,255,255,0.6); font-size: 12px; margin-bottom: 8px">
          当前用户: {{ authStore.user?.username }}
        </div>
        <button class="btn btn-default" style="width: 100%" @click="handleLogout">
          退出登录
        </button>
      </div>
    </aside>

    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>
