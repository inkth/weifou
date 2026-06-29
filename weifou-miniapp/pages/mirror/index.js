// 镜子分身：分身据画像，反过来对主人本人说「我眼中的你」。
// 建完分身的即时情感回报 + 驱动继续喂养的钩子。纯实时生成（GET /mirror），每次现拍。
const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');

Page({
  data: {
    loading: true,
    error: '',
    needProfile: false, // 还没建分身 → 引导创建
    realName: '我',
    tags: [],
    strength: '',
    blindspot: '',
    quip: '',
  },

  onLoad() { this.load(); },

  async load() {
    this.setData({ loading: true, error: '', needProfile: false });
    try {
      await ensureLogin();
      const res = await request({ url: '/mirror' });
      this.setData({
        realName: (res && res.realName) || '我',
        tags: (res && res.tags) || [],
        strength: (res && res.strength) || '',
        blindspot: (res && res.blindspot) || '',
        quip: (res && res.quip) || '',
        loading: false,
      });
    } catch (e) {
      if (e && e.code === 'NO_PROFILE') {
        this.setData({ loading: false, needProfile: true });
      } else {
        this.setData({ loading: false, error: (e && e.message) || '生成失败' });
      }
    }
  },

  reroll() { if (!this.data.loading) this.load(); },
  goCreate() { wx.navigateTo({ url: '/pages/onboarding/index' }); },

  onShareAppMessage() {
    return { title: '让 AI 分身说说，它眼中的我是什么样', path: '/pages/discover/index' };
  },
});
