const { fenToYuan } = require('../../utils/pay');
const { fmtDateTime } = require('../../utils/datetime');
const { ensureLogin } = require('../../utils/auth');
const { myQuestions } = require('../../utils/asyncq');

Page({
  data: { list: [], loading: true },

  async onShow() {
    try { await ensureLogin(); } catch (e) {}
    this.load();
  },

  async load() {
    try {
      const list = await myQuestions();
      this.setData({
        list: (list || []).map((q) => ({
          ...q,
          priceYuan: fenToYuan(q.price),
          timeText: fmtDateTime(q.createdAt),
          statusText: q.status === 'paid' ? '等待回答' : q.status === 'answered' ? '已回答' : '已退款',
        })),
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  goDetail(e) {
    const id = e.currentTarget.dataset.id;
    wx.navigateTo({ url: `/pages/question-detail/index?id=${id}` });
  },
});
