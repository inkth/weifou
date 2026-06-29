const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fenToYuan } = require('../../utils/pay');
const { sendTip } = require('../../utils/consult');
const { track } = require('../../utils/track');
const { tierForPreset, getPreset, DEFAULT_LIHE } = require('../../utils/avatars');
const { requestNewQuestionNotify, requestLeadNotify } = require('../../utils/subscribe');
const { buildTrustLine } = require('../../utils/trust');

Page({
  data: {
    profileId: '',
    profile: null,
    stats: null,
    isMine: false,
    loading: true,
    trustLine: '', // 信任徽章文案（社会证明，冷启动数字过小时为空）
    notifyAsked: false, // 首胜里点过"开启通知"后给即时反馈
    pricing: { enabled: false },
    // 沉浸式 hero 氛围（与对话舞台同一套温度档/光晕词汇）
    stageTier: 'warm',
    ambStyle: '',
    liheSrc: '', // 全屏立绘背景（与首页/对话统一，所有场景一致）
    freshDelight: false, // 刚创建（fresh+本人）时 hero 头像庆祝一跳
    // 打赏
    tipVisible: false,
    tipPresets: [6, 18, 66, 88],
    tipAmount: 18,
    tipMessage: '',
    tipPaying: false,
  },

  async onLoad(query) {
    // id 三来源：query.id（站内/旧分享）、query.scene（旧海报小程序码，URL-encoded 的 "id=xxx"）
    let id = query.id;
    if (!id && query.scene) {
      const m = decodeURIComponent(query.scene).match(/(?:^|&)id=([^&]+)/);
      if (m) id = m[1];
    }
    const isMine = query.mine === '1';
    const fromChat = query.from === 'chat';

    // chat-first 分流：无 from 标记的访客（含旧分享链接/旧海报码）直接送进对话；
    // 主人(mine=1)、对话内跳转(from=chat)、朋友圈(from=timeline，单页模式不能进 chat)走完整名片。
    // 该分支不 fetch、不报 visit——到访统计由 chat 页统一上报。
    if (id && !isMine && !query.from) {
      wx.redirectTo({ url: `/pages/chat/index?profileId=${id}` });
      return;
    }

    this.setData({ profileId: id, isMine, fresh: query.fresh === '1', freshDelight: query.fresh === '1' && isMine });

    try {
      await ensureLogin();
    } catch (e) {}

    await this.fetchProfile();

    // 写访客记录（自己访问也写，本人统计里用 distinct visitorOpenid 计 UV）
    // from=chat 来源不重复上报：同一次到访已在 chat 入口统计过
    if (id && !fromChat) {
      request({ url: `/visit/${id}`, method: 'POST' }).catch(() => {});
    }

    if (isMine) {
      try {
        const stats = await request({ url: '/visit/stats/mine' });
        // 收入「可见」：累计成交 + 本月（分账由微信直接打到主人零钱，无需提现）
        stats.incomeGrossYuan = fenToYuan(stats.incomeGross || 0);
        stats.incomeMonthYuan = fenToYuan(stats.incomeMonth || 0);
        this.setData({ stats });
      } catch (e) {}
    } else {
      // 访客：提问对所有分身免费开放（AI 即时答 + 本人可异步补一句）
      this.setData({ pricing: { asyncEnabled: true } });
    }
  },

  onShow() {
    // 从设置/重新生成回来时刷新
    if (this.data.profileId && this.data.profile) {
      this.fetchProfile();
    }
  },

  async fetchProfile() {
    try {
      const data = await request({ url: `/profile/${this.data.profileId}` });
      this.setData({ profile: data, loading: false, trustLine: buildTrustLine(data.trust, 'consulted') });
      this._applyStageTheme();
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  // hero 氛围：由 avatarStyle 推导温度档 + 注入头像同色系光晕（与对话舞台同一套词汇）
  _applyStageTheme() {
    const id = (this.data.profile && this.data.profile.avatarStyle) || '';
    const seed = this.data.profileId;
    const tier = tierForPreset(id, seed);
    const p = getPreset(id, seed);
    const c0 = (p.colors && p.colors[0]) || '#18b690';
    const c1 = (p.colors && p.colors[1]) || c0;
    // 所有场景统一全屏立绘：有专属 image 形象用专属，否则回退默认立绘
    const liheSrc = (p.type === 'image' && p.images && p.images.idle) ? p.images.idle : DEFAULT_LIHE;
    this.setData({ stageTier: tier.id, ambStyle: `--amb-a:${c0}; --amb-b:${c1};`, liheSrc });
  },

  goChat() {
    const p = this.data.profile;
    const mine = this.data.isMine ? '&mine=1' : '';
    wx.navigateTo({
      url: `/pages/chat/index?profileId=${this.data.profileId}&realName=${encodeURIComponent(p.realName)}&avatarStyle=${p.avatarStyle || ''}${mine}`,
    });
  },

  goSettings() {
    wx.navigateTo({ url: '/pages/settings/index' });
  },

  // 首胜第三步：把订阅授权前置到创建时，保证第一个访客动作（提问/留言）就能推达主人。
  // 需先在公众平台配好模板 ID；未配置时静默降级（不报错），仍给"已开启"的正反馈。
  async enableNotify() {
    try { await requestNewQuestionNotify(); } catch (e) {}
    try { await requestLeadNotify(); } catch (e) {}
    this.setData({ notifyAsked: true });
    wx.showToast({ title: '已开启来访通知', icon: 'success' });
  },

  goMyQuestions() {
    wx.navigateTo({ url: '/pages/my-questions/index' });
  },

  // 裂变：访客页脚 → 对话式创建自己的助理（带来源归因；已有主页会自动进入编辑态）
  goCreateOwn() {
    track('own_hook_click', this.data.profileId, 'profile');
    wx.navigateTo({ url: `/pages/onboarding/index?ref=${this.data.profileId}` });
  },

  goPoster() {
    wx.navigateTo({ url: `/pages/poster/index?profileId=${this.data.profileId}` });
  },

  async onContact() {
    if (this.data.isMine) {
      wx.navigateTo({ url: '/pages/settings/index' });
      return;
    }
    try {
      const c = await request({ url: `/profile/${this.data.profileId}/contact` });
      const lines = [];
      if (c.wechat) lines.push(`微信：${c.wechat}`);
      if (c.phone) lines.push(`电话：${c.phone}`);
      const content = lines.join('\n') || '本人未填写联系方式';
      wx.showModal({
        title: '联系本人',
        content,
        confirmText: '复制',
        cancelText: '关闭',
        success: (r) => {
          if (r.confirm) {
            wx.setClipboardData({ data: c.wechat || c.phone || '' });
          }
        },
      });
    } catch (e) {
      wx.showToast({ title: e.message || '本人未公开联系方式', icon: 'none' });
    }
  },

  // ---------- 打赏 ----------
  openTip() {
    this.setData({ tipVisible: true });
  },
  closeTip() {
    this.setData({ tipVisible: false });
  },
  noop() {},
  selectTip(e) {
    this.setData({ tipAmount: e.currentTarget.dataset.amount });
  },
  onTipMsg(e) {
    this.setData({ tipMessage: e.detail.value });
  },
  async payTip() {
    if (this.data.tipPaying) return;
    this.setData({ tipPaying: true });
    try {
      await sendTip(
        this.data.profileId,
        Math.round(this.data.tipAmount * 100),
        this.data.tipMessage,
        'profile'
      );
      this.setData({ tipVisible: false, tipMessage: '' });
      wx.showToast({ title: '感谢你的支持', icon: 'success' });
    } catch (e) {
      if (e.code !== 'PAY_CANCEL') {
        wx.showToast({ title: e.message || '打赏失败', icon: 'none' });
      }
    } finally {
      this.setData({ tipPaying: false });
    }
  },


  onShareAppMessage() {
    track('share_tap', this.data.profileId, 'profile');
    const p = this.data.profile;
    return {
      title: p?.persona?.oneLiner
        ? `和 ${p.realName} 的 AI 分身聊聊：${p.persona.oneLiner}`
        : `和 ${p?.realName || ''} 的 AI 分身聊聊`,
      // chat-first：好友分享直接落对话
      path: `/pages/chat/index?profileId=${this.data.profileId}&realName=${encodeURIComponent(p?.realName || '')}&avatarStyle=${p?.avatarStyle || ''}`,
      imageUrl: p?.avatarUrl,
    };
  },

  // 朋友圈**有意**保留落 profile：单页模式无法 wx.login，chat 的提问需要 JWT 会卡在登录；
  // 静态名片可正常展示，访客点"问 TA 的 AI"进完整模式后再进对话。不要"顺手"改成 chat。
  onShareTimeline() {
    const p = this.data.profile;
    return {
      title: p?.persona?.oneLiner || `${p?.realName} 的 AI 主页`,
      query: `id=${this.data.profileId}&from=timeline`,
    };
  },
});
