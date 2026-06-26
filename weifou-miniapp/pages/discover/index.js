const { ensureLogin } = require('../../utils/auth');
const { listAgents } = require('../../utils/agent');
const { loadEntries, agentVisible } = require('../../utils/entries');
const { request } = require('../../utils/request');
const { getPreset, initial } = require('../../utils/avatars');

// 分身 = AI 助手目录（对外逛）：聚光分身 + 推荐分身 + 平台工具 Agent。
// 刻意定位为「AI 内容 / 工具」目录，不做真人社交信号（在线 / 距离 / 打招呼），规避社交类目。
const CATS = ['推荐', '情感陪伴', '职业咨询', '学习辅导', '生活答疑', '我收藏'];

// —— 种子分身（TEMP）——
// 后端「发现流」未就绪前的填充态，让旗舰页不空场；上线接 /discover 后整体替换为真数据。
// tier: warm|cool|lively（温度档，驱动立绘渐变+光晕，见 styles/stage.wxss .art-*）。
// lihe: 可选真立绘图（走 scripts/gen-avatars.mjs 产出后填入；为空则用渐变+首字兜底）。
const USE_SEED_DISCOVER = true; // 后端发现流就绪后置 false，spotlight/recos 改接 API
const SEED_SPOTLIGHT = { id: 'seed-xiaoman', name: '小满', initial: '小', role: '深夜树洞', tier: 'warm', tagline: '陪你聊到能睡着' };
const SEED_RECOS = [
  { id: 'seed-avery', name: 'Avery', initial: 'A', role: '职业咨询', tier: 'cool', tagline: '留学选校 · 文书把关', metric: '本周答了 128 问' },
  { id: 'seed-ale', name: '阿乐', initial: '阿', role: '生活答疑', tier: 'lively', tagline: '装修 · 数码 · 省钱', metric: '4.9 · 已服务 600+' },
  { id: 'seed-zhou', name: '周明', initial: '周', role: '夜诊医生', tier: 'warm', tagline: '用药 · 症状初判', metric: '已服务 2300+ 人' },
  { id: 'seed-xia', name: '林夏', initial: '林', role: '情感陪伴', tier: 'cool', tagline: '失眠 · 情绪 · 陪聊', metric: '4.9 · 夜间在线' },
];

Page({
  data: {
    statusBarH: 20,
    cats: CATS,
    activeCat: '推荐',
    spotlight: SEED_SPOTLIGHT,
    recos: [],
    recents: [], // 继续聊（真实最近会话）
    agentEntry: false, // 工具 Agent 入口（iOS 隐藏，见 utils/entries）
    agents: [],
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

      // 并行：工具 Agent（iOS 门控）+ 最近会话（继续聊真数据）
      const [agents, recents] = await Promise.all([
        show ? listAgents().catch(() => []) : Promise.resolve([]),
        request({ url: '/chat/sessions/mine' }).catch(() => []),
      ]);

      this.setData({
        agentEntry: show,
        agents: (agents || []).slice(0, 6),
        recents: (recents || []).slice(0, 8).map((s) => {
          const p = getPreset(null, s.profileId || s.realName);
          const c = (p && p.colors) || ['#18b690', '#0e9c7a'];
          return {
            profileId: s.profileId,
            name: s.realName || '·',
            initial: initial(s.realName),
            grad: `linear-gradient(140deg, ${c[0]}, ${c[1] || c[0]})`,
          };
        }),
        recos: USE_SEED_DISCOVER ? SEED_RECOS : [],
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // 分类目前为占位（视觉），未接后端筛选
  pickCat(e) {
    this.setData({ activeCat: e.currentTarget.dataset.cat });
  },

  // 继续聊：真实会话 → 回对话页续聊
  enterRecent(e) {
    const id = e.currentTarget.dataset.profile;
    if (!id) return;
    wx.navigateTo({ url: `/pages/chat/index?profileId=${id}` });
  },

  // 聚光 / 推荐分身：后端发现流就绪前为种子，优雅占位
  openReco() {
    wx.showToast({ title: '更多分身陆续上线', icon: 'none' });
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
