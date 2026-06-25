// 登录态下的小文件上传（multipart）。沿用 request.js 的响应解包：{ success, data }。
const { API_BASE } = require('./config');
const { getToken } = require('./request');
const { ensureLogin } = require('./auth');

async function uploadFile(filePath, { url = '/upload/voice', name = 'file' } = {}) {
  await ensureLogin();
  const token = getToken();
  return new Promise((resolve, reject) => {
    wx.uploadFile({
      url: API_BASE + url,
      filePath,
      name,
      header: {
        'X-Client-Type': 'miniapp',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      timeout: 60000,
      success(res) {
        let body = {};
        try { body = JSON.parse(res.data || '{}'); } catch (e) {}
        if (res.statusCode >= 200 && res.statusCode < 300 && body.success !== false) {
          return resolve(body.data || {});
        }
        reject({ code: body.code || `HTTP_${res.statusCode}`, message: body.message || '上传失败' });
      },
      fail(err) { reject({ code: 'NETWORK_ERROR', message: err.errMsg || '上传失败' }); },
    });
  });
}

// 上传语音临时文件，返回公开 URL 字符串。
async function uploadVoice(tempFilePath) {
  const data = await uploadFile(tempFilePath, { url: '/upload/voice' });
  return data.url;
}

module.exports = { uploadFile, uploadVoice };
