import 'package:dio/dio.dart';

import '../config/env.dart';
import '../storage/secure_store.dart';
import 'api_exception.dart';
import 'interceptors.dart';

/// 封装 Dio，集中拦截器与错误归一。API 层通过 get/post/patch/delete
/// 直接拿到「已解包的 data」，与小程序 request() 返回 body.data 一致。
class DioClient {
  DioClient(this._store, {void Function()? onUnauthorized})
      : _dio = Dio(
          BaseOptions(
            baseUrl: Env.apiBase,
            connectTimeout: const Duration(milliseconds: Env.httpTimeoutMs),
            receiveTimeout: const Duration(milliseconds: Env.httpTimeoutMs),
            sendTimeout: const Duration(milliseconds: Env.httpTimeoutMs),
            contentType: 'application/json',
            // 让所有状态码都进入拦截器，由 ResponseInterceptor 统一判定。
            validateStatus: (_) => true,
          ),
        ) {
    _dio.interceptors.addAll([
      AuthInterceptor(_store),
      ClientInterceptor(),
      ResponseInterceptor(_store, onUnauthorized: onUnauthorized),
    ]);
  }

  final Dio _dio;
  final SecureStore _store;

  Future<dynamic> get(String path, {Map<String, dynamic>? query}) =>
      _unwrap(() => _dio.get(path, queryParameters: query));

  Future<dynamic> post(String path, {Object? data}) =>
      _unwrap(() => _dio.post(path, data: data));

  Future<dynamic> patch(String path, {Object? data}) =>
      _unwrap(() => _dio.patch(path, data: data));

  Future<dynamic> delete(String path, {Object? data}) =>
      _unwrap(() => _dio.delete(path, data: data));

  /// 把拦截器抛出的 DioException 还原成 ApiException 给上层。
  Future<dynamic> _unwrap(Future<Response> Function() call) async {
    try {
      final res = await call();
      return res.data;
    } on DioException catch (e) {
      if (e.error is ApiException) throw e.error as ApiException;
      throw ApiException('NETWORK_ERROR', e.message ?? '网络异常');
    }
  }
}
