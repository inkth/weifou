import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/providers.dart';
import '../../data/api/auth_api.dart';
import '../../data/models/auth_user.dart';
import 'wechat_auth_service.dart';

/// 登录态。
sealed class AuthState {
  const AuthState();
}

/// 启动恢复中（未知）。
class AuthUnknown extends AuthState {
  const AuthUnknown();
}

/// 未登录（访客）。
class AuthGuest extends AuthState {
  const AuthGuest();
}

/// 已登录。
class AuthAuthed extends AuthState {
  const AuthAuthed(this.user);
  final AuthUser user;
}

class AuthController extends Notifier<AuthState> {
  @override
  AuthState build() {
    // 监听 401 信号：token 失效即退回访客态。
    ref.listen(unauthorizedSignalProvider, (_, _) {
      state = const AuthGuest();
    });
    // 启动时尝试用已存 token 恢复会话。
    Future.microtask(_restore);
    return const AuthUnknown();
  }

  Future<void> _restore() async {
    try {
      final token = await ref.read(secureStoreProvider).getToken();
      if (token == null || token.isEmpty) {
        state = const AuthGuest();
        return;
      }
      final user = await ref.read(authApiProvider).me();
      state = AuthAuthed(user);
    } catch (_) {
      // token 失效 / 网络异常 / 插件缺失（测试环境）→ 访客态。
      state = const AuthGuest();
    }
  }

  /// 微信授权登录。成功返回 true。
  Future<void> loginWithWeChat() async {
    final code = await WeChatAuthService.instance.getAuthCode();
    final result = await ref.read(authApiProvider).login(code);
    await ref.read(secureStoreProvider).setToken(result.token);
    state = AuthAuthed(result.user);
  }

  /// 测试登录：手机号 + 固定验证码 654321（仅联调用，后端生产环境会拒绝）。
  Future<void> loginWithTestCode(String phone, String code) async {
    final result = await ref.read(authApiProvider).testLogin(phone, code);
    await ref.read(secureStoreProvider).setToken(result.token);
    state = AuthAuthed(result.user);
  }

  Future<void> logout() async {
    await ref.read(secureStoreProvider).clearToken();
    state = const AuthGuest();
  }
}

final authControllerProvider =
    NotifierProvider<AuthController, AuthState>(AuthController.new);
