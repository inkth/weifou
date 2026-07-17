// 技能 tab =「人类基本功计划」：列表只负责课程发现，VIP 边界在具体章节地图中表达。
// 首页已收敛为纯名片，不再承载「添加到首页」，故此处只做浏览 + 进入。
// 上架范围由服务端 /agents（enabled=true）决定，前端不再做名单过滤。
const { ensureLogin } = require('../../utils/auth');
const { listAgents, learningSummary } = require('../../utils/agent');

// 全部课程可直接浏览，也可按四个用户目标筛选。课程的营销表达独立于教学长介绍：
// 卡片先说清「学完能做什么」，再用服务端的课程介绍补足内容与方法。
const CATEGORIES = [
  { key: 'all', name: '全部' },
  { key: 'cognition', name: '认知' },
  { key: 'work', name: '事业' },
  { key: 'life', name: '成长' },
  { key: 'relation', name: '关系' },
];

const COURSE_PRESENTATION = {
  'spoken-english': {
    category: 'work', outcome: '真实场景一来，听得懂，也接得住',
    highlights: ['日常办事', '旅行应急', '职场协作'], courseMeta: '32 关 · 4 幕',
  },
  'learn-psychology': {
    category: 'cognition', outcome: '看懂情绪、关系与影响行为的心理机制',
    highlights: ['认识自己', '经营关系', '识别影响'], courseMeta: '80 关 · 3 幕',
  },
  'learn-logic': {
    category: 'cognition', outcome: '面对观点、数据和证据，形成可靠判断',
    highlights: ['拆论证', '识谬误', '查信源'], courseMeta: '68 关 · 6 幕',
  },
  'learn-marketing': {
    category: 'work', outcome: '从看见需求，到让价值被理解和选择',
    highlights: ['用户洞察', '市场定位', '成交增长'], courseMeta: '50 关 · 3 幕',
  },
  'learn-ai': {
    category: 'work', outcome: '让 AI 从“会聊天”变成“能交付”',
    highlights: ['拆任务', '补上下文', '验事实'], courseMeta: '28 关 · 2 幕',
  },
  'learn-speaking': {
    category: 'relation', outcome: '拒绝不伤人，冲突时还能把事情往前推',
    highlights: ['开口拒绝', '给出反馈', '处理冲突'], courseMeta: '28 关 · 2 幕',
  },
  'learn-lifedesign': {
    category: 'life', outcome: '把职业迷茫变成可以验证的下一步',
    highlights: ['好时光日志', '三版人生', '原型体验'], courseMeta: '21 关 · 3 幕',
  },
  'learn-love': {
    category: 'relation', outcome: '看懂心动、接住冲突、养住长期关系',
    highlights: ['识别吸引', '修复冲突', '长期联结'], courseMeta: '21 关 · 3 幕',
  },
  'learn-dating': {
    category: 'relation', outcome: '在心动中保持判断，真诚地开始或结束关系',
    highlights: ['认识自己', '保持判断', '真约起来'], courseMeta: '28 关 · 4 幕',
  },
  'learn-meditation': {
    category: 'life', outcome: '把注意力练稳，在走神和情绪起伏中随时回来',
    highlights: ['注意力训练', '情绪调节', '日常正念'], courseMeta: '21 关 · 3 幕',
  },
  'learn-happiness': {
    category: 'life', outcome: '识别幸福错觉，把有效行动放进这一周',
    highlights: ['幸福误判', '情绪纠偏', '行动设计'], courseMeta: '21 关 · 3 幕',
  },
  'learn-writing': {
    category: 'work', outcome: '让消息、邮件和汇报真正推动事情发生',
    highlights: ['结论先行', '写清请求', '说服行动'], courseMeta: '21 关 · 3 幕',
  },
  'learn-learning': {
    category: 'life', outcome: '把学过的知识记得住，也迁移得出来',
    highlights: ['检索练习', '间隔重复', '反馈迁移'], courseMeta: '21 关 · 3 幕',
  },
  'learn-negotiation': {
    category: 'work', outcome: '在砍价、加薪和合作里争取更多',
    highlights: ['准备底牌', '校准提问', '设计让步'], courseMeta: '21 关 · 3 幕',
  },
  'learn-habits': {
    category: 'life', outcome: '不靠意志力，重新设计行为发生的条件',
    highlights: ['两分钟开始', '环境设计', '反馈系统'], courseMeta: '21 关 · 3 幕',
  },
  'learn-business': {
    category: 'work', outcome: '看懂一门生意为何成立、如何持续增长',
    highlights: ['利润现金', '战略优势', '增长杠杆'], courseMeta: '21 关 · 3 幕',
  },
  'daodejing-full': {
    category: 'cognition', outcome: '把《老子》的思想带回今天的选择与关系',
    highlights: ['逐章读懂', '当代处境', '人生实践'], courseMeta: '81 章 · 9 幕',
  },
};

const COURSE_COVER_BASE = '/assets/courses/covers';

function decorate(a, index) {
  const order = index + 1;
  const presentation = COURSE_PRESENTATION[a.slug] || {};
  const hasCover = Object.prototype.hasOwnProperty.call(COURSE_PRESENTATION, a.slug);
  return {
    ...a,
    courseNo: order < 10 ? `0${order}` : `${order}`,
    cardTone: order % 3,
    cover: hasCover ? `${COURSE_COVER_BASE}/${a.slug}.webp` : '',
    marketCategory: presentation.category || 'cognition',
    outcome: presentation.outcome || a.tagline,
    highlights: presentation.highlights || [],
    courseMeta: presentation.courseMeta || '',
  };
}

