import axios from 'axios'

const instance = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
instance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
instance.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    if (error.response) {
      if (error.response.status === 401) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        window.location.href = '/login'
      }
      throw new Error(error.response.data.message || '请求失败')
    }
    throw error
  }
)

const api = {
  // 认证相关
  login(username, password) {
    return instance.post('/auth/login', { username, password })
  },

  sendVerificationCode(email) {
    return instance.post('/auth/send-code', { email })
  },

  register(username, password, email, code) {
    return instance.post('/auth/register', { username, password, email, code })
  },

  getCurrentUser() {
    return instance.get('/auth/me')
  },

  // 账户相关
  getAccounts(page = 1, pageSize = 20) {
    return instance.get('/accounts', { params: { page, page_size: pageSize } })
  },

  getAccount(id) {
    return instance.get(`/accounts/${id}`)
  },

  createAccount(data) {
    return instance.post('/accounts', data)
  },

  updateAccount(id, data) {
    return instance.put(`/accounts/${id}`, data)
  },

  deleteAccount(id) {
    return instance.delete(`/accounts/${id}`)
  },

  // 交易相关
  getTransactions(params = {}) {
    return instance.get('/transactions', { params })
  },

  getTransaction(id) {
    return instance.get(`/transactions/${id}`)
  },

  createTransaction(data) {
    return instance.post('/transactions', data)
  },

  updateTransaction(id, data) {
    return instance.put(`/transactions/${id}`, data)
  },

  deleteTransaction(id) {
    return instance.delete(`/transactions/${id}`)
  },

  // 分类相关
  getCategories(type = '') {
    return instance.get('/categories', { params: { type } })
  },

  createCategory(data) {
    return instance.post('/categories', data)
  },

  updateCategory(id, data) {
    return instance.put(`/categories/${id}`, data)
  },

  deleteCategory(id) {
    return instance.delete(`/categories/${id}`)
  },

  // 报表相关
  getStats(startDate = '', endDate = '') {
    return instance.get('/reports/stats', { params: { start_date: startDate, end_date: endDate } })
  },

  getMonthlyReport(year, month) {
    return instance.get('/reports/monthly', { params: { year, month } })
  },

  // 菜品相关
  getDishes(params = {}) {
    return instance.get('/dishes', { params })
  },

  getDish(id) {
    return instance.get(`/dishes/${id}`)
  },

  createDish(data) {
    return instance.post('/dishes', data)
  },

  updateDish(id, data) {
    return instance.put(`/dishes/${id}`, data)
  },

  deleteDish(id) {
    return instance.delete(`/dishes/${id}`)
  },

  getDishCategories() {
    return instance.get('/dishes/categories')
  },

  // 订单相关
  getOrders(params = {}) {
    return instance.get('/orders', { params })
  },

  getOrder(id) {
    return instance.get(`/orders/${id}`)
  },

  createOrder(data) {
    return instance.post('/orders', data)
  },

  updateOrderStatus(id, status) {
    return instance.put(`/orders/${id}/status`, { status })
  },

  deleteOrder(id) {
    return instance.delete(`/orders/${id}`)
  },

  // 上传相关
  uploadImage(file) {
    const formData = new FormData()
    formData.append('image', file)
    return instance.post('/upload/image', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  uploadBase64Image(base64Data) {
    return instance.post('/upload/image', { image: base64Data })
  }
}

export default api
