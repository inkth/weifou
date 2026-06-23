const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fmtDateTime } = require('../../utils/datetime');
const { hostQuestions } = require('../../utils/asyncq');

// 消息 = 我的 AI 会话（我↔AI）+ 通知中心（主人收件）。
// 刻意不做真人↔真人实时 IM：主人对访客的承接走「收件箱 / 微信侧」，这里只聚合会话与通知。
Page({
  data: {
    loading: true,
    loadError: false,
    sessions: [],
    isHost: false,
    todo: { paid: 0, total: 0 },
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 1 });
    }
    this.load();
  },

  async load() {
    this.setData({ loading: true, loadError: false });
    try {
      await ensureLogin();
      const [me, list] = await Promise.all([
        request({ url: '/user/me' }).catch(() => ({})),
        request({ url: '/chat/sessions/mine' }).catch(() => []),
      ]);
      this.setData({
        sessions: (list || []).map((s) => ({
          ...s,
          timeText: fmtDateTime(s.updatedAt),
          lastText: s.lastMessage || '继续聊聊',
          initial: (s.realName || '·').slice(0, 1),
        })),
        isHost: !!me.profileId,
        loading: false,
      });
      if (me.profileId) this.loadTodo();
    } catch (e) {
      this.setData({ loading: false, loadError: true });
    }
  },

  // 主人收件计数：知识缺口 + 未跟进线索 + 付费提问（付费最该召回，单列出来）。
  async loadTodo() {
    try {
      const [gaps, leads, paid] = await Promise.all([
        request({ url: '/profile/gaps' }).catch(() => []),
        request({ url: '/profile/leads' }).catch(() => []),
        hostQuestions('paid').catch(() => []),
      ]);
      const g = (gaps || []).length;
      const l = (leads || []).filter((x) => x.status === 'new').length;
      const p = (paid || []).length;
      this.setData({ todo: { paid: p, total: g + l + p } });
    } catch (e) {}
  },

  retry() { this.load(); },

  enterChat(e) {
    const id = e.currentTarget.dataset.profile;
    wx.navigateTo({ url: `/pages/chat/index?profileId=${id}` });
  },

  goInbox() {
    wx.navigateTo({ url: '/pages/inbox/index' });
  },
});
