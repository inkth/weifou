import 'package:json_annotation/json_annotation.dart';

part 'profile.g.dart';

/// AI 人格，对应后端 publicByID 的 persona 字段。
@JsonSerializable()
class Persona {
  Persona({
    this.oneLiner,
    this.fullIntro,
    this.tags = const [],
    this.starters = const [],
    this.greeting,
    this.tone,
    this.voiceStyle,
    this.avatarUrl,
  });

  final String? oneLiner;
  final String? fullIntro;
  final List<String> tags;
  final List<String> starters;

  /// 开场白：进入对话的首条 AI 消息（沉浸式）。
  final String? greeting;

  /// 语气/性格描述（人设深度定制可编辑）。
  final String? tone;

  /// 音色标识（Phase B TTS 用）。
  final String? voiceStyle;
  final String? avatarUrl;

  factory Persona.fromJson(Map<String, dynamic> json) =>
      _$PersonaFromJson(json);
  Map<String, dynamic> toJson() => _$PersonaToJson(this);
}

/// 主页（访客视图），对应 GET /profile/:id。
@JsonSerializable()
class Profile {
  Profile({
    required this.id,
    required this.realName,
    required this.title,
    this.company,
    this.city,
    this.nickname,
    this.avatarUrl,
    this.avatarStyle,
    this.status,
    this.contactVisible = false,
    this.discoverable = false,
    this.contactWechat,
    this.contactPhone,
    this.persona,
  });

  final String id;
  final String realName;
  final String title;
  final String? company;
  final String? city;
  final String? nickname;
  final String? avatarUrl;
  final String? avatarStyle;
  final String? status;
  final bool contactVisible;

  /// 是否公开到人物广场（opt-in）。
  final bool discoverable;

  /// 仅 /profile/mine 返回（本人视图）。
  final String? contactWechat;
  final String? contactPhone;

  final Persona? persona;

  factory Profile.fromJson(Map<String, dynamic> json) =>
      _$ProfileFromJson(json);
  Map<String, dynamic> toJson() => _$ProfileToJson(this);
}
