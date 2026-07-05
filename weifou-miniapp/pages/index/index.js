const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { loadEntries, agentVisible } = require('../../utils/entries');

Page({
  data: {
    loading: true,
    profileId: null,
    agentEntry: false, // AI 工具箱入口（iOS 隐藏，见 utils/entries）
  },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      // 拉「我的助理」+ 入口可见性（决定 AI 工具箱入口是否出现，iOS 隐藏）。
      const [me] = await Promise.all([
        request({ url: '/user/me' }),
        loadEntries(),
      ]);
      this.setData({
        profileId: me.profileId,
        agentEntry: true, // 工具箱免费体验对所有端开放（iOS 也能用，只是开通会员在 iOS 走留意向）
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // AI 工具箱（平台预设的付费工具 Agent，如学英语 / 面试）
  goAgents() {
    wx.navigateTo({ url: '/pages/agents/index' });
  },

  // 唯一途径：对话式（创建与重新编辑都走 onboarding，已无表单页）
  goOnboarding() {
    wx.navigateTo({ url: '/pages/onboarding/index' });
  },

  goMyProfile() {
    if (!this.data.profileId) return;
    wx.navigateTo({ url: `/pages/profile/index?id=${this.data.profileId}&mine=1` });
  },

  onShareAppMessage() {
    return {
      title: '别人加你微信前，先和你的 AI 聊聊 — 微否',
      path: '/pages/index/index',
    };
  },
});
