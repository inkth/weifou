import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/config/env.dart';
import '../../core/network/api_exception.dart';
import '../../core/theme/app_theme.dart';
import 'auth_controller.dart';

/// 登录页。微信一键授权登录（fluwx）。
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  bool _loading = false;
  final _phoneCtrl = TextEditingController();
  final _codeCtrl = TextEditingController();

  @override
  void dispose() {
    _phoneCtrl.dispose();
    _codeCtrl.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    setState(() => _loading = true);
    try {
      await ref.read(authControllerProvider.notifier).loginWithWeChat();
      // 登录成功后由路由 redirect 自动离开登录页。
    } on ApiException catch (e) {
      if (mounted) _toast(e.message);
    } catch (e) {
      if (mounted) _toast('登录失败：$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _testLogin() async {
    final phone = _phoneCtrl.text.trim();
    final code = _codeCtrl.text.trim();
    if (phone.isEmpty) {
      _toast('请输入手机号');
      return;
    }
    setState(() => _loading = true);
    try {
      await ref
          .read(authControllerProvider.notifier)
          .loginWithTestCode(phone, code);
      // 登录成功后由路由 redirect 自动离开登录页。
    } on ApiException catch (e) {
      if (mounted) _toast(e.message);
    } catch (e) {
      if (mounted) _toast('登录失败：$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _toast(String msg) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(msg)));
  }

  /// 测试验证码登录区（手机号 + 654321），仅联调阶段显示。
  List<Widget> _buildTestLogin() {
    return [
      const SizedBox(height: 32),
      const Row(children: [
        Expanded(child: Divider()),
        Padding(
          padding: EdgeInsets.symmetric(horizontal: 12),
          child: Text('测试登录', style: TextStyle(color: AppTheme.sub, fontSize: 12)),
        ),
        Expanded(child: Divider()),
      ]),
      const SizedBox(height: 16),
      TextField(
        controller: _phoneCtrl,
        keyboardType: TextInputType.phone,
        decoration: const InputDecoration(
          labelText: '手机号',
          hintText: '任意手机号',
          border: OutlineInputBorder(),
        ),
      ),
      const SizedBox(height: 12),
      TextField(
        controller: _codeCtrl,
        keyboardType: TextInputType.number,
        decoration: const InputDecoration(
          labelText: '验证码',
          hintText: '654321',
          border: OutlineInputBorder(),
        ),
      ),
      const SizedBox(height: 12),
      OutlinedButton(
        onPressed: _loading ? null : _testLogin,
        child: const Text('验证码登录'),
      ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Spacer(),
              const Text(
                '微否',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 36, fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 10),
              const Text(
                '每个人的 AI 助理',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 8),
              const Text(
                '加微信前，先和我的 AI 聊聊',
                textAlign: TextAlign.center,
                style: TextStyle(color: AppTheme.sub, fontSize: 14),
              ),
              const Spacer(),
              ElevatedButton.icon(
                onPressed: _loading ? null : _login,
                icon: _loading
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Icon(Icons.wechat),
                label: Text(_loading ? '登录中…' : '微信登录'),
              ),
              if (Env.enableTestLogin) ..._buildTestLogin(),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ),
    );
  }
}
