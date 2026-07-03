// 闯关地图：概念型学习 Agent 的课程主页（多邻国式蜿蜒路径）。
// 学心理 = 概念图鉴路径；英语陪练 = 真实场景闯关路径。数据同源 GET /agents/concepts/:id。
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

Page({
  data: {
    id: '',
    name: '',
    accent: '#18b690',
    loading: true,
    skill: null,        // 技能型(英语):四格=段位/流利/准确/表达
    streakDays: 0,      // 概念型四格:连学/点亮/掌握/复习
    lit: 0,
    total: 0,
    mastered: 0,
    due: 0,
    sections: [],       // [{ key, tier, theme, tone, lit, total, nodes: [{slug,name,blurb,hook,state,emoji,x}] }]
    current: null,      // 当前关(第一个 level=0)
    currentAnchor: '',  // scroll-into-view 锚点
    card: null,         // 关卡卡片抽屉
  },

  async onLoad(query) {
    const id = query.id;
    if (!id) { this.setData({ loading: false }); return; }
    this.setData({
      id,
      name: query.name ? decodeURIComponent(query.name) : '',
      accent: query.accent ? decodeURIComponent(query.accent) : '#18b690',
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
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: (e && e.message) || '加载失败', icon: 'none' });
    }
  },

  // 把 tiers[].concepts[] 摊平成「主题分段 + 蜿蜒节点」。
  // 课程表内 theme 天然聚簇(按 sort 序),遍历时 theme 变化即开新段。
  _build(cp) {
    const sections = [];
    let cur = null;          // 第一个 level=0 = 当前关
    let afterCur = false;    // 当前关之后的第一个未点亮 = available
    let gi = 0;              // 全局序号,驱动蜿蜒相位
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
          slug: c.slug, name: c.name, blurb: c.blurb, hook: c.hook || '',
          state, emoji, icon: ICON_BASE + STATE_ICONS[state] + '.webp', x: SNAKE[gi % SNAKE.length],
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

  // 点节点 → 关卡卡片(软锁:锁定关也能点开,CTA 变「提前解锁」)
  openCard(e) {
    const { slug } = e.currentTarget.dataset;
    let node = null;
    this.data.sections.some((s) => {
      node = s.nodes.find((n) => n.slug === slug) || null;
      return !!node;
    });
    if (!node) return;
    const stateText = {
      mastered: '👑 已掌握',
      lit: '⭐ 已点亮 · 可冲掌握',
      current: '当前关卡',
      available: '下一关',
      locked: '未解锁',
    }[node.state];
    const cta = {
      mastered: '再练一遍',
      lit: '冲击掌握',
      current: '开始这一关',
      available: '开始这一关',
      locked: '提前解锁这一关',
    }[node.state];
    // 角落风味角标:只正向/中性,不做"运气不佳"式负面
    const flavor = {
      mastered: '传说达成',
      lit: '可再战',
      current: '今日主线',
      available: '新篇章',
      locked: '前方迷雾',
    }[node.state];
    this.setData({ card: { ...node, stateText, cta, flavor } });
  },
  closeCard() { this.setData({ card: null }); },
  noop() {},

  // 卡片 CTA / 底部「继续学习」→ 对话页自动开课
  startNode() {
    const c = this.data.card;
    if (!c) return;
    this.setData({ card: null });
    this._go(c.slug);
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
