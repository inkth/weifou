const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');

// 首页 = 我的 AI 名片，由 /home/agents 的 primary 卡驱动；清新浅色 + 薄荷青（大卡 + 真实数据）。
// 学习工具已独立到「技能」Tab，首页只承载名片本体（接待入口在「消息」Tab）。
const FALLBACK = {
  chief: { name: '我的 AI 分身', initial: '分', online: true, hasProfile: false, profileId: '', line: '替你接待每个来访的人，有结果就喊你', stats: null },
};

function greet() {
  const h = new Date().getHours();
  if (h < 6) return '夜深了';
  if (h < 11) return '上午好';
  if (h < 14) return '中午好';
  if (h < 18) return '下午好';
  return '晚上好';
}

Page({
  data: { statusBarH: 20, greeting: '你好', chief: FALLBACK.chief, loading: true, loaded: false, errored: false },

  onLoad() {
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this.setData({ statusBarH: info.statusBarHeight || 20 });
    } catch (e) { /* 兜底默认 20 */ }
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 0 });
    }
    this.setData({ greeting: greet() });
    this.load();
  },

  async load() {
    this.setData({ loading: true, errored: false });
    try { await ensureLogin(); } catch (e) { /* 未登录也照常铺卡 */ }

    // 区分「真网络失败」与「空/新用户」：失败 → 顶部可重试条（保留现有内容，首屏兜底铺 demo）；
    // 空结果 → 新用户 demo 卡（原逻辑）。两者过去都静默铺 FALLBACK，主人分不清是挂了还是本该空。
    let cards = null;
    let failed = false;
    try { cards = await request({ url: '/home/agents' }); }
    catch (e) { failed = true; }

    if (failed) {
      const patch = { loading: false, loaded: true, errored: true };
      if (!this.data.loaded) { patch.chief = FALLBACK.chief; }
      this.setData(patch);
      return;
    }

    if (!cards || !cards.length) {
      this.setData({ chief: FALLBACK.chief, loading: false, loaded: true });
      return;
    }

    const primary = cards.find((c) => c.primary) || cards[0];
    const chief = {
      name: primary.name,
      initial: primary.initial || '名',
      line: primary.line,
      hasProfile: !!primary.ready,
      online: !!primary.ready,
      profileId: primary.profileId || '',
      stats: null,
    };
    // 已建名片：取真实数据填进大卡（成熟产品的 dashboard 感）。
    if (chief.hasProfile) {
      const s = await request({ url: '/visit/stats/mine' }).catch(() => null);
      if (s) {
        chief.stats = [
          { n: s.pv || 0, label: '浏览' },
          { n: s.uv || 0, label: '访客' },
          { n: s.askCount || 0, label: '问答' },
        ];
      }
    }

    this.setData({ chief, loading: false, loaded: true });
  },

  retry() { this.load(); },

  enterChief() {
    if (this.data.chief.hasProfile) {
      wx.navigateTo({ url: `/pages/chat/index?profileId=${this.data.chief.profileId}` });
    } else {
      wx.navigateTo({ url: '/pages/onboarding/index' });
    }
  },

  onShareAppMessage() {
    return { title: '来微否，养一个替你把事办成的 AI 分身', path: '/pages/discover/index' };
  },
});
