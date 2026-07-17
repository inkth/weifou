const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');

function formatTime(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getMonth() + 1}月${date.getDate()}日 ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

Page({
  data: {
    loading: true,
    refreshing: false,
    isAgency: false,
    agencyCode: '',
    totalInvited: 0,
    newUserCount: 0,
    paidCount: 0,
    todayCount: 0,
    monthCount: 0,
    conversionRate: 0,
    recent: [],
    qrcode: '',
    qrLoading: false,
  },

  onLoad() {
    wx.showShareMenu({ menus: ['shareAppMessage'] });
    this.load(true);
  },

  async onPullDownRefresh() {
    await this.load(false);
    wx.stopPullDownRefresh();
  },

  async load(loadQrcode) {
    try {
      await ensureLogin();
      const dashboard = await request({ url: '/agency/dashboard' });
      if (!dashboard || !dashboard.isAgency) {
        wx.redirectTo({ url: '/pages/agency-register/index' });
        return;
      }
      const total = Number(dashboard.totalInvited || 0);
      const paid = Number(dashboard.paidCount || 0);
      this.setData({
        loading: false,
        isAgency: true,
        agencyCode: dashboard.agencyCode || '',
        totalInvited: total,
        newUserCount: Number(dashboard.newUserCount || 0),
        paidCount: paid,
        todayCount: Number(dashboard.todayCount || 0),
        monthCount: Number(dashboard.monthCount || 0),
        conversionRate: total > 0 ? Math.round((paid / total) * 100) : 0,
        recent: (dashboard.recent || []).map((item) => ({
          ...item,
          displayInitial: String(item.displayName || '用户').slice(-2),
          invitedText: formatTime(item.invitedAt),
        })),
      });
      if (loadQrcode && !this.data.qrcode) {
        let cachedQrcode = '';
        try { cachedQrcode = wx.getStorageSync(`weifou_agency_qr_${dashboard.agencyCode}`) || ''; } catch (e) {}
        if (cachedQrcode) this.setData({ qrcode: cachedQrcode });
        else this.loadQrcode();
      }
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '数据加载失败', icon: 'none' });
    }
  },

  async loadQrcode() {
    if (this.data.qrLoading) return;
    this.setData({ qrLoading: true });
    try {
      const result = await request({ url: '/agency/qrcode' });
      const qrcode = (result && result.wxacodeBase64) || '';
      this.setData({ qrcode });
      if (qrcode) {
        try { wx.setStorageSync(`weifou_agency_qr_${this.data.agencyCode}`, qrcode); } catch (e) {}
      }
    } catch (e) {
      // 小程序码依赖微信接口；失败不影响邀请码和原生分享。
    } finally {
      this.setData({ qrLoading: false });
    }
  },

  copyCode() {
    if (!this.data.agencyCode) return;
    wx.setClipboardData({ data: this.data.agencyCode });
  },

  saveQrcode() {
    const uri = this.data.qrcode || '';
    const comma = uri.indexOf(',');
    if (comma < 0) {
      wx.showToast({ title: '小程序码尚未生成', icon: 'none' });
      return;
    }
    const filePath = `${wx.env.USER_DATA_PATH}/weifou-agency-${this.data.agencyCode}.png`;
    wx.getFileSystemManager().writeFile({
      filePath,
      data: uri.slice(comma + 1),
      encoding: 'base64',
      success: () => wx.saveImageToPhotosAlbum({
        filePath,
        success: () => wx.showToast({ title: '已保存到相册', icon: 'success' }),
        fail: () => wx.showToast({ title: '保存失败，请检查相册权限', icon: 'none' }),
      }),
      fail: () => wx.showToast({ title: '小程序码保存失败', icon: 'none' }),
    });
  },

  onShareAppMessage() {
    return {
      title: '送你一张通往 AI 时代的成长通行证',
      path: `/pages/agency-invite/index?code=${this.data.agencyCode}`,
    };
  },

});
