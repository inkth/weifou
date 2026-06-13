const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fmtDateTime } = require('../../utils/datetime');

Page({
  data: {
    tab: 'gaps', // gaps | leads | knowledge | sessions
    loading: true,
    gaps: [],
    leads: [],
    knowledge: [],
    sessions: [], // 会话回放：助理替我接待的访客对话
  },

  async onShow() {
    try {
      await ensureLogin();
    } catch (e) {}
    this.loadAll();
  },

  async loadAll() {
    this.setData({ loading: true });
    try {
      const [gaps, leads, knowledge, sessions] = await Promise.all([
        request({ url: '/profile/gaps' }),
        request({ url: '/profile/leads' }),
        request({ url: '/profile/knowledge' }),
        // 会话列表失败不拖垮整页（如尚未创建主页）
        request({ url: '/chat/sessions/host' }).catch(() => []),
      ]);
      this.setData({
        gaps: gaps || [],
        leads: (leads || []).map((l) => ({ ...l, timeText: fmtDateTime(l.createdAt) })),
        knowledge: knowledge || [],
        sessions: (sessions || []).map((s) => ({ ...s, timeText: fmtDateTime(s.updatedAt) })),
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  switchTab(e) {
    this.setData({ tab: e.currentTarget.dataset.tab });
  },

  // 会话回放：看助理替我说了什么
  goReplay(e) {
    const { id, name } = e.currentTarget.dataset;
    wx.navigateTo({ url: `/pages/replay/index?sessionId=${id}&name=${encodeURIComponent(name || '访客')}` });
  },

  // 回答一条缺口 → 入知识库，Agent 立刻变聪明
  answerGap(e) {
    const id = e.currentTarget.dataset.id;
    const question = e.currentTarget.dataset.q;
    wx.showModal({
      title: '回答这个问题',
      content: '',
      editable: true,
      placeholderText: `回答「${question}」，AI 之后就会这样答`,
      confirmText: '保存',
      success: async (r) => {
        if (!r.confirm) return;
        const content = (r.content || '').trim();
        if (!content) {
          wx.showToast({ title: '请填写回答', icon: 'none' });
          return;
        }
        try {
          await request({
            url: `/profile/gaps/${id}/answer`,
            method: 'POST',
            data: { topic: question, content },
          });
          wx.showToast({ title: '已加入知识库', icon: 'success' });
          this.loadAll();
        } catch (err) {
          wx.showToast({ title: err.message || '保存失败', icon: 'none' });
        }
      },
    });
  },

  dismissGap(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '忽略这个问题？',
      content: '忽略后不再提醒，但不影响访客继续提问。',
      success: async (r) => {
        if (!r.confirm) return;
        try {
          await request({ url: `/profile/gaps/${id}/dismiss`, method: 'POST' });
          this.loadAll();
        } catch (err) {
          wx.showToast({ title: err.message || '操作失败', icon: 'none' });
        }
      },
    });
  },

  // 手动新增一条知识
  addKnowledge() {
    wx.showModal({
      title: '新增一条资料',
      editable: true,
      placeholderText: '例如：我的报价是按项目计，5000 元起',
      confirmText: '保存',
      success: async (r) => {
        if (!r.confirm) return;
        const content = (r.content || '').trim();
        if (!content) return;
        try {
          await request({
            url: '/profile/knowledge',
            method: 'POST',
            data: { content },
          });
          wx.showToast({ title: '已保存', icon: 'success' });
          this.loadAll();
        } catch (err) {
          wx.showToast({ title: err.message || '保存失败', icon: 'none' });
        }
      },
    });
  },

  // 粘贴一段长文本，AI 自动整理成多条知识（轻量灌入）
  ingestKnowledge() {
    wx.showModal({
      title: '粘贴资料自动整理',
      editable: true,
      placeholderText: '粘贴简历 / 朋友圈 / 介绍文章，AI 会拆成多条问答',
      confirmText: '整理',
      success: async (r) => {
        if (!r.confirm) return;
        const text = (r.content || '').trim();
        if (!text) return;
        wx.showLoading({ title: 'AI 整理中…', mask: true });
        try {
          const res = await request({
            url: '/profile/knowledge/ingest',
            method: 'POST',
            data: { text },
          });
          wx.hideLoading();
          wx.showToast({
            title: res.count > 0 ? `已整理 ${res.count} 条` : '未提取到有效信息',
            icon: res.count > 0 ? 'success' : 'none',
          });
          if (res.count > 0) this.loadAll();
        } catch (err) {
          wx.hideLoading();
          wx.showToast({ title: err.message || '整理失败', icon: 'none' });
        }
      },
    });
  },

  deleteKnowledge(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '删除这条资料？',
      success: async (r) => {
        if (!r.confirm) return;
        try {
          await request({ url: `/profile/knowledge/${id}`, method: 'DELETE' });
          this.loadAll();
        } catch (err) {
          wx.showToast({ title: err.message || '删除失败', icon: 'none' });
        }
      },
    });
  },

  copyContact(e) {
    const contact = e.currentTarget.dataset.contact;
    if (!contact) return;
    wx.setClipboardData({ data: contact });
  },
});
