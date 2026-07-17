const { ensureLogin } = require('../../utils/auth');
const { request } = require('../../utils/request');

const CHANNELS = [
  { value: 'enterprise', label: '企业 / 机构' },
  { value: 'creator', label: '内容创作者' },
  { value: 'community', label: '社群主理人' },
  { value: 'consultant', label: '顾问 / 培训师' },
  { value: 'local', label: '本地服务者' },
  { value: 'other', label: '其他渠道' },
];

const AUDIENCES = [
  { value: 'under_500', label: '500 人以内' },
  { value: '500_2000', label: '500～2,000 人' },
  { value: '2000_10000', label: '2,000～10,000 人' },
  { value: '10000_50000', label: '10,000～50,000 人' },
  { value: 'over_50000', label: '50,000 人以上' },
];

const STATUS_META = {
  pending: {
    eyebrow: 'APPLICATION RECEIVED',
    title: '申请已进入审核',
    desc: '我们会根据渠道匹配度进行审核，请保持联系电话畅通。',
    action: '修改申请资料',
    tone: 'pending',
  },
  approved: {
    eyebrow: 'WELCOME ON BOARD',
    title: '注册成功，欢迎加入',
    desc: '你的微否代理商资格已即时开通，后续合作信息将通过预留联系方式同步。',
    action: '',
    tone: 'approved',
  },
  rejected: {
    eyebrow: 'MORE INFO NEEDED',
    title: '申请资料需要补充',
    desc: '请根据审核建议完善资料后重新提交。',
    action: '补充申请资料',
    tone: 'rejected',
  },
  suspended: {
    eyebrow: 'ACCOUNT REVIEW',
    title: '当前合作资格已暂停',
    desc: '如需了解详情，请联系微否合作团队。',
    action: '',
    tone: 'suspended',
  },
};

function optionIndex(options, value) {
  const index = options.findIndex((item) => item.value === value);
  return index >= 0 ? index : -1;
}

function incomingInviteCode(options) {
  const raw = options.inviteCode || options.scene || '';
  try {
    const decoded = decodeURIComponent(raw).trim();
    const match = decoded.match(/^(?:inviteCode|code|ic)=([^&]+)$/i);
    return (match ? match[1] : decoded).trim().slice(0, 32);
  } catch (e) {
    return String(raw).trim().slice(0, 32);
  }
}

Page({
  data: {
    loading: true,
    saving: false,
    editing: true,
    application: null,
    statusMeta: null,
    channels: CHANNELS,
    audiences: AUDIENCES,
    channelIndex: -1,
    audienceIndex: -1,
    channelLabel: '',
    audienceLabel: '',
    regionParts: [],
    experienceCount: 0,
    form: {
      name: '',
      phone: '',
      region: '',
      channelType: '',
      audienceSize: '',
      experience: '',
      inviteCode: '',
      consent: false,
    },
  },

  async onLoad(options) {
    const inviteCode = incomingInviteCode(options || {});
    this.setData({ 'form.inviteCode': inviteCode });
    try {
      await ensureLogin();
      const application = await request({ url: '/agency/application' });
      if (application) this.applyApplication(application);
    } catch (e) {
      wx.showToast({ title: e.message || '加载失败，请稍后重试', icon: 'none' });
    } finally {
      this.setData({ loading: false });
    }
  },

  applyApplication(application) {
    const channelIndex = optionIndex(CHANNELS, application.channelType);
    const audienceIndex = optionIndex(AUDIENCES, application.audienceSize);
    const experience = application.experience || '';
    this.setData({
      application,
      statusMeta: STATUS_META[application.status] || STATUS_META.pending,
      editing: false,
      channelIndex,
      audienceIndex,
      channelLabel: channelIndex >= 0 ? CHANNELS[channelIndex].label : '',
      audienceLabel: audienceIndex >= 0 ? AUDIENCES[audienceIndex].label : '',
      regionParts: (application.region || '').split(' ').filter(Boolean),
      experienceCount: experience.length,
      form: {
        name: application.name || '',
        phone: application.phone || '',
        region: application.region || '',
        channelType: application.channelType || '',
        audienceSize: application.audienceSize || '',
        experience,
        inviteCode: application.inviteCode || this.data.form.inviteCode,
        consent: false,
      },
    });
  },

  startEdit() {
    if (!this.data.statusMeta || !this.data.statusMeta.action) return;
    this.setData({ editing: true, 'form.consent': false });
    setTimeout(() => wx.pageScrollTo({ selector: '#apply-form', duration: 320 }), 50);
  },

  cancelEdit() {
    if (!this.data.application) return;
    this.applyApplication(this.data.application);
    wx.pageScrollTo({ scrollTop: 0, duration: 260 });
  },

  goAgencyCenter() {
    wx.redirectTo({ url: '/pages/agency-center/index' });
  },

  onInput(e) {
    const key = e.currentTarget.dataset.key;
    const value = e.detail.value;
    const patch = { [`form.${key}`]: value };
    if (key === 'experience') patch.experienceCount = value.length;
    this.setData(patch);
  },

  onRegionChange(e) {
    const parts = e.detail.value || [];
    this.setData({ regionParts: parts, 'form.region': parts.join(' ') });
  },

  onChannelChange(e) {
    const channelIndex = Number(e.detail.value);
    this.setData({ channelIndex, channelLabel: CHANNELS[channelIndex].label, 'form.channelType': CHANNELS[channelIndex].value });
  },

  onAudienceChange(e) {
    const audienceIndex = Number(e.detail.value);
    this.setData({ audienceIndex, audienceLabel: AUDIENCES[audienceIndex].label, 'form.audienceSize': AUDIENCES[audienceIndex].value });
  },

  toggleConsent() {
    this.setData({ 'form.consent': !this.data.form.consent });
  },

  validate() {
    const form = this.data.form;
    if (!form.name.trim() || form.name.trim().length < 2) return '请填写真实姓名';
    if (!/^1[3-9][0-9]{9}$/.test(form.phone.trim())) return '请填写正确的手机号';
    if (!form.region) return '请选择所在地区';
    if (!form.channelType) return '请选择主要渠道类型';
    if (!form.audienceSize) return '请选择可触达用户规模';
    if (form.experience.trim().length < 10) return '请至少用 10 个字介绍渠道经验';
    if (!form.consent) return '请确认资料真实并同意用于代理商注册';
    return '';
  },

  async submit() {
    if (this.data.saving) return;
    const validationMessage = this.validate();
    if (validationMessage) {
      wx.showToast({ title: validationMessage, icon: 'none' });
      return;
    }

    this.setData({ saving: true });
    try {
      await ensureLogin();
      const application = await request({
        url: '/agency/application',
        method: 'POST',
        data: { ...this.data.form, name: this.data.form.name.trim(), phone: this.data.form.phone.trim(), experience: this.data.form.experience.trim() },
      });
      this.applyApplication(application);
      wx.pageScrollTo({ scrollTop: 0, duration: 360 });
      wx.showToast({ title: '注册成功', icon: 'success' });
    } catch (e) {
      wx.showToast({ title: e.message || '提交失败，请稍后重试', icon: 'none' });
    } finally {
      this.setData({ saving: false });
    }
  },
});
