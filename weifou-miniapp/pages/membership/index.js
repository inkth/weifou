const { ensureLogin } = require('../../utils/auth');
const { loadEntries, agentVisible } = require('../../utils/entries');
const { status, openMembership, referralSummary, bindReferral } = require('../../utils/membership');
const { fenToYuan } = require('../../utils/pay');

Page({
  data: {
    loading: true,
    canPay: true, // 安卓可在小程序内开通；iOS 不行（复用 agentVisible 闸门）
    isMember: false,
    expiresText: '',
    plans: [],
    paying: false,
    // 邀请返奖：refCode=我的邀请参数；invitee=我是被邀人（首开加赠横幅）
    refCode: '',
    invitedCount: 0,
    pendingDays: 0,
    grantedDays: 0,
    isInvitee: false,
  },

  onLoad(options) {
    // 好友邀请链接带来的推荐人参数：登录后绑定（只试一次，服务端幂等）
    this._pendingRef = (options && options.ref) || '';
  },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      await loadEntries();
      await this.bindPendingRef();
      this.loadReferral();
      const s = await status();
      const plans = (s.plans || []).map((p) => {
        const savePct = p.origPrice > p.price ? Math.round((1 - p.price / p.origPrice) * 100) : 0;
        return {
          ...p,
          priceYuan: fenToYuan(p.price),
          perDay: p.days > 0 ? (p.price / 100 / p.days).toFixed(1) : '',
          origYuan: p.origPrice > p.price ? fenToYuan(p.origPrice) : '',
          saveText: savePct > 0 ? `限时立省 ${savePct}%` : '',
        };
      });
      this.setData({
        canPay: agentVisible(),
        isMember: !!s.isMember,
        expiresText: s.expiresAt ? String(s.expiresAt).slice(0, 10) : '',
        plans,
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  async openPlan(e) {
    if (!this.data.canPay || this.data.paying) return;
    const planId = e.currentTarget.dataset.id;
    this.setData({ paying: true });
    try {
      await openMembership(planId);
      // 虚拟支付异步发货：支付成功后会员到账靠服务端回调（秒级），稍候刷新状态。
      wx.showToast({ title: '支付成功，开通中', icon: 'success' });
      setTimeout(() => this.refreshStatus(2), 1500);
    } catch (e) {
      if (e.code !== 'PAY_CANCEL') wx.showToast({ title: e.message || '开通失败', icon: 'none' });
    } finally {
      this.setData({ paying: false });
    }
  },

  // 刷新会员状态；未到账则按递减次数重试（覆盖回调到账的短延迟）。
  async refreshStatus(retries) {
    try {
      const s = await status();
      this.setData({
        isMember: !!s.isMember,
        expiresText: s.expiresAt ? String(s.expiresAt).slice(0, 10) : '',
      });
      if (!s.isMember && retries > 0) {
        setTimeout(() => this.refreshStatus(retries - 1), 2500);
      }
    } catch (e) {}
  },

  // 绑定邀请链接携带的推荐人（失败静默，不打断购买流程）。
  async bindPendingRef() {
    const ref = this._pendingRef;
    if (!ref) return;
    this._pendingRef = '';
    try {
      await bindReferral(ref);
    } catch (e) {}
  },

  // 拉取我的邀请概况（分享参数 + 战报 + 被邀标记）。
  async loadReferral() {
    try {
      const r = await referralSummary();
      this.setData({
        refCode: r.refCode || '',
        invitedCount: r.invitedCount || 0,
        pendingDays: r.pendingDays || 0,
        grantedDays: r.grantedDays || 0,
        isInvitee: !!r.isInvitee,
      });
    } catch (e) {}
  },

  // 邀请好友：奖励只挂「好友完成开通」，分享动作本身无奖励（利诱分享红线）。
  onShareAppMessage() {
    const ref = this.data.refCode;
    return {
      title: '给你留了份会员加赠：首次开通「人类基本功计划」多得会员天数',
      path: ref ? `/pages/membership/index?ref=${ref}` : '/pages/membership/index',
    };
  },

});
