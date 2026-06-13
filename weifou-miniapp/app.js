const { ensureLogin } = require('./utils/auth');

App({
  globalData: {
    userInfo: null,
  },

  async onLaunch() {
    // 启动即静默登录，方便后续接口直接调用
    try {
      await ensureLogin();
    } catch (e) {
      console.warn('[weifou] auto login failed', e);
    }
  },
});
