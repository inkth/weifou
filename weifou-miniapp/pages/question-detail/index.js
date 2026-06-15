const { fenToYuan } = require('../../utils/pay');
const { fmtDateTime } = require('../../utils/datetime');
const { ensureLogin } = require('../../utils/auth');
const { questionDetail, answerQuestion } = require('../../utils/asyncq');

Page({
  data: {
    id: '',
    q: null,
    role: '',
    loading: true,
    answer: '',
    submitting: false,
  },

  async onLoad(query) {
    this.setData({ id: query.id || '' });
    try { await ensureLogin(); } catch (e) {}
    this.load();
  },

  async load() {
    try {
      const q = await questionDetail(this.data.id);
      this.setData({ q: this._decorate(q), role: q.role, loading: false });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  _decorate(q) {
    const statusText =
      q.status === 'paid' ? '待回答'
      : q.status === 'answered' ? '已回答'
      : q.status === 'refunded' ? '已退款'
      : q.status === 'pending_payment' ? '支付确认中' : '';
    return {
      ...q,
      priceYuan: fenToYuan(q.price),
      createdText: fmtDateTime(q.createdAt),
      deadlineText: q.answerDeadline ? fmtDateTime(q.answerDeadline) : '',
      answeredText: q.answeredAt ? fmtDateTime(q.answeredAt) : '',
      statusText,
    };
  },

  onAnswerInput(e) {
    this.setData({ answer: e.detail.value });
  },

  async submit() {
    const a = (this.data.answer || '').trim();
    if (!a) {
      wx.showToast({ title: '回答不能为空', icon: 'none' });
      return;
    }
    if (this.data.submitting) return;
    this.setData({ submitting: true });
    try {
      await answerQuestion(this.data.id, a);
      wx.showToast({ title: '已回答', icon: 'success' });
      this.setData({ answer: '' });
      this.load();
    } catch (e) {
      wx.showToast({ title: e.message || '提交失败', icon: 'none' });
    } finally {
      this.setData({ submitting: false });
    }
  },
});
