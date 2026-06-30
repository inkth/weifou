// 发现 tab = Agent 学习市场「用 AI 学习一切」：全集工具 Agent，先免费体验、会员畅用、可添加到首页。
const { ensureLogin } = require('../../utils/auth');
const { listAgents, pinAgent, unpinAgent } = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');

function decorate(a, isMember) {
  let status, statusKind;
  if (isMember) {
    status = '会员畅用';
    statusKind = 'member';
  } else if (a.freeTrialRemaining > 0) {
    status = `免费体验剩 ${a.freeTrialRemaining} 次`;
    statusKind = 'trial';
  } else {
    status = '免费体验已用完';
    statusKind = 'used';
  }
  return { ...a, status, statusKind };
}

Page({
  data: { statusBarH: 20, loading: true, isMember: false, agents: [] },

  onLoad() {
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this.setData({ statusBarH: info.statusBarHeight || 20 });
    } catch (e) { /* 兜底默认 20 */ }
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 1 });
    }
    this.load();
  },

  async load() {
    this.setData({ loading: true });
    try { await ensureLogin(); } catch (e) {}
    try {
      const [list, ms] = await Promise.all([
        listAgents().catch(() => []),
        membershipStatus().catch(() => ({ isMember: false })),
      ]);
      const isMember = !!ms.isMember;
      this.setData({ agents: (list || []).map((a) => decorate(a, isMember)), isMember, loading: false });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: (e && e.message) || '加载失败', icon: 'none' });
    }
  },

  // 点卡片主体 → 进 Agent 对话（试用）。
  open(e) {
    const { id, name } = e.currentTarget.dataset;
    wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}` });
  },

  // 添加 / 从首页移除（catchtap 阻止冒泡，不触发 open）。
  async togglePin(e) {
    const { id, pinned } = e.currentTarget.dataset;
    try {
      if (pinned) {
        await unpinAgent(id);
        wx.showToast({ title: '已从首页移除', icon: 'none' });
      } else {
        await pinAgent(id);
        wx.showToast({ title: '已添加到首页', icon: 'success' });
      }
      this.setData({ agents: this.data.agents.map((a) => (a.id === id ? { ...a, pinned: !pinned } : a)) });
    } catch (err) {
      wx.showToast({ title: (err && err.message) || '操作失败', icon: 'none' });
    }
  },

  goMembership() { wx.navigateTo({ url: '/pages/membership/index' }); },
});
