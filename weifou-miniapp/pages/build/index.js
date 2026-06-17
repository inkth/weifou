const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { track } = require('../../utils/track');

// 气质/说话调（4 选 1）：定对外沟通风格（style → 语气）
const STYLES = [
  { key: 'steady', label: '专业冷静', desc: '严谨克制 · 先结论' },
  { key: 'warm', label: '温暖亲和', desc: '口语 · 先共情' },
  { key: 'sharp', label: '犀利直接', desc: '一针见血 · 不绕弯' },
  { key: 'humorous', label: '轻松幽默', desc: '有梗 · 不油腻' },
];
const DOMAINS = ['顾问·教练', '设计·创意', '开发·技术', '教育·培训', '医美·健康', '法律·财税', '电商·带货', '内容·创作', '生活服务'];
const AUDIENCES = [
  { key: '找合作的人', label: '找合作' },
  { key: '想买我服务的人', label: '想买你服务' },
  { key: '同行或想招募我的人', label: '同行 · 招募' },
  { key: '我的粉丝或读者', label: '粉丝 · 读者' },
  { key: '各种来访者', label: '都行' },
];

Page({
  data: {
    step: 0, // 0 名字 / 1 做什么 / 2 接待谁 / 3 气质 / 4 一句话
    total: 5,
    STYLES, DOMAINS, AUDIENCES,
    name: '', domain: '', audienceLabel: '', style: '', substance: '',
    submitting: false,
  },

  onLoad(q) {
    this._ref = (q && q.ref) || '';
    track('build_enter', this._ref);
  },

  // 返回上一步；首步则退出本页
  back() {
    if (this.data.step > 0) this.setData({ step: this.data.step - 1 });
    else wx.navigateBack({ delta: 1 });
  },

  onName(e) { this.setData({ name: e.detail.value }); },
  nameNext() {
    if (!this.data.name.trim()) { wx.showToast({ title: '先告诉我怎么称呼你', icon: 'none' }); return; }
    this.setData({ step: 1 });
  },

  // 点选即前进
  pickDomain(e) { this.setData({ domain: e.currentTarget.dataset.v, step: 2 }); },
  pickAudience(e) { this.setData({ audienceLabel: e.currentTarget.dataset.l, step: 3 }); },
  pickStyle(e) {
    const k = e.currentTarget.dataset.k;
    this.setData({ style: k, step: 4 });
  },

  onSubstance(e) { this.setData({ substance: e.detail.value }); },

  async finish() {
    if (this.data.submitting) return;
    if (!this.data.substance.trim()) { wx.showToast({ title: '说一句你最能帮上的事', icon: 'none' }); return; }
    this.setData({ submitting: true });
    wx.showLoading({ title: 'AI 生成中…', mask: true });
    try {
      await ensureLogin();
      // 点选 + 一句话 → 复用现有 POST /profile（persona-gen 据 strengths/title/style 产出人设）
      const body = {
        realName: this.data.name.trim(),
        title: this.data.domain,
        strengths: this.data.substance.trim(), // ← 血肉：让 AI 不通用
        recentWork: '',
        howToKnow: this.data.audienceLabel ? `主要想接待：${this.data.audienceLabel}` : '',
        style: this.data.style,
        avatarStyle: '',
      };
      if (this._ref) body.ref = this._ref;
      const data = await request({ url: '/profile', method: 'POST', data: body });
      wx.hideLoading();
      wx.redirectTo({ url: `/pages/profile/index?id=${data.id}&mine=1&fresh=1` });
    } catch (e) {
      wx.hideLoading();
      this.setData({ submitting: false });
      wx.showModal({ title: '生成失败', content: e.message || '请稍后再试', showCancel: false });
    }
  },

  // 更想自由说 → 对话式创建
  goOnboarding() { wx.redirectTo({ url: '/pages/onboarding/index' }); },
});
