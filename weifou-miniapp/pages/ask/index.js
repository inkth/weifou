const { fenToYuan } = require('../../utils/pay');
const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const { askQuestion, fetchPricing } = require('../../utils/asyncq');
const { requestQuestionNotify } = require('../../utils/subscribe');
const { buildTrustLine } = require('../../utils/trust');

Page({
  data: {
    profileId: '',
    realName: 'TA',
    source: 'profile',
    asyncEnabled: false,
    priceYuan: '',
    trustLine: '', // 付费前的社会证明（数字过小时为空）
    question: '',
    loading: true,
    paying: false,
  },

  async onLoad(query) {
    const profileId = query.profileId || '';
    this.setData({
      profileId,
      realName: query.realName ? decodeURIComponent(query.realName) : 'TA',
      source: query.source === 'chat_card' ? 'chat_card' : 'profile',
    });
    try { await ensureLogin(); } catch (e) {}
    try {
      // 价格 + 信任并行：付费前最后一程展示社会证明，消解陌生人风险感
      const [p, prof] = await Promise.all([
        fetchPricing(profileId),
        request({ url: `/profile/${profileId}` }).catch(() => null),
      ]);
      this.setData({
        asyncEnabled: !!p.asyncEnabled,
        priceYuan: p.asyncEnabled ? fenToYuan(p.asyncPrice) : '',
        trustLine: buildTrustLine(prof && prof.trust, 'consulted'),
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  onInput(e) {
    this.setData({ question: e.detail.value });
  },

  async pay() {
    const q = (this.data.question || '').trim();
    if (q.length < 5) {
      wx.showToast({ title: '问题再具体一点（至少 5 字）', icon: 'none' });
      return;
    }
    if (this.data.paying) return;
    this.setData({ paying: true });
    try {
      const order = await askQuestion(this.data.profileId, q, this.data.source);
      // 付费成功后请求「已回答 / 已退款」订阅授权（未配置模板则静默跳过）
      try { await requestQuestionNotify(); } catch (e) {}
      wx.showToast({ title: '已提交，等待回答', icon: 'success' });
      const qid = order && order.asyncQuestionId;
      setTimeout(() => {
        wx.redirectTo({
          url: qid ? `/pages/question-detail/index?id=${qid}` : '/pages/my-questions/index',
        });
      }, 800);
    } catch (e) {
      if (e.code !== 'PAY_CANCEL') {
        wx.showToast({ title: e.message || '提问失败', icon: 'none' });
      }
    } finally {
      this.setData({ paying: false });
    }
  },
});
