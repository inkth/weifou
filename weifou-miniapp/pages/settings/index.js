const { request, clearToken } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');

Page({
  data: {
    form: {
      contactPhone: '',
      contactVisible: false,
    },
    saving: false,
    regenerating: false,
  },

  async onLoad() {
    try {
      await ensureLogin();
      const mine = await request({ url: '/profile/mine' });
      if (mine) {
        this.setData({
          form: {
            contactPhone: mine.contactPhone || '',
            contactVisible: !!mine.contactVisible,
          },
        });
      }
    } catch (e) {}
  },

  onInput(e) {
    const key = e.currentTarget.dataset.key;
    this.setData({ [`form.${key}`]: e.detail.value });
  },

  onSwitch(e) {
    this.setData({ 'form.contactVisible': e.detail.value });
  },

  async saveContact() {
    if (this.data.saving) return;
    this.setData({ saving: true });
    try {
      // 微信号字段已下线（站内连接，不导流微信）：显式传空清掉历史存量
      await request({ url: '/profile/contact', method: 'PATCH', data: { ...this.data.form, contactWechat: '' } });
      wx.showToast({ title: '已保存', icon: 'success' });
    } catch (e) {
      wx.showToast({ title: e.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  goEdit() {
    wx.navigateTo({ url: '/pages/card-editor/index' });
  },

  async regenerate() {
    if (this.data.regenerating) return;
    this.setData({ regenerating: true });
    wx.showLoading({ title: 'AI 重新生成中…', mask: true });
    try {
      await request({ url: '/profile/regenerate', method: 'POST' });
      wx.hideLoading();
      wx.showToast({ title: '已更新', icon: 'success' });
      setTimeout(() => wx.navigateBack(), 600);
    } catch (e) {
      wx.hideLoading();
      wx.showToast({ title: e.message || '失败', icon: 'none' });
    } finally {
      this.setData({ regenerating: false });
    }
  },

  logout() {
    wx.showModal({
      title: '退出登录',
      content: '确定要退出当前账号吗？',
      success: (r) => {
        if (r.confirm) {
          clearToken();
          wx.reLaunch({ url: '/pages/discover/index' });
        }
      },
    });
  },
});
