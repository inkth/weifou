const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const {
  PRESETS, getPreset, portraitStage, tierForPreset, initial, DEFAULT_PRESET_ID,
} = require('../../utils/avatars');
const { track } = require('../../utils/track');

// 首页就是用户的成品名片。新用户也先看到一个完整范例，只需补齐少量信息即可替换成自己。
const DEFAULT_STAGE = cardStage(DEFAULT_PRESET_ID, 'new-card');
const FALLBACK = {
  chief: {
    ...DEFAULT_STAGE,
    name: '你的名字',
    initial: '你',
    identity: '你的身份 · 正在做的事',
    title: '',
    company: '',
    city: '',
    oneLiner: '把你的经历和擅长告诉我，我会替你介绍自己、接住每一次来访。',
    tags: ['会介绍你', '能回答问题', '随时在线'],
    allTags: ['会介绍你', '能回答问题', '随时在线'],
    online: false,
    hasProfile: false,
    profileId: '',
    avatarUrl: '',
    avatarStyle: DEFAULT_PRESET_ID,
    cardNo: '0001',
    stats: null,
  },
};

function greet() {
  const h = new Date().getHours();
  if (h < 6) return '夜深了';
  if (h < 11) return '上午好';
  if (h < 14) return '中午好';
  if (h < 18) return '下午好';
  return '晚上好';
}

function identityLine(profile) {
  return [profile.title, profile.company, profile.city].filter(Boolean).join(' · ') || '一张会说话的 AI 名片';
}

function cardStyle(avatarStyle, profileId) {
  const preset = getPreset(avatarStyle, profileId);
  const colors = preset.colors || ['#7772c8', '#b9dded'];
  return `--card-a:${colors[0]}; --card-b:${colors[1] || colors[0]}; --amb-a:${colors[0]}; --amb-b:${colors[1] || colors[0]};`;
}

function cardStage(avatarStyle, profileId) {
  const portrait = portraitStage(avatarStyle, profileId);
  return {
    cardStyle: cardStyle(avatarStyle, profileId),
    portraitFrames: portrait.frames,
    portraitKind: portrait.kind,
    portraitLabel: portrait.label,
    portraitCapabilities: portrait.capabilities,
    stageTier: tierForPreset(avatarStyle, profileId).id,
  };
}

function cardNo(profileId) {
  const source = String(profileId || '1');
  let sum = 0;
  for (let i = 0; i < source.length; i++) sum = (sum * 31 + source.charCodeAt(i)) % 10000;
  return (`0000${sum || 1}`).slice(-4);
}

const PORTRAIT_OPTIONS = PRESETS.map((p) => ({
  id: p.id,
  name: p.name,
  previewUrl: (p.type === 'image' && p.images && p.images.idle) || '',
  style: `background:linear-gradient(145deg, ${(p.colors && p.colors[0]) || '#7772c8'}, ${(p.colors && p.colors[1]) || '#b9dded'});`,
}));

