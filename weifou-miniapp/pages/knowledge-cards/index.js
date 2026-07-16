const { ensureLogin } = require('../../utils/auth');
const { knowledgeCards } = require('../../utils/agent');

// 我的卡片：跨全部课程收集到的章末知识卡片（我的→我的卡片）。
Page({
  data: { loading: true, totalCards: 0, unlockedCount: 0, courses: [] },

  onShow() { this.load(); },

  async load() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      const r = await knowledgeCards();
      this.setData({
        loading: false,
        totalCards: r.totalCards || 0,
        unlockedCount: r.unlockedCount || 0,
        courses: r.courses || [],
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // 分享单张已解锁的卡片：从触发按钮的 dataset 取 slug 找到对应卡片拼文案（文字转发，不生成海报图）。
  onShareAppMessage(res) {
    const slug = res && res.target && res.target.dataset && res.target.dataset.slug;
    if (slug) {
      for (const course of this.data.courses) {
        const card = (course.cards || []).find((c) => c.slug === slug && c.unlocked);
        if (card) return { title: `${card.takeaway}（${card.source}）`, path: '/pages/knowledge-cards/index' };
      }
    }
    return { title: `我已经收集了 ${this.data.unlockedCount} 张知识卡片`, path: '/pages/knowledge-cards/index' };
  },
});
