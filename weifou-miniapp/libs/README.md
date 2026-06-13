# 第三方运行时（vendored）

本目录放第三方运行时文件（非本项目源码）：

- `trtc-wx.js` — 腾讯云 TRTC 小程序 SDK（`trtc-wx-sdk@1.1.15`），通话页 `pages/call` 用。
- `lottie-miniprogram.js` — Lottie 播放运行时（`lottie-miniprogram@1.0.12`），动态形象组件 `components/avatar` 用。形象用法见 `assets/lottie/README.md`。

---

## TRTC 小程序 SDK

通话页 `pages/call` 依赖腾讯云 TRTC 小程序 SDK。

## 当前状态

已放入官方 SDK：`trtc-wx.js`（来自 npm 包 `trtc-wx-sdk@1.1.15`，腾讯官方发布）。

`pages/call/index.js` 已按该版本的真实 API 对接：
- `new TRTC(this)` → `createPusher` → `enterRoom`（字符串房间）→ `getPusherInstance().start()`
- 事件 `EVENT` 取自实例（`this.trtc.EVENT`），监听 `REMOTE_VIDEO_ADD/REMOVE`、`REMOTE_AUDIO_ADD/REMOVE`、`REMOTE_USER_LEAVE`、`ERROR`，同步 `playerList` 到页面
- 开关麦克风/摄像头通过 `setPusherAttributes({enableMic/enableCamera})`

## 升级 SDK

```bash
# 取最新版并替换本文件
npm pack trtc-wx-sdk        # 或指定版本 trtc-wx-sdk@x.y.z
tar xzf trtc-wx-sdk-*.tgz
cp package/trtc-wx.js <项目>/weifou-miniapp/libs/trtc-wx.js
```
升级后若 SDK 大版本 API 有变，需同步检查 `pages/call/index.js`。

## 提审前置（缺一不可）

1. 小程序后台开通「实时音视频」服务类目
2. 申请 `live-pusher` / `live-player` 组件使用权限（**个人主体不可用，需企业主体**）
3. 后端配置 `TRTC_SDK_APPID` / `TRTC_SECRET_KEY`（用于签发 UserSig）
