const { ensureLogin } = require('../../utils/auth');
const { listAgents } = require('../../utils/agent');
const { loadEntries, agentVisible } = require('../../utils/entries');
const { request } = require('../../utils/request');

// 分身（首页）= 我的私有 Agent 小队：总管（关系锚）+ 专才（平台工具 Agent / 种子兜底）。
// 「把事办成」的魂：每张卡用第一人称说要替你办的那件事，不做进度条仪表盘。
// 暗场沉浸（借猫箱/星野的壳），主角是「我养的那一个」，不是逛别人（逛别人在「发现」tab）。
const SEED_SPECIALISTS = [
  { id: 'seed-en', name: '英语教练', initial: 'EN', tier: 'cool', line: '就差开口一次，陪我念两句？' },
  { id: 'seed-date', name: '约会复盘', initial: '约', tier: 'lively', line: '昨晚那场，我帮你捋捋？' },
  { id: 'seed-life', name: '生活助理', initial: '活', tier: 'warm', line: '杂事交给我，省你半天。' },
];

Page({
  data: {
    statusBarH: 20,
    chief: { name: '小否 · 总管', initial: '否', tier: 'warm', line: '先告诉我你想办成的一件事，我替你张罗。', hasProfile: false },
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

      // 总管 = 我的分身（若已创建则它对外替我办事 + 回报结果），否则引导创建
      const chief = (me && me.profileId)
        ? {
            name: me.realName ? me.realName + ' 的分身' : '我的分身',
            initial: (me.realName || '否').slice(0, 1),
            tier: 'warm', hasProfile: true, profileId: me.profileId,
            line: '我替你把对外的事看着，有结果就喊你。',
          }
        : {
            name: '小否 · 总管', initial: '否', tier: 'warm', hasProfile: false,
            line: '先告诉我你想办成的一件事，我替你张罗。',
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
        specialists: real.length ? real : SEED_SPECIALISTS,
        agentEntry: show,
        loading: false,
      });
    } catch (e) {
      this.setData({ specialists: SEED_SPECIALISTS, loading: false });
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
    const { id, name, real } = e.currentTarget.dataset;
    if (real) {
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
