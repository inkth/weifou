const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const { askQuestion } = require('../../utils/asyncq');
const { requestQuestionNotify } = require('../../utils/subscribe');
const { buildTrustLine } = require('../../utils/trust');

Page({
  data: {
    profileId: '',
    realName: 'TA',
    trustLine: '', // 社会证明（数字过小时为空）
    question: '',
    loading: true,
    submitting: false,
  },

  async onLoad(query) {
    const profileId = query.profileId || '';
    this.setData({
      profileId,
      realName: query.realName ? decodeURIComponent(query.realName) : 'TA',
    });
    try { await ensureLogin(); } catch (e) {}
    try {
      const prof = await request({ url: `/profile/${profileId}` }).catch(() => null);
      this.setData({
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

  async submit() {
    const q = (this.data.question || '').trim();
    if (q.length < 5) {
      wx.showToast({ title: '问题再具体一点（至少 5 字）', icon: 'none' });
      return;
    }
    if (this.data.submitting) return;
    this.setData({ submitting: true });
    try {
      const res = await askQuestion(this.data.profileId, q);
      // 提交后请求「已回答」订阅授权（未配置模板则静默跳过）
      try { await requestQuestionNotify(); } catch (e) {}
      wx.showToast({ title: '已提交，等待回答', icon: 'success' });
      const qid = res && res.id;
      setTimeout(() => {
        wx.redirectTo({
          url: qid ? `/pages/question-detail/index?id=${qid}` : '/pages/my-questions/index',
        });
      }, 800);
    } catch (e) {
      wx.showToast({ title: e.message || '提交失败', icon: 'none' });
    } finally {
      this.setData({ submitting: false });
    }
  },
});
