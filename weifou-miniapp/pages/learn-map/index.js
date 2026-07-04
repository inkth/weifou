// 学习区主页:双视图。默认「第N课·卡片流」(借挂机游戏事件卡骨架:成果卡+悬念卡+大按钮,
// 但序号只随真实完成的课推进——真开口才点亮,无随机无负面);右上可切回「闯关地图」(蜿蜒路径,自由跳关)。
// 数据同源 GET /agents/concepts/:id,两视图共用同一 sections(80 关课程 setData 体积可控的关键)。
// 节点状态:locked(软锁,仍可点) / available(下一关) / current(当前关,脉冲) / lit(已点亮) / mastered(金冠)。
const { ensureLogin } = require('../../utils/auth');
const { agentConcepts, agentSkill, learnStreak } = require('../../utils/agent');

// 蜿蜒路径:按全局序号取横向偏移(rpx),正弦相位 8 步一循环。
const SNAKE = [0, 88, 140, 88, 0, -88, -140, -88];
// checkpoint 横幅糖果色轮换
const TONES = ['butter', 'sky', 'mint', 'lilac'];
// 贴纸图标(scripts/gen-learn-icons.mjs 产物);icon 缺失时 wxml 回退 emoji
const ICON_BASE = '/assets/icons/learn/';
const STATE_ICONS = { mastered: 'crown', lit: 'star', current: 'star', available: 'sparkle', locked: 'lock' };
// 状态词表(抽屉与卡流共用一套,风味角标只正向/中性,不做"运气不佳"式负面)
const STATE_TEXT = { mastered: '👑 已掌握', lit: '⭐ 已点亮 · 可冲掌握', current: '当前关卡', available: '下一关', locked: '未解锁' };
const STATE_CTA = { mastered: '再练一遍', lit: '冲击掌握', current: '开始这一关', available: '开始这一关', locked: '提前解锁这一关' };
const FLAVOR = { mastered: '传说达成', lit: '可再战', current: '今日主线', available: '新篇章', locked: '前方迷雾' };
const VIEW_MODE_KEY = 'learnMapViewMode'; // 全局一份:用户的视图偏好跨课程记忆

// Boss 关(章末综合关):卡流里金卡。覆盖 综合关·(心理/营销/会用AI/会说话)、Boss(逻辑)、全英模拟面(英语)。
function isBoss(n) {
  const name = n.name || '';
  return n.slug === 'mock-full-interview' || n.slug.indexOf('boss-') === 0
    || name.indexOf('综合关') >= 0 || name.indexOf('Boss') >= 0;
}

