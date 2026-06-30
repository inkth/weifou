// 记忆管理：你的 AI 名片/分身记住的关于你的资料（KnowledgeItem），对外回答时会用上。
// 复用 /profile/knowledge：列出 / 手动加 / 粘贴自动整理 / 删除。
const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');

Page({
  data: { loading: true, items: [] },

  onShow() { this.load(); },

  async load() {
    this.setData({ loading: true });
    try { await ensureLogin(); } catch (e) {}
    try {
      const list = await request({ url: '/profile/knowledge' });
      this.setData({ items: list || [], loading: false });
    } catch (e) {
      this.setData({ items: [], loading: false }); // 没名片/无记忆 → 空态
    }
  },

  // 粘贴一段长文本，AI 拆成多条记忆。
  ingestMemory() {
    wx.showModal({
      title: '粘贴资料自动整理',
      editable: true,
      placeholderText: '粘贴简历 / 朋友圈 / 介绍文章，AI 会拆成多条',
      confirmText: '整理',
      success: async (r) => {
        if (!r.confirm) return;
        const text = (r.content || '').trim();
        if (!text) return;
        wx.showLoading({ title: 'AI 整理中…', mask: true });
        try {
          const res = await request({ url: '/profile/knowledge/ingest', method: 'POST', data: { text } });
          wx.hideLoading();
          wx.showToast({
            title: res.count > 0 ? `已记住 ${res.count} 条` : '未提取到有效信息',
            icon: res.count > 0 ? 'success' : 'none',
          });
          if (res.count > 0) this.load();
        } catch (err) {
          wx.hideLoading();
          wx.showToast({ title: err.message || '整理失败', icon: 'none' });
        }
      },
    });
  },

  // 手动加一条记忆。
  addMemory() {
    wx.showModal({
      title: '加一条记忆',
      editable: true,
      placeholderText: '例如：我的报价按项目计，5000 元起',
      confirmText: '保存',
      success: async (r) => {
        if (!r.confirm) return;
        const content = (r.content || '').trim();
        if (!content) return;
        try {
          await request({ url: '/profile/knowledge', method: 'POST', data: { content } });
          wx.showToast({ title: '已记住', icon: 'success' });
          this.load();
        } catch (err) {
          wx.showToast({ title: err.message || '保存失败', icon: 'none' });
        }
      },
    });
  },

  deleteMemory(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '删除这条记忆？',
      success: async (r) => {
        if (!r.confirm) return;
        try {
          await request({ url: `/profile/knowledge/${id}`, method: 'DELETE' });
          this.load();
        } catch (err) {
          wx.showToast({ title: err.message || '删除失败', icon: 'none' });
        }
      },
    });
  },
});
