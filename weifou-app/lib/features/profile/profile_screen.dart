import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/app_theme.dart';
import '../../data/api/profile_api.dart';
import '../../data/models/profile.dart';

/// 按 id 拉取主页（访客视图）。
final profileProvider =
    FutureProvider.family<Profile, String>((ref, id) async {
  return ref.read(profileApiProvider).findOne(id);
});

/// 主页展示页：访客在此浏览并进入与 AI 的对话。
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key, required this.profileId});

  final String profileId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(profileProvider(profileId));
    return Scaffold(
      appBar: AppBar(title: const Text('主页')),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text('加载失败：$e', textAlign: TextAlign.center),
          ),
        ),
        data: (p) => _ProfileBody(profile: p),
      ),
      bottomNavigationBar: async.maybeWhen(
        data: (p) => SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: ElevatedButton.icon(
              icon: const Icon(Icons.chat_bubble_outline),
              label: Text('和 ${p.realName} 的 AI 聊聊'),
              onPressed: () => context.pushNamed(
                'chat',
                pathParameters: {'profileId': p.id},
                extra: p,
              ),
            ),
          ),
        ),
        orElse: () => null,
      ),
    );
  }
}

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({required this.profile});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final persona = profile.persona;
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        Text(
          profile.realName,
          style: const TextStyle(fontSize: 26, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 6),
        Text(
          [profile.title, profile.company, profile.city]
              .where((e) => e != null && e.isNotEmpty)
              .join(' · '),
          style: const TextStyle(color: AppTheme.sub, fontSize: 14),
        ),
        if (persona?.oneLiner != null) ...[
          const SizedBox(height: 20),
          Text(persona!.oneLiner!,
              style: const TextStyle(fontSize: 17, height: 1.5)),
        ],
        if (persona?.tags.isNotEmpty ?? false) ...[
          const SizedBox(height: 16),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final t in persona!.tags)
                Chip(
                  label: Text(t),
                  backgroundColor: Colors.white,
                  side: const BorderSide(color: AppTheme.border),
                ),
            ],
          ),
        ],
        if (persona?.fullIntro != null) ...[
          const SizedBox(height: 24),
          const Text('关于',
              style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          Text(persona!.fullIntro!,
              style: const TextStyle(height: 1.6, color: AppTheme.ink)),
        ],
        const SizedBox(height: 80),
      ],
    );
  }
}
