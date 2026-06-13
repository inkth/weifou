import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;

import '../../core/avatar/weifou_avatar.dart';
import '../../core/network/api_exception.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/profile_api.dart';

/// 对话式创建：用户自由回答，服务端 /profile/extract 抽字段 + 按语气定风格；
/// 缺必填则追问一句，齐了即可上岗。最终走现有 createOrUpdate。与小程序 pages/onboarding 对齐。
const _opener =
    '嗨，我是来帮你建主页的 AI 助理～ 先用一两句话介绍下你自己就好：你是谁、平时做什么、最能帮别人解决什么问题？想到哪说到哪。';

const Map<String, String> _styleAvatar = {
  'steady': 'toon-steady',
  'warm': 'toon-warm',
  'sharp': 'toon-sharp',
  'humorous': 'toon-humorous',
};

class _OMsg {
  _OMsg(this.fromAi, this.text);
  final bool fromAi;
  final String text;
}

class ConversationalCreateScreen extends ConsumerStatefulWidget {
  const ConversationalCreateScreen({super.key});

  @override
  ConsumerState<ConversationalCreateScreen> createState() =>
      _ConversationalCreateScreenState();
}

class _ConversationalCreateScreenState
    extends ConsumerState<ConversationalCreateScreen> {
  final _input = TextEditingController();
  final _scroll = ScrollController();
  final List<_OMsg> _msgs = [];

  bool _thinking = false;
  bool _submitting = false;
  bool _canFinish = false;
  bool _confirmed = false;
  bool _recording = false;
  String _avatarStyle = 'toon-warm';
  AvatarState _avatarState = AvatarState.speaking;

  final _speech = stt.SpeechToText();
  bool _speechReady = false;

  // 累积抽取出的字段
  String _realName = '', _title = '', _strengths = '';
  String _recentWork = '', _howToKnow = '', _style = '';

  @override
  void initState() {
    super.initState();
    _msgs.add(_OMsg(true, _opener));
  }

  @override
  void dispose() {
    if (_recording) _speech.cancel();
    _input.dispose();
    _scroll.dispose();
    super.dispose();
  }

  // —— 语音输入（speech_to_text，iOS Speech / Android SpeechRecognizer）——
  Future<bool> _ensureSpeech() async {
    if (_speechReady) return true;
    _speechReady = await _speech.initialize(
      onStatus: (s) {
        if ((s == 'done' || s == 'notListening') && mounted && _recording) {
          _micStop();
        }
      },
      onError: (_) {
        if (mounted) setState(() => _recording = false);
      },
    );
    return _speechReady;
  }

  Future<void> _micStart() async {
    if (_thinking || _submitting || _recording) return;
    final ok = await _ensureSpeech();
    if (!ok) {
      _snack('语音暂不可用，请打字');
      return;
    }
    setState(() => _recording = true);
    await _speech.listen(
      onResult: (r) => setState(() {
        _input.text = r.recognizedWords;
        _input.selection =
            TextSelection.collapsed(offset: _input.text.length);
      }),
      listenOptions: stt.SpeechListenOptions(
        localeId: 'zh_CN',
        partialResults: true,
        cancelOnError: true,
      ),
    );
  }

  Future<void> _micStop() async {
    if (!_recording) return;
    setState(() => _recording = false);
    await _speech.stop();
    if (_input.text.trim().isNotEmpty) _send();
  }

  void _scrollEnd() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(_scroll.position.maxScrollExtent,
            duration: const Duration(milliseconds: 200), curve: Curves.easeOut);
      }
    });
  }

  String _fallbackFollowup() {
    if (_realName.isEmpty) return '我该怎么称呼你呢？';
    if (_title.isEmpty) return '你平时主要是做什么的？';
    if (_strengths.isEmpty) return '你最能帮别人解决什么问题？';
    return '还想补充点什么吗？';
  }

  Future<void> _send() async {
    final v = _input.text.trim();
    if (v.isEmpty || _thinking || _submitting) return;
    setState(() {
      _msgs.add(_OMsg(false, v));
      _input.clear();
      _thinking = true;
      _avatarState = AvatarState.thinking;
    });
    _scrollEnd();
    await _extract();
  }

  Future<void> _extract() async {
    final messages = _msgs
        .map((m) => {'role': m.fromAi ? 'ai' : 'me', 'text': m.text})
        .toList();
    try {
      final res = await ref.read(profileApiProvider).extract(messages);
      if (!mounted) return;
      _applyExtract(res);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _thinking = false;
        _avatarState = AvatarState.speaking;
        _msgs.add(_OMsg(true, '（网络好像有点慢，刚那句再说一遍试试？）'));
      });
      _scrollEnd();
    }
  }

  void _applyExtract(ExtractedProfile res) {
    if (res.realName.isNotEmpty) _realName = res.realName;
    if (res.title.isNotEmpty) _title = res.title;
    if (res.strengths.isNotEmpty) _strengths = res.strengths;
    if (res.recentWork.isNotEmpty) _recentWork = res.recentWork;
    if (res.howToKnow.isNotEmpty) _howToKnow = res.howToKnow;
    if (res.style.isNotEmpty) _style = res.style;

    final complete =
        _realName.isNotEmpty && _title.isNotEmpty && _strengths.isNotEmpty;
    setState(() {
      _thinking = false;
      _canFinish = complete;
      _avatarStyle = _styleAvatar[_style] ?? 'toon-warm'; // 按语气实时变脸
      _avatarState = AvatarState.speaking;
      if (complete) {
        if (!_confirmed) {
          _confirmed = true;
          _msgs.add(_OMsg(true, '好，我大概了解你了～ 随时可以点「先上岗」，也能再补两句让我更懂你。'));
        } else {
          _msgs.add(_OMsg(true, '记下了～ 还想补充就继续说，或者点「先上岗」。'));
        }
      } else {
        final ask =
            res.followup.trim().isNotEmpty ? res.followup.trim() : _fallbackFollowup();
        _msgs.add(_OMsg(true, ask));
      }
    });
    _scrollEnd();
  }

  Future<void> _finish() async {
    if (_realName.isEmpty || _title.isEmpty || _strengths.isEmpty) {
      _snack('还差一点点信息哦');
      return;
    }
    if (_submitting) return;
    setState(() {
      _submitting = true;
      _avatarState = AvatarState.thinking;
      _msgs.add(_OMsg(true, '好，我这就替你把主页生成出来，稍等 5–15 秒 ✨'));
    });
    _scrollEnd();
    try {
      final profile = await ref.read(profileApiProvider).createOrUpdate(
            ProfileInput(
              realName: _realName,
              title: _title,
              strengths: _strengths,
              recentWork: _recentWork,
              howToKnow: _howToKnow,
              style: _style.isEmpty ? null : _style,
              avatarStyle: _styleAvatar[_style] ?? 'toon-warm',
            ),
          );
      ref.invalidate(myProfileProvider);
      if (mounted) {
        context.goNamed('profile', pathParameters: {'id': profile.id});
      }
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _submitting = false;
          _avatarState = AvatarState.speaking;
        });
        _snack(e.message);
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _submitting = false;
          _avatarState = AvatarState.speaking;
        });
        _snack('生成失败：$e');
      }
    }
  }

  void _snack(String m) =>
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('创建我的 AI 助理')),
      body: Column(
        children: [
          Container(
            width: double.infinity,
            color: Colors.white,
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 14),
            child: Column(
              children: [
                WeifouAvatar(
                  style: _avatarStyle,
                  name: _realName,
                  size: 64,
                  active: _thinking || _submitting,
                  state: _avatarState,
                ),
                const SizedBox(height: 8),
                const Text('先聊几句，我来替你写主页',
                    style:
                        TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
                const Text('不用填表，说说就好',
                    style: TextStyle(fontSize: 12, color: AppTheme.sub)),
              ],
            ),
          ),
          Expanded(
            child: ListView.builder(
              controller: _scroll,
              padding: const EdgeInsets.all(16),
              itemCount: _msgs.length + (_thinking ? 1 : 0),
              itemBuilder: (_, i) {
                if (i >= _msgs.length) return const _TypingBubble();
                return _Bubble(msg: _msgs[i]);
              },
            ),
          ),
          if (_canFinish && !_submitting)
            Align(
              alignment: Alignment.centerLeft,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: ActionChip(
                  label: const Text('信息够了，先上岗 ›'),
                  backgroundColor: AppTheme.accentSoft,
                  labelStyle: const TextStyle(
                      color: AppTheme.accentInk, fontWeight: FontWeight.w500),
                  side: BorderSide.none,
                  onPressed: _finish,
                ),
              ),
            ),
          if (_submitting) const LinearProgressIndicator(minHeight: 2),
          _Composer(
            controller: _input,
            enabled: !_submitting,
            recording: _recording,
            onMicStart: _micStart,
            onMicStop: _micStop,
            onSend: _send,
          ),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: TextButton(
                onPressed: () => context.pushNamed('create'),
                child: const Text('更习惯填表？切换手动填写 ›',
                    style: TextStyle(fontSize: 12, color: AppTheme.sub)),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  const _Bubble({required this.msg});
  final _OMsg msg;

  @override
  Widget build(BuildContext context) {
    final ai = msg.fromAi;
    return Align(
      alignment: ai ? Alignment.centerLeft : Alignment.centerRight,
      child: Container(
        margin: const EdgeInsets.only(bottom: 12),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        constraints: BoxConstraints(
            maxWidth: MediaQuery.of(context).size.width * 0.78),
        decoration: BoxDecoration(
          color: ai ? Colors.white : AppTheme.ink,
          borderRadius: BorderRadius.circular(14),
          border: ai ? Border.all(color: const Color(0xFFF0E6DC)) : null,
        ),
        child: Text(msg.text,
            style: TextStyle(
                color: ai ? AppTheme.ink : Colors.white, height: 1.5)),
      ),
    );
  }
}

class _TypingBubble extends StatelessWidget {
  const _TypingBubble();

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 12),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: const Color(0xFFF0E6DC)),
        ),
        child: const Text('正在理解…',
            style: TextStyle(color: AppTheme.sub, fontSize: 13)),
      ),
    );
  }
}

