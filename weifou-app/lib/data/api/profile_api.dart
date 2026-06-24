import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/providers.dart';
import '../models/profile.dart';

/// 创建主页的表单输入，对应后端 createReq。
class ProfileInput {
  ProfileInput({
    required this.realName,
    required this.title,
    this.company,
    this.city,
    required this.strengths,
    required this.recentWork,
    required this.howToKnow,
    this.avatarStyle,
    this.style,
  });

  final String realName;
  final String title;
  final String? company;
  final String? city;
  final String strengths;
  final String recentWork;
  final String howToKnow;
  final String? avatarStyle;
  final String? style; // 对外沟通风格，白名单见后端 persona.StyleDescriptions

  Map<String, dynamic> toJson() => {
        'realName': realName,
        'title': title,
        'company': ?company,
        'city': ?city,
        'strengths': strengths,
        'recentWork': recentWork,
        'howToKnow': howToKnow,
        'avatarStyle': ?avatarStyle,
        'style': ?style,
      };
}

/// 对话式创建的抽取结果，对应后端 persona.ExtractedProfile。
class ExtractedProfile {
  ExtractedProfile({
    this.realName = '',
    this.title = '',
    this.strengths = '',
    this.recentWork = '',
    this.howToKnow = '',
    this.style = '',
    this.followup = '',
    this.complete = false,
  });

  final String realName;
  final String title;
  final String strengths;
  final String recentWork;
  final String howToKnow;
  final String style;
  final String followup;
  final bool complete;

  factory ExtractedProfile.fromJson(Map<String, dynamic> j) => ExtractedProfile(
        realName: (j['realName'] ?? '') as String,
        title: (j['title'] ?? '') as String,
        strengths: (j['strengths'] ?? '') as String,
        recentWork: (j['recentWork'] ?? '') as String,
        howToKnow: (j['howToKnow'] ?? '') as String,
        style: (j['style'] ?? '') as String,
        followup: (j['followup'] ?? '') as String,
        complete: (j['complete'] ?? false) as bool,
      );
}

/// 对话式编辑的预填草稿：/profile/mine 的基础字段 + personaInput 原始输入。
/// Profile 模型只含生成结果（persona），不含这些原始输入，故单独解析原始 map。
class ProfileDraft {
  ProfileDraft({
    required this.realName,
    required this.title,
    required this.strengths,
    required this.recentWork,
    required this.howToKnow,
    required this.style,
    required this.company,
    required this.city,
  });

  final String realName;
  final String title;
  final String strengths;
  final String recentWork;
  final String howToKnow;
  final String style;
  final String company;
  final String city;
}

/// 主页相关接口。
class ProfileApi {
  ProfileApi(this._ref);

  final Ref _ref;

  /// GET /profile/:id —— 访客视图，无需登录。
  Future<Profile> findOne(String id) async {
    final data = await _ref.read(dioClientProvider).get('/profile/$id');
    return Profile.fromJson(Map<String, dynamic>.from(data as Map));
  }

  /// GET /profile/mine —— 我的主页，未创建返回 null。
  Future<Profile?> mine() async {
    final data = await _ref.read(dioClientProvider).get('/profile/mine');
    if (data == null) return null;
    return Profile.fromJson(Map<String, dynamic>.from(data as Map));
  }

  /// GET /profile/mine（原始）—— 取对话式编辑预填所需的原始输入；未创建返回 null。
  Future<ProfileDraft?> mineDraft() async {
    final data = await _ref.read(dioClientProvider).get('/profile/mine');
    if (data == null) return null;
    final m = Map<String, dynamic>.from(data as Map);
    final input = m['personaInput'] is Map
        ? Map<String, dynamic>.from(m['personaInput'] as Map)
        : <String, dynamic>{};
    String s(dynamic v) => (v ?? '').toString();
    return ProfileDraft(
      realName: s(m['realName']),
      title: s(m['title']),
      strengths: s(input['strengths']),
      recentWork: s(input['recentWork']),
      howToKnow: s(input['howToKnow']),
      style: s(input['style']),
      company: s(m['company']),
      city: s(m['city']),
    );
  }

  /// POST /profile —— 创建/更新主页并同步生成 AI 人格（耗时，调用方需显示加载）。
  Future<Profile> createOrUpdate(ProfileInput input) async {
    final data =
        await _ref.read(dioClientProvider).post('/profile', data: input.toJson());
    return Profile.fromJson(Map<String, dynamic>.from(data as Map));
  }

  /// POST /profile/extract —— 对话式创建：从对话消息抽取字段 + 自动判定风格，不写库。
  Future<ExtractedProfile> extract(List<Map<String, String>> messages) async {
    final data = await _ref
        .read(dioClientProvider)
        .post('/profile/extract', data: {'messages': messages});
    return ExtractedProfile.fromJson(Map<String, dynamic>.from(data as Map));
  }

  /// POST /profile/regenerate —— 重新生成 AI 人格。
  Future<Profile> regenerate() async {
    final data = await _ref.read(dioClientProvider).post('/profile/regenerate');
    return Profile.fromJson(Map<String, dynamic>.from(data as Map));
  }

  /// PATCH /profile/contact —— 更新联系方式与可见性。
  Future<void> updateContact({
    String? wechat,
    String? phone,
    bool? visible,
  }) async {
    await _ref.read(dioClientProvider).patch('/profile/contact', data: {
      'contactWechat': ?wechat,
      'contactPhone': ?phone,
      'contactVisible': ?visible,
    });
  }

  /// PATCH /profile/avatar —— 更新形象标识。
  Future<void> updateAvatar(String avatarStyle) async {
    await _ref
        .read(dioClientProvider)
        .patch('/profile/avatar', data: {'avatarStyle': avatarStyle});
  }

  /// PATCH /profile/discoverable —— 切换"公开到人物广场"。
  Future<void> updateDiscoverable(bool discoverable) async {
    await _ref
        .read(dioClientProvider)
        .patch('/profile/discoverable', data: {'discoverable': discoverable});
  }

  /// PATCH /profile/persona —— 人设深度定制：手动微调开场白/语气/音色/一句话。
  Future<void> updatePersona({
    String? oneLiner,
    String? greeting,
    String? tone,
    String? voiceStyle,
  }) async {
    await _ref.read(dioClientProvider).patch('/profile/persona', data: {
      'oneLiner': ?oneLiner,
      'greeting': ?greeting,
      'tone': ?tone,
      'voiceStyle': ?voiceStyle,
    });
  }
}

/// 可选音色（与后端 persona.allowedVoices 对齐，Phase B 映射 TTS 音色）。
const kVoiceStyles = [
  '温暖男声',
  '沉稳男声',
  '清朗男声',
  '温暖女声',
  '知性女声',
  '活泼女声',
];

final profileApiProvider = Provider<ProfileApi>((ref) => ProfileApi(ref));

/// 我的主页（响应式）。创建/编辑后 invalidate 刷新。
final myProfileProvider = FutureProvider<Profile?>((ref) {
  return ref.read(profileApiProvider).mine();
});
