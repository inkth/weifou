const { ensureLogin } = require('../../utils/auth');
const { listAgents } = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');
const { loadEntries, agentVisible } = require('../../utils/entries');

function decorate(a, isMember) {
  let status, statusKind;
  if (isMember) {
    status = '会员畅用';
    statusKind = 'member';
  } else if (a.freeTrialRemaining > 0) {
    status = `免费体验剩 ${a.freeTrialRemaining} 次`;
    statusKind = 'trial';
  } else {
    status = '免费体验已用完';
    statusKind = 'used';
  }
  return { ...a, status, statusKind };
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
