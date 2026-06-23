const { ensureLogin } = require('../../utils/auth');
const { listAgents } = require('../../utils/agent');
const { loadEntries, agentVisible } = require('../../utils/entries');

// 分身 = AI 助手目录（对外逛）：平台工具 Agent + 真人 AI 分身。
// 刻意定位为「AI 内容 / 工具」目录，不做真人社交信号（在线 / 距离 / 打招呼），规避社交类目。
const CATS = ['推荐', '学英语', '情感倾诉', '行业咨询', '名人IP', '我收藏'];

Page({
  data: {
    cats: CATS,
    activeCat: '推荐',
    agentEntry: false, // 工具 Agent 入口（iOS 隐藏，见 utils/entries）
    agents: [],
    loading: true,
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
      let agents = [];
      if (show) {
        const list = await listAgents().catch(() => []);
        agents = (list || []).slice(0, 6);
      }
      this.setData({ agentEntry: show, agents, loading: false });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // 分类目前为占位（视觉），未接后端筛选。
  pickCat(e) {
    this.setData({ activeCat: e.currentTarget.dataset.cat });
  },

  openAgent(e) {
    const { id, name } = e.currentTarget.dataset;
    wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}` });
  },

  goAgents() {
    wx.navigateTo({ url: '/pages/agents/index' });
  },

  onSearchTap() {
    wx.showToast({ title: '搜索即将上线', icon: 'none' });
  },

  onShareAppMessage() {
    return { title: '来微否，和各种 AI 分身聊聊', path: '/pages/discover/index' };
  },
});
