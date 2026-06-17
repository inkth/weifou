const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { track } = require('../../utils/track');

// 对外沟通风格（id 白名单与服务端 internal/persona/persona.go 的 StyleDescriptions 同步维护）
const STYLE_OPTIONS = [
  { id: 'steady', label: '沉稳专业', sub: '克制、有条理' },
  { id: 'warm', label: '亲和健谈', sub: '口语化、爱追问' },
  { id: 'sharp', label: '犀利直接', sub: '先给结论、不绕弯' },
  { id: 'humorous', label: '幽默轻松', sub: '会开玩笑、不油腻' },
];

Page({
  data: {
    styleOptions: STYLE_OPTIONS,
    quick: false, // 快速模式（裂变通道）：姓名+职业+一句话，30 秒上岗
    form: {
      realName: '',
      title: '',
      company: '',
      city: '',
      strengths: '',
      recentWork: '',
      howToKnow: '',
      quickIntro: '', // 快速模式的"一句话"，提交时映射到 strengths
      style: '', // 选填，空=AI 自行判断
    },
    submitting: false,
    canSubmit: false,
  },

  async onLoad(query) {
    // 裂变归因：从谁的 Agent 转化来的（chat 页钩子带入），随创建提交
    this._ref = (query && query.ref) || '';
    if (query && query.quick === '1') {
      this.setData({ quick: true });
      track('quick_create_enter', this._ref);
    }
    try {
      await ensureLogin();
      const mine = await request({ url: '/profile/mine' });
      if (mine) {
        const input = mine.personaInput || {};
        this.setData({
          // 已有主页 = 编辑场景，回到完整模式
          quick: false,
          form: {
            realName: mine.realName || '',
            title: mine.title || '',
            company: mine.company || '',
            city: mine.city || '',
            strengths: input.strengths || '',
            recentWork: input.recentWork || '',
            howToKnow: input.howToKnow || '',
            quickIntro: '',
            style: input.style || '',
          },
        });
        this.refreshCanSubmit();
      }
    } catch (e) {
      // ignore - 创建态
    }
  },

  // 快速模式 → 完整模式（单向；一句话带入 Q1 不丢内容）
  switchToFull() {
    this.setData({
      quick: false,
      'form.strengths': this.data.form.strengths || this.data.form.quickIntro,
    });
    this.refreshCanSubmit();
  },

  onInput(e) {
    const key = e.currentTarget.dataset.key;
    this.setData({ [`form.${key}`]: e.detail.value });
    this.refreshCanSubmit();
  },

  // 风格单选：再点一次取消（回到 AI 自行判断），不计入必填。
  selectStyle(e) {
    const id = e.currentTarget.dataset.id;
    const next = this.data.form.style === id ? '' : id;
    this.setData({ 'form.style': next });
  },

  refreshCanSubmit() {
    const f = this.data.form;
    const ok = this.data.quick
      ? !!(f.realName && f.title && f.quickIntro)
      : !!(f.realName && f.title && f.strengths && f.recentWork && f.howToKnow);
    this.setData({ canSubmit: ok });
  },

  async submit() {
    if (this.data.submitting || !this.data.canSubmit) return;
    this.setData({ submitting: true });
    wx.showLoading({ title: 'AI 生成中…', mask: true });
    try {
      await ensureLogin();
      const f = this.data.form;
      const body = this.data.quick
        ? {
            realName: f.realName,
            title: f.title,
            strengths: f.quickIntro, // 快速模式：一句话即 Q1，Q2/Q3 后续在完整编辑里补
            recentWork: '',
            howToKnow: '',
            style: f.style,
          }
        : { ...f };
      delete body.quickIntro;
      if (this._ref) body.ref = this._ref;
      const data = await request({
        url: '/profile',
        method: 'POST',
        data: body,
      });
      wx.hideLoading();
      wx.redirectTo({ url: `/pages/profile/index?id=${data.id}&mine=1&fresh=1` });
    } catch (e) {
      wx.hideLoading();
      wx.showModal({
        title: '生成失败',
        content: e.message || '请稍后再试',
        showCancel: false,
      });
    } finally {
      this.setData({ submitting: false });
    }
  },
});
