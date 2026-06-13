import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/app_theme.dart';

/// 对话状态，驱动头像的眼/嘴表情（对齐小程序 components/avatar 的 s-idle/thinking/speaking）。
enum AvatarState { idle, thinking, speaking }

/// 形象预设。与小程序 utils/avatars.js 的 PRESETS 同源维护：
/// toonLook 非空 = 纯绘制卡通脸（steady/warm/sharp/humorous）；为空 = 渐变 + 首字形象。
class AvatarPreset {
  const AvatarPreset(this.id, {this.toonLook, required this.colors});
  final String id;
  final String? toonLook;
  final List<Color> colors;
}

const Map<String, AvatarPreset> kAvatarPresets = {
  'aurora': AvatarPreset('aurora',
      colors: [Color(0xFF6366F1), Color(0xFFA855F7), Color(0xFFEC4899)]),
  'ocean': AvatarPreset('ocean', colors: [Color(0xFF0EA5E9), Color(0xFF2563EB)]),
  'mint': AvatarPreset('mint', colors: [Color(0xFF10B981), Color(0xFF34D399)]),
  'sunset': AvatarPreset('sunset', colors: [Color(0xFFF59E0B), Color(0xFFEF4444)]),
  'graphite':
      AvatarPreset('graphite', colors: [Color(0xFF374151), Color(0xFF111827)]),
  'lavender':
      AvatarPreset('lavender', colors: [Color(0xFF8B5CF6), Color(0xFFC4B5FD)]),
  'coral': AvatarPreset('coral', colors: [Color(0xFFFB7185), Color(0xFFF43F5E)]),
  'forest': AvatarPreset('forest', colors: [Color(0xFF16A34A), Color(0xFF065F46)]),
  'toon-steady': AvatarPreset('toon-steady',
      toonLook: 'steady', colors: [Color(0xFF475569), Color(0xFF1F2330)]),
  'toon-warm': AvatarPreset('toon-warm',
      toonLook: 'warm', colors: [Color(0xFFFB923C), Color(0xFFF43F5E)]),
  'toon-sharp': AvatarPreset('toon-sharp',
      toonLook: 'sharp', colors: [Color(0xFF7C3AED), Color(0xFF4F46E5)]),
  'toon-humorous': AvatarPreset('toon-humorous',
      toonLook: 'humorous', colors: [Color(0xFF10B981), Color(0xFF22D3EE)]),
};

int _hashStr(String s) {
  var h = 0;
  for (final c in s.codeUnits) {
    h = (h * 31 + c) & 0x7fffffff;
  }
  return h;
}

/// 取预设；style 为空/未知时按 seed 确定性兜底（与小程序 getPreset 一致）。
AvatarPreset resolveAvatarPreset(String? style, {String seed = ''}) {
  final hit = kAvatarPresets[style];
  if (hit != null) return hit;
  final keys = kAvatarPresets.keys.toList();
  return kAvatarPresets[keys[_hashStr(seed) % keys.length]]!;
}

/// 姓名首字符（中文取首字，英文大写），空名回落「微」。
String avatarInitial(String? name) {
  final n = (name ?? '').trim();
  if (n.isEmpty) return '微';
  return String.fromCharCode(n.runes.first).toUpperCase();
}

/// 微否动态形象：会呼吸/漂浮的头像，支持卡通脸三态与渐变首字两种渲染。
/// active=true（如 AI 正在回应）时呼吸幅度更大、节奏更快。
class WeifouAvatar extends StatefulWidget {
  const WeifouAvatar({
    super.key,
    required this.style,
    required this.name,
    this.size = 64,
    this.state = AvatarState.idle,
    this.active = false,
  });

  final String? style;
  final String name;
  final double size;
  final AvatarState state;
  final bool active;

  @override
  State<WeifouAvatar> createState() => _WeifouAvatarState();
}

class _WeifouAvatarState extends State<WeifouAvatar>
    with TickerProviderStateMixin {
  late final AnimationController _alive; // 呼吸 + 漂浮
  late final AnimationController _blink; // 眨眼

  @override
  void initState() {
    super.initState();
    _alive = AnimationController(vsync: this, duration: _aliveDuration)..repeat(reverse: true);
    _blink = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 4600))
      ..repeat();
  }

  Duration get _aliveDuration => Duration(milliseconds: widget.active ? 1300 : 3100);

  @override
  void didUpdateWidget(WeifouAvatar old) {
    super.didUpdateWidget(old);
    if (old.active != widget.active) {
      _alive
        ..duration = _aliveDuration
        ..repeat(reverse: true);
    }
  }

  @override
  void dispose() {
    _alive.dispose();
    _blink.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final preset = resolveAvatarPreset(widget.style, seed: widget.name);
    final s = widget.size;
    return AnimatedBuilder(
      animation: Listenable.merge([_alive, _blink]),
      builder: (_, _) {
        final phase = math.sin(_alive.value * math.pi); // 0→1→0
        final scale = 1 + (widget.active ? 0.075 : 0.035) * _alive.value;
        final dy = -(widget.active ? 7.0 : 4.0) * phase;
        // 眨眼：4.6s 周期内末段快速合眼
        final bv = _blink.value;
        final eyeOpen = bv > 0.93 && bv < 0.985 ? 0.12 : 1.0;
        return Transform.translate(
          offset: Offset(0, dy),
          child: Transform.scale(
            scale: scale,
            child: Container(
              width: s,
              height: s,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: preset.colors,
                ),
                boxShadow: const [
                  BoxShadow(
                      color: Color(0x1F000000),
                      blurRadius: 12,
                      offset: Offset(0, 6)),
                ],
              ),
              child: preset.toonLook == null
                  ? Center(
                      child: Text(
                        avatarInitial(widget.name),
                        style: TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.w700,
                          fontSize: s * 0.4,
                        ),
                      ),
                    )
                  : CustomPaint(
                      painter: _ToonPainter(
                        look: preset.toonLook!,
                        state: widget.state,
                        eyeOpen: eyeOpen,
                        talk: phase, // speaking 时嘴开合用
                      ),
                    ),
            ),
          ),
        );
      },
    );
  }
}

