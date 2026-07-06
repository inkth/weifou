// 消息 = 接待收件箱：分身替你接待的访客提问都汇总在这里，你可以再补一句。
// 这是数字经纪人闭环的「收成」一环——分身干了活，主人回来看得到、跟得上。
// 单列聚焦访客提问（/async-question/host）；知识库在「记忆」页，此处不重复。
const { ensureLogin } = require('../../utils/auth');
const { fmtDateTime } = require('../../utils/datetime');
const { hostQuestions, answerQuestion } = require('../../utils/asyncq');
const { request } = require('../../utils/request');
const { requestNewQuestionNotify, NEW_QUESTION_TMPL_ID } = require('../../utils/subscribe');

// 访客点名要本人亲自答（escalatedAt）→ 比普通「分身已答」更优先、更醒目。
function statusText(q) {
  if (q.status === 'answered') return '已回答';
  if (q.escalatedAt) return '🙋 访客点名要你亲自答';
  if (q.status === 'pending') return '待回答';
  if (q.status === 'ai_answered') return '分身已答 · 可补一句';
  return '';
}

Page({
  data: {
    loading: true,
    errored: false,
    profileId: '', // 空态「分享你的问答箱」的分享落点
    questions: [],
    pendingCount: 0, // 待你跟进（未亲自答）条数
    canNotify: !!NEW_QUESTION_TMPL_ID, // 新提问订阅模板已配才显示入口（未配静默降级）
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 2 });
    }
    this.load();
  },

  async load() {
    this.setData({ loading: true, errored: false });
    try { await ensureLogin(); } catch (e) { /* 未登录也照常走，拿到空列表 */ }

    let list;
    try {
      const [qs, me] = await Promise.all([
        hostQuestions(),
        request({ url: '/user/me' }).catch(() => null),
      ]);
      list = qs || [];
      this.setData({ profileId: (me && me.profileId) || '' });
    } catch (e) {
      // 真失败：保留旧内容，顶部给可重试态，避免「挂了」被伪装成「没消息」。
      this.setData({ loading: false, errored: true });
      return;
    }

    const questions = list.map((q) => ({
      ...q,
      timeText: fmtDateTime(q.createdAt),
      statusText: statusText(q),
      // 主人还没亲自答（answered=已亲自答）→ 高亮成待跟进
      needsReply: q.status !== 'answered',
    }));
    this.setData({
      questions,
      pendingCount: questions.filter((q) => q.needsReply).length,
      loading: false,
    });
  },

  // 亲自补一句 / 作答。文字即可（answerQuestion 接受纯字符串）；语音在详情场景，此处从简。
  reply(e) {
    const { id, q, status } = e.currentTarget.dataset;
    if (status === 'answered') return; // 已亲自答，不重复弹窗
    wx.showModal({
      title: '亲自回一句',
      editable: true,
      placeholderText: `回答「${q}」，对方会收到你本人的答复`,
      confirmText: '发送',
      success: async (r) => {
        if (!r.confirm) return;
        const answer = (r.content || '').trim();
        if (!answer) {
          wx.showToast({ title: '请填写回答', icon: 'none' });
          return;
        }
        try {
          await answerQuestion(id, answer);
          wx.showToast({ title: '已回复', icon: 'success' });
          this.load();
        } catch (err) {
          wx.showToast({ title: err.message || '发送失败', icon: 'none' });
        }
      },
    });
  },

  // 主人召回（推）：点击授权「新提问」一次性订阅。微信要求必须由点击触发，故挂按钮而非 onShow。
  // 一次授权只换一条推送，所以入口常驻——主人每次来收件箱都能续上下一条。
  async enableNotify() {
    const res = await requestNewQuestionNotify();
    if (res.skipped) return; // 模板未配（此时入口本就不显示）
    if (res[NEW_QUESTION_TMPL_ID] === 'accept') {
      wx.showToast({ title: '已开启，新提问会微信通知你', icon: 'success' });
    } else {
      wx.showToast({ title: '未开启提醒', icon: 'none' });
    }
  },

  // 空收件箱的破局：分享自己的问答箱，让人匿名来问（"还没人来"变成可行动的下一步）
  onShareAppMessage() {
    const id = this.data.profileId;
    return {
      title: '匿名问问我的 AI 分身 👀',
      path: id ? `/pages/qabox/index?profileId=${id}` : '/pages/index/index',
    };
  },
});
