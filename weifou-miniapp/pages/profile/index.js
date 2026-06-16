const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fenToYuan } = require('../../utils/pay');
const { fmtDateTime } = require('../../utils/datetime');
const { bookConsult, sendTip } = require('../../utils/consult');
const { track } = require('../../utils/track');
const { tierForPreset, getPreset } = require('../../utils/avatars');

Page({
  data: {
    profileId: '',
    profile: null,
    stats: null,
    isMine: false,
    loading: true,
    pricing: { enabled: false },
    // 沉浸式 hero 氛围（与对话舞台同一套温度档/光晕词汇）
    stageTier: 'warm',
    ambStyle: '',
    freshDelight: false, // 刚创建（fresh+本人）时 hero 头像庆祝一跳
    // 打赏
    tipVisible: false,
    tipPresets: [6, 18, 66, 88],
    tipAmount: 18,
    tipMessage: '',
    tipPaying: false,
    // 预约档期
    slotsVisible: false,
    slotList: [],
    selectedSlotId: '',
    booking: false,
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
        this.setData({ stats });
      } catch (e) {}
    } else {
      // 访客：拉取咨询定价
      try {
        const pricing = await request({ url: `/consult/pricing/${id}` });
        if (pricing.enabled) {
          pricing.price30Yuan = fenToYuan(pricing.price30);
          pricing.price60Yuan = fenToYuan(pricing.price60);
        }
        if (pricing.asyncEnabled) {
          pricing.asyncPriceYuan = fenToYuan(pricing.asyncPrice);
        }
        this.setData({ pricing });
      } catch (e) {}
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
      this.setData({ profile: data, loading: false });
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
    const c0 = (p.colors && p.colors[0]) || '#fb923c';
    const c1 = (p.colors && p.colors[1]) || c0;
    this.setData({ stageTier: tier.id, ambStyle: `--amb-a:${c0}; --amb-b:${c1};` });
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

  // 付费向本人提问（异步咨询）
  goAsk() {
    const p = this.data.profile;
    wx.navigateTo({
      url: `/pages/ask/index?profileId=${this.data.profileId}&realName=${encodeURIComponent(p.realName)}&source=profile`,
    });
  },

  goMyQuestions() {
    wx.navigateTo({ url: '/pages/my-questions/index' });
  },

  // 裂变：访客页脚 → 快速创建（带来源归因；已有主页的用户会自动进入完整编辑态）
  goCreateOwn() {
    track('own_hook_click', this.data.profileId, 'profile');
    wx.navigateTo({ url: `/pages/create/index?quick=1&ref=${this.data.profileId}` });
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

  // ---------- 付费咨询（选档预约） ----------
  async openSlots() {
    this.setData({ slotsVisible: true, selectedSlotId: '' });
    try {
      const slots = await request({ url: `/consult/slots/public/${this.data.profileId}` });
      slots.forEach((s) => {
        s.timeText = fmtDateTime(s.startAt);
        s.priceYuan =
          s.durationMin === 60 ? this.data.pricing.price60Yuan : this.data.pricing.price30Yuan;
      });
      this.setData({ slotList: slots });
    } catch (e) {
      this.setData({ slotList: [] });
    }
  },
  closeSlots() {
    this.setData({ slotsVisible: false });
  },
  selectSlot(e) {
    this.setData({ selectedSlotId: e.currentTarget.dataset.id });
  },
  async paySlot() {
    if (!this.data.selectedSlotId || this.data.booking) return;
    this.setData({ booking: true });
    try {
      await bookConsult(this.data.profileId, this.data.selectedSlotId, 'profile');
      this.setData({ slotsVisible: false });
      wx.showModal({
        title: '预约成功',
        content: '已为你预约，可在「我的通话」按时进入。',
        showCancel: false,
        success: () => wx.navigateTo({ url: '/pages/sessions/index' }),
      });
    } catch (e) {
      if (e.code !== 'PAY_CANCEL') {
        wx.showToast({ title: e.message || '预约失败', icon: 'none' });
      }
    } finally {
      this.setData({ booking: false });
    }
  },

  onShareAppMessage() {
    track('share_tap', this.data.profileId, 'profile');
    const p = this.data.profile;
    return {
      title: p?.persona?.oneLiner
        ? `和 ${p.realName} 的 AI 助理聊聊：${p.persona.oneLiner}`
        : `和 ${p?.realName || ''} 的 AI 助理聊聊`,
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
