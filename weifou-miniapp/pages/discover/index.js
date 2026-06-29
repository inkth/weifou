const { ensureLogin } = require('../../utils/auth');
const { listAgents } = require('../../utils/agent');
const { loadEntries, agentVisible } = require('../../utils/entries');
const { request } = require('../../utils/request');

// 分身（首页）= 我的私有 Agent 小队：我的分身（主卡 / 关系锚）+ 专才（平台工具 Agent / 种子兜底）。
// 「把事办成」的魂：每张卡用第一人称说要替你办的那件事，不做进度条仪表盘。
// 暗场沉浸（借猫箱/星野的壳），主角是「我养的那一个」，不是逛别人（逛别人在「发现」tab）。
// 找对象：不依赖工具会员、两端都可用，固定作为首位专才（点击进择偶测试，结果回喂我的分身画像）。
const DATING_SPECIALIST = { id: 'dating', name: '找对象', initial: '❤', tier: 'lively', line: '测测你和谁最配，顺手喂懂我的分身。', kind: 'dating' };

Page({
  data: {
    statusBarH: 20,
    chief: { name: '我的分身', initial: '+', tier: 'warm', line: '先建一个，替你对外接待、有结果喊你。', hasProfile: false },
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
    try {
      await ensureLogin();
      await loadEntries();
      const show = agentVisible();

      const [me, agents] = await Promise.all([
        request({ url: '/user/me' }).catch(() => ({})),
        show ? listAgents().catch(() => []) : Promise.resolve([]),
      ]);

      // 主卡 = 我的分身（若已创建则它对外替我办事 + 回报结果），否则引导创建
      const chief = (me && me.profileId)
        ? {
            name: me.realName ? me.realName + ' 的分身' : '我的分身',
            initial: (me.realName || '否').slice(0, 1),
            tier: 'warm', hasProfile: true, profileId: me.profileId,
            line: '我替你把对外的事看着，有结果就喊你。',
          }
        : {
            name: '我的分身', initial: '+', tier: 'warm', hasProfile: false,
            line: '先建一个，替你对外接待、有结果喊你。',
          };

      // 专才 = 平台工具 Agent（真）；iOS 隐藏或为空时用种子兜底，保证不空场
      const real = (agents || []).slice(0, 6).map((a, i) => ({
        id: a.id,
        name: a.name,
        initial: (a.name || 'A').slice(0, 1),
        tier: ['cool', 'lively', 'warm'][i % 3],
        line: a.tagline || '点开，交给我一件事。',
        real: true,
      }));

      this.setData({
        chief,
        specialists: [DATING_SPECIALIST, ...real], // 找对象置首，其后接真实工具 Agent（iOS/空场则仅找对象，不再放点不动的假卡）
        agentEntry: show,
        loading: false,
      });
    } catch (e) {
      this.setData({ specialists: [DATING_SPECIALIST], loading: false });
    }
  },

  enterChief() {
    if (this.data.chief.hasProfile) {
      wx.navigateTo({ url: `/pages/chat/index?profileId=${this.data.chief.profileId}` });
    } else {
      wx.navigateTo({ url: '/pages/onboarding/index' });
    }
  },

  enterSpecialist(e) {
    const { id, name, real, kind } = e.currentTarget.dataset;
    if (kind === 'dating') {
      wx.navigateTo({ url: '/pages/dating/index' });
    } else if (real) {
      wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}` });
    } else {
      wx.showToast({ title: '更多专才陆续上线', icon: 'none' });
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
