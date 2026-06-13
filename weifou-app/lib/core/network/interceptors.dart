import 'dart:io' show Platform;

import 'package:dio/dio.dart';

import '../storage/secure_store.dart';
import 'api_exception.dart';

/// 注入 `Authorization: Bearer <token>`，对应 request.js 的 header 逻辑。
class AuthInterceptor extends Interceptor {
  AuthInterceptor(this._store);

  final SecureStore _store;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await _store.getToken();
    if (token != null && token.isNotEmpty) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }
}

/// 注入客户端来源标识，供后端做支付分流与 iOS 合规网关。
/// X-Client-Type: app（区别于 miniapp）；X-Platform: ios | android。
class ClientInterceptor extends Interceptor {
  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) {
    options.headers['X-Client-Type'] = 'app';
    options.headers['X-Platform'] = Platform.isIOS ? 'ios' : 'android';
    handler.next(options);
  }
}

/// 复刻 request.js 的响应约定：
/// - 401 → 清 token + 抛 UNAUTHORIZED
/// - 2xx 且 body.success != false → 解包返回 body.data
/// - 其它 → 抛 ApiException(code, message)
/// 网络层异常 → NETWORK_ERROR。
class ResponseInterceptor extends Interceptor {
  ResponseInterceptor(this._store, {this.onUnauthorized});

  final SecureStore _store;

  /// 401 回调，供路由层跳转登录页。
  final void Function()? onUnauthorized;

  @override
  Future<void> onResponse(
    Response response,
    ResponseInterceptorHandler handler,
  ) async {
    final status = response.statusCode ?? 0;
    final body = response.data is Map ? response.data as Map : const {};

    if (status == 401) {
      await _store.clearToken();
      onUnauthorized?.call();
      return handler.reject(
        _wrap(response, ApiException('UNAUTHORIZED', '请重新登录')),
      );
    }

    if (status >= 200 && status < 300 && body['success'] != false) {
      // 解包 data，让上层直接拿到业务数据。
      response.data = body['data'];
      return handler.next(response);
    }

    return handler.reject(
      _wrap(
        response,
        ApiException(
          (body['code'] as String?) ?? 'HTTP_$status',
          (body['message'] as String?) ?? '请求失败',
        ),
      ),
    );
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    // 已是包装过的业务异常，直接透传。
    if (err.error is ApiException) return handler.next(err);

    final status = err.response?.statusCode ?? 0;
    if (status == 401) {
      await _store.clearToken();
      onUnauthorized?.call();
      return handler.next(
        _wrap(err.response, ApiException('UNAUTHORIZED', '请重新登录')),
      );
    }

    final body = err.response?.data is Map ? err.response!.data as Map : null;
    if (body != null) {
      return handler.next(
        _wrap(
          err.response,
          ApiException(
            (body['code'] as String?) ?? 'HTTP_$status',
            (body['message'] as String?) ?? '请求失败',
          ),
        ),
      );
    }

    return handler.next(
      _wrap(err.response, ApiException('NETWORK_ERROR', err.message ?? '网络异常')),
    );
  }

  DioException _wrap(Response? response, ApiException e) => DioException(
        requestOptions: response?.requestOptions ??
            RequestOptions(path: ''),
        response: response,
        error: e,
      );
}
