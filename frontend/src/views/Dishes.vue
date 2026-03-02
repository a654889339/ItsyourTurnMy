<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">菜品管理</h1>
      <button class="btn btn-primary" @click="openAddModal">添加菜品</button>
    </div>

    <div v-if="errorMsg" class="message message-error">{{ errorMsg }}</div>

    <!-- 筛选区域 -->
    <div class="card" style="margin-bottom: 20px; padding: 16px;">
      <div style="display: flex; gap: 16px; flex-wrap: wrap; align-items: center;">
        <div>
          <label style="margin-right: 8px;">分类：</label>
          <select v-model="filters.category" class="form-select" style="width: 150px;" @change="fetchDishes">
            <option value="">全部</option>
            <option v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</option>
          </select>
        </div>
        <div>
          <label style="margin-right: 8px;">状态：</label>
          <select v-model="filters.status" class="form-select" style="width: 150px;" @change="fetchDishes">
            <option value="">全部</option>
            <option value="available">可用</option>
            <option value="sold_out">售罄</option>
            <option value="disabled">下架</option>
          </select>
        </div>
        <div>
          <input v-model="filters.keyword" type="text" class="form-input" style="width: 200px;"
            placeholder="搜索菜品名称..." @input="handleSearch" />
        </div>
      </div>
    </div>

    <div class="card">
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="dishes.length === 0" class="empty-state">
        暂无菜品，点击上方按钮添加
      </div>
      <div v-else class="dishes-grid">
        <div v-for="dish in dishes" :key="dish.id" class="dish-card">
          <button class="dish-history-btn" @click="showChangeLogs(dish)" title="查看变化记录">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
          </button>
          <div class="dish-image">
            <img v-if="dish.image" :src="dish.image" :alt="dish.name" />
            <div v-else class="dish-image-placeholder">暂无图片</div>
            <span class="dish-status" :class="dish.status">{{ getStatusName(dish.status) }}</span>
          </div>
          <div class="dish-info">
            <h3 class="dish-name">{{ dish.name }}</h3>
            <p class="dish-category">{{ dish.category }}</p>
            <p class="dish-price">¥{{ formatMoney(dish.price) }}</p>
            <p class="dish-stock">
              库存：{{ dish.stock === -1 ? '不限' : dish.stock }}
            </p>
            <div v-if="dish.dietary_tags" class="dish-tags">
              <span v-for="tag in dish.dietary_tags.split(',')" :key="tag" class="dish-tag">{{ tag }}</span>
            </div>
          </div>
          <div class="dish-actions">
            <button class="btn btn-default" @click="editDish(dish)">编辑</button>
            <button class="btn btn-danger" @click="deleteDish(dish)">删除</button>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <div v-if="total > pageSize" class="pagination">
        <button class="btn btn-default" :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
        <span style="margin: 0 16px;">第 {{ page }} / {{ Math.ceil(total / pageSize) }} 页</span>
        <button class="btn btn-default" :disabled="page >= Math.ceil(total / pageSize)" @click="changePage(page + 1)">下一页</button>
      </div>
    </div>

    <!-- 添加/编辑菜品模态框 -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal" style="max-width: 600px;">
        <div class="modal-header">
          <h3 class="modal-title">{{ editingDish ? '编辑菜品' : '添加菜品' }}</h3>
          <button class="modal-close" @click="closeModal">&times;</button>
        </div>

        <form @submit.prevent="handleSubmit">
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
            <div class="form-group">
              <label class="form-label">菜品名称 *</label>
              <input v-model="form.name" type="text" class="form-input" required />
            </div>

            <div class="form-group">
              <label class="form-label">价格 *</label>
              <input v-model.number="form.price" type="number" step="0.01" min="0" class="form-input" required />
            </div>

            <div class="form-group">
              <label class="form-label">分类 *</label>
              <select v-model="form.category" class="form-select" required>
                <option value="">请选择</option>
                <option value="荤菜">荤菜</option>
                <option value="素菜">素菜</option>
                <option value="汤类">汤类</option>
                <option value="主食">主食</option>
                <option value="饮品">饮品</option>
                <option value="甜点">甜点</option>
                <option value="小吃">小吃</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">库存</label>
              <input v-model.number="form.stock" type="number" min="-1" class="form-input" placeholder="-1表示不限" />
            </div>

            <div class="form-group">
              <label class="form-label">状态</label>
              <select v-model="form.status" class="form-select">
                <option value="available">可用</option>
                <option value="sold_out">售罄</option>
                <option value="disabled">下架</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">排序</label>
              <input v-model.number="form.sort_order" type="number" min="0" class="form-input" />
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">饮食偏好标签</label>
            <input v-model="form.dietary_tags" type="text" class="form-input" placeholder="用逗号分隔，如：辣,素食,清真" />
          </div>

          <div class="form-group">
            <label class="form-label">描述</label>
            <textarea v-model="form.description" class="form-input" rows="2" placeholder="菜品描述..."></textarea>
          </div>

          <div class="form-group">
            <label class="form-label">图片</label>
            <div class="image-upload">
              <div v-if="form.image || imagePreview" class="image-preview">
                <img :src="imagePreview || form.image" alt="预览" />
                <button type="button" class="image-remove" @click="removeImage">&times;</button>
              </div>
              <div v-else class="image-placeholder" @click="triggerFileInput">
                点击上传图片
              </div>
              <input ref="fileInput" type="file" accept="image/*" style="display: none;" @change="handleFileChange" />
            </div>
          </div>

          <div class="modal-footer">
            <button type="button" class="btn btn-default" @click="closeModal">取消</button>
            <button type="submit" class="btn btn-primary" :disabled="submitting">
              {{ submitting ? '提交中...' : '确定' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- 变化记录弹窗 -->
    <div v-if="showChangeLogsModal" class="modal-overlay" @click.self="showChangeLogsModal = false">
      <div class="modal" style="max-width: 500px;">
        <div class="modal-header">
          <h3 class="modal-title">{{ currentDishName }} - 变化记录</h3>
          <button class="modal-close" @click="showChangeLogsModal = false">&times;</button>
        </div>
        <div class="change-logs-content">
          <div v-if="changeLogsLoading" class="loading">加载中...</div>
          <div v-else-if="changeLogs.length === 0" class="empty-state" style="padding: 20px;">
            暂无变化记录
          </div>
          <div v-else class="change-logs-list">
            <div v-for="log in changeLogs" :key="log.id" class="change-log-item">
              <div class="change-log-header">
                <div class="change-log-type" :class="log.type">
                  {{ log.type === 'stock' ? '库存' : '价格' }}
                </div>
                <span class="change-log-time">{{ formatTime(log.created_at) }}</span>
              </div>
              <div class="change-log-detail">
                <span class="old-value">{{ formatLogValue(log.type, log.old_value) }}</span>
                <span class="arrow">→</span>
                <span class="new-value">{{ formatLogValue(log.type, log.new_value) }}</span>
              </div>
              <div class="change-log-meta">
                <span class="remark">{{ log.remark }}</span>
                <span v-if="log.order_no" class="order-no">订单: {{ formatOrderNo(log.order_no) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'

const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const errorMsg = ref('')
const dishes = ref([])
const categories = ref([])
const editingDish = ref(null)
const fileInput = ref(null)
const imagePreview = ref('')
const selectedFile = ref(null)

// 变化记录相关
const showChangeLogsModal = ref(false)
const changeLogsLoading = ref(false)
const changeLogs = ref([])
const currentDishName = ref('')

const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const filters = reactive({
  category: '',
  status: '',
  keyword: ''
})

const form = reactive({
  name: '',
  description: '',
  price: 0,
  image: '',
  category: '',
  dietary_tags: '',
  stock: -1,
  status: 'available',
  sort_order: 0
})

const statusNames = {
  available: '可用',
  sold_out: '售罄',
  disabled: '下架'
}

function getStatusName(status) {
  return statusNames[status] || status
}

function formatMoney(value) {
  return Number(value).toFixed(2)
}

let searchTimer = null
function handleSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    fetchDishes()
  }, 300)
}

async function fetchDishes() {
  loading.value = true
  errorMsg.value = ''
  try {
    const params = {
      page: page.value,
      page_size: pageSize.value
    }
    if (filters.category) params.category = filters.category
    if (filters.status) params.status = filters.status
    if (filters.keyword) params.keyword = filters.keyword

    const res = await api.getDishes(params)
    dishes.value = res.dishes || []
    total.value = res.total || 0
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

function changePage(newPage) {
  page.value = newPage
  fetchDishes()
}

// 显示变化记录
async function showChangeLogs(dish) {
  currentDishName.value = dish.name
  showChangeLogsModal.value = true
  changeLogsLoading.value = true
  changeLogs.value = []

  try {
    const res = await api.getDishChangeLogs(dish.id)
    changeLogs.value = res.logs || []
  } catch (error) {
    console.error('获取变化记录失败:', error)
  } finally {
    changeLogsLoading.value = false
  }
}

function formatLogValue(type, value) {
  if (type === 'price') {
    return '¥' + Number(value).toFixed(2)
  }
  return Math.round(value)
}

function formatTime(timeStr) {
  if (!timeStr) return ''
  // 手动解析时间字符串，避免浏览器UTC时区转换问题
  const match = timeStr.match(/(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/)
  if (match) {
    const [, , m, d, h, min] = match
    return `${m}-${d} ${h}:${min}`
  }
  return timeStr
}

function formatOrderNo(orderNo) {
  if (!orderNo) return ''
  return orderNo
}

function openAddModal() {
  editingDish.value = null
  resetForm()
  showModal.value = true
}

function editDish(dish) {
  editingDish.value = dish
  form.name = dish.name
  form.description = dish.description || ''
  form.price = dish.price
  form.image = dish.image || ''
  form.category = dish.category
  form.dietary_tags = dish.dietary_tags || ''
  form.stock = dish.stock
  form.status = dish.status
  form.sort_order = dish.sort_order || 0
  imagePreview.value = ''
  selectedFile.value = null
  showModal.value = true
}

function resetForm() {
  form.name = ''
  form.description = ''
  form.price = 0
  form.image = ''
  form.category = ''
  form.dietary_tags = ''
  form.stock = -1
  form.status = 'available'
  form.sort_order = 0
  imagePreview.value = ''
  selectedFile.value = null
}

function closeModal() {
  showModal.value = false
  editingDish.value = null
  resetForm()
}

function triggerFileInput() {
  fileInput.value?.click()
}

function handleFileChange(event) {
  const file = event.target.files[0]
  if (!file) return

  if (!file.type.startsWith('image/')) {
    errorMsg.value = '请选择图片文件'
    return
  }

  if (file.size > 5 * 1024 * 1024) {
    errorMsg.value = '图片大小不能超过5MB'
    return
  }

  selectedFile.value = file
  const reader = new FileReader()
  reader.onload = (e) => {
    imagePreview.value = e.target.result
  }
  reader.readAsDataURL(file)
}

function removeImage() {
  form.image = ''
  imagePreview.value = ''
  selectedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

async function handleSubmit() {
  submitting.value = true
  errorMsg.value = ''
  try {
    // 如果有新上传的图片，先上传
    if (selectedFile.value) {
      const uploadRes = await api.uploadImage(selectedFile.value)
      form.image = uploadRes.url
    }

    const data = {
      name: form.name,
      description: form.description,
      price: form.price,
      image: form.image,
      category: form.category,
      dietary_tags: form.dietary_tags,
      stock: form.stock,
      status: form.status,
      sort_order: form.sort_order
    }

    if (editingDish.value) {
      await api.updateDish(editingDish.value.id, data)
    } else {
      await api.createDish(data)
    }
    closeModal()
    fetchDishes()
    fetchCategories()
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    submitting.value = false
  }
}

async function deleteDish(dish) {
  if (!confirm(`确定要删除菜品 "${dish.name}" 吗？`)) return

  errorMsg.value = ''
  try {
    await api.deleteDish(dish.id)
    fetchDishes()
    fetchCategories()
  } catch (error) {
    errorMsg.value = error.message
  }
}

onMounted(() => {
  fetchDishes()
  fetchCategories()
})
</script>

<style scoped>
.dishes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
  padding: 20px;
}

.dish-card {
  position: relative;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s, box-shadow 0.2s;
}

.dish-history-btn {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 10;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.9);
  color: #666;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.dish-history-btn:hover {
  background: #1890ff;
  color: #fff;
}

.dish-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.dish-image {
  position: relative;
  width: 100%;
  height: 180px;
  background: #f5f5f5;
}

.dish-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.dish-image-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 14px;
}

.dish-status {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  color: #fff;
}

.dish-status.available {
  background: #52c41a;
}

.dish-status.sold_out {
  background: #faad14;
}

.dish-status.disabled {
  background: #999;
}

.dish-info {
  padding: 16px;
}

.dish-name {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.dish-category {
  margin: 0 0 4px;
  font-size: 13px;
  color: #666;
}

.dish-price {
  margin: 0 0 4px;
  font-size: 18px;
  font-weight: 600;
  color: #f5222d;
}

.dish-stock {
  margin: 0 0 8px;
  font-size: 13px;
  color: #999;
}

.dish-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.dish-tag {
  padding: 2px 8px;
  background: #e6f7ff;
  color: #1890ff;
  border-radius: 4px;
  font-size: 12px;
}

.dish-actions {
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  gap: 8px;
}

.dish-actions .btn {
  flex: 1;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  border-top: 1px solid #f0f0f0;
}

.image-upload {
  width: 100%;
}

.image-preview {
  position: relative;
  width: 200px;
  height: 150px;
  border-radius: 8px;
  overflow: hidden;
}

.image-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-placeholder {
  width: 200px;
  height: 150px;
  border: 2px dashed #d9d9d9;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  cursor: pointer;
  transition: border-color 0.2s;
}

.image-placeholder:hover {
  border-color: #1890ff;
  color: #1890ff;
}

/* 变化记录弹窗样式 */
.change-logs-content {
  max-height: 400px;
  overflow-y: auto;
}

.change-logs-list {
  padding: 10px 20px;
}

.change-log-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.change-log-item:last-child {
  border-bottom: none;
}

.change-log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.change-log-type {
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.change-log-type.stock {
  background: #e6f7ff;
  color: #1890ff;
}

.change-log-type.price {
  background: #fff7e6;
  color: #fa8c16;
}

.change-log-time {
  font-size: 12px;
  color: #999;
}

.change-log-detail {
  font-size: 14px;
}

.change-log-detail .old-value {
  color: #999;
  text-decoration: line-through;
}

.change-log-detail .arrow {
  margin: 0 8px;
  color: #999;
}

.change-log-detail .new-value {
  color: #333;
  font-weight: 500;
}

.change-log-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #999;
}

.change-log-meta .order-no {
  color: #1890ff;
  font-family: monospace;
}
</style>
