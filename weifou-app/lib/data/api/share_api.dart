import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/config/env.dart';
import '../../core/providers.dart';

/// 海报数据，对应 GET /share/bundle/:profileId。
/// 注意：后端 wxacodeBase64 是「小程序码」，App 不用；改用本地生成指向
/// 落地页的二维码（见 shareUrl）。
class ShareBundle {
  ShareBundle({
    required this.profileId,
    this.nickname,
    this.realName,
    this.avatarUrl,
    this.oneLiner,
    this.tags = const [],
  });

  final String profileId;
  final String? nickname;
  final String? realName;
  final String? avatarUrl;
  final String? oneLiner;
  final List<String> tags;

  /// 分享落地页：已装唤起 App，未装落 H5。
  String get shareUrl => '${Env.shareWebBase}/$profileId';

  factory ShareBundle.fromJson(Map<String, dynamic> json) => ShareBundle(
        profileId: json['profileId'] as String,
        nickname: json['nickname'] as String?,
        realName: json['realName'] as String?,
        avatarUrl: json['avatarUrl'] as String?,
        oneLiner: json['oneLiner'] as String?,
        tags: (json['tags'] as List?)?.cast<String>() ?? const [],
      );
}

class ShareApi {
  ShareApi(this._ref);

  final Ref _ref;

  Future<ShareBundle> bundle(String profileId) async {
    final data =
        await _ref.read(dioClientProvider).get('/share/bundle/$profileId');
    return ShareBundle.fromJson(Map<String, dynamic>.from(data as Map));
  }
}

final shareApiProvider = Provider<ShareApi>((ref) => ShareApi(ref));
