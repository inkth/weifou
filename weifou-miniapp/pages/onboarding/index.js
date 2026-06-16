const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { track } = require('../../utils/track');
const { tierForPreset, getPreset } = require('../../utils/avatars');

// 对话式创建（自由回答版）：用户一段话讲完，服务端 /profile/extract 抽字段 + 按语气定风格；
// 缺必填（realName/title/strengths）则按 followup 追问一句，齐了即可上岗。最终走现有 POST /profile，最小后端改动。
const OPENER = '嗨，我是来帮你建主页的 AI 助理～ 先用一两句话介绍下你自己就好：你是谁、平时做什么、最能帮别人解决什么问题？想到哪说到哪，也可以点麦克风说。';

// 风格 → 卡通形象（与 create 页 STYLE_AVATAR 一致）
const STYLE_AVATAR = {
  steady: 'toon-steady', warm: 'toon-warm', sharp: 'toon-sharp', humorous: 'toon-humorous',
};

// 必填缺失时的兜底追问（服务端没给 followup 时用）
function fallbackFollowup(form) {
  if (!form.realName) return '我该怎么称呼你呢？';
  if (!form.title) return '你平时主要是做什么的？';
  if (!form.strengths) return '你最能帮别人解决什么问题？';
  return '还想补充点什么吗？';
}