class _Composer extends StatelessWidget {
  const _Composer({
    required this.controller,
    required this.enabled,
    required this.recording,
    required this.onMicStart,
    required this.onMicStop,
    required this.onSend,
  });

  final TextEditingController controller;
  final bool enabled;
  final bool recording;
  final VoidCallback onMicStart;
  final VoidCallback onMicStop;
  final VoidCallback onSend;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        children: [
          GestureDetector(
            onTapDown: enabled ? (_) => onMicStart() : null,
            onTapUp: (_) => onMicStop(),
            onTapCancel: onMicStop,
            child: Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: recording ? AppTheme.accentSoft : Colors.white,
                border: Border.all(
                    color: recording ? AppTheme.accent : AppTheme.border),
              ),
              child: Icon(Icons.mic,
                  size: 22,
                  color: recording ? AppTheme.accent : AppTheme.sub),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: TextField(
              controller: controller,
              enabled: enabled && !recording,
              minLines: 1,
              maxLines: 4,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => onSend(),
              decoration: InputDecoration(
                hintText: recording ? '正在听…松开结束' : '按住麦克风说，或打字',
                border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(24)),
                contentPadding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              ),
            ),
          ),
          const SizedBox(width: 8),
          IconButton.filled(
            onPressed: enabled ? onSend : null,
            style: IconButton.styleFrom(
              backgroundColor: AppTheme.accent,
              foregroundColor: Colors.white,
            ),
            icon: const Icon(Icons.arrow_upward),
          ),
        ],
      ),
    );
  }
}