Page({
  data: {
    id: '',
    name: '',
    accent: '#18b690',
    icon: '🎓',         // 课程 emoji:吉祥物舞台占位(真立绘后续用 gen-learn-icons 管线补)
    loading: true,
    viewMode: 'stream', // 'stream' 第N课卡片流(默认) / 'map' 闯关地图
    skill: null,        // 技能型(英语):四格=段位/流利/准确/表达
    streakDays: 0,      // 概念型四格:连学/点亮/掌握/复习
    lit: 0,
    total: 0,
    mastered: 0,
    due: 0,
    sections: [],       // [{ key, tier, theme, tone, lit, total, nodes: [{slug,name,blurb,hook,note,state,emoji,x,num,boss,flavor}] }]
    current: null,      // 当前关(第一个 level=0)
    currentAnchor: '',  // 地图视图 scroll-into-view 锚点
    streamAnchor: '',   // 卡流视图 scroll-into-view 锚点
    card: null,         // 关卡卡片抽屉
  },

  async onLoad(query) {
    const id = query.id;
    if (!id) { this.setData({ loading: false }); return; }
    let viewMode = 'stream';
    try { if (wx.getStorageSync(VIEW_MODE_KEY) === 'map') viewMode = 'map'; } catch (e) {}
    this.setData({
      id,
      name: query.name ? decodeURIComponent(query.name) : '',
      accent: query.accent ? decodeURIComponent(query.accent) : '#18b690',
      icon: query.icon ? decodeURIComponent(query.icon) : '🎓',
      viewMode,
    });
    if (this.data.name) wx.setNavigationBarTitle({ title: this.data.name });
    try { await ensureLogin(); } catch (e) {}
    this.load();
  },

  onShow() {
    // 从对话页回来时刷新点亮状态(首次 onLoad 后 load 已在跑,靠 _loaded 防重)
    if (this._loaded) this.load();
  },

  async load() {
    try {
      const [cp, sk, st] = await Promise.all([
        agentConcepts(this.data.id),
        agentSkill(this.data.id).catch(() => ({ enabled: false })),
        learnStreak().catch(() => null),
      ]);
      this._loaded = true;
      if (!cp || !cp.enabled) {
        // 非概念型(不该进来):退回对话页
        wx.redirectTo({ url: `/pages/agent-chat/index?id=${this.data.id}&name=${encodeURIComponent(this.data.name)}` });
        return;
      }
      const view = this._build(cp);
      this.setData({
        loading: false,
        streakDays: (st && st.days) || 0,
        skill: sk && sk.enabled && sk.assessed > 0 ? sk : null,
        lit: cp.lit || 0,
        total: cp.total || 0,
        mastered: cp.mastered || 0,
        due: cp.due || 0,
        sections: view.sections,
        current: view.current,
        currentAnchor: view.current ? 'node-' + view.current.slug : '',
        streamAnchor: view.current ? 'card-' + view.current.slug : '',
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: (e && e.message) || '加载失败', icon: 'none' });
    }
  },

  // 把 tiers[].concepts[] 摊平成「主题分段 + 节点」(两视图共用)。
  // 课程表内 theme 天然聚簇(按 sort 序),遍历时 theme 变化即开新段。
  _build(cp) {
    const sections = [];
    let cur = null;          // 第一个 level=0 = 当前关
    let afterCur = false;    // 当前关之后的第一个未点亮 = available
    let gi = 0;              // 全局序号:蜿蜒相位 + 第N课编号
    let sec = null;
    (cp.tiers || []).forEach((t) => {
      (t.concepts || []).forEach((c) => {
        if (!sec || sec.theme !== c.theme) {
          sec = {
            key: t.tier + '-' + c.theme,
            tier: t.tier,
            theme: c.theme,
            tone: TONES[sections.length % TONES.length],
            lit: 0,
            total: 0,
            nodes: [],
          };
          sections.push(sec);
        }
        let state, emoji;
        if (c.level >= 2) { state = 'mastered'; emoji = '👑'; }
        else if (c.level >= 1) { state = 'lit'; emoji = '⭐'; }
        else if (!cur) { state = 'current'; emoji = '⭐'; }
        else if (!afterCur) { state = 'available'; emoji = '✨'; afterCur = true; }
        else { state = 'locked'; emoji = '🔒'; }
        const node = {
          slug: c.slug, name: c.name, blurb: c.blurb, hook: c.hook || '', note: c.note || '',
          state, emoji, icon: ICON_BASE + STATE_ICONS[state] + '.webp', x: SNAKE[gi % SNAKE.length],
          num: gi + 1, boss: isBoss(c), flavor: FLAVOR[state],
        };
        if (state === 'current') cur = node;
        sec.nodes.push(node);
        sec.total++;
        if (c.level >= 1) sec.lit++;
        gi++;
      });
    });
    return { sections, current: cur };
  },

  // 视图切换:卡片流 ⇄ 地图(偏好落 storage,跨课程记忆)
  toggleView() {
    const m = this.data.viewMode === 'stream' ? 'map' : 'stream';
    this.setData({ viewMode: m });
    try { wx.setStorageSync(VIEW_MODE_KEY, m); } catch (e) {}
    // wx:elif 重挂 scroll-view 后重设锚点,确保定位到当前关
    const { current } = this.data;
    if (current) {
      wx.nextTick(() => {
        this.setData(m === 'map'
          ? { currentAnchor: 'node-' + current.slug }
          : { streamAnchor: 'card-' + current.slug });
      });
    }
  },

  // 点节点/课卡 → 关卡卡片(软锁:锁定关也能点开,CTA 变「提前解锁」)
  openCard(e) {
    const { slug } = e.currentTarget.dataset;
    let node = null;
    this.data.sections.some((s) => {
      node = s.nodes.find((n) => n.slug === slug) || null;
      return !!node;
    });
    if (!node) return;
    this.setData({ card: { ...node, stateText: STATE_TEXT[node.state], cta: STATE_CTA[node.state] } });
  },
  closeCard() { this.setData({ card: null }); },
  noop() {},

  // 卡片 CTA / 底部「继续学习」/ 卡流内嵌「上这一课」→ 对话页自动开课
  startNode() {
    const c = this.data.card;
    if (!c) return;
    this.setData({ card: null });
    this._go(c.slug);
  },
  startFromCard(e) {
    const { slug } = e.currentTarget.dataset;
    if (slug) this._go(slug);
  },
  startCurrent() {
    if (this.data.current) this._go(this.data.current.slug);
    else this._go(''); // 全部点亮:直接进对话(复习/冲掌握)
  },
  _go(slug) {
    const q = `id=${this.data.id}&name=${encodeURIComponent(this.data.name)}`;
    wx.navigateTo({
      url: slug
        ? `/pages/agent-chat/index?${q}&concept=${slug}&auto=1`
        : `/pages/agent-chat/index?${q}`,
    });
  },
});
