const api = require('../../utils/api')
const app = getApp()

Page({
  data: {
    token: '',
    tableInfo: null,
    categories: [],
    dishes: [],
    filteredDishes: [],  // 过滤后的菜品
    selectedCategory: '',
    cart: [], // [{dish, quantity, remark}]
    cartCount: 0,        // 购物车数量
    cartTotal: '0.00',   // 购物车总价
    customerName: '',
    orderRemark: '',
    showCartModal: false,
    submitting: false
  },

  onLoad(options) {
    // 处理扫码进入的token
    if (options.token) {
      this.setData({ token: options.token })
      app.globalData.token = options.token
      this.loadMenu()
    } else if (app.globalData.token) {
      this.setData({ token: app.globalData.token })
      this.loadMenu()
    }

    // 注册回调
    app.menuPageCallback = (data) => {
      const dishes = data.dishes || []
      this.setData({
        tableInfo: data.table,
        categories: data.categories || [],
        dishes: dishes,
        filteredDishes: dishes
      })
    }
  },

  onShow() {
    if (app.globalData.token && !this.data.token) {
      this.setData({ token: app.globalData.token })
      this.loadMenu()
    }
  },

  // 扫码
  scanCode() {
    const that = this
    wx.scanCode({
      onlyFromCamera: false,
      scanType: ['qrCode'],
      success(res) {
        // 解析二维码内容，提取token
        const url = res.result
        let token = null

        // URL格式：https://domain.com/scan/{token}
        if (url.includes('/scan/')) {
          const match = url.match(/\/scan\/([^\/\?]+)/)
          if (match && match[1]) {
            token = match[1]
          }
        } else if (url.length > 20 && !url.includes('://')) {
          // 可能直接是token
          token = url
        }

        if (token) {
          app.globalData.token = token
          that.setData({ token })
          that.loadMenu()
        } else {
          wx.showToast({
            title: '无效的二维码',
            icon: 'none'
          })
        }
      },
      fail() {
        wx.showToast({
          title: '扫码取消',
          icon: 'none'
        })
      }
    })
  },

  // 加载菜单
  async loadMenu() {
    if (!this.data.token) return

    wx.showLoading({ title: '加载中' })
    try {
      const res = await api.getMenu(this.data.token)
      const dishes = res.dishes || []
      this.setData({
        tableInfo: res.table,
        categories: res.categories || [],
        dishes: dishes,
        filteredDishes: dishes,
        selectedCategory: ''
      })
    } catch (err) {
      wx.showToast({ title: err.message, icon: 'none' })
    } finally {
      wx.hideLoading()
    }
  },

  // 选择分类
  selectCategory(e) {
    const selectedCategory = e.currentTarget.dataset.category
    this.setData({ selectedCategory })
    this.updateFilteredDishes()
  },

  // 更新过滤后的菜品
  updateFilteredDishes() {
    const { dishes, selectedCategory } = this.data
    let filteredDishes = dishes
    if (selectedCategory) {
      filteredDishes = dishes.filter(d => d.category === selectedCategory)
    }
    this.setData({ filteredDishes })
  },

  // 更新购物车统计
  updateCartStats() {
    const { cart } = this.data
    const cartCount = cart.reduce((sum, item) => sum + item.quantity, 0)
    const cartTotal = cart.reduce((sum, item) => sum + item.dish.price * item.quantity, 0).toFixed(2)
    this.setData({ cartCount, cartTotal })
  },

  // 添加到购物车
  addToCart(e) {
    const dish = e.currentTarget.dataset.dish
    const cart = [...this.data.cart]
    const existingIndex = cart.findIndex(c => c.dish.id === dish.id)

    if (existingIndex >= 0) {
      cart[existingIndex].quantity++
    } else {
      cart.push({ dish, quantity: 1, remark: '' })
    }

    this.setData({ cart })
    this.updateCartStats()
  },

  // 减少购物车数量
  decreaseCart(e) {
    const dish = e.currentTarget.dataset.dish
    let cart = [...this.data.cart]
    const existingIndex = cart.findIndex(c => c.dish.id === dish.id)

    if (existingIndex >= 0) {
      if (cart[existingIndex].quantity > 1) {
        cart[existingIndex].quantity--
      } else {
        cart.splice(existingIndex, 1)
      }
      this.setData({ cart })
      this.updateCartStats()
    }
  },

  // 清空购物车
  clearCart() {
    this.setData({ cart: [], cartCount: 0, cartTotal: '0.00', showCartModal: false })
  },

  // 更新菜品备注
  updateItemRemark(e) {
    const dishId = e.currentTarget.dataset.id
    const remark = e.detail.value
    const cart = [...this.data.cart]
    const item = cart.find(c => c.dish.id === dishId)
    if (item) {
      item.remark = remark
      this.setData({ cart })
    }
  },

  // 显示购物车
  showCart() {
    this.setData({ showCartModal: true })
  },

  // 隐藏购物车
  hideCart() {
    this.setData({ showCartModal: false })
  },

  // 阻止事件冒泡
  stopPropagation() {},

  // 输入称呼
  onNameInput(e) {
    this.setData({ customerName: e.detail.value })
  },

  // 输入备注
  onRemarkInput(e) {
    this.setData({ orderRemark: e.detail.value })
  },

  // 提交订单
  async submitOrder() {
    if (this.data.cart.length === 0) {
      wx.showToast({ title: '购物车是空的', icon: 'none' })
      return
    }

    this.setData({ submitting: true })

    try {
      const orderData = {
        customer_name: this.data.customerName,
        items: this.data.cart.map(item => ({
          dish_id: item.dish.id,
          quantity: item.quantity,
          remark: item.remark || ''
        })),
        remark: this.data.orderRemark
      }

      await api.createOrder(this.data.token, orderData)

      wx.showToast({ title: '下单成功', icon: 'success' })

      // 清空购物车
      this.setData({
        cart: [],
        cartCount: 0,
        cartTotal: '0.00',
        customerName: '',
        orderRemark: '',
        showCartModal: false
      })

      // 刷新菜单(更新库存)
      this.loadMenu()

      // 跳转到订单页面
      wx.switchTab({ url: '/pages/orders/orders' })
    } catch (err) {
      wx.showToast({ title: err.message, icon: 'none' })
    } finally {
      this.setData({ submitting: false })
    }
  }
})
