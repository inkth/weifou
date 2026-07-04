const { ensureLogin } = require('../../utils/auth');
const {
  agentDetail, agentSessions, sessionMessages, agentSkill, agentConcepts, chatAgent,
  remindLearn,
  getWork, updateWork, addChapter, updateChapter,
  genMusic, musicStatus, myMusic,
} = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');
const { fmtDateTime } = require('../../utils/datetime');
const { requestLearnRemind, LEARN_REMIND_TMPL_ID } = require('../../utils/subscribe');

// streak 里程碑：只在这些天数弹庆祝，其余日子安静（轻而真，不做焦虑轰炸）
const STREAK_MILESTONES = [3, 7, 14, 30, 60, 100];

Page({
  data: {
    id: '',
    name: '',
    icon: '',
    accent: '#18b690',
    greeting: '',
    member: false,
    remaining: 0,
    quotaText: '',
    messages: [],
    draft: '',
    options: [],          // 本轮可点选项（服务端从回复剥离下发；点选即发送，输入框兜底）
    pending: false,
    loading: true,
    scrollTo: '',
    sessions: [],         // 历史会话（抽屉）
    currentSessionId: '', // 当前续聊的会话；空 = 新开一段（下一条消息创建）
    historyVisible: false,
    skill: null,          // 学习型 Agent 的三维段位档案（null/enabled=false 时不展示）
    concept: null,        // 概念型 Agent 的点亮进度（null/enabled=false 时不展示）
    reviewDue: 0,         // 到期待复习的概念数（>0 时进度条上出现「复习挑战」徽章）
    remindState: '',      // 提醒承诺条：'' 隐藏 / offer 邀请 / done 已订
    conceptMapVisible: false, // 概念地图抽屉
    celebrate: null,      // 庆祝浮层：{ up, name, sub }，触发后短暂展示（升级 / 点亮 / 掌握共用）

    // 写小说
    novel: false,
    work: null,           // 作品视图（含 stage/stages/wordCount/chapters）
    workVisible: false,   // 作品抽屉

    // 做音乐
    music: false,
    songs: [],            // 我的曲库
    libVisible: false,    // 曲库抽屉
    composeVisible: false,// 写词/生成弹层
    draftLyrics: '',
    draftStyle: '',
    draftTitle: '',
    genStatus: '',        // '' | 'generating'：生成中提示
    playingSongId: '',    // 正在播放的歌曲 id
  },

  async onLoad(query) {
    const id = query.id;
    if (!id) {
      this.setData({ loading: false });
      return;
    }
    // 从闯关地图点关进来：记下目标关卡，数据就绪后自动开课
    this._targetConcept = query.concept || '';
    this._autoStart = query.auto === '1';
    this.setData({ id, name: query.name ? decodeURIComponent(query.name) : '' });
    try {
      await ensureLogin();
    } catch (e) {}
    try {
      const [d, sessions, ms, sk, cp] = await Promise.all([
        agentDetail(id),
        agentSessions(id).catch(() => []),
        membershipStatus().catch(() => ({ isMember: false })),
        agentSkill(id).catch(() => ({ enabled: false })),
        agentConcepts(id).catch(() => ({ enabled: false })),
      ]);
      const member = !!ms.isMember;
      // 默认载入最近一段会话；没有则以开场白起新的一段
      let messages = [];
      let currentSessionId = '';
      if (sessions && sessions.length) {
        currentSessionId = sessions[0].sessionId;
        const msgs = await sessionMessages(currentSessionId).catch(() => []);
        messages = (msgs || []).map((m) => ({ role: m.role, content: m.content }));
      }
      if (messages.length === 0 && d.greeting) messages = [{ role: 'assistant', content: d.greeting }];
      this.setData({
        name: d.name,
        icon: d.icon,
        accent: d.accent || '#18b690',
        greeting: d.greeting || '',
        member,
        remaining: d.freeTrialRemaining,
        quotaText: this._quota(member, d.freeTrialRemaining),
        messages,
        currentSessionId,
        sessions: this._decorate(sessions || []),
        skill: sk && sk.enabled ? sk : null,
        concept: cp && cp.enabled ? this._concept(cp) : null,
        reviewDue: cp && cp.enabled ? (cp.due || 0) : 0,
        novel: !!d.novel,
        music: !!d.music,
        loading: false,
      });
      if (d.name) wx.setNavigationBarTitle({ title: d.name });
      if (d.novel) getWork(id).then((w) => this.setData({ work: this._work(w) })).catch(() => {});
      if (d.music) myMusic(id).then((list) => this.setData({ songs: this._decorateSongs(list) })).catch(() => {});
      this._scrollEnd();
      // 指定关卡自动开课：新开一段会话，带 concept 参数让教练直接用这一关的钩子开场
      if (this._autoStart && this._targetConcept) {
        this._autoStart = false;
        this.setData({
          messages: this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [],
          currentSessionId: '',
        });
        this._ask('开始这一关', undefined, this._targetConcept);
      }
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  _quota(member, remaining) {
    return member ? '会员 · 畅用' : `免费体验剩 ${remaining} 次`;
  },

  // 庆祝浮层：升级 / 点亮 / 掌握共用，2.4s 后自动收起。payload = { up, name, sub }
  _celebrate(payload) {
    if (this._celebTimer) clearTimeout(this._celebTimer);
    wx.vibrateShort && wx.vibrateShort({ type: 'medium' });
    this.setData({ celebrate: payload || null });
    this._celebTimer = setTimeout(() => this.setData({ celebrate: null }), 2400);
  },

  // 概念进度装饰：挑出「当前正在攻的档」(第一个未点满的档，都满则最后一档) 给头部进度条用。
  _concept(cp) {
    const tiers = cp.tiers || [];
    const cur = tiers.find((t) => t.lit < t.total) || tiers[tiers.length - 1] || null;
    const curPct = cur && cur.total ? Math.round((cur.lit / cur.total) * 100) : 0;
    const allDone = tiers.length > 0 && tiers.every((t) => t.lit >= t.total);
    return { ...cp, cur, curPct, allDone };
  },

  openConceptMap() {
    if (this.data.concept) this.setData({ conceptMapVisible: true });
  },
  closeConceptMap() {
    this.setData({ conceptMapVisible: false });
  },

  _decorate(sessions) {
    return (sessions || []).map((s) => ({ ...s, timeText: fmtDateTime(s.updatedAt) }));
  },

  onInput(e) {
    this.setData({ draft: e.detail.value });
  },

  send() {
    const content = (this.data.draft || '').trim();
    if (!content) return;
    this.setData({ draft: '' });
    this._ask(content);
  },

  // 复习挑战：快问快答已点亮概念（检索练习，答对保住/升级档位）。
  startReview() {
    if (this.data.pending) return;
    this.setData({ reviewDue: 0 }); // 先清徽章：防连点，真实到期数下次进页刷新
    this._ask('开始复习挑战', 'review');
  },

  // —— 提醒承诺（implementation intention）：一天只邀请一次，点了才弹微信授权 ——
  _remindKey() {
    const d = new Date();
    const day = `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`;
    return `weifou_learn_remind_${day}`;
  },
  _remindAskedToday() {
    try { return !!wx.getStorageSync(this._remindKey()); } catch (e) { return false; }
  },
  async onRemindTap() {
    try { wx.setStorageSync(this._remindKey(), 1); } catch (e) {} // 无论结果，今天不再邀请
    const res = await requestLearnRemind();
    if (res && res[LEARN_REMIND_TMPL_ID] === 'accept') {
      try {
        const r = await remindLearn(this.data.id);
        this.setData({ remindState: 'done' });
        wx.showToast({ title: `说定了，${(r && r.sendAt) || '明天'}见`, icon: 'none' });
        setTimeout(() => this.setData({ remindState: '' }), 2600);
      } catch (e) {
        this.setData({ remindState: '' });
      }
    } else {
      this.setData({ remindState: '' }); // 拒绝/跳过：安静收起，明天再说
    }
  },

  // 点选选项 = 原样作为回答发送（点选为主、输入兜底）
  pickOption(e) {
    const t = e.currentTarget.dataset.t;
    if (t) this._ask(t);
  },

  async _ask(content, mode, concept) {
    if (!content || this.data.pending) return;
    this.setData({
      pending: true,
      options: [],
      messages: this.data.messages.concat({ role: 'user', content }),
    });
    this._scrollEnd();
    try {
      const data = await chatAgent(this.data.id, content, this.data.currentSessionId, mode, concept);
      const member = !!data.member;
      const remaining = member ? this.data.remaining : data.remaining;
      const patch = {
        messages: this.data.messages.concat({ role: 'assistant', content: data.answer }),
        member,
        remaining,
        quotaText: this._quota(member, remaining),
        // 新开一段时服务端回传新建会话 id，记下来后续消息续到同一段
        currentSessionId: data.sessionId || this.data.currentSessionId,
        options: data.options || [],
        pending: false,
      };
      // 学习型 Agent：更新三维段位，升级时弹庆祝浮层
      if (data.skill) {
        patch.skill = { enabled: true, ...data.skill };
        if (data.levelUp) {
          this._celebrate({ up: '升级！', name: data.skill.levelName, sub: '你的英语又上了一个台阶' });
        }
      }
      // 概念型 Agent：更新点亮进度。庆祝优先级：打通一档 > 掌握新概念 > 点亮新概念
      if (data.concept) {
        patch.concept = this._concept({ enabled: true, ...data.concept });
        const cleared = data.tierCleared || [];
        const mastered = data.newlyMastered || [];
        const lit = data.newlyLit || [];
        if (cleared.length) {
          this._celebrate({ up: '打通一档！', name: `${cleared[0]}篇`, sub: `你已通关「${cleared[0]}」——继续下一档` });
        } else if (mastered.length) {
          this._celebrate({ up: '掌握！', name: mastered[0], sub: mastered.length > 1 ? `x${mastered.length} 连击！${mastered.length} 个概念你已能讲透` : '你已经能自己讲透它了' });
        } else if (lit.length) {
          this._celebrate({ up: '点亮！', name: lit[0], sub: lit.length > 1 ? `x${lit.length} 连击！本轮点亮 ${lit.length} 个` : '又一个被你打开了' });
        }
      }
      // 连续学习：里程碑庆祝 + 补签提示 + 今日首学时递上「明天叫我」的提醒承诺
      if (data.streak && data.streak.newDay) {
        const days = data.streak.days || 0;
        if (data.streak.freeze) {
          wx.showToast({ title: '自动补签，连续没断 🔥', icon: 'none' });
        } else if (STREAK_MILESTONES.indexOf(days) >= 0) {
          this._celebrate({ up: '连续学习', name: `${days} 天`, sub: '能把学习变成习惯的人是少数，你在其中' });
        }
        if (LEARN_REMIND_TMPL_ID && !this.data.remindState && !this._remindAskedToday()) {
          patch.remindState = 'offer';
        }
      }
      this.setData(patch);
      this._scrollEnd();
    } catch (e) {
      this.setData({ pending: false });
      if (e.code === 'MEMBERSHIP_REQUIRED') {
        wx.showModal({
          title: '免费体验已用完',
          content: '开通会员即可畅用全部 AI 助手，不限次数。',
          confirmText: '去开通',
          cancelText: '再看看',
          success: (r) => {
            if (r.confirm) this.goMembership();
          },
        });
      } else {
        this.setData({
          messages: this.data.messages.concat({ role: 'assistant', content: e.message || '出错了，请稍后再试' }),
        });
        this._scrollEnd();
      }
    }
  },

  // —— 多会话 ——
  // 新开一段：清回开场白、清空 currentSessionId，下一条消息会创建新会话
  newSession() {
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    this.setData({ messages, currentSessionId: '', historyVisible: false, options: [] });
    this._scrollEnd();
  },

  async openHistory() {
    this.setData({ historyVisible: true });
    try {
      const sessions = await agentSessions(this.data.id);
      this.setData({ sessions: this._decorate(sessions || []) });
    } catch (e) {}
  },

  closeHistory() {
    this.setData({ historyVisible: false });
  },

  noop() {},

  async switchSession(e) {
    const sid = e.currentTarget.dataset.id;
    if (!sid) return;
    if (sid === this.data.currentSessionId) {
      this.setData({ historyVisible: false });
      return;
    }
    this.setData({ historyVisible: false });
    try {
      const msgs = await sessionMessages(sid);
      this.setData({
        messages: (msgs || []).map((m) => ({ role: m.role, content: m.content })),
        currentSessionId: sid,
        options: [],
      });
      this._scrollEnd();
    } catch (err) {
      wx.showToast({ title: '加载失败', icon: 'none' });
    }
  },

  // ==================== 写小说 ====================
  _work(w) {
    if (!w) return null;
    const wc = w.wordCount || 0;
    const wanText = wc >= 10000 ? (wc / 10000).toFixed(1) + '万字' : wc + '字';
    return { ...w, wanText };
  },
  openWork() { if (this.data.work) this.setData({ workVisible: true }); },
  closeWork() { this.setData({ workVisible: false }); },

  _deriveTitle(content) {
    const first = (content || '').split('\n').map((s) => s.trim()).find((s) => s) || '';
    const clean = first.replace(/^[#>*\-\s]+/, '').slice(0, 20);
    return clean || '新章节';
  },

  // assistant 气泡：存为大纲
  async saveAsOutline(e) {
    const idx = e.currentTarget.dataset.idx;
    const content = (this.data.messages[idx] || {}).content;
    if (!content) return;
    const hadOutline = !!(this.data.work && this.data.work.outline);
    try {
      await updateWork(this.data.id, { outline: content });
      const w = this._work(await getWork(this.data.id));
      this.setData({ work: w });
      wx.showToast({ title: '已存为大纲', icon: 'success' });
      if (!hadOutline) this._celebrate({ up: '大纲成型', name: '大纲', sub: '骨架搭好，开始写正文吧' });
    } catch (err) { wx.showToast({ title: err.message || '保存失败', icon: 'none' }); }
  },

  // assistant 气泡：存为新一章
  async saveAsChapter(e) {
    const idx = e.currentTarget.dataset.idx;
    const content = (this.data.messages[idx] || {}).content;
    if (!content) return;
    const prevWord = this.data.work ? this.data.work.wordCount : 0;
    try {
      await addChapter(this.data.id, { title: this._deriveTitle(content), content });
      const w = this._work(await getWork(this.data.id));
      this.setData({ work: w });
      wx.showToast({ title: '已存入作品', icon: 'success' });
      if (prevWord < 10000 && w.wordCount >= 10000) {
        this._celebrate({ up: '里程碑', name: '破万字', sub: '你的小说破一万字了！' });
      } else {
        this._celebrate({ up: '写完一章', name: '第 ' + w.chapterCount + ' 章', sub: '钩子留好，继续往下写' });
      }
    } catch (err) { wx.showToast({ title: err.message || '保存失败', icon: 'none' }); }
  },

  // 编辑作品字段（title/logline/genre/outline）
  editWorkField(e) {
    const field = e.currentTarget.dataset.field;
    const labels = { title: '标题', logline: '一句话立意', genre: '题材', outline: '大纲' };
    const cur = (this.data.work || {})[field] || '';
    wx.showModal({
      title: '编辑' + (labels[field] || ''), editable: true, content: cur,
      placeholderText: '写点什么…',
      success: async (r) => {
        if (!r.confirm) return;
        try {
          await updateWork(this.data.id, { [field]: r.content });
          this.setData({ work: this._work(await getWork(this.data.id)) });
        } catch (err) { wx.showToast({ title: err.message || '保存失败', icon: 'none' }); }
      },
    });
  },

  editChapter(e) {
    const { cid, content } = e.currentTarget.dataset;
    wx.showModal({
      title: '编辑章节', editable: true, content: content || '',
      success: async (r) => {
        if (!r.confirm) return;
        try {
          await updateChapter(this.data.id, cid, { content: r.content });
          this.setData({ work: this._work(await getWork(this.data.id)) });
        } catch (err) { wx.showToast({ title: err.message || '保存失败', icon: 'none' }); }
      },
    });
  },

  async toggleFinalized() {
    const cur = !!(this.data.work && this.data.work.finalized);
    try {
      await updateWork(this.data.id, { finalized: !cur });
      const w = this._work(await getWork(this.data.id));
      this.setData({ work: w });
      if (!cur) this._celebrate({ up: '定稿！', name: w.title || '你的小说', sub: '恭喜完成一部作品 🎉' });
    } catch (err) { wx.showToast({ title: err.message || '操作失败', icon: 'none' }); }
  },

  // ==================== 做音乐 ====================
  _decorateSongs(list) {
    return (list || []).map((s) => ({ ...s, timeText: fmtDateTime(s.createdAt) }));
  },
  openCompose() {
    let lyrics = this.data.draftLyrics;
    if (!lyrics) {
      for (let i = this.data.messages.length - 1; i >= 0; i--) {
        if (this.data.messages[i].role === 'assistant') { lyrics = this.data.messages[i].content; break; }
      }
    }
    this.setData({ composeVisible: true, draftLyrics: lyrics || '' });
  },
  closeCompose() { this.setData({ composeVisible: false }); },
  onLyrics(e) { this.setData({ draftLyrics: e.detail.value }); },
  onStyle(e) { this.setData({ draftStyle: e.detail.value }); },
  onTitle(e) { this.setData({ draftTitle: e.detail.value }); },

  async doGenerate() {
    const lyrics = (this.data.draftLyrics || '').trim();
    if (!lyrics) { wx.showToast({ title: '先写好歌词', icon: 'none' }); return; }
    this.setData({ composeVisible: false, genStatus: 'generating' });
    try {
      const { songId } = await genMusic(this.data.id, { lyrics, style: this.data.draftStyle, title: this.data.draftTitle });
      this._pollSong(songId, 0);
    } catch (e) {
      this.setData({ genStatus: '' });
      if (e.code === 'MEMBERSHIP_REQUIRED') {
        wx.showModal({ title: '免费体验已用完', content: '开通会员即可畅用，不限次数。', confirmText: '去开通', cancelText: '再看看',
          success: (r) => { if (r.confirm) this.goMembership(); } });
      } else { wx.showToast({ title: e.message || '生成失败', icon: 'none' }); }
    }
  },

  _pollSong(songId, tries) {
    if (tries > 40) { this.setData({ genStatus: '' }); wx.showToast({ title: '生成较慢，稍后到曲库查看', icon: 'none' }); return; }
    musicStatus(songId).then((s) => {
      if (s.status === 'done') {
        const songs = this._decorateSongs([s].concat(this.data.songs.filter((x) => x.songId !== s.songId)));
        this.setData({ genStatus: '', songs });
        this._celebrate({ up: '歌曲生成', name: s.title || '新歌', sub: '点开曲库就能听 🎧' });
        this._play(s.audioUrl, s.songId);
      } else if (s.status === 'failed') {
        this.setData({ genStatus: '' });
        wx.showToast({ title: s.err || '生成失败', icon: 'none' });
      } else {
        setTimeout(() => this._pollSong(songId, tries + 1), 3000);
      }
    }).catch(() => setTimeout(() => this._pollSong(songId, tries + 1), 3000));
  },

  openLibrary() { myMusic(this.data.id).then((l) => this.setData({ libVisible: true, songs: this._decorateSongs(l) })).catch(() => this.setData({ libVisible: true })); },
  closeLibrary() { this.setData({ libVisible: false }); },
  playSong(e) { this._play(e.currentTarget.dataset.src, e.currentTarget.dataset.id); },

  _audio() {
    if (this._ac) return this._ac;
    const ac = wx.createInnerAudioContext();
    ac.onEnded(() => this.setData({ playingSongId: '' }));
    ac.onStop(() => this.setData({ playingSongId: '' }));
    ac.onError(() => { this.setData({ playingSongId: '' }); wx.showToast({ title: '播放失败', icon: 'none' }); });
    this._ac = ac;
    return ac;
  },
  _play(src, songId) {
    if (!src) return;
    if (this.data.playingSongId === (songId || src)) { this._stopAudio(); return; }
    const ac = this._audio();
    ac.src = src;
    ac.play();
    this.setData({ playingSongId: songId || src });
  },
  _stopAudio() { if (this._ac) this._ac.stop(); this.setData({ playingSongId: '' }); },
  onUnload() { if (this._ac) { this._ac.stop(); this._ac.destroy(); } if (this._celebTimer) clearTimeout(this._celebTimer); },

  goMembership() {
    wx.navigateTo({ url: '/pages/membership/index' });
  },

  _scrollEnd() {
    const n = this.data.messages.length;
    if (n > 0) this.setData({ scrollTo: 'm' + (n - 1) });
  },
});
