import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;

import '../../core/network/api_exception.dart';
import '../../core/theme/app_theme.dart';
import '../../data/api/profile_api.dart';

/// 对话式创建：把原「分步点选」融进对话——AI 逐步引导，结构化项（做什么/接待谁/气质）直接给
/// 可点「快捷气泡」，点了即答；名字与一句话走输入/语音；也可自由说一段由 /profile/extract 一次
/// 抽多字段并跳过已答步骤。必填 realName/title/strengths 齐即可上岗。与小程序 pages/onboarding 全面
/// 对齐：人物在场的雾蓝紫亮场 + 高对比气泡/选项/输入条。
const _opener =
    '嗨，我是来帮你建主页的 AI 助理～ 先用一两句话介绍下你自己：你是谁、平时做什么、最能帮别人解决什么问题？想到哪说到哪，也可以按住下方麦克风说。';

const Map<String, String> _styleAvatar = {
  'steady': 'toon-steady',
  'warm': 'toon-warm',
  'sharp': 'toon-sharp',
  'humorous': 'toon-humorous',
};

// —— 雾蓝紫亮场（与小程序 onboarding.wxss 同源）——
const _accent = AppTheme.accent;
const _accentSoft = AppTheme.accentSoft;
const _accentBorder = Color(0xFFCCC8E7);
const _glassFill = Color(0xF2FFFFFF);
const _glassBorder = AppTheme.border;
const _aiBubble = Color(0xF7FFFFFF);
const _aiBorder = AppTheme.border;
const _userBubble = AppTheme.accent;

// —— 结构化快捷选项（与小程序 pages/onboarding 同源）——
const _domains = [
  '顾问·教练',
  '设计·创意',
  '开发·技术',
  '教育·培训',
  '医美·健康',
  '法律·财税',
  '电商·带货',
  '内容·创作',
  '生活服务',
];

class _Audience {
  const _Audience(this.label, this.hk);
  final String label; // 气泡显示
  final String hk; // 写入 howToKnow 的干净取值
}

const _audiences = [
  _Audience('找合作', '主要想接待：找合作'),
  _Audience('想买你服务', '主要想接待：想买我服务的人'),
  _Audience('同行 · 招募', '主要想接待：同行或想招募我的人'),
  _Audience('粉丝 · 读者', '主要想接待：我的粉丝或读者'),
  _Audience('都行', '主要想接待：各种来访者'),
];

class _StyleOpt {
  const _StyleOpt(this.label, this.value, this.desc);
  final String label;
  final String value; // style 白名单 key
  final String desc;
}

const _styles = [
  _StyleOpt('专业冷静', 'steady', '严谨克制 · 先结论'),
  _StyleOpt('温暖亲和', 'warm', '口语 · 先共情'),
  _StyleOpt('犀利直接', 'sharp', '一针见血 · 不绕弯'),
  _StyleOpt('轻松幽默', 'humorous', '有梗 · 不油腻'),
];

// 引导阶段（与小程序同序）：字段 / 是否必填 / 提问 / 快捷气泡类型
class _Stage {
  const _Stage(this.key, this.field, this.required, this.ask, this.chips);
  final String key;
  final String field;
  final bool required;
  final String ask;
  final String? chips; // 'domain' | 'audience' | 'style' | null
}

