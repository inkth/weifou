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
// PAD 须 ≥ HERO_GAP + 主角半宽（33），否则第一关的对峙位会被视口左沿裁掉半张脸
const ROAD_STEP = 132;
const ROAD_PAD = 144;
const ROAD_VW = 750;
// 目标关滚到视口这个比例处（左侧留出「已走过的路」的成就感）
const ROAD_LEAD = 0.33;
// 主角站当前关左侧这个距离处对峙（不再踩在关卡上）——关卡即对手，出招才有的放矢
const HERO_GAP = 80;

// 战斗动作层只给判断型课程（题型=看穿一个说法，关卡天然是对手）。
// 开口（英语）是开口练习、知常（道德经）调性不合，都保持纯行走舞台。
const DUEL_SLUGS = {
  'learn-psychology': 1, 'learn-logic': 1, 'learn-marketing': 1,
  'learn-ai': 1, 'learn-speaking': 1,
};

// 吉祥物「负鼠」姿势图（scripts/gen-mascot.mjs 派生自同一张母版）。它是玩家小人、不是课程角色，
// 所以全部概念课共用一只——与「身份面不戴别人的脸」不冲突：那条规矩管的是分身立绘。
const MASCOT = {
  idle:  '/assets/mascot/possum_idle.webp',
  walk:  '/assets/mascot/possum_walk.webp',
  atk:   '/assets/mascot/possum_atk.webp',
  win:   '/assets/mascot/possum_win.webp',
  dead:  '/assets/mascot/possum_dead.webp',
  think: '/assets/mascot/possum_think.webp',
};

