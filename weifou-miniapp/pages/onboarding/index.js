const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');
const { track } = require('../../utils/track');

// 对话式创建（唯一创建方式）：把原「5 步点选」融进对话——AI 逐步引导，结构化项（做什么/接待谁/气质）
// 直接给可点「快捷选项」气泡，点了即答；名字与一句话走输入/语音；也可自由说一段，由 /profile/extract
// 一次抽多字段并跳过已答步骤。必填 realName/title/strengths 齐即可上岗。最终走现有 POST /profile，零后端改动。
const OPENER = '嗨，我是来帮你建主页的 AI 助理～ 先用一两句话介绍下你自己：你是谁、平时做什么、最能帮别人解决什么问题？想到哪说到哪，也可以按住下方麦克风说。';

// —— 结构化选项（承自原 5 步点选）——
const DOMAINS = ['顾问·教练', '设计·创意', '开发·技术', '教育·培训', '医美·健康', '法律·财税', '电商·带货', '内容·创作', '生活服务'];
const AUDIENCES = [
  { label: '找合作', hk: '主要想接待：找合作' },
  { label: '想买你服务', hk: '主要想接待：想买我服务的人' },
  { label: '同行 · 招募', hk: '主要想接待：同行或想招募我的人' },
  { label: '粉丝 · 读者', hk: '主要想接待：我的粉丝或读者' },
  { label: '都行', hk: '主要想接待：各种来访者' },
];
const STYLES = [
  { label: '专业冷静', value: 'steady', desc: '严谨克制 · 先结论' },
  { label: '温暖亲和', value: 'warm', desc: '口语 · 先共情' },
  { label: '犀利直接', value: 'sharp', desc: '一针见血 · 不绕弯' },
  { label: '轻松幽默', value: 'humorous', desc: '有梗 · 不油腻' },
];

// 引导阶段（承自 5 步顺序）：字段 / 是否必填 / 提问 / 快捷选项类型
const STAGES = [
  { key: 'name', field: 'realName', required: true, ask: '先问一句——我该怎么称呼你？', chips: null },
  { key: 'domain', field: 'title', required: true, ask: '你主要是做什么的？挑一个最接近的，或直接说～', chips: 'domain' },
  { key: 'audience', field: 'howToKnow', required: false, ask: '主要想接待谁？', chips: 'audience' },
  { key: 'style', field: 'style', required: false, ask: '希望你的 AI 什么气质、说话调？', chips: 'style' },
  // strengths 走 AI 动态候选（/profile/suggest 按职业+受众现生成，可换一批）；打字/语音仍是兜底
  { key: 'substance', field: 'strengths', required: true, ask: '最后——你最能帮别人解决的一件事是什么？我按你的方向想了几个，点一个就行，说你自己的更好。', chips: 'strengths' },
];

function filled(form, field) {
  return !!(form[field] && String(form[field]).trim());
}

