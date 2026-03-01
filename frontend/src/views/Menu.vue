<template>
  <div class="menu-page">
    <div class="page-header">
      <h1 class="page-title">点餐</h1>
      <div style="display: flex; gap: 12px;">
        <button class="btn btn-default" @click="showOrders = true">
          我的订单 ({{ orders.length }})
        </button>
        <button class="btn btn-primary" :disabled="cart.length === 0" @click="submitOrder">
          结算 ({{ cartTotal.toFixed(2) }})
        </button>
      </div>
    </div>

    <div v-if="errorMsg" class="message message-error">{{ errorMsg }}</div>
    <div v-if="successMsg" class="message message-success">{{ successMsg }}</div>

    <div class="menu-container">
      <!-- 左侧分类 -->
      <div class="menu-categories">
        <div
          v-for="cat in ['全部', ...categories]"
          :key="cat"
          class="category-item"
          :class="{ active: selectedCategory === cat }"
          @click="selectCategory(cat)"
        >
          {{ cat }}
        </div>
      </div>

      <!-- 中间菜品列表 -->
      <div class="menu-dishes">
        <div v-if="loading" class="loading">加载中...</div>
        <div v-else-if="filteredDishes.length === 0" class="empty-state">
          暂无可点菜品
        </div>
        <div v-else class="menu-grid">
          <div
            v-for="dish in filteredDishes"
            :key="dish.id"
            class="menu-dish-card"
            :class="{ disabled: dish.status !== 'available' || (dish.stock !== -1 && dish.stock <= 0) }"
          >
            <div class="menu-dish-image">
              <img v-if="dish.image" :src="dish.image" :alt="dish.name" />
              <div v-else class="menu-dish-placeholder">暂无图片</div>
            </div>
            <div class="menu-dish-info">
              <h4 class="menu-dish-name">{{ dish.name }}</h4>
              <p v-if="dish.description" class="menu-dish-desc">{{ dish.description }}</p>
              <div v-if="dish.dietary_tags" class="menu-dish-tags">
                <span v-for="tag in dish.dietary_tags.split(',')" :key="tag" class="menu-dish-tag">{{ tag }}</span>
              </div>
              <div class="menu-dish-bottom">
                <span class="menu-dish-price">¥{{ formatMoney(dish.price) }}</span>
                <span v-if="dish.stock !== -1" class="menu-dish-stock">剩余{{ dish.stock }}</span>
              </div>
            </div>
            <div class="menu-dish-actions">
              <template v-if="dish.status === 'available' && (dish.stock === -1 || dish.stock > 0)">
                <div v-if="getCartQuantity(dish.id) > 0" class="quantity-control">
                  <button class="quantity-btn" @click="decreaseCart(dish)">-</button>
                  <span class="quantity-value">{{ getCartQuantity(dish.id) }}</span>
                  <button class="quantity-btn" @click="addToCart(dish)" :disabled="dish.stock !== -1 && getCartQuantity(dish.id) >= dish.stock">+</button>
                </div>
                <button v-else class="btn btn-primary btn-sm" @click="addToCart(dish)">加入</button>
              </template>
              <span v-else class="sold-out-text">
                {{ dish.status === 'sold_out' ? '售罄' : dish.status === 'disabled' ? '下架' : '库存不足' }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧购物车 -->
      <div class="menu-cart">
        <h3 class="cart-title">购物车</h3>
        <div v-if="cart.length === 0" class="cart-empty">
          还没有添加菜品
        </div>
        <div v-else class="cart-items">
          <div v-for="item in cart" :key="item.dish.id" class="cart-item">
            <div class="cart-item-info">
              <span class="cart-item-name">{{ item.dish.name }}</span>
              <span class="cart-item-price">¥{{ formatMoney(item.dish.price * item.quantity) }}</span>
            </div>
            <div class="cart-item-quantity">
              <button class="quantity-btn small" @click="decreaseCart(item.dish)">-</button>
              <span>{{ item.quantity }}</span>
              <button class="quantity-btn small" @click="addToCart(item.dish)" :disabled="item.dish.stock !== -1 && item.quantity >= item.dish.stock">+</button>
            </div>
          </div>
        </div>
        <div v-if="cart.length > 0" class="cart-footer">
          <div class="cart-remark">
            <input v-model="orderRemark" type="text" class="form-input" placeholder="订单备注..." />
          </div>
          <div class="cart-total">
            <span>合计：</span>
            <span class="cart-total-price">¥{{ cartTotal.toFixed(2) }}</span>
          </div>
          <button class="btn btn-primary" style="width: 100%;" @click="submitOrder" :disabled="submitting">
            {{ submitting ? '提交中...' : '确认下单' }}
          </button>
          <button class="btn btn-default" style="width: 100%; margin-top: 8px;" @click="clearCart">
            清空购物车
          </button>
        </div>
      </div>
    </div>

    <!-- 订单列表弹窗 -->
    <div v-if="showOrders" class="modal-overlay" @click.self="showOrders = false">
      <div class="modal" style="max-width: 800px; max-height: 80vh; overflow-y: auto;">
        <div class="modal-header">
          <h3 class="modal-title">我的订单</h3>
          <button class="modal-close" @click="showOrders = false">&times;</button>
        </div>

        <div style="padding: 20px;">
          <div style="margin-bottom: 16px;">
            <select v-model="orderFilter" class="form-select" style="width: 150px;" @change="fetchOrders">
              <option value="">全部状态</option>
              <option value="pending">待确认</option>
              <option value="confirmed">已确认</option>
              <option value="preparing">制作中</option>
              <option value="completed">已完成</option>
              <option value="cancelled">已取消</option>
            </select>
          </div>

          <div v-if="ordersLoading" class="loading">加载中...</div>
          <div v-else-if="orders.length === 0" class="empty-state">
            暂无订单
          </div>
          <div v-else class="orders-list">
            <div v-for="order in orders" :key="order.id" class="order-card">
              <div class="order-header">
                <span class="order-no">{{ order.order_no }}</span>
                <span class="order-status" :class="order.status">{{ getOrderStatusName(order.status) }}</span>
              </div>
              <div class="order-items">
                <div v-for="item in order.items" :key="item.id" class="order-item">
                  <span>{{ item.dish_name }} x {{ item.quantity }}</span>
                  <span>¥{{ formatMoney(item.price * item.quantity) }}</span>
                </div>
              </div>
              <div class="order-footer">
                <span class="order-time">{{ formatDateTime(order.created_at) }}</span>
                <span class="order-total">合计：¥{{ formatMoney(order.total_price) }}</span>
              </div>
              <div v-if="order.status === 'pending'" class="order-actions">
                <button class="btn btn-danger btn-sm" @click="cancelOrder(order)">取消订单</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import api from '../api'

const loading = ref(false)
const submitting = ref(false)
const ordersLoading = ref(false)
const showOrders = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const dishes = ref([])
const categories = ref([])
const selectedCategory = ref('全部')
const cart = ref([])
const orderRemark = ref('')
const orders = ref([])
const orderFilter = ref('')

const orderStatusNames = {
  pending: '待确认',
  confirmed: '已确认',
  preparing: '制作中',
  completed: '已完成',
  cancelled: '已取消'
}

function getOrderStatusName(status) {
  return orderStatusNames[status] || status
}

function formatMoney(value) {
  return Number(value).toFixed(2)
}

function formatDateTime(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

const filteredDishes = computed(() => {
  if (selectedCategory.value === '全部') {
    return dishes.value
  }
  return dishes.value.filter(d => d.category === selectedCategory.value)
})

const cartTotal = computed(() => {
  return cart.value.reduce((sum, item) => sum + item.dish.price * item.quantity, 0)
})

function getCartQuantity(dishId) {
  const item = cart.value.find(i => i.dish.id === dishId)
  return item ? item.quantity : 0
}

async function fetchDishes() {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await api.getDishes({ status: 'available', page_size: 100 })
    dishes.value = res.dishes || []
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    loading.value = false
  }
}

async function fetchCategories() {
  try {
    const res = await api.getDishCategories()
    categories.value = res.categories || []
  } catch (error) {
    console.error('获取分类失败:', error)
  }
}

async function fetchOrders() {
  ordersLoading.value = true
  try {
    const params = { page_size: 50 }
    if (orderFilter.value) params.status = orderFilter.value
    const res = await api.getOrders(params)
    orders.value = res.orders || []
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    ordersLoading.value = false
  }
}

function selectCategory(cat) {
  selectedCategory.value = cat
}

function addToCart(dish) {
  const existingItem = cart.value.find(i => i.dish.id === dish.id)
  if (existingItem) {
    if (dish.stock === -1 || existingItem.quantity < dish.stock) {
      existingItem.quantity++
    }
  } else {
    cart.value.push({ dish, quantity: 1, remark: '' })
  }
}

function decreaseCart(dish) {
  const index = cart.value.findIndex(i => i.dish.id === dish.id)
  if (index !== -1) {
    if (cart.value[index].quantity > 1) {
      cart.value[index].quantity--
    } else {
      cart.value.splice(index, 1)
    }
  }
}

function clearCart() {
  cart.value = []
  orderRemark.value = ''
}

async function submitOrder() {
  if (cart.value.length === 0) return

  submitting.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const orderData = {
      items: cart.value.map(item => ({
        dish_id: item.dish.id,
        quantity: item.quantity,
        remark: item.remark
      })),
      remark: orderRemark.value
    }
    await api.createOrder(orderData)
    successMsg.value = '下单成功！'
    clearCart()
    fetchDishes() // 刷新库存
    fetchOrders() // 刷新订单列表
    setTimeout(() => {
      successMsg.value = ''
    }, 3000)
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    submitting.value = false
  }
}

