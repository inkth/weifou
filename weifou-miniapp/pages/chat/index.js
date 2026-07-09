const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { track } = require('../../utils/track');
const { tierForPreset, getPreset, initial } = require('../../utils/avatars');
const { buildTrustLine } = require('../../utils/trust');

// 成交阶梯顺序：分身下发的 stage 映射成进度条下标（-1=未知/不显示）。
// 打赏已下线，阶梯收敛到 聊→问→约；旧数据/服务端偶发的 reward 归并到终点 book。
const STAGE_ORDER = ['chat', 'ask', 'book'];
function stageIndex(stage) {
  if (stage === 'reward') return STAGE_ORDER.indexOf('book');
  return STAGE_ORDER.indexOf(stage);
}

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
    options: [], // 每轮回答后分身沿成交阶梯现编的可点选项（点选即发，代替打字）
    stage: '', // 当前成交阶梯 chat|ask|book（分身判定，仅驱动顶部进度条）
    stageIdx: -1, // stage 在 STAGE_ORDER 中的下标，驱动进度条高亮（-1=不显示）
    stageLabels: ['聊', '问', '约'], // 进度条三段固定文案（打赏已下线）
    shownOptions: [], // 本会话已展示过的选项，换一批/追问时回传服务端避重
    startersRevealed: false, // 开场动画结束后才淡入引导问题
    answeredOnce: false, // 首轮回答后才出现行动 chips，开场聚焦"开始聊"
    contactAvailable: false,
    asyncAvailable: false, // 是否可向本人提问（免费，AI 即时答 + 本人可异步补一句）
    trustLine: '', // 信任徽章文案（社会证明，来自既有交易数据；冷启动数字过小时为空）
    isMine: false, // 主人预览自己的 AI（profile 页透传 mine=1）
    hasOwnProfile: true, // 默认 true：确认是"无主页访客"前不展示裂变钩子
    notFound: false,
    messages: [],
    pending: false,
    introState: '', // 开场动画期间覆盖 avatar 状态（thinking/speaking）
    // —— 沉浸式对话舞台 ——
    stageTier: 'warm', // 氛围档 cool|warm|lively（驱动 .tier-* class），骨架首帧用默认暖底
    ambStyle: '', // 注入头像同色系光晕的内联 CSS 变量（--amb-a/--amb-b）
    stageEntered: false, // 开幕（curtain-up）触发；与拉取资料的等待重叠
    delight: false, // 一次性"欣喜"beat（开场 & 成交/留资后）
    liheSrc: '', // 全屏立绘图：avatar 为 image 类型时启用"星野式"全屏立绘模式
    // —— 名片开场（无立绘时）：正面/翻面/收起三态 ——
    cardFlipped: false, // 名片翻到背面（完整介绍）
    cardCompact: false, // 开聊后名片收成顶部胶囊条
    cardTags: [],       // 正面只放前 3 个标签，全量在背面
    // —— 访客页内浮层（不跳 profile）：关于 ——
    title: '', company: '', city: '', tags: [], fullIntro: '',
    aboutVisible: false,
    connected: false, // 已与主人交换名片（点选连接后置位，chip 变「已交换」）
    // —— 交换名片弹层：连接（关系）+ 可选「捎句话」（意向点选，跳过则纯连接）——
    exchangeVisible: false, exchangeSending: false, exchangeNote: '',
    exchangePresets: ['想约个时间聊聊', '想合作，请联系我', '对你的服务很感兴趣'],
    // —— 让 TA 找我（无自己分身时的匿名举手，点选代替打字留言）——
    leaveVisible: false, leaveSending: false, leaveNote: '',
    leavePresets: ['想约个时间聊聊', '对你的服务很感兴趣', '想合作，请联系我', '单纯欣赏，给你点个赞'],
    celebrate: null, // 成交庆祝浮层：{ up, name, sub }，只在约/留话等终点动作触发，2.4s 自动收起
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
      stageEntered: true, // 立刻开幕：开幕动画与拉取资料的等待重叠
    });
    this._applyStageTheme(); // query 已带形象时先推一版氛围；拿到 /profile 后再覆盖

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
          ? `你好，我是 ${p.realName} 的 AI 分身。${persona.oneLiner}\n想了解些什么？`
          : `你好，我是 ${p.realName} 的 AI 分身，TA 的事都可以问我。`);
      this.setData({
        realName: p.realName || this.data.realName,
        avatarStyle: this.data.avatarStyle || p.avatarStyle || '',
        oneLiner: persona.oneLiner || '',
        starters: persona.starters || [],
        contactAvailable: !!p.contactVisible,
        title: p.title || '', company: p.company || '', city: p.city || '',
        tags: persona.tags || [], fullIntro: persona.fullIntro || '',
        cardTags: (persona.tags || []).slice(0, 3),
        trustLine: buildTrustLine(p.trust, 'asked'),
      });
      this._applyStageTheme(); // 用最终 avatarStyle 推导氛围档 + 光晕色
      // 打字机开场只惊艳一次：回访（含主人反复自测）直接显示全文，把效率还给第二次。
      // oneLiner 不再作为第二条气泡复读——名片正面已常驻展示。
      const introKey = `weifou_intro_${profileId}`;
      if (wx.getStorageSync(introKey)) {
        this.setData({ messages: [{ role: 'assistant', content: greeting }], introState: '', startersRevealed: true });
      } else {
        try { wx.setStorageSync(introKey, 1); } catch (e) {}
        this._playIntro(greeting, '');
      }
    } catch (e) {
      this.setData({ notFound: true, introState: '' });
      return;
    }

    // 提问对所有分身免费开放（AI 即时答 + 本人可异步补一句）
    this.setData({ asyncAvailable: true });
  },

  onUnload() {
    this._abortIntro();
    if (this._answerTimer) { clearInterval(this._answerTimer); this._answerTimer = null; this._answerDone = null; }
    if (this._delightTimer) { clearTimeout(this._delightTimer); this._delightTimer = null; }
    if (this._celebTimer) { clearTimeout(this._celebTimer); this._celebTimer = null; }
  },

  // 成交庆祝浮层：约/留话达成时弹一次（复用课程事件卡），2.4s 后自动收起。payload = { up, name, sub }
  _celebrate(payload) {
    if (this._celebTimer) clearTimeout(this._celebTimer);
    wx.vibrateShort && wx.vibrateShort({ type: 'medium' });
    this.setData({ celebrate: payload || null });
    this._celebTimer = setTimeout(() => this.setData({ celebrate: null }), 2400);
  },

  // 答复打字机：逐字写入第 index 条消息；_flushAnswer 可被新提问/卸载抢断为整条落定。
  // 与开场 _typeMessage 分开：后者受 _introAborted 闸门，ask() 一开始就 abort，会让答复秒显。
  _typeAnswer(index, fullText) {
    return new Promise((resolve) => {
      this._flushAnswer(); // 落定上一条（若有），再开新的
      this._answerDone = () => {
        if (this._answerTimer) { clearInterval(this._answerTimer); this._answerTimer = null; }
        this._answerDone = null;
        this.setData({ [`messages[${index}].content`]: fullText });
        resolve();
      };
      if (!fullText) { this._answerDone(); return; }
      let i = 0;
      this._answerTimer = setInterval(() => {
        i += 3;
        if (i >= fullText.length) this._answerDone();
        else this.setData({ [`messages[${index}].content`]: fullText.slice(0, i) });
      }, 40);
    });
  },

  _flushAnswer() {
    if (this._answerDone) this._answerDone();
  },

  // 由当前 avatarStyle 推导舞台氛围档（驱动 .tier-*）+ 注入头像同色系光晕（--amb-a/b）
  _applyStageTheme() {
    const id = this.data.avatarStyle || '';
    const tier = tierForPreset(id, this.data.profileId);
    const p = getPreset(id, this.data.profileId);
    const c0 = (p.colors && p.colors[0]) || '#18b690';
    const c1 = (p.colors && p.colors[1]) || c0;
    // 立绘只给有专属 image 形象的人；没有就走"画框舞台"（首字 toon 呼吸形象）——
    // 不再全员回退共用默认立绘：张三和李四的分身不该是同一张脸。
    const liheSrc = (p.type === 'image' && p.images && p.images.idle) ? p.images.idle : '';
    this.setData({
      stageTier: tier.id,
      ambStyle: `--amb-a:${c0}; --amb-b:${c1};`,
      liheSrc,
      stageInitial: initial(this.data.realName || ''),
    });
  },

  // —— 开场：台上沉思一拍 → greeting 打字机 → 一句话介绍 → starters 淡入 ——
  // 开幕(stageEntered)已在 onLoad 早置、与拉取资料的等待重叠；这里只管"说话"节奏。
  async _playIntro(greeting, oneLiner) {
    this._introAborted = false;
    await this._wait(500); // 台上沉思一拍（introState 此时已是 thinking）
    if (this._introAborted) return this._settleIntro();
    if (typeof wx.vibrateShort === 'function') {
      try { wx.vibrateShort({ type: 'light' }); } catch (e) {} // 首访开口轻震，失败静默
    }
    this._fireDelight(); // 开口同时一次性"欣喜"
    this.setData({ introState: 'speaking', messages: [{ role: 'assistant', content: '' }] });
    await this._typeMessage(0, greeting);
    if (oneLiner) {
      if (this._introAborted) return this._settleIntro();
      await this._wait(400);
      this.setData({
        messages: this.data.messages.concat({ role: 'assistant', content: '' }),
      });
      await this._typeMessage(this.data.messages.length - 1, `“${oneLiner}”`);
    }
    this._settleIntro();
  },

  _settleIntro() {
    this.setData({ introState: '', startersRevealed: true });
  },

  // 一次性"欣喜"beat：开场 / 预约成交 / 留资成功时让台上角色轻跳一下
  _fireDelight() {
    this.setData({ delight: true });
    if (this._delightTimer) clearTimeout(this._delightTimer);
    this._delightTimer = setTimeout(() => {
      this._delightTimer = null;
      this.setData({ delight: false });
    }, 750); // 与 delightBounce keyframe 时长一致
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

  // 等待 ms；被 _abortIntro 抢断时立即 resolve（不留悬挂的 await）
  _wait(ms) {
    return new Promise((r) => {
      this._waitResolve = r;
      this._waitTimer = setTimeout(() => {
        this._waitTimer = null;
        this._waitResolve = null;
        r();
      }, ms);
    });
  },

  // 用户提前交互（点 chip / 输入）→ 立即放完动画，绝不让用户等
  _abortIntro() {
    this._introAborted = true;
    if (this._typeTimer) { clearInterval(this._typeTimer); this._typeTimer = null; }
    if (this._waitTimer) { clearTimeout(this._waitTimer); this._waitTimer = null; }
    if (this._waitResolve) { const r = this._waitResolve; this._waitResolve = null; r(); } // 解除悬挂的 _wait
    if (!this.data.startersRevealed || !this.data.stageEntered) {
      this.setData({ startersRevealed: true, stageEntered: true });
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
      ? `和 ${name} 的 AI 分身聊聊：${this.data.oneLiner}`
      : `加微信前，先和 ${name} 的 AI 分身聊聊`;
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

  // 点成交阶梯选项 → 作为下一句发送（全程点选，无自由输入；ask 开头会清空整组避免重复）
  pickOption(e) {
    const q = e.currentTarget.dataset.q;
    if (q) this.ask(q);
  },

  // 换一批：不产生新对话轮，只让分身基于当前上下文另出一组可点选项（复用 exclude 避重）
  async reshuffle() {
    if (this.data.pending || this._reshuffling) return;
    this._reshuffling = true;
    try {
      const data = await request({
        url: `/chat/${this.data.profileId}/reoptions`,
        method: 'POST',
        data: { exclude: this.data.shownOptions.slice(-12) },
      });
      const opts = data.options || [];
      if (opts.length) {
        const stage = data.stage || this.data.stage;
        this.setData({
          options: opts,
          stage,
          stageIdx: stageIndex(stage),
          shownOptions: this.data.shownOptions.concat(opts),
        });
      } else {
        wx.showToast({ title: '暂时没有更多了', icon: 'none' });
      }
    } catch (e) {
      wx.showToast({ title: '换一批失败', icon: 'none' });
    } finally {
      this._reshuffling = false;
    }
  },

  // —— 名片三态 ——
  flipCard() {
    this.setData({ cardFlipped: !this.data.cardFlipped });
  },
  expandCard() {
    this.setData({ cardCompact: false, cardFlipped: false });
  },
  // "关于 TA" chip：立绘模式走原毛玻璃抽屉；名片模式展开并翻到背面
  openAboutEntry() {
    if (this.data.liheSrc) return this.openAbout();
    this.setData({ cardCompact: false, cardFlipped: true });
  },

  async ask(content) {
    if (!content || this.data.pending) return;
    this._abortIntro();
    this._flushAnswer(); // 上一条还在逐字时立即落定，避免两条打字机叠加
    try {
      await ensureLogin();
    } catch (e) {
      wx.showToast({ title: '请先登录', icon: 'none' });
      return;
    }

    const messages = this.data.messages.concat({ role: 'user', content });
    // 第三幕结束：一旦开聊，名片收成顶部胶囊条；上一轮选项作废
    this.setData({ messages, pending: true, cardCompact: true, cardFlipped: false, options: [] });

    try {
      const data = await request({
        url: `/chat/${this.data.profileId}/ask`,
        method: 'POST',
        data: { content, exclude: this.data.shownOptions.slice(-12) },
      });
      // 答复也走打字机：先落一条空气泡，introState=speaking 让舞台维持"在说话"，再逐字显现，
      // 每一轮都有"真人在说"的临场感（开场打字机只是第一拍，问答同样有节奏）。
      const msgs = this.data.messages.concat({ role: 'assistant', content: '' });
      const idx = msgs.length - 1;
      this.setData({ messages: msgs, pending: false, answeredOnce: true, introState: 'speaking' });
      await this._typeAnswer(idx, data.answer || '');
      // 回答落定后亮出成交阶梯选项：访客不打字也能一路被带着往成交走
      const opts = data.options || [];
      const stage = data.stage || this.data.stage;
      this.setData({
        introState: '',
        options: opts,
        stage,
        stageIdx: stageIndex(stage),
        shownOptions: this.data.shownOptions.concat(opts),
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
    wx.navigateTo({ url: `/pages/onboarding/index?ref=${this.data.profileId}` });
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
            this._celebrate({ up: '联系方式', name: '已拿到 ✓', sub: `主动联系 ${this.data.realName}，趁热打铁` });
            this._showOwnHook('strong');
          }
        },
      });
    } catch (e) {
      wx.showToast({ title: e.message || '本人未公开联系方式', icon: 'none' });
    }
  },

  // 行动选项：让 TA 找我（零门槛「举手」→ 主人侧线索）。点选意向代替打字留言。
  openLeave() {
    this.setData({ leaveVisible: true, leaveNote: '' });
  },
  closeLeave() {
    this.setData({ leaveVisible: false });
  },
  pickLeave(e) {
    this.setData({ leaveNote: e.currentTarget.dataset.msg });
  },
  async sendLeave() {
    const note = (this.data.leaveNote || '').trim();
    if (!note || this.data.leaveSending) return;
    try {
      await ensureLogin();
    } catch (e) {
      wx.showToast({ title: '请先登录', icon: 'none' });
      return;
    }
    this.setData({ leaveSending: true });
    try {
      await request({
        url: `/chat/${this.data.profileId}/lead`,
        method: 'POST',
        data: { note },
      });
      this.setData({
        leaveVisible: false,
        messages: this.data.messages.concat({
          role: 'assistant',
          content: `已把「${note}」转达给 ${this.data.realName}，TA 会尽快看到并联系你 ✅`,
        }),
      });
      this._celebrate({ up: '已举手', name: '送达 ✓', sub: `${this.data.realName} 会亲自看到你` });
      this._showOwnHook('strong');
    } catch (err) {
      wx.showToast({ title: err.message || '发送失败', icon: 'none' });
    } finally {
      this.setData({ leaveSending: false });
    }
  },

  // 行动选项：交换名片（站内连接，不导流微信）。有自己分身 → 打开弹层可捎句话；没有 → 引导创建（裂变）。
  openExchange() {
    if (this.data.isMine || this.data.connected) return;
    if (!this.data.hasOwnProfile) { this.goCreateOwn(); return; } // 先建一个你的分身才能交换
    this.setData({ exchangeVisible: true, exchangeNote: '' });
  },
  closeExchange() { this.setData({ exchangeVisible: false }); },
  // 捎句话点选（选填）：再点一次已选的取消
  pickExchangeNote(e) {
    const msg = e.currentTarget.dataset.msg;
    this.setData({ exchangeNote: this.data.exchangeNote === msg ? '' : msg });
  },
  // 交换=建立关系；exchangeNote=可选携带的具体诉求（有则后端同时落线索、通知带话）
  async confirmExchange() {
    if (this.data.exchangeSending) return;
    this.setData({ exchangeSending: true });
    try {
      await ensureLogin();
      const note = this.data.exchangeNote || '';
      const r = await request({ url: `/connect/${this.data.profileId}`, method: 'POST', data: note ? { note } : {} });
      const withNote = !!note;
      this.setData({
        connected: true,
        exchangeVisible: false,
        messages: this.data.messages.concat({
          role: 'assistant',
          content: r.already
            ? `我们早就交换过名片啦${withNote ? '，你的话我也转达给 ' + this.data.realName + ' 了' : ''} ✅`
            : `已和 ${this.data.realName} 交换名片，进了你的名片夹 ✅${withNote ? ` 你捎的话 TA 会看到` : ' 随时能回来问我'}`,
        }),
      });
      this._celebrate({ up: '交换名片', name: '成功 ✓', sub: withNote ? `${this.data.realName} 会看到你的话` : `${this.data.realName} 进了你的名片夹` });
      this._showOwnHook('strong');
    } catch (e) {
      if (e.code === 'NO_PROFILE') { this.setData({ exchangeVisible: false }); this.goCreateOwn(); return; }
      wx.showToast({ title: e.message || '交换失败', icon: 'none' });
    } finally {
      this.setData({ exchangeSending: false });
    }
  },

  // —— 页内浮层：关于 TA ——
  openAbout() { this.setData({ aboutVisible: true }); },
  closeAbout() { this.setData({ aboutVisible: false }); },
  noop() {},

});
