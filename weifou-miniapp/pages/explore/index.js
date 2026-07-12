// 技能 tab =「人类基本功计划」：多条能力路径，第一幕免费、全课会员解锁后续。
// 首页已收敛为纯名片，不再承载「添加到首页」，故此处只做浏览 + 进入。
// 上架范围由服务端 /agents（enabled=true）决定，前端不再做名单过滤。
const { ensureLogin } = require('../../utils/auth');
const { listAgents, learnStreak } = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');

// 成交克制：非会员卡片不挂任何会员/额度角标（第二幕起触到会员关时再弹窗提示）；
// 仅给会员保留「会员畅用」正向确认。
function decorate(a, isMember) {
  return {
    ...a,
    status: isMember ? '会员畅用' : '',
    statusKind: isMember ? 'member' : '',
  };
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

  // 点卡片主体：一律进对话页。概念型课在对话页顶部铺「横版会走的路」（舞台=地图合一），
  // 角色停在当前关、进来即续/秒开；非概念型就是普通试用对话。
  open(e) {
    const { id, name } = e.currentTarget.dataset;
    const a = (this.data.agents || []).find((x) => x.id === id);
    // 透传 icon/accent：对话页无网时也能立刻画出对的头像+配色骨架
    // game=1：技能页进来的都是课，首帧即套游戏皮，免得加载窗口先闪一下普通聊天顶栏（含「解锁全课」）
    wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}&accent=${encodeURIComponent((a && a.accent) || '')}&icon=${encodeURIComponent((a && a.icon) || '')}&game=1` });
  },

  goMembership() { wx.navigateTo({ url: '/pages/membership/index' }); },
});
