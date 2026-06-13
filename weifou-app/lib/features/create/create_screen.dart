import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_exception.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/profile_api.dart';

/// 创建/编辑 AI 主页。提交后同步生成 AI 人格（耗时），显示生成中动画。
class CreateScreen extends ConsumerStatefulWidget {
  const CreateScreen({super.key});

  @override
  ConsumerState<CreateScreen> createState() => _CreateScreenState();
}

class _CreateScreenState extends ConsumerState<CreateScreen> {
  final _formKey = GlobalKey<FormState>();
  final _realName = TextEditingController();
  final _title = TextEditingController();
  final _company = TextEditingController();
  final _city = TextEditingController();
  final _strengths = TextEditingController();
  final _recentWork = TextEditingController();
  final _howToKnow = TextEditingController();

  bool _generating = false;

  @override
  void dispose() {
    for (final c in [
      _realName,
      _title,
      _company,
      _city,
      _strengths,
      _recentWork,
      _howToKnow,
    ]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _generating = true);
    try {
      final profile = await ref.read(profileApiProvider).createOrUpdate(
            ProfileInput(
              realName: _realName.text.trim(),
              title: _title.text.trim(),
              company: _emptyToNull(_company.text),
              city: _emptyToNull(_city.text),
              strengths: _strengths.text.trim(),
              recentWork: _recentWork.text.trim(),
              howToKnow: _howToKnow.text.trim(),
            ),
          );
      ref.invalidate(myProfileProvider);
      if (mounted) {
        context.goNamed('profile', pathParameters: {'id': profile.id});
      }
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _generating = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.message)));
      }
    } catch (e) {
      if (mounted) {
        setState(() => _generating = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('生成失败：$e')));
      }
    }
  }

  String? _emptyToNull(String s) => s.trim().isEmpty ? null : s.trim();

  @override
  Widget build(BuildContext context) {
    if (_generating) return const _GeneratingView();

    return Scaffold(
      appBar: AppBar(title: const Text('创建 AI 主页')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            _field(_realName, '姓名', required: true),
            _field(_title, '职位 / 身份', required: true),
            _field(_company, '公司 / 机构'),
            _field(_city, '城市'),
            _field(_strengths, '专业 / 擅长', required: true, maxLines: 3),
            _field(_recentWork, '近期作品 / 项目', required: true, maxLines: 3),
            _field(_howToKnow, '希望别人怎么认识你', required: true, maxLines: 3),
            const SizedBox(height: 24),
            ElevatedButton(
              onPressed: _submit,
              child: const Text('生成我的 AI 主页'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _field(
    TextEditingController c,
    String label, {
    bool required = false,
    int maxLines = 1,
  }) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: TextFormField(
        controller: c,
        maxLines: maxLines,
        decoration: InputDecoration(
          labelText: required ? '$label *' : label,
          border: const OutlineInputBorder(),
          alignLabelWithHint: true,
        ),
        validator: required
            ? (v) => (v == null || v.trim().isEmpty) ? '请填写$label' : null
            : null,
      ),
    );
  }
}

/// 生成中：简单的加载动画（lottie 资源到位后可替换）。
class _GeneratingView extends StatelessWidget {
  const _GeneratingView();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            CircularProgressIndicator(),
            SizedBox(height: 24),
            Text('AI 正在生成你的主页…',
                style: TextStyle(fontSize: 16, color: AppTheme.ink)),
            SizedBox(height: 8),
            Text('约需十几秒，请稍候',
                style: TextStyle(fontSize: 13, color: AppTheme.sub)),
          ],
        ),
      ),
    );
  }
}
