import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../data/models/profile.dart';
import '../../features/auth/auth_controller.dart';
import '../../features/auth/login_screen.dart';
import '../../features/chat/chat_screen.dart';
import '../../features/chat/chats_screen.dart';
import '../../features/create/conversational_create_screen.dart';
import '../../features/create/persona_edit_screen.dart';
import '../../features/home/home_screen.dart';
import '../../features/plaza/plaza_screen.dart';
import '../../features/poster/poster_screen.dart';
import '../../features/profile/profile_screen.dart';
import '../../features/settings/settings_screen.dart';

/// 全局路由表 + 登录守卫。星野式底部三 Tab：广场 / 对话 / 我的。
final routerProvider = Provider<GoRouter>((ref) {
  final refresh = ValueNotifier<int>(0);
  ref.listen(authControllerProvider, (_, _) => refresh.value++);
  ref.onDispose(refresh.dispose);

  return GoRouter(
    initialLocation: '/plaza',
    refreshListenable: refresh,
    redirect: (context, state) {
      final auth = ref.read(authControllerProvider);
      if (auth is AuthUnknown) return null; // 启动恢复中

      final loc = state.matchedLocation;
      // 访客可访问：登录页、广场、主页只读页。
      final guestAllowed = loc == '/login' ||
          loc == '/plaza' ||
          loc.startsWith('/profile/');
      if (auth is AuthGuest && !guestAllowed) return '/login';
      if (auth is AuthAuthed && loc == '/login') return '/plaza';
      return null;
    },
    routes: [
      // 底部 Tab 外壳
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) =>
            _TabShell(shell: navigationShell),
        branches: [
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/plaza',
              name: 'plaza',
              builder: (context, state) => const PlazaScreen(),
            ),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/chats',
              name: 'chats',
              builder: (context, state) => const ChatsScreen(),
            ),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/me',
              name: 'me',
              builder: (context, state) => const HomeScreen(),
            ),
          ]),
        ],
      ),
      // 全屏详情页（覆盖 Tab 栏）
      GoRoute(
        path: '/login',
        name: 'login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/onboarding',
        name: 'onboarding',
        builder: (context, state) => const ConversationalCreateScreen(),
      ),
      GoRoute(
        path: '/settings',
        name: 'settings',
        builder: (context, state) => const SettingsScreen(),
      ),
      GoRoute(
        path: '/persona-edit',
        name: 'persona-edit',
        builder: (context, state) => const PersonaEditScreen(),
      ),
      GoRoute(
        path: '/profile/:id',
        name: 'profile',
        builder: (context, state) =>
            ProfileScreen(profileId: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/chat/:profileId',
        name: 'chat',
        builder: (context, state) => ChatScreen(
          profileId: state.pathParameters['profileId']!,
          profile: state.extra is Profile ? state.extra as Profile : null,
        ),
      ),
      GoRoute(
        path: '/poster/:profileId',
        name: 'poster',
        builder: (context, state) =>
            PosterScreen(profileId: state.pathParameters['profileId']!),
      ),
    ],
  );
});

/// 底部导航外壳。
class _TabShell extends StatelessWidget {
  const _TabShell({required this.shell});
  final StatefulNavigationShell shell;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: shell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: shell.currentIndex,
        onDestinationSelected: (i) =>
            shell.goBranch(i, initialLocation: i == shell.currentIndex),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.explore_outlined),
            selectedIcon: Icon(Icons.explore),
            label: '广场',
          ),
          NavigationDestination(
            icon: Icon(Icons.chat_bubble_outline),
            selectedIcon: Icon(Icons.chat_bubble),
            label: '对话',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person),
            label: '我的',
          ),
        ],
      ),
    );
  }
}