Page({
  data: {
    statusBarH: 20,
    greeting: '你好',
    chief: FALLBACK.chief,
    loading: true,
    loaded: false,
    errored: false,
    editing: false,
    editorOpen: false,
    editorKind: '',
    editorTitle: '',
    savingEditor: false,
    portraitPose: 'idle',
    portraitMotion: '',
    draft: {},
    portraitOptions: PORTRAIT_OPTIONS,
  },

  onLoad() {
    this._pageVisible = true;
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this.setData({ statusBarH: info.statusBarHeight || 20 });
    } catch (e) { /* 兜底默认 20 */ }
  },

  onShow() {
    this._pageVisible = true;
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 0 });
    }
    this.setData({ greeting: greet() });
    this._loadCard();
  },

  onHide() {
    this._pageVisible = false;
    this._clearCardIdleTimers();
  },

  onUnload() {
    this._pageVisible = false;
    this._clearCardIdleTimers();
  },

  async load() {
    this.setData({ loading: true, errored: false });
    try { await ensureLogin(); } catch (e) { /* 未登录仍展示成品范例 */ }

    let cards = null;
    try { cards = await request({ url: '/home/agents' }); }
    catch (e) {
      const patch = { loading: false, loaded: true, errored: true };
      if (!this.data.loaded) patch.chief = FALLBACK.chief;
      this.setData(patch);
      return;
    }

    const primary = cards && (cards.find((c) => c.primary) || cards[0]);
    if (!primary || !primary.ready || !primary.profileId) {
      this.setData({ chief: FALLBACK.chief, loading: false, loaded: true });
      return;
    }

    const [profile, stats] = await Promise.all([
      request({ url: `/profile/${primary.profileId}` }).catch(() => null),
      request({ url: '/visit/stats/mine' }).catch(() => null),
    ]);

    if (!profile) {
      const stage = cardStage('', primary.profileId);
      this.setData({
        chief: {
          ...FALLBACK.chief,
          ...stage,
          name: primary.name || '我的 AI 名片',
          initial: primary.initial || '名',
          oneLiner: primary.line || FALLBACK.chief.oneLiner,
          profileId: primary.profileId,
          hasProfile: true,
          online: true,
          cardNo: cardNo(primary.profileId),
        },
        loading: false,
        loaded: true,
      });
      return;
    }

    const persona = profile.persona || {};
    const stage = cardStage(profile.avatarStyle, primary.profileId);
    const chief = {
      ...stage,
      name: profile.realName || '我的 AI 名片',
      initial: initial(profile.realName),
      identity: identityLine(profile),
      title: profile.title || '',
      company: profile.company || '',
      city: profile.city || '',
      oneLiner: persona.oneLiner || '我的 AI 分身已在线，关于我的事都可以先问它。',
      tags: (persona.tags || []).slice(0, 3),
      allTags: persona.tags || [],
      online: true,
      hasProfile: true,
      profileId: primary.profileId,
      avatarUrl: profile.avatarUrl || '',
      avatarStyle: profile.avatarStyle || DEFAULT_PRESET_ID,
      cardNo: cardNo(primary.profileId),
      stats: stats ? [
        { n: stats.pv || 0, label: '浏览' },
        { n: stats.uv || 0, label: '访客' },
        { n: stats.askCount || 0, label: '问答' },
      ] : null,
    };

    this.setData({ chief, loading: false, loaded: true });
  },

  _loadCard() {
    this._clearCardIdleTimers();
    return this.load().then(
      () => this._startCardIdle(true),
      () => this._startCardIdle(true),
    );
  },

  retry() { this._loadCard(); },

  _clearCardIdleTimers() {
    if (this._cardPoseTimer) clearTimeout(this._cardPoseTimer);
    if (this._cardPoseResetTimer) clearTimeout(this._cardPoseResetTimer);
    if (this._cardMotionTimer) clearTimeout(this._cardMotionTimer);
    if (this._cardMotionResetTimer) clearTimeout(this._cardMotionResetTimer);
    this._cardPoseTimer = null;
    this._cardPoseResetTimer = null;
    this._cardMotionTimer = null;
    this._cardMotionResetTimer = null;
  },

  _canCardIdle() {
    return this._pageVisible && this.data.loaded && !this.data.loading && !this.data.editorOpen;
  },

  // 与课程舞台一致：idle 帧常驻，有同一张脸的变体帧才会低频覆盖。
  // 当前立绘库多数只有 idle，此时仅保留身体重心微动，不拿课程导师的脸硬套。
  _startCardIdle(reset) {
    this._clearCardIdleTimers();
    if (reset && (this.data.portraitPose !== 'idle' || this.data.portraitMotion)) {
      this.setData({ portraitPose: 'idle', portraitMotion: '' });
    }
    if (!this._canCardIdle()) return;
    this._scheduleCardPose();
    this._scheduleCardMotion();
  },

  _scheduleCardPose() {
    const capabilities = (this.data.chief && this.data.chief.portraitCapabilities) || {};
    const poses = [];
    if (capabilities.blink) poses.push('blink');
    if (capabilities.glance) poses.push('glance');
    if (!poses.length || !this._canCardIdle()) return;
    this._cardPoseTimer = setTimeout(() => {
      if (!this._canCardIdle()) return;
      const pose = poses[Math.floor(Math.random() * poses.length)];
      this.setData({ portraitPose: pose });
      const duration = pose === 'blink' ? 120 : 900;
      this._cardPoseResetTimer = setTimeout(() => {
        if (!this._canCardIdle()) return;
        this.setData({ portraitPose: 'idle' });
        this._scheduleCardPose();
      }, duration);
    }, 4200 + Math.floor(Math.random() * 3800));
  },

  _scheduleCardMotion() {
    if (!this._canCardIdle()) return;
    this._cardMotionTimer = setTimeout(() => {
      if (!this._canCardIdle()) return;
      this.setData({ portraitMotion: 'shift' });
      this._cardMotionResetTimer = setTimeout(() => {
        if (!this._canCardIdle()) return;
        this.setData({ portraitMotion: '' });
        this._scheduleCardMotion();
      }, 2200);
    }, 9000 + Math.floor(Math.random() * 5000));
  },

  enterCardEditor() {
    this.startCardSetup();
  },

  startCardSetup() { wx.navigateTo({ url: '/pages/card-editor/index' }); },

  viewPublicCard() {
    const chief = this.data.chief;
    if (!chief.hasProfile) { this.enterCardEditor(); return; }
    wx.navigateTo({ url: `/pages/profile/index?id=${chief.profileId}&mine=1` });
  },

  async toggleEditor() {
    wx.navigateTo({ url: '/pages/card-editor/index' });
  },

  openIdentity() {
    if (!this.data.editing) return;
    const c = this.data.chief;
    this.setData({ editorOpen: true, editorKind: 'identity', editorTitle: '修改身份资料', draft: {
      realName: c.name, title: c.title || '', company: c.company || '', city: c.city || '',
    } });
  },

  openIntro() {
    if (!this.data.editing) return;
    this.setData({ editorOpen: true, editorKind: 'intro', editorTitle: '修改一句话介绍', draft: { oneLiner: this.data.chief.oneLiner } });
  },

  openTags() {
    if (!this.data.editing) return;
    const tags = this.data.chief.allTags || this.data.chief.tags || [];
    this.setData({ editorOpen: true, editorKind: 'tags', editorTitle: '修改名片标签', draft: { tagsText: tags.join('、') } });
  },

  openPortrait() {
    this.setData({ editorOpen: true, editorKind: 'portrait', editorTitle: '选择名片气质', draft: {} });
  },

  closeSheet() { this.setData({ editorOpen: false }, () => this._startCardIdle(true)); },

  onDraftInput(e) {
    this.setData({ [`draft.${e.currentTarget.dataset.key}`]: e.detail.value });
  },

  async saveEditor() {
    if (this.data.savingEditor) return;
    const kind = this.data.editorKind;
    const d = this.data.draft;
    let url = '/profile/persona';
    let data = {};
    if (kind === 'identity') {
      if (!(d.realName || '').trim() || !(d.title || '').trim()) {
        wx.showToast({ title: '姓名和身份不能为空', icon: 'none' }); return;
      }
      url = '/profile/basic';
      data = { realName: d.realName, title: d.title, company: d.company || '', city: d.city || '' };
    } else if (kind === 'intro') {
      if (!(d.oneLiner || '').trim()) { wx.showToast({ title: '介绍不能为空', icon: 'none' }); return; }
      data = { oneLiner: d.oneLiner };
    } else if (kind === 'tags') {
      const tags = (d.tagsText || '').split(/[，,、]/).map((s) => s.trim()).filter(Boolean).slice(0, 5);
      if (!tags.length) { wx.showToast({ title: '至少保留一个标签', icon: 'none' }); return; }
      data = { tags };
    } else return;
    // 新用户先在眼前的示例名片上完成本地编辑，点右上角“完成”时再一次性创建。
    if (!this.data.chief.hasProfile) {
      if (kind === 'identity') {
        const identity = [data.title, data.company, data.city].filter(Boolean).join(' · ');
        this.setData({
          'chief.name': data.realName.trim(),
          'chief.initial': initial(data.realName),
          'chief.title': data.title.trim(),
          'chief.company': (data.company || '').trim(),
          'chief.city': (data.city || '').trim(),
          'chief.identity': identity,
          editorOpen: false,
        });
      } else if (kind === 'intro') {
        this.setData({ 'chief.oneLiner': data.oneLiner.trim(), editorOpen: false });
      } else if (kind === 'tags') {
        this.setData({ 'chief.tags': data.tags.slice(0, 3), 'chief.allTags': data.tags, editorOpen: false });
      }
      wx.showToast({ title: '已放到名片上', icon: 'success' });
      return;
    }

    this.setData({ savingEditor: true });
    try {
      await request({ url, method: 'PATCH', data });
      this.setData({ editorOpen: false });
      await this.load();
      wx.showToast({ title: '名片已更新', icon: 'success' });
    } catch (e) {
      wx.showToast({ title: e.message || '保存失败', icon: 'none' });
    } finally { this.setData({ savingEditor: false }); }
  },

  async choosePortrait(e) {
    if (this.data.savingEditor) return;
    const id = e.currentTarget.dataset.id;
    if (!this.data.chief.hasProfile) {
      const stage = cardStage(id, 'new-card');
      this.setData({
        'chief.avatarStyle': id,
        'chief.portraitFrames': stage.portraitFrames,
        'chief.cardStyle': stage.cardStyle,
        'chief.stageTier': stage.stageTier,
        portraitPose: 'idle',
        portraitMotion: '',
        editorOpen: false,
      }, () => this._startCardIdle(true));
      wx.showToast({ title: '气质已切换', icon: 'success' });
      return;
    }
    this.setData({ savingEditor: true });
    try {
      await request({ url: '/profile/avatar', method: 'PATCH', data: { avatarStyle: id } });
      this.setData({ editorOpen: false });
      await this.load();
      wx.showToast({ title: '气质已切换', icon: 'success' });
    } catch (err) { wx.showToast({ title: err.message || '切换失败', icon: 'none' }); }
    finally { this.setData({ savingEditor: false }); }
  },

  editCard() {
    wx.navigateTo({ url: '/pages/card-editor/index' });
  },

  portraitError(e) {
    const pose = (e && e.currentTarget && e.currentTarget.dataset.pose) || 'idle';
    const path = `chief.portraitFrames.${pose}`;
    if (pose === 'idle') {
      this.setData({
        [path]: '',
        'chief.portraitKind': 'identity',
        'chief.portraitLabel': '个人身份气场',
        'chief.portraitCapabilities': { blink: false, glance: false, thinking: false, speaking: false },
        portraitPose: 'idle',
      });
      return;
    }
    this.setData({
      [path]: '',
      [`chief.portraitCapabilities.${pose}`]: false,
      portraitPose: 'idle',
    });
  },

  noop() {},

  onShareAppMessage() {
    const chief = this.data.chief;
    if (!chief.hasProfile) {
      return { title: '来微否，做一张会说话的 AI 名片', path: '/pages/discover/index' };
    }
    track('share_tap', chief.profileId, 'home_card');
    return {
      title: `和 ${chief.name} 的 AI 分身聊聊：${chief.oneLiner}`,
      path: `/pages/chat/index?profileId=${chief.profileId}&realName=${encodeURIComponent(chief.name)}&avatarStyle=${chief.avatarStyle || ''}`,
      imageUrl: chief.avatarUrl || undefined,
    };
  },
});
