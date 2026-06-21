const { ensureLogin } = require('../../utils/auth');
const { loadEntries, agentVisible } = require('../../utils/entries');
const { status, buyMembership, leaveIntent } = require('../../utils/membership');
const { fenToYuan } = require('../../utils/pay');

Page({
  data: {
    loading: true,
    canPay: true, // 安卓可在小程序内开通；iOS 不行（复用 agentVisible 闸门）
    isMember: false,
    expiresText: '',
    plans: [],
    paying: false,
    intentDone: false,
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
      await buyMembership(planId);
      const s = await status();
      this.setData({
        isMember: !!s.isMember,
        expiresText: s.expiresAt ? String(s.expiresAt).slice(0, 10) : '',
      });
      wx.showToast({ title: '开通成功', icon: 'success' });
    } catch (e) {
      if (e.code !== 'PAY_CANCEL') wx.showToast({ title: e.message || '开通失败', icon: 'none' });
    } finally {
      this.setData({ paying: false });
    }
  },

  async onIntent() {
    try {
      await leaveIntent();
      this.setData({ intentDone: true });
      wx.showToast({ title: '已记录', icon: 'success' });
    } catch (e) {
      wx.showToast({ title: e.message || '提交失败', icon: 'none' });
    }
  },

});
