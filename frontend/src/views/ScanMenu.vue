<template>
  <div class="scan-menu">
    <!-- 错误状态 -->
    <div v-if="error" class="error-page">
      <div class="error-icon">!</div>
      <h2>{{ error }}</h2>
      <p>请确认二维码是否有效</p>
    </div>

    <!-- 加载中 -->
    <div v-else-if="loading" class="loading-page">
      <div class="loading-spinner"></div>
      <p>加载菜单中...</p>
    </div>

    <!-- 主内容 -->
    <template v-else>
      <!-- 顶部信息 -->
      <div class="header">
        <div class="table-info">
          <span class="table-badge">{{ tableInfo.table_no }}</span>
          <span class="table-capacity">{{ tableInfo.capacity }}人桌</span>
        </div>
        <div class="header-actions">
          <button class="orders-btn" @click="showOrders = true">
            <span>📋</span>
            <span v-if="tableOrders.length > 0" class="orders-badge">{{ tableOrders.length }}</span>
          </button>
          <button class="cart-btn" @click="showCart = true">
            <span class="cart-icon">🛒</span>
            <span v-if="cartCount > 0" class="cart-badge">{{ cartCount }}</span>
          </button>
        </div>
      </div>

      <!-- 分类导航 -->
      <div class="category-nav">
        <button
          class="category-btn"
          :class="{ active: selectedCategory === '' }"
          @click="selectedCategory = ''"
        >
          全部
        </button>
        <button
          v-for="cat in categories"
          :key="cat"
          class="category-btn"
          :class="{ active: selectedCategory === cat }"
          @click="selectedCategory = cat"
        >
          {{ cat }}
        </button>
      </div>

      <!-- 菜品列表 -->
      <div class="dishes-list">
        <div
          v-for="dish in filteredDishes"
          :key="dish.id"
          class="dish-item"
          :class="{ unavailable: dish.status !== 'available' }"
        >
          <div class="dish-image">
            <img v-if="dish.image" :src="dish.image" :alt="dish.name" />
            <div v-else class="dish-no-image">暂无图片</div>
            <span v-if="dish.status !== 'available'" class="dish-sold-out">
              {{ dish.status === 'sold_out' ? '售罄' : '下架' }}
            </span>
          </div>
          <div class="dish-content">
            <h3 class="dish-name">{{ dish.name }}</h3>
            <p v-if="dish.description" class="dish-desc">{{ dish.description }}</p>
            <div v-if="dish.dietary_tags" class="dish-tags">
              <span v-for="tag in dish.dietary_tags.split(',')" :key="tag" class="tag">{{ tag }}</span>
            </div>
            <div class="dish-footer">
              <span class="dish-price">¥{{ dish.price.toFixed(2) }}</span>
              <div v-if="dish.status === 'available'" class="dish-actions">
                <template v-if="getCartQuantity(dish.id) > 0">
                  <button class="qty-btn" @click="updateCart(dish, -1)">-</button>
                  <span class="qty-num">{{ getCartQuantity(dish.id) }}</span>
                </template>
                <button class="add-btn" @click="updateCart(dish, 1)">+</button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部下单栏 -->
      <div v-if="cartCount > 0" class="footer-bar">
        <div class="footer-info">
          <span class="footer-total">¥{{ cartTotal.toFixed(2) }}</span>
          <span class="footer-count">共 {{ cartCount }} 件</span>
        </div>
        <button class="submit-btn" @click="showCart = true">去下单</button>
      </div>

      <!-- 本桌订单侧边栏 -->
      <div v-if="showOrders" class="cart-overlay" @click.self="showOrders = false">
        <div class="cart-panel orders-panel">
          <div class="cart-header">
            <h3>本桌订单</h3>
            <button class="cart-close" @click="showOrders = false">&times;</button>
          </div>

          <div v-if="ordersLoading" class="cart-empty">
            <div class="loading-spinner"></div>
            <p>加载中...</p>
          </div>

          <div v-else-if="tableOrders.length === 0" class="cart-empty">
            暂无订单
          </div>

          <div v-else class="orders-content">
            <div v-for="order in tableOrders" :key="order.order_no" class="order-card">
              <div class="order-card-header">
                <span class="order-no">{{ order.order_no.slice(-8) }}</span>
                <span class="order-status" :class="order.status">{{ order.status_text }}</span>
              </div>
              <div v-if="order.customer_name" class="order-customer">
                {{ order.customer_name }}
              </div>
              <div class="order-items-list">
                <div v-for="item in order.items" :key="item.id" class="order-item-row">
                  <span>{{ item.dish_name }} × {{ item.quantity }}</span>
                  <span>¥{{ (item.price * item.quantity).toFixed(2) }}</span>
                </div>
              </div>
              <div class="order-card-footer">
                <span class="order-time">{{ formatTime(order.created_at) }}</span>
                <span class="order-total">¥{{ order.total_price.toFixed(2) }}</span>
              </div>
            </div>
          </div>

          <div class="cart-footer">
            <button class="refresh-btn" @click="fetchTableOrders" :disabled="ordersLoading">
              刷新订单
            </button>
          </div>
        </div>
      </div>

      <!-- 购物车侧边栏 -->
      <div v-if="showCart" class="cart-overlay" @click.self="showCart = false">
        <div class="cart-panel">
          <div class="cart-header">
            <h3>购物车</h3>
            <button class="cart-close" @click="showCart = false">&times;</button>
          </div>

          <div v-if="cartItems.length === 0" class="cart-empty">
            购物车是空的
          </div>

          <div v-else class="cart-content">
            <div class="cart-items">
              <div v-for="item in cartItems" :key="item.dish.id" class="cart-item">
                <div class="cart-item-info">
                  <span class="cart-item-name">{{ item.dish.name }}</span>
                  <span class="cart-item-price">¥{{ item.dish.price.toFixed(2) }}</span>
                </div>
                <div class="cart-item-actions">
                  <button class="qty-btn" @click="updateCart(item.dish, -1)">-</button>
                  <span class="qty-num">{{ item.quantity }}</span>
                  <button class="qty-btn" @click="updateCart(item.dish, 1)">+</button>
                </div>
                <input
                  v-model="item.remark"
                  type="text"
                  class="cart-item-remark"
                  placeholder="备注（如：少辣）"
                />
              </div>
            </div>

            <div class="cart-form">
              <div class="form-group">
                <label>您的称呼（选填）</label>
                <input v-model="customerName" type="text" placeholder="方便叫号" />
              </div>
              <div class="form-group">
                <label>订单备注（选填）</label>
                <textarea v-model="orderRemark" placeholder="其他要求..."></textarea>
              </div>
            </div>

            <div class="cart-footer">
              <div class="cart-total">
                <span>合计：</span>
                <span class="total-price">¥{{ cartTotal.toFixed(2) }}</span>
              </div>
              <button class="order-btn" :disabled="submitting" @click="submitOrder">
                {{ submitting ? '提交中...' : '确认下单' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { publicApi } from '../api'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref('')
const submitting = ref(false)
const showCart = ref(false)
const showOrders = ref(false)
const ordersLoading = ref(false)
const tableOrders = ref([])

const tableInfo = ref({ table_no: '', capacity: 0 })
const categories = ref([])
const dishes = ref([])
const selectedCategory = ref('')

const cart = ref([]) // [{ dish, quantity, remark }]
const customerName = ref('')
const orderRemark = ref('')

const token = computed(() => route.params.token)

const filteredDishes = computed(() => {
  if (!selectedCategory.value) return dishes.value
  return dishes.value.filter(d => d.category === selectedCategory.value)
})

const cartItems = computed(() => cart.value.filter(item => item.quantity > 0))

const cartCount = computed(() => {
  return cart.value.reduce((sum, item) => sum + item.quantity, 0)
})

const cartTotal = computed(() => {
  return cart.value.reduce((sum, item) => sum + item.dish.price * item.quantity, 0)
})

function getCartQuantity(dishId) {
  const item = cart.value.find(c => c.dish.id === dishId)
  return item ? item.quantity : 0
}

function updateCart(dish, delta) {
  const existingItem = cart.value.find(c => c.dish.id === dish.id)

  if (existingItem) {
    existingItem.quantity += delta
    if (existingItem.quantity <= 0) {
      cart.value = cart.value.filter(c => c.dish.id !== dish.id)
    }
  } else if (delta > 0) {
    // 检查库存
    if (dish.stock !== -1 && delta > dish.stock) {
      alert('库存不足')
      return
    }
    cart.value.push({ dish, quantity: 1, remark: '' })
  }
}

async function fetchMenu() {
  loading.value = true
  error.value = ''

  try {
    const res = await publicApi.getMenu(token.value)
    tableInfo.value = res.table
    categories.value = res.categories || []
    dishes.value = res.dishes || []
  } catch (err) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function fetchTableOrders() {
  ordersLoading.value = true
  try {
    const res = await publicApi.getTableOrders(token.value)
    tableOrders.value = res.orders || []
  } catch (err) {
    console.error('获取订单失败:', err)
  } finally {
    ordersLoading.value = false
  }
}

function formatTime(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

async function submitOrder() {
  if (cartItems.value.length === 0) {
    alert('购物车是空的')
    return
  }

  submitting.value = true

  try {
    const orderData = {
      customer_name: customerName.value,
      items: cartItems.value.map(item => ({
        dish_id: item.dish.id,
        quantity: item.quantity,
        remark: item.remark || ''
      })),
      remark: orderRemark.value
    }

    const res = await publicApi.createOrder(token.value, orderData)

    // 清空购物车
    cart.value = []
    customerName.value = ''
    orderRemark.value = ''
    showCart.value = false

    // 刷新订单列表
    fetchTableOrders()
    fetchMenu() // 刷新库存

    // 显示订单
    showOrders.value = true
  } catch (err) {
    alert(err.message || '下单失败')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchMenu()
  fetchTableOrders()
})
</script>

<style scoped>
.scan-menu {
  min-height: 100vh;
  background: #f5f5f5;
  padding-bottom: 80px;
}

.error-page,
.loading-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 20px;
  text-align: center;
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

.header {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.table-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.table-badge {
  padding: 6px 16px;
  background: #1890ff;
  color: #fff;
  border-radius: 20px;
  font-weight: 600;
}

.table-capacity {
  color: #666;
  font-size: 14px;
}

.cart-btn {
  position: relative;
  width: 44px;
  height: 44px;
  border: none;
  background: #f5f5f5;
  border-radius: 50%;
  font-size: 20px;
  cursor: pointer;
}

.cart-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  background: #ff4d4f;
  color: #fff;
  border-radius: 10px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.category-nav {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  background: #fff;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.category-nav::-webkit-scrollbar {
  display: none;
}

.category-btn {
  flex-shrink: 0;
  padding: 8px 16px;
  border: 1px solid #d9d9d9;
  background: #fff;
  border-radius: 20px;
  font-size: 14px;
  color: #666;
  cursor: pointer;
  transition: all 0.2s;
}

.category-btn.active {
  background: #1890ff;
  border-color: #1890ff;
  color: #fff;
}

.dishes-list {
  padding: 12px;
}

.dish-item {
  display: flex;
  gap: 12px;
  padding: 12px;
  margin-bottom: 12px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.05);
}

.dish-item.unavailable {
  opacity: 0.6;
}

.dish-image {
  position: relative;
  width: 100px;
  height: 100px;
  flex-shrink: 0;
  border-radius: 8px;
  overflow: hidden;
  background: #f5f5f5;
}

.dish-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.dish-no-image {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 12px;
}

.dish-sold-out {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  padding: 4px 12px;
  background: rgba(0,0,0,0.7);
  color: #fff;
  border-radius: 4px;
  font-size: 12px;
}

.dish-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.dish-name {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.dish-desc {
  margin: 0 0 4px;
  font-size: 12px;
  color: #999;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.dish-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}

.tag {
  padding: 2px 8px;
  background: #fff7e6;
  color: #fa8c16;
  border-radius: 4px;
  font-size: 11px;
}

.dish-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
}

.dish-price {
  font-size: 18px;
  font-weight: 600;
  color: #ff4d4f;
}

.dish-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.qty-btn,
.add-btn {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 50%;
  font-size: 18px;
  font-weight: bold;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qty-btn {
  background: #f5f5f5;
  color: #666;
}

.add-btn {
  background: #1890ff;
  color: #fff;
}

.qty-num {
  min-width: 20px;
  text-align: center;
  font-weight: 600;
}

.footer-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  box-shadow: 0 -2px 8px rgba(0,0,0,0.1);
  z-index: 100;
}

.footer-info {
  display: flex;
  flex-direction: column;
}

.footer-total {
  font-size: 20px;
  font-weight: 600;
  color: #ff4d4f;
}

.footer-count {
  font-size: 12px;
  color: #999;
}

.submit-btn {
  padding: 12px 32px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 24px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
}

/* 购物车侧边栏 */
.cart-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.5);
  z-index: 200;
}

.cart-panel {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  max-height: 80vh;
  background: #fff;
  border-radius: 16px 16px 0 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.cart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.cart-header h3 {
  margin: 0;
  font-size: 18px;
}

.cart-close {
  width: 32px;
  height: 32px;
  border: none;
  background: #f5f5f5;
  border-radius: 50%;
  font-size: 20px;
  cursor: pointer;
}

.cart-empty {
  padding: 60px 20px;
  text-align: center;
  color: #999;
}

.cart-content {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.cart-items {
  flex: 1;
  padding: 12px 16px;
  overflow-y: auto;
}

.cart-item {
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.cart-item-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.cart-item-name {
  font-weight: 500;
}

.cart-item-price {
  color: #ff4d4f;
}

.cart-item-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.cart-item-remark {
  width: 100%;
  padding: 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
}

.cart-form {
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
}

.cart-form .form-group {
  margin-bottom: 12px;
}

.cart-form label {
  display: block;
  margin-bottom: 4px;
  font-size: 13px;
  color: #666;
}

.cart-form input,
.cart-form textarea {
  width: 100%;
  padding: 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 14px;
}

.cart-form textarea {
  height: 60px;
  resize: none;
}

.cart-footer {
  padding: 16px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cart-total {
  font-size: 14px;
}

.total-price {
  font-size: 22px;
  font-weight: 600;
  color: #ff4d4f;
}

.order-btn {
  padding: 12px 32px;
  background: #ff4d4f;
  color: #fff;
  border: none;
  border-radius: 24px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
}

.order-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.orders-btn {
  position: relative;
  width: 44px;
  height: 44px;
  border: none;
  background: #f5f5f5;
  border-radius: 50%;
  font-size: 18px;
  cursor: pointer;
}

.orders-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  background: #1890ff;
  color: #fff;
  border-radius: 10px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.orders-panel {
  max-height: 85vh;
}

.orders-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
}

.order-card {
  background: #f9f9f9;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}

.order-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.order-no {
  font-size: 12px;
  color: #666;
  font-family: monospace;
}

.order-status {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  color: #fff;
}

.order-status.pending { background: #faad14; }
.order-status.confirmed { background: #1890ff; }
.order-status.preparing { background: #722ed1; }
.order-status.completed { background: #52c41a; }
.order-status.cancelled { background: #999; }

.order-customer {
  font-size: 13px;
  color: #666;
  margin-bottom: 8px;
}

.order-items-list {
  border-top: 1px dashed #e8e8e8;
  border-bottom: 1px dashed #e8e8e8;
  padding: 8px 0;
}

.order-item-row {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #333;
  padding: 4px 0;
}

.order-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}

.order-time {
  font-size: 12px;
  color: #999;
}

.order-total {
  font-size: 16px;
  font-weight: 600;
  color: #ff4d4f;
}

.refresh-btn {
  width: 100%;
  padding: 12px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 24px;
  font-size: 16px;
  cursor: pointer;
}

.refresh-btn:disabled {
  opacity: 0.6;
}
</style>
