import 'package:flutter/material.dart';

/// 与小程序一致的人物编辑感 + 雾蓝紫 AI 气场。
class AppTheme {
  AppTheme._();

  // 设计令牌（双端真源见 docs/design-tokens.md）。
  static const Color ink = Color(0xFF17181C);
  static const Color ink2 = Color(0xFF5E616D);
  static const Color sub = Color(0xFF8D909B);
  static const Color faint = Color(0xFFB5B7C0);
  static const Color line = Color(0xFFF0F1F5);
  static const Color border = Color(0xFFE7E8EE);
  static const Color fill = Color(0xFFF3F4F7);
  static const Color fillPressed = Color(0xFFECEEF3);

  // —— 表面与底色 ——
  static const Color bg = Color(0xFFF6F7FB);
  static const Color surface = Color(0xFFFFFFFF); // 卡片 / 浮层表面
  static const Color surfaceSunken = Color(0xFFF1F2F7);

  // —— 雾蓝紫品牌与 AI 气场 ——
  static const Color accent = Color(0xFF7772C8);
  static const Color accentStrong = Color(0xFF5B569F);
  static const Color accentDeep = Color(0xFF4F4A8B);
  static const Color accentSoft = Color(0xFFEFEDFA);
  static const Color accentInk = Color(0xFF47427D);
  static const Color mistBlue = Color(0xFFB9DDED);
  static const Color mistLilac = Color(0xFFC9C3EB);
  static const Color success = Color(0xFF4F9D78);
  static const Color warn = Color(0xFFD39A4A);
  static const Color danger = Color(0xFFD85C68);

  // —— 圆角阶（对应小程序 --r-*）——
  static const double rSm = 6,
      rMd = 10,
      rLg = 13,
      rXl = 16,
      r2xl = 20,
      rFull = 999;

  // —— 柔和空间阴影 ——
  static const List<BoxShadow> cardShadow = [
    BoxShadow(color: Color(0x0F222038), blurRadius: 14, offset: Offset(0, 4)),
  ];
  static const List<BoxShadow> softShadow = [
    BoxShadow(color: Color(0x1C5B569F), blurRadius: 19, offset: Offset(0, 6)),
  ];
  static const List<BoxShadow> accentShadow = [
    BoxShadow(color: Color(0x335B569F), blurRadius: 14, offset: Offset(0, 5)),
  ];

  static ThemeData get light {
    final base = ThemeData.light(useMaterial3: true);
    return base.copyWith(
      scaffoldBackgroundColor: bg,
      colorScheme: base.colorScheme.copyWith(
        primary: ink,
        secondary: accent,
        error: danger,
        surface: Colors.white,
      ),
      textTheme: base.textTheme.apply(bodyColor: ink, displayColor: ink),
      appBarTheme: const AppBarTheme(
        backgroundColor: bg,
        foregroundColor: ink,
        elevation: 0,
        scrolledUnderElevation: 0,
        surfaceTintColor: Colors.transparent,
        centerTitle: true,
      ),
      cardTheme: CardThemeData(
        color: surface,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(rXl),
          side: const BorderSide(color: border),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: surface,
        hintStyle: const TextStyle(color: sub),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(rMd),
          borderSide: const BorderSide(color: border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(rMd),
          borderSide: const BorderSide(color: border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(rMd),
          borderSide: const BorderSide(color: accent, width: 1.4),
        ),
      ),
      chipTheme: base.chipTheme.copyWith(
        backgroundColor: fill,
        selectedColor: accentSoft,
        side: const BorderSide(color: border),
        labelStyle: const TextStyle(color: ink2),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(rSm)),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: surface,
        indicatorColor: Colors.transparent,
        elevation: 0,
        height: 64,
        labelTextStyle: WidgetStateProperty.resolveWith(
          (states) => TextStyle(
            color: states.contains(WidgetState.selected) ? accentDeep : sub,
            fontSize: 12,
            fontWeight: states.contains(WidgetState.selected)
                ? FontWeight.w700
                : FontWeight.w500,
          ),
        ),
        iconTheme: WidgetStateProperty.resolveWith(
          (states) => IconThemeData(
            color: states.contains(WidgetState.selected) ? accentDeep : sub,
          ),
        ),
      ),
      progressIndicatorTheme: const ProgressIndicatorThemeData(color: accent),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith(
          (states) =>
              states.contains(WidgetState.selected) ? Colors.white : faint,
        ),
        trackColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.selected) ? accent : fill,
        ),
      ),
      // 通用主按钮使用墨黑，AI 专属行动使用 accentButton。
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: ink,
          foregroundColor: Colors.white,
          minimumSize: const Size.fromHeight(50),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(rMd),
          ),
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
        ),
      ),
      // 白底细边次按钮。
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: ink,
          backgroundColor: Colors.white,
          minimumSize: const Size.fromHeight(50),
          side: const BorderSide(color: border),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(rMd),
          ),
        ),
      ),
    );
  }

  /// AI 强调主按钮样式（对应小程序 .btn-accent）。
  /// 用法：ElevatedButton(style: AppTheme.accentButton, ...)
  static final ButtonStyle accentButton = ElevatedButton.styleFrom(
    backgroundColor: accent,
    foregroundColor: Colors.white,
    minimumSize: const Size.fromHeight(50),
    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(rMd)),
    textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
  );
}
