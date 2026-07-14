const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { loadEntries, entryVisible } = require('../../utils/entries');
const { learningSummary } = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');

function expiryText(value) {
  if (!value) return '';
  const parts = String(value).slice(0, 10).split('-');
  if (parts.length !== 3) return '';
  return `${Number(parts[0])}年${Number(parts[1])}月${Number(parts[2])}日`;
}

// 我的 = 学习控制中心：继续学习 + 成长摘要 + 会员状态 + 账户服务。
Page({
  data: {
    loading: true,
    profileId: null,
    realName: '',
    nameInitial: '我',
    showMembership: true,
    isMember: false,
    expiresText: '',
    summary: { streak: { days: 0, best: 0, todayDone: false }, mastered: 0, learningCourses: 0 },
    currentCourse: null,
    statusBarH: 20, // 自定义导航：顶部留出状态栏高度，去掉原生白色标题栏
  },

  onLoad() {
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this.setData({ statusBarH: info.statusBarHeight || 20 });
    } catch (e) { /* 兜底默认 20 */ }
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 3 });
    }
    this.load();
  },

  async load() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      await loadEntries();
      const [me, summary, membership] = await Promise.all([
        request({ url: '/user/me' }).catch(() => ({})),
        learningSummary().catch(() => null),
        membershipStatus().catch(() => ({ isMember: false })),
      ]);
      const isMember = !!membership.isMember;
      const memberEntry = entryVisible('membership', true);
      this.setData({
        profileId: me.profileId || null,
        realName: me.realName || '',
        nameInitial: (me.realName || '我').trim().slice(0, 1) || '我',
        showMembership: memberEntry || isMember,
        isMember,
        expiresText: expiryText(membership.expiresAt),
        summary: summary || this.data.summary,
        currentCourse: (summary && summary.current) || null,
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
    }
  },

  goMyProfile() {
    if (!this.data.profileId) {
      this.goCardEditor();
      return;
    }
    wx.navigateTo({ url: `/pages/profile/index?id=${this.data.profileId}&mine=1` });
  },
  continueLearning() {
    const course = this.data.currentCourse;
    if (!course || !course.id) {
      this.goSkills();
      return;
    }
    wx.navigateTo({
      url: `/pages/agent-chat/index?id=${course.id}&name=${encodeURIComponent(course.name || '')}&accent=${encodeURIComponent(course.accent || '')}&icon=${encodeURIComponent(course.icon || '')}&game=1`,
    });
  },
  goSkills() { wx.switchTab({ url: '/pages/explore/index' }); },
  goCardEditor() { wx.navigateTo({ url: '/pages/card-editor/index' }); },
  goMembership() { wx.navigateTo({ url: '/pages/membership/index' }); },
  // 名片夹：我交换过名片的人（点开直接问对方分身）
  goConnections() { wx.navigateTo({ url: '/pages/connections/index' }); },
  goSettings() { wx.navigateTo({ url: '/pages/settings/index' }); },

  // 分享活名片：落到 chat（会说话的你）—— 别人点开能直接问你、和你聊，而不只是看一段简介。
  onShareAppMessage() {
    const id = this.data.profileId;
    const name = this.data.realName || '';
    return {
      title: name ? `这是 ${name} 的 AI 分身，有事直接问 TA 👋` : '这是我的 AI 分身，有事直接问 TA 👋',
      path: id ? `/pages/chat/index?profileId=${id}` : '/pages/discover/index',
    };
  },
});
