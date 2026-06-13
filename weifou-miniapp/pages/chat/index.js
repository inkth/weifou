const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fenToYuan } = require('../../utils/pay');
const { fmtDateTime } = require('../../utils/datetime');
const { bookConsult } = require('../../utils/consult');
const { track } = require('../../utils/track');

// 从小程序码 scene（URL-encoded 的 "id=xxx"）解析 profileId
function parseScene(scene) {
  if (!scene) return '';
  const decoded = decodeURIComponent(scene);
  const m = decoded.match(/(?:^|&)id=([^&]+)/);
  return m ? m[1] : '';
}

Page({
  data: {
    profileId: '',
    realName: '',
    avatarStyle: '',
    oneLiner: '',
    starters: [], // AI 生成的引导问题（点了即移除，避免重复）
    startersRevealed: false, // 开场动画结束后才淡入引导问题
    answeredOnce: false, // 首轮回答后才出现行动 chips，开场聚焦"开始聊"
    contactAvailable: false,
    consultAvailable: false,
    isMine: false, // 主人预览自己的 AI（profile 页透传 mine=1）
    hasOwnProfile: true, // 默认 true：确认是"无主页访客"前不展示裂变钩子
    notFound: false,
    messages: [],
    draft: '',
    pending: false,
    introState: '', // 开场动画期间覆盖 avatar 状态（thinking/speaking）
    acting: false, // 对话内成交卡片支付中
    showInput: false, // “自己问”展开自由输入
  },

  async onLoad(query) {
    // chat 是访客首落点（chat-first）：profileId 三来源归一
    // 1) 站内跳转 query.profileId  2) 兼容 profile 风格的 query.id  3) 海报小程序码 query.scene
    const profileId = query.profileId || query.id || parseScene(query.scene);
    if (!profileId) {
      this.setData({ notFound: true });
      return;
    }
    this.setData({
      profileId,
      realName: query.realName ? decodeURIComponent(query.realName) : '', // 空=骨架占位，等接口补齐
      avatarStyle: query.avatarStyle || '',
      isMine: query.mine === '1',
      introState: 'thinking',
    });

    try {
      await ensureLogin();
    } catch (e) {}

    // 访客到访统计落在对话入口（profile 页只在非 chat 来源时上报，避免重复）
    request({ url: `/visit/${profileId}`, method: 'POST' }).catch(() => {});

    // 裂变钩子前提：访客自己还没有 Agent（失败时保持 true，宁可不展示）
    request({ url: '/user/me' })
      .then((me) => this.setData({ hasOwnProfile: !!me.profileId }))
      .catch(() => {});

    // 拉取主页资料：开场白 + 引导选项 + 是否可联系
    try {
      const p = await request({ url: `/profile/${profileId}` });
      const persona = p.persona || {};
      // persona.greeting 是 AI 生成的第一人称开场白；早期记录可能为空，兜底拼接
      const greeting = persona.greeting
        || (persona.oneLiner
          ? `你好，我是 ${p.realName} 的 AI 助理。${persona.oneLiner}\n想了解些什么？`
          : `你好，我是 ${p.realName} 的 AI 助理，TA 的事都可以问我。`);
      this.setData({
        realName: p.realName || this.data.realName,
        avatarStyle: this.data.avatarStyle || p.avatarStyle || '',
        oneLiner: persona.oneLiner || '',
        starters: persona.starters || [],
        contactAvailable: !!p.contactVisible,
      });
      // 打字机开场只惊艳一次：回访（含主人反复自测）直接显示全文，把效率还给第二次
      const introKey = `weifou_intro_${profileId}`;
      const introOneLiner = persona.greeting ? persona.oneLiner : '';
      if (wx.getStorageSync(introKey)) {
        const msgs = [{ role: 'assistant', content: greeting }];
        if (introOneLiner) msgs.push({ role: 'assistant', content: `“${introOneLiner}”` });
        this.setData({ messages: msgs, introState: '', startersRevealed: true });
      } else {
        try { wx.setStorageSync(introKey, 1); } catch (e) {}
        this._playIntro(greeting, introOneLiner);
      }
    } catch (e) {
      this.setData({ notFound: true, introState: '' });
      return;
    }

    // 是否开放付费咨询（用于行动选项）
    try {
      const pricing = await request({ url: `/consult/pricing/${profileId}` });
      this.setData({ consultAvailable: !!pricing.enabled });
    } catch (e) {}
  },

  onUnload() {
    this._abortIntro();
  },

  // —— 开场三拍：thinking → greeting 打字机 → 一句话介绍 → starters 淡入 ——
  async _playIntro(greeting, oneLiner) {
    this._introAborted = false;
    await this._wait(300);
    this.setData({ introState: 'speaking', messages: [{ role: 'assistant', content: '' }] });
    await this._typeMessage(0, greeting);
    if (oneLiner && !this._introAborted) {
      await this._wait(400);
      this.setData({
        messages: this.data.messages.concat({ role: 'assistant', content: '' }),
      });
      await this._typeMessage(this.data.messages.length - 1, `“${oneLiner}”`);
    }
    this.setData({ introState: '', startersRevealed: true });
  },

  // 打字机渲染单条消息；用路径 setData 只更新该条，避免整组重设
  _typeMessage(index, fullText) {
    return new Promise((resolve) => {
      if (this._introAborted) {
        this.setData({ [`messages[${index}].content`]: fullText });
        return resolve();
      }
      let i = 0;
      this._typeTimer = setInterval(() => {
        if (this._introAborted) {
          clearInterval(this._typeTimer);
          this.setData({ [`messages[${index}].content`]: fullText });
          return resolve();
        }
        i += 3;
        this.setData({ [`messages[${index}].content`]: fullText.slice(0, i) });
        if (i >= fullText.length) {
          clearInterval(this._typeTimer);
          resolve();
        }
      }, 40);
    });
  },

  _wait(ms) {
    return new Promise((r) => {
      this._waitTimer = setTimeout(r, ms);
    });
  },

  // 用户提前交互（点 chip / 输入）→ 立即放完动画，绝不让用户等
  _abortIntro() {
    this._introAborted = true;
    if (this._typeTimer) clearInterval(this._typeTimer);
    if (this._waitTimer) clearTimeout(this._waitTimer);
    if (!this.data.startersRevealed) {
      this.setData({ introState: '', startersRevealed: true });
    }
  },

  // 头部 → 名片/转化中心（带 from=chat 闸门，防 profile 分流弹回死循环）
  goProfile() {
    if (!this.data.profileId) return;
    const mine = this.data.isMine ? '&mine=1' : '';
    wx.navigateTo({ url: `/pages/profile/index?id=${this.data.profileId}&from=chat${mine}` });
  },

  onShareAppMessage() {
    track('share_tap', this.data.profileId, 'chat');
    const name = this.data.realName || 'TA';
    const title = this.data.oneLiner
      ? `和 ${name} 的 AI 助理聊聊：${this.data.oneLiner}`
      : `加微信前，先和 ${name} 的 AI 助理聊聊`;
    return {
      title,
      path: `/pages/chat/index?profileId=${this.data.profileId}&realName=${encodeURIComponent(name)}&avatarStyle=${this.data.avatarStyle || ''}`,
    };
  },

  // 点引导选项 → 作为问题发送
  pickStarter(e) {
    this._abortIntro();
    const q = e.currentTarget.dataset.q;
    const rest = this.data.starters.filter((s) => s !== q);
    this.setData({ starters: rest });
    this.ask(q);
  },

  toggleInput() {
    this._abortIntro();
    this.setData({ showInput: !this.data.showInput });
  },

  onInput(e) {
    this.setData({ draft: e.detail.value });
  },

  send() {
    const content = (this.data.draft || '').trim();
    if (!content) return;
    this.setData({ draft: '' });
    this.ask(content);
  },

  async ask(content) {
    if (!content || this.data.pending) return;
    this._abortIntro();
    try {
      await ensureLogin();
    } catch (e) {
      wx.showToast({ title: '请先登录', icon: 'none' });
      return;
    }

    const messages = this.data.messages.concat({ role: 'user', content });
    this.setData({ messages, pending: true });

    try {
      const data = await request({
        url: `/chat/${this.data.profileId}/ask`,
        method: 'POST',
        data: { content },
      });
      const msg = { role: 'assistant', content: data.answer };
      if (data.card && data.card.type === 'consult_offer') {
        msg.card = this.decorateCard(data.card);
      }
      this.setData({
        messages: this.data.messages.concat(msg),
        pending: false,
        answeredOnce: true,
      });
      // 轻线索：访客被第 2 个回答"种草"后，给一条不抢戏的"我也要一个"入口（仅一次）
      this._answerCount = (this._answerCount || 0) + 1;
      if (this._answerCount === 2) this._showOwnHook('light');
    } catch (e) {
      this.setData({ pending: false, answeredOnce: true });
      const tip = e.code === 'CHAT_QUOTA_EXCEEDED' ? e.message : e.message || '请求失败';
      this.setData({
        messages: this.data.messages.concat({ role: 'assistant', content: tip }),
      });
    }
  },

  // 把后端档期卡片加工成可展示结构（时间文案 + 价格）
  decorateCard(card) {
    const slots = (card.slots || []).map((s) => ({
      id: s.id,
      durationMin: s.durationMin,
      timeText: fmtDateTime(s.startAt),
      priceYuan: fenToYuan(s.durationMin === 60 ? card.price60 : card.price30),
    }));
    return { type: card.type, slots, selectedSlotId: '' };
  },

  // 选择卡片里的某个档期
  selectCardSlot(e) {
    const mi = e.currentTarget.dataset.mi;
    const id = e.currentTarget.dataset.id;
    const messages = this.data.messages.slice();
    messages[mi] = { ...messages[mi], card: { ...messages[mi].card, selectedSlotId: id } };
    this.setData({ messages });
  },

  // 对话内直接预约并支付（成交闭环）
  async bookFromCard(e) {
    const mi = e.currentTarget.dataset.mi;
    const card = this.data.messages[mi] && this.data.messages[mi].card;
    if (!card || !card.selectedSlotId || this.data.acting) return;
    this.setData({ acting: true });
    try {
      await bookConsult(this.data.profileId, card.selectedSlotId, 'chat_card');
      this.setData({
        acting: false,
        messages: this.data.messages.concat({
          role: 'assistant',
          content: '预约成功 ✅ 可在「我的通话」按时进入。',
          action: 'go-sessions',
        }),
      });
      wx.showToast({ title: '预约成功', icon: 'success' });
      this._showOwnHook('strong');
    } catch (err) {
      this.setData({ acting: false });
      if (err.code !== 'PAY_CANCEL') {
        wx.showToast({ title: err.message || '预约失败', icon: 'none' });
      }
    }
  },

  goSessions() {
    wx.navigateTo({ url: '/pages/sessions/index' });
  },

  // —— 裂变钩子：体验过别人的 Agent → 想要自己的 ——
  // light：对话中的一行系统线索；strong：目的达成（留资/预约/拿到联系方式）后的身份反转 CTA。
  // 各只出现一次；主人预览、已有主页的访客一律不展示，绝不打断与 Agent 的核心互动。
  _showOwnHook(kind) {
    if (this.data.isMine || this.data.hasOwnProfile) return;
    if (kind === 'light') {
      if (this._lightHookShown || this._strongHookShown) return;
      this._lightHookShown = true;
      this.setData({
        messages: this.data.messages.concat({ role: 'system', content: '', action: 'create-own' }),
      });
    } else {
      if (this._strongHookShown) return;
      this._strongHookShown = true;
      this.setData({
        messages: this.data.messages.concat({ role: 'system', content: '', action: 'create-own-strong' }),
      });
    }
    track('own_hook_show', this.data.profileId, kind);
  },

  goCreateOwn() {
    track('own_hook_click', this.data.profileId, 'chat');
    wx.navigateTo({ url: `/pages/create/index?quick=1&ref=${this.data.profileId}` });
  },

  // 行动选项：联系本人（内联弹层）
  async onContact() {
    try {
      const c = await request({ url: `/profile/${this.data.profileId}/contact` });
      const lines = [];
      if (c.wechat) lines.push(`微信：${c.wechat}`);
      if (c.phone) lines.push(`电话：${c.phone}`);
      wx.showModal({
        title: '联系本人',
        content: lines.join('\n') || '本人未填写联系方式',
        confirmText: '复制',
        cancelText: '关闭',
        success: (r) => {
          if (r.confirm) {
            wx.setClipboardData({ data: c.wechat || c.phone || '' });
            this._showOwnHook('strong');
          }
        },
      });
    } catch (e) {
      wx.showToast({ title: e.message || '本人未公开联系方式', icon: 'none' });
    }
  },

  // 行动选项：留个话给 TA（零门槛留资 → 主人侧线索）
  async onLeaveMessage() {
    try {
      await ensureLogin();
    } catch (e) {
      wx.showToast({ title: '请先登录', icon: 'none' });
      return;
    }
    wx.showModal({
      title: `留个话给 ${this.data.realName}`,
      editable: true,
      placeholderText: '想说的话 / 怎么联系你，TA 会看到',
      confirmText: '发送',
      success: async (r) => {
        if (!r.confirm) return;
        const note = (r.content || '').trim();
        if (!note) {
          wx.showToast({ title: '请填写留言', icon: 'none' });
          return;
        }
        try {
          await request({
            url: `/chat/${this.data.profileId}/lead`,
            method: 'POST',
            data: { note },
          });
          this.setData({
            messages: this.data.messages.concat({
              role: 'assistant',
              content: `已把你的话转达给 ${this.data.realName}，TA 会尽快看到 ✅`,
            }),
          });
          wx.showToast({ title: '已送达', icon: 'success' });
          this._showOwnHook('strong');
        } catch (err) {
          wx.showToast({ title: err.message || '发送失败', icon: 'none' });
        }
      },
    });
  },

  // 行动选项：预约咨询 / 打赏 → 主页（转化中心）
  // chat-first 后 chat 是首落点，导航栈里通常没有 profile，直接前进打开。
  // from=chat 必带：否则 profile 的访客分流会把页面弹回 chat 造成死循环。
  goProfileAction() {
    wx.navigateTo({ url: `/pages/profile/index?id=${this.data.profileId}&from=chat` });
  },
});
