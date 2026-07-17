const { ensureLogin } = require('../../utils/auth');
const {
  agentDetail, agentEnter, agentSessions, sessionMessages, agentConcepts, chatAgent,
  remindLearn,
} = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');
const { fmtDateTime } = require('../../utils/datetime');
const { requestLearnRemind, loadTmpls, tmplReady } = require('../../utils/subscribe');
const { buildNodes, iconForLevel, STATE_TEXT, STATE_CTA } = require('../../utils/learn-nodes');

// streak 里程碑：只在这些天数弹庆祝，其余日子安静（轻而真，不做焦虑轰炸）
const STREAK_MILESTONES = [3, 7, 14, 30, 60, 100];

// 听力门标记（与服务端 script_learn_english.go 的 listenMark 同一约定）：
// 英语课消息里该标记行的下一行英文「只播不显」——正文遮成占位，自动朗读，🔊 可重听。
const LISTEN_MARK = '🎧 只听不看：';
const LISTEN_MASK = '▂ ▂ ▂ ▂ ▂ ▂ ▂ ▂';

// 课程语音偏好跨课程保留；首次默认自动播放，但只在用户主动进入学习页后触发。
const VOICE_PREFS_KEY = 'weifou_lesson_voice_prefs';

// 产出节点（Free）兜底选项：与服务端 script.go 的 optFreeSkip 逐字一致。
// 服务端只在产出节点下发这一个选项——它就是「本轮要学员自己开口/动笔」的识别信号，
// 前端据此把底部点选换成产出条（打字 + 按住说话），真开口/真动笔从此不再只能跳过。
const FREE_SKIP = '这句先欠着，直接过关';

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
    freeMode: false,      // 产出节点：底栏换成「打字/按住说话」产出条（服务端只发兜底跳过选项时置真）
    freeText: '',         // 产出条草稿（语音识别结果也填这里，学员确认后再发）
    recState: '',         // '' / recording：按住说话状态（驱动麦克风按钮样式与提示）
    englishAudio: false,  // 仅英语课（spoken-english）：英文选项 + 示范句配🔊朗读、示范句自动播
    pending: false,
    loading: true,
    errored: false,       // 首屏并发加载失败 → 页内重试骨架（避免白屏）
    scrollTo: '',
    sessions: [],         // 历史会话（抽屉）
    currentSessionId: '', // 当前续聊的会话；空 = 新开一段（下一条消息创建）
    historyVisible: false,
    concept: null,        // 概念型 Agent 的点亮进度（null/enabled=false 时不展示）
    gameSkin: false,      // 学习课（有点亮进度）套「游戏事件卡」皮：固定舞台+事件卡流+底部点选；分身聊天保持原样
    mapVisible: false,    // 学习中默认收起课程地图；点击进度后以底部抽屉展开
    lessonCard: null,     // 单卡学习：当前助手任务/反馈，不再把整段聊天铺在屏幕上
    lessonChoice: '',     // 本轮用户选择，作为卡片内的轻量上下文
    lessonTitle: '当前练习',
    lessonProgress: '',
    reviewDue: 0,         // 到期待复习的概念数（>0 时进度条上出现「复习挑战」徽章）
    remindState: '',      // 提醒承诺条：'' 隐藏 / offer 邀请 / done 已订
    celebrate: null,      // 庆祝浮层：{ up, name, sub }，触发后短暂展示（升级 / 点亮 / 掌握共用）
    // —— 课程地图（列表抽屉）——
    nodes: [],            // 扁平关卡节点 [{ idx,slug,name,state,icon,boss,memberLocked,... }]
    currentIndex: -1,     // 当前关扁平下标（-1=全通关）
    card: null,           // 关卡卡片抽屉
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
    // 从闯关地图点关进来：记下目标关卡，数据就绪后自动开课
    this._targetConcept = query.concept || '';
    this._autoStart = query.auto === '1';
    // 从技能页「到期复习」条进来：数据就绪后直接开复习挑战
    this._autoReview = query.review === '1';
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

  // 本页数据加载（并发拉 详情/会话/会员/概念）。抽成独立方法，供「网络失败重试」复用。
  // 失败不再只弹 toast——置 errored 让页内出现「重试」骨架，避免慢网/断网时白屏发愣。
  async _load() {
    const id = this.data.id;
    this.setData({ loading: true, errored: false });
    loadTmpls().catch(() => {}); // 预热订阅模板缓存：streak newDay 时同步判断 tmplReady('learnRemind')
    try {
      // 聚合首屏（/agents/enter）：详情+会话+最近消息+概念进度+会员态一次往返，
      // 消掉旧路「4 并发 + sessions→messages 串行」在移动网络上多付的 RTT。
      // 老服务端没有该接口（404）时回落旧的并发路径，前后端发版时序互不卡脖子。
      let d; let sessions; let member; let cp; let msgs;
      try {
        const en = await agentEnter(id);
        d = en.detail; sessions = en.sessions || []; member = !!en.member;
        cp = en.concept || { enabled: false }; msgs = en.messages || [];
      } catch (err) {
        const [d2, s2, ms2, cp2] = await Promise.all([
          agentDetail(id),
          agentSessions(id).catch(() => []),
          membershipStatus().catch(() => ({ isMember: false })),
          agentConcepts(id).catch(() => ({ enabled: false })),
        ]);
        d = d2; sessions = s2 || []; member = !!ms2.isMember; cp = cp2; msgs = null;
      }
      // 英语课（开口 spoken-english）专属：英文句配朗读。早于建 messages 置位，供 _decorateMsgs 用。
      this._englishAudio = !!(d && d.slug === 'spoken-english');
      // 默认载入最近一段会话；没有则以开场白起新的一段
      let messages = [];
      let currentSessionId = '';
      let restoredOptions = [];
      if (sessions.length) {
        currentSessionId = sessions[0].sessionId;
        if (msgs === null) msgs = await sessionMessages(currentSessionId).catch(() => []);
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
      // 完整消息数组存实例变量；游戏皮界面只渲染 lessonCard，整段历史随轮次线性膨胀，
      // 每轮全量 setData 是长会话卡顿的头号来源——课程模式下不喂渲染层（见 _msgData）。
      const gs = !!(d.concept || d.assess || (cp && cp.enabled));
      this._messages = messages;
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
        messages: gs ? [] : messages,
        lessonCard: lesson.card,
        lessonChoice: lesson.choice,
        lessonTitle: d.guide || d.name || '当前练习',
        lessonProgress: '',
        options: restoredOptions,
        freeMode: this._isFreeOpts(restoredOptions),
        currentSessionId,
        sessions: this._decorate(sessions || []),
        concept: cp && cp.enabled ? this._concept(cp) : null,
        // 游戏皮（含隐藏顶部商业条）以主接口的 concept/assess 布尔为准——它必然成功（名字/图标都靠它）；
        // 次要的 concept 进度接口带 catch 回落，单独当判据会在慢网时漏出「开通会员」商业条。
        gameSkin: gs,
        reviewDue: cp && cp.enabled ? (cp.due || 0) : 0,
        englishAudio: this._englishAudio,
        loading: false,
      });
      this._scrollEnd();
      // 概念课：铺课程地图节点，定位当前关
      if (cp && cp.enabled) this._applyRoad(cp);
      let openingRound = false;
      if (this._autoReview && cp && cp.enabled) {
        // 到期复习直达：进页即开复习挑战（先清徽章防连点，与 startReview 一致）
        this._autoReview = false;
        openingRound = true;
        this.setData({ reviewDue: 0, mapVisible: false, lessonTitle: '复习挑战', lessonProgress: '快速复习' });
        this._ask('开始复习挑战', 'review');
      } else if (this._autoStart && this._targetConcept) {
        // 兼容旧入口（带 concept&auto=1）：新开一段直接开该关
        this._autoStart = false;
        this.setData({
          ...this._msgData(this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : []),
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

  // 铺课程地图：摊平节点、定位当前关（地图=列表抽屉；横版路与负鼠舞台已退役）。
  _applyRoad(cp) {
    const built = buildNodes(cp);
    const nodes = built.nodes;
    const idx = built.currentIndex >= 0 ? built.currentIndex : Math.max(0, nodes.length - 1);
    this._roadNodes = nodes; // 实例副本，供查找/增量，不进 setData
    this.setData({
      nodes,
      currentIndex: built.currentIndex,
      lessonTitle: nodes[idx] ? nodes[idx].name : '课程已完成',
      lessonProgress: nodes.length ? `第 ${Math.min(idx + 1, nodes.length)} / ${nodes.length} 节` : '',
    });
  },

  // 维护完整消息数组：真源在 this._messages；游戏皮（课程）模式不喂渲染层——
  // 界面只渲染 lessonCard 一张卡，整段历史每轮全量 setData 纯属陪跑且随会话线性膨胀。
  // 普通聊天流模式（分身对话）仍照常同步到 data.messages。
  _msgData(messages) {
    this._messages = messages;
    return this.data.gameSkin ? {} : { messages };
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
    this.setData({ card: null });
    if (c.memberLocked) { this.goMembership(); return; }
    // 新开一段会话 + 带 concept 让教练用该关钩子开场（与秒开一致）
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    const lesson = this._lessonFocus(messages);
    this.setData({
      ...this._msgData(messages),
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
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    this.setData({
      ...this._msgData(messages),
      lessonCard: this._lessonFocus(messages).card,
      lessonChoice: '',
      lessonTitle: cur ? cur.name : this.data.lessonTitle,
      currentSessionId: '',
      options: [],
      freeMode: false,
      freeText: '',
      mapVisible: false,
    });
    this._ask('开始这一关', undefined, cur ? cur.slug : undefined);
  },

  _decorate(sessions) {
    return (sessions || []).map((s) => ({ ...s, timeText: fmtDateTime(s.updatedAt) }));
  },

  // 复习挑战：快问快答已点亮概念（检索练习，答对保住/升级档位）。
  startReview() {
    if (this.data.pending) return;
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
    if (res && res.accepted) {
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

  // ============ 产出节点：打字 / 按住说话（真开口·真动笔） ============
  // 服务端 Free 节点只发「先欠着」兜底选项；这里给出真正的产出入口：
  // 打字直接发，语音走 WechatSI 识别→填入输入框→学员确认后发（识别难免出错，不自动发送）。

  onFreeInput(e) {
    this.setData({ freeText: e.detail.value || '' });
  },

  sendFree() {
    const t = (this.data.freeText || '').trim();
    if (!t || this.data.pending) return;
    this.setData({ freeText: '' });
    this._ask(t);
  },

  skipFree() {
    if (this.data.pending) return;
    this.setData({ freeText: '' });
    this._ask(FREE_SKIP);
  },

  // WechatSI 语音识别管理器（懒加载单例）。识别结果只填输入框，不代发。
  _recorder() {
    if (this._rec !== undefined) return this._rec;
    const plugin = this._siPlugin();
    if (!plugin || !plugin.getRecordRecognitionManager) { this._rec = null; return null; }
    const rec = plugin.getRecordRecognitionManager();
    rec.onStop = (res) => {
      const text = ((res && res.result) || '').trim();
      const patch = { recState: '' };
      if (text) {
        // 追加而不是覆盖：支持分几口气说完一句话
        patch.freeText = ((this.data.freeText || '') + text).slice(0, 300);
      } else if (this.data.recState === 'recording') {
        wx.showToast({ title: '没听清，再说一遍？', icon: 'none' });
      }
      this.setData(patch);
    };
    rec.onError = () => {
      if (this.data.recState) this.setData({ recState: '' });
      wx.showToast({ title: '语音识别不可用，打字也一样算数', icon: 'none' });
    };
    this._rec = rec;
    return rec;
  },

  onMicStart() {
    if (this.data.pending || this.data.recState === 'recording') return;
    const rec = this._recorder();
    if (!rec) { wx.showToast({ title: '语音暂不可用，打字也一样算数', icon: 'none' }); return; }
    // 录音时停掉导师语音，避免 TTS 混进麦克风
    if (this._audio && this.data.mentorSpeaking) this._audio.stop();
    this.setData({ recState: 'recording' });
    try {
      rec.start({ lang: 'zh_CN', duration: 60000 });
      wx.vibrateShort && wx.vibrateShort({ type: 'light' });
    } catch (e) {
      this.setData({ recState: '' });
    }
  },

  onMicEnd() {
    if (this.data.recState !== 'recording') return;
    const rec = this._recorder();
    try { rec && rec.stop(); } catch (e) { this.setData({ recState: '' }); }
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
    this._ask(t);
  },

  async _ask(content, mode, concept) {
    if (!content || this.data.pending) return;
    this._stopMentorIdle(true);
    this._clearMentorThinkTimer();
    const visibleChoice = (content === '开始这一关' || content === '开始复习挑战') ? '' : content;
    const askedMessages = (this._messages || this.data.messages || []).concat({ role: 'user', content });
    this.setData({
      ...this._msgData(askedMessages),
      pending: true,
      answerFeedback: '',
      mentorPhase: 'listening',
      mentorPose: 'idle',
      options: [],
      freeMode: false,
      freeText: '',
      recState: '',
      lessonChoice: visibleChoice,
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
      const answeredMessages = (this._messages || []).concat(answerMsg);
      const patch = {
        ...this._msgData(answeredMessages),
        lessonCard: answerMsg,
        lessonChoice: visibleChoice,
        member,
        remaining,
        quotaText: this._quota(member, remaining, this.data.freeTier),
        // 新开一段时服务端回传新建会话 id，记下来后续消息续到同一段
        currentSessionId: data.sessionId || this.data.currentSessionId,
        options: this._toOpts(data.options),
        pending: false,
        freeMode: this._isFreeOpts(this._toOpts(data.options)),
        mentorPhase: data.verdict === 'right' ? 'affirm' : (data.verdict === 'wrong' ? 'reassure' : 'idle'),
        mentorPose: 'idle',
      };
      let milestone = false;
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
            } else {
              patch.currentIndex = -1; // 全通关
              patch.lessonTitle = '课程已完成';
              patch.lessonProgress = `${rn.length} / ${rn.length} 节`;
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
        if (tmplReady('learnRemind') && !this.data.remindState && !this._remindAskedToday()) {
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
      // 一致性护栏：极少数情况（复习跨幕等）增量与服务端 current 不符，才全量校准（正常路径不触发）。
      // 必须确认 concept 带完整 tiers，否则精简版会摊平出空节点表 → 误清空整张地图。
      if (data.concept && data.concept.tiers && data.concept.tiers.length && this._roadNodes) {
        const auth = buildNodes({ enabled: true, ...data.concept });
        if (auth.currentIndex !== this.data.currentIndex) {
          this._applyRoad({ enabled: true, ...data.concept });
        }
      }
    } catch (e) {
      this._clearMentorThinkTimer();
      this.setData({ pending: false, mentorPhase: 'idle', mentorPose: 'idle' }, () => this._startMentorIdle(true));
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
          ...this._msgData((this._messages || []).concat(errorMsg)),
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
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    const lesson = this._lessonFocus(messages);
    this.setData({ ...this._msgData(messages), lessonCard: lesson.card, lessonChoice: '', currentSessionId: '', historyVisible: false, options: [], freeMode: false, freeText: '', mapVisible: false }, () => this._startMentorIdle(true));
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
    this.setData({ historyVisible: false });
    try {
      const msgs = await sessionMessages(sid);
      const messages = this._decorateMsgs((msgs || []).map((m) => ({ role: m.role, content: m.content })));
      const lesson = this._lessonFocus(messages);
      this.setData({
        ...this._msgData(messages),
        lessonCard: lesson.card,
        lessonChoice: lesson.choice,
        currentSessionId: sid,
        options: [],
        freeMode: false,
        freeText: '',
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
    if (this.data.recState === 'recording') { try { this._rec && this._rec.stop(); } catch (e) {} }
    this._clearMentorIdleTimers();
    this._clearMentorThinkTimer();
    this._stopMentorMouth(false);
    this._ttsRequestId = (this._ttsRequestId || 0) + 1;
    if (this._celebTimer) clearTimeout(this._celebTimer);
    if (this._feedbackTimer) clearTimeout(this._feedbackTimer);
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
    if (this.data.recState === 'recording') { try { this._rec && this._rec.stop(); } catch (e) {} }
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

  // 本轮是否产出节点：服务端在 Free 节点只发唯一兜底选项（跳过），据此切换产出条。
  _isFreeOpts(opts) {
    return !!(opts && opts.length === 1 && opts[0].t === FREE_SKIP);
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
    const n = (this._messages || this.data.messages || []).length;
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
