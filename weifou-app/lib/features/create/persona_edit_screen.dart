import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/theme/app_theme.dart';
import '../../data/api/profile_api.dart';
import '../../data/models/profile.dart';

/// 人设深度定制：手动微调 AI 的开场白 / 语气性格 / 音色 / 一句话介绍。
class PersonaEditScreen extends ConsumerWidget {
  const PersonaEditScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(myProfileProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('编辑人设')),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text('加载失败：$e')),
        data: (p) => (p == null || p.persona == null)
            ? const Center(child: Text('请先创建 AI 主页'))
            : _Form(profile: p),
      ),
    );
  }
}

class _Form extends ConsumerStatefulWidget {
  const _Form({required this.profile});
  final Profile profile;

  @override
  ConsumerState<_Form> createState() => _FormState();
}

class _FormState extends ConsumerState<_Form> {
  late final _oneLiner =
      TextEditingController(text: widget.profile.persona?.oneLiner ?? '');
  late final _greeting =
      TextEditingController(text: widget.profile.persona?.greeting ?? '');
  late final _tone =
      TextEditingController(text: widget.profile.persona?.tone ?? '');
  late String _voice = _initialVoice;
  bool _saving = false;

  String get _initialVoice {
    final v = widget.profile.persona?.voiceStyle;
    return (v != null && kVoiceStyles.contains(v)) ? v : kVoiceStyles.first;
  }

  @override
  void dispose() {
    _oneLiner.dispose();
    _greeting.dispose();
    _tone.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      await ref.read(profileApiProvider).updatePersona(
            oneLiner: _oneLiner.text.trim(),
            greeting: _greeting.text.trim(),
            tone: _tone.text.trim(),
            voiceStyle: _voice,
          );
      ref.invalidate(myProfileProvider);
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('已保存')));
        Navigator.of(context).pop();
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('保存失败：$e')));
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
        _label('一句话介绍'),
        TextField(
          controller: _oneLiner,
          maxLines: 2,
          decoration: const InputDecoration(border: OutlineInputBorder()),
        ),
        const SizedBox(height: 20),
        _label('开场白（进入对话的第一句）'),
        TextField(
          controller: _greeting,
          maxLines: 3,
          decoration: const InputDecoration(border: OutlineInputBorder()),
        ),
        const SizedBox(height: 20),
        _label('语气与性格'),
        TextField(
          controller: _tone,
          maxLines: 3,
          decoration: const InputDecoration(
            border: OutlineInputBorder(),
            hintText: '例如：亲切、爱用比喻、回答先共情再给建议',
          ),
        ),
        const SizedBox(height: 20),
        _label('音色'),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final v in kVoiceStyles)
              ChoiceChip(
                label: Text(v),
                selected: _voice == v,
                onSelected: (_) => setState(() => _voice = v),
              ),
          ],
        ),
        const SizedBox(height: 32),
        ElevatedButton(
          onPressed: _saving ? null : _save,
          child: Text(_saving ? '保存中…' : '保存'),
        ),
      ],
    );
  }

  Widget _label(String t) => Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Text(t,
            style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: AppTheme.ink)),
      );
}