const _stages = [
  _Stage('name', 'realName', true, '先问一句——我该怎么称呼你？', null),
  _Stage('domain', 'title', true, '你主要是做什么的？挑一个最接近的，或直接说～', 'domain'),
  _Stage('audience', 'howToKnow', false, '主要想接待谁？', 'audience'),
  _Stage('style', 'style', false, '希望你的 AI 什么气质、说话调？', 'style'),
  _Stage(
    'substance',
    'strengths',
    true,
    '最后——你最能帮别人解决的一件事是什么？越具体它越懂你（也可写个代表作）。',
    null,
  ),
];

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
  bool _edit = false; // 已有主页 = 编辑态：预填、开场回显、改完即更新
  bool _initializing = true; // 进入时探测是否已有主页

  final _speech = stt.SpeechToText();
  bool _speechReady = false;

  // 累积抽取出的字段
  String _realName = '', _title = '', _strengths = '';
  String _recentWork = '', _howToKnow = '', _style = '';
  String _company = '', _city = ''; // 编辑态透传，避免提交被服务端空值清掉

  final Map<String, bool> _asked = {}; // 已问过的阶段（可选项只问一次）
  final Map<String, bool> _locked = {}; // 经点选确定的字段：后续抽取不再覆盖
  String? _chipKind; // 当前展示的快捷气泡：domain | audience | style | null

  // 字段名 ↔ 累积变量（供分步引导/抽取合并统一存取）
  String _fieldValue(String field) {
    switch (field) {
      case 'realName':
        return _realName;
      case 'title':
        return _title;
      case 'strengths':
        return _strengths;
      case 'recentWork':
        return _recentWork;
      case 'howToKnow':
        return _howToKnow;
      case 'style':
        return _style;
    }
    return '';
  }

  void _setField(String field, String v) {
    switch (field) {
      case 'realName':
        _realName = v;
        break;
      case 'title':
        _title = v;
        break;
      case 'strengths':
        _strengths = v;
        break;
      case 'recentWork':
        _recentWork = v;
        break;
      case 'howToKnow':
        _howToKnow = v;
        break;
      case 'style':
        _style = v;
        break;
    }
  }

  bool _filled(String field) => _fieldValue(field).trim().isNotEmpty;

  @override
  void initState() {
    super.initState();
    _init();
  }

  // 创建/编辑统一走对话：已有主页则预填进入编辑态，否则用创建开场。
  Future<void> _init() async {
    try {
      final d = await ref.read(profileApiProvider).mineDraft();
      if (!mounted) return;
      if (d != null) {
        _edit = true;
        _realName = d.realName;
        _title = d.title;
        _strengths = d.strengths;
        _recentWork = d.recentWork;
        _howToKnow = d.howToKnow;
        _style = d.style;
        _company = d.company;
        _city = d.city;
        _canFinish = true;
        _confirmed = true;
        _msgs.add(
          _OMsg(
            true,
            '这是你现在的 AI 主页——${d.realName}｜${d.title}。想更新点什么？换一句话简介、改说话语气、补个最近在做的事……跟我说就行，没提到的都给你留着。改完点「更新主页」。',
          ),
        );
      } else {
        _msgs.add(_OMsg(true, _opener));
      }
    } catch (_) {
      if (!mounted) return;
      _msgs.add(_OMsg(true, _opener)); // 兜底当创建
    } finally {
      if (mounted) setState(() => _initializing = false);
    }
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
        _input.selection = TextSelection.collapsed(offset: _input.text.length);
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
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  // 统一推进（与小程序 _afterAnswer 对齐）：算必填齐没 → 找下一个该问的阶段+气泡 → 抛问题。
  // 须在 setState 内调用（直接改 _msgs/_canFinish/_chipKind）。
  void _afterAnswer() {
    final complete =
        _realName.isNotEmpty && _title.isNotEmpty && _strengths.isNotEmpty;
    _canFinish = complete;

    // 编辑态：不逐项追问，改了回一句确认，随时可「更新主页」
    if (_edit) {
      _chipKind = null;
      _msgs.add(_OMsg(true, '好，记下了～ 还想改别的就继续说，或点「更新主页」。'));
      return;
    }

    if (complete && !_confirmed) {
      _confirmed = true;
      _msgs.add(_OMsg(true, '好，我大概了解你了～ 随时点「先上岗」，也能再补两句让我更懂你。'));
    }

    _Stage? next;
    for (final s in _stages) {
      if (!_filled(s.field) && (s.required || !(_asked[s.key] ?? false))) {
        next = s;
        break;
      }
    }
    if (next == null) {
      _chipKind = null;
      return;
    }
    _asked[next.key] = true;
    _msgs.add(_OMsg(true, next.ask));
    _chipKind = next.chips;
  }

  // 点气泡 = 直接填字段 + 记一条用户气泡，不走服务端抽取（快、稳、零误判）
  void _pickChip(String field, String value, String label) {
    if (_thinking || _submitting) return;
    setState(() {
      _locked[field] = true; // 点选即锁定，避免后续抽取覆盖
      _msgs.add(_OMsg(false, label));
      _setField(field, value);
      _chipKind = null;
      _afterAnswer();
    });
    _scrollEnd();
  }

  Future<void> _send() async {
    final v = _input.text.trim();
    if (v.isEmpty || _thinking || _submitting) return;
    setState(() {
      _msgs.add(_OMsg(false, v));
      _input.clear();
      _thinking = true;
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
        _msgs.add(_OMsg(true, '（网络好像有点慢，刚那句再说一遍试试？）'));
      });
      _scrollEnd();
    }
  }

  void _applyExtract(ExtractedProfile res) {
    // 合并：已点选锁定的字段保留不动；其余服务端非空则覆盖、空则保留（防 LLM 偶发漏带）
    String pick(String field, String val) => (_locked[field] ?? false)
        ? _fieldValue(field)
        : (val.isNotEmpty ? val : _fieldValue(field));
    setState(() {
      _setField('realName', pick('realName', res.realName));
      _setField('title', pick('title', res.title));
      _setField('strengths', pick('strengths', res.strengths));
      _setField('recentWork', pick('recentWork', res.recentWork));
      _setField('howToKnow', pick('howToKnow', res.howToKnow));
      _setField('style', pick('style', res.style));
      _thinking = false;
      _afterAnswer();
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
      _msgs.add(
        _OMsg(true, _edit ? '好，这就替你更新主页，稍等几秒 ✨' : '好，我这就替你把主页生成出来，稍等 5–15 秒 ✨'),
      );
    });
    _scrollEnd();
    try {
      final profile = await ref
          .read(profileApiProvider)
          .createOrUpdate(
            ProfileInput(
              realName: _realName,
              title: _title,
              strengths: _strengths,
              recentWork: _recentWork,
              howToKnow: _howToKnow,
              style: _style.isEmpty ? null : _style,
              company: _company.isEmpty ? null : _company,
              city: _city.isEmpty ? null : _city,
              avatarStyle: _styleAvatar[_style] ?? 'toon-warm',
            ),
          );
      ref.invalidate(myProfileProvider);
      if (mounted) {
        context.goNamed('profile', pathParameters: {'id': profile.id});
      }
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _submitting = false);
        _snack(e.message);
      }
    } catch (e) {
      if (mounted) {
        setState(() => _submitting = false);
        _snack('生成失败：$e');
      }
    }
  }

  void _snack(String m) =>
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      extendBodyBehindAppBar: true,
      backgroundColor: AppTheme.bg,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        surfaceTintColor: Colors.transparent,
        foregroundColor: AppTheme.ink,
        systemOverlayStyle: SystemUiOverlayStyle.dark,
      ),
      body: Stack(
        children: [
          // 雾蓝紫亮场兜底。
          const DecoratedBox(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
                colors: [
                  Color(0xFFE2DFF5),
                  Color(0xFFE2F0F6),
                  Color(0xFFF6F7FB),
                ],
                stops: [0, 0.58, 1],
              ),
            ),
            child: SizedBox.expand(),
          ),
          // 创建阶段不预设人物形象，只保留安静的 AI 环境光。
          const Positioned.fill(
            child: DecoratedBox(
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: [
                    Color(0x66FFFFFF),
                    Color(0x1AFFFFFF),
                    Color(0x99F6F7FB),
                    Color(0xF7F6F7FB),
                  ],
                  stops: [0, 0.26, 0.62, 1],
                ),
              ),
            ),
          ),
          SafeArea(
            child: _initializing
                ? const Center(child: CircularProgressIndicator(color: _accent))
                : Column(
                    children: [
                      const SizedBox(height: kToolbarHeight),
                      Text(
                        _edit ? '想更新什么？说一句就改' : '先聊几句，我来替你写主页',
                        style: const TextStyle(
                          color: AppTheme.ink,
                          fontSize: 15,
                          fontWeight: FontWeight.w600,
                          shadows: [Shadow(color: Colors.white, blurRadius: 8)],
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        _edit ? '没提到的都给你留着' : '说一句，或点下方选项都行',
                        style: const TextStyle(
                          color: AppTheme.ink2,
                          fontSize: 12,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Expanded(
                        child: ListView.builder(
                          controller: _scroll,
                          padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                          itemCount: _msgs.length + (_thinking ? 1 : 0),
                          itemBuilder: (_, i) {
                            if (i >= _msgs.length) return const _TypingBubble();
                            return _Bubble(msg: _msgs[i]);
                          },
                        ),
                      ),
                      Padding(
                        padding: const EdgeInsets.fromLTRB(16, 0, 16, 10),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            if (_chipKind != null && !_submitting)
                              _buildChips(),
                            if (_canFinish && !_submitting)
                              Padding(
                                padding: const EdgeInsets.only(bottom: 12),
                                child: _finishChip(),
                              ),
                            if (_submitting)
                              const Padding(
                                padding: EdgeInsets.only(bottom: 10),
                                child: LinearProgressIndicator(
                                  minHeight: 2,
                                  color: _accent,
                                ),
                              ),
                            _Composer(
                              controller: _input,
                              enabled: !_submitting,
                              recording: _recording,
                              onMicStart: _micStart,
                              onMicStop: _micStop,
                              onSend: _send,
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
          ),
        ],
      ),
    );
  }

  // 当前阶段的快捷气泡：行业/接待=白底选项；气质=两行卡片。
  Widget _buildChips() {
    final List<Widget> chips;
    switch (_chipKind) {
      case 'domain':
        chips = _domains
            .map((d) => _pill(d, () => _pickChip('title', d, d)))
            .toList();
        break;
      case 'audience':
        chips = _audiences
            .map(
              (a) =>
                  _pill(a.label, () => _pickChip('howToKnow', a.hk, a.label)),
            )
            .toList();
        break;
      case 'style':
        chips = _styles.map(_styleCard).toList();
        break;
      default:
        chips = const [];
    }
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Wrap(spacing: 10, runSpacing: 10, children: chips),
    );
  }

  Widget _pill(String label, VoidCallback onTap) => Material(
    color: Colors.transparent,
    child: InkWell(
      borderRadius: BorderRadius.circular(100),
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 9),
        decoration: BoxDecoration(
          color: _glassFill,
          borderRadius: BorderRadius.circular(100),
          border: Border.all(color: _glassBorder),
        ),
        child: Text(
          label,
          style: const TextStyle(color: AppTheme.ink, fontSize: 13),
        ),
      ),
    ),
  );

  Widget _styleCard(_StyleOpt s) => Material(
    color: Colors.transparent,
    child: InkWell(
      borderRadius: BorderRadius.circular(14),
      onTap: () => _pickChip('style', s.value, s.label),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: _glassFill,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: _glassBorder),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              s.label,
              style: const TextStyle(
                color: AppTheme.ink,
                fontSize: 14,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 2),
            Text(
              s.desc,
              style: const TextStyle(color: AppTheme.sub, fontSize: 11),
            ),
          ],
        ),
      ),
    ),
  );

  Widget _finishChip() => Material(
    color: Colors.transparent,
    child: InkWell(
      borderRadius: BorderRadius.circular(100),
      onTap: _finish,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 10),
        decoration: BoxDecoration(
          color: _accentSoft,
          borderRadius: BorderRadius.circular(100),
          border: Border.all(color: _accentBorder),
        ),
        child: Text(
          _edit ? '更新主页 ›' : '信息够了，先上岗 ›',
          style: const TextStyle(
            color: AppTheme.accentInk,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    ),
  );
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
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 11),
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.78,
        ),
        decoration: BoxDecoration(
          color: ai ? _aiBubble : _userBubble,
          borderRadius: BorderRadius.only(
            topLeft: const Radius.circular(18),
            topRight: const Radius.circular(18),
            bottomLeft: Radius.circular(ai ? 6 : 18),
            bottomRight: Radius.circular(ai ? 18 : 6),
          ),
          border: ai ? Border.all(color: _aiBorder) : null,
        ),
        child: Text(
          msg.text,
          style: TextStyle(
            color: ai ? AppTheme.ink : Colors.white,
            height: 1.5,
          ),
        ),
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
          color: _aiBubble,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(color: _aiBorder),
        ),
        child: const Text(
          '正在理解…',
          style: TextStyle(color: AppTheme.sub, fontSize: 13),
        ),
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
    return Row(
      children: [
        GestureDetector(
          onTapDown: enabled ? (_) => onMicStart() : null,
          onTapUp: (_) => onMicStop(),
          onTapCancel: onMicStop,
          child: Container(
            width: 46,
            height: 46,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: recording ? _accentSoft : _glassFill,
              border: Border.all(
                color: recording ? _accentBorder : _glassBorder,
              ),
            ),
            child: Icon(
              Icons.mic,
              size: 22,
              color: recording ? AppTheme.accentDeep : AppTheme.ink2,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: TextField(
            controller: controller,
            enabled: enabled && !recording,
            minLines: 1,
            maxLines: 4,
            style: const TextStyle(color: AppTheme.ink),
            cursorColor: _accent,
            textInputAction: TextInputAction.send,
            onSubmitted: (_) => onSend(),
            decoration: InputDecoration(
              hintText: recording ? '正在听…松开结束' : '按住麦克风说，或打字',
              hintStyle: const TextStyle(color: AppTheme.sub),
              filled: true,
              fillColor: _glassFill,
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 10,
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(24),
                borderSide: const BorderSide(color: _glassBorder),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(24),
                borderSide: const BorderSide(color: _accent),
              ),
              disabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(24),
                borderSide: const BorderSide(color: _glassBorder),
              ),
            ),
          ),
        ),
        const SizedBox(width: 10),
        Container(
          decoration: const BoxDecoration(
            shape: BoxShape.circle,
            color: _accent,
          ),
          child: IconButton(
            onPressed: enabled ? onSend : null,
            icon: const Icon(Icons.arrow_upward, color: Colors.white),
          ),
        ),
      ],
    );
  }
}
