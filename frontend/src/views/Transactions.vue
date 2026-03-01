<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">收支记录</h1>
      <button class="btn btn-primary" @click="showModal = true">添加记录</button>
    </div>

    <div v-if="errorMsg" class="message message-error">{{ errorMsg }}</div>

    <!-- 筛选条件 -->
    <div class="card" style="margin-bottom: 16px">
      <div style="display: flex; gap: 16px; flex-wrap: wrap">
        <div class="form-group" style="margin-bottom: 0; min-width: 150px">
          <label class="form-label">类型</label>
          <select v-model="filters.type" class="form-select" @change="fetchTransactions">
            <option value="">全部</option>
            <option value="income">收入</option>
            <option value="expense">支出</option>
          </select>
        </div>
        <div class="form-group" style="margin-bottom: 0; min-width: 150px">
          <label class="form-label">账户</label>
          <select v-model="filters.account_id" class="form-select" @change="fetchTransactions">
            <option value="">全部账户</option>
            <option v-for="acc in accounts" :key="acc.id" :value="acc.id">
              {{ acc.name }}
            </option>
          </select>
        </div>
        <div class="form-group" style="margin-bottom: 0; min-width: 150px">
          <label class="form-label">开始日期</label>
          <input v-model="filters.start_date" type="date" class="form-input" @change="fetchTransactions" />
        </div>
        <div class="form-group" style="margin-bottom: 0; min-width: 150px">
          <label class="form-label">结束日期</label>
          <input v-model="filters.end_date" type="date" class="form-input" @change="fetchTransactions" />
        </div>
      </div>
    </div>

    <div class="card">
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="transactions.length === 0" class="empty-state">
        暂无交易记录
      </div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>日期</th>
            <th>类型</th>
            <th>分类</th>
            <th>金额</th>
            <th>描述</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in transactions" :key="item.id">
            <td>{{ formatDate(item.transaction_date) }}</td>
            <td>
              <span :class="['tag', item.type === 'income' ? 'tag-income' : 'tag-expense']">
                {{ item.type === 'income' ? '收入' : '支出' }}
              </span>
            </td>
            <td>{{ item.category_name || '-' }}</td>
            <td :style="{ color: item.type === 'income' ? '#52c41a' : '#f5222d' }">
              {{ item.type === 'income' ? '+' : '-' }}¥{{ formatMoney(item.amount) }}
            </td>
            <td>{{ item.description || '-' }}</td>
            <td>
              <button class="btn btn-default" style="margin-right: 8px" @click="editTransaction(item)">
                编辑
              </button>
              <button class="btn btn-danger" @click="deleteTransaction(item)">
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- 分页 -->
      <div v-if="total > pageSize" class="pagination">
        <button :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
        <span style="padding: 6px 12px">第 {{ page }} 页 / 共 {{ Math.ceil(total / pageSize) }} 页</span>
        <button :disabled="page >= Math.ceil(total / pageSize)" @click="changePage(page + 1)">下一页</button>
      </div>
    </div>

    <!-- 添加/编辑交易模态框 -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <div class="modal-header">
          <h3 class="modal-title">{{ editingTransaction ? '编辑记录' : '添加记录' }}</h3>
          <button class="modal-close" @click="closeModal">&times;</button>
        </div>

        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label class="form-label">类型</label>
            <select v-model="form.type" class="form-select" required @change="onTypeChange">
              <option value="income">收入</option>
              <option value="expense">支出</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">账户</label>
            <select v-model="form.account_id" class="form-select" required>
              <option v-for="acc in accounts" :key="acc.id" :value="acc.id">
                {{ acc.name }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">分类</label>
            <select v-model="form.category_id" class="form-select">
              <option value="">请选择分类</option>
              <option v-for="cat in filteredCategories" :key="cat.id" :value="cat.id">
                {{ cat.name }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">金额</label>
            <input v-model.number="form.amount" type="number" step="0.01" min="0.01" class="form-input" required />
          </div>

          <div class="form-group">
            <label class="form-label">日期</label>
            <input v-model="form.transaction_date" type="date" class="form-input" required />
          </div>

          <div class="form-group">
            <label class="form-label">描述</label>
            <input v-model="form.description" type="text" class="form-input" placeholder="可选" />
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
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import api from '../api'

const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const errorMsg = ref('')
const transactions = ref([])
const accounts = ref([])
const categories = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const editingTransaction = ref(null)

const filters = reactive({
  type: '',
  account_id: '',
  start_date: '',
  end_date: ''
})

const form = reactive({
  type: 'expense',
  account_id: '',
  category_id: '',
  amount: '',
  transaction_date: new Date().toISOString().split('T')[0],
  description: ''
})

const filteredCategories = computed(() => {
  return categories.value.filter(c => c.type === form.type)
})

function formatMoney(value) {
  return Number(value).toFixed(2)
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN')
}

async function fetchTransactions() {
  loading.value = true
  errorMsg.value = ''
  try {
    const params = {
      page: page.value,
      page_size: pageSize.value,
      ...filters
    }
    const res = await api.getTransactions(params)
    transactions.value = res.transactions || []
    total.value = res.total || 0
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    loading.value = false
  }
}

async function fetchAccounts() {
  try {
    const res = await api.getAccounts(1, 100)
    accounts.value = res.accounts || []
    if (accounts.value.length > 0 && !form.account_id) {
      form.account_id = accounts.value[0].id
    }
  } catch (error) {
    console.error('获取账户失败:', error)
  }
}

async function fetchCategories() {
  try {
    const res = await api.getCategories()
    categories.value = res.categories || []
  } catch (error) {
    console.error('获取分类失败:', error)
  }
}

function changePage(newPage) {
  page.value = newPage
  fetchTransactions()
}

function onTypeChange() {
  form.category_id = ''
}

function editTransaction(item) {
  editingTransaction.value = item
  form.type = item.type
  form.account_id = item.account_id
  form.category_id = item.category_id || ''
  form.amount = item.amount
  form.transaction_date = item.transaction_date.split('T')[0]
  form.description = item.description
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingTransaction.value = null
  form.type = 'expense'
  form.account_id = accounts.value.length > 0 ? accounts.value[0].id : ''
  form.category_id = ''
  form.amount = ''
  form.transaction_date = new Date().toISOString().split('T')[0]
  form.description = ''
}

async function handleSubmit() {
  submitting.value = true
  errorMsg.value = ''
  try {
    const data = {
      account_id: Number(form.account_id),
      type: form.type,
      amount: Number(form.amount),
      category_id: form.category_id ? Number(form.category_id) : 0,
      description: form.description,
      transaction_date: form.transaction_date
    }

    if (editingTransaction.value) {
      await api.updateTransaction(editingTransaction.value.id, data)
    } else {
      await api.createTransaction(data)
    }
    closeModal()
    fetchTransactions()
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    submitting.value = false
  }
}

async function deleteTransaction(item) {
  if (!confirm('确定要删除这条记录吗？')) return

  errorMsg.value = ''
  try {
    await api.deleteTransaction(item.id)
    fetchTransactions()
  } catch (error) {
    errorMsg.value = error.message
  }
}

onMounted(() => {
  fetchAccounts()
  fetchCategories()
  fetchTransactions()
})
</script>
