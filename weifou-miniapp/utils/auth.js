const { request, setToken, getToken } = require('./request');

/**
 * 静默登录：仅换 token，不强制要求昵称头像。
 * 用于访客匿名问答前的授权。
 */
function login({ nickname, avatarUrl } = {}) {
  return new Promise((resolve, reject) => {
    wx.login({
      success: async (res) => {
        if (!res.code) return reject({ code: 'WX_LOGIN_FAILED', message: '微信登录失败' });
        try {
          const data = await request({
            url: '/auth/login',
            method: 'POST',
            data: { code: res.code, nickname, avatarUrl },
          });
          setToken(data.token);
          resolve(data);
        } catch (e) {
          reject(e);
        }
      },
      fail: (err) => reject({ code: 'WX_LOGIN_FAILED', message: err.errMsg }),
    });
  });
}

async function ensureLogin(extra) {
  if (getToken()) return null;
  return login(extra);
}

module.exports = { login, ensureLogin };
