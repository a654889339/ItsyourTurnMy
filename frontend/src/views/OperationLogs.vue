<template>
  <div class="operation-logs">
    <div class="page-header">
      <h2>操作记录</h2>
    </div>

    <!-- 筛选区域 -->
    <div class="filter-section">
      <div class="filter-row">
        <!-- 日期快捷选择 -->
        <div class="filter-group">
          <label>时间范围</label>
          <div class="date-shortcuts">
            <button
              v-for="item in dateShortcuts"
              :key="item.value"
              :class="['shortcut-btn', { active: dateMode === item.value }]"
              @click="setDateFilter(item.value)"
            >
              {{ item.label }}
            </button>
          </div>
        </div>

        <!-- 自定义日期 -->
        <div class="filter-group" v-if="dateMode === 'custom'">
          <label>自定义日期</label>
          <div class="date-range">
            <input type="date" v-model="filters.startDate" @change="loadLogs" />
            <span>至</span>
            <input type="date" v-model="filters.endDate" @change="loadLogs" />
          </div>
        </div>
      </div>

      <div class="filter-row">
        <!-- 模块筛选 -->
        <div class="filter-group">
          <label>模块</label>
          <select v-model="filters.module" @change="loadLogs">
            <option value="">全部模块</option>
            <option value="dish">菜品管理</option>
            <option value="order">订单管理</option>
            <option value="table">餐桌管理</option>
            <option value="account">账户管理</option>
            <option value="transaction">收支记录</option>
          </select>
        </div>

        <!-- 操作类型 -->
        <div class="filter-group">
          <label>操作类型</label>
          <select v-model="filters.action" @change="loadLogs">
            <option value="">全部操作</option>
            <option value="create">新增</option>
            <option value="update">修改</option>
            <option value="delete">删除</option>
          </select>
        </div>

        <!-- 关键词搜索 -->
        <div class="filter-group search-group">
          <label>搜索</label>
          <input
            type="text"
            v-model="filters.keyword"
            placeholder="搜索操作内容..."
            @keyup.enter="loadLogs"
          />
          <button class="search-btn" @click="loadLogs">搜索</button>
        </div>
      </div>
    </div>

    <!-- 日志列表 -->
    <div class="logs-container">
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="logs.length === 0" class="empty">暂无操作记录</div>
      <div v-else class="logs-list">
        <div class="log-item" v-for="log in logs" :key="log.id" @click="showLogDetail(log)">
          <div class="log-icon" :class="log.action">
            <span v-if="log.action === 'create'">+</span>
            <span v-else-if="log.action === 'update'">~</span>
            <span v-else-if="log.action === 'delete'">-</span>
          </div>
          <div class="log-content">
            <div class="log-header">
              <span class="log-module">{{ getModuleText(log.module) }}</span>
              <span class="log-action" :class="log.action">{{ getActionText(log.action) }}</span>
              <span class="log-target">{{ log.target_name }}</span>
            </div>
            <div class="log-desc">{{ log.description }}</div>
            <div class="log-meta">
              <span class="log-time">{{ formatTime(log.created_at) }}</span>
              <span class="log-user" v-if="log.username">操作人: {{ log.username }}</span>
              <span class="log-ip" v-if="log.ip">IP: {{ log.ip }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <div class="pagination" v-if="total > pageSize">
        <button :disabled="page === 1" @click="changePage(page - 1)">上一页</button>
        <span class="page-info">第 {{ page }} / {{ totalPages }} 页，共 {{ total }} 条</span>
        <button :disabled="page >= totalPages" @click="changePage(page + 1)">下一页</button>
      </div>
    </div>

    <!-- 详情弹窗 -->
    <div class="modal" v-if="showDetail" @click.self="showDetail = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>操作详情</h3>
          <button class="close-btn" @click="showDetail = false">&times;</button>
        </div>
        <div class="modal-body" v-if="currentLog">
          <div class="detail-row">
            <label>操作时间</label>
            <span>{{ formatTime(currentLog.created_at) }}</span>
          </div>
          <div class="detail-row">
            <label>操作模块</label>
            <span>{{ getModuleText(currentLog.module) }}</span>
          </div>
          <div class="detail-row">
            <label>操作类型</label>
            <span class="action-badge" :class="currentLog.action">{{ getActionText(currentLog.action) }}</span>
          </div>
          <div class="detail-row">
            <label>操作对象</label>
            <span>{{ currentLog.target_name }}</span>
          </div>
          <div class="detail-row">
            <label>操作描述</label>
            <span>{{ currentLog.description }}</span>
          </div>
          <div class="detail-row" v-if="currentLog.username">
            <label>操作人</label>
            <span>{{ currentLog.username }}</span>
          </div>
          <div class="detail-row" v-if="currentLog.ip">
            <label>IP地址</label>
            <span>{{ currentLog.ip }}</span>
          </div>
          <div class="detail-section" v-if="currentLog.old_value && currentLog.old_value !== '{}'">
            <label>修改前</label>
            <pre class="json-view">{{ formatJson(currentLog.old_value) }}</pre>
          </div>
          <div class="detail-section" v-if="currentLog.new_value && currentLog.new_value !== '{}'">
            <label>修改后</label>
            <pre class="json-view">{{ formatJson(currentLog.new_value) }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api'

const logs = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const dateMode = ref('today')
const showDetail = ref(false)
const currentLog = ref(null)

const filters = ref({
  module: '',
  action: '',
  startDate: '',
  endDate: '',
  keyword: ''
})

const dateShortcuts = [
  { label: '今天', value: 'today' },
  { label: '本周', value: 'week' },
  { label: '本月', value: 'month' },
  { label: '全部', value: 'all' },
  { label: '自定义', value: 'custom' }
]

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

function formatDate(date) {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
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
    case 'all':
      filters.value.startDate = ''
      filters.value.endDate = ''
      break
    case 'custom':
      // 保持当前值
      break
  }

  if (mode !== 'custom') {
    loadLogs()
  }
}

async function loadLogs() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      page_size: pageSize.value,
      module: filters.value.module,
      action: filters.value.action,
      start_date: filters.value.startDate,
      end_date: filters.value.endDate,
      keyword: filters.value.keyword
    }
    const res = await api.get('/operation-logs', { params })
    logs.value = res.logs || []
    total.value = res.total || 0
  } catch (err) {
    console.error('加载操作日志失败', err)
  } finally {
    loading.value = false
  }
}

