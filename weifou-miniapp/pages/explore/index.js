// 发现 tab = Agent 学习市场「用 AI 学习一切」：全集工具 Agent，先免费体验、会员畅用、可添加到首页。
const { ensureLogin } = require('../../utils/auth');
const { listAgents, pinAgent, unpinAgent, learnStreak } = require('../../utils/agent');
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
      const [list, ms, st] = await Promise.all([
        listAgents().catch(() => []),
        membershipStatus().catch(() => ({ isMember: false })),
        learnStreak().catch(() => null),
      ]);
      const isMember = !!ms.isMember;
      this.setData({
        agents: (list || []).map((a) => decorate(a, isMember)),
        isMember,
        // 连续 ≥2 天才展示（第 1 天谈不上"连续"，安静）
        streak: st && st.days >= 2 ? st : null,
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: (e && e.message) || '加载失败', icon: 'none' });
    }
  },

  // 点卡片主体：概念型（学心理/英语陪练…）→ 闯关地图；其余 → 直进对话（试用）。
  open(e) {
    const { id, name } = e.currentTarget.dataset;
    const a = (this.data.agents || []).find((x) => x.id === id);
    if (a && a.concept) {
      wx.navigateTo({ url: `/pages/learn-map/index?id=${id}&name=${encodeURIComponent(name || '')}&accent=${encodeURIComponent(a.accent || '')}&icon=${encodeURIComponent(a.icon || '')}` });
      return;
    }
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