async function cancelOrder(order) {
  if (!confirm('确定要取消这个订单吗？')) return

  try {
    await api.updateOrderStatus(order.id, 'cancelled')
    fetchOrders()
    fetchDishes() // 刷新库存
  } catch (error) {
    errorMsg.value = error.message
  }
}

onMounted(() => {
  fetchDishes()
  fetchCategories()
  fetchOrders()
})
</script>

<style scoped>
.menu-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.menu-container {
  flex: 1;
  display: flex;
  gap: 20px;
  min-height: 0;
}

.menu-categories {
  width: 120px;
  background: #fff;
  border-radius: 8px;
  padding: 12px;
  overflow-y: auto;
}

.category-item {
  padding: 12px;
  border-radius: 6px;
  cursor: pointer;
  text-align: center;
  transition: all 0.2s;
  margin-bottom: 4px;
}

.category-item:hover {
  background: #f5f5f5;
}

.category-item.active {
  background: #1890ff;
  color: #fff;
}

.menu-dishes {
  flex: 1;
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  overflow-y: auto;
}

.menu-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

.menu-dish-card {
  background: #fafafa;
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.2s;
}

.menu-dish-card:hover:not(.disabled) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.menu-dish-card.disabled {
  opacity: 0.6;
}

.menu-dish-image {
  width: 100%;
  height: 140px;
  background: #f0f0f0;
}

