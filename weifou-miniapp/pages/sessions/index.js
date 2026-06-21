const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fmtDateTime } = require('../../utils/datetime');

function fmtSec(sec) {
  const m = String(Math.floor(sec / 60)).padStart(2, '0');
  const s = String(sec % 60).padStart(2, '0');
  return `${m}:${s}`;
}

Page({
  data: {
    list: [],
    loading: true,
    loadError: false,
    statusText: {
      pending: '待开始',
      ongoing: '进行中',
      ended: '已结束',
      canceled: '已取消',
    },
  },

  async onShow() {
    this.setData({ loading: true, loadError: false });
    try {
      await ensureLogin();
      const list = await request({ url: '/consult/sessions/mine' });
      list.forEach((s) => {
        if (s.durationSec) s.durationSecText = fmtSec(s.durationSec);
        if (s.scheduledAt) s.scheduledText = fmtDateTime(s.scheduledAt);
        // 未开始的通话可申请退款（双方均可）
        s.canRefund = s.status === 'pending';
      });
      this.setData({ list, loading: false });
    } catch (e) {
      // 标记错误态：避免把网络失败显示成"还没有通话记录"误导用户
      this.setData({ loading: false, loadError: true });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  retry() {
    this.onShow();
  },

  enter(e) {
    const id = e.currentTarget.dataset.id;
    wx.navigateTo({ url: `/pages/call/index?sessionId=${id}` });
  },

  refund(e) {
    const orderId = e.currentTarget.dataset.order;
    wx.showModal({
      title: '申请退款',
      content: '通话尚未开始可全额退款，确定申请吗？',
      success: async (r) => {
        if (!r.confirm) return;
        try {
          await request({ url: '/payment/refund', method: 'POST', data: { orderId } });
          wx.showToast({ title: '退款已发起', icon: 'success' });
          this.onShow();
        } catch (e) {
          wx.showToast({ title: e.message || '退款失败', icon: 'none' });
        }
      },
    });
  },
});
