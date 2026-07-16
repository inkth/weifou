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

// 听力门标记（与服务端 script_learn_english.go 的 listenMark 同一约定）：
// 英语课消息里该标记行的下一行英文「只播不显」——正文遮成占位，自动朗读，🔊 可重听。
const LISTEN_MARK = '🎧 只听不看：';
const LISTEN_MASK = '▂ ▂ ▂ ▂ ▂ ▂ ▂ ▂';

// 课程语音偏好跨课程保留；首次默认自动播放，但只在用户主动进入学习页后触发。
const VOICE_PREFS_KEY = 'weifou_lesson_voice_prefs';

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

// 同一数字人的低帧率待机状态。全部预加载后只切透明度，避免眼神切换时闪白。
const MENTOR_PORTRAITS = {
  idle: '/assets/avatars/course-mentor-idle-v2-540.webp',
  blink: '/assets/avatars/course-mentor-blink-540.webp',
  glance: '/assets/avatars/course-mentor-glance-540.webp',
  speakSmall: '/assets/avatars/course-mentor-speak-small-540.webp',
  speakWide: '/assets/avatars/course-mentor-speak-wide-540.webp',
  speakRound: '/assets/avatars/course-mentor-speak-round-540.webp',
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
    mapVisible: false,    // 学习中默认收起课程地图；点击进度后以底部抽屉展开
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
    mentorSpeaking: false,// 数字人语音播放状态，只驱动轻量口播反馈
    mentorPaused: false,  // 语音被用户暂停，与播放结束分开，供“继续”操作
    voiceAutoPlay: true,  // 新讲解默认自动播放（跨课程持久化）
    voiceMuted: false,    // 静音仅影响课程语音，不阻断答题与课程进度
    answerFeedback: '',   // '' / right / wrong：原位短反馈，不叠全屏庆祝
    mentorPose: 'idle',   // idle / blink / glance：待机时低频切换
    mentorMotion: '',     // '' / shift：长时等待时的轻微重心调整
    mentorPhase: 'idle',  // idle / listening / thinking / speaking / affirm / reassure
    mentorMouth: 'closed',// closed / small / wide / round：按朗读文本节奏驱动嘴型
    mentorIdleHint: false,// 45 秒无操作后仅出现一次的安静提示
    mentorPortraits: MENTOR_PORTRAITS,
  },

  async onLoad(query) {
    this._pageDestroyed = false;
    this._pageVisible = true;
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
    const voicePrefs = this._readVoicePrefs();
    // 跳转参数带上就先用：name（顶栏）、icon/accent（骨架外壳配色），无网也能画出「像样的空页」
    this.setData({
      id,
      name: query.name ? decodeURIComponent(query.name) : '',
      icon: query.icon ? decodeURIComponent(query.icon) : '',
      accent: query.accent ? decodeURIComponent(query.accent) : '#7772c8',
      // 入口已知是课（修炼页带 game=1）就首帧套游戏皮，避免数据回来前闪一下普通聊天顶栏；
      // 加载完成后仍以主接口布尔为准回正（见 _load）
      gameSkin: query.game === '1' || this.data.gameSkin,
      voiceAutoPlay: voicePrefs.autoPlay,
      voiceMuted: voicePrefs.muted,
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
      let restoredOptions = [];
      if (sessions && sessions.length) {
        currentSessionId = sessions[0].sessionId;
        const msgs = await sessionMessages(currentSessionId).catch(() => []);
        messages = this._decorateMsgs((msgs || []).map((m) => ({ role: m.role, content: m.content })));
        // 恢复「最后一张助手卡」（与 _lessonFocus 取的卡一致）的点选项——服务端随消息回传。
        // 本课纯点选无输入栏，不回填气泡则复原的卡片没法作答成死局；老会话未存 options 时为空，走页面兜底重开。
        for (let i = (msgs || []).length - 1; i >= 0; i--) {
          if (msgs[i].role !== 'assistant') continue;
          if (msgs[i].options && msgs[i].options.length) restoredOptions = this._toOpts(msgs[i].options);
          break;
        }
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
        lessonProgress: sk && sk.enabled ? `阶段 ${sk.level || 1}` : '',
        options: restoredOptions,
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
      // 课程页改为数字人陪学，不再触发对峙/战斗表演。
      this._duelOn = false;
      this._scrollEnd();
      // 概念课：铺横版路，角色停当前关，镜头瞬移定位（首屏不见滚动飞入）
      if (cp && cp.enabled) this._applyRoad(cp, false);
      let openingRound = false;
      if (this._autoStart && this._targetConcept) {
        // 兼容旧入口（带 concept&auto=1）：新开一段直接开该关
        this._autoStart = false;
        this.setData({
          messages: this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [],
          currentSessionId: '',
        });
        openingRound = true;
        this._ask('开始这一关', undefined, this._targetConcept);
      } else if (cp && cp.enabled && !currentSessionId && this.data.currentIndex >= 0) {
        // 秒开当前关：无历史会话时自动开场（有会话则续，上面已载入，不重复烧额度）
        const cur = this._roadNodes && this._roadNodes[this.data.currentIndex];
        if (cur && !cur.memberLocked) {
          openingRound = true;
          this._ask('开始这一关', undefined, cur.slug);
        }
      }
      // 有新开场请求时由 _ask 播新讲解；续学时则播当前卡片，不重复抢音频。
      if (!openingRound && this.data.gameSkin && lesson.card) this._autoSpeakLesson(lesson.card);
      setTimeout(() => this._startMentorIdle(true), 0);
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
    this._stopMentorIdle(true);
    if (this._celebTimer) clearTimeout(this._celebTimer);
    wx.vibrateShort && wx.vibrateShort({ type: 'medium' });
    this.setData({ celebrate: payload || null });
    this._celebTimer = setTimeout(() => {
      this.setData({ celebrate: null }, () => this._startMentorIdle(true));
    }, 2400);
  },

  // 章末知识卡片：Boss 关通关专属，不自动消失——用户主动点掉或分享。
  _showChapterCard(node) {
    this._stopMentorIdle(true);
    wx.vibrateShort && wx.vibrateShort({ type: 'medium' });
    this.setData({
      chapterCard: {
        name: node.name, takeaway: node.takeaway, source: node.source,
        accent: this.data.accent,
      },
    });
  },
  onCloseChapterCard() {
    this.setData({ chapterCard: null }, () => this._startMentorIdle(true));
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
      lessonTitle: nodes[idx] ? nodes[idx].name : '课程已完成',
      lessonProgress: nodes.length ? `第 ${Math.min(idx + 1, nodes.length)} / ${nodes.length} 节` : '',
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
    if (visible) this._stopMentorIdle(true);
    this.setData({ mapVisible: visible }, () => {
      if (!visible) this._startMentorIdle(true);
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

  // 兜底：复原出的卡片没有点选项（老会话未存 options / 或安全拦截清空）时，纯点选课会成死局——
  // 给一个「重新开始这一关」，新开一段以当前关钩子重开，跟秒开/抽屉开关走同一条路。
  restartLevel() {
    if (this.data.pending) return;
    const idx = this.data.currentIndex;
    const cur = this._roadNodes && idx >= 0 ? this._roadNodes[idx] : null;
    if (cur && cur.memberLocked) { this.goMembership(); return; }
    this._reviewing = false;
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    this.setData({
      messages,
      lessonCard: this._lessonFocus(messages).card,
      lessonChoice: '',
      lessonTitle: cur ? cur.name : this.data.lessonTitle,
      currentSessionId: '',
      options: [],
      mapVisible: false,
    });
    this._ask('开始这一关', undefined, cur ? cur.slug : undefined);
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
    if (this._englishAudio && this._isEnglishText(t) && this.data.voiceAutoPlay && !this.data.voiceMuted) {
      this._speak(t);
    }
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
    this._stopMentorIdle(true);
    this._clearMentorThinkTimer();
    const visibleChoice = (content === '开始这一关' || content === '开始复习挑战') ? '' : content;
    const askedMessages = this.data.messages.concat({ role: 'user', content });
    this.setData({
      pending: true,
      answerFeedback: '',
      mentorPhase: 'listening',
      mentorPose: 'idle',
      options: [],
      messages: askedMessages,
      lessonChoice: visibleChoice,
      // 出招动作未播完时不抢姿势（_pose 里动作压过 pending），播完自然落到挠头
      heroImg: this._pose(this.data.heroAct, this.data.walking, true),
    });
    // 先用一个很短的“我听到了”身体反馈承接点击，再进入思考，避免点击后人物立即僵住。
    this._mentorThinkTimer = setTimeout(() => {
      if (!this._pageDestroyed && this.data.pending) {
        this.setData({ mentorPhase: 'thinking', mentorPose: 'glance' });
      }
    }, 320);
    this._scrollEnd();
    try {
      const data = await chatAgent(this.data.id, content, this.data.currentSessionId, mode, concept);
      this._clearMentorThinkTimer();
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
        mentorPhase: data.verdict === 'right' ? 'affirm' : (data.verdict === 'wrong' ? 'reassure' : 'idle'),
        mentorPose: 'idle',
        heroImg: this._pose(this.data.heroAct, this.data.walking, false),
      };
      let milestone = false;
      // 学习型 Agent：更新三维段位，升级时弹庆祝浮层
      if (data.skill) {
        patch.skill = { enabled: true, ...data.skill };
        if (data.levelUp) {
          milestone = true;
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
          milestone = true;
          this._celebrate({ up: '完成阶段', name: `${cleared[0]}篇`, sub: `你已完成「${cleared[0]}」，继续下一阶段` });
        } else if (mastered.length) {
          milestone = true;
          const bossNode = (this._roadNodes || []).find((x) => x.name === mastered[0]);
          if (bossNode && bossNode.boss && bossNode.takeaway) {
            this._showChapterCard(bossNode);
          } else {
            this._celebrate({ up: '已掌握', name: mastered[0], sub: mastered.length > 1 ? `${mastered.length} 个概念你已经能说清楚` : '你已经能自己说清楚了' });
          }
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
              patch.lessonProgress = `第 ${ni + 1} / ${rn.length} 节`;
              this._walkDest = ni;
            } else {
              patch.currentIndex = -1; // 全通关：角色停末关
              patch.lessonTitle = '课程已完成';
              patch.lessonProgress = `${rn.length} / ${rn.length} 节`;
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
          milestone = true;
          this._celebrate({ up: '连续学习', name: `${days} 天`, sub: '能把学习变成习惯的人是少数，你在其中' });
        }
        if (LEARN_REMIND_TMPL_ID && !this.data.remindState && !this._remindAskedToday()) {
          patch.remindState = 'offer';
        }
      }
      // 普通答对只在原位给一次认可式微庆祝；里程碑已有完整庆祝，不双重叠加。
      if (!milestone && data.verdict === 'right') patch.answerFeedback = 'right';
      else if (data.verdict === 'wrong') patch.answerFeedback = 'wrong';
      this.setData(patch);
      if (patch.answerFeedback) {
        if (this._feedbackTimer) clearTimeout(this._feedbackTimer);
        if (patch.answerFeedback === 'right') {
          wx.vibrateShort && wx.vibrateShort({ type: 'light' });
        }
        this._feedbackTimer = setTimeout(() => {
          const feedbackPatch = { answerFeedback: '' };
          if (!this.data.mentorSpeaking) feedbackPatch.mentorPhase = 'idle';
          this.setData(feedbackPatch, () => this._startMentorIdle(true));
        }, 1200);
      }
      this._scrollEnd();
      // 新卡片默认播讲解；英语课优先播示范句/听力句，避免中英文两段争抢。
      this._autoSpeakLesson(data.answer);
      setTimeout(() => this._startMentorIdle(!patch.answerFeedback), 0);
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
      this._clearMentorThinkTimer();
      this._answerTurn = false;
      this._duelWin = null;
      this.setData({ pending: false, mentorPhase: 'idle', mentorPose: 'idle', heroImg: this._pose(this.data.heroAct, this.data.walking, false) }, () => this._startMentorIdle(true));
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
    this._stopMentorIdle(true);
    this._reviewing = false;
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    const lesson = this._lessonFocus(messages);
    this.setData({ messages, lessonCard: lesson.card, lessonChoice: '', currentSessionId: '', historyVisible: false, options: [], mapVisible: false }, () => this._startMentorIdle(true));
    this._scrollEnd();
  },

  async openHistory() {
    this._stopMentorIdle(true);
    this.setData({ historyVisible: true });
    try {
      const sessions = await agentSessions(this.data.id);
      this.setData({ sessions: this._decorate(sessions || []) });
    } catch (e) {}
  },

  closeHistory() {
    this.setData({ historyVisible: false }, () => this._startMentorIdle(true));
  },

  noop() {},

  async switchSession(e) {
    const sid = e.currentTarget.dataset.id;
    if (!sid) return;
    if (sid === this.data.currentSessionId) {
      this.setData({ historyVisible: false }, () => this._startMentorIdle(true));
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
      }, () => this._startMentorIdle(true));
      this._scrollEnd();
    } catch (err) {
      this._startMentorIdle(true);
      wx.showToast({ title: '加载失败', icon: 'none' });
    }
  },

  onUnload() {
    this._pageDestroyed = true;
    this._pageVisible = false;
    this._clearMentorIdleTimers();
    this._clearMentorThinkTimer();
    this._stopMentorMouth(false);
    this._ttsRequestId = (this._ttsRequestId || 0) + 1;
    if (this._celebTimer) clearTimeout(this._celebTimer);
    if (this._feedbackTimer) clearTimeout(this._feedbackTimer);
    if (this._walkTimer) clearTimeout(this._walkTimer);
    if (this._duelTimer) clearTimeout(this._duelTimer);
    if (this._walkDelay) clearTimeout(this._walkDelay);
    if (this._audio) { this._audio.destroy(); this._audio = null; }
  },

  // 页面被其他页覆盖时不在后台继续讲，回来后由用户点“继续”。
  onShow() {
    this._pageVisible = true;
    if (this.data.pending) {
      this.setData({ mentorPhase: 'thinking', mentorPose: 'glance' });
      return;
    }
    setTimeout(() => this._startMentorIdle(true), 0);
  },

  onHide() {
    this._pageVisible = false;
    this._clearMentorIdleTimers();
    this._clearMentorThinkTimer();
    this._stopMentorMouth(false);
    this._ttsRequestId = (this._ttsRequestId || 0) + 1;
    if (this._audio && this.data.mentorSpeaking) this._audio.pause();
  },

  // ============ 英语课 · 英文朗读（TTS） ============
  // 只在开口课（spoken-english）启用：英文选项句可点🔊听，跟读示范句自动播 + 可重播。
  // 语音走微信同声传译插件 WechatSI 的 textToSpeech（en_US），零成本、与同插件 ASR 配合使用。

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

  // 听力门：找到「🎧 只听不看：」标记行，其下一行英文即隐藏音频句。非英语课/无标记返回空串。
  _extractListen(content) {
    if (!this._englishAudio || !content) return '';
    const lines = content.split('\n');
    for (let i = lines.length - 2; i >= 0; i--) {
      if (lines[i].trim() === LISTEN_MARK) return (lines[i + 1] || '').trim();
    }
    return '';
  },

  // 听力门遮罩：把每个标记行的下一行英文换成占位——只能听，不能看。
  _maskListen(content) {
    const lines = content.split('\n');
    for (let i = 0; i < lines.length - 1; i++) {
      if (lines[i].trim() === LISTEN_MARK) lines[i + 1] = LISTEN_MASK;
    }
    return lines.join('\n');
  },

  // 给助手消息挂上 say（示范句），供气泡下方渲染🔊「听发音」。非英语课原样返回。
  // 消息带听力门时：正文遮住英文原句，原句挂到 say（🔊 即「重听」）。
  _decorateMsgs(list) {
    if (!this._englishAudio) return list || [];
    return (list || []).map((m) => {
      if (m.role !== 'assistant') return m;
      const listen = this._extractListen(m.content);
      if (listen) return { ...m, content: this._maskListen(m.content), say: listen };
      return { ...m, say: this._extractSay(m.content) };
    });
  },

  _siPlugin() {
    if (this._si !== undefined) return this._si;
    try { this._si = requirePlugin('WechatSI'); } catch (e) { this._si = null; } // 插件未添加/不支持
    return this._si;
  },

  _audioCtx() {
    if (!this._audio) {
      const a = wx.createInnerAudioContext();
      // 默认自动播放仍必须尊重 iOS 系统静音开关。
      a.obeyMuteSwitch = true;
      a.onPlay(() => {
        if (this._pageDestroyed) return;
        this._stopMentorIdle(true);
        this.setData({ mentorSpeaking: true, mentorPaused: false, mentorPhase: 'speaking', mentorPose: 'idle' }, () => {
          this._startMentorMouth(this._spokenText);
        });
      });
      a.onPause(() => {
        if (!this._pageDestroyed) this._settleMentorVoice(true);
      });
      a.onStop(() => {
        if (!this._pageDestroyed) this._settleMentorVoice(false);
      });
      a.onEnded(() => {
        if (!this._pageDestroyed) this._settleMentorVoice(false);
      });
      a.onError(() => {
        if (!this._pageDestroyed) this._settleMentorVoice(false);
      });
      this._audio = a;
    }
    return this._audio;
  },

  // TTS 合成临时音频 → innerAudioContext 播放。英语示范与中文数字人共用缓存和播放状态。
  _speak(text, lang, options) {
    const clean = (text || '').trim();
    if (!clean) return;
    const voiceLang = lang || 'en_US';
    const cacheKey = `${voiceLang}:${clean}`;
    const manual = !!(options && options.manual);
    if (this.data.voiceMuted && !manual) return;
    if (manual && this.data.voiceMuted) {
      this.setData({ voiceMuted: false });
      this._saveVoicePrefs({ muted: false });
    }
    const requestId = (this._ttsRequestId || 0) + 1;
    this._ttsRequestId = requestId;
    this._audioKey = cacheKey;
    this._spokenText = clean;
    this._ttsCache = this._ttsCache || {};
    const play = (src) => {
      if (requestId !== this._ttsRequestId || this.data.voiceMuted || this._pageVisible === false) return;
      const a = this._audioCtx();
      a.stop();
      a.src = src;
      a.play();
    };
    if (this._ttsCache[cacheKey]) { play(this._ttsCache[cacheKey]); return; }
    const plugin = this._siPlugin();
    if (!plugin || !plugin.textToSpeech) { wx.showToast({ title: '朗读暂不可用', icon: 'none' }); return; }
    plugin.textToSpeech({
      lang: voiceLang, tts: true, content: clean,
      success: (res) => { const f = res && res.filename; if (f) { this._ttsCache[cacheKey] = f; play(f); } },
      fail: () => {
        if (requestId === this._ttsRequestId) wx.showToast({ title: '朗读失败，再试一次', icon: 'none' });
      },
    });
  },

  // 数字人只讲当前卡片的前两个短句，避免把长文整段念成「AI 主播」。
  _mentorExcerpt(text) {
    const clean = String(text || '').replace(/[🎧🔊]/g, '').replace(/\s+/g, ' ').trim();
    const short = clean.split(/[。！？]/).filter(Boolean).slice(0, 2).join('。');
    return short.length > 72 ? `${short.slice(0, 72)}。` : short;
  },

  // InnerAudioContext 不提供实时音素/振幅数据，因此用朗读文本生成带停顿的嘴型节奏；
  // 标点闭口，元音与常见汉字按字符稳定映射到三种开口形态，避免机械的两帧循环。
  _mouthFrameFor(char) {
    if (!char || /[\s，。！？、,.!?;；:：]/.test(char)) return 'closed';
    if (/[oOuUwW]/.test(char)) return 'round';
    if (/[aAeEiI]/.test(char)) return 'wide';
    const frames = ['small', 'wide', 'round', 'small'];
    return frames[char.charCodeAt(0) % frames.length];
  },

  _startMentorMouth(text) {
    this._stopMentorMouth(false);
    const speech = String(text || '').trim() || '正在讲解';
    this._mentorMouthCursor = 0;
    const tick = () => {
      if (this._pageDestroyed || !this.data.mentorSpeaking) return;
      const char = speech[this._mentorMouthCursor % speech.length];
      this._mentorMouthCursor += 1;
      const frame = this._mouthFrameFor(char);
      this.setData({ mentorMouth: frame });
      const pause = frame === 'closed';
      this._mentorMouthTimer = setTimeout(tick, pause ? this._idleDelay(170, 260) : this._idleDelay(92, 145));
    };
    tick();
  },

  _stopMentorMouth(updateView) {
    if (this._mentorMouthTimer) clearTimeout(this._mentorMouthTimer);
    this._mentorMouthTimer = null;
    if (updateView !== false && !this._pageDestroyed && this.data.mentorMouth !== 'closed') {
      this.setData({ mentorMouth: 'closed' });
    }
  },

  _settleMentorVoice(paused) {
    this._stopMentorMouth(false);
    const feedback = this.data.answerFeedback;
    const phase = feedback === 'right' ? 'affirm' : (feedback === 'wrong' ? 'reassure' : 'idle');
    this.setData({
      mentorSpeaking: false,
      mentorPaused: !!paused,
      mentorMouth: 'closed',
      mentorPhase: phase,
      mentorPose: 'idle',
    }, () => this._startMentorIdle(true));
  },

  // ============ 数字人待机生命感 ============
  _idleDelay(min, max) {
    return Math.round(min + Math.random() * (max - min));
  },

  _canMentorIdle() {
    const d = this.data;
    return !this._pageDestroyed && this._pageVisible !== false && d.gameSkin && !d.loading && !d.errored &&
      !d.pending && !d.mentorSpeaking && !d.mapVisible && !d.historyVisible &&
      !d.celebrate && !d.answerFeedback;
  },

  _clearMentorIdleTimers() {
    [
      '_mentorBlinkTimer', '_mentorBlinkReturnTimer', '_mentorGlanceTimer',
      '_mentorGlanceReturnTimer', '_mentorShiftTimer', '_mentorShiftReturnTimer',
      '_mentorHintTimer', '_mentorHintReturnTimer',
    ].forEach((key) => {
      if (this[key]) clearTimeout(this[key]);
      this[key] = null;
    });
  },

  _clearMentorThinkTimer() {
    if (this._mentorThinkTimer) clearTimeout(this._mentorThinkTimer);
    this._mentorThinkTimer = null;
  },

  _stopMentorIdle(clearHint) {
    this._clearMentorIdleTimers();
    const patch = { mentorPose: 'idle', mentorMotion: '' };
    if (clearHint) patch.mentorIdleHint = false;
    this.setData(patch);
  },

  // resetClock=true 表示用户刚有操作，45 秒安静提示重新计时。
  _startMentorIdle(resetClock) {
    this._clearMentorIdleTimers();
    if (!this._canMentorIdle()) return;
    if (resetClock) {
      this._mentorHintShown = false;
      this.setData({ mentorPose: 'idle', mentorMotion: '', mentorPhase: 'idle', mentorIdleHint: false });
    }
    this._scheduleMentorBlink();
    this._scheduleMentorGlance();
    this._scheduleMentorShift();
    if (!this._mentorHintShown) {
      this._mentorHintTimer = setTimeout(() => {
        if (!this._canMentorIdle() || !this.data.options.length) return;
        this._mentorHintShown = true;
        this.setData({ mentorIdleHint: true });
        this._mentorHintReturnTimer = setTimeout(() => {
          this.setData({ mentorIdleHint: false });
        }, 6000);
      }, 45000);
    }
  },

  _scheduleMentorBlink(delay) {
    this._mentorBlinkTimer = setTimeout(() => {
      if (!this._canMentorIdle()) return;
      if (this.data.mentorPose !== 'idle') {
        this._scheduleMentorBlink(900);
        return;
      }
      this.setData({ mentorPose: 'blink' });
      this._mentorBlinkReturnTimer = setTimeout(() => {
        if (!this._canMentorIdle()) return;
        this.setData({ mentorPose: 'idle' });
        // 少量双眨眼打破机械节拍，但不让它成为可预测的循环。
        if (Math.random() < 0.14) {
          this._mentorBlinkTimer = setTimeout(() => {
            if (!this._canMentorIdle() || this.data.mentorPose !== 'idle') return;
            this.setData({ mentorPose: 'blink' });
            this._mentorBlinkReturnTimer = setTimeout(() => {
              if (this._canMentorIdle()) this.setData({ mentorPose: 'idle' });
              this._scheduleMentorBlink();
            }, 130);
          }, 170);
        } else {
          this._scheduleMentorBlink();
        }
      }, 140);
    }, typeof delay === 'number' ? delay : this._idleDelay(3000, 7000));
  },

  _scheduleMentorGlance() {
    this._mentorGlanceTimer = setTimeout(() => {
      if (!this._canMentorIdle()) return;
      if (this.data.mentorPose !== 'idle') {
        this._scheduleMentorGlance();
        return;
      }
      this.setData({ mentorPose: 'glance' });
      this._mentorGlanceReturnTimer = setTimeout(() => {
        if (this._canMentorIdle()) this.setData({ mentorPose: 'idle' });
        this._scheduleMentorGlance();
      }, this._idleDelay(1400, 2300));
    }, this._idleDelay(8000, 20000));
  },

  _scheduleMentorShift() {
    this._mentorShiftTimer = setTimeout(() => {
      if (!this._canMentorIdle()) return;
      this.setData({ mentorMotion: 'shift' });
      this._mentorShiftReturnTimer = setTimeout(() => {
        this.setData({ mentorMotion: '' });
        this._scheduleMentorShift();
      }, 2400);
    }, this._idleDelay(15000, 30000));
  },

  _readVoicePrefs() {
    try {
      const saved = wx.getStorageSync(VOICE_PREFS_KEY) || {};
      return { autoPlay: saved.autoPlay !== false, muted: !!saved.muted };
    } catch (e) {
      return { autoPlay: true, muted: false };
    }
  },

  _saveVoicePrefs(patch) {
    const next = {
      autoPlay: patch && typeof patch.autoPlay === 'boolean' ? patch.autoPlay : this.data.voiceAutoPlay,
      muted: patch && typeof patch.muted === 'boolean' ? patch.muted : this.data.voiceMuted,
    };
    try { wx.setStorageSync(VOICE_PREFS_KEY, next); } catch (e) {}
  },

  // 自动播放只走课程讲解；手动“听发音”始终保留为用户可用的明确操作。
  _autoSpeakLesson(card) {
    if (!this.data.gameSkin || !this.data.voiceAutoPlay || this.data.voiceMuted || this._pageVisible === false || !card) return;
    const content = typeof card === 'string' ? card : (card.content || '');
    const sayLine = (typeof card === 'object' && card.say) || this._extractSay(content) || this._extractListen(content);
    if (sayLine) this._speak(sayLine);
    else this._speak(this._mentorExcerpt(content), 'zh_CN');
  },

  onToggleVoiceAutoPlay(e) {
    const autoPlay = !!e.detail.value;
    this.setData({ voiceAutoPlay: autoPlay });
    this._saveVoicePrefs({ autoPlay });
  },

  onToggleMute() {
    this._stopMentorIdle(true);
    const muted = !this.data.voiceMuted;
    this.setData({ voiceMuted: muted, mentorPaused: false });
    this._saveVoicePrefs({ muted });
    if (muted) {
      this._ttsRequestId = (this._ttsRequestId || 0) + 1;
      if (this._audio) this._audio.stop();
    }
    setTimeout(() => this._startMentorIdle(true), 0);
  },

  // 选项上的🔊：只朗读，不作答（catchtap 拦截，不触发 pickOption）
  onSpeakChip(e) { this._speak(e.currentTarget.dataset.t, undefined, { manual: true }); },
  // 气泡下方的🔊：重播这一关的示范句
  onSpeakSay(e) { this._speak(e.currentTarget.dataset.t, undefined, { manual: true }); },
  // 舞台数字人：播放中点击暂停；同一段已暂停则继续；静音时主动点播会同时取消静音。
  onSpeakMentor(e) {
    this._stopMentorIdle(true);
    const content = e.currentTarget.dataset.t || '';
    const sayLine = e.currentTarget.dataset.say || this._extractSay(content) || this._extractListen(content);
    const text = sayLine || this._mentorExcerpt(content);
    const lang = sayLine ? 'en_US' : 'zh_CN';
    const key = `${lang}:${text}`;
    if (this.data.mentorSpeaking && this._audio) {
      this._audio.pause();
      return;
    }
    if (this.data.mentorPaused && this._audio && (this._audioKey === key || this.data.pending)) {
      this._audio.play();
      return;
    }
    this._speak(text, lang, { manual: true });
  },

  goMembership() {
    wx.navigateTo({ url: '/pages/membership/index' });
  },

  _scrollEnd() {
    const n = this.data.messages.length;
    if (n > 0) this.setData({ scrollTo: 'm' + (n - 1) });
  },

  // 章末知识卡片「分享这张卡」：文字转发，不生成海报图（海报长图是另一个待做项目）。
  onShareAppMessage() {
    const cc = this.data.chapterCard;
    if (cc) {
      return {
        title: `${cc.takeaway}（${cc.source}）`,
        path: `/pages/agent-chat/index?id=${this.data.id}`,
      };
    }
    return { title: this.data.name || '来看看这个 AI 分身', path: `/pages/agent-chat/index?id=${this.data.id}` };
  },
});
