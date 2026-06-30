// 谁看过我：列出浏览过我名片的访客（全实名，只露登录访客）。点有名片的访客 → 看 TA 的名片。
const { ensureLogin } = require('../../utils/auth');
const { listVisitors } = require('../../utils/agent');

function fmtTime(iso) {
  if (!iso) return '';
  const t = new Date(iso).getTime();
  if (!t) return '';
  const diff = Date.now() - t;
  const m = Math.floor(diff / 60000);
  if (m < 1) return '刚刚';
  if (m < 60) return `${m} 分钟前`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} 小时前`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d} 天前`;
  return new Date(iso).toLocaleDateString();
}

Page({
  data: { loading: true, visitors: [] },

  async onLoad() {
    try { await ensureLogin(); } catch (e) {}
    try {
      const list = await listVisitors();
      this.setData({
        visitors: (list || []).map((v) => ({
          ...v,
          timeText: fmtTime(v.lastVisitAt),
          initial: (v.name || '微').slice(0, 1),
        })),
        loading: false,
      });
    } catch (e) {
      this.setData({ loading: false }); // 没名片/无访客 → 空态
    }
  },

  openVisitor(e) {
    const { id } = e.currentTarget.dataset;
    if (!id) { wx.showToast({ title: 'TA 还没有名片', icon: 'none' }); return; }
    wx.navigateTo({ url: `/pages/chat/index?profileId=${id}` });
  },
});
