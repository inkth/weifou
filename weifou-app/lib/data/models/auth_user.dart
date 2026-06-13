import 'package:json_annotation/json_annotation.dart';

part 'auth_user.g.dart';

/// 登录用户，对应 /auth/login 与 /user/me 的返回。
@JsonSerializable()
class AuthUser {
  AuthUser({
    required this.id,
    this.nickname,
    this.avatarUrl,
    this.profileId,
    this.profileStatus,
  });

  final String id;
  final String? nickname;
  final String? avatarUrl;

  /// 当前用户的主页 ID（未创建则为 null）。/user/me 才返回。
  final String? profileId;
  final String? profileStatus;

  bool get hasProfile => profileId != null && profileId!.isNotEmpty;

  factory AuthUser.fromJson(Map<String, dynamic> json) =>
      _$AuthUserFromJson(json);
  Map<String, dynamic> toJson() => _$AuthUserToJson(this);
}
