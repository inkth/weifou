import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/app_theme.dart';
import '../../data/api/chat_api.dart';

final _sessionsProvider = FutureProvider<List<ChatSessionItem>>((ref) {
  return ref.read(chatApiProvider).mySessions();
});

/// 对话 Tab：我聊过的 AI 分身（最近会话）。
class ChatsScreen extends ConsumerWidget {
  const ChatsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(_sessionsProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('对话')),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(_sessionsProvider),
        child: async.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => ListView(children: [
            const SizedBox(height: 120),
            Center(child: Text('加载失败：$e')),
          ]),
          data: (items) => items.isEmpty
              ? ListView(children: const [
                  SizedBox(height: 120),
                  Center(
                    child: Text('还没有对话，去广场找人聊聊吧',
                        style: TextStyle(color: AppTheme.sub)),
                  ),
                ])
              : ListView.separated(
                  itemCount: items.length,
                  separatorBuilder: (_, _) =>
                      const Divider(height: 1, indent: 76),
                  itemBuilder: (_, i) {
                    final s = items[i];
                    return ListTile(
                      contentPadding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 6),
                      leading: CircleAvatar(
                        radius: 24,
                        backgroundColor: AppTheme.bg,
                        backgroundImage: (s.avatarUrl?.isNotEmpty ?? false)
                            ? NetworkImage(s.avatarUrl!)
                            : null,
                        child: (s.avatarUrl?.isEmpty ?? true)
                            ? Text(s.realName.isNotEmpty ? s.realName[0] : '?')
                            : null,
                      ),
                      title: Text(s.realName,
                          style:
                              const TextStyle(fontWeight: FontWeight.w600)),
                      subtitle: Text(
                        s.lastMessage ?? '',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(color: AppTheme.sub),
                      ),
                      onTap: () => context.pushNamed(
                        'chat',
                        pathParameters: {'profileId': s.profileId},
                      ),
                    );
                  },
                ),
        ),
      ),
    );
  }
}
