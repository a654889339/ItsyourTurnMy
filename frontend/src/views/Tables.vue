<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">餐桌管理</h1>
      <button class="btn btn-primary" @click="openAddModal">添加餐桌</button>
    </div>

    <div v-if="errorMsg" class="message message-error">{{ errorMsg }}</div>
    <div v-if="successMsg" class="message message-success">{{ successMsg }}</div>

    <div class="card">
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="tables.length === 0" class="empty-state">
        暂无餐桌，点击上方按钮添加
      </div>
      <div v-else class="table-grid">
        <div v-for="table in tables" :key="table.id" class="table-card" :class="{ disabled: table.status === 'disabled' }">
          <div class="table-header">
            <span class="table-no">{{ table.table_no }}</span>
            <span class="table-status" :class="table.status">{{ table.status === 'active' ? '启用' : '禁用' }}</span>
          </div>
          <div class="table-info">
            <p class="table-capacity">座位数：{{ table.capacity }} 人</p>
          </div>
          <div class="table-actions">
            <button class="btn btn-default btn-sm" @click="showQRCode(table)">查看二维码</button>
            <button class="btn btn-default btn-sm" @click="editTable(table)">编辑</button>
            <button class="btn btn-danger btn-sm" @click="deleteTable(table)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑餐桌模态框 -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal" style="max-width: 400px;">
        <div class="modal-header">
          <h3 class="modal-title">{{ editingTable ? '编辑餐桌' : '添加餐桌' }}</h3>
          <button class="modal-close" @click="closeModal">&times;</button>
        </div>

        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label class="form-label">桌号 *</label>
            <input v-model="form.table_no" type="text" class="form-input" placeholder="如：A01, 1号桌" required />
          </div>

          <div class="form-group">
            <label class="form-label">座位数</label>
            <input v-model.number="form.capacity" type="number" min="1" max="20" class="form-input" />
          </div>

          <div v-if="editingTable" class="form-group">
            <label class="form-label">状态</label>
            <select v-model="form.status" class="form-select">
              <option value="active">启用</option>
              <option value="disabled">禁用</option>
            </select>
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

    <!-- 二维码模态框 -->
    <div v-if="showQRModal" class="modal-overlay" @click.self="closeQRModal">
      <div class="modal" style="max-width: 400px;">
        <div class="modal-header">
          <h3 class="modal-title">{{ currentTable?.table_no }} - 点餐二维码</h3>
          <button class="modal-close" @click="closeQRModal">&times;</button>
        </div>

        <div class="qr-content">
          <div class="qr-code-container">
            <canvas ref="qrCanvas"></canvas>
          </div>
          <p class="qr-url">{{ qrUrl }}</p>
          <p class="qr-tip">顾客扫码后可直接点餐</p>
        </div>

        <div class="modal-footer">
          <button class="btn btn-default" @click="regenerateQR">重新生成</button>
          <button class="btn btn-primary" @click="downloadQR">下载二维码</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick } from 'vue'
import api from '../api'
import QRCode from 'qrcode'

const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const showQRModal = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const tables = ref([])
const editingTable = ref(null)
const currentTable = ref(null)
const qrUrl = ref('')
const qrCanvas = ref(null)

const form = reactive({
  table_no: '',
  capacity: 4,
  status: 'active'
})

async function fetchTables() {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await api.getTables()
    tables.value = res.tables || []
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    loading.value = false
  }
}

function openAddModal() {
  editingTable.value = null
  resetForm()
  showModal.value = true
}

function editTable(table) {
  editingTable.value = table
  form.table_no = table.table_no
  form.capacity = table.capacity
  form.status = table.status
  showModal.value = true
}

function resetForm() {
  form.table_no = ''
  form.capacity = 4
  form.status = 'active'
}

function closeModal() {
  showModal.value = false
  editingTable.value = null
  resetForm()
}

async function handleSubmit() {
  submitting.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const data = {
      table_no: form.table_no,
      capacity: form.capacity
    }

    if (editingTable.value) {
      data.status = form.status
      await api.updateTable(editingTable.value.id, data)
      successMsg.value = '餐桌更新成功'
    } else {
      await api.createTable(data)
      successMsg.value = '餐桌创建成功'
    }
    closeModal()
    fetchTables()
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    submitting.value = false
  }
}

async function deleteTable(table) {
  if (!confirm(`确定要删除餐桌 "${table.table_no}" 吗？`)) return

  errorMsg.value = ''
  successMsg.value = ''
  try {
    await api.deleteTable(table.id)
    successMsg.value = '餐桌删除成功'
    fetchTables()
  } catch (error) {
    errorMsg.value = error.message
  }
}

async function showQRCode(table) {
  currentTable.value = table
  showQRModal.value = true

  try {
    const res = await api.getTableQRCode(table.id)
    const token = res.token
    // 构建扫码URL
    const baseUrl = window.location.origin
    qrUrl.value = `${baseUrl}/scan/${token}`

    // 生成二维码
    await nextTick()
    if (qrCanvas.value) {
      await QRCode.toCanvas(qrCanvas.value, qrUrl.value, {
        width: 256,
        margin: 2,
        color: {
          dark: '#000000',
          light: '#ffffff'
        }
      })
    }
  } catch (error) {
    errorMsg.value = error.message
    showQRModal.value = false
  }
}

function closeQRModal() {
  showQRModal.value = false
  currentTable.value = null
  qrUrl.value = ''
}

async function regenerateQR() {
  if (!currentTable.value) return

  if (!confirm('重新生成后，旧二维码将失效。确定继续？')) return

  try {
    await api.regenerateTableToken(currentTable.value.id)
    successMsg.value = '二维码已重新生成'
    await showQRCode(currentTable.value)
  } catch (error) {
    errorMsg.value = error.message
  }
}

function downloadQR() {
  if (!qrCanvas.value || !currentTable.value) return

  const link = document.createElement('a')
  link.download = `餐桌${currentTable.value.table_no}-二维码.png`
  link.href = qrCanvas.value.toDataURL('image/png')
  link.click()
}

onMounted(() => {
  fetchTables()
})
</script>

<style scoped>
.table-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 20px;
  padding: 20px;
}

.table-card {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s, box-shadow 0.2s;
}

.table-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.table-card.disabled {
  opacity: 0.6;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.table-no {
  font-size: 24px;
  font-weight: 600;
  color: #333;
}

.table-status {
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.table-status.active {
  background: #e6f7e6;
  color: #52c41a;
}

.table-status.disabled {
  background: #f5f5f5;
  color: #999;
}

.table-info {
  margin-bottom: 12px;
}

.table-capacity {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.table-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.table-actions .btn-sm {
  padding: 6px 12px;
  font-size: 12px;
}

.qr-content {
  padding: 20px;
  text-align: center;
}

.qr-code-container {
  display: flex;
  justify-content: center;
  margin-bottom: 16px;
}

.qr-code-container canvas {
  border: 1px solid #eee;
  border-radius: 8px;
}

.qr-url {
  word-break: break-all;
  color: #666;
  font-size: 12px;
  margin-bottom: 8px;
  padding: 8px;
  background: #f5f5f5;
  border-radius: 4px;
}

.qr-tip {
  color: #999;
  font-size: 14px;
  margin: 0;
}

.message-success {
  background: #e6f7e6;
  color: #52c41a;
  padding: 12px 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}
</style>
