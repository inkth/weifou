const { ensureLogin } = require('../../utils/auth');
const { loadEntries, agentVisible } = require('../../utils/entries');
const { status, openMembership } = require('../../utils/membership');
const { fenToYuan } = require('../../utils/pay');

Page({
  data: {
    loading: true,
    canPay: true, // 安卓可在小程序内开通；iOS 不行（复用 agentVisible 闸门）
    isMember: false,
    expiresText: '',
    plans: [],
    paying: false,
  },

  async onShow() {
    this.setData({ loading: true });
    try {
      await ensureLogin();
      await loadEntries();
      const s = await status();
      const plans = (s.plans || []).map((p) => ({
        ...p,
        priceYuan: fenToYuan(p.price),
        perDay: p.days > 0 ? (p.price / 100 / p.days).toFixed(1) : '',
      }));
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

});
