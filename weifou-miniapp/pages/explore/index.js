const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const { getPreset, initial } = require('../../utils/avatars');

// 发现 = 逛「别人的 AI 分身」目录（聚光 + 最近看过 + 推荐墙）。
// 刻意定位为「AI 内容 / 工具」目录，不做真人社交信号（在线 / 距离 / 打招呼），规避社交类目。
// 对外默认 AI 出面、外界免费聊（不做付费解锁 AI 回答，见战略 iOS 红线）。
const CATS = ['推荐', '情感咨询', '职业咨询', '学习辅导', '生活答疑', '我收藏'];

// —— 种子分身（TEMP）—— 后端「发现流」就绪后置 false，spotlight/recos 改接 /discover。
const USE_SEED_DISCOVER = true;
const SEED_SPOTLIGHT = { id: 'seed-xiaoman', name: '小满', initial: '小', role: '情绪解压', tier: 'warm', tagline: '烦心事 · 帮你理清思路' };
const SEED_RECOS = [
  { id: 'seed-avery', name: 'Avery', initial: 'A', role: '职业咨询', tier: 'cool', tagline: '留学选校 · 文书把关', metric: '本周答了 128 问' },
  { id: 'seed-ale', name: '阿乐', initial: '阿', role: '生活答疑', tier: 'lively', tagline: '装修 · 数码 · 省钱', metric: '4.9 · 已服务 600+' },
  { id: 'seed-zhou', name: '周明', initial: '周', role: '健身计划', tier: 'warm', tagline: '增肌 · 减脂 · 训练', metric: '已服务 2300+ 人' },
  { id: 'seed-xia', name: '林夏', initial: '林', role: '情感咨询', tier: 'cool', tagline: '关系 · 沟通 · 情绪', metric: '4.9 · 已答 800 问' },
];

Page({
  data: {
    statusBarH: 20,
    cats: CATS,
    activeCat: '推荐',
    spotlight: SEED_SPOTLIGHT,
    recos: [],
    recents: [], // 最近看过（真实最近会话兜底）
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
      this.getTabBar().setData({ selected: 1 });
    }
    this.load();
  },

  async load() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      const recents = await request({ url: '/chat/sessions/mine' }).catch(() => []);
      this.setData({
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

  pickCat(e) {
    this.setData({ activeCat: e.currentTarget.dataset.cat });
  },

  enterRecent(e) {
    const id = e.currentTarget.dataset.profile;
    if (!id) return;
    wx.navigateTo({ url: `/pages/chat/index?profileId=${id}` });
  },

  // 聚光 / 推荐分身：后端发现流就绪前为种子，优雅占位
  openReco() {
    wx.showToast({ title: '更多分身陆续上线', icon: 'none' });
  },

  onSearchTap() {
    wx.showToast({ title: '搜索即将上线', icon: 'none' });
  },

  onShareAppMessage() {
    return { title: '来微否，逛逛各种 AI 分身', path: '/pages/explore/index' };
  },
});
