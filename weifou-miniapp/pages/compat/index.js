// 契合度测试（访客侧）：别人来测「你和 TA 合不合」。
// intro → quiz（逐题作答，复用择偶测试交互）→ result（契合度 % + 趣味报告）。
// 纯娱乐定位：不撮合、不下发任何联系方式；结果只给访客看，主人侧只看到「N 人测过」。
const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');

// 小程序码 scene 仅传 id=xxx，需在此解析。
function parseScene(scene) {
  if (!scene) return '';
  const m = decodeURIComponent(scene).match(/(?:^|&)id=([^&]+)/);
  return m ? m[1] : '';
}

Page({
  data: {
    stage: 'intro', // intro | quiz | result
    loading: false,
    profileId: '',
    realName: 'TA',
    hasOwnClone: false,
    quizId: '',
    questions: [],
    index: 0,
    answers: {},
    cur: null,
    progressText: '',
    // 结果
    score: 0,
    headline: '',
    points: [],
    summary: '',
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
      hasOwnClone: !!(me && me.profileId),
    });
    wx.setNavigationBarTitle({ title: `测测你和 ${this.data.realName} 合不合` });
  },

  // ---------- intro ----------
  async onStart() {
    if (this.data.loading || !this.data.profileId) return;
    this.setData({ loading: true });
    try {
      const r = await request({ url: '/dating/compat/start', method: 'POST', data: { profileId: this.data.profileId } });
      const questions = r.questions || [];
      this.setData({
        stage: 'quiz', loading: false, quizId: r.quizId, questions,
        index: 0, answers: {}, cur: questions[0] || null,
        progressText: `1 / ${questions.length}`,
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: (e && e.message) || '出题失败，请重试', icon: 'none' });
    }
  },

  // ---------- quiz ----------
  onPick(e) {
    const key = e.currentTarget.dataset.key;
    const cur = this.data.cur;
    if (!cur) return;
    const answers = Object.assign({}, this.data.answers, { [cur.id]: key });
    this.setData({ answers });
    setTimeout(() => this.next(), 220);
  },

  next() {
    const { index, questions } = this.data;
    if (index + 1 < questions.length) {
      const ni = index + 1;
      this.setData({ index: ni, cur: questions[ni], progressText: `${ni + 1} / ${questions.length}` });
    } else {
      this.submit();
    }
  },

  onBack() {
    const { index, questions } = this.data;
    if (index === 0) return;
    const pi = index - 1;
    this.setData({ index: pi, cur: questions[pi], progressText: `${pi + 1} / ${questions.length}` });
  },

  async submit() {
    this.setData({ loading: true });
    const answers = Object.keys(this.data.answers).map((qid) => ({ questionId: qid, key: this.data.answers[qid] }));
    try {
      const r = await request({ url: '/dating/compat/submit', method: 'POST', data: { quizId: this.data.quizId, answers } });
      this.setData({
        stage: 'result', loading: false,
        score: r.score, headline: r.headline, points: r.points || [], summary: r.summary,
      });
    } catch (e) {
      this.setData({ loading: false, stage: 'quiz' });
      wx.showToast({ title: (e && e.message) || '分析失败，请重试', icon: 'none' });
    }
  },

  // ---------- result ----------
  onRetake() {
    this.setData({ stage: 'intro', score: 0, headline: '', points: [], summary: '' });
  },
  goCreate() { wx.navigateTo({ url: '/pages/onboarding/index' }); },
  goMine() { wx.switchTab({ url: '/pages/discover/index' }); },

  onShareAppMessage() {
    const id = this.data.profileId;
    return {
      title: `测测你和 ${this.data.realName} 的契合度 💘`,
      path: id ? `/pages/compat/index?profileId=${id}` : '/pages/index/index',
    };
  },
});
