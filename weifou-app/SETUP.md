# weifou-app 配置清单

Flutter App 端。代码已就绪，以下为依赖**微信开放平台资质**与平台原生配置的待办，AppID 到位后按此填。

## 运行参数（--dart-define）

```bash
flutter run \
  --dart-define=API_BASE=https://api.weifou.com/api \
  --dart-define=WX_APP_ID=wxXXXXXXXXXXXX \
  --dart-define=WX_UNIVERSAL_LINK=https://weifou.com/app/ \
  --dart-define=SHARE_WEB_BASE=https://weifou.com/p
```

未配置 `WX_APP_ID` 时，微信登录会返回明确错误 `WX_NOT_CONFIGURED`，其余功能不受影响。

## 后端对应配置（weifou-server/.env）

```
WX_MOBILE_APPID=wxXXXXXXXXXXXX      # 移动应用 AppID（与上面一致）
WX_MOBILE_APPSECRET=xxxxxxxx        # 移动应用 AppSecret
```

后端按请求头 `X-Client-Type: app` 走移动应用 oauth2 登录分支，并用 unionid 与小程序账号打通（路线 A）。

## M1 · 微信登录（fluwx）原生配置 — 待 AppID

### iOS（ios/Runner/Info.plist）
- `CFBundleURLTypes` 增加 URL scheme：`wx<AppID>`
- `LSApplicationQueriesSchemes` 增加：`weixin`、`weixinULAPI`、`weixinURLParamsAPI`
- 开启 Associated Domains 能力，配置 Universal Link（与 `WX_UNIVERSAL_LINK` 一致）

### Android
- `applicationId` 须与开放平台移动应用登记的**包名**一致
- 签名证书的 **MD5/SHA** 须登记到开放平台（debug/release 各一套）

## M3 · 微信支付（fluwx pay）— 待商户号绑定移动应用 AppID

- 商户号（现小程序在用）需在微信支付后台**绑定移动应用 AppID**，开通 APP 支付
- 后端 `internal/wxpay` 需新增 `/v3/pay/transactions/app` 下单与 App 调起签名（M3 实现）

## M4 · TRTC 与权限 — 待集成 tencent_rtc_sdk

### iOS（Info.plist）
- `NSCameraUsageDescription` = 用于音视频咨询通话
- `NSMicrophoneUsageDescription` = 用于音视频咨询通话
- 如需锁屏续话：Background Modes 勾选 `audio`
- `NSPhotoLibraryAddUsageDescription` = 保存分享海报（poster 页）

### Android（android/app/src/main/AndroidManifest.xml）
- `CAMERA`、`RECORD_AUDIO`、`MODIFY_AUDIO_SETTINGS`、`INTERNET`、`BLUETOOTH_CONNECT`

## M5 · 上架
- iOS Privacy Manifest（含 fluwx / TRTC 第三方 SDK 域名与数据类型）
- 打赏入口 iOS 端隐藏（已由 `core/platform/entry_gate.dart` 控制，后端 `X-Platform` 兜底）
