const { fmtDateTime } = require('../../utils/datetime');
const { ensureLogin } = require('../../utils/auth');
const { questionDetail, answerQuestion } = require('../../utils/asyncq');
const { uploadVoice } = require('../../utils/upload');

const MAX_REC_MS = 300000; // 5 分钟，与后端 5MB 上限对齐

Page({
  data: {
    id: '',
    q: null,
    role: '',
    loading: true,
    answer: '',
    submitting: false,
    // 录音（主人语音答）
    recState: 'idle', // idle | recording | recorded
    recSeconds: 0, // 录制中实时秒 / 录完后时长
    voiceTempPath: '',
    voiceDuration: 0,
    uploading: false,
    // 播放（主人试听 / 访客收听）
    playing: false,
  },

  async onLoad(query) {
    this.setData({ id: query.id || '' });
    try { await ensureLogin(); } catch (e) {}
    this.load();
  },

  async load() {
    try {
      const q = await questionDetail(this.data.id);
      this.setData({ q: this._decorate(q), role: q.role, loading: false });
    } catch (e) {
      this.setData({ loading: false });
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  _decorate(q) {
    const statusText =
      q.status === 'pending' ? '待回答'
      : q.status === 'ai_answered' ? '分身已答'
      : q.status === 'answered' ? '已回答' : '';
    return {
      ...q,
      createdText: fmtDateTime(q.createdAt),
      answeredText: q.answeredAt ? fmtDateTime(q.answeredAt) : '',
      statusText,
    };
  },

  onAnswerInput(e) {
    this.setData({ answer: e.detail.value });
  },

  // ---------- 录音（主人语音答） ----------
  _recorder() {
    if (this._rec) return this._rec;
    const rec = wx.getRecorderManager();
    rec.onStart(() => {
      this.setData({ recState: 'recording', recSeconds: 0 });
      this._recTimer = setInterval(() => {
        this.setData({ recSeconds: this.data.recSeconds + 1 });
      }, 1000);
    });
    rec.onStop((res) => {
      if (this._recTimer) { clearInterval(this._recTimer); this._recTimer = null; }
      const dur = Math.max(1, Math.round((res.duration || 0) / 1000));
      this.setData({ recState: 'recorded', voiceTempPath: res.tempFilePath, voiceDuration: dur, recSeconds: dur });
    });
    rec.onError(() => {
      if (this._recTimer) { clearInterval(this._recTimer); this._recTimer = null; }
      this.setData({ recState: 'idle', recSeconds: 0 });
      wx.showToast({ title: '录音失败，请在设置里允许录音', icon: 'none' });
    });
    this._rec = rec;
    return rec;
  },

  toggleRecord() {
    if (this.data.recState === 'recording') {
      this._recorder().stop();
    } else {
      this._stopAudio();
      this._recorder().start({ format: 'mp3', duration: MAX_REC_MS, sampleRate: 16000, numberOfChannels: 1, encodeBitRate: 48000 });
    }
  },

  reRecord() {
    this._stopAudio();
    this.setData({ recState: 'idle', recSeconds: 0, voiceTempPath: '', voiceDuration: 0 });
  },

  // ---------- 播放（试听本地 / 收听远端） ----------
  _audio() {
    if (this._ac) return this._ac;
    const ac = wx.createInnerAudioContext();
    ac.onEnded(() => this.setData({ playing: false }));
    ac.onStop(() => this.setData({ playing: false }));
    ac.onError(() => { this.setData({ playing: false }); wx.showToast({ title: '播放失败', icon: 'none' }); });
    this._ac = ac;
    return ac;
  },
  _stopAudio() {
    if (this._ac) { try { this._ac.stop(); } catch (e) {} }
    if (this.data.playing) this.setData({ playing: false });
  },
  _play(src) {
    if (!src) return;
    if (this.data.playing) { this._stopAudio(); return; }
    const ac = this._audio();
    ac.src = src;
    ac.play();
    this.setData({ playing: true });
  },
  playPreview() { this._play(this.data.voiceTempPath); }, // 主人试听本地录音
  playAnswer() { this._play(this.data.q && this.data.q.voiceUrl); }, // 收听已提交语音

  async submit() {
    const text = (this.data.answer || '').trim();
    const hasVoice = this.data.recState === 'recorded' && this.data.voiceTempPath;
    if (!text && !hasVoice) {
      wx.showToast({ title: '打字或录一段语音都行', icon: 'none' });
      return;
    }
    if (this.data.submitting || this.data.uploading) return;
    this._stopAudio();
    try {
      let voiceUrl = '';
      let voiceDuration = 0;
      if (hasVoice) {
        this.setData({ uploading: true });
        voiceUrl = await uploadVoice(this.data.voiceTempPath);
        voiceDuration = this.data.voiceDuration;
        this.setData({ uploading: false });
      }
      this.setData({ submitting: true });
      await answerQuestion(this.data.id, { answer: text, voiceUrl, voiceDuration });
      wx.showToast({ title: '已回答', icon: 'success' });
      this.setData({ answer: '', recState: 'idle', recSeconds: 0, voiceTempPath: '', voiceDuration: 0 });
      this.load();
    } catch (e) {
      wx.showToast({ title: e.message || '提交失败', icon: 'none' });
    } finally {
      this.setData({ submitting: false, uploading: false });
    }
  },

  onUnload() {
    if (this._recTimer) { clearInterval(this._recTimer); this._recTimer = null; }
    if (this._rec && this.data.recState === 'recording') { try { this._rec.stop(); } catch (e) {} }
    if (this._ac) { try { this._ac.destroy(); } catch (e) {} this._ac = null; }
  },
});
