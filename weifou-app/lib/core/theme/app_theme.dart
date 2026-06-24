import 'package:flutter/material.dart';

/// 与小程序 app.wxss 一致的视觉规范：墨黑主色、浅灰底、胶囊按钮。
class AppTheme {
  AppTheme._();

  // 设计令牌（双端真源见 docs/design-tokens.md）。墨黑仍为主色，暖橙只做强调。
  // —— 中性色阶（同一冷蓝色相递进，统一旧版散落的灰）——
  static const Color ink = Color(0xFF1F2330); // 主文字 / 墨黑主按钮
  static const Color ink2 = Color(0xFF4F5564); // 次级文字：副标题、正文辅助
  static const Color sub = Color(0xFF8A8F9C); // 三级文字：说明、占位、非活跃
  static const Color faint = Color(0xFFB4B9C4); // 四级文字：时间戳、极弱信息
  static const Color line = Color(0xFFEEF0F4); // 发丝分隔线 / 内部描边
  static const Color border = Color(0xFFE5E7EC); // 卡片 / 输入框边框
  static const Color fill = Color(0xFFF0F1F5); // 中性填充：标签底 / 头像回退
  static const Color fillPressed = Color(0xFFF6F7F9); // 单元格按下底

  // —— 表面与底色 ——
  static const Color bg = Color(0xFFF5F6FA); // 页面底色
  static const Color surface = Color(0xFFFFFFFF); // 卡片 / 浮层表面
  static const Color surfaceSunken = Color(0xFFEEF0F5); // 内嵌凹陷区

  // —— 强调色（碧绿点睛，绝不替换主色）——
  static const Color accent = Color(0xFF18B690); // CTA 高亮 / 活跃态 / 强调
  static const Color accentStrong = Color(0xFF0E9C7A); // accent 按下态
  static const Color accentSoft = Color(0xFFE2F5EF); // 浅绿底：高亮区背景 / 标签
  static const Color accentInk = Color(0xFF0C5A48); // accentSoft 上的文字
  static const Color success = Color(0xFF16A34A); // 草绿，与碧绿 accent 拉开避免撞色
  static const Color warn = Color(0xFFF59E0B);
  static const Color danger = Color(0xFFE0404B);

  // —— 圆角阶（对应小程序 --r-*）——
  static const double rSm = 12, rMd = 16, rLg = 20, rXl = 24, r2xl = 32, rFull = 999;

  // —— 高度 / 阴影（极淡、分层、微暖；成交克制区不滥用）——
  // 默认卡片：分层近中性，对应小程序 --shadow-card
  static const List<BoxShadow> cardShadow = [
    BoxShadow(color: Color(0x0D1F2330), blurRadius: 9, offset: Offset(0, 2)),
    BoxShadow(color: Color(0x0A1F2330), blurRadius: 2, offset: Offset(0, 1)),
  ];
  // 绿柔阴影：推荐位 / hero 卡，对应小程序 --shadow-soft
  static const List<BoxShadow> softShadow = [
    BoxShadow(color: Color(0x1F18B690), blurRadius: 13, offset: Offset(0, 3)),
  ];
  // accent CTA 光晕，对应小程序 --shadow-accent
  static const List<BoxShadow> accentShadow = [
    BoxShadow(color: Color(0x5218B690), blurRadius: 12, offset: Offset(0, 5)),
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