function changePage(newPage) {
  page.value = newPage
  loadLogs()
}

function getModuleText(module) {
  const map = {
    dish: '菜品管理',
    order: '订单管理',
    table: '餐桌管理',
    account: '账户管理',
    transaction: '收支记录',
    category: '分类管理',
    user: '用户管理'
  }
  return map[module] || module
}

function getActionText(action) {
  const map = {
    create: '新增',
    update: '修改',
    delete: '删除'
  }
  return map[action] || action
}

function formatTime(timeStr) {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  return `${m}-${d} ${h}:${min}:${s}`
}

function formatJson(str) {
  try {
    const obj = JSON.parse(str)
    return JSON.stringify(obj, null, 2)
  } catch {
    return str
  }
}

function showLogDetail(log) {
  currentLog.value = log
  showDetail.value = true
}

onMounted(() => {
  setDateFilter('today')
})
</script>

<style scoped>
.operation-logs {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 24px;
  color: #333;
}

/* 筛选区域 */
.filter-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  margin-bottom: 15px;
}

.filter-row:last-child {
  margin-bottom: 0;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.filter-group label {
  font-size: 13px;
  color: #666;
}

.filter-group select,
.filter-group input[type="text"],
.filter-group input[type="date"] {
  padding: 8px 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 14px;
  min-width: 150px;
}

.date-shortcuts {
  display: flex;
  gap: 8px;
}

.shortcut-btn {
  padding: 6px 14px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}

.shortcut-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
}

.shortcut-btn.active {
  background: #1890ff;
  border-color: #1890ff;
  color: #fff;
}

.date-range {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-group {
  flex: 1;
  flex-direction: row;
  align-items: flex-end;
}

.search-group input {
  flex: 1;
}

.search-btn {
  padding: 8px 20px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

/* 日志列表 */
.logs-container {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.loading, .empty {
  text-align: center;
  padding: 40px;
  color: #999;
}

.log-item {
  display: flex;
  gap: 15px;
  padding: 15px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.2s;
}

.log-item:last-child {
  border-bottom: none;
}

.log-item:hover {
  background: #f8f8f8;
}

.log-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  font-size: 18px;
  flex-shrink: 0;
}

.log-icon.create {
  background: #e6f7ff;
  color: #1890ff;
}

.log-icon.update {
  background: #fff7e6;
  color: #fa8c16;
}

.log-icon.delete {
  background: #fff1f0;
  color: #f5222d;
}

.log-content {
  flex: 1;
  min-width: 0;
}

.log-header {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 6px;
}

.log-module {
  font-size: 12px;
  padding: 2px 8px;
  background: #f0f0f0;
  border-radius: 3px;
  color: #666;
}

.log-action {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 3px;
}

.log-action.create {
  background: #e6f7ff;
  color: #1890ff;
}

.log-action.update {
  background: #fff7e6;
  color: #fa8c16;
}

.log-action.delete {
  background: #fff1f0;
  color: #f5222d;
}

.log-target {
  font-weight: 500;
  color: #333;
}

.log-desc {
  font-size: 14px;
  color: #666;
  margin-bottom: 6px;
}

.log-meta {
  font-size: 12px;
  color: #999;
  display: flex;
  gap: 15px;
}

/* 分页 */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 15px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #f0f0f0;
}

.pagination button {
  padding: 6px 15px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
}

.pagination button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  font-size: 14px;
  color: #666;
}

/* 详情弹窗 */
.modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
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
  padding: 15px 20px;
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  margin: 0;
  font-size: 18px;
}

.close-btn {
  width: 30px;
  height: 30px;
  border: none;
  background: none;
  font-size: 24px;
  cursor: pointer;
  color: #999;
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
}

.detail-row {
  display: flex;
  margin-bottom: 15px;
}

.detail-row label {
  width: 80px;
  flex-shrink: 0;
  color: #999;
  font-size: 14px;
}

.detail-row span {
  color: #333;
  font-size: 14px;
}

.action-badge {
  padding: 2px 10px;
  border-radius: 3px;
  font-size: 12px;
}

.action-badge.create {
  background: #e6f7ff;
  color: #1890ff;
}

.action-badge.update {
  background: #fff7e6;
  color: #fa8c16;
}

.action-badge.delete {
  background: #fff1f0;
  color: #f5222d;
}

.detail-section {
  margin-top: 20px;
}

.detail-section label {
  display: block;
  margin-bottom: 10px;
  color: #999;
  font-size: 14px;
}

.json-view {
  background: #f5f5f5;
  padding: 15px;
  border-radius: 4px;
  font-size: 12px;
  line-height: 1.5;
  overflow-x: auto;
  margin: 0;
}
</style>
