// 会话回放：主人查看 AI 助理替自己接待访客时说了什么。
// 这是主人信任助理、敢把链接发给重要的人的前提。
const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fmtDateTime } = require('../../utils/datetime');
const { DEFAULT_LIHE } = require('../../utils/avatars');

Page({
  data: {
    visitorName: '访客',
    messages: [],
    loading: true,
    loadError: false,
    sessionId: '',
    liheSrc: DEFAULT_LIHE, // 全屏立绘背景（与首页/对话统一）
  },

  async onLoad(query) {
    this.setData({
      visitorName: decodeURIComponent(query.name || '访客'),
      sessionId: query.sessionId || '',
    });
    try {
      await ensureLogin();
    } catch (e) {}
    this.load();
  },

  async load() {
    this.setData({ loading: true, loadError: false });
    try {
      const msgs = await request({ url: `/chat/sessions/${this.data.sessionId}/messages` });
      this.setData({
        messages: (msgs || []).map((m) => ({ ...m, timeText: fmtDateTime(m.createdAt) })),
        loading: false,
      });
    } catch (e) {
      // 标记错误态：避免把网络失败显示成"这个会话还没有消息"
      this.setData({ loading: false, loadError: true });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  retry() { this.load(); },
});
