const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const { PRESETS, getPreset, initial, DEFAULT_LIHE } = require('../../utils/avatars');
const { track } = require('../../utils/track');

// 首页就是用户的成品名片。新用户也先看到一个完整范例，只需补齐少量信息即可替换成自己。
const FALLBACK = {
  chief: {
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
    portraitUrl: DEFAULT_LIHE,
    cardNo: '0001',
    cardStyle: '--card-a:#7772c8; --card-b:#b9dded;',
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
  return `--card-a:${colors[0]}; --card-b:${colors[1] || colors[0]};`;
}

function portraitUrl(avatarStyle, profileId) {
  const preset = getPreset(avatarStyle, profileId);
  return (preset.type === 'image' && preset.images && preset.images.idle) || DEFAULT_LIHE;
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
    draft: {},
    portraitOptions: PORTRAIT_OPTIONS,
  },

  onLoad() {
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this.setData({ statusBarH: info.statusBarHeight || 20 });
    } catch (e) { /* 兜底默认 20 */ }
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 0 });
    }
    this.setData({ greeting: greet() });
    this.load();
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
      this.setData({
        chief: {
          ...FALLBACK.chief,
          name: primary.name || '我的 AI 名片',
          initial: primary.initial || '名',
          oneLiner: primary.line || FALLBACK.chief.oneLiner,
          profileId: primary.profileId,
          hasProfile: true,
          online: true,
          portraitUrl: portraitUrl('', primary.profileId),
          cardNo: cardNo(primary.profileId),
        },
        loading: false,
        loaded: true,
      });
      return;
    }

    const persona = profile.persona || {};
    const chief = {
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
      avatarStyle: profile.avatarStyle || '',
      portraitUrl: portraitUrl(profile.avatarStyle, primary.profileId),
      cardNo: cardNo(primary.profileId),
      cardStyle: cardStyle(profile.avatarStyle, primary.profileId),
      stats: stats ? [
        { n: stats.pv || 0, label: '浏览' },
        { n: stats.uv || 0, label: '访客' },
        { n: stats.askCount || 0, label: '问答' },
      ] : null,
    };

    this.setData({ chief, loading: false, loaded: true });
  },

  retry() { this.load(); },

  enterCardEditor() {
    wx.navigateTo({ url: '/pages/card-editor/index' });
  },

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

  closeSheet() { this.setData({ editorOpen: false }); },

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
      this.setData({
        'chief.avatarStyle': id,
        'chief.portraitUrl': portraitUrl(id, 'new-card'),
        'chief.cardStyle': cardStyle(id, 'new-card'),
        editorOpen: false,
      });
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

  async publishNewCard() {
    if (this.data.savingEditor) return;
    const c = this.data.chief;
    if (!c.title || !c.name || c.name === FALLBACK.chief.name) {
      wx.showToast({ title: '先点姓名，填写你的身份', icon: 'none' });
      this.openIdentity();
      return;
    }
    this.setData({ savingEditor: true });
    wx.showLoading({ title: '正在生成名片…', mask: true });
    try {
      await ensureLogin();
      const created = await request({
        url: '/profile', method: 'POST', data: {
          realName: c.name,
          title: c.title,
          company: c.company || '',
          city: c.city || '',
          strengths: c.oneLiner || (c.allTags || c.tags || []).join('、'),
          recentWork: '',
          howToKnow: (c.allTags || c.tags || []).join('、'),
          avatarStyle: c.avatarStyle || 'gf-meinv',
          style: '',
        },
      });
      const personaPatch = { oneLiner: c.oneLiner };
      if ((c.allTags || []).length) personaPatch.tags = c.allTags;
      await request({ url: '/profile/persona', method: 'PATCH', data: personaPatch }).catch(() => {});
      wx.hideLoading();
      this.setData({ editing: false, editorOpen: false });
      wx.redirectTo({ url: `/pages/profile/index?id=${created.id}&mine=1&fresh=1` });
    } catch (e) {
      wx.hideLoading();
      wx.showToast({ title: e.message || '创建失败', icon: 'none' });
    } finally { this.setData({ savingEditor: false }); }
  },

  portraitError() {
    if (this.data.chief.portraitUrl !== DEFAULT_LIHE) {
      this.setData({ 'chief.portraitUrl': DEFAULT_LIHE });
    }
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
