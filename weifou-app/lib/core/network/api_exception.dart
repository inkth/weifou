/// 统一的业务异常模型，对应小程序 request.js reject 的 `{code, message}`。
class ApiException implements Exception {
  ApiException(this.code, this.message);

  final String code;
  final String message;

  /// 401：token 失效，需重新登录。
  bool get isUnauthorized => code == 'UNAUTHORIZED';

  @override
  String toString() => 'ApiException($code): $message';
}
