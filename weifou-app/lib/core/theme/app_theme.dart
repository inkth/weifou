import 'package:flutter/material.dart';

/// 与小程序 app.wxss 一致的视觉规范：墨黑主色、浅灰底、胶囊按钮。
class AppTheme {
  AppTheme._();

  // 设计令牌（双端真源见 docs/design-tokens.md）。墨黑仍为主色，暖橙只做强调。
  static const Color ink = Color(0xFF1F2330); // 主色 / 文字
  static const Color bg = Color(0xFFF5F6FA); // 页面底色
  static const Color border = Color(0xFFE5E7EC); // 描边
  static const Color sub = Color(0xFF8A8F9C); // 次要文字

  static const Color accent = Color(0xFFFB923C); // 暖强调色：仅 CTA 高亮 / 活跃态 / 强调
  static const Color accentStrong = Color(0xFFEF7D1F); // 强调色按下态
  static const Color accentSoft = Color(0xFFFFF3E9); // 浅暖底：高亮区背景 / 标签
  static const Color accentInk = Color(0xFF9A4D12); // accentSoft 上的文字
  static const Color success = Color(0xFF10B981);
  static const Color warn = Color(0xFFF59E0B);
  static const Color danger = Color(0xFFE0404B);

  // 柔和暖阴影（卡片 / 浮层），对应小程序 --shadow-soft
  static const List<BoxShadow> softShadow = [
    BoxShadow(color: Color(0x1AF98C3C), blurRadius: 13, offset: Offset(0, 3)),
  ];

  static ThemeData get light {
    final base = ThemeData.light(useMaterial3: true);
    return base.copyWith(
      scaffoldBackgroundColor: bg,
      colorScheme: base.colorScheme.copyWith(
        primary: ink,
        secondary: accent, // 暖强调色，CTA 高亮取 Theme.of(context).colorScheme.secondary
        error: danger,
        surface: Colors.white,
      ),
      textTheme: base.textTheme.apply(
        bodyColor: ink,
        displayColor: ink,
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.white,
        foregroundColor: ink,
        elevation: 0,
        centerTitle: true,
      ),
      // 胶囊主按钮（对应 .btn-primary）。
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: ink,
          foregroundColor: Colors.white,
          minimumSize: const Size.fromHeight(50),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(100),
          ),
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
        ),
      ),
      // 胶囊幽灵按钮（对应 .btn-ghost）。
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: ink,
          backgroundColor: Colors.white,
          minimumSize: const Size.fromHeight(50),
          side: const BorderSide(color: border),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(100),
          ),
        ),
      ),
    );
  }

  /// 暖色强调主按钮样式（对应小程序 .btn-accent）。仅用在关键转化点
  /// （预约 / 立即开聊 等），不替代默认墨黑 ElevatedButton。
  /// 用法：ElevatedButton(style: AppTheme.accentButton, ...)
  static final ButtonStyle accentButton = ElevatedButton.styleFrom(
    backgroundColor: accent,
    foregroundColor: Colors.white,
    minimumSize: const Size.fromHeight(50),
    shape: RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(100),
    ),
    textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
  );
}
