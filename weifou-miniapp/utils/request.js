const { API_BASE } = require('./config');

function getToken() {
  return wx.getStorageSync('weifou_token') || '';
}

function setToken(token) {
  wx.setStorageSync('weifou_token', token);
}

function clearToken() {
  wx.removeStorageSync('weifou_token');
}

function request({ url, method = 'GET', data, header = {}, showLoading = false }) {
  return new Promise((resolve, reject) => {
    const token = getToken();
    if (showLoading) wx.showLoading({ title: '加载中', mask: true });
    wx.request({
      url: API_BASE + url,
      method,
      data,
      header: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...header,
      },
      timeout: 60000,
      success(res) {
        if (showLoading) wx.hideLoading();
        const body = res.data || {};
        if (res.statusCode === 401) {
          clearToken();
          return reject({ code: 'UNAUTHORIZED', message: '请重新登录' });
        }
        if (res.statusCode >= 200 && res.statusCode < 300 && body.success !== false) {
          return resolve(body.data);
        }
        reject({
          code: body.code || `HTTP_${res.statusCode}`,
          message: body.message || '请求失败',
        });
      },
      fail(err) {
        if (showLoading) wx.hideLoading();
        reject({ code: 'NETWORK_ERROR', message: err.errMsg || '网络异常' });
      },
    });
  });
}

module.exports = {
  request,
  getToken,
  setToken,
  clearToken,
};
