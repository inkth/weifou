const { ensureLogin } = require('../../utils/auth');
const { listAgents } = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');
const { loadEntries, agentVisible } = require('../../utils/entries');

// 成交克制：非会员卡片不挂任何会员/额度角标（第二幕起触到会员关时再弹窗提示）；
// 仅给会员保留「会员畅用」正向确认。
function decorate(a, isMember) {
  return {
    ...a,
    status: isMember ? '会员畅用' : '',
    statusKind: isMember ? 'member' : '',
  };
}

Page({
  data: { loading: true, blocked: false, isMember: false, agents: [] },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      await loadEntries(); // 预热入口可见性，供会员页 canPay 判定
      const [list, ms] = await Promise.all([
        listAgents(),
        membershipStatus().catch(() => ({ isMember: false })),
      ]);
      const isMember = !!ms.isMember;
      this.setData({
        agents: (list || []).map((a) => decorate(a, isMember)),
        isMember,
        loading: false,
        blocked: false,
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  open(e) {
    const { id, name } = e.currentTarget.dataset;
    const a = (this.data.agents || []).find((x) => x.id === id);
    // 透传 icon/accent：对话页无网时也能立刻画出对的头像+配色骨架
    wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}&accent=${encodeURIComponent((a && a.accent) || '')}&icon=${encodeURIComponent((a && a.icon) || '')}` });
  },

  goMembership() {
    wx.navigateTo({ url: '/pages/membership/index' });
  },
});
