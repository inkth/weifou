// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Persona _$PersonaFromJson(Map<String, dynamic> json) => Persona(
  oneLiner: json['oneLiner'] as String?,
  fullIntro: json['fullIntro'] as String?,
  tags:
      (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  starters:
      (json['starters'] as List<dynamic>?)?.map((e) => e as String).toList() ??
      const [],
  greeting: json['greeting'] as String?,
  tone: json['tone'] as String?,
  voiceStyle: json['voiceStyle'] as String?,
  avatarUrl: json['avatarUrl'] as String?,
);

Map<String, dynamic> _$PersonaToJson(Persona instance) => <String, dynamic>{
  'oneLiner': instance.oneLiner,
  'fullIntro': instance.fullIntro,
  'tags': instance.tags,
  'starters': instance.starters,
  'greeting': instance.greeting,
  'tone': instance.tone,
  'voiceStyle': instance.voiceStyle,
  'avatarUrl': instance.avatarUrl,
};

Profile _$ProfileFromJson(Map<String, dynamic> json) => Profile(
  id: json['id'] as String,
  realName: json['realName'] as String,
  title: json['title'] as String,
  company: json['company'] as String?,
  city: json['city'] as String?,
  nickname: json['nickname'] as String?,
  avatarUrl: json['avatarUrl'] as String?,
  avatarStyle: json['avatarStyle'] as String?,
  status: json['status'] as String?,
  contactVisible: json['contactVisible'] as bool? ?? false,
  discoverable: json['discoverable'] as bool? ?? false,
  contactWechat: json['contactWechat'] as String?,
  contactPhone: json['contactPhone'] as String?,
  persona: json['persona'] == null
      ? null
      : Persona.fromJson(json['persona'] as Map<String, dynamic>),
);

Map<String, dynamic> _$ProfileToJson(Profile instance) => <String, dynamic>{
  'id': instance.id,
  'realName': instance.realName,
  'title': instance.title,
  'company': instance.company,
  'city': instance.city,
  'nickname': instance.nickname,
  'avatarUrl': instance.avatarUrl,
  'avatarStyle': instance.avatarStyle,
  'status': instance.status,
  'contactVisible': instance.contactVisible,
  'discoverable': instance.discoverable,
  'contactWechat': instance.contactWechat,
  'contactPhone': instance.contactPhone,
  'persona': instance.persona,
};
