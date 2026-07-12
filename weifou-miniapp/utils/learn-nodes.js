// 概念型学习课的「关卡节点」摊平逻辑（现仅 agent-chat 横版路使用；
// 纵版 learn-map 页与图鉴抽屉已退役——路即唯一地图）。
// 把 agentConcepts 返回的 tiers[].concepts[] 摊平成一条扁平序列 + 分幕信息，
// 逐节点算出状态机（当前关/下一关/已点亮/已掌握/未解锁）与会员锁、Boss 标记。
// 本模块不产坐标——只给扁平 index 与分幕 startIndex，由使用方自行计算 x。

const ICON_BASE = '/assets/icons/learn/';
const STATE_ICONS = { mastered: 'crown', lit: 'star', current: 'star', available: 'sparkle', locked: 'lock' };
// 状态词表（抽屉共用；风味角标只正向/中性，不做「运气不佳」式负面）
const STATE_TEXT = { mastered: '👑 已掌握', lit: '⭐ 已点亮 · 可冲掌握', current: '当前关卡', available: '下一关', locked: '未解锁' };
const STATE_CTA = { mastered: '再练一遍', lit: '冲击掌握', current: '开始这一关', available: '开始这一关', locked: '提前解锁这一关' };
const FLAVOR = { mastered: '传说达成', lit: '可再战', current: '今日主线', available: '新篇章', locked: '前方迷雾' };

// Boss 关（章末综合关）：覆盖 综合关·(心理/营销/会用AI/会说话)、Boss(逻辑)、全英模拟面(英语，slug 前缀 boss-)。
function isBoss(n) {
  const name = n.name || '';
  const slug = n.slug || '';
  return slug.indexOf('boss-') === 0
    || name.indexOf('综合关') >= 0 || name.indexOf('Boss') >= 0;
}

// 摊平 cp.tiers[].concepts[] → { nodes:[扁平], sections:[分幕], currentIndex, current }。
// 课程表内 theme 天然聚簇（按 sort 序），遍历时 theme 变化即开新段。
function buildNodes(cp) {
  const nodes = [];
  const sections = [];    // 每段记 startIndex（扁平下标），供横版在路上插幕旗
  let cur = null;         // 第一个 level=0 = 当前关
  let afterCur = false;   // 当前关之后的第一个未点亮 = available
  let sec = null;
  // 会员锁：非会员 + 概念课启用了幕门控（freeTier>0）时，Tier>freeTier 的整幕锁。
  const gate = !cp.isMember && (cp.freeTier || 0) > 0;
  const freeTier = cp.freeTier || 0;
  let idx = 0;
  (cp.tiers || []).forEach((t) => {
    const tierLocked = gate && (t.tierNum || 1) > freeTier;
    (t.concepts || []).forEach((c) => {
      if (!sec || sec.theme !== c.theme) {
        sec = {
          key: t.tier + '-' + c.theme,
          tier: t.tier,
          theme: c.theme,
          startIndex: idx,
          lit: 0,
          total: 0,
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
        idx,
        slug: c.slug,
        name: c.name,
        blurb: c.blurb,
        hook: c.hook || '',
        note: c.note || '',
        state,
        emoji,
        icon: ICON_BASE + STATE_ICONS[state] + '.webp',
        num: idx + 1,
        boss: isBoss(c),
        flavor: FLAVOR[state],
        memberLocked: tierLocked,
        secKey: sec.key,
      };
      if (state === 'current') cur = node;
      nodes.push(node);
      sec.total++;
      if (c.level >= 1) sec.lit++;
      idx++;
    });
  });
  return { nodes, sections, currentIndex: cur ? cur.idx : -1, current: cur };
}

// 给定节点掌握档 level（1 点亮 / 2 掌握），返回该状态对应的 webp 图标路径。
function iconForLevel(level) {
  return ICON_BASE + (level >= 2 ? 'crown' : 'star') + '.webp';
}

module.exports = { buildNodes, isBoss, iconForLevel, STATE_TEXT, STATE_CTA, FLAVOR, ICON_BASE };
