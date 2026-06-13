# 微否设计令牌 · 双端真源

本文件是 **App（Flutter）+ 小程序** 视觉令牌的唯一真源。改令牌先改这里，再同步两端实现：

- 小程序：`weifou-miniapp/app.wxss` 顶部 `page { --* }`
- App：`weifou-app/lib/core/theme/app_theme.dart` 的 `AppTheme` 常量

设计原则：**冷热分区**。墨黑 `#1F2330` 仍是主色与底盘色调；暖橙 `#FB923C` 只做强调，绝不替换主色。成交关键区（支付 / 预约确认 / 主页资质）保持克制，不上卡通、不滥用暖色。

---

## 颜色令牌

| 语义 | 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|---|
| 主色 / 文字 | `--ink` / `ink` | `#1F2330` | 文字、墨黑主按钮、用户气泡 |
| 页面底色 | `--bg` / `bg` | `#F5F6FA` | scaffold 背景 |
| 描边 | `--border` / `border` | `#E5E7EC` | 卡片/输入框边框、分隔线 |
| 次要文字 | `--sub` / `sub` | `#8A8F9C` | 辅助说明 |
| **强调色** | `--accent` / `accent` | `#FB923C` | **仅** CTA 高亮、活跃态、强调图标 |
| 强调按下态 | `--accent-strong` / `accentStrong` | `#EF7D1F` | accent 元素的 hover/pressed |
| 浅暖底 | `--accent-soft` / `accentSoft` | `#FFF3E9` | 推荐位/强调标签背景 |
| 暖底上文字 | `--accent-ink` / `accentInk` | `#9A4D12` | accent-soft 上的文字 |
| 成功 | `--success` / `success` | `#10B981` | 完成态、正向反馈 |
| 警示 | `--warn` / `warn` | `#F59E0B` | 提醒 |
| 危险 | `--danger` / `danger` | `#E0404B` | 删除、错误、挂断 |

**阴影**：`--shadow-soft` / `AppTheme.softShadow` = 柔和暖阴影（`rgba(249,140,60,0.10)`），替代旧的冷灰阴影，用于卡片/浮层。

**强调色使用红线**：accent 是「点睛」不是「主调」。一屏中带 accent 的实心块通常 ≤1 个（主 CTA / 发送键）。底盘按钮、导航、普通卡片一律保持墨黑+灰。

---

## 圆角 / 按钮

| 元素 | 圆角 | 说明 |
|---|---|---|
| 主按钮 / 标签 / 头像 | `100rpx` / `100` 胶囊 | 沿用现状 |
| 卡片 | `24rpx` / `16` | 沿用现状 |
| 聊天气泡 | `24rpx`，发送角 `8rpx` | 沿用现状 |

按钮三类：
- `.btn-primary` / 默认 `ElevatedButton` —— 墨黑，底盘通用主按钮
- `.btn-accent` / `AppTheme.accentButton` —— 暖橙，**仅关键转化点**（预约、立即开聊）
- `.btn-ghost` / `OutlinedButton` —— 白底描边，次级操作

---

## 温度档（弹性体系）

跨行业用户「都要覆盖」，故把沟通风格归到 3 档温度。**一档同时建议 头像气质 + 文案语气**；全局强调色不随档变（始终暖橙），保证品牌一致——档位差异体现在「头像 + 开场白语气」这两个低风险、强可感知的点上。

实现见 `weifou-miniapp/utils/avatars.js` 的 `TONES` / `toneForStyle(style)`。沟通风格 `style` 白名单与服务端 `internal/persona/persona.go` 的 `StyleDescriptions` 同步维护。

| 档 | 沟通风格(style) | 适配行业 | 头像气质(toon look / 渐变) | 文案语气 |
|---|---|---|---|---|
| 专业冷静 `cool` | steady / sharp | 顾问·律师·财务·医美 | toon-steady / 石墨·深海 | 严谨克制，先结论后依据 |
| **中性亲和 `warm`（默认）** | warm | 大多数·生活服务 | toon-warm / 珊瑚·日落 | 友好专业，口语化、先共情 |
| 活泼年轻 `lively` | humorous | 创作者·网红·IP | toon-humorous / 极光·薄荷 | 轻松有趣，适度玩笑不油腻 |

风格为空 / 未知时回落到默认档 `warm`（`DEFAULT_TONE`）。

---

## 维护约定

1. 改色值：先改本文件表格 → 再改 `app.wxss` 与 `app_theme.dart`，两端值必须一致。
2. 新增语义色：三处同步（本文件 + 两端）。
3. 头像资产升级（toon → image/lottie）：保持 preset `id` 不变即可无缝迁移，见 `utils/avatars.js` 注释与 `assets/avatars/README.md`。
