/// 全局环境配置。
///
/// 通过 `--dart-define` 注入，便于 dev / prod 切换，避免把密钥写死进包：
/// flutter run --dart-define=API_BASE=https://api.weifou.com/api
///
/// 生产 API_BASE 必须是已备案的 HTTPS 域名（对应小程序 utils/config.js 的注意事项）。
class Env {
  Env._();

  /// 后端 API 根地址。默认指向本地后端（模拟器/真机调试需改为局域网 IP 或 HTTPS 域名）。
  static const String apiBase = String.fromEnvironment(
    'API_BASE',
    defaultValue: 'http://localhost:3000/api',
  );

  /// 微信开放平台「移动应用」AppID（区别于小程序 AppID）。登录/支付/分享用。
  static const String wxAppId = String.fromEnvironment('WX_APP_ID');

  /// iOS Universal Link，fluwx 回跳需要。
  static const String wxUniversalLink =
      String.fromEnvironment('WX_UNIVERSAL_LINK');

  /// 分享落地页根地址，用于海报二维码（已装唤起 App，未装落 H5）。
  static const String shareWebBase = String.fromEnvironment(
    'SHARE_WEB_BASE',
    defaultValue: 'https://weifou.com/p',
  );

  /// 网络超时（毫秒），与小程序 request.js 的 60s 对齐。
  static const int httpTimeoutMs = 60000;

  /// 是否显示「测试验证码登录」入口（手机号 + 654321，绕开微信授权）。
  /// 联调阶段默认开启；上线前用 --dart-define=ENABLE_TEST_LOGIN=false 关闭，
  /// 后端也需 ENV=production 双重兜底。
  static const bool enableTestLogin = bool.fromEnvironment(
    'ENABLE_TEST_LOGIN',
    defaultValue: true,
  );
}
