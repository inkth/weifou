const { ensureLogin } = require('../../utils/auth');
const { listAgents } = require('../../utils/agent');
const { loadEntries, agentVisible } = require('../../utils/entries');
const { request } = require('../../utils/request');

// 分身（首页）= 我的分身小队，固定四卡：我的主分身（代表你的对外分身）+ 学英语 + 学商业 + 找对象。
// 主分身是「你」、是核心；其余三个是能力分身（工具/玩法）。暗场沉浸，主角是「我养的这一队」。
// 找对象分身：不依赖工具会员、两端都可用（点击进择偶测试，结果回喂我的主分身画像）。
const DATING_SPECIALIST = { id: 'dating', name: '找对象分身', initial: '❤', tier: 'lively', line: '测测你和谁最配，顺手喂懂我的主分身。', kind: 'dating' };
// 学习型工具分身（会员畅用 · 非会员免费体验几次）；id 运行时按 slug 从 /agents 取真实 Agent。
const TOOL_CARDS = [
  { slug: 'spoken-english', name: '学英语分身', initial: 'EN', tier: 'cool', line: '陪你开口练——纠音、对话、一段段升级。' },
  { slug: 'business-coach', name: '学商业分身', initial: '商', tier: 'lively', line: '生意卡哪了？我陪你拆，给能落地的下一步。' },
];

Page({
  data: {
    statusBarH: 20,
    chief: { name: '我的主分身', initial: '+', tier: 'warm', line: '先建一个，替你对外接待、有结果喊你。', hasProfile: false },
    specialists: [],
    agentEntry: false, // 工具 Agent 入口（iOS 隐藏，见 utils/entries）
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

    // 登录 / entries 失败都不阻断四卡渲染（四卡是本地常量，不该被网络错误带崩）。
    try { await ensureLogin(); } catch (e) { /* 未登录也照常铺卡，点进去再引导 */ }
    try { await loadEntries(); } catch (e) { /* entries 取失败不影响首页四卡 */ }

    const [me, agents] = await Promise.all([
      request({ url: '/user/me' }).catch(() => ({})),
      listAgents().catch(() => []), // 工具分身两端固定展示（iOS 虚拟支付已开放，不再隐藏）
    ]);

    // 主分身 = 代表你的对外分身（已建则替你接待 + 回报结果），否则引导创建
    const chief = (me && me.profileId)
      ? {
          name: me.realName ? me.realName + ' 的主分身' : '我的主分身',
          initial: (me.realName || '否').slice(0, 1),
          tier: 'warm', hasProfile: true, profileId: me.profileId,
          line: '我替你把对外的事看着，有结果就喊你。',
        }
      : {
          name: '我的主分身', initial: '+', tier: 'warm', hasProfile: false,
          line: '先建一个，替你对外接待、有结果喊你。',
        };

    // 工具分身按 slug 取真实 Agent id（学英语 / 学商业）；取不到 id 不影响卡片展示
    const bySlug = {};
    (agents || []).forEach((a) => { if (a.slug) bySlug[a.slug] = a; });
    const toolCards = TOOL_CARDS.map((t) => {
      const a = bySlug[t.slug];
      return { id: a ? a.id : '', name: t.name, initial: t.initial, tier: t.tier, line: t.line, kind: 'tool' };
    });

    let entry = false;
    try { entry = agentVisible(); } catch (e) { /* 默认隐藏召新入口即可 */ }

    // 首页固定四卡：我的主分身（大卡）+ 学英语分身 + 学商业分身 + 找对象分身
    this.setData({
      chief,
      specialists: [...toolCards, DATING_SPECIALIST],
      agentEntry: entry,
      loading: false,
    });
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
    if (this.data.agentEntry) wx.navigateTo({ url: '/pages/agents/index' });
    else wx.navigateTo({ url: '/pages/onboarding/index' });
  },

  onShareAppMessage() {
    return { title: '来微否，养一个替你把事办成的 AI 分身', path: '/pages/discover/index' };
  },
});
