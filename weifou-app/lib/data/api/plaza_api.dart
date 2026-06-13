import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/providers.dart';

/// 广场卡片，对应 GET /plaza 返回项。
class PlazaCard {
  PlazaCard({
    required this.profileId,
    required this.realName,
    this.title,
    this.nickname,
    this.avatarUrl,
    this.oneLiner,
    this.tags = const [],
  });

  final String profileId;
  final String realName;
  final String? title;
  final String? nickname;
  final String? avatarUrl;
  final String? oneLiner;
  final List<String> tags;

  factory PlazaCard.fromJson(Map<String, dynamic> json) => PlazaCard(
        profileId: json['profileId'] as String,
        realName: json['realName'] as String? ?? '',
        title: json['title'] as String?,
        nickname: json['nickname'] as String?,
        avatarUrl: json['avatarUrl'] as String?,
        oneLiner: json['oneLiner'] as String?,
        tags: (json['tags'] as List?)?.cast<String>() ?? const [],
      );
}

class PlazaApi {
  PlazaApi(this._ref);
  final Ref _ref;

  /// GET /plaza?sort=hot|new&q=&page= —— 无需登录。
  Future<List<PlazaCard>> list({
    String sort = 'new',
    String? q,
    int page = 1,
  }) async {
    final data = await _ref.read(dioClientProvider).get('/plaza', query: {
      'sort': sort,
      'page': page,
      if (q != null && q.isNotEmpty) 'q': q,
    });
    final map = Map<String, dynamic>.from(data as Map);
    final items = (map['items'] as List?) ?? const [];
    return items
        .map((e) => PlazaCard.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList();
  }
}

final plazaApiProvider = Provider<PlazaApi>((ref) => PlazaApi(ref));
