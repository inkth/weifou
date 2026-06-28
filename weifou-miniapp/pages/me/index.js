const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { loadEntries, entryVisible } = require('../../utils/entries');

// 我的 = 管自己：我的分身（资产 / 收益入口）+ 会员 + 付费往来 + 设置 + 合规位。全内向，天然非社交。
Page({
  data: {
    loading: true,
    profileId: null,
    realName: '',
    memberEntry: true,
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 2 });
    }
    this.load();
  },

  async load() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      await loadEntries();
      const me = await request({ url: '/user/me' }).catch(() => ({}));
      this.setData({
        profileId: me.profileId || null,
        realName: me.realName || '',
        memberEntry: entryVisible('membership', true),
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  goMyProfile() {
    if (!this.data.profileId) return;
    wx.navigateTo({ url: `/pages/profile/index?id=${this.data.profileId}&mine=1` });
  },
  // 创建与编辑统一走对话式 onboarding（已无表单页）
  goOnboarding() { wx.navigateTo({ url: '/pages/onboarding/index' }); },
  goInbox() { wx.navigateTo({ url: '/pages/inbox/index' }); },
  goSettings() { wx.navigateTo({ url: '/pages/settings/index' }); },
  goMembership() { wx.navigateTo({ url: '/pages/membership/index' }); },
  goQuestions() { wx.navigateTo({ url: '/pages/my-questions/index' }); },
  goDating() { wx.navigateTo({ url: '/pages/dating/index' }); },
});
