// 技能 tab = Agent 学习市场：六门核心课 + 道德经「知常」，先免费体验、会员畅用。
// 首页已收敛为纯名片，不再承载「添加到首页」，故此处只做浏览 + 进入。
// 上架范围由服务端 /agents（enabled=true）决定，前端不再做名单过滤。
const { ensureLogin } = require('../../utils/auth');
const { listAgents, learnStreak } = require('../../utils/agent');
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
    // 透传 icon/accent：对话页无网时也能立刻画出对的头像+配色骨架
    wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}&accent=${encodeURIComponent((a && a.accent) || '')}&icon=${encodeURIComponent((a && a.icon) || '')}` });
  },

  goMembership() { wx.navigateTo({ url: '/pages/membership/index' }); },
});