Page({
  data: {
    statusBarH: 20,
    loading: true,
    hasCourses: false,
    categories: CATEGORIES,
    selectedCategory: 'all',
    courses: [],
  },

  onLoad() {
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this.setData({ statusBarH: info.statusBarHeight || 20 });
    } catch (e) { /* 兜底默认 20 */ }
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 1 });
    }
    this.load();
  },

  async load() {
    // stale-while-revalidate：有旧数据就先画着、后台静默刷新，骨架屏只留给真正的首次进入。
    // 否则每次切回本 Tab 都闪一次骨架——最高频的「不流畅」感知点。
    const firstLoad = !this._agents;
    if (firstLoad) this.setData({ loading: true });
    try { await ensureLogin(); } catch (e) {}
    try {
      const [list, summary] = await Promise.all([
        listAgents().catch(() => []),
        learningSummary().catch(() => null),
      ]);
      this._agents = (list || []).map((a, index) => decorate(a, index));
      this._summary = summary || null;

      // 回访用户先看到最近所学；手动切过分类后，返回页面仍尊重用户选择。
      let selectedCategory = this.data.selectedCategory;
      if (!this._categoryChosen && summary && summary.current) {
        const current = this._agents.find((a) => a.id === summary.current.id);
        if (current) selectedCategory = current.marketCategory;
      }
      this.setData({
        hasCourses: this._agents.length > 0,
        // 连续 ≥2 天才展示（第 1 天谈不上"连续"，安静）
        streak: summary && summary.streak && summary.streak.days >= 2 ? summary.streak : null,
        // 到期复习：间隔重复的到期总数 + 到期最多的那门课（点条直达它的复习挑战）
        reviewDue: (summary && summary.reviewDue) || 0,
        reviewAgent: (summary && summary.reviewAgent) || null,
        loading: false,
      });
      this.renderCategory(selectedCategory);
    } catch (e) {
      // 静默刷新失败：旧内容继续可用，不打扰；首载失败才提示
      if (firstLoad) {
        this.setData({ loading: false });
        wx.showToast({ title: (e && e.message) || '加载失败', icon: 'none' });
      }
    }
  },

  selectCategory(e) {
    const key = e.currentTarget.dataset.category;
    if (!CATEGORIES.some((item) => item.key === key)) return;
    this._categoryChosen = true;
    this.renderCategory(key);
  },

  renderCategory(key) {
    const category = CATEGORIES.find((item) => item.key === key) || CATEGORIES[0];
    const all = this._agents || [];
    const candidates = category.key === 'all' ? all : all.filter((item) => item.marketCategory === category.key);
    const current = this._summary && this._summary.current;
    const currentCourse = current && candidates.find((item) => item.id === current.id);
    const ordered = currentCourse
      ? [currentCourse, ...candidates.filter((item) => item.id !== currentCourse.id)]
      : candidates;
    const courses = ordered.map((item) => {
      const isCurrent = !!(current && item.id === current.id);
      const total = isCurrent ? Number(current.total || 0) : 0;
      const lit = isCurrent ? Number(current.lit || 0) : 0;
      const courseCategory = CATEGORIES.find((entry) => entry.key === item.marketCategory);
      return {
        ...item,
        categoryName: (courseCategory && courseCategory.name) || '认知',
        isCurrent,
        cardLabel: isCurrent ? '继续学习' : `${(courseCategory && courseCategory.name) || '认知'}课程`,
        enterLabel: isCurrent ? '继续学习' : '查看课程',
        showProgress: isCurrent && total > 0,
        progressPercent: isCurrent ? Number(current.progressPercent || 0) : 0,
        progressText: isCurrent && total > 0 ? `已点亮 ${lit}/${total}` : '',
      };
    });

    this.setData({
      categories: CATEGORIES,
      selectedCategory: category.key,
      courses,
    });
  },

  // 点卡片主体：一律进对话页。概念型课在对话页顶部铺「横版会走的路」（舞台=地图合一），
  // 角色停在当前关、进来即续/秒开；非概念型就是普通试用对话。
  open(e) {
    const { id, name } = e.currentTarget.dataset;
    const a = (this._agents || []).find((x) => x.id === id);
    // 透传 icon/accent：对话页无网时也能立刻画出对的头像+配色骨架
    // game=1：技能页进来的都是课，首帧即套游戏皮，免得加载窗口先闪一下普通聊天顶栏（含「解锁全课」）
    wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}&accent=${encodeURIComponent((a && a.accent) || '')}&icon=${encodeURIComponent((a && a.icon) || '')}&game=1` });
  },

  // 到期复习条：直达到期最多那门课的复习挑战（review=1 由对话页自动开局）。
  goReview() {
    const r = this.data.reviewAgent;
    if (!r) return;
    wx.navigateTo({ url: `/pages/agent-chat/index?id=${r.id}&name=${encodeURIComponent(r.name || '')}&accent=${encodeURIComponent(r.accent || '')}&icon=${encodeURIComponent(r.icon || '')}&game=1&review=1` });
  },
});