Page({
  data: {
    msgs: [],            // { id, role: 'ai' | 'me', text }
    input: '',
    thinking: false,     // 抽取请求在途
    submitting: false,
    recording: false,    // 语音录制中
    canFinish: false,    // realName+title+strengths 已齐
    confirmed: false,    // 已发过"了解你了"的确认语，避免重复
    toLast: '',
    chipKind: null,      // 当前展示的快捷选项：domain | audience | style | strengths | null
    strengthOpts: [],    // 卖点动态候选（AI 现生成）
    optsLoading: false,  // 候选生成中
    editMode: false,     // 已有主页 = 编辑态：预填资料、不逐项追问、改完即更新
    DOMAINS, AUDIENCES, STYLES,
    form: { realName: '', title: '', strengths: '', recentWork: '', howToKnow: '', style: '', company: '', city: '' },
  },

  async onLoad(query) {
    this._ref = (query && query.ref) || '';
    this._asked = {}; // 已问过的阶段（可选项只问一次，避免纠缠）
    this._locked = {}; // 经点选确定的字段：后续抽取不再覆盖，保住干净取值
    this._edit = false;
    track('onboarding_enter', this._ref);
    try {
      await ensureLogin();
      // 创建/编辑统一走对话（已无表单页）：已有主页则进入编辑态
      const mine = await request({ url: '/profile/mine' });
      if (mine) { this._enterEditMode(mine); return; }
    } catch (e) { /* 未登录/无主页 → 创建态，提交时再兜底 */ }
    this._pushAi(OPENER);
  },

  // 编辑态：用 /profile/mine 预填全部字段（含 company/city，避免提交被空值清掉），
  // 开场回显当前主页，必填已齐 → 直接可「更新主页」；不再逐项追问。
  _enterEditMode(mine) {
    const input = mine.personaInput || {};
    const form = {
      realName: mine.realName || '',
      title: mine.title || '',
      strengths: input.strengths || '',
      recentWork: input.recentWork || '',
      howToKnow: input.howToKnow || '',
      style: input.style || '',
      company: mine.company || '',
      city: mine.city || '',
    };
    this._edit = true;
    this.setData({ editMode: true, form, canFinish: true, confirmed: true });
    this._pushAi(`这是你现在的 AI 主页——${form.realName}｜${form.title}。想更新点什么？换一句话简介、改说话语气、补个最近在做的事……跟我说就行，没提到的都给你留着。改完点「更新主页」。`);
  },

  _push(role, text) {
    const id = 'm' + this.data.msgs.length;
    this.setData({ msgs: this.data.msgs.concat([{ id, role, text }]), toLast: id });
  },
  _pushAi(text) { this._push('ai', text); },

  onInput(e) { this.setData({ input: e.detail.value }); },

  async onSend() {
    if (this.data.thinking || this.data.submitting) return;
    const val = (this.data.input || '').trim();
    if (!val) return;
    if (val.length > 600) { wx.showToast({ title: '太长了，分两次说吧', icon: 'none' }); return; }
    this._push('me', val);
    this.setData({ input: '', thinking: true, chipKind: null });
    await this._extract();
  },

  // 供语音输入复用：把转写文本当作一次发送
  sendText(text) {
    const val = (text || '').trim();
    if (!val || this.data.thinking || this.data.submitting) return;
    this.setData({ input: val });
    this.onSend();
  },

  // 点选快捷选项 = 直接填字段 + 记一条用户气泡，不走服务端抽取（快、稳、零误判）
  pickChip(e) {
    if (this.data.thinking || this.data.submitting) return;
    const { field, value, label } = e.currentTarget.dataset;
    this._locked[field] = true; // 点选即锁定，避免后续 LLM 抽取覆盖
    this._push('me', label);
    const form = Object.assign({}, this.data.form, { [field]: value });
    this.setData({ form, chipKind: null });
    this._afterAnswer();
  },

  // —— 语音输入（微信同声传译插件 WechatSI）——
  // 需在小程序后台「设置→第三方设置→插件管理」添加「微信同声传译」插件（appid wx069ba97219f66d99）。
  // 插件不可用时静默降级为提示打字，不阻断流程。
  _asrManager() {
    if (this._asr !== undefined) return this._asr;
    try {
      const plugin = requirePlugin('WechatSI');
      const mgr = plugin.getRecordRecognitionManager();
      mgr.onStop = (res) => {
        this.setData({ recording: false });
        const t = ((res && res.result) || '').trim();
        if (t) this.sendText(t);
        else wx.showToast({ title: '没听清，再说一次', icon: 'none' });
      };
      mgr.onError = () => {
        this.setData({ recording: false });
        wx.showToast({ title: '语音识别失败，试试打字', icon: 'none' });
      };
      this._asr = mgr;
    } catch (e) {
      this._asr = null; // 插件未添加 / 不支持
    }
    return this._asr;
  },

  onMicStart() {
    if (this.data.thinking || this.data.submitting) return;
    const mgr = this._asrManager();
    if (!mgr) { wx.showToast({ title: '语音暂不可用，请打字', icon: 'none' }); return; }
    this.setData({ recording: true });
    mgr.start({ lang: 'zh_CN' });
  },

  onMicEnd() {
    if (!this.data.recording) return;
    if (this._asr) this._asr.stop(); // 结果在 onStop 回调，并复位 recording
  },

  async _extract() {
    const payload = this.data.msgs.map((m) => ({ role: m.role, text: m.text }));
    try {
      const res = await request({ url: '/profile/extract', method: 'POST', data: { messages: payload } });
      const cur = this.data.form;
      const lk = this._locked || {};
      // 合并：已点选锁定的字段保留不动；其余服务端非空则覆盖、空则保留（防 LLM 偶发漏带）
      const pick = (field, val) => (lk[field] ? cur[field] : (val || cur[field]));
      const form = {
        realName: pick('realName', res.realName),
        title: pick('title', res.title),
        strengths: pick('strengths', res.strengths),
        recentWork: pick('recentWork', res.recentWork),
        howToKnow: pick('howToKnow', res.howToKnow),
        style: pick('style', res.style),
      };
      this.setData({ form, thinking: false });
      this._afterAnswer();
    } catch (e) {
      this.setData({ thinking: false });
      this._pushAi('（网络好像有点慢，刚那句再说一遍试试？）');
    }
  },

  // 统一推进：算必填是否齐 → 找下一个该问的阶段 + 快捷选项 → 抛出问题。
  // 自由说一段可一次填多字段，已填的阶段自动跳过。
  _afterAnswer() {
    const form = this.data.form;
    const complete = filled(form, 'realName') && filled(form, 'title') && filled(form, 'strengths');
    this.setData({ canFinish: complete });

    // 编辑态：不逐项追问，改了回一句确认，随时可「更新主页」
    if (this._edit) {
      this.setData({ chipKind: null });
      this._pushAi('好，记下了～ 还想改别的就继续说，或点「更新主页」。');
      return;
    }

    // 必填刚齐：发一次确认语
    if (complete && !this.data.confirmed) {
      this.setData({ confirmed: true });
      this._pushAi('好，我大概了解你了～ 随时点「先上岗」，也可以再补两句让我更懂你。');
    }

    // 下一个该问的阶段：字段为空，且（必填 或 还没问过）
    const next = STAGES.find((s) => !filled(form, s.field) && (s.required || !this._asked[s.key]));
    if (!next) { this.setData({ chipKind: null }); return; }

    this._asked[next.key] = true;
    this._pushAi(next.ask);
    this.setData({ chipKind: next.chips || null });
    if (next.chips === 'strengths') this._loadStrengthOpts();
  },

  // —— 卖点动态候选：按已选职业+受众现生成 4 条供点选；换一批带 exclude 防重复 ——
  async _loadStrengthOpts() {
    const f = this.data.form;
    if (!filled(f, 'title')) return; // 没有职业信息生成不出好候选，留打字/语音兜底
    this._shownOpts = this._shownOpts || [];
    this.setData({ optsLoading: true, strengthOpts: [] });
    try {
      const res = await request({
        url: '/profile/suggest', method: 'POST',
        data: { title: f.title, audience: f.howToKnow || '', exclude: this._shownOpts },
      });
      const opts = (res && res.options) || [];
      this._shownOpts = this._shownOpts.concat(opts).slice(-24);
      // 仍停在卖点一步才展示（防慢返回覆盖已推进的界面）
      this.setData({ optsLoading: false, strengthOpts: this.data.chipKind === 'strengths' ? opts : [] });
    } catch (e) {
      this.setData({ optsLoading: false }); // 静默降级：只剩输入兜底，不打断流程
    }
  },

  moreStrengthOpts() {
    if (!this.data.optsLoading) this._loadStrengthOpts();
  },

  onFinishNow() {
    if (!this.data.canFinish || this.data.submitting) return;
    this._finish();
  },

  async _finish() {
    const f = this.data.form;
    if (!f.realName || !f.title || !f.strengths) {
      wx.showToast({ title: '还差一点点信息哦', icon: 'none' });
      return;
    }
    if (this.data.submitting) return;
    const edit = this.data.editMode;
    this.setData({ submitting: true, chipKind: null });
    this._pushAi(edit ? '好，这就替你更新主页，稍等几秒 ✨' : '好，我这就替你把主页生成出来，稍等 5–15 秒 ✨');
    wx.showLoading({ title: edit ? '更新中…' : 'AI 生成中…', mask: true });
    try {
      await ensureLogin();
      const body = {
        realName: f.realName,
        title: f.title,
        strengths: f.strengths,
        recentWork: f.recentWork || '',
        howToKnow: f.howToKnow || '',
        style: f.style || '',                 // 语气：点选了用点选值，否则由 AI 自行判断
        company: f.company || '',             // 编辑态透传，避免被服务端空值清掉；创建态为空=不落库
        city: f.city || '',
        avatarStyle: '',
      };
      if (this._ref) body.ref = this._ref;
      const data = await request({ url: '/profile', method: 'POST', data: body });
      wx.hideLoading();
      // fresh=1（"新鲜出炉"庆祝）仅创建时用，编辑不放
      const url = edit
        ? `/pages/profile/index?id=${data.id}&mine=1`
        : `/pages/profile/index?id=${data.id}&mine=1&fresh=1`;
      wx.redirectTo({ url });
    } catch (e) {
      wx.hideLoading();
      this.setData({ submitting: false });
      wx.showModal({ title: edit ? '更新失败' : '生成失败', content: e.message || '请稍后再试', showCancel: false });
    }
  },
});
