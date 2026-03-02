<template>
  <div class="order-management">
    <div class="page-header">
      <h2>订单管理</h2>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-group">
        <label>日期筛选：</label>
        <div class="date-shortcuts">
          <button :class="{ active: dateMode === 'today' }" @click="setDateFilter('today')">今日</button>
          <button :class="{ active: dateMode === 'week' }" @click="setDateFilter('week')">本周</button>
          <button :class="{ active: dateMode === 'month' }" @click="setDateFilter('month')">本月</button>
          <button :class="{ active: dateMode === 'year' }" @click="setDateFilter('year')">本年</button>
          <button :class="{ active: dateMode === 'all' }" @click="setDateFilter('all')">全部</button>
        </div>
      </div>
      <div class="filter-group">
        <label>自定义日期：</label>
        <input type="date" v-model="filters.startDate" @change="dateMode = 'custom'; loadOrders()" />
        <span>至</span>
        <input type="date" v-model="filters.endDate" @change="dateMode = 'custom'; loadOrders()" />
      </div>
      <div class="filter-group">
        <label>餐桌：</label>
        <select v-model="filters.tableId" @change="loadOrders">
          <option value="">全部餐桌</option>
          <option v-for="table in tables" :key="table.id" :value="table.id">{{ table.table_no }}</option>
        </select>
      </div>
      <div class="filter-group">
        <label>状态：</label>
        <select v-model="filters.status" @change="loadOrders">
          <option value="">全部状态</option>
          <option value="pending">待确认</option>
          <option value="confirmed">已确认</option>
          <option value="preparing">制作中</option>
          <option value="completed">已完成</option>
          <option value="cancelled">已取消</option>
        </select>
      </div>
      <div class="filter-group">
        <label>来源：</label>
        <select v-model="filters.orderSource" @change="loadOrders">
          <option value="">全部来源</option>
          <option value="admin">后台下单</option>
          <option value="scan">扫码点单</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="orders.length === 0" class="empty">暂无订单</div>

    <div v-else class="order-list">
      <div v-for="order in orders" :key="order.id" class="order-card">
        <div class="order-header">
          <div class="order-info">
            <span class="order-no">{{ order.order_no }}</span>
            <span class="order-source" :class="order.order_source">
              {{ order.order_source === 'scan' ? '扫码点单' : '后台下单' }}
            </span>
            <span v-if="order.table_no" class="table-no">{{ order.table_no }}</span>
            <span v-if="order.customer_name" class="customer-name">{{ order.customer_name }}</span>
          </div>
          <div class="order-status">
            <select
              :value="order.status"
              @change="updateStatus(order, $event.target.value)"
              :class="'status-' + order.status"
            >
              <option value="pending">待确认</option>
              <option value="confirmed">已确认</option>
              <option value="preparing">制作中</option>
              <option value="completed">已完成</option>
              <option value="cancelled">已取消</option>
            </select>
          </div>
        </div>

        <div class="order-items">
          <div v-for="(item, index) in order.items" :key="item.id" class="order-item">
            <img v-if="item.dish_image" :src="item.dish_image" class="item-image" />
            <div v-else class="item-image placeholder">无图</div>
            <div class="item-info">
              <div class="item-name">{{ item.dish_name }}</div>
              <div class="item-price">¥{{ item.price.toFixed(2) }}</div>
            </div>
            <div class="item-quantity">
              <button
                class="qty-btn"
                @click="decreaseQuantity(order, index)"
                :disabled="order.status !== 'pending'"
              >-</button>
              <span>{{ item.quantity }}</span>
              <button
                class="qty-btn"
                @click="increaseQuantity(order, index)"
                :disabled="order.status !== 'pending'"
              >+</button>
            </div>
            <div class="item-subtotal">¥{{ (item.price * item.quantity).toFixed(2) }}</div>
            <div class="item-remark">
              <input
                type="text"
                v-model="item.remark"
                placeholder="备注"
                @blur="updateItemRemark(order, item)"
                :disabled="order.status !== 'pending'"
              />
            </div>
            <button
              class="remove-btn"
              @click="removeItem(order, index)"
              :disabled="order.status !== 'pending' || order.items.length <= 1"
              title="移除"
            >×</button>
          </div>

          <!-- 添加菜品按钮 -->
          <button
            v-if="order.status === 'pending'"
            class="add-item-btn"
            @click="openAddDishModal(order)"
          >
            + 添加菜品
          </button>
        </div>

        <div class="order-footer">
          <div class="order-remark">
            <span>订单备注：</span>
            <input
              type="text"
              v-model="order.remark"
              placeholder="无备注"
              @blur="updateOrderRemark(order)"
              :disabled="order.status !== 'pending'"
            />
          </div>
          <div class="order-time">{{ formatTime(order.created_at) }}</div>
          <div class="order-total">合计：¥{{ order.total_price.toFixed(2) }}</div>
          <button
            v-if="order.status === 'cancelled' || order.status === 'completed'"
            class="delete-btn"
            @click="deleteOrder(order)"
          >删除订单</button>
        </div>
      </div>
    </div>

    <div v-if="total > pageSize" class="pagination">
      <button :disabled="page <= 1" @click="page--; loadOrders()">上一页</button>
      <span>{{ page }} / {{ Math.ceil(total / pageSize) }}</span>
      <button :disabled="page >= Math.ceil(total / pageSize)" @click="page++; loadOrders()">下一页</button>
    </div>

    <!-- 添加菜品弹窗 -->
    <div v-if="showAddDishModal" class="modal-overlay" @click.self="showAddDishModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>添加菜品</h3>
          <button class="close-btn" @click="showAddDishModal = false">×</button>
        </div>
        <div class="modal-body">
          <div class="dish-search">
            <input type="text" v-model="dishSearch" placeholder="搜索菜品..." />
          </div>
          <div class="dish-list">
            <div
              v-for="dish in filteredDishes"
              :key="dish.id"
              class="dish-item"
              @click="addDishToOrder(dish)"
            >
              <img v-if="dish.image" :src="dish.image" class="dish-image" />
              <div v-else class="dish-image placeholder">无图</div>
              <div class="dish-info">
                <div class="dish-name">{{ dish.name }}</div>
                <div class="dish-price">¥{{ dish.price.toFixed(2) }}</div>
              </div>
              <div class="dish-stock" :class="{ low: dish.stock >= 0 && dish.stock < 5 }">
                {{ dish.stock === -1 ? '充足' : `剩${dish.stock}` }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api'

const orders = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const tables = ref([])
const dateMode = ref('today')
const filters = ref({
  status: '',
  orderSource: '',
  tableId: '',
  startDate: '',
  endDate: ''
})

const showAddDishModal = ref(false)
const currentEditOrder = ref(null)
const dishes = ref([])
const dishSearch = ref('')

const filteredDishes = computed(() => {
  if (!dishSearch.value) return dishes.value
  const search = dishSearch.value.toLowerCase()
  return dishes.value.filter(d =>
    d.name.toLowerCase().includes(search) ||
    (d.category && d.category.toLowerCase().includes(search))
  )
})

function formatDate(date) {
  return date.toISOString().split('T')[0]
}

function setDateFilter(mode) {
  dateMode.value = mode
  const today = new Date()

  switch (mode) {
    case 'today':
      filters.value.startDate = formatDate(today)
      filters.value.endDate = formatDate(today)
      break
    case 'week':
      const weekStart = new Date(today)
      weekStart.setDate(today.getDate() - today.getDay())
      filters.value.startDate = formatDate(weekStart)
      filters.value.endDate = formatDate(today)
      break
    case 'month':
      const monthStart = new Date(today.getFullYear(), today.getMonth(), 1)
      filters.value.startDate = formatDate(monthStart)
      filters.value.endDate = formatDate(today)
      break
    case 'year':
      const yearStart = new Date(today.getFullYear(), 0, 1)
      filters.value.startDate = formatDate(yearStart)
      filters.value.endDate = formatDate(today)
      break
    case 'all':
      filters.value.startDate = ''
      filters.value.endDate = ''
      break
  }
  loadOrders()
}

async function loadTables() {
  try {
    const res = await api.getTables()
    tables.value = res.tables || []
  } catch (error) {
    console.error('加载餐桌失败:', error)
  }
}

async function loadOrders() {
  loading.value = true
  try {
    const params = {
      status: filters.value.status,
      order_source: filters.value.orderSource,
      page: page.value,
      page_size: pageSize.value
    }
    if (filters.value.tableId) {
      params.table_id = filters.value.tableId
    }
    if (filters.value.startDate) {
      params.start_date = filters.value.startDate
    }
    if (filters.value.endDate) {
      params.end_date = filters.value.endDate
    }
    const res = await api.getOrders(params)
    orders.value = res.orders || []
    total.value = res.total || 0
  } catch (error) {
    alert('加载订单失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

async function loadDishes() {
  try {
    const res = await api.getDishes({ status: 'available', page_size: 1000 })
    dishes.value = res.dishes || []
  } catch (error) {
    console.error('加载菜品失败:', error)
  }
}

async function updateStatus(order, newStatus) {
  if (order.status === newStatus) return

  const statusNames = {
    pending: '待确认',
    confirmed: '已确认',
    preparing: '制作中',
    completed: '已完成',
    cancelled: '已取消'
  }

  if (!confirm(`确定要将订单状态改为"${statusNames[newStatus]}"吗？`)) {
    return
  }

  try {
    await api.updateOrderStatus(order.id, newStatus)
    order.status = newStatus
  } catch (error) {
    alert('更新状态失败: ' + error.message)
  }
}

async function updateOrder(order) {
  try {
    await api.updateOrder(order.id, {
      items: order.items.map(item => ({
        dish_id: item.dish_id,
        quantity: item.quantity,
        remark: item.remark || ''
      })),
      remark: order.remark || ''
    })
    // 重新计算总价
    order.total_price = order.items.reduce((sum, item) => sum + item.price * item.quantity, 0)
  } catch (error) {
    alert('更新订单失败: ' + error.message)
    loadOrders() // 刷新数据
  }
}

function decreaseQuantity(order, index) {
  if (order.items[index].quantity > 1) {
    order.items[index].quantity--
    updateOrder(order)
  }
}

function increaseQuantity(order, index) {
  order.items[index].quantity++
  updateOrder(order)
}

function updateItemRemark(order, item) {
  updateOrder(order)
}

function updateOrderRemark(order) {
  updateOrder(order)
}

function removeItem(order, index) {
  if (order.items.length <= 1) {
    alert('订单至少需要一个菜品')
    return
  }
  if (!confirm(`确定要移除"${order.items[index].dish_name}"吗？`)) return
  order.items.splice(index, 1)
  updateOrder(order)
}

function openAddDishModal(order) {
  currentEditOrder.value = order
  dishSearch.value = ''
  showAddDishModal.value = true
}

function addDishToOrder(dish) {
  if (!currentEditOrder.value) return

  // 检查是否已有该菜品
  const existingItem = currentEditOrder.value.items.find(item => item.dish_id === dish.id)
  if (existingItem) {
    existingItem.quantity++
  } else {
    currentEditOrder.value.items.push({
      id: 0,
      dish_id: dish.id,
      dish_name: dish.name,
      dish_image: dish.image,
      price: dish.price,
      quantity: 1,
      remark: ''
    })
  }

  updateOrder(currentEditOrder.value)
  showAddDishModal.value = false
}

async function deleteOrder(order) {
  if (!confirm('确定要删除此订单吗？此操作不可恢复。')) return

  try {
    await api.deleteOrder(order.id)
    loadOrders()
  } catch (error) {
    alert('删除失败: ' + error.message)
  }
}

function formatTime(dateStr) {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  setDateFilter('today') // 默认显示今日订单
  loadDishes()
  loadTables()
})
</script>

<style scoped>
.order-management {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
}

.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-group label {
  font-size: 13px;
  color: #666;
  white-space: nowrap;
}

.filter-group select,
.filter-group input[type="date"] {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 13px;
}

.date-shortcuts {
  display: flex;
  gap: 4px;
}

.date-shortcuts button {
  padding: 6px 12px;
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.date-shortcuts button:hover {
  border-color: #1890ff;
  color: #1890ff;
}

.date-shortcuts button.active {
  background: #1890ff;
  border-color: #1890ff;
  color: #fff;
}

.loading, .empty {
  text-align: center;
  padding: 40px;
  color: #999;
}

.order-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.order-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
}

.order-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: #f8f9fa;
  border-bottom: 1px solid #eee;
}

.order-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.order-no {
  font-weight: 600;
  font-size: 14px;
  color: #333;
}

.order-source {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.order-source.scan {
  background: #e3f2fd;
  color: #1976d2;
}

.order-source.admin {
  background: #f3e5f5;
  color: #7b1fa2;
}

.table-no {
  padding: 2px 8px;
  background: #fff3e0;
  color: #e65100;
  border-radius: 4px;
  font-size: 12px;
}

.customer-name {
  color: #666;
  font-size: 13px;
}

.order-status select {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
}

.status-pending { background: #fff3e0; color: #e65100; }
.status-confirmed { background: #e3f2fd; color: #1976d2; }
.status-preparing { background: #fce4ec; color: #c2185b; }
.status-completed { background: #e8f5e9; color: #388e3c; }
.status-cancelled { background: #ffebee; color: #c62828; }

.order-items {
  padding: 16px;
}

.order-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.order-item:last-of-type {
  border-bottom: none;
}

.item-image {
  width: 50px;
  height: 50px;
  border-radius: 4px;
  object-fit: cover;
}

.item-image.placeholder {
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 12px;
}

.item-info {
  flex: 1;
  min-width: 100px;
}

.item-name {
  font-weight: 500;
  margin-bottom: 4px;
}

.item-price {
  color: #f44336;
  font-size: 13px;
}

.item-quantity {
  display: flex;
  align-items: center;
  gap: 8px;
}

.qty-btn {
  width: 28px;
  height: 28px;
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
  font-size: 16px;
}

.qty-btn:hover:not(:disabled) {
  background: #f5f5f5;
}

.qty-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.item-subtotal {
  width: 80px;
  text-align: right;
  font-weight: 500;
  color: #f44336;
}

.item-remark {
  width: 150px;
}

.item-remark input {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 13px;
}

.item-remark input:disabled {
  background: #f5f5f5;
}

.remove-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: #ffebee;
  color: #c62828;
  border-radius: 4px;
  cursor: pointer;
  font-size: 18px;
}

.remove-btn:hover:not(:disabled) {
  background: #ffcdd2;
}

.remove-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.add-item-btn {
  margin-top: 12px;
  padding: 10px 20px;
  background: #e3f2fd;
  color: #1976d2;
  border: 1px dashed #1976d2;
  border-radius: 4px;
  cursor: pointer;
  width: 100%;
}

.add-item-btn:hover {
  background: #bbdefb;
}

.order-footer {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 16px;
  background: #f8f9fa;
  border-top: 1px solid #eee;
}

.order-remark {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.order-remark span {
  color: #666;
  font-size: 13px;
  white-space: nowrap;
}

.order-remark input {
  flex: 1;
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 13px;
}

.order-remark input:disabled {
  background: #f5f5f5;
}

.order-time {
  color: #999;
  font-size: 13px;
}

.order-total {
  font-weight: 600;
  font-size: 16px;
  color: #f44336;
}

.delete-btn {
  padding: 6px 12px;
  background: #ffebee;
  color: #c62828;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.delete-btn:hover {
  background: #ffcdd2;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 20px;
}

.pagination button {
  padding: 8px 16px;
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
}

.pagination button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 弹窗样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #fff;
  border-radius: 8px;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  margin: 0;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: none;
  font-size: 24px;
  cursor: pointer;
  color: #999;
}

.close-btn:hover {
  color: #333;
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
}

.dish-search {
  margin-bottom: 16px;
}

.dish-search input {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.dish-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dish-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid #eee;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.dish-item:hover {
  border-color: #1976d2;
  background: #f5f9ff;
}

.dish-image {
  width: 50px;
  height: 50px;
  border-radius: 4px;
  object-fit: cover;
}

.dish-info {
  flex: 1;
}

.dish-name {
  font-weight: 500;
  margin-bottom: 4px;
}

.dish-price {
  color: #f44336;
  font-size: 14px;
}

.dish-stock {
  padding: 4px 8px;
  background: #e8f5e9;
  color: #388e3c;
  border-radius: 4px;
  font-size: 12px;
}

.dish-stock.low {
  background: #fff3e0;
  color: #e65100;
}
</style>
