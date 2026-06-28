// 问答箱（访客侧）：匿名向「TA 的 AI 分身」问一句，分身据画像即时作答。
// 这是拉新楔子的落点——答完即抛「也给自己建一个分身」的转化钩子。
const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const { qaboxAsk } = require('../../utils/asyncq');

// 小程序码 scene 仅传 id=xxx，需在此解析。
function parseScene(scene) {
  if (!scene) return '';
  const decoded = decodeURIComponent(scene);
  const m = decoded.match(/(?:^|&)id=([^&]+)/);
  return m ? m[1] : '';
}

Page({
  data: {
    profileId: '',
    realName: 'TA',
    oneLiner: '',
    question: '',
    submitting: false,
    asked: '', // 已提交的问题（结果屏回显）
    answer: '', // 分身的即时回答（提交后填充 → 进入结果屏）
    hasOwnClone: false, // 访客自己是否已有分身（决定转化 CTA 形态）
    loading: true,
  },

  async onLoad(query) {
    const profileId = query.profileId || query.id || parseScene(query.scene) || '';
    this.setData({
      profileId,
      realName: query.realName ? decodeURIComponent(query.realName) : 'TA',
    });
    try { await ensureLogin(); } catch (e) {}
    const [prof, me] = await Promise.all([
      request({ url: `/profile/${profileId}` }).catch(() => null),
      request({ url: '/user/me' }).catch(() => null),
    ]);
    this.setData({
      realName: (prof && prof.realName) || this.data.realName,
      oneLiner: (prof && prof.oneLiner) || '',
      hasOwnClone: !!(me && me.profileId),
      loading: false,
    });
    wx.setNavigationBarTitle({ title: `问问 ${this.data.realName} 的分身` });
  },

  onInput(e) { this.setData({ question: e.detail.value }); },

  async submit() {
    const q = (this.data.question || '').trim();
    if (q.length < 2) {
      wx.showToast({ title: '再多写一点', icon: 'none' });
      return;
    }
    if (this.data.submitting) return;
    this.setData({ submitting: true });
    try {
      const res = await qaboxAsk(this.data.profileId, q);
      this.setData({ asked: q, answer: (res && res.answer) || '', question: '' });
    } catch (e) {
      wx.showToast({ title: e.message || '提交失败', icon: 'none' });
    } finally {
      this.setData({ submitting: false });
    }
  },

  askAnother() { this.setData({ asked: '', answer: '' }); },

  // 病毒转化：访客也去建一个自己的分身（拉新闭环的关键一跳）。
  goCreate() { wx.navigateTo({ url: '/pages/onboarding/index' }); },
  goMyClone() { wx.switchTab({ url: '/pages/discover/index' }); },

  onShareAppMessage() {
    const id = this.data.profileId;
    return {
      title: `匿名问问 ${this.data.realName} 的 AI 分身 👀`,
      path: id ? `/pages/qabox/index?profileId=${id}` : '/pages/index/index',
    };
  },
  onShareTimeline() {
    const id = this.data.profileId;
    return {
      title: `匿名问问 ${this.data.realName} 的 AI 分身 👀`,
      query: id ? `profileId=${id}` : '',
    };
  },
});
