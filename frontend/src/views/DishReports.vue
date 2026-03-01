<template>
  <div class="dish-reports">
    <div class="page-header">
      <h1 class="page-title">菜品销售报表</h1>
      <div class="header-actions">
        <select v-model="period" class="form-select" @change="fetchReport">
          <option value="daily">今日</option>
          <option value="weekly">本周</option>
          <option value="monthly">本月</option>
          <option value="quarterly">本季度</option>
        </select>
      </div>
    </div>

    <div v-if="errorMsg" class="message message-error">{{ errorMsg }}</div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon orders">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14h-2v-2h2v2zm0-4h-2V7h2v6z"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ report.total_orders || 0 }}</div>
          <div class="stat-label">订单数</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon quantity">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M11 9h2v2h-2V9zm0 4h2v2h-2v-2zm-6-4h2v2H5V9zm0 4h2v2H5v-2zm12-4h2v2h-2V9zm0 4h2v2h-2v-2zm-6-8h2v2h-2V5zm0 12h2v2h-2v-2z"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ report.total_quantity || 0 }}</div>
          <div class="stat-label">售出数量</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon revenue">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M11.8 10.9c-2.27-.59-3-1.2-3-2.15 0-1.09 1.01-1.85 2.7-1.85 1.78 0 2.44.85 2.5 2.1h2.21c-.07-1.72-1.12-3.3-3.21-3.81V3h-3v2.16c-1.94.42-3.5 1.68-3.5 3.61 0 2.31 1.91 3.46 4.7 4.13 2.5.6 3 1.48 3 2.41 0 .69-.49 1.79-2.7 1.79-2.06 0-2.87-.92-2.98-2.1h-2.2c.12 2.19 1.76 3.42 3.68 3.83V21h3v-2.15c1.95-.37 3.5-1.5 3.5-3.55 0-2.84-2.43-3.81-4.7-4.4z"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">¥{{ formatMoney(report.total_revenue || 0) }}</div>
          <div class="stat-label">总收入</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon dishes">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M8.1 13.34l2.83-2.83L3.91 3.5c-1.56 1.56-1.56 4.09 0 5.66l4.19 4.18zm6.78-1.81c1.53.71 3.68.21 5.27-1.38 1.91-1.91 2.28-4.65.81-6.12-1.46-1.46-4.2-1.1-6.12.81-1.59 1.59-2.09 3.74-1.38 5.27L3.7 19.87l1.41 1.41L12 14.41l6.88 6.88 1.41-1.41L13.41 13l1.47-1.47z"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ report.dish_count || 0 }}</div>
          <div class="stat-label">上架菜品</div>
        </div>
      </div>
    </div>

    <div class="charts-row">
      <!-- 销售趋势图 -->
      <div class="card chart-card">
        <h3 class="card-title">销售趋势</h3>
        <div class="trend-controls">
          <select v-model="trendPeriod" class="form-select small" @change="fetchTrend">
            <option value="daily">按日</option>
            <option value="weekly">按周</option>
            <option value="monthly">按月</option>
          </select>
        </div>
        <div class="chart-container">
          <div v-if="trendLoading" class="loading">加载中...</div>
          <div v-else-if="trendData.length === 0" class="empty-chart">暂无数据</div>
          <div v-else class="bar-chart">
            <div class="chart-bars">
              <div v-for="(item, index) in trendData" :key="index" class="bar-item">
                <div class="bar-wrapper">
                  <div class="bar revenue-bar" :style="{ height: getBarHeight(item.total_revenue, maxRevenue) + '%' }">
                    <span class="bar-tooltip">¥{{ formatMoney(item.total_revenue) }}</span>
                  </div>
                </div>
                <div class="bar-label">{{ formatTrendLabel(item.date) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 分类占比 -->
      <div class="card chart-card">
        <h3 class="card-title">分类占比</h3>
        <div class="category-stats">
          <div v-if="loading" class="loading">加载中...</div>
          <div v-else-if="!report.category_stats || report.category_stats.length === 0" class="empty-chart">暂无数据</div>
          <div v-else>
            <div v-for="(cat, index) in report.category_stats" :key="index" class="category-item">
              <div class="category-header">
                <span class="category-name">{{ cat.category }}</span>
                <span class="category-value">¥{{ formatMoney(cat.revenue) }}</span>
              </div>
              <div class="category-bar-bg">
                <div class="category-bar" :style="{ width: cat.percentage + '%', backgroundColor: getCategoryColor(index) }"></div>
              </div>
              <div class="category-meta">
                <span>{{ cat.quantity }} 份</span>
                <span>{{ cat.percentage.toFixed(1) }}%</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 热销菜品排行 -->
    <div class="card">
      <h3 class="card-title">热销菜品 TOP10</h3>
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="!report.top_dishes || report.top_dishes.length === 0" class="empty-state">
        暂无销售数据
      </div>
      <table v-else class="table">
        <thead>
          <tr>
            <th style="width: 60px;">排名</th>
            <th>菜品名称</th>
            <th>分类</th>
            <th style="text-align: right;">销量</th>
            <th style="text-align: right;">收入</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(dish, index) in report.top_dishes" :key="dish.dish_id">
            <td>
              <span class="rank-badge" :class="'rank-' + (index + 1)">{{ index + 1 }}</span>
            </td>
            <td>{{ dish.dish_name }}</td>
            <td>{{ dish.category }}</td>
            <td style="text-align: right;">{{ dish.quantity }}</td>
            <td style="text-align: right; color: #f5222d; font-weight: 600;">¥{{ formatMoney(dish.revenue) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 每日明细 -->
    <div class="card">
      <h3 class="card-title">{{ getPeriodTitle() }}明细</h3>
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="!report.daily_stats || report.daily_stats.length === 0" class="empty-state">
        暂无数据
      </div>
      <table v-else class="table">
        <thead>
          <tr>
            <th>日期</th>
            <th style="text-align: right;">订单数</th>
            <th style="text-align: right;">销售数量</th>
            <th style="text-align: right;">收入</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="stat in report.daily_stats" :key="stat.date">
            <td>{{ stat.date }}</td>
            <td style="text-align: right;">{{ stat.order_count }}</td>
            <td style="text-align: right;">{{ stat.total_quantity }}</td>
            <td style="text-align: right; color: #f5222d; font-weight: 600;">¥{{ formatMoney(stat.total_revenue) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api'

const loading = ref(false)
const trendLoading = ref(false)
const errorMsg = ref('')
const period = ref('monthly')
const trendPeriod = ref('daily')
const report = ref({})
const trendData = ref([])

const categoryColors = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16']

function getCategoryColor(index) {
  return categoryColors[index % categoryColors.length]
}

function formatMoney(value) {
  return Number(value || 0).toFixed(2)
}

function formatTrendLabel(date) {
  if (!date) return ''
  if (date.length === 7) return date.substring(5) // 2026-03 -> 03
  if (date.length === 10) return date.substring(5) // 2026-03-01 -> 03-01
  return date
}

const maxRevenue = computed(() => {
  if (!trendData.value || trendData.value.length === 0) return 0
  return Math.max(...trendData.value.map(d => d.total_revenue || 0))
})

function getBarHeight(value, max) {
  if (max === 0) return 0
  return Math.max(5, (value / max) * 100)
}

function getPeriodTitle() {
  const titles = {
    daily: '今日',
    weekly: '本周',
    monthly: '本月',
    quarterly: '本季度'
  }
  return titles[period.value] || ''
}

async function fetchReport() {
  loading.value = true
  errorMsg.value = ''
  try {
    report.value = await api.getDishReport({ period: period.value })
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    loading.value = false
  }
}

async function fetchTrend() {
  trendLoading.value = true
  try {
    const countMap = { daily: 14, weekly: 8, monthly: 6 }
    const res = await api.getDishReportTrend({
      period: trendPeriod.value,
      count: countMap[trendPeriod.value] || 7
    })
    trendData.value = res.data || []
  } catch (error) {
    console.error('获取趋势数据失败:', error)
  } finally {
    trendLoading.value = false
  }
}

onMounted(() => {
  fetchReport()
  fetchTrend()
})
</script>

<style scoped>
.dish-reports {
  padding-bottom: 40px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon svg {
  width: 28px;
  height: 28px;
}

.stat-icon.orders {
  background: #e6f7ff;
  color: #1890ff;
}

.stat-icon.quantity {
  background: #f6ffed;
  color: #52c41a;
}

.stat-icon.revenue {
  background: #fff2e8;
  color: #fa541c;
}

.stat-icon.dishes {
  background: #f9f0ff;
  color: #722ed1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #333;
  line-height: 1.2;
}

.stat-label {
  font-size: 14px;
  color: #999;
  margin-top: 4px;
}

.charts-row {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 20px;
  margin-bottom: 24px;
}

@media (max-width: 1000px) {
  .charts-row {
    grid-template-columns: 1fr;
  }
}

.chart-card {
  padding: 20px;
  position: relative;
}

.card-title {
  margin: 0 0 16px;
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.trend-controls {
  position: absolute;
  top: 20px;
  right: 20px;
}

.form-select.small {
  padding: 4px 24px 4px 8px;
  font-size: 12px;
}

.chart-container {
  height: 250px;
  display: flex;
  align-items: flex-end;
}

.empty-chart {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
}

.bar-chart {
  width: 100%;
  height: 100%;
}

.chart-bars {
  display: flex;
  align-items: flex-end;
  justify-content: space-around;
  height: 100%;
  padding-bottom: 30px;
}

.bar-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  max-width: 60px;
}

.bar-wrapper {
  height: 180px;
  width: 100%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.bar {
  width: 24px;
  border-radius: 4px 4px 0 0;
  transition: height 0.3s ease;
  position: relative;
  cursor: pointer;
}

.revenue-bar {
  background: linear-gradient(180deg, #1890ff 0%, #69c0ff 100%);
}

.bar-tooltip {
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.75);
  color: #fff;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  white-space: nowrap;
  opacity: 0;
  transition: opacity 0.2s;
  pointer-events: none;
}

.bar:hover .bar-tooltip {
  opacity: 1;
}

.bar-label {
  margin-top: 8px;
  font-size: 11px;
  color: #999;
  text-align: center;
}

.category-stats {
  min-height: 200px;
}

.category-item {
  margin-bottom: 16px;
}

.category-item:last-child {
  margin-bottom: 0;
}

.category-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
}

.category-name {
  font-size: 14px;
  color: #333;
}

.category-value {
  font-size: 14px;
  font-weight: 600;
  color: #f5222d;
}

.category-bar-bg {
  height: 8px;
  background: #f0f0f0;
  border-radius: 4px;
  overflow: hidden;
}

.category-bar {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.category-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 12px;
  color: #999;
}

.rank-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 600;
  background: #f0f0f0;
  color: #666;
}

.rank-badge.rank-1 {
  background: linear-gradient(135deg, #ffd700 0%, #ffb800 100%);
  color: #fff;
}

.rank-badge.rank-2 {
  background: linear-gradient(135deg, #c0c0c0 0%, #a0a0a0 100%);
  color: #fff;
}

.rank-badge.rank-3 {
  background: linear-gradient(135deg, #cd7f32 0%, #b87333 100%);
  color: #fff;
}
</style>