.menu-dish-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.menu-dish-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 13px;
}

.menu-dish-info {
  padding: 12px;
}

.menu-dish-name {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
  color: #333;
}

.menu-dish-desc {
  margin: 0 0 8px;
  font-size: 12px;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-dish-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}

.menu-dish-tag {
  padding: 2px 6px;
  background: #fff7e6;
  color: #fa8c16;
  border-radius: 4px;
  font-size: 11px;
}

.menu-dish-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.menu-dish-price {
  font-size: 16px;
  font-weight: 600;
  color: #f5222d;
}

.menu-dish-stock {
  font-size: 12px;
  color: #999;
}

.menu-dish-actions {
  padding: 12px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  justify-content: center;
}

.btn-sm {
  padding: 6px 16px;
  font-size: 13px;
}

.quantity-control {
  display: flex;
  align-items: center;
  gap: 12px;
}

.quantity-btn {
  width: 28px;
  height: 28px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.quantity-btn:hover:not(:disabled) {
  border-color: #1890ff;
  color: #1890ff;
}

.quantity-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.quantity-btn.small {
  width: 22px;
  height: 22px;
  font-size: 14px;
}

.quantity-value {
  min-width: 24px;
  text-align: center;
  font-weight: 600;
}

.sold-out-text {
  color: #999;
  font-size: 13px;
}

.menu-cart {
  width: 280px;
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
}

.cart-title {
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.cart-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 14px;
}

.cart-items {
  flex: 1;
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
  font-size: 14px;
  color: #333;
}

.cart-item-price {
  font-size: 14px;
  color: #f5222d;
}

.cart-item-quantity {
  display: flex;
  align-items: center;
  gap: 8px;
  justify-content: flex-end;
}

.cart-footer {
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.cart-remark {
  margin-bottom: 12px;
}

.cart-total {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 14px;
}

.cart-total-price {
  font-size: 20px;
  font-weight: 600;
  color: #f5222d;
}

/* 订单列表样式 */
.orders-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.order-card {
  background: #fafafa;
  border-radius: 8px;
  padding: 16px;
}

.order-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.order-no {
  font-size: 14px;
  font-weight: 600;
  color: #333;
}

.order-status {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  color: #fff;
}

.order-status.pending {
  background: #faad14;
}

.order-status.confirmed {
  background: #1890ff;
}

.order-status.preparing {
  background: #722ed1;
}

.order-status.completed {
  background: #52c41a;
}

.order-status.cancelled {
  background: #999;
}

.order-items {
  border-top: 1px solid #e8e8e8;
  border-bottom: 1px solid #e8e8e8;
  padding: 12px 0;
}

.order-item {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
  font-size: 13px;
  color: #666;
}

.order-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
}

.order-time {
  font-size: 12px;
  color: #999;
}

.order-total {
  font-size: 14px;
  font-weight: 600;
  color: #f5222d;
}

.order-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}

.message-success {
  background-color: #f6ffed;
  border: 1px solid #b7eb8f;
  color: #52c41a;
  padding: 12px 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}
</style>
