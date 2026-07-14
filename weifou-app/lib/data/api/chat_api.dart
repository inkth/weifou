import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/providers.dart';

/// AI 问答返回。
class ChatAnswer {
  ChatAnswer({required this.sessionId, required this.answer});

  final String sessionId;
  final String answer;
}

/// 最近对话项，对应 GET /chat/sessions/mine。
class ChatSessionItem {
  ChatSessionItem({
    required this.sessionId,
    required this.profileId,
    required this.realName,
    this.avatarUrl,
    this.lastMessage,
  });

  final String sessionId;
  final String profileId;
  final String realName;
  final String? avatarUrl;
  final String? lastMessage;

  factory ChatSessionItem.fromJson(Map<String, dynamic> json) =>
      ChatSessionItem(
        sessionId: json['sessionId'] as String,
        profileId: json['profileId'] as String,
        realName: json['realName'] as String? ?? '',
        avatarUrl: json['avatarUrl'] as String?,
        lastMessage: json['lastMessage'] as String?,
      );
}

class ChatApi {
  ChatApi(this._ref);

  final Ref _ref;

  /// POST /chat/:profileId/ask —— 需登录。提问限 200 字。
  Future<ChatAnswer> ask(String profileId, String content) async {
    final data = await _ref
        .read(dioClientProvider)
        .post('/chat/$profileId/ask', data: {'content': content});
    final map = Map<String, dynamic>.from(data as Map);
    return ChatAnswer(
      sessionId: map['sessionId'] as String,
      answer: map['answer'] as String,
    );
  }

  /// GET /chat/sessions/mine —— 我作为访客的最近对话（需登录）。
  Future<List<ChatSessionItem>> mySessions() async {
    final data = await _ref.read(dioClientProvider).get('/chat/sessions/mine');
    final list = (data as List?) ?? const [];
    return list
        .map(
          (e) => ChatSessionItem.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList();
  }
}

final chatApiProvider = Provider<ChatApi>((ref) => ChatApi(ref));
