const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const {
  PRESETS, getPreset, portraitStage, initial, DEFAULT_PRESET_ID,
} = require('../../utils/avatars');

const TITLES = ['顾问·教练', '设计·创意', '开发·技术', '教育·培训', '内容·创作', '品牌·营销', '法律·财税', '生活服务'];
const TAGS = ['策略思考', '内容创作', '品牌增长', '产品设计', '技术开发', '职业成长', '创业实践', '长期主义'];
const TONES = [
  { id: 'steady', label: '专业冷静', desc: '先结论，再依据', tone: '严谨克制，先结论后依据，不寒暄' },
  { id: 'warm', label: '温暖亲和', desc: '友好、有共情', tone: '友好专业，口语化，先共情再回答' },
  { id: 'sharp', label: '犀利直接', desc: '不绕弯、有判断', tone: '直接清晰，有判断，不说空话' },
  { id: 'humorous', label: '轻松幽默', desc: '有趣但不油腻', tone: '轻松有趣，可适度玩笑但不油腻' },
];

function portraitOptions() {
  return PRESETS.map((p) => ({
    id: p.id,
    name: p.name,
    previewUrl: (p.type === 'image' && p.images && p.images.idle) || '',
    style: `background:linear-gradient(145deg, ${(p.colors && p.colors[0]) || '#66776f'}, ${(p.colors && p.colors[1]) || '#d9c18d'});`,
  }));
}

function styleFor(id, seed) {
  const p = getPreset(id, seed);
  const colors = p.colors || ['#66776f', '#d9c18d'];
  return `--card-a:${colors[0]};--card-b:${colors[1] || colors[0]};`;
}

function identity(card) {
  return [card.title, card.company, card.city].filter(Boolean).join(' · ') || '点这里选择你的身份';
}

function introChoices(card) {
  const role = card.title || '这件事';
  return [
    `专注${role}，把复杂问题讲清楚、做扎实。`,
    `关于${role}，我愿意分享真实经验，也能陪你找到下一步。`,
    `正在认真做好${role}，欢迎来聊想法、合作和可能性。`,
  ];
}

function tagChoices(selected) {
  return TAGS.map((label) => ({ label, selected: selected.indexOf(label) >= 0 }));
}

