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
  },

  _bindEvents() {
    const t = this.trtc;
    const E = this.EVENT;
    if (!t || !E) return;
    // SDK 自动维护 playerList，远端流变化时同步到页面
    const syncPlayers = (event) => {
      const list = (event && event.data && event.data.playerList) || [];
      this.setData({ playerList: list });
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
