const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');

Page({
  data: {
    loading: true,
    profileId: null,
    chatCount: 0,
  },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      // 并行取「我的助理」与「我聊过的对话数」——后者决定访客回流入口是否出现。
      const [me, chats] = await Promise.all([
        request({ url: '/user/me' }),
        request({ url: '/chat/sessions/mine' }).catch(() => []),
      ]);
      this.setData({
        profileId: me.profileId,
        chatCount: (chats || []).length,
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // 访客回流：我作为访客聊过的真人 AI 分身（文字对话 / 通话 / 付费提问）
  goMyChats() {
    wx.navigateTo({ url: '/pages/my-chats/index' });
  },

  // 首次创建走对话式 onboarding；重新编辑走表单。
  goOnboarding() {
    wx.navigateTo({ url: '/pages/onboarding/index' });
  },

  // 捏 Agent：点选式创建（Phase 1）
  goBuild() {
    wx.navigateTo({ url: '/pages/build/index' });
  },

  goCreate() {
    wx.navigateTo({ url: '/pages/create/index' });
  },

  goMyProfile() {
    if (!this.data.profileId) return;
    wx.navigateTo({ url: `/pages/profile/index?id=${this.data.profileId}&mine=1` });
  },

  goInbox() {
    wx.navigateTo({ url: '/pages/inbox/index' });
  },

  onShareAppMessage() {
    return {
      title: '别人加你微信前，先和你的 AI 聊聊 — 微否',
      path: '/pages/index/index',
    };
  },
});
