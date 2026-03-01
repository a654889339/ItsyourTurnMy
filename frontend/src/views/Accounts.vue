<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">账户管理</h1>
      <button class="btn btn-primary" @click="showModal = true">添加账户</button>
    </div>

    <div v-if="errorMsg" class="message message-error">{{ errorMsg }}</div>

    <div class="card">
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="accounts.length === 0" class="empty-state">
        暂无账户，点击上方按钮添加
      </div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>账户名称</th>
            <th>类型</th>
            <th>余额</th>
            <th>币种</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="account in accounts" :key="account.id">
            <td>{{ account.name }}</td>
            <td>{{ getAccountTypeName(account.type) }}</td>
            <td :style="{ color: account.balance >= 0 ? '#52c41a' : '#f5222d' }">
              ¥{{ formatMoney(account.balance) }}
            </td>
            <td>{{ account.currency }}</td>
            <td>{{ formatDate(account.created_at) }}</td>
            <td>
              <button class="btn btn-default" style="margin-right: 8px" @click="editAccount(account)">
                编辑
              </button>
              <button class="btn btn-danger" @click="deleteAccount(account)">
                删除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 添加/编辑账户模态框 -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <div class="modal-header">
          <h3 class="modal-title">{{ editingAccount ? '编辑账户' : '添加账户' }}</h3>
          <button class="modal-close" @click="closeModal">&times;</button>
        </div>

        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label class="form-label">账户名称</label>
            <input v-model="form.name" type="text" class="form-input" required />
          </div>

          <div class="form-group">
            <label class="form-label">账户类型</label>
            <select v-model="form.type" class="form-select" required>
              <option value="cash">现金</option>
              <option value="bank">银行卡</option>
              <option value="credit">信用卡</option>
              <option value="investment">投资账户</option>
            </select>
          </div>

          <div v-if="!editingAccount" class="form-group">
            <label class="form-label">初始余额</label>
            <input v-model.number="form.initial_balance" type="number" step="0.01" class="form-input" />
          </div>

          <div v-if="!editingAccount" class="form-group">
            <label class="form-label">币种</label>
            <select v-model="form.currency" class="form-select">
              <option value="CNY">人民币 (CNY)</option>
              <option value="USD">美元 (USD)</option>
              <option value="EUR">欧元 (EUR)</option>
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'

const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const errorMsg = ref('')
const accounts = ref([])
const editingAccount = ref(null)

const form = reactive({
  name: '',
  type: 'bank',
  initial_balance: 0,
  currency: 'CNY'
})

const accountTypes = {
  cash: '现金',
  bank: '银行卡',
  credit: '信用卡',
  investment: '投资账户'
}

function getAccountTypeName(type) {
  return accountTypes[type] || type
}

function formatMoney(value) {
  return Number(value).toFixed(2)
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN')
}

async function fetchAccounts() {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await api.getAccounts(1, 100)
    accounts.value = res.accounts || []
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    loading.value = false
  }
}

function editAccount(account) {
  editingAccount.value = account
  form.name = account.name
  form.type = account.type
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingAccount.value = null
  form.name = ''
  form.type = 'bank'
  form.initial_balance = 0
  form.currency = 'CNY'
}

async function handleSubmit() {
  submitting.value = true
  errorMsg.value = ''
  try {
    if (editingAccount.value) {
      await api.updateAccount(editingAccount.value.id, {
        name: form.name,
        type: form.type
      })
    } else {
      await api.createAccount({
        name: form.name,
        type: form.type,
        initial_balance: form.initial_balance,
        currency: form.currency
      })
    }
    closeModal()
    fetchAccounts()
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    submitting.value = false
  }
}

async function deleteAccount(account) {
  if (!confirm(`确定要删除账户 "${account.name}" 吗？`)) return

  errorMsg.value = ''
  try {
    await api.deleteAccount(account.id)
    fetchAccounts()
  } catch (error) {
    errorMsg.value = error.message
  }
}

onMounted(fetchAccounts)
</script>
