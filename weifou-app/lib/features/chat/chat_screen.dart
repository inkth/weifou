import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/avatar/weifou_avatar.dart';
import '../../core/network/api_exception.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/chat_api.dart';
import '../../data/models/profile.dart';

class _Msg {
  _Msg(this.fromMe, this.text, {this.animate = false});
  final bool fromMe;
  final String text;
  // 是否需要打字机动画（仅新到达的 AI 消息）；动画完成后置 false 防重播。
  bool animate;
}

/// 与某主页 AI 的沉浸式对话。需登录（路由守卫已拦截访客）。
class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key, required this.profileId, this.profile});

  final String profileId;
  final Profile? profile;

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final _input = TextEditingController();
  final _scroll = ScrollController();
  final List<_Msg> _msgs = [];
  bool _sending = false;

  List<String> get _starters => widget.profile?.persona?.starters ?? const [];

  @override
  void initState() {
    super.initState();
    // 沉浸式：进入即由 AI 开场白打招呼（打字机呈现）。
    final greeting = widget.profile?.persona?.greeting;
    if (greeting != null && greeting.trim().isNotEmpty) {
      _msgs.add(_Msg(false, greeting.trim(), animate: true));
    }
  }

  @override
  void dispose() {
    _input.dispose();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _send(String text) async {
    final content = text.trim();
    if (content.isEmpty || _sending) return;
    if (content.runes.length > 200) {
      _toast('问题太长（限 200 字）');
      return;
    }
    setState(() {
      _msgs.add(_Msg(true, content));
      _sending = true;
      _input.clear();
    });
    _scrollToEnd();
    try {
      final ans = await ref
          .read(chatApiProvider)
          .ask(widget.profileId, content);
      setState(() => _msgs.add(_Msg(false, ans.answer, animate: true)));
    } on ApiException catch (e) {
      setState(() => _msgs.add(_Msg(false, '（${e.message}）')));
    } catch (e) {
      setState(() => _msgs.add(_Msg(false, '（出错了：$e）')));
    } finally {
      if (mounted) setState(() => _sending = false);
      _scrollToEnd();
    }
  }

  void _scrollToEnd() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _toast(String m) =>
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));

  @override
  Widget build(BuildContext context) {
    final name = widget.profile?.realName ?? 'AI';
    final showStarters =
        _starters.isNotEmpty && _msgs.where((m) => m.fromMe).isEmpty;
    return Scaffold(
      appBar: AppBar(title: Text('与 $name 的 AI')),
      body: Column(
        children: [
          Container(
            width: double.infinity,
            decoration: const BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [Colors.white, AppTheme.accentSoft],
              ),
            ),
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 14),
            child: Column(
              children: [
                WeifouAvatar(
                  style: widget.profile?.avatarStyle,
                  name: name,
                  size: 64,
                  active: _sending,
                  state: _sending ? AvatarState.thinking : AvatarState.idle,
                ),
                const SizedBox(height: 8),
                Text(
                  name,
                  style: const TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                if (widget.profile?.title != null)
                  Text(
                    widget.profile!.title,
                    style: const TextStyle(fontSize: 12, color: AppTheme.sub),
                  ),
              ],
            ),
          ),
          Expanded(
            child: ListView.builder(
              controller: _scroll,
              padding: const EdgeInsets.all(16),
              itemCount: _msgs.length,
              itemBuilder: (_, i) => _Bubble(
                msg: _msgs[i],
                onAnimated: () => _msgs[i].animate = false,
                onTick: _scrollToEnd,
              ),
            ),
          ),
          if (showStarters) _StartersBar(starters: _starters, onTap: _send),
          if (_sending) const LinearProgressIndicator(minHeight: 2),
          _Composer(
            controller: _input,
            enabled: !_sending,
            onSend: () => _send(_input.text),
          ),
        ],
      ),
    );
  }
}

class _StartersBar extends StatelessWidget {
  const _StartersBar({required this.starters, required this.onTap});

  final List<String> starters;
  final void Function(String) onTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 44,
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: ListView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        children: [
          for (final s in starters)
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: ActionChip(
                label: Text(s),
                onPressed: () => onTap(s),
                backgroundColor: Colors.white,
                side: const BorderSide(color: AppTheme.border),
              ),
            ),
        ],
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  const _Bubble({
    required this.msg,
    required this.onAnimated,
    required this.onTick,
  });

  final _Msg msg;
  final VoidCallback onAnimated;
  final VoidCallback onTick;

  @override
  Widget build(BuildContext context) {
    final me = msg.fromMe;
    final Widget content = (!me && msg.animate)
        ? _TypewriterText(
            text: msg.text,
            onDone: onAnimated,
            onTick: onTick,
            style: const TextStyle(color: AppTheme.ink, height: 1.5),
          )
        : Text(
            msg.text,
            style: TextStyle(
              color: me ? Colors.white : AppTheme.ink,
              height: 1.5,
            ),
          );
    return Align(
      alignment: me ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 12),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.75,
        ),
        decoration: BoxDecoration(
          color: me ? AppTheme.accent : Colors.white,
          borderRadius: BorderRadius.circular(14),
          border: me ? null : Border.all(color: AppTheme.border),
        ),
        child: content,
      ),
    );
  }
}

/// 打字机：逐字显示。完成后回调 onDone 让父级标记不再重播。
class _TypewriterText extends StatefulWidget {
  const _TypewriterText({
    required this.text,
    required this.onDone,
    required this.onTick,
    required this.style,
  });

  final String text;
  final VoidCallback onDone;
  final VoidCallback onTick;
  final TextStyle style;

  @override
  State<_TypewriterText> createState() => _TypewriterTextState();
}

class _TypewriterTextState extends State<_TypewriterText> {
  int _count = 0;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    final runes = widget.text.runes.toList();
    _timer = Timer.periodic(const Duration(milliseconds: 28), (t) {
      if (_count >= runes.length) {
        t.cancel();
        widget.onDone();
        return;
      }
      setState(() => _count++);
      if (_count % 6 == 0) widget.onTick();
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final shown = String.fromCharCodes(widget.text.runes.take(_count));
    return Text(shown, style: widget.style);
  }
}

class _Composer extends StatelessWidget {
  const _Composer({
    required this.controller,
    required this.enabled,
    required this.onSend,
  });

  final TextEditingController controller;
  final bool enabled;
  final VoidCallback onSend;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            Expanded(
              child: TextField(
                controller: controller,
                enabled: enabled,
                minLines: 1,
                maxLines: 4,
                textInputAction: TextInputAction.send,
                onSubmitted: (_) => onSend(),
                decoration: InputDecoration(
                  hintText: '输入问题（限 200 字）',
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(24),
                  ),
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 10,
                  ),
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
              icon: const Icon(Icons.send),
            ),
          ],
        ),
      ),
    );
  }
}
