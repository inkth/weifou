import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/theme/app_theme.dart';
import '../../data/api/profile_api.dart';
import '../../data/models/profile.dart';

/// 设置页（首版）：联系方式与可见性、形象标识。
class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(myProfileProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('设置')),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text('加载失败：$e')),
        data: (p) => p == null
            ? const Center(child: Text('请先创建 AI 主页'))
            : _SettingsForm(profile: p),
      ),
    );
  }
}

class _SettingsForm extends ConsumerStatefulWidget {
  const _SettingsForm({required this.profile});
  final Profile profile;

  @override
  ConsumerState<_SettingsForm> createState() => _SettingsFormState();
}

class _SettingsFormState extends ConsumerState<_SettingsForm> {
  late final TextEditingController _wechat = TextEditingController(
    text: widget.profile.contactWechat ?? '',
  );
  late final TextEditingController _phone = TextEditingController(
    text: widget.profile.contactPhone ?? '',
  );
  late bool _visible = widget.profile.contactVisible;
  late bool _discoverable = widget.profile.discoverable;
  bool _saving = false;

  @override
  void dispose() {
    _wechat.dispose();
    _phone.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      await ref
          .read(profileApiProvider)
          .updateContact(
            wechat: _wechat.text.trim(),
            phone: _phone.text.trim(),
            visible: _visible,
          );
      ref.invalidate(myProfileProvider);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('已保存')));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('保存失败：$e')));
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        const Text(
          '联系方式',
          style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _wechat,
          decoration: const InputDecoration(
            labelText: '微信号',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _phone,
          keyboardType: TextInputType.phone,
          decoration: const InputDecoration(
            labelText: '手机号',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 8),
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('对访客公开联系方式'),
          subtitle: const Text(
            '关闭后访客看不到你的微信/手机',
            style: TextStyle(color: AppTheme.sub, fontSize: 12),
          ),
          value: _visible,
          onChanged: (v) => setState(() => _visible = v),
        ),
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: _saving ? null : _save,
          child: Text(_saving ? '保存中…' : '保存'),
        ),
        const Divider(height: 48),
        const Text(
          '人物广场',
          style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
        ),
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('公开到人物广场'),
          subtitle: const Text(
            '开启后他人可在广场发现你的 AI 分身；关闭则仅靠分享链接访问',
            style: TextStyle(color: AppTheme.sub, fontSize: 12),
          ),
          value: _discoverable,
          onChanged: (v) async {
            final messenger = ScaffoldMessenger.of(context);
            setState(() => _discoverable = v);
            try {
              await ref.read(profileApiProvider).updateDiscoverable(v);
              ref.invalidate(myProfileProvider);
            } catch (e) {
              if (mounted) setState(() => _discoverable = !v);
              messenger.showSnackBar(SnackBar(content: Text('设置失败：$e')));
            }
          },
        ),
      ],
    );
  }
}