Page({
  data: {
    loading: true,
    saving: false,
    existing: false,
    profileId: '',
    topic: 'identity',
    card: {
      realName: '你的名字', title: '', company: '', city: '',
      initial: '你', avatarUrl: '',
      identity: '点这里选择你的身份',
      oneLiner: '用一句话，让别人立刻知道你是谁、能带来什么。',
      tags: ['会介绍你', '随时在线'],
      avatarStyle: DEFAULT_PRESET_ID, portraitUrl: '', portraitKind: 'identity',
      cardStyle: '--card-a:#6366f1;--card-b:#a855f7;', toneId: 'warm',
    },
    customTag: '',
    titleOptions: TITLES,
    tagOptions: tagChoices(['会介绍你', '随时在线']),
    introChoices: introChoices({ title: '' }),
    toneOptions: TONES,
    portraitOptions: portraitOptions(),
    prompts: {
      identity: '先决定别人第一眼怎么认识你。点一个身份，再补上称呼就好。',
      intro: '哪一句最像你？选中后还可以直接改字。',
      tags: '选 2–5 个你希望别人记住的关键词。',
      portrait: '选择名片气场或虚拟分身形象。缺省使用你的头像/姓名首字，不会自动套用陌生人物。',
      tone: '访客和你的 AI 分身聊天时，希望它怎么说话？',
    },
  },

  async onLoad() {
    try {
      await ensureLogin();
      const mine = await request({ url: '/profile/mine' }).catch(() => null);
      if (!mine) {
        const portrait = portraitStage(DEFAULT_PRESET_ID, 'new-card');
        this.setData({
          loading: false,
          existing: false,
          'card.portraitUrl': portrait.frames.idle,
          'card.portraitKind': portrait.kind,
          'card.cardStyle': styleFor(DEFAULT_PRESET_ID, 'new-card'),
        });
        return;
      }
      this.applyMine(mine);
    } catch (e) {
      wx.switchTab({ url: '/pages/discover/index' });
      return;
    }
    this.setData({ loading: false });
  },

  applyMine(mine) {
    const persona = mine.persona || {};
    const input = mine.personaInput || {};
    const avatarStyle = mine.avatarStyle || DEFAULT_PRESET_ID;
    const portrait = portraitStage(avatarStyle, mine.id);
    const toneHit = TONES.find((t) => t.id === input.style) || TONES.find((t) => (persona.tone || '').indexOf(t.label) >= 0) || TONES[1];
    const card = {
      realName: mine.realName || '你的名字',
      initial: initial(mine.realName), avatarUrl: mine.avatarUrl || '',
      title: mine.title || '', company: mine.company || '', city: mine.city || '',
      oneLiner: persona.oneLiner || '用一句话，让别人立刻知道你是谁、能带来什么。',
      tags: (persona.tags || []).slice(0, 5),
      avatarStyle,
      portraitUrl: portrait.frames.idle,
      portraitKind: portrait.kind,
      cardStyle: styleFor(avatarStyle, mine.id),
      toneId: toneHit.id,
    };
    card.identity = identity(card);
    this.setData({ existing: true, profileId: mine.id, card, tagOptions: tagChoices(card.tags), introChoices: introChoices(card) });
  },

  selectTopic(e) { this.setData({ topic: e.currentTarget.dataset.topic }); },

  onCardField(e) {
    const topic = e.currentTarget.dataset.topic;
    if (topic) this.setData({ topic });
  },

  onInput(e) {
    const key = e.currentTarget.dataset.key;
    const patch = { [`card.${key}`]: e.detail.value };
    if (['title', 'company', 'city'].indexOf(key) >= 0) {
      const card = { ...this.data.card, [key]: e.detail.value };
      patch['card.identity'] = identity(card);
      if (key === 'title') patch.introChoices = introChoices(card);
    }
    if (key === 'realName') patch['card.initial'] = initial(e.detail.value);
    this.setData(patch);
  },

  chooseTitle(e) {
    const title = e.currentTarget.dataset.value;
    const card = { ...this.data.card, title };
    this.setData({ 'card.title': title, 'card.identity': identity(card), introChoices: introChoices(card) });
  },

  chooseIntro(e) { this.setData({ 'card.oneLiner': e.currentTarget.dataset.value }); },

  toggleTag(e) {
    const value = e.currentTarget.dataset.value;
    const tags = this.data.card.tags.slice();
    const i = tags.indexOf(value);
    if (i >= 0) tags.splice(i, 1);
    else if (tags.length < 5) tags.push(value);
    else { wx.showToast({ title: '最多选择 5 个', icon: 'none' }); return; }
    this.setData({ 'card.tags': tags, tagOptions: tagChoices(tags) });
  },

  onCustomTag(e) { this.setData({ customTag: e.detail.value }); },

  addCustomTag() {
    const tag = (this.data.customTag || '').trim();
    if (!tag) return;
    if (tag.length > 8) { wx.showToast({ title: '标签不超过 8 个字', icon: 'none' }); return; }
    if (this.data.card.tags.length >= 5) { wx.showToast({ title: '最多选择 5 个', icon: 'none' }); return; }
    if (this.data.card.tags.indexOf(tag) < 0) {
      const tags = this.data.card.tags.concat(tag);
      this.setData({ 'card.tags': tags, tagOptions: tagChoices(tags), customTag: '' });
    }
  },

  choosePortrait(e) {
    const id = e.currentTarget.dataset.value;
    const seed = this.data.profileId || 'new-card';
    const portrait = portraitStage(id, seed);
    this.setData({
      'card.avatarStyle': id,
      'card.portraitUrl': portrait.frames.idle,
      'card.portraitKind': portrait.kind,
      'card.cardStyle': styleFor(id, seed),
    });
  },

  portraitError() {
    this.setData({ 'card.portraitUrl': '', 'card.portraitKind': 'identity' });
  },

  chooseTone(e) { this.setData({ 'card.toneId': e.currentTarget.dataset.value }); },

  async save() {
    if (this.data.saving) return;
    const c = this.data.card;
    if (!c.realName.trim() || c.realName === '你的名字') { this.setData({ topic: 'identity' }); wx.showToast({ title: '先填写你的名字', icon: 'none' }); return; }
    if (!c.title.trim()) { this.setData({ topic: 'identity' }); wx.showToast({ title: '请选择你的身份', icon: 'none' }); return; }
    if (!c.oneLiner.trim()) { this.setData({ topic: 'intro' }); wx.showToast({ title: '请补一句介绍', icon: 'none' }); return; }
    if (!c.tags.length) { this.setData({ topic: 'tags' }); wx.showToast({ title: '至少选择一个标签', icon: 'none' }); return; }
    const tone = TONES.find((t) => t.id === c.toneId) || TONES[1];
    this.setData({ saving: true });
    wx.showLoading({ title: '正在保存…', mask: true });
    try {
      await ensureLogin();
      let id = this.data.profileId;
      if (this.data.existing) {
        await Promise.all([
          request({ url: '/profile/basic', method: 'PATCH', data: { realName: c.realName, title: c.title, company: c.company, city: c.city } }),
          request({ url: '/profile/persona', method: 'PATCH', data: { oneLiner: c.oneLiner, tags: c.tags, tone: tone.tone } }),
          request({ url: '/profile/avatar', method: 'PATCH', data: { avatarStyle: c.avatarStyle } }),
        ]);
      } else {
        const created = await request({
          url: '/profile', method: 'POST', data: {
            realName: c.realName, title: c.title, company: c.company, city: c.city,
            strengths: c.oneLiner, recentWork: '', howToKnow: '',
            avatarStyle: c.avatarStyle, style: c.toneId,
          },
        });
        id = created.id;
        await request({
          url: '/profile/persona', method: 'PATCH',
          data: { oneLiner: c.oneLiner, tags: c.tags, tone: tone.tone },
        });
      }
      wx.hideLoading();
      wx.showToast({ title: '名片已更新', icon: 'success' });
      setTimeout(() => wx.redirectTo({ url: `/pages/profile/index?id=${id}&mine=1` }), 450);
    } catch (e) {
      wx.hideLoading();
      wx.showToast({ title: e.message || '保存失败', icon: 'none' });
    } finally { this.setData({ saving: false }); }
  },
});
