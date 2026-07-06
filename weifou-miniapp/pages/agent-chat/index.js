const { ensureLogin } = require('../../utils/auth');
const {
  agentDetail, agentSessions, sessionMessages, agentSkill, agentConcepts, chatAgent,
  remindLearn,
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
    freeTier: 0,          // >0：概念课「第一幕免费」模型（配额文案改为「第一幕免费」，不显示剩N次）
    quotaText: '',
    messages: [],
    draft: '',
    options: [],          // 本轮可点选项（服务端从回复剥离下发；点选即发送，输入框兜底）
    voice: false,         // 产出型三门课（开口/言值/驭手）显示麦克风：可按住说，不用打字
    recording: false,     // 语音录制中
    pending: false,
    loading: true,
    errored: false,       // 首屏并发加载失败 → 页内重试骨架（避免白屏）
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
    // 跳转参数带上就先用：name（顶栏）、icon/accent（骨架外壳配色），无网也能画出「像样的空页」
    this.setData({
      id,
      name: query.name ? decodeURIComponent(query.name) : '',
      icon: query.icon ? decodeURIComponent(query.icon) : '',
      accent: query.accent ? decodeURIComponent(query.accent) : '#18b690',
    });
    try {
      await ensureLogin();
    } catch (e) {}
    this._load();
  },

  // 本页数据加载（并发拉 详情/会话/会员/技能/概念）。抽成独立方法，供「网络失败重试」复用。
  // 失败不再只弹 toast——置 errored 让页内出现「重试」骨架，避免慢网/断网时白屏发愣。
  async _load() {
    const id = this.data.id;
    this.setData({ loading: true, errored: false });
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
        freeTier: d.freeTier || 0,
        quotaText: this._quota(member, d.freeTrialRemaining, d.freeTier || 0),
        messages,
        currentSessionId,
        sessions: this._decorate(sessions || []),
        skill: sk && sk.enabled ? sk : null,
        concept: cp && cp.enabled ? this._concept(cp) : null,
        reviewDue: cp && cp.enabled ? (cp.due || 0) : 0,
        // 产出型三门课给语音兜底：开口说英文用 en_US，言值/驭手说中文用 zh_CN。
        voice: ['spoken-english', 'learn-speaking', 'learn-ai'].indexOf(d.slug) >= 0,
        loading: false,
      });
      this._voiceLang = d.slug === 'spoken-english' ? 'en_US' : 'zh_CN';
      if (d.name) wx.setNavigationBarTitle({ title: d.name });
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
      // 白屏止血：不再只弹 toast，改为页内「重试」态（errored），点了重来
      this.setData({ loading: false, errored: true });
    }
  },

  // 网络失败后重试（页内「没能连上，点这里重试」）
  retryLoad() {
    this._load();
  },

  _quota(member, remaining, freeTier) {
    if (member) return '会员 · 畅用';
    if (freeTier > 0) return '第一幕免费 · 会员畅用全部';
    return `免费体验剩 ${remaining} 次`;
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

  // —— 语音输入（微信同声传译插件 WechatSI）：按住麦克风说，松开转写即发 ——
  // 插件不可用时静默降级为提示打字，不阻断。开口课用 en_US 识别，其余用 zh_CN（见 _voiceLang）。
  _asrManager() {
    if (this._asr !== undefined) return this._asr;
    try {
      const plugin = requirePlugin('WechatSI');
      const mgr = plugin.getRecordRecognitionManager();
      mgr.onStop = (res) => {
        this.setData({ recording: false });
        const t = ((res && res.result) || '').trim();
        if (t) this._ask(t);
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
    if (this.data.pending) return;
    const mgr = this._asrManager();
    if (!mgr) { wx.showToast({ title: '语音暂不可用，请打字', icon: 'none' }); return; }
    this.setData({ recording: true });
    mgr.start({ lang: this._voiceLang || 'zh_CN' });
  },

  onMicEnd() {
    if (!this.data.recording) return;
    if (this._asr) this._asr.stop(); // 结果在 onStop 回调，并复位 recording
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
        quotaText: this._quota(member, remaining, this.data.freeTier),
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
          title: this.data.freeTier > 0 ? '第一幕已学完' : '免费体验已用完',
          content: e.message || '开通会员即可畅用全部 AI 助手，不限次数。',
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

  onUnload() { if (this._celebTimer) clearTimeout(this._celebTimer); },

  goMembership() {
    wx.navigateTo({ url: '/pages/membership/index' });
  },

  _scrollEnd() {
    const n = this.data.messages.length;
    if (n > 0) this.setData({ scrollTo: 'm' + (n - 1) });
  },
});
