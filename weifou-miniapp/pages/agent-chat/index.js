const { ensureLogin } = require('../../utils/auth');
const {
  agentDetail, agentSessions, sessionMessages, agentSkill, agentConcepts, chatAgent,
  remindLearn,
} = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');
const { fmtDateTime } = require('../../utils/datetime');
const { requestLearnRemind, LEARN_REMIND_TMPL_ID } = require('../../utils/subscribe');
const { buildNodes, iconForLevel, STATE_TEXT, STATE_CTA } = require('../../utils/learn-nodes');

// streak 里程碑：只在这些天数弹庆祝，其余日子安静（轻而真，不做焦虑轰炸）
const STREAK_MILESTONES = [3, 7, 14, 30, 60, 100];

// 横版「会走的路」几何（rpx）：相邻节点圆心步距 / 轨道左右留白 / 视口宽（rpx 恒为 750）
const ROAD_STEP = 132;
const ROAD_PAD = 80;
const ROAD_VW = 750;
// 目标关滚到视口这个比例处（左侧留出「已走过的路」的成就感）
const ROAD_LEAD = 0.33;

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
    gameSkin: false,      // 学习课（有段位或点亮进度）套「游戏事件卡」皮：固定舞台+事件卡流+底部点选；分身聊天保持原样
    reviewDue: 0,         // 到期待复习的概念数（>0 时进度条上出现「复习挑战」徽章）
    remindState: '',      // 提醒承诺条：'' 隐藏 / offer 邀请 / done 已订
    conceptMapVisible: false, // 概念地图抽屉
    celebrate: null,      // 庆祝浮层：{ up, name, sub }，触发后短暂展示（升级 / 点亮 / 掌握共用）
    // —— 横版「会走的路」（概念课舞台=地图合一）——
    nodes: [],            // 扁平关卡节点 [{ idx,slug,name,state,icon,rx,boss,memberLocked,... }]
    roadSections: [],     // 幕旗 [{ key,tier,theme,tone,flagX }]
    currentIndex: -1,     // 当前关扁平下标（角色停靠点；-1=全通关）
    trackWidth: 0,        // 轨道总宽 rpx
    heroX: 0,             // 角色 translateX rpx
    walking: false,       // 走路动画开关
    roadScrollLeft: 0,    // 横向 scroll 位置 px（rpx→px 已换算）
    roadAnim: false,      // 横向 scroll 是否带动画（首屏 false 瞬移，走路 true 跟随）
    card: null,           // 关卡卡片抽屉
  },

  async onLoad(query) {
    const id = query.id;
    if (!id) {
      this.setData({ loading: false });
      return;
    }
    // rpx→px 系数：横向 scroll-left 单位是 px，而坐标全用 rpx，须换算（漏换算高分屏镜头会偏）
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this._rpx = (info.windowWidth || 375) / 750;
    } catch (e) { this._rpx = 0.5; }
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
        quotaText: this._quota(member),
        messages,
        currentSessionId,
        sessions: this._decorate(sessions || []),
        skill: sk && sk.enabled ? sk : null,
        concept: cp && cp.enabled ? this._concept(cp) : null,
        // 游戏皮（含隐藏顶部商业条）以主接口的 concept/assess 布尔为准——它必然成功（名字/图标都靠它）；
        // 次要的 skill/concept 进度接口带 catch 回落，单独当判据会在慢网时漏出「开通会员」商业条。
        gameSkin: !!(d.concept || d.assess || (sk && sk.enabled) || (cp && cp.enabled)),
        reviewDue: cp && cp.enabled ? (cp.due || 0) : 0,
        // 产出型三门课给语音兜底：开口说英文用 en_US，言值/驭手说中文用 zh_CN。
        voice: ['spoken-english', 'learn-speaking', 'learn-ai'].indexOf(d.slug) >= 0,
        loading: false,
      });
      this._voiceLang = d.slug === 'spoken-english' ? 'en_US' : 'zh_CN';
      if (d.name) wx.setNavigationBarTitle({ title: d.name });
      this._scrollEnd();
      // 概念课：铺横版路，角色停当前关，镜头瞬移定位（首屏不见滚动飞入）
      if (cp && cp.enabled) this._applyRoad(cp, false);
      if (this._autoStart && this._targetConcept) {
        // 兼容旧入口（带 concept&auto=1）：新开一段直接开该关
        this._autoStart = false;
        this.setData({
          messages: this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [],
          currentSessionId: '',
        });
        this._ask('开始这一关', undefined, this._targetConcept);
      } else if (cp && cp.enabled && !currentSessionId && this.data.currentIndex >= 0) {
        // 秒开当前关：无历史会话时自动开场（有会话则续，上面已载入，不重复烧额度）
        const cur = this._roadNodes && this._roadNodes[this.data.currentIndex];
        if (cur && !cur.memberLocked) this._ask('开始这一关', undefined, cur.slug);
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

  // 全站统一「幕门控」：非会员第一幕免费无限、不计次，第二幕起开通会员。
  _quota(member) {
    if (member) return '会员 · 畅用';
    return '第一幕免费 · 会员畅用全部';
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

  // 铺「横版会走的路」：摊平节点 → 补横向坐标 rx → 角色停当前关 → 镜头定位。
  // animate=false 首屏瞬移；true 仅用于罕见的一致性全量校准（见 _ask 末尾）。
  _applyRoad(cp, animate) {
    const built = buildNodes(cp);
    const nodes = built.nodes.map((n) => ({ ...n, rx: ROAD_PAD + n.idx * ROAD_STEP }));
    const TONES = ['butter', 'sky', 'mint', 'lilac'];
    const roadSections = built.sections.map((s, i) => ({
      key: s.key, tier: s.tier, theme: s.theme,
      tone: TONES[i % TONES.length],
      flagX: ROAD_PAD + s.startIndex * ROAD_STEP,
    }));
    const idx = built.currentIndex >= 0 ? built.currentIndex : Math.max(0, nodes.length - 1);
    this._roadNodes = nodes; // 实例副本，供查找/增量，不进 setData
    this.setData({
      nodes,
      roadSections,
      currentIndex: built.currentIndex,
      trackWidth: nodes.length ? ROAD_PAD * 2 + (nodes.length - 1) * ROAD_STEP : 0,
      heroX: ROAD_PAD + idx * ROAD_STEP,
      roadAnim: !!animate,
      roadScrollLeft: this._roadLeft(idx),
    });
  },

  // 目标关对应的横向 scroll-left（px）：把它滚到视口 ~1/3 处
  _roadLeft(idx) {
    const leftRpx = Math.max(0, ROAD_PAD + idx * ROAD_STEP - ROAD_VW * ROAD_LEAD);
    return Math.round(leftRpx * (this._rpx || 0.5));
  },

  // 角色走到目标关：translateX 平移 + 走路 bob + 镜头跟随（rx 与节点同式，对齐恒等不错位）
  _walkTo(destIndex) {
    this.setData({
      walking: true,
      heroX: ROAD_PAD + destIndex * ROAD_STEP,
      roadAnim: true,
      roadScrollLeft: this._roadLeft(destIndex),
    });
    if (this._walkTimer) clearTimeout(this._walkTimer);
    this._walkTimer = setTimeout(() => this.setData({ walking: false }), 720);
  },

  // 点横版节点 → 关卡卡片抽屉（软锁：锁定关也能点开预览，CTA 变「提前解锁」）
  openCard(e) {
    const slug = e.currentTarget.dataset.slug;
    const n = (this._roadNodes || []).find((x) => x.slug === slug);
    if (!n) return;
    const locked = !!n.memberLocked;
    this.setData({ card: {
      ...n,
      stateText: locked ? '🔐 会员解锁' : STATE_TEXT[n.state],
      cta: locked ? '开通会员 · 解锁' : STATE_CTA[n.state],
    } });
  },
  closeCard() { this.setData({ card: null }); },

  // 抽屉 CTA → 同页开这一关（会员锁走会员页）。等价旧 learn-map 的 _go，但不跨页。
  startNode() {
    const c = this.data.card;
    if (!c) return;
    this.setData({ card: null });
    if (c.memberLocked) { this.goMembership(); return; }
    // 新开一段会话 + 带 concept 让教练用该关钩子开场（与秒开一致）
    this.setData({
      messages: this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [],
      currentSessionId: '',
      options: [],
    });
    this._ask('开始这一关', undefined, c.slug);
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
        // 横版路增量点亮 + 角色前进（只 patch 变化的节点，不整表重建，防闪 & 省 setData 体积）
        if (this._roadNodes && this._roadNodes.length) {
          const rn = this._roadNodes;
          const relight = (name, lvl) => {
            const node = rn.find((x) => x.name === name);
            if (!node) return;
            const st = lvl >= 2 ? 'mastered' : 'lit';
            patch['nodes[' + node.idx + '].state'] = st;
            patch['nodes[' + node.idx + '].icon'] = iconForLevel(lvl);
            node.state = st; node.icon = iconForLevel(lvl);
          };
          mastered.forEach((nm) => relight(nm, 2));
          lit.forEach((nm) => relight(nm, 1));
          // 仅当「当前关」本身被点亮/掌握，角色才前进到下一关（复习点亮别处不误前进）
          const oldIdx = this.data.currentIndex;
          const curNode = oldIdx >= 0 ? rn[oldIdx] : null;
          const curDone = curNode && (lit.indexOf(curNode.name) >= 0 || mastered.indexOf(curNode.name) >= 0);
          if (curDone) {
            let ni = -1;
            for (let i = oldIdx + 1; i < rn.length; i++) {
              if (rn[i].state !== 'lit' && rn[i].state !== 'mastered') { ni = i; break; }
            }
            if (ni >= 0) {
              patch['nodes[' + ni + '].state'] = 'current';
              rn[ni].state = 'current';
              patch.currentIndex = ni;
              this._walkDest = ni;
            } else {
              patch.currentIndex = -1; // 全通关：角色停末关
              this._walkDest = rn.length - 1;
            }
          }
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
      // 角色走到新当前关（在 concept setData 落地后，与彩带同拍）
      if (typeof this._walkDest === 'number') {
        const dest = this._walkDest;
        this._walkDest = null;
        this._walkTo(dest);
      }
      // 一致性护栏：极少数情况（复习跨幕等）增量与服务端 current 不符，才全量校准（正常路径不触发）。
      // 必须确认 concept 带完整 tiers，否则精简版会摊平出空路 → 误清空整条路。
      if (data.concept && data.concept.tiers && data.concept.tiers.length && this._roadNodes) {
        const auth = buildNodes({ enabled: true, ...data.concept });
        if (auth.currentIndex !== this.data.currentIndex) {
          this._applyRoad({ enabled: true, ...data.concept }, true);
        }
      }
    } catch (e) {
      this.setData({ pending: false });
      if (e.code === 'MEMBERSHIP_REQUIRED') {
        wx.showModal({
          title: '第一幕已学完',
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

  onUnload() {
    if (this._celebTimer) clearTimeout(this._celebTimer);
    if (this._walkTimer) clearTimeout(this._walkTimer);
  },

  goMembership() {
    wx.navigateTo({ url: '/pages/membership/index' });
  },

  _scrollEnd() {
    const n = this.data.messages.length;
    if (n > 0) this.setData({ scrollTo: 'm' + (n - 1) });
  },
});
