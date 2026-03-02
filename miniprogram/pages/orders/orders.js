const api = require('../../utils/api')

Page({
  data: {
    token: '',
    tableInfo: null,
    orders: [],
    loading: false,
    showDetail: false,
    currentOrder: null
  },

  onLoad(options) {
    // 尝试从全局获取token
    const app = getApp()
    if (app.globalData.token) {
      this.setData({
        token: app.globalData.token,
        tableInfo: app.globalData.tableInfo
      })
      this.loadOrders()
    }
  },

  onShow() {
    // 每次显示时检查token是否更新
    const app = getApp()
    if (app.globalData.token && app.globalData.token !== this.data.token) {
      this.setData({
        token: app.globalData.token,
        tableInfo: app.globalData.tableInfo
      })
      this.loadOrders()
    } else if (this.data.token) {
      // 刷新订单
      this.loadOrders()
    }
  },

  onPullDownRefresh() {
    if (this.data.token) {
      this.loadOrders().then(() => {
        wx.stopPullDownRefresh()
      })
    } else {
      wx.stopPullDownRefresh()
    }
  },

  // 扫码
  scanQRCode() {
    wx.scanCode({
      onlyFromCamera: false,
      scanType: ['qrCode'],
      success: (res) => {
        // 解析二维码内容，获取token
        const url = res.result
        const token = this.parseToken(url)
        if (token) {
          const app = getApp()
          app.globalData.token = token
          this.setData({ token })
          this.loadTableInfo()
          this.loadOrders()
        } else {
          wx.showToast({
            title: '无效的二维码',
            icon: 'none'
          })
        }
      },
      fail: () => {
        wx.showToast({
          title: '扫码失败',
          icon: 'none'
        })
      }
    })
  },

  // 解析token
  parseToken(url) {
    try {
      // URL格式：https://domain.com/scan/TOKEN 或直接是 TOKEN
      if (url.includes('/scan/')) {
        const parts = url.split('/scan/')
        return parts[1].split('?')[0].split('/')[0]
      }
      // 可能直接是token
      if (url.length > 20 && !url.includes('://')) {
        return url
      }
      return null
    } catch (e) {
      return null
    }
  },

  // 加载餐桌信息
  async loadTableInfo() {
    try {
      const res = await api.getMenu(this.data.token)
      if (res.table) {
        const tableInfo = {
          tableNo: res.table.table_no,
          capacity: res.table.capacity
        }
        const app = getApp()
        app.globalData.tableInfo = tableInfo
        this.setData({ tableInfo })
      }
    } catch (e) {
      console.error('加载餐桌信息失败', e)
    }
  },

  // 加载订单列表
  async loadOrders() {
    if (!this.data.token) return

    this.setData({ loading: true })
    try {
      const res = await api.getTableOrders(this.data.token)
      let orders = res.orders || []

      // 处理订单数据，添加摘要信息
      orders = orders.map(order => {
        const items = order.items || []
        // 菜品名称摘要
        const dishNames = items.map(item => item.dish_name).join('、')
        const dishSummary = dishNames.length > 30 ? dishNames.substring(0, 30) + '...' : dishNames
        // 菜品总数
        const itemCount = items.reduce((sum, item) => sum + item.quantity, 0)

        return {
          ...order,
          dishSummary: dishSummary || '无菜品',
          itemCount: itemCount
        }
      })

      this.setData({
        orders: orders,
        loading: false
      })
    } catch (e) {
      console.error('加载订单失败', e)
      this.setData({ loading: false })
      wx.showToast({
        title: '加载订单失败',
        icon: 'none'
      })
    }
  },

  // 显示订单详情
  showOrderDetail(e) {
    const order = e.currentTarget.dataset.order
    this.setData({
      currentOrder: order,
      showDetail: true
    })
  },

  // 隐藏订单详情
  hideOrderDetail() {
    this.setData({
      showDetail: false,
      currentOrder: null
    })
  },

  // 阻止事件冒泡
  stopPropagation() {},

  // 预览菜品图片
  previewDishImage(e) {
    const image = e.currentTarget.dataset.image
    if (image) {
      wx.previewImage({
        urls: [image],
        current: image
      })
    }
  },

  // 跳转到点餐页
  goToMenu() {
    wx.switchTab({
      url: '/pages/menu/menu'
    })
  }
})
