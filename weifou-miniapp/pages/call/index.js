const { request } = require('../../utils/request');
const { ensureLogin } = require('../../utils/auth');

// 动态加载 TRTC SDK（缺失时降级提示）
let TRTC = null;
try {
  TRTC = require('../../libs/trtc-wx');
} catch (e) {
  TRTC = null;
}

Page({
  data: {
    sessionId: '',
    sdkReady: false,
    fallbackText: '正在准备通话…',
    pusher: null,
    playerList: [],
    micOn: true,
    cameraOn: true,
    timerText: '00:00',
    durationMin: 30,
    remoteJoined: false, // 对方是否已进房（用于"未接"超时提示）
    waitHint: '',        // 等待区的二级提示文案
  },

  async onLoad(query) {
    this.setData({ sessionId: query.sessionId });

    if (!TRTC) {
      this.setData({
        sdkReady: false,
        fallbackText: '音视频组件未就绪：请将 TRTC SDK 放入 libs/（详见 libs/README.md），并在小程序后台开通实时音视频类目',
      });
      return;
    }

    try {
      await ensureLogin();
      const join = await request({
        url: `/rtc/consult/${query.sessionId}/join`,
        method: 'POST',
      });
      this.startCall(join);
    } catch (e) {
      this.setData({
        sdkReady: false,
        fallbackText: e.message || '进入通话失败',
      });
    }
  },

  startCall(join) {
    this.setData({ durationMin: join.durationMin, sdkReady: true });

    // 初始化 TRTC 实例（trtc-wx-sdk）。ctx 传 Page 实例。
    this.trtc = new TRTC(this);
    this.EVENT = this.trtc.EVENT;

    const pusherInstance = this.trtc.createPusher({
      enableCamera: true,
      enableMic: true,
      beautyLevel: 0,
    });
    this.setData({ pusher: pusherInstance.pusherAttributes });

    this._bindEvents();

    // 进房：字符串房间（后端房间号形如 consult_xxx）
    const pusherAttrs = this.trtc.enterRoom({
      sdkAppID: join.sdkAppId,
      userID: join.userId,
      userSig: join.userSig,
      strRoomID: join.roomId,
      scene: 'rtc',
    });
    this.setData({ pusher: pusherAttrs });
    // 启动本地推流
    this.trtc.getPusherInstance().start();

    // 通知后端通话开始
    request({ url: `/rtc/consult/${this.data.sessionId}/start`, method: 'POST' }).catch(() => {});
    this._startTimer();

    // 对方未接兜底：60s 内无人进房则给出可操作提示（不自动挂断，留给用户决定继续等或退出）。
    this._noShowTimer = setTimeout(() => {
      if (!this.data.remoteJoined) {
        this.setData({ waitHint: '对方还没进来，可以再等等；如长时间无人，可先退出稍后再约' });
      }
    }, 60000);
  },

  _bindEvents() {
    const t = this.trtc;
    const E = this.EVENT;
    if (!t || !E) return;
    // SDK 自动维护 playerList，远端流变化时同步到页面
    const syncPlayers = (event) => {
      const list = (event && event.data && event.data.playerList) || [];
      this.setData({ playerList: list });
      // 对方一旦进房，取消"未接"超时提示
      if (list.length > 0 && !this.data.remoteJoined) {
        this.setData({ remoteJoined: true, waitHint: '' });
        if (this._noShowTimer) { clearTimeout(this._noShowTimer); this._noShowTimer = null; }
      }
    };
    t.on(E.REMOTE_VIDEO_ADD, syncPlayers);
    t.on(E.REMOTE_VIDEO_REMOVE, syncPlayers);
    t.on(E.REMOTE_AUDIO_ADD, syncPlayers);
    t.on(E.REMOTE_AUDIO_REMOVE, syncPlayers);
    t.on(E.REMOTE_USER_LEAVE, (event) => {
      syncPlayers(event);
      wx.showToast({ title: '对方已离开', icon: 'none' });
    });
    t.on(E.ERROR, (event) => {
      const msg = (event && event.data && event.data.message) || '通话出错';
      wx.showToast({ title: msg, icon: 'none' });
    });
  },

  _startTimer() {
    const limitSec = this.data.durationMin * 60;
    let elapsed = 0;
    this.timer = setInterval(() => {
      elapsed += 1;
      const remain = limitSec - elapsed;
      const m = String(Math.floor(Math.abs(remain) / 60)).padStart(2, '0');
      const s = String(Math.abs(remain) % 60).padStart(2, '0');
      this.setData({ timerText: remain >= 0 ? `剩余 ${m}:${s}` : `超时 ${m}:${s}` });
      if (remain === 0) {
        wx.showToast({ title: '通话时长已到', icon: 'none' });
      }
      if (remain <= -120) {
        // 超时 2 分钟自动挂断
        this.hangup();
      }
    }, 1000);
  },

  // ---- live-pusher / live-player 原生事件代理给 SDK ----
  _pusherStateChangeHandler(e) {
    if (this.trtc) this.trtc.pusherEventHandler(e);
  },
  _pusherNetStatusHandler(e) {
    if (this.trtc) this.trtc.pusherNetStatusHandler(e);
  },
  _pusherErrorHandler(e) {
    if (this.trtc) this.trtc.pusherErrorHandler(e);
    // 本地推流出错（最常见是摄像头/麦克风未授权）：原先完全静默，用户会卡在"等待对方加入"黑屏。
    // 这里显式兜底——权限类引导去设置，其它错误给一次 toast；只提示一次避免重复弹窗。
    if (this._pusherErrShown) return;
    const errMsg = (e && e.detail && e.detail.errMsg) || '';
    if (/permission|auth|deny|授权|权限/i.test(errMsg)) {
      this._pusherErrShown = true;
      wx.showModal({
        title: '需要摄像头/麦克风权限',
        content: '请在设置中允许使用摄像头和麦克风，再重新进入通话',
        confirmText: '去设置',
        success: (m) => { if (m.confirm) wx.openSetting(); },
      });
    } else if (errMsg) {
      this._pusherErrShown = true;
      wx.showToast({ title: '设备出错，请检查摄像头/麦克风', icon: 'none' });
    }
  },
  _playerStateChange(e) {
    if (this.trtc) this.trtc.playerEventHandler(e);
  },

  toggleMic() {
    if (!this.trtc) return;
    const next = !this.data.micOn;
    const pusher = this.trtc.setPusherAttributes({ enableMic: next });
    this.setData({ micOn: next, pusher });
  },

  toggleCamera() {
    if (!this.trtc) return;
    const next = !this.data.cameraOn;
    const pusher = this.trtc.setPusherAttributes({ enableCamera: next });
    this.setData({ cameraOn: next, pusher });
  },

  async hangup() {
    await this._cleanup();
    wx.navigateBack();
  },

  back() {
    wx.navigateBack();
  },

  async _cleanup() {
    if (this._noShowTimer) {
      clearTimeout(this._noShowTimer);
      this._noShowTimer = null;
    }
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    if (this.trtc) {
      try {
        this.trtc.exitRoom();
      } catch (e) {}
    }
    await request({ url: `/rtc/consult/${this.data.sessionId}/end`, method: 'POST' }).catch(() => {});
  },

  onUnload() {
    this._cleanup();
  },
});
