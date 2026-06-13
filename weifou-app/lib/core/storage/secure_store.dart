import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// token 安全存储，替代小程序的 wx.getStorageSync('weifou_token')。
///
/// iOS 走 Keychain、Android 走 EncryptedSharedPreferences，比明文存储更安全。
class SecureStore {
  SecureStore({FlutterSecureStorage? storage})
      : _storage = storage ??
            const FlutterSecureStorage(
              aOptions: AndroidOptions(encryptedSharedPreferences: true),
            );

  final FlutterSecureStorage _storage;

  static const _kToken = 'weifou_token';

  Future<String?> getToken() => _storage.read(key: _kToken);

  Future<void> setToken(String token) =>
      _storage.write(key: _kToken, value: token);

  Future<void> clearToken() => _storage.delete(key: _kToken);
}
