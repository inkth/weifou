const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fmtDateTime } = require('../../utils/datetime');

// 访客回流中心：我作为访客，和别人的真人 AI 分身聊过/约过/问过什么。
// 主体是文字对话（/chat/sessions/mine），并收拢已有的「语音通话」「付费提问」入口。
Page({
  data: { list: [], loading: true },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      const list = await request({ url: '/chat/sessions/mine' });
      this.setData({
        list: (list || []).map((s) => ({
          ...s,
          timeText: fmtDateTime(s.updatedAt),
          lastText: s.lastMessage || '继续聊聊',
          initial: (s.realName || '·').slice(0, 1),
        })),
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // 回到该 AI 分身的对话（chat 页按 profileId 续聊）
  enterChat(e) {
    const id = e.currentTarget.dataset.profile;
    wx.navigateTo({ url: `/pages/chat/index?profileId=${id}` });
  },

  goCalls() {
    wx.navigateTo({ url: '/pages/sessions/index' });
  },

  goQuestions() {
    wx.navigateTo({ url: '/pages/my-questions/index' });
  },
});
