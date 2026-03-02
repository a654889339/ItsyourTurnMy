App({
  globalData: {
    baseUrl: 'http://106.54.50.88/api/v1/public', // 服务器地址 (通过80端口Nginx代理)
    token: '',
    tableInfo: null
  },

  onLaunch(options) {
    // 检查是否通过扫码进入
    if (options.query && options.query.token) {
      this.globalData.token = options.query.token
      this.loadTableInfo()
    }
  },

  onShow(options) {
    // 处理扫码场景
    if (options.query && options.query.token) {
      this.globalData.token = options.query.token
      this.loadTableInfo()
    }
  },

  // 加载餐桌信息
  loadTableInfo() {
    const that = this
    if (!this.globalData.token) return

    wx.request({
      url: `${this.globalData.baseUrl}/menu/${this.globalData.token}`,
      method: 'GET',
      success(res) {
        if (res.statusCode === 200) {
          that.globalData.tableInfo = res.data.table
          // 通知页面更新
          if (that.menuPageCallback) {
            that.menuPageCallback(res.data)
          }
        } else {
          wx.showToast({
            title: res.data.message || '加载失败',
            icon: 'none'
          })
        }
      },
      fail() {
        wx.showToast({
          title: '网络错误',
          icon: 'none'
        })
      }
    })
  },

  // 扫码获取token
  scanCode() {
    const that = this
    wx.scanCode({
      onlyFromCamera: false,
      scanType: ['qrCode'],
      success(res) {
        // 解析二维码内容，提取token
        // 假设二维码内容格式为: https://domain.com/scan/{token}
        const url = res.result
        const match = url.match(/\/scan\/([^\/\?]+)/)
        if (match && match[1]) {
          that.globalData.token = match[1]
          that.loadTableInfo()
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
  }
})
