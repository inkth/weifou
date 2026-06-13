import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'network/dio_client.dart';
import 'storage/secure_store.dart';

/// token 安全存储单例。
final secureStoreProvider = Provider<SecureStore>((ref) => SecureStore());

/// 全局 Dio 客户端。401 时清空 auth 状态，触发路由跳登录。
final dioClientProvider = Provider<DioClient>((ref) {
  final store = ref.watch(secureStoreProvider);
  return DioClient(
    store,
    onUnauthorized: () {
      // 由 auth_controller 监听并跳转；这里仅作占位，M1 接入。
      ref.read(unauthorizedSignalProvider.notifier).state++;
    },
  );
});

/// 401 计数信号，路由层可监听以触发重定向。
final unauthorizedSignalProvider = StateProvider<int>((ref) => 0);
