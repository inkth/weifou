const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');

function inviteCode(options) {
  const raw = options.code || options.scene || '';
  try {
    const decoded = decodeURIComponent(raw).trim();
    const match = decoded.match(/^(?:ac|code)=([^&]+)$/i);
    return (match ? match[1] : decoded).toUpperCase().slice(0, 16);
  } catch (e) {
    return String(raw).toUpperCase().slice(0, 16);
  }
}

Page({
  data: {
    loading: true,
    code: '',
    state: 'binding',
    title: '正在确认邀请',
    desc: '请稍候，我们正在为你连接专属邀请关系。',
  },

  async onLoad(options) {
    const code = inviteCode(options || {});
    this.setData({ code });
    if (!/^[0-9]{4}$/.test(code)) {
      this.setState('invalid');
      return;
    }
    try {
      await ensureLogin();
      const result = await request({ url: '/agency/bind', method: 'POST', data: { agencyCode: code } });
      if (result && result.bound) this.setState('success');
      else if (result && result.reason === 'self') this.setState('self');
      else this.setState('returning');
    } catch (e) {
      this.setState('invalid');
    }
  },

  setState(state) {
    const copy = {
      success: { title: '邀请已确认', desc: '你已通过专属邀请加入微否，现在可以创建自己的 AI 分身并体验完整成长路径。' },
      returning: { title: '欢迎回到微否', desc: '你的账号已经有邀请归属，不会重复绑定。继续探索属于你的 AI 分身。' },
      self: { title: '这是你的专属邀请', desc: '代理商不能邀请自己，把这张邀请卡分享给朋友吧。' },
      invalid: { title: '邀请暂时无法确认', desc: '邀请码可能已失效，但你仍然可以正常进入微否体验产品。' },
    }[state];
    this.setData({ loading: false, state, ...copy });
  },

  goHome() {
    wx.reLaunch({ url: '/pages/discover/index' });
  },
});
