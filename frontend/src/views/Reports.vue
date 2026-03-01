<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">统计报表</h1>
      <div style="display: flex; gap: 8px">
        <select v-model="selectedYear" class="form-select" style="width: auto" @change="fetchReport">
          <option v-for="y in years" :key="y" :value="y">{{ y }}年</option>
        </select>
        <select v-model="selectedMonth" class="form-select" style="width: auto" @change="fetchReport">
          <option v-for="m in 12" :key="m" :value="m">{{ m }}月</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="card">
      <div class="loading">加载中...</div>
    </div>

    <template v-else>
      <!-- 月度概览 -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-label">本月收入</div>
          <div class="stat-value income">¥{{ formatMoney(report.Summary?.total_income || 0) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">本月支出</div>
          <div class="stat-value expense">¥{{ formatMoney(report.Summary?.total_expense || 0) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">本月结余</div>
          <div class="stat-value" :class="(report.Summary?.balance || 0) >= 0 ? 'income' : 'expense'">
            ¥{{ formatMoney(report.Summary?.balance || 0) }}
          </div>
        </div>
      </div>

      <!-- 图表区域 -->
      <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 16px">
        <!-- 每日收支趋势 -->
        <div class="card">
          <div class="card-title">每日收支趋势</div>
          <div ref="dailyChartRef" class="chart-container"></div>
        </div>

        <!-- 支出分类占比 -->
        <div class="card">
          <div class="card-title">支出分类占比</div>
          <div ref="expenseChartRef" class="chart-container"></div>
        </div>
      </div>

      <!-- 分类明细 -->
      <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 16px; margin-top: 16px">
        <!-- 收入分类 -->
        <div class="card">
          <div class="card-title">收入分类</div>
          <div v-if="!report.IncomeByCategory?.length" class="empty-state">暂无数据</div>
          <table v-else class="table">
            <thead>
              <tr>
                <th>分类</th>
                <th>金额</th>
                <th>占比</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in report.IncomeByCategory" :key="item.category_id">
                <td>{{ item.category_name }}</td>
                <td style="color: #52c41a">¥{{ formatMoney(item.amount) }}</td>
                <td>{{ item.percentage.toFixed(1) }}%</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 支出分类 -->
        <div class="card">
          <div class="card-title">支出分类</div>
          <div v-if="!report.ExpenseByCategory?.length" class="empty-state">暂无数据</div>
          <table v-else class="table">
            <thead>
              <tr>
                <th>分类</th>
                <th>金额</th>
                <th>占比</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in report.ExpenseByCategory" :key="item.category_id">
                <td>{{ item.category_name }}</td>
                <td style="color: #f5222d">¥{{ formatMoney(item.amount) }}</td>
                <td>{{ item.percentage.toFixed(1) }}%</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'
import * as echarts from 'echarts'
import api from '../api'

const loading = ref(false)
const report = ref({})
const dailyChartRef = ref(null)
const expenseChartRef = ref(null)

let dailyChart = null
let expenseChart = null

const now = new Date()
const selectedYear = ref(now.getFullYear())
const selectedMonth = ref(now.getMonth() + 1)

const years = []
for (let i = now.getFullYear(); i >= now.getFullYear() - 5; i--) {
  years.push(i)
}

function formatMoney(value) {
  return Number(value).toFixed(2)
}

async function fetchReport() {
  loading.value = true
  try {
    const res = await api.getMonthlyReport(selectedYear.value, selectedMonth.value)
    report.value = res
    await nextTick()
    renderCharts()
  } catch (error) {
    console.error('获取报表失败:', error)
  } finally {
    loading.value = false
  }
}

function renderCharts() {
  renderDailyChart()
  renderExpenseChart()
}

function renderDailyChart() {
  if (!dailyChartRef.value) return

  if (!dailyChart) {
    dailyChart = echarts.init(dailyChartRef.value)
  }

  const dailyStats = report.value.DailyStats || []
  const dates = dailyStats.map(d => d.date)
  const incomes = dailyStats.map(d => d.income || 0)
  const expenses = dailyStats.map(d => d.expense || 0)

  dailyChart.setOption({
    tooltip: {
      trigger: 'axis'
    },
    legend: {
      data: ['收入', '支出']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: dates,
      axisLabel: {
        formatter: (value) => value.slice(5)
      }
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '收入',
        type: 'bar',
        data: incomes,
        itemStyle: { color: '#52c41a' }
      },
      {
        name: '支出',
        type: 'bar',
        data: expenses,
        itemStyle: { color: '#f5222d' }
      }
    ]
  })
}

function renderExpenseChart() {
  if (!expenseChartRef.value) return

  if (!expenseChart) {
    expenseChart = echarts.init(expenseChartRef.value)
  }

  const expenseByCategory = report.value.ExpenseByCategory || []
  const data = expenseByCategory.map(item => ({
    name: item.category_name,
    value: item.amount
  }))

  expenseChart.setOption({
    tooltip: {
      trigger: 'item',
      formatter: '{b}: ¥{c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        type: 'pie',
        radius: '60%',
        data: data,
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  })
}

// 窗口大小变化时重新调整图表
function handleResize() {
  dailyChart?.resize()
  expenseChart?.resize()
}

onMounted(() => {
  fetchReport()
  window.addEventListener('resize', handleResize)
})
</script>