Page({
  data: {
    msgs: [],            // { id, role: 'ai' | 'me', text }
    input: '',
    thinking: false,     // 抽取请求在途
    submitting: false,
    recording: false,    // 语音录制中
    canFinish: false,    // realName+title+strengths 已齐
    confirmed: false,    // 已发过"了解你了"的确认语，避免重复
    avatarPreset: 'toon-warm',
    avatarState: 'speaking',
    stageTier: 'warm',   // hero 氛围档（随 avatarPreset 实时变档）
    ambStyle: '',        // 头像同色系光晕
    delight: false,      // "我大概了解你了"时庆祝一跳
    toLast: '',
    turns: 0,            // 用户发话轮数，用于超过若干轮仍未齐时提示手动填写
    form: { realName: '', title: '', strengths: '', recentWork: '', howToKnow: '', style: '' },
  },

  async onLoad(query) {
    this._ref = (query && query.ref) || '';
    track('onboarding_enter', this._ref);
    try { await ensureLogin(); } catch (e) { /* 提交时再兜底 */ }
    this._pushAi(OPENER);
    this._applyStageTheme();
  },

  // hero 氛围：跟随 avatarPreset 推导温度档 + 头像同色系光晕（与对话舞台/主页同一套词汇）
  _applyStageTheme() {
    const id = this.data.avatarPreset || 'toon-warm';
    const tier = tierForPreset(id, 'onboarding');
    const p = getPreset(id, 'onboarding');
    const c0 = (p.colors && p.colors[0]) || '#fb923c';
    const c1 = (p.colors && p.colors[1]) || c0;
    this.setData({ stageTier: tier.id, ambStyle: `--amb-a:${c0}; --amb-b:${c1};` });
  },

  _fireDelight() {
    this.setData({ delight: true });
    if (this._delightTimer) clearTimeout(this._delightTimer);
    this._delightTimer = setTimeout(() => { this._delightTimer = null; this.setData({ delight: false }); }, 750);
  },

  onUnload() {
    if (this._delightTimer) { clearTimeout(this._delightTimer); this._delightTimer = null; }
  },

  _push(role, text) {
    const id = 'm' + this.data.msgs.length;
    this.setData({ msgs: this.data.msgs.concat([{ id, role, text }]), toLast: id });
  },
  _pushAi(text) {
    this._push('ai', text);
    this.setData({ avatarState: 'speaking' });
  },

  onInput(e) {
    this.setData({ input: e.detail.value });
  },

  async onSend() {
    if (this.data.thinking || this.data.submitting) return;
    const val = (this.data.input || '').trim();
    if (!val) return;
    if (val.length > 600) {
      wx.showToast({ title: '太长了，分两次说吧', icon: 'none' });
      return;
    }
    this._push('me', val);
    this.setData({ input: '', thinking: true, avatarState: 'thinking', turns: this.data.turns + 1 });
    await this._extract();
  },

  // 供语音输入复用：把转写文本当作一次发送
  sendText(text) {
    const val = (text || '').trim();
    if (!val || this.data.thinking || this.data.submitting) return;
    this.setData({ input: val });
    this.onSend();
  },

  // —— 语音输入（微信同声传译插件 WechatSI）——
  // 需在小程序后台「设置→第三方设置→插件管理」添加「微信同声传译」插件（appid wx069ba97219f66d99）。
  // 插件不可用时静默降级为提示打字，不阻断流程。
  _asrManager() {
    if (this._asr !== undefined) return this._asr;
    try {
      const plugin = requirePlugin('WechatSI');
      const mgr = plugin.getRecordRecognitionManager();
      mgr.onStop = (res) => {
        this.setData({ recording: false });
        const t = ((res && res.result) || '').trim();
        if (t) this.sendText(t);
        else wx.showToast({ title: '没听清，再说一次', icon: 'none' });
      };
      mgr.onError = () => {
        this.setData({ recording: false });
        wx.showToast({ title: '语音识别失败，试试打字', icon: 'none' });
      };
      this._asr = mgr;
    } catch (e) {
      this._asr = null; // 插件未添加 / 不支持
    }
    return this._asr;
  },

  onMicStart() {
    if (this.data.thinking || this.data.submitting) return;
    const mgr = this._asrManager();
    if (!mgr) {
      wx.showToast({ title: '语音暂不可用，请打字', icon: 'none' });
      return;
    }
    this.setData({ recording: true });
    mgr.start({ lang: 'zh_CN' });
  },

  onMicEnd() {
    if (!this.data.recording) return;
    if (this._asr) this._asr.stop(); // 结果在 onStop 回调，并复位 recording
  },

  async _extract() {
    const payload = this.data.msgs.map((m) => ({ role: m.role, text: m.text }));
    try {
      const res = await request({ url: '/profile/extract', method: 'POST', data: { messages: payload } });
      this._applyExtract(res);
    } catch (e) {
      this.setData({ thinking: false, avatarState: 'speaking' });
      this._pushAi('（网络好像有点慢，刚那句再说一遍试试？）');
    }
  },

  _applyExtract(res) {
    const cur = this.data.form;
    // 合并：服务端非空字段覆盖，空则保留已有（防 LLM 偶发漏带）
    const form = {
      realName: res.realName || cur.realName,
      title: res.title || cur.title,
      strengths: res.strengths || cur.strengths,
      recentWork: res.recentWork || cur.recentWork,
      howToKnow: res.howToKnow || cur.howToKnow,
      style: res.style || cur.style,
    };
    const complete = !!(form.realName && form.title && form.strengths);
    const preset = STYLE_AVATAR[form.style] || 'toon-warm'; // 按语气实时变脸

    this.setData({
      form,
      thinking: false,
      canFinish: complete,
      avatarPreset: preset,
      avatarState: 'speaking',
    });
    this._applyStageTheme(); // 氛围随检测到的语气实时变档（与头像变脸同步）

    if (complete) {
      if (!this.data.confirmed) {
        this.setData({ confirmed: true });
        this._fireDelight();
        this._pushAi(`好，我大概了解你了～ 随时可以点「先上岗」，也能再补两句让我更懂你。`);
      } else {
        this._pushAi('记下了～ 还想补充就继续说，或者点「先上岗」。');
      }
      return;
    }
    // 未齐：追问一句
    const ask = (res.followup && res.followup.trim()) || fallbackFollowup(form);
    this._pushAi(ask);
    // 多轮仍未齐，温和提示可手动填写
    if (this.data.turns >= 5) {
      this._pushAi('要是觉得聊着费劲，也可以「切换手动填写」直接填表～');
    }
  },

  onFinishNow() {
    if (!this.data.canFinish || this.data.submitting) return;
    this._finish();
  },

  async _finish() {
    const f = this.data.form;
    if (!f.realName || !f.title || !f.strengths) {
      wx.showToast({ title: '还差一点点信息哦', icon: 'none' });
      return;
    }
    if (this.data.submitting) return;
    this.setData({ submitting: true, avatarState: 'thinking' });
    this._pushAi('好，我这就替你把主页生成出来，稍等 5–15 秒 ✨');
    wx.showLoading({ title: 'AI 生成中…', mask: true });
    try {
      await ensureLogin();
      const body = {
        realName: f.realName,
        title: f.title,
        strengths: f.strengths,
        recentWork: f.recentWork || '',
        howToKnow: f.howToKnow || '',
        style: f.style || '',                 // 语气自动判定，空则由 AI 自行判断
        avatarStyle: STYLE_AVATAR[f.style] || 'toon-warm',
      };
      if (this._ref) body.ref = this._ref;
      const data = await request({ url: '/profile', method: 'POST', data: body });
      wx.hideLoading();
      wx.redirectTo({ url: `/pages/profile/index?id=${data.id}&mine=1&fresh=1` });
    } catch (e) {
      wx.hideLoading();
      this.setData({ submitting: false, avatarState: 'speaking' });
      wx.showModal({ title: '生成失败', content: e.message || '请稍后再试', showCancel: false });
    }
  },

  goForm() {
    wx.redirectTo({ url: '/pages/create/index' });
  },
});
