<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-title">99点单</div>
      <ul class="sidebar-menu">
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/' }"
          @click="router.push('/')"
        >
          菜品管理
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/tables' }"
          @click="router.push('/tables')"
        >
          餐桌管理
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/menu' }"
          @click="router.push('/menu')"
        >
          点餐
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/orders' }"
          @click="router.push('/orders')"
        >
          订单管理
        </li>
        <li
          class="sidebar-menu-item"
          :class="{ active: route.path === '/dish-reports' }"
          @click="router.push('/dish-reports')"
        >
          销售报表
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
