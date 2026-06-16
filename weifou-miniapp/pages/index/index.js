const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');

Page({
  data: {
    loading: true,
    profileId: null,
  },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      const me = await request({ url: '/user/me' });
      this.setData({ profileId: me.profileId, loading: false });
    } catch (e) {
      this.setData({ loading: false });
    }
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
