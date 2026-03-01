<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">首页概览</h1>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">本月收入</div>
        <div class="stat-value income">¥{{ formatMoney(stats.total_income || 0) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">本月支出</div>
        <div class="stat-value expense">¥{{ formatMoney(stats.total_expense || 0) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">本月结余</div>
        <div class="stat-value" :class="stats.balance >= 0 ? 'income' : 'expense'">
          ¥{{ formatMoney(stats.balance || 0) }}
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">账户总数</div>
        <div class="stat-value">{{ accountCount }}</div>
      </div>
    </div>

    <!-- 最近交易 -->
    <div class="card">
      <div class="card-title">最近交易记录</div>
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
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 账户列表 -->
    <div class="card">
      <div class="card-title">我的账户</div>
      <div v-if="accounts.length === 0" class="empty-state">
        暂无账户，请先添加账户
      </div>
      <div v-else class="stats-grid">
        <div v-for="account in accounts" :key="account.id" class="stat-card">
          <div class="stat-label">{{ account.name }} ({{ getAccountTypeName(account.type) }})</div>
          <div class="stat-value" :class="account.balance >= 0 ? 'income' : 'expense'">
            ¥{{ formatMoney(account.balance) }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const loading = ref(false)
const stats = ref({})
const transactions = ref([])
const accounts = ref([])
const accountCount = ref(0)

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

async function fetchData() {
  loading.value = true
  try {
    // 获取本月统计
    const now = new Date()
    const year = now.getFullYear()
    const month = now.getMonth() + 1
    const startDate = `${year}-${String(month).padStart(2, '0')}-01`
    const endDate = new Date(year, month, 0).toISOString().split('T')[0]

    const [statsRes, transRes, accountsRes] = await Promise.all([
      api.getStats(startDate, endDate),
      api.getTransactions({ page: 1, page_size: 10 }),
      api.getAccounts(1, 100)
    ])

    stats.value = statsRes.summary || {}
    transactions.value = transRes.transactions || []
    accounts.value = accountsRes.accounts || []
    accountCount.value = accountsRes.total || 0
  } catch (error) {
    console.error('获取数据失败:', error)
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>
