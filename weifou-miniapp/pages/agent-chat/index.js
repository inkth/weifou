const { ensureLogin } = require('../../utils/auth');
const { agentDetail, agentSessions, sessionMessages, chatAgent } = require('../../utils/agent');
const { status: membershipStatus } = require('../../utils/membership');
const { fmtDateTime } = require('../../utils/datetime');

Page({
  data: {
    id: '',
    name: '',
    icon: '',
    accent: '#18b690',
    greeting: '',
    member: false,
    remaining: 0,
    quotaText: '',
    messages: [],
    draft: '',
    pending: false,
    loading: true,
    scrollTo: '',
    sessions: [],         // 历史会话（抽屉）
    currentSessionId: '', // 当前续聊的会话；空 = 新开一段（下一条消息创建）
    historyVisible: false,
  },

  async onLoad(query) {
    const id = query.id;
    if (!id) {
      this.setData({ loading: false });
      return;
    }
    this.setData({ id, name: query.name ? decodeURIComponent(query.name) : '' });
    try {
      await ensureLogin();
    } catch (e) {}
    try {
      const [d, sessions, ms] = await Promise.all([
        agentDetail(id),
        agentSessions(id).catch(() => []),
        membershipStatus().catch(() => ({ isMember: false })),
      ]);
      const member = !!ms.isMember;
      // 默认载入最近一段会话；没有则以开场白起新的一段
      let messages = [];
      let currentSessionId = '';
      if (sessions && sessions.length) {
        currentSessionId = sessions[0].sessionId;
        const msgs = await sessionMessages(currentSessionId).catch(() => []);
        messages = (msgs || []).map((m) => ({ role: m.role, content: m.content }));
      }
      if (messages.length === 0 && d.greeting) messages = [{ role: 'assistant', content: d.greeting }];
      this.setData({
        name: d.name,
        icon: d.icon,
        accent: d.accent || '#18b690',
        greeting: d.greeting || '',
        member,
        remaining: d.freeTrialRemaining,
        quotaText: this._quota(member, d.freeTrialRemaining),
        messages,
        currentSessionId,
        sessions: this._decorate(sessions || []),
        loading: false,
      });
      if (d.name) wx.setNavigationBarTitle({ title: d.name });
      this._scrollEnd();
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  _quota(member, remaining) {
    return member ? '会员 · 畅用' : `免费体验剩 ${remaining} 次`;
  },

  _decorate(sessions) {
    return (sessions || []).map((s) => ({ ...s, timeText: fmtDateTime(s.updatedAt) }));
  },

  onInput(e) {
    this.setData({ draft: e.detail.value });
  },

  async send() {
    const content = (this.data.draft || '').trim();
    if (!content || this.data.pending) return;
    this.setData({
      draft: '',
      pending: true,
      messages: this.data.messages.concat({ role: 'user', content }),
    });
    this._scrollEnd();
    try {
      const data = await chatAgent(this.data.id, content, this.data.currentSessionId);
      const member = !!data.member;
      const remaining = member ? this.data.remaining : data.remaining;
      this.setData({
        messages: this.data.messages.concat({ role: 'assistant', content: data.answer }),
        member,
        remaining,
        quotaText: this._quota(member, remaining),
        // 新开一段时服务端回传新建会话 id，记下来后续消息续到同一段
        currentSessionId: data.sessionId || this.data.currentSessionId,
        pending: false,
      });
      this._scrollEnd();
    } catch (e) {
      this.setData({ pending: false });
      if (e.code === 'MEMBERSHIP_REQUIRED') {
        wx.showModal({
          title: '免费体验已用完',
          content: '开通会员即可畅用全部 AI 助手，不限次数。',
          confirmText: '去开通',
          cancelText: '再看看',
          success: (r) => {
            if (r.confirm) this.goMembership();
          },
        });
      } else {
        this.setData({
          messages: this.data.messages.concat({ role: 'assistant', content: e.message || '出错了，请稍后再试' }),
        });
        this._scrollEnd();
      }
    }
  },

  // —— 多会话 ——
  // 新开一段：清回开场白、清空 currentSessionId，下一条消息会创建新会话
  newSession() {
    const messages = this.data.greeting ? [{ role: 'assistant', content: this.data.greeting }] : [];
    this.setData({ messages, currentSessionId: '', historyVisible: false });
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
    this.setData({ historyVisible: false });
    try {
      const msgs = await sessionMessages(sid);
      this.setData({
        messages: (msgs || []).map((m) => ({ role: m.role, content: m.content })),
        currentSessionId: sid,
      });
      this._scrollEnd();
    } catch (err) {
      wx.showToast({ title: '加载失败', icon: 'none' });
    }
  },

  goMembership() {
    wx.navigateTo({ url: '/pages/membership/index' });
  },

  _scrollEnd() {
    const n = this.data.messages.length;
    if (n > 0) this.setData({ scrollTo: 'm' + (n - 1) });
  },
});
