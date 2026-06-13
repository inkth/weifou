import 'dart:async';
import 'dart:typed_data';

import 'package:fluwx/fluwx.dart';

import '../../core/config/env.dart';
import '../../core/network/api_exception.dart';

/// 封装 fluwx：注册移动应用 + 发起微信授权拿 code。
///
/// 注意：真正可用需先在微信开放平台注册「移动应用」并配置 [Env.wxAppId] /
/// [Env.wxUniversalLink]（通过 --dart-define 注入）。AppID 未配置时
/// [getAuthCode] 会抛出明确错误，不影响其余流程编译与运行。
class WeChatAuthService {
  WeChatAuthService._();
  static final WeChatAuthService instance = WeChatAuthService._();

  final Fluwx _fluwx = Fluwx();
  bool _registered = false;

  /// 幂等注册。未配置 AppID 时直接返回 false。
  Future<bool> ensureRegistered() async {
    if (_registered) return true;
    if (Env.wxAppId.isEmpty) return false;
    _registered = await _fluwx.registerApi(
      appId: Env.wxAppId,
      universalLink:
          Env.wxUniversalLink.isEmpty ? null : Env.wxUniversalLink,
    );
    return _registered;
  }

  Future<bool> get isWeChatInstalled => _fluwx.isWeChatInstalled;

  /// 分享图片到微信。[timeline] 为 true 发朋友圈，否则发好友会话。
  /// 未配置 AppID 时抛错，调用方应回退系统分享。
  Future<void> shareImage(Uint8List bytes, {bool timeline = false}) async {
    final ok = await ensureRegistered();
    if (!ok) {
      throw ApiException('WX_NOT_CONFIGURED', '微信分享未配置（缺少移动应用 AppID）');
    }
    if (!await isWeChatInstalled) {
      throw ApiException('WX_NOT_INSTALLED', '未安装微信');
    }
    await _fluwx.share(WeChatShareImageModel(
      WeChatImageToShare(uint8List: bytes),
      scene: timeline ? WeChatScene.timeline : WeChatScene.session,
    ));
  }

  /// 发起微信授权，返回授权 code（交给后端换 token）。
  Future<String> getAuthCode() async {
    final ok = await ensureRegistered();
    if (!ok) {
      throw ApiException('WX_NOT_CONFIGURED', '微信登录未配置（缺少移动应用 AppID）');
    }
    if (!await isWeChatInstalled) {
      throw ApiException('WX_NOT_INSTALLED', '未安装微信');
    }

    final completer = Completer<String>();
    late final FluwxCancelable sub;
    sub = _fluwx.addSubscriber((response) {
      if (response is! WeChatAuthResponse) return;
      sub.cancel();
      if (completer.isCompleted) return;
      if (response.errCode == 0 && (response.code?.isNotEmpty ?? false)) {
        completer.complete(response.code!);
      } else if (response.errCode == -2) {
        completer.completeError(ApiException('WX_AUTH_CANCELED', '已取消登录'));
      } else {
        completer.completeError(
          ApiException('WX_AUTH_FAILED', response.errStr ?? '微信授权失败'),
        );
      }
    });

    final launched =
        await _fluwx.authBy(which: NormalAuth(scope: 'snsapi_userinfo'));
    if (!launched) {
      sub.cancel();
      throw ApiException('WX_AUTH_FAILED', '无法唤起微信授权');
    }

    // 兜底超时，避免用户中途离开导致永久挂起。
    return completer.future.timeout(
      const Duration(seconds: 120),
      onTimeout: () {
        sub.cancel();
        throw ApiException('WX_AUTH_TIMEOUT', '微信授权超时');
      },
    );
  }
}
