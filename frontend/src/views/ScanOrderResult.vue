<template>
  <div class="order-result">
    <!-- 加载中 -->
    <div v-if="loading" class="loading-page">
      <div class="loading-spinner"></div>
      <p>加载订单中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error-page">
      <div class="error-icon">!</div>
      <h2>{{ error }}</h2>
      <button class="btn-primary" @click="goBack">返回菜单</button>
    </div>

    <!-- 订单详情 -->
    <template v-else>
      <div class="result-header">
        <div class="status-icon" :class="order.status">
          <span v-if="order.status === 'pending'">⏳</span>
          <span v-else-if="order.status === 'confirmed'">✓</span>
          <span v-else-if="order.status === 'preparing'">🍳</span>
          <span v-else-if="order.status === 'completed'">✓</span>
          <span v-else-if="order.status === 'cancelled'">✗</span>
        </div>
        <h1 class="status-text">{{ order.status_text }}</h1>
        <p class="table-no">{{ order.table_no }}</p>
      </div>

      <div class="order-card">
        <div class="order-info">
          <div class="info-row">
            <span class="label">订单号</span>
            <span class="value">{{ order.order_no }}</span>
          </div>
          <div v-if="order.customer_name" class="info-row">
            <span class="label">顾客</span>
            <span class="value">{{ order.customer_name }}</span>
          </div>
          <div class="info-row">
            <span class="label">下单时间</span>
            <span class="value">{{ formatTime(order.created_at) }}</span>
          </div>
        </div>

        <div class="order-items">
          <h3>订单内容</h3>
          <div v-for="item in order.items" :key="item.id" class="order-item">
            <div class="item-main">
              <span class="item-name">{{ item.dish_name }}</span>
              <span class="item-qty">x{{ item.quantity }}</span>
              <span class="item-price">¥{{ (item.price * item.quantity).toFixed(2) }}</span>
            </div>
            <p v-if="item.remark" class="item-remark">备注：{{ item.remark }}</p>
          </div>
        </div>

        <div v-if="order.remark" class="order-remark">
          <h3>订单备注</h3>
          <p>{{ order.remark }}</p>
        </div>

        <div class="order-total">
          <span>合计</span>
          <span class="total-price">¥{{ order.total_price.toFixed(2) }}</span>
        </div>
      </div>

      <div class="actions">
        <button class="btn-outline" @click="refreshStatus">刷新状态</button>
        <button class="btn-primary" @click="goBack">继续点餐</button>
      </div>

      <p class="tip">订单状态会自动更新，请耐心等待</p>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { publicApi } from '../api'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref('')
const order = ref({
  order_no: '',
  table_no: '',
  total_price: 0,
  status: '',
  status_text: '',
  customer_name: '',
  remark: '',
  items: [],
  created_at: '',
  updated_at: ''
})

let refreshTimer = null

async function fetchOrder() {
  try {
    const res = await publicApi.getOrderStatus(route.params.token, route.params.orderNo)
    order.value = res
    error.value = ''
  } catch (err) {
    error.value = err.message || '订单不存在'
  } finally {
    loading.value = false
  }
}

function refreshStatus() {
  loading.value = true
  fetchOrder()
}

function formatTime(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  return `${month}-${day} ${hours}:${minutes}`
}

function goBack() {
  router.push({
    name: 'ScanMenu',
    params: { token: route.params.token }
  })
}

onMounted(() => {
  fetchOrder()
  // 每30秒自动刷新状态
  refreshTimer = setInterval(fetchOrder, 30000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.order-result {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px;
}

.loading-page,
.error-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 80vh;
  text-align: center;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #f3f3f3;
  border-top: 3px solid #1890ff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error-icon {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: #ff4d4f;
  color: #fff;
  font-size: 48px;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}

.result-header {
  text-align: center;
  padding: 40px 20px;
  background: #fff;
  border-radius: 12px;
  margin-bottom: 16px;
}

.status-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto 16px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 40px;
}

.status-icon.pending {
  background: #fff7e6;
}

.status-icon.confirmed {
  background: #e6f7ff;
}

.status-icon.preparing {
  background: #fff7e6;
}

.status-icon.completed {
  background: #e6f7e6;
}

.status-icon.cancelled {
  background: #fff1f0;
}

.status-text {
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 600;
}

.table-no {
  margin: 0;
  color: #666;
}

.order-card {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 16px;
}

.order-info {
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
}

.info-row .label {
  color: #999;
}

.info-row .value {
  color: #333;
  font-weight: 500;
}

.order-items {
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
}

.order-items h3 {
  margin: 0 0 12px;
  font-size: 16px;
}

.order-item {
  padding: 8px 0;
}

.item-main {
  display: flex;
  align-items: center;
}

.item-name {
  flex: 1;
}

.item-qty {
  color: #999;
  margin: 0 12px;
}

.item-price {
  font-weight: 500;
}

.item-remark {
  margin: 4px 0 0;
  font-size: 12px;
  color: #999;
}

.order-remark {
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
}

.order-remark h3 {
  margin: 0 0 8px;
  font-size: 16px;
}

.order-remark p {
  margin: 0;
  color: #666;
}

.order-total {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  font-size: 16px;
}

.total-price {
  font-size: 24px;
  font-weight: 600;
  color: #ff4d4f;
}

.actions {
  display: flex;
  gap: 12px;
}

.btn-outline,
.btn-primary {
  flex: 1;
  padding: 14px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
}

.btn-outline {
  background: #fff;
  border: 1px solid #d9d9d9;
  color: #333;
}

.btn-primary {
  background: #1890ff;
  border: none;
  color: #fff;
}

.tip {
  text-align: center;
  color: #999;
  font-size: 13px;
  margin-top: 16px;
}
</style>
