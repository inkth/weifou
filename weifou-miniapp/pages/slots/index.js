const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { fmtDateTime, todayStr, toISO } = require('../../utils/datetime');

Page({
  data: {
    today: todayStr(),
    date: todayStr(),
    time: '20:00',
    durationMin: 30,
    slots: [],
    saving: false,
    loadError: false,
    statusText: { open: '可预约', booked: '已被约', canceled: '已取消' },
  },

  async onShow() {
    await this.load();
  },

  async load() {
    this.setData({ loadError: false });
    try {
      await ensureLogin();
      const slots = await request({ url: '/consult/slots/mine' });
      slots.forEach((s) => (s.timeText = fmtDateTime(s.startAt)));
      this.setData({ slots });
    } catch (e) {
      // 不再静默吞错：标记错误态并提示，避免把加载失败显示成"还没有可约档期"
      this.setData({ loadError: true });
      wx.showToast({ title: e.message || '档期加载失败', icon: 'none' });
    }
  },

  onDate(e) {
    this.setData({ date: e.detail.value });
  },
  onTime(e) {
    this.setData({ time: e.detail.value });
  },
  setDur(e) {
    this.setData({ durationMin: Number(e.currentTarget.dataset.min) });
  },

  async addSlot() {
    if (this.data.saving) return;
    const startAt = toISO(this.data.date, this.data.time);
    if (new Date(startAt).getTime() <= Date.now()) {
      wx.showToast({ title: '请选择未来时间', icon: 'none' });
      return;
    }
    this.setData({ saving: true });
    try {
      await request({
        url: '/consult/slots',
        method: 'POST',
        data: { slots: [{ startAt, durationMin: this.data.durationMin }] },
      });
      wx.showToast({ title: '已添加', icon: 'success' });
      await this.load();
    } catch (e) {
      wx.showToast({ title: e.message || '添加失败', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },

  del(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '删除档期',
      content: '确定删除该档期吗？',
      success: async (r) => {
        if (!r.confirm) return;
        try {
          await request({ url: `/consult/slots/${id}`, method: 'DELETE' });
          await this.load();
        } catch (e) {
          wx.showToast({ title: e.message || '删除失败', icon: 'none' });
        }
      },
    });
  },
});
