const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');
const { unpinAgent } = require('../../utils/agent');

// 首页 = 我的 Agent 小队，由 /home/agents 驱动；清新浅色 + 薄荷青，成熟产品质感（大卡 + 真实数据）。
const FALLBACK = {
  chief: { name: '我的 AI 名片', initial: '名', online: true, hasProfile: false, profileId: '', line: '替你接待每个来访的人，有结果就喊你', stats: null },
  specialists: [
    { id: '', name: '学英语', initial: 'EN', line: '随时开口练，纠音、对话、一段段升级', kind: 'tool', pill: '剩 5 次' },
    { id: '', name: '学商业', initial: '商', line: '把卡住的生意拆开，给你能落地的下一步', kind: 'tool', pill: '剩 3 次' },
  ],
};

function greet() {
  const h = new Date().getHours();
  if (h < 6) return '夜深了';
  if (h < 11) return '上午好';
  if (h < 14) return '中午好';
  if (h < 18) return '下午好';
  return '晚上好';
}

Page({
  data: { statusBarH: 20, greeting: '你好', chief: FALLBACK.chief, specialists: [], loading: true },

  onLoad() {
    try {
      const info = (wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync()) || {};
      this.setData({ statusBarH: info.statusBarHeight || 20 });
    } catch (e) { /* 兜底默认 20 */ }
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 0 });
    }
    this.setData({ greeting: greet() });
    this.load();
  },

  async load() {
    this.setData({ loading: true });
    try { await ensureLogin(); } catch (e) { /* 未登录也照常铺卡 */ }

    const cards = await request({ url: '/home/agents' }).catch(() => null);
    if (!cards || !cards.length) {
      this.setData({ chief: FALLBACK.chief, specialists: FALLBACK.specialists, loading: false });
      return;
    }

    const primary = cards.find((c) => c.primary) || cards[0];
    const chief = {
      name: primary.name,
      initial: primary.initial || '名',
      line: primary.line,
      hasProfile: !!primary.ready,
      online: !!primary.ready,
      profileId: primary.profileId || '',
      stats: null,
    };
    // 已建名片：取真实数据填进大卡（成熟产品的 dashboard 感）。
    if (chief.hasProfile) {
      const s = await request({ url: '/visit/stats/mine' }).catch(() => null);
      if (s) {
        chief.stats = [
          { n: s.pv || 0, label: '浏览' },
          { n: s.uv || 0, label: '访客' },
          { n: s.askCount || 0, label: '问答' },
        ];
      }
    }

    const specialists = cards.filter((c) => c !== primary).map((c) => {
      const fr = c.freeRemaining;
      const pill = c.member ? '会员畅用' : (typeof fr === 'number' && fr >= 0 ? `免费剩 ${fr} 次` : '');
      // nudge=催课条：line 已是服务端算好的动态学习状态（下一个概念/待复习/段位弱项），高亮显示
      return { id: c.agentId || '', name: c.name, initial: c.initial, line: c.line, nudge: !!c.nudge, kind: c.type === 'dating' ? 'dating' : 'tool', pill, concept: !!c.concept, accent: c.accent || '' };
    });

    this.setData({ chief, specialists, loading: false });
  },

  enterChief() {
    if (this.data.chief.hasProfile) {
      wx.navigateTo({ url: `/pages/chat/index?profileId=${this.data.chief.profileId}` });
    } else {
      wx.navigateTo({ url: '/pages/onboarding/index' });
    }
  },

  goVisitors() { wx.navigateTo({ url: '/pages/visitors/index' }); },

  enterSpecialist(e) {
    const { id, name, kind } = e.currentTarget.dataset;
    if (kind === 'dating') {
      wx.navigateTo({ url: '/pages/dating/index' });
    } else if (kind === 'tool') {
      if (!id) { wx.showToast({ title: '正在上线，稍后再来', icon: 'none' }); return; }
      const sp = (this.data.specialists || []).find((s) => s.id === id);
      if (sp && sp.concept) {
        // 概念型学习 Agent → 闯关地图
        // initial 即 ToolAgent.Icon(emoji),透传给卡流的吉祥物舞台占位
        wx.navigateTo({ url: `/pages/learn-map/index?id=${id}&name=${encodeURIComponent(name || '')}&accent=${encodeURIComponent(sp.accent || '')}&icon=${encodeURIComponent(sp.initial || '')}` });
        return;
      }
      wx.navigateTo({ url: `/pages/agent-chat/index?id=${id}&name=${encodeURIComponent(name || '')}` });
    }
  },

  removeSpecialist(e) {
    const { id, name } = e.currentTarget.dataset;
    if (!id) return;
    wx.showActionSheet({
      itemList: [`从首页移除「${name}」`],
      success: async (res) => {
        if (res.tapIndex !== 0) return;
        try { await unpinAgent(id); this.load(); }
        catch (err) { wx.showToast({ title: (err && err.message) || '移除失败', icon: 'none' }); }
      },
      fail: () => {},
    });
  },

  onShareAppMessage() {
    return { title: '来微否，养一个替你把事办成的 AI 名片', path: '/pages/discover/index' };
  },
});
