// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_user.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

AuthUser _$AuthUserFromJson(Map<String, dynamic> json) => AuthUser(
  id: json['id'] as String,
  nickname: json['nickname'] as String?,
  avatarUrl: json['avatarUrl'] as String?,
  profileId: json['profileId'] as String?,
  profileStatus: json['profileStatus'] as String?,
);

Map<String, dynamic> _$AuthUserToJson(AuthUser instance) => <String, dynamic>{
  'id': instance.id,
  'nickname': instance.nickname,
  'avatarUrl': instance.avatarUrl,
  'profileId': instance.profileId,
  'profileStatus': instance.profileStatus,
};
