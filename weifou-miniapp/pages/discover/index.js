const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');

// 分身（首页）= 我的 Agent 小队，由服务端 /home/agents 驱动（阵容/顺序/状态都在后端）。
// 主分身（primary）渲染为大卡，其余为专才卡；前端只渲染、按 type 路由。
// 兜底：接口失败时铺一份最简四卡，绝不空场、也不被网络错误带崩。
const FALLBACK = {
  chief: { name: '我的主分身', initial: '+', tier: 'warm', hasProfile: false, profileId: '', line: '先建一个，替你对外接待、有结果喊你。' },
  specialists: [
    { id: '', name: '学英语分身', initial: 'EN', tier: 'cool', line: '陪你开口练——纠音、对话、一段段升级。', kind: 'tool' },
    { id: '', name: '学商业分身', initial: '商', tier: 'lively', line: '生意卡哪了？我陪你拆，给能落地的下一步。', kind: 'tool' },
    { id: 'dating', name: '找对象分身', initial: '❤', tier: 'lively', line: '测测你和谁最配，顺手喂懂我的主分身。', kind: 'dating' },
  ],
};

Page({
  data: {
    statusBarH: 20,
    chief: FALLBACK.chief,
    specialists: [],
    loading: true,
  },

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
    this.load();
  },

  async load() {
    this.setData({ loading: true });
    try { await ensureLogin(); } catch (e) { /* 未登录也照常铺卡，点进去再引导 */ }

    const cards = await request({ url: '/home/agents' }).catch(() => null);
    if (!cards || !cards.length) {
      this.setData({ chief: FALLBACK.chief, specialists: FALLBACK.specialists, loading: false });
      return;
    }

    const primary = cards.find((c) => c.primary) || cards[0];
    const chief = {
      name: primary.name,
      initial: primary.initial || '+',
      tier: primary.tier || 'warm',
      line: primary.line,
      hasProfile: !!primary.ready,
      profileId: primary.profileId || '',
    };
    const specialists = cards
      .filter((c) => c !== primary)
      .map((c) => ({
        id: c.agentId || '',
        name: c.name,
        initial: c.initial,
        tier: c.tier,
        line: c.line,
        kind: c.type === 'dating' ? 'dating' : 'tool',
      }));

    this.setData({ chief, specialists, loading: false });
  },

  enterChief() {
    if (this.data.chief.hasProfile) {
      wx.navigateTo({ url: `/pages/chat/index?profileId=${this.data.chief.profileId}` });
    } else {
      wx.navigateTo({ url: '/pages/onboarding/index' });
    }
  },

  enterSpecialist(e) {
    const { id, name, kind } = e.currentTarget.dataset;
    if (kind === 'dating') {
      wx.navigateTo({ url: '/pages/dating/index' });
    } else if (kind === 'tool') {
      if (!id) { wx.showToast({ title: '正在上线，稍后再来', icon: 'none' }); return; }
      wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}` });
    }
  },

  addAgent() {
    wx.navigateTo({ url: '/pages/agents/index' });
  },

  onShareAppMessage() {
    return { title: '来微否，养一个替你把事办成的 AI 分身', path: '/pages/discover/index' };
  },
});
