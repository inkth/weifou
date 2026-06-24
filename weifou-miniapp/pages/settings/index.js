const { request, clearToken } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fenToYuan } = require('../../utils/pay');

Page({
  data: {
    form: {
      contactWechat: '',
      contactPhone: '',
      contactVisible: false,
    },
    consult: {
      enabled: false,
      price30Yuan: '99',
      price60Yuan: '199',
      intro: '',
      asyncEnabled: false,
      asyncPriceYuan: '49',
    },
    saving: false,
    savingConsult: false,
    regenerating: false,
  },

  async onLoad() {
    try {
      await ensureLogin();
      const mine = await request({ url: '/profile/mine' });
      if (mine) {
        this.setData({
          form: {
            contactWechat: mine.contactWechat || '',
            contactPhone: mine.contactPhone || '',
            contactVisible: !!mine.contactVisible,
          },
        });
      }
      const c = await request({ url: '/consult/setting/mine' });
      this.setData({
        consult: {
          enabled: !!c.enabled,
          price30Yuan: fenToYuan(c.price30 || 9900),
          price60Yuan: fenToYuan(c.price60 || 19900),
          intro: c.intro || '',
          asyncEnabled: !!c.asyncEnabled,
          asyncPriceYuan: fenToYuan(c.asyncPrice || 4900),
        },
      });
    } catch (e) {}
  },

  onAsyncSwitch(e) {
    this.setData({ 'consult.asyncEnabled': e.detail.value });
  },

  onConsultInput(e) {
    const key = e.currentTarget.dataset.key;
    this.setData({ [`consult.${key}`]: e.detail.value });
  },

  async saveConsult() {
    if (this.data.savingConsult) return;
    this.setData({ savingConsult: true });
    try {
      const c = this.data.consult;
      await request({
        url: '/consult/setting',
        method: 'PATCH',
        data: {
          enabled: false, // 实时语音/视频咨询已下线，保存时一并关闭（payload 形状保持不变，price 字段沿用已加载值）
          price30: Math.round(parseFloat(c.price30Yuan || '0') * 100),
          price60: Math.round(parseFloat(c.price60Yuan || '0') * 100),
          intro: c.intro || undefined,
          asyncEnabled: c.asyncEnabled,
          asyncPrice: Math.round(parseFloat(c.asyncPriceYuan || '0') * 100),
        },
      });
      wx.showToast({ title: '已保存', icon: 'success' });
    } catch (e) {
      wx.showToast({ title: e.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ savingConsult: false });
    }
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
      await request({ url: '/profile/contact', method: 'PATCH', data: this.data.form });
      wx.showToast({ title: '已保存', icon: 'success' });
    } catch (e) {
      wx.showToast({ title: e.message || '保存失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  goEdit() {
    wx.navigateTo({ url: '/pages/create/index' });
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
          wx.reLaunch({ url: '/pages/index/index' });
        }
      },
    });
  },
});
