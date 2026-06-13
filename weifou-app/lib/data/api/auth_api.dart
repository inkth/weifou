import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/providers.dart';
import '../models/auth_user.dart';

/// 登录态相关接口。后端按 X-Client-Type:app 走移动应用 oauth2 分支。
class AuthApi {
  AuthApi(this._ref);

  final Ref _ref;

  /// POST /auth/login —— 用 fluwx 授权得到的 code 换 token。
  /// 返回 (token, user)。
  Future<({String token, AuthUser user})> login(String code) async {
    final data = await _ref.read(dioClientProvider).post(
      '/auth/login',
      data: {'code': code},
    );
    final map = Map<String, dynamic>.from(data as Map);
    return (
      token: map['token'] as String,
      user: AuthUser.fromJson(Map<String, dynamic>.from(map['user'] as Map)),
    );
  }

  /// POST /auth/test-login —— 测试登录：手机号 + 固定验证码 654321。
  /// 仅后端非生产环境放行，绕开微信授权用于客户端联调。
  Future<({String token, AuthUser user})> testLogin(
    String phone,
    String code,
  ) async {
    final data = await _ref.read(dioClientProvider).post(
      '/auth/test-login',
      data: {'phone': phone, 'code': code},
    );
    final map = Map<String, dynamic>.from(data as Map);
    return (
      token: map['token'] as String,
      user: AuthUser.fromJson(Map<String, dynamic>.from(map['user'] as Map)),
    );
  }

  /// GET /user/me —— 校验 token 并取当前用户（含 profileId）。
  Future<AuthUser> me() async {
    final data = await _ref.read(dioClientProvider).get('/user/me');
    return AuthUser.fromJson(Map<String, dynamic>.from(data as Map));
  }
}

final authApiProvider = Provider<AuthApi>((ref) => AuthApi(ref));
