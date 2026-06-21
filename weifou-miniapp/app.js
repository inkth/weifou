const { ensureLogin } = require('./utils/auth');
const { loadEntries } = require('./utils/entries');

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
    // 异步拉取入口可见性（iOS 隐藏虚拟商品/工具 Agent），不阻塞启动。
    loadEntries();
  },
});
