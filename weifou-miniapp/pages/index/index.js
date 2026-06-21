const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { hostQuestions } = require('../../utils/asyncq');
const { loadEntries, agentVisible } = require('../../utils/entries');

Page({
  data: {
    loading: true,
    profileId: null,
    chatCount: 0,
    todoCount: 0,
    inboxLabel: '收件箱 · 待回答 / 访客线索',
    agentEntry: false, // AI 工具箱入口（iOS 隐藏，见 utils/entries）
  },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      // 并行取「我的助理」与「我聊过的对话数」——后者决定访客回流入口是否出现。
      // 顺带拉入口可见性（决定 AI 工具箱入口是否出现，iOS 隐藏）。
      const [me, chats] = await Promise.all([
        request({ url: '/user/me' }),
        request({ url: '/chat/sessions/mine' }).catch(() => []),
        loadEntries(),
      ]);
      this.setData({
        profileId: me.profileId,
        chatCount: (chats || []).length,
        agentEntry: true, // 工具箱免费体验对所有端开放（iOS 也能用，只是开通会员在 iOS 走留意向）
        loading: false,
      });
      // 主人态：拉收件箱待办计数，让主人一打开就被"有人在等你"勾住（付费提问最优先）。
      if (me.profileId) this.loadTodo();
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  // 收件箱待办计数：知识缺口(open) + 未跟进线索(new) + 付费提问(paid，有钱有时限，最该召回)。
  // 三个接口各自兜底，任一失败不影响其余计数；非主人态不调用。
  async loadTodo() {
    try {
      const [gaps, leads, paid] = await Promise.all([
        request({ url: '/profile/gaps' }).catch(() => []),
        request({ url: '/profile/leads' }).catch(() => []),
        hostQuestions('paid').catch(() => []),
      ]);
      const paidCount = (paid || []).length;
      const newLeads = (leads || []).filter((l) => l.status === 'new').length;
      const todoCount = (gaps || []).length + newLeads + paidCount;
      let inboxLabel = '收件箱 · 待回答 / 访客线索';
      if (paidCount > 0) inboxLabel = `收件箱 · ${paidCount} 条付费提问待答`;
      else if (todoCount > 0) inboxLabel = `收件箱 · ${todoCount} 条待处理`;
      this.setData({ todoCount, inboxLabel });
    } catch (e) {}
  },

  // 访客回流：我作为访客聊过的真人 AI 分身（文字对话 / 通话 / 付费提问）
  goMyChats() {
    wx.navigateTo({ url: '/pages/my-chats/index' });
  },

  // AI 工具箱（平台预设的付费工具 Agent，如学英语 / 面试）
  goAgents() {
    wx.navigateTo({ url: '/pages/agents/index' });
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