Page({
  data: {
    id: '',
    name: '',
    guide: '',
    icon: '',
    accent: '#7772c8',
    greeting: '',
    member: false,
    remaining: 0,
    freeTier: 0,          // >0：概念课「第一幕免费」模型（配额文案改为「第一幕免费」，不显示剩N次）
    quotaText: '',
    messages: [],
    options: [],          // 本轮可点选项 [{ t, en }]（服务端剥离下发；点选即答；en=英文句可朗读）
    englishAudio: false,  // 仅英语课（spoken-english）：英文选项 + 示范句配🔊朗读、示范句自动播
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
    mapVisible: false,    // 学习中默认收起地图；点击紧凑进度头才展开
    lessonCard: null,     // 单卡学习：当前助手任务/反馈，不再把整段聊天铺在屏幕上
    lessonChoice: '',     // 本轮用户选择，作为卡片内的轻量上下文
    lessonTitle: '当前练习',
    lessonProgress: '',
    reviewDue: 0,         // 到期待复习的概念数（>0 时进度条上出现「复习挑战」徽章）
    remindState: '',      // 提醒承诺条：'' 隐藏 / offer 邀请 / done 已订
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
    // —— 战斗动作层（判断型概念课）：舞台不承载信息，只把对话事件翻译成身体语言 ——
    heroAct: '',          // 主角动作：'atk' 出招冲刺 / 'win' 击破欢呼 / 'dead' 装死诈尸
    foeAct: '',           // 对手动作：'hit' 受击 / 'taunt' 挑衅（答对未拿下）/ 'strike' 反击（答错）/ 'down' 被击破
    duelIdx: -1,          // 对手＝哪个节点（通常=currentIndex；击破瞬间锁旧关，防状态翻转后丢动画）
    heroImg: MASCOT.idle, // 负鼠当前姿势图（由 _pose 按 动作>走路>思考>待机 的优先级算出）
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
      accent: query.accent ? decodeURIComponent(query.accent) : '#7772c8',
      // 入口已知是课（修炼页带 game=1）就首帧套游戏皮，避免数据回来前闪一下普通聊天顶栏；
      // 加载完成后仍以主接口布尔为准回正（见 _load）
      gameSkin: query.game === '1' || this.data.gameSkin,
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
      // 英语课（开口 spoken-english）专属：英文句配朗读。早于建 messages 置位，供 _decorateMsgs 用。
      this._englishAudio = !!(d && d.slug === 'spoken-english');
      // 默认载入最近一段会话；没有则以开场白起新的一段
      let messages = [];
      let currentSessionId = '';
      if (sessions && sessions.length) {
        currentSessionId = sessions[0].sessionId;
        const msgs = await sessionMessages(currentSessionId).catch(() => []);
        messages = this._decorateMsgs((msgs || []).map((m) => ({ role: m.role, content: m.content })));
      }
      if (messages.length === 0 && d.greeting) messages = [{ role: 'assistant', content: d.greeting }];
      const lesson = this._lessonFocus(messages);
      this.setData({
        name: d.name,
        guide: d.guide || '',
        icon: d.icon,
        accent: d.accent || '#7772c8',
        greeting: d.greeting || '',
        member,
        remaining: d.freeTrialRemaining,
        freeTier: d.freeTier || 0,
        quotaText: this._quota(member),
        messages,
        lessonCard: lesson.card,
        lessonChoice: lesson.choice,
        lessonTitle: d.guide || d.name || '当前练习',
        lessonProgress: sk && sk.enabled ? `Lv.${sk.level || 1}` : '',
        currentSessionId,
        sessions: this._decorate(sessions || []),
        skill: sk && sk.enabled ? sk : null,
        concept: cp && cp.enabled ? this._concept(cp) : null,
        // 游戏皮（含隐藏顶部商业条）以主接口的 concept/assess 布尔为准——它必然成功（名字/图标都靠它）；
        // 次要的 skill/concept 进度接口带 catch 回落，单独当判据会在慢网时漏出「开通会员」商业条。
        gameSkin: !!(d.concept || d.assess || (sk && sk.enabled) || (cp && cp.enabled)),
        reviewDue: cp && cp.enabled ? (cp.due || 0) : 0,
        englishAudio: this._englishAudio,
        loading: false,
      });
      if (d.name) wx.setNavigationBarTitle({ title: d.name });
      // 战斗动作层开关：按 slug 门控（详情接口必回 slug）
      this._duelOn = !!DUEL_SLUGS[d.slug];
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
    return '第一幕免费 · 全课会员解锁后续';
  },

  // 庆祝浮层只用于升级 / 掌握 / 通关里程碑，普通点亮保持安静。
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
      heroX: ROAD_PAD + idx * ROAD_STEP - HERO_GAP,
      roadAnim: !!animate,
      roadScrollLeft: this._roadLeft(idx),
      lessonTitle: nodes[idx] ? nodes[idx].name : '全部完成',
      lessonProgress: nodes.length ? `第 ${Math.min(idx + 1, nodes.length)} / ${nodes.length} 关` : '',
    });
  },

  // 学习页只呈现最后一张助手卡；完整消息仍保留用于历史与服务端上下文。
  _lessonFocus(messages) {
    const list = messages || [];
    let card = null;
    let choice = '';
    let cardIndex = -1;
    for (let i = list.length - 1; i >= 0; i--) {
      if (!card && list[i].role === 'assistant') {
        card = list[i];
        cardIndex = i;
        continue;
      }
      if (card && i < cardIndex && list[i].role === 'user') {
        choice = list[i].content || '';
        break;
      }
    }
    return { card, choice };
  },

  toggleMap() {
    const visible = !this.data.mapVisible;
    this.setData({ mapVisible: visible }, () => {
      if (visible && this.data.concept) this.recenterRoad();
    });
  },

  // 目标关对应的横向 scroll-left（px）：把它滚到视口 ~1/3 处
  _roadLeft(idx) {
    const leftRpx = Math.max(0, ROAD_PAD + idx * ROAD_STEP - ROAD_VW * ROAD_LEAD);
    return Math.round(leftRpx * (this._rpx || 0.5));
  },

  // 角色走到目标关的对峙位：translateX 平移 + 走路 bob + 镜头跟随（rx 与节点同式，对齐恒等不错位）
  _walkTo(destIndex) {
    this.setData({
      walking: true,
      heroX: ROAD_PAD + destIndex * ROAD_STEP - HERO_GAP,
      roadAnim: true,
      roadScrollLeft: this._roadLeft(destIndex),
      duelIdx: destIndex, // 新对手就位
      heroImg: this._pose(this.data.heroAct, true, this.data.pending),
    });
    if (this._walkTimer) clearTimeout(this._walkTimer);
    this._walkTimer = setTimeout(() => this.setData({
      walking: false,
      heroImg: this._pose(this.data.heroAct, false, this.data.pending),
    }), 720);
  },

  // 负鼠该摆哪个姿势：动作 > 走路 > 思考 > 待机。
  // 动作压过 pending，所以「点选出招 → 服务端还在想」会先冲刺、动作播完再自然落到挠头。
  _pose(act, walking, pending) {
    if (act && MASCOT[act]) return MASCOT[act];
    if (walking) return MASCOT.walk;
    if (pending) return MASCOT.think;
    return MASCOT.idle;
  },

  // 战斗动作一拍：切姿势图 + 设动作类名，一次性播完自清。idx 缺省＝当前关。
  // 姿势由图管（装死图本身就是四脚朝天），CSS 只管位移与时序，两边不要都做同一件事。
  _duel(heroAct, foeAct, ms, idx) {
    if (this._duelTimer) clearTimeout(this._duelTimer);
    const act = heroAct || '';
    this.setData({
      heroAct: act,
      foeAct: foeAct || '',
      duelIdx: typeof idx === 'number' ? idx : this.data.currentIndex,
      heroImg: this._pose(act, this.data.walking, this.data.pending),
    });
    this._duelTimer = setTimeout(() => this.setData({
      heroAct: '', foeAct: '',
      heroImg: this._pose('', this.data.walking, this.data.pending),
    }), ms || 520);
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
      cta: locked ? '解锁完整路径' : STATE_CTA[n.state],
    } });
  },
  closeCard() { this.setData({ card: null }); },

  // 抽屉 CTA → 同页开这一关（会员锁走会员页）。等价旧 learn-map 的 _go，但不跨页。
  startNode() {
    const c = this.data.card;
    if (!c) return;
    this._reviewing = false; // 开新关＝回到对峙，战斗动作层恢复
    this.setData({ card: null });
    if (c.memberLocked) { this.goMembership(); return; }
    // 新开一段会话 + 带 concept 让教练用该关钩子开场（与秒开一致）
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    const lesson = this._lessonFocus(messages);
    this.setData({
      messages,
      lessonCard: lesson.card,
      lessonChoice: '',
      lessonTitle: c.name || '当前练习',
      mapVisible: false,
      currentSessionId: '',
      options: [],
    });
    this._ask('开始这一关', undefined, c.slug);
  },

  // 点进度条＝镜头拉回当前关（用户横滚看远处后的「回家」键）。
  // 路即唯一地图：图鉴抽屉与 learn-map 页已退役，总览/跳转都在路上完成。
  recenterRoad() {
    if (!this.data.nodes.length) return;
    const idx = this.data.currentIndex >= 0 ? this.data.currentIndex : this.data.nodes.length - 1;
    let left = this._roadLeft(idx);
    // 手滚不回写 data，同值 setData 是 no-op 滚不回来——同值时挪 1px 强制生效
    if (left === this.data.roadScrollLeft) left += 1;
    this.setData({ roadAnim: true, roadScrollLeft: left });
  },

  _decorate(sessions) {
    return (sessions || []).map((s) => ({ ...s, timeText: fmtDateTime(s.updatedAt) }));
  },

  // 复习挑战：快问快答已点亮概念（检索练习，答对保住/升级档位）。
  startReview() {
    if (this.data.pending) return;
    this._reviewing = true; // 复习态：战斗动作层歇场，开新关/新会话时复位
    this.setData({ reviewDue: 0, mapVisible: false, lessonTitle: '复习挑战', lessonProgress: '快速复习' }); // 先清徽章：防连点
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
    if (!t || this.data.pending) return;
    // 英语课（纯点选）：点选即答的同时把这句英文读出来——「选一句就听一句」的自动播放。
    // 老结构靠跟读 Say 节点自动播；重构去掉 Say 后，改由点选驱动（见 _ask 里 sayLine 兜底）。
    if (this._englishAudio && this._isEnglishText(t)) this._speak(t);
    // 出招：点选即答，主角向当前关冲一记、对手受击晃动。
    // 复习挑战不打当前关（考的是别处的概念，冲错对象比不冲更违和）。
    if (this._duelOn && !this._reviewing && this.data.currentIndex >= 0) {
      this._answerTurn = true;
      this._duel('atk', 'hit', 520);
    }
    this._ask(t);
  },

  async _ask(content, mode, concept) {
    if (!content || this.data.pending) return;
    const visibleChoice = (content === '开始这一关' || content === '开始复习挑战') ? '' : content;
    const askedMessages = this.data.messages.concat({ role: 'user', content });
    this.setData({
      pending: true,
      options: [],
      messages: askedMessages,
      lessonChoice: visibleChoice,
      // 出招动作未播完时不抢姿势（_pose 里动作压过 pending），播完自然落到挠头
      heroImg: this._pose(this.data.heroAct, this.data.walking, true),
    });
    this._scrollEnd();
    try {
      const data = await chatAgent(this.data.id, content, this.data.currentSessionId, mode, concept);
      const member = !!data.member;
      const remaining = member ? this.data.remaining : data.remaining;
      const answerMsg = this._decorateMsgs([{ role: 'assistant', content: data.answer }])[0];
      const answeredMessages = this.data.messages.concat(answerMsg);
      const patch = {
        messages: answeredMessages,
        lessonCard: answerMsg,
        lessonChoice: visibleChoice,
        member,
        remaining,
        quotaText: this._quota(member, remaining, this.data.freeTier),
        // 新开一段时服务端回传新建会话 id，记下来后续消息续到同一段
        currentSessionId: data.sessionId || this.data.currentSessionId,
        options: this._toOpts(data.options),
        pending: false,
        heroImg: this._pose(this.data.heroAct, this.data.walking, false),
      };
      // 学习型 Agent：更新三维段位，升级时弹庆祝浮层
      if (data.skill) {
        patch.skill = { enabled: true, ...data.skill };
        if (data.levelUp) {
          this._celebrate({ up: '升级！', name: data.skill.levelName, sub: '你的英语又上了一个台阶' });
        }
      }
      // 概念型 Agent：更新点亮进度。仅打通一档或真正掌握时庆祝。
      if (data.concept) {
        patch.concept = this._concept({ enabled: true, ...data.concept });
        const cleared = data.tierCleared || [];
        const mastered = data.newlyMastered || [];
        const lit = data.newlyLit || [];
        if (cleared.length) {
          this._celebrate({ up: '打通一档！', name: `${cleared[0]}篇`, sub: `你已通关「${cleared[0]}」——继续下一档` });
        } else if (mastered.length) {
          this._celebrate({ up: '掌握！', name: mastered[0], sub: mastered.length > 1 ? `x${mastered.length} 连击！${mastered.length} 个概念你已能讲透` : '你已经能自己讲透它了' });
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
          if (curDone) this._duelWin = oldIdx; // 击破动画锁定旧关（patch 落地后 currentIndex 已翻页）
          if (curDone) {
            let ni = -1;
            for (let i = oldIdx + 1; i < rn.length; i++) {
              if (rn[i].state !== 'lit' && rn[i].state !== 'mastered') { ni = i; break; }
            }
            if (ni >= 0) {
              patch['nodes[' + ni + '].state'] = 'current';
              rn[ni].state = 'current';
              patch.currentIndex = ni;
              patch.lessonTitle = rn[ni].name;
              patch.lessonProgress = `第 ${ni + 1} / ${rn.length} 关`;
              this._walkDest = ni;
            } else {
              patch.currentIndex = -1; // 全通关：角色停末关
              patch.lessonTitle = '全部完成';
              patch.lessonProgress = `${rn.length} / ${rn.length} 关`;
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
      // 英语课：新回合若带「示范句」（跟读目标），自动朗读一遍——学员先听发音再开口
      const sayLine = this._extractSay(data.answer);
      if (sayLine) this._speak(sayLine);
      // 战斗动作层第二拍（第一拍「出招」在 pickOption 即时播完）。按服务端 verdict 分派：
      // 点亮＝击破 > 答错＝反击+装死诈尸 > 答对未拿下＝对手挑衅还站着 > 教学轮＝安静。
      // verdict 为空（讲解/开场/模型拿不准）时不演戏——舞台宁可静，也不要每轮乱晃。
      const won = typeof this._duelWin === 'number';
      if (this._duelOn && !this._reviewing) {
        if (won) {
          this._duel('win', 'down', 660, this._duelWin);
        } else if (this._answerTurn && this.data.currentIndex >= 0) {
          const v = data.verdict || '';
          if (v === 'wrong') this._duel('dead', 'strike', 1500);
          else if (v === 'right') this._duel('', 'taunt', 560);
        }
      }
      this._answerTurn = false;
      this._duelWin = null;
      // 角色走到新当前关（在 concept setData 落地后，与彩带同拍；有击破戏时先播完再起步）
      if (typeof this._walkDest === 'number') {
        const dest = this._walkDest;
        this._walkDest = null;
        if (this._duelOn && won) {
          if (this._walkDelay) clearTimeout(this._walkDelay);
          this._walkDelay = setTimeout(() => this._walkTo(dest), 700);
        } else {
          this._walkTo(dest);
        }
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
      this._answerTurn = false;
      this._duelWin = null;
      this.setData({ pending: false, heroImg: this._pose(this.data.heroAct, this.data.walking, false) });
      if (e.code === 'MEMBERSHIP_REQUIRED') {
        wx.showModal({
          title: '第一幕已学完',
          content: e.message || '加入人类基本功计划，继续后续能力路径。',
          confirmText: '去开通',
          cancelText: '再看看',
          success: (r) => {
            if (r.confirm) this.goMembership();
          },
        });
      } else {
        const errorMsg = { role: 'assistant', content: e.message || '出错了，请稍后再试' };
        this.setData({
          messages: this.data.messages.concat(errorMsg),
          lessonCard: errorMsg,
        });
        this._scrollEnd();
      }
    }
  },

  // —— 多会话 ——
  // 新开一段：清回开场白、清空 currentSessionId，下一条消息会创建新会话
  newSession() {
    this._reviewing = false;
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    const lesson = this._lessonFocus(messages);
    this.setData({ messages, lessonCard: lesson.card, lessonChoice: '', currentSessionId: '', historyVisible: false, options: [], mapVisible: false });
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
    this._reviewing = false;
    this.setData({ historyVisible: false });
    try {
      const msgs = await sessionMessages(sid);
      const messages = this._decorateMsgs((msgs || []).map((m) => ({ role: m.role, content: m.content })));
      const lesson = this._lessonFocus(messages);
      this.setData({
        messages,
        lessonCard: lesson.card,
        lessonChoice: lesson.choice,
        currentSessionId: sid,
        options: [],
        mapVisible: false,
      });
      this._scrollEnd();
    } catch (err) {
      wx.showToast({ title: '加载失败', icon: 'none' });
    }
  },

  onUnload() {
    if (this._celebTimer) clearTimeout(this._celebTimer);
    if (this._walkTimer) clearTimeout(this._walkTimer);
    if (this._duelTimer) clearTimeout(this._duelTimer);
    if (this._walkDelay) clearTimeout(this._walkDelay);
    if (this._audio) { this._audio.destroy(); this._audio = null; }
  },

  // ============ 英语课 · 英文朗读（TTS） ============
  // 只在开口课（spoken-english）启用：英文选项句可点🔊听，跟读示范句自动播 + 可重播。
  // 语音走微信同声传译插件 WechatSI 的 textToSpeech（en_US），零成本、与 onboarding 的 ASR 同插件。

  // 一个字符串是否「该配朗读」：纯英文（无汉字）且含字母。检验/复习的中文点选项天然排除。
  _isEnglishText(s) {
    return !!s && !/[一-龥]/.test(s) && /[A-Za-z]/.test(s);
  },

  // 把服务端下发的选项字符串数组映射为 [{ t, en }]：en=英语课里的英文句（配🔊）。
  _toOpts(list) {
    const on = this._englishAudio;
    return (list || []).map((t) => ({ t, en: on && this._isEnglishText(t) }));
  },

  // 从助手气泡里抽出「跟读示范句」：剧本把它放在「读出这句：\n」/「读一遍：\n」标记后的那一行。
  // 非英语课或无标记返回空串（气泡照常渲染，只是不挂朗读）。
  _extractSay(content) {
    if (!this._englishAudio || !content) return '';
    const marks = ['读出这句：\n', '读一遍：\n'];
    let pos = -1, mlen = 0;
    marks.forEach((mk) => { const p = content.lastIndexOf(mk); if (p > pos) { pos = p; mlen = mk.length; } });
    if (pos < 0) return '';
    return (content.slice(pos + mlen).split('\n')[0] || '').trim();
  },

  // 给助手消息挂上 say（示范句），供气泡下方渲染🔊「听发音」。非英语课原样返回。
  _decorateMsgs(list) {
    if (!this._englishAudio) return list || [];
    return (list || []).map((m) => (m.role === 'assistant' ? { ...m, say: this._extractSay(m.content) } : m));
  },

  _siPlugin() {
    if (this._si !== undefined) return this._si;
    try { this._si = requirePlugin('WechatSI'); } catch (e) { this._si = null; } // 插件未添加/不支持
    return this._si;
  },

  _audioCtx() {
    if (!this._audio) {
      const a = wx.createInnerAudioContext();
      a.obeyMuteSwitch = false; // 语言课：静音键也要能听发音
      this._audio = a;
    }
    return this._audio;
  },

  // 朗读一句英文：TTS 合成临时音频 → innerAudioContext 播放。按文本缓存 filename，重播不再走网络。
  _speak(text) {
    const clean = (text || '').trim();
    if (!clean) return;
    this._ttsCache = this._ttsCache || {};
    const play = (src) => { const a = this._audioCtx(); a.stop(); a.src = src; a.play(); };
    if (this._ttsCache[clean]) { play(this._ttsCache[clean]); return; }
    const plugin = this._siPlugin();
    if (!plugin || !plugin.textToSpeech) { wx.showToast({ title: '朗读暂不可用', icon: 'none' }); return; }
    plugin.textToSpeech({
      lang: 'en_US', tts: true, content: clean,
      success: (res) => { const f = res && res.filename; if (f) { this._ttsCache[clean] = f; play(f); } },
      fail: () => { wx.showToast({ title: '朗读失败，再试一次', icon: 'none' }); },
    });
  },

  // 选项上的🔊：只朗读，不作答（catchtap 拦截，不触发 pickOption）
  onSpeakChip(e) { this._speak(e.currentTarget.dataset.t); },
  // 气泡下方的🔊：重播这一关的示范句
  onSpeakSay(e) { this._speak(e.currentTarget.dataset.t); },

  goMembership() {
    wx.navigateTo({ url: '/pages/membership/index' });
  },

  _scrollEnd() {
    const n = this.data.messages.length;
    if (n > 0) this.setData({ scrollTo: 'm' + (n - 1) });
  },
});
