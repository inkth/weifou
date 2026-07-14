// 技能 tab =「人类基本功计划」：多条能力路径，第一幕免费、全课会员解锁后续。
// 首页已收敛为纯名片，不再承载「添加到首页」，故此处只做浏览 + 进入。
// 上架范围由服务端 /agents（enabled=true）决定，前端不再做名单过滤。
const { ensureLogin } = require('../../utils/auth');
const { listAgents, learnStreak, learningSummary } = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');

// 四个用户目标，而不是十五个并列入口。课程的营销表达独立于教学长介绍：
// 列表负责说清「学完能做什么」，进课后再展开研究依据与完整路径。
const CATEGORIES = [
  { key: 'cognition', name: '认知', copy: '看清世界与自己', featuredSlug: 'learn-logic' },
  { key: 'work', name: '成事', copy: '把想法变成结果', featuredSlug: 'learn-ai' },
  { key: 'life', name: '生活', copy: '设计更好的日常', featuredSlug: 'learn-habits' },
  { key: 'relation', name: '关系', copy: '理解人，也表达自己', featuredSlug: 'learn-speaking' },
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

function decorate(a, isMember, index) {
  const order = index + 1;
  const presentation = COURSE_PRESENTATION[a.slug] || {};
  return {
    ...a,
    courseNo: order < 10 ? `0${order}` : `${order}`,
    marketCategory: presentation.category || 'cognition',
    outcome: presentation.outcome || a.tagline,
    highlights: presentation.highlights || [],
    courseMeta: presentation.courseMeta || '',
    accessLabel: isMember ? '会员畅学' : '第一幕免费',
    ctaLabel: isMember ? '开始学习' : '免费开始',
  };
}

Page({
  data: {
    statusBarH: 20,
    loading: true,
    isMember: false,
    hasCourses: false,
    courseCount: 0,
    categories: CATEGORIES,
    selectedCategory: 'cognition',
    selectedCategoryName: '认知',
    selectedCategoryCopy: '看清世界与自己',
    featured: null,
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
    this.setData({ loading: true });
    try { await ensureLogin(); } catch (e) {}
    try {
      const [list, ms, st, summary] = await Promise.all([
        listAgents().catch(() => []),
        membershipStatus().catch(() => ({ isMember: false })),
        learnStreak().catch(() => null),
        learningSummary().catch(() => null),
      ]);
      const isMember = !!ms.isMember;
      this._agents = (list || []).map((a, index) => decorate(a, isMember, index));
      this._summary = summary || null;

      // 回访用户先看到最近所学；手动切过分类后，返回页面仍尊重用户选择。
      let selectedCategory = this.data.selectedCategory;
      if (!this._categoryChosen && summary && summary.current) {
        const current = this._agents.find((a) => a.id === summary.current.id);
        if (current) selectedCategory = current.marketCategory;
      }
      this.setData({
        isMember,
        hasCourses: this._agents.length > 0,
        courseCount: this._agents.length,
        // 连续 ≥2 天才展示（第 1 天谈不上"连续"，安静）
        streak: st && st.days >= 2 ? st : null,
        loading: false,
      });
      this.renderCategory(selectedCategory);
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: (e && e.message) || '加载失败', icon: 'none' });
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
    const candidates = all.filter((item) => item.marketCategory === category.key);
    const current = this._summary && this._summary.current;
    let featured = current && candidates.find((item) => item.id === current.id);
    const isCurrent = !!featured;
    if (!featured) {
      featured = candidates.find((item) => item.slug === category.featuredSlug) || candidates[0] || null;
    }

    if (featured) {
      const total = isCurrent ? Number(current.total || 0) : 0;
      const lit = isCurrent ? Number(current.lit || 0) : 0;
      featured = {
        ...featured,
        featuredLabel: isCurrent ? '继续学习' : '本类推荐',
        ctaLabel: isCurrent ? '继续学习' : (this.data.isMember ? '开始学习' : '免费开始第一幕'),
        showProgress: isCurrent && total > 0,
        progressPercent: isCurrent ? Number(current.progressPercent || 0) : 0,
        progressText: isCurrent && total > 0 ? `已点亮 ${lit}/${total}` : '',
      };
    }

    this.setData({
      categories: CATEGORIES.map((item) => ({
        ...item,
        count: all.filter((course) => course.marketCategory === item.key).length,
      })),
      selectedCategory: category.key,
      selectedCategoryName: category.name,
      selectedCategoryCopy: category.copy,
      featured,
      courses: candidates.filter((item) => !featured || item.id !== featured.id),
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

  goMembership() { wx.navigateTo({ url: '/pages/membership/index' }); },
});
