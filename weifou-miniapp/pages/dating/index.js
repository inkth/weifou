// 找对象 · 择偶测试页：intro（开场）→ quiz（逐题作答）→ result（画像 + 匹配度）。
// 匹配对象是平台预设原型（非真人），自测性质，避开婚恋交友红线。
const { startQuiz, submitQuiz, latestResult } = require('../../utils/dating');
const { request } = require('../../utils/request');

Page({
  data: {
    stage: 'intro',      // intro | quiz | result
    loading: false,      // 出题 / 提交中的全屏态
    myProfileId: '',     // 本人分身（有则可分享「对外契合测试」给别人测）
    myName: '',
    quizId: '',
    questions: [],
    index: 0,            // 当前题序
    answers: {},         // { questionId: key }
    cur: null,           // 当前题
    progressText: '',    // "3 / 8"
    profile: '',
    matches: [],
    headline: null,      // 头条：最契合原型 + 3 维拆解 {archetype,total,dimensions,summary}
  },

  onLoad() {
    // 有历史结果则可直接回看（不强制重测）。
    latestResult()
      .then((r) => {
        if (r && r.profile) {
          this.setData({ hasHistory: true });
        }
      })
      .catch(() => {});
    // 取本人分身：有则展示「让别人测和你合不合」分享入口。
    request({ url: '/user/me' })
      .then((me) => {
        if (me && me.profileId) this.setData({ myProfileId: me.profileId, myName: me.realName || '' });
      })
      .catch(() => {});
  },

  // 分享对外契合测试：别人点开会测「和你」的契合度（落 compat 页，带本人 profileId）。
  onShareAppMessage() {
    const id = this.data.myProfileId;
    return {
      title: id ? `测测你和${this.data.myName || '我'}的契合度 💘` : '择偶画像测试',
      path: id ? `/pages/compat/index?profileId=${id}` : '/pages/dating/index',
    };
  },

  // ---------- intro ----------
  async onStart() {
    if (this.data.loading) return;
    this.setData({ loading: true });
    try {
      const r = await startQuiz();
      const questions = r.questions || [];
      this.setData({
        stage: 'quiz',
        loading: false,
        quizId: r.quizId,
        questions,
        index: 0,
        answers: {},
        cur: questions[0] || null,
        progressText: `1 / ${questions.length}`,
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: (e && e.message) || '出题失败，请重试', icon: 'none' });
    }
  },

  async onViewHistory() {
    this.setData({ loading: true });
    try {
      const r = await latestResult();
      if (r && r.profile) {
        this.renderResult(r.profile, r.headline, r.matches || []);
      } else {
        wx.showToast({ title: '还没有测试记录', icon: 'none' });
      }
    } catch (e) {
      wx.showToast({ title: '加载失败', icon: 'none' });
    }
    this.setData({ loading: false });
  },

  // ---------- quiz ----------
  onPick(e) {
    const key = e.currentTarget.dataset.key;
    const cur = this.data.cur;
    if (!cur) return;
    const answers = Object.assign({}, this.data.answers, { [cur.id]: key });
    this.setData({ answers });
    // 选完短暂停顿后自动进入下一题（手感顺滑）。
    setTimeout(() => this.next(), 220);
  },

  next() {
    const { index, questions } = this.data;
    if (index + 1 < questions.length) {
      const ni = index + 1;
      this.setData({
        index: ni,
        cur: questions[ni],
        progressText: `${ni + 1} / ${questions.length}`,
      });
    } else {
      this.submit();
    }
  },

  onBack() {
    const { index, questions } = this.data;
    if (index === 0) return;
    const pi = index - 1;
    this.setData({
      index: pi,
      cur: questions[pi],
      progressText: `${pi + 1} / ${questions.length}`,
    });
  },

  async submit() {
    this.setData({ loading: true });
    const answers = Object.keys(this.data.answers).map((qid) => ({
      questionId: qid,
      key: this.data.answers[qid],
    }));
    try {
      const r = await submitQuiz(this.data.quizId, answers);
      this.renderResult(r.profile, r.headline, r.matches || []);
    } catch (e) {
      this.setData({ loading: false, stage: 'quiz' });
      wx.showToast({ title: (e && e.message) || '分析失败，请重试', icon: 'none' });
    }
  },

  // ---------- result ----------
  renderResult(profile, headline, matches) {
    // 兜底：老数据无 headline 时用 matches 第一名拼一个最小头条。
    const head = (headline && headline.archetype)
      ? headline
      : (matches[0]
        ? { archetype: matches[0].archetype, total: matches[0].score, dimensions: [], summary: matches[0].reason || '' }
        : null);
    this.setData({
      stage: 'result',
      loading: false,
      profile,
      matches,
      headline: head,
    });
  },

  onRetake() {
    this.setData({ stage: 'intro', profile: '', matches: [], headline: null });
  },

  goClone() {
    // 画像已喂回分身，引导去看自己的分身。
    wx.switchTab({ url: '/pages/me/index' });
  },
});
