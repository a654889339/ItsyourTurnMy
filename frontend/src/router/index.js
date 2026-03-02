import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../store/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue')
  },
  // 扫码点单页面（无需登录）
  {
    path: '/scan/:token',
    name: 'ScanMenu',
    component: () => import('../views/ScanMenu.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/scan/:token/order/:orderNo',
    name: 'ScanOrderResult',
    component: () => import('../views/ScanOrderResult.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dishes',
        component: () => import('../views/Dishes.vue')
      },
      {
        path: 'menu',
        name: 'Menu',
        component: () => import('../views/Menu.vue')
      },
      {
        path: 'dish-reports',
        name: 'DishReports',
        component: () => import('../views/DishReports.vue')
      },
      {
        path: 'tables',
        name: 'Tables',
        component: () => import('../views/Tables.vue')
      },
      {
        path: 'orders',
        name: 'OrderManagement',
        component: () => import('../views/OrderManagement.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()

  // 扫码页面不需要登录
  if (to.meta.requiresAuth === false) {
    next()
  } else if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/')
  } else {
    next()
  }
})

export default router
