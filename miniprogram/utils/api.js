const app = getApp()

// 获取菜单
function getMenu(token) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${app.globalData.baseUrl}/menu/${token}`,
      method: 'GET',
      success(res) {
        if (res.statusCode === 200) {
          resolve(res.data)
        } else {
          reject(new Error(res.data.message || '获取菜单失败'))
        }
      },
      fail(err) {
        reject(new Error('网络错误'))
      }
    })
  })
}

// 创建订单
function createOrder(token, data) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${app.globalData.baseUrl}/order/${token}`,
      method: 'POST',
      data: data,
      header: {
        'Content-Type': 'application/json'
      },
      success(res) {
        if (res.statusCode === 200) {
          resolve(res.data)
        } else {
          reject(new Error(res.data.message || '下单失败'))
        }
      },
      fail(err) {
        reject(new Error('网络错误'))
      }
    })
  })
}

// 获取本桌订单
function getTableOrders(token) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${app.globalData.baseUrl}/orders/${token}`,
      method: 'GET',
      success(res) {
        if (res.statusCode === 200) {
          resolve(res.data)
        } else {
          reject(new Error(res.data.message || '获取订单失败'))
        }
      },
      fail(err) {
        reject(new Error('网络错误'))
      }
    })
  })
}

// 获取订单状态
function getOrderStatus(token, orderNo) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${app.globalData.baseUrl}/order/${token}/${orderNo}`,
      method: 'GET',
      success(res) {
        if (res.statusCode === 200) {
          resolve(res.data)
        } else {
          reject(new Error(res.data.message || '获取订单状态失败'))
        }
      },
      fail(err) {
        reject(new Error('网络错误'))
      }
    })
  })
}

module.exports = {
  getMenu,
  createOrder,
  getTableOrders,
  getOrderStatus
}
