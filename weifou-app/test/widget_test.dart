import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:weifou_app/app.dart';
import 'package:weifou_app/core/network/dio_client.dart';
import 'package:weifou_app/core/providers.dart';
import 'package:weifou_app/core/storage/secure_store.dart';

/// 不打真实网络的 DioClient，避免测试中残留 dio 超时 timer。
class _FakeDio extends DioClient {
  _FakeDio() : super(SecureStore());

  @override
  Future<dynamic> get(String path, {Map<String, dynamic>? query}) async {
    if (path == '/plaza') return {'items': [], 'page': 1, 'pageSize': 20};
    return null;
  }

  @override
  Future<dynamic> post(String path, {Object? data}) async => null;
  @override
  Future<dynamic> patch(String path, {Object? data}) async => null;
  @override
  Future<dynamic> delete(String path, {Object? data}) async => null;
}

void main() {
  testWidgets('App 启动渲染（广场 Tab）', (WidgetTester tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [dioClientProvider.overrideWithValue(_FakeDio())],
        child: const WeifouApp(),
      ),
    );
    await tester.pump();
    await tester.pump();

    expect(find.byType(MaterialApp), findsOneWidget);
    expect(find.text('广场'), findsWidgets);
  });
}
