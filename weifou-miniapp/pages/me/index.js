const { request, clearToken } = require('../../utils/request');
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
  goMembership() { wx.navigateTo({ url: '/pages/membership/index' }); },
  goQuestions() { wx.navigateTo({ url: '/pages/my-questions/index' }); },
  // 记忆管理 = 分身记住的关于你的资料（KnowledgeItem），对外回答时会用上
  goMemory() { wx.navigateTo({ url: '/pages/memory/index' }); },

  logout() {
    wx.showModal({
      title: '退出登录',
      content: '确定要退出当前账号吗？',
      success: (r) => {
        if (r.confirm) {
          clearToken();
          wx.reLaunch({ url: '/pages/index/index' });
        }
      },
    });
  },

  // 预览：主人看看自己这张活名片被问的样子
  goQabox() {
    if (!this.data.profileId) return;
    wx.navigateTo({
      url: `/pages/qabox/index?profileId=${this.data.profileId}&realName=${encodeURIComponent(this.data.realName || '')}`,
    });
  },

  // 生成海报：可贴到朋友圈 / 线下的活名片物料（含二维码）
  goPoster() {
    if (!this.data.profileId) return;
    wx.navigateTo({ url: `/pages/poster/index?profileId=${this.data.profileId}` });
  },

  // 分享活名片：落到 chat（会说话的你）—— 别人点开能直接问你、和你聊，而不只是看一段简介。
  onShareAppMessage() {
    const id = this.data.profileId;
    const name = this.data.realName || '';
    return {
      title: name ? `这是 ${name} 的 AI 分身，有事直接问 TA 👋` : '这是我的 AI 分身，有事直接问 TA 👋',
      path: id ? `/pages/chat/index?profileId=${id}` : '/pages/index/index',
    };
  },
});
