const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { initial } = require('../../utils/avatars');

// 名片夹 = 我交换过名片的人。合规上只留「关系 + 可问对方 AI 分身」，不做真人私信。
Page({
  data: { loading: true, connections: [] },

  onShow() { this.load(); },

  async load() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      const r = await request({ url: '/connections' });
      const list = (r.connections || []).map((c) => ({ ...c, ini: initial(c.realName || '') }));
      this.setData({ connections: list, loading: false });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // 点开对方名片 → 直接和 TA 的 AI 分身对话
  goChat(e) {
    const { id, name } = e.currentTarget.dataset;
    if (!id) return;
    wx.navigateTo({ url: `/pages/chat/index?profileId=${id}&realName=${encodeURIComponent(name || '')}` });
  },

  goDiscover() { wx.switchTab({ url: '/pages/discover/index' }); },
});