/// 纯绘制卡通脸，几何参数对齐小程序 components/avatar/index.wxss 的 toon-* 规则。
class _ToonPainter extends CustomPainter {
  _ToonPainter({
    required this.look,
    required this.state,
    required this.eyeOpen,
    required this.talk,
  });

  final String look;
  final AvatarState state;
  final double eyeOpen; // 0~1 睁眼程度
  final double talk; // 0~1 说话开合相位

  static const _white = Colors.white;
  static const _pupil = AppTheme.ink;

  @override
  void paint(Canvas canvas, Size size) {
    final s = size.width;
    final white = Paint()..color = _white;
    final pupil = Paint()..color = _pupil;

    final sharp = look == 'sharp';
    final eyeW = 0.17 * s;
    final eyeH = (sharp ? 0.17 : 0.22) * s * eyeOpen;
    final eyeCY = (sharp ? 0.455 : 0.45) * s;
    final eyeLX = 0.325 * s;
    final eyeRX = 0.675 * s;
    final thinking = state == AvatarState.thinking;

    void drawEye(double cx) {
      canvas.drawOval(
        Rect.fromCenter(center: Offset(cx, eyeCY), width: eyeW, height: eyeH),
        white,
      );
      final py = eyeCY - (thinking ? 0.035 * s : 0);
      final px = cx + (thinking ? 0.012 * s : 0);
      canvas.drawCircle(Offset(px, py), 0.038 * s * eyeOpen.clamp(0.25, 1), pupil);
    }

    drawEye(eyeLX);
    drawEye(eyeRX);

    // 眉毛：仅犀利显示
    if (sharp) {
      final brow = Paint()
        ..color = _white
        ..strokeCap = StrokeCap.round
        ..strokeWidth = 0.045 * s;
      _rotLine(canvas, brow, Offset(0.33 * s, 0.27 * s), 0.14 * s, 0.24);
      _rotLine(canvas, brow, Offset(0.67 * s, 0.27 * s), 0.14 * s, -0.24);
    }

    // 腮红：warm / humorous 显示
    if (look == 'warm' || look == 'humorous') {
      final blush = Paint()..color = _white.withValues(alpha: 0.38);
      canvas.drawOval(
          Rect.fromCenter(
              center: Offset(0.20 * s, 0.62 * s),
              width: 0.14 * s,
              height: 0.09 * s),
          blush);
      canvas.drawOval(
          Rect.fromCenter(
              center: Offset(0.80 * s, 0.62 * s),
              width: 0.14 * s,
              height: 0.09 * s),
          blush);
    }

    // 嘴
    _drawMouth(canvas, s, white);
  }

  void _rotLine(
      Canvas canvas, Paint p, Offset center, double len, double angle) {
    final dx = math.cos(angle) * len / 2;
    final dy = math.sin(angle) * len / 2;
    canvas.drawLine(
        Offset(center.dx - dx, center.dy - dy),
        Offset(center.dx + dx, center.dy + dy),
        p);
  }

  void _drawMouth(Canvas canvas, double s, Paint white) {
    double w = 0.26 * s, h = 0.13 * s;
    final cx = 0.5 * s, cy = 0.68 * s;
    if (look == 'steady') {
      w = 0.20 * s;
      h = 0.06 * s;
    } else if (look == 'humorous') {
      w = 0.32 * s;
      h = 0.16 * s;
    }
    if (state == AvatarState.thinking) {
      // 收成小点
      canvas.drawCircle(Offset(cx, cy), 0.045 * s, white);
      return;
    }
    if (state == AvatarState.speaking) {
      h = h * (0.5 + talk * 0.9); // 开合
    }
    // 微笑：上沿直线 + 下沿外凸的半透镜形
    final path = Path()
      ..moveTo(cx - w / 2, cy - h * 0.2)
      ..quadraticBezierTo(cx, cy + h, cx + w / 2, cy - h * 0.2)
      ..close();
    canvas.drawPath(path, white);
  }

  @override
  bool shouldRepaint(_ToonPainter old) =>
      old.look != look ||
      old.state != state ||
      old.eyeOpen != eyeOpen ||
      old.talk != talk;
}
