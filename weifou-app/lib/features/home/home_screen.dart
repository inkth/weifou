import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/avatar/weifou_avatar.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/profile_api.dart';
import '../../data/models/profile.dart';
import '../auth/auth_controller.dart';

/// 首页：未登录显示加载/跳登录；已登录按是否有主页展示创建或管理入口。
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authControllerProvider);
    if (auth is AuthUnknown) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    final user = auth is AuthAuthed ? auth.user : null;

    return Scaffold(
      appBar: AppBar(
        title: const Text('微否'),
        actions: [
          if (user != null)
            IconButton(
              tooltip: '退出登录',
              icon: const Icon(Icons.logout),
              onPressed: () =>
                  ref.read(authControllerProvider.notifier).logout(),
            ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(myProfileProvider),
        child: ref.watch(myProfileProvider).when(
              loading: () =>
                  const Center(child: CircularProgressIndicator()),
              error: (e, _) => ListView(
                children: [
                  const SizedBox(height: 120),
                  Center(child: Text('加载失败：$e')),
                ],
              ),
              data: (p) => p == null
                  ? const _NoProfile()
                  : _HasProfile(profile: p),
            ),
      ),
    );
  }
}

class _NoProfile extends StatelessWidget {
  const _NoProfile();

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        const SizedBox(height: 60),
        const Text('加微信前，先和我的 AI 聊聊',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 22, fontWeight: FontWeight.w700)),
        const SizedBox(height: 12),
        const Text('创建你的 AI 主页，让别人先认识你',
            textAlign: TextAlign.center,
            style: TextStyle(color: AppTheme.sub)),
        const SizedBox(height: 40),
        ElevatedButton(
          style: AppTheme.accentButton,
          onPressed: () => context.pushNamed('onboarding'),
          child: const Text('和 AI 聊几句，生成我的主页'),
        ),
      ],
    );
  }
}

class _HasProfile extends StatelessWidget {
  const _HasProfile({required this.profile});
  final Profile profile;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: const BorderSide(color: AppTheme.border),
          ),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    WeifouAvatar(
                      style: profile.avatarStyle,
                      name: profile.realName,
                      size: 56,
                    ),
                    const SizedBox(width: 14),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(profile.realName,
                              style: const TextStyle(
                                  fontSize: 20, fontWeight: FontWeight.w700)),
                          const SizedBox(height: 4),
                          Text(profile.title,
                              style: const TextStyle(color: AppTheme.sub)),
                        ],
                      ),
                    ),
                  ],
                ),
                if (profile.persona?.oneLiner != null) ...[
                  const SizedBox(height: 14),
                  Text(profile.persona!.oneLiner!,
                      style: const TextStyle(height: 1.5)),
                ],
              ],
            ),
          ),
        ),
        const SizedBox(height: 20),
        ElevatedButton.icon(
          icon: const Icon(Icons.visibility_outlined),
          label: const Text('查看主页'),
          onPressed: () =>
              context.pushNamed('profile', pathParameters: {'id': profile.id}),
        ),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          icon: const Icon(Icons.ios_share),
          label: const Text('分享海报'),
          onPressed: () => context
              .pushNamed('poster', pathParameters: {'profileId': profile.id}),
        ),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          icon: const Icon(Icons.settings_outlined),
          label: const Text('设置'),
          onPressed: () => context.pushNamed('settings'),
        ),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          icon: const Icon(Icons.auto_awesome_outlined),
          label: const Text('编辑人设（开场白·语气·音色）'),
          onPressed: () => context.pushNamed('persona-edit'),
        ),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          icon: const Icon(Icons.edit_outlined),
          label: const Text('重填资料·重新生成'),
          onPressed: () => context.pushNamed('create'),
        ),
      ],
    );
  }
}
