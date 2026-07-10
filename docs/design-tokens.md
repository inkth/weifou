# 微否设计令牌 · 双端真源

本文件是 **App（Flutter）+ 小程序** 视觉令牌的唯一真源。改令牌先改这里，再同步两端实现：

- 小程序：`weifou-miniapp/app.wxss` 顶部 `page { --* }`（已切换糖果风）
- App：`weifou-app/lib/core/theme/app_theme.dart` 的 `AppTheme` 常量（⚠ 仍是旧新粗野值，糖果化另行排期；App 保活暂缓期间以小程序为准）

设计原则（2026-07-02 用户拍板，替换此前的新粗野方向）：**糖果 Candy（多邻国式）**。纯白底 `#FFFFFF` + 浅灰细边 + **垂直 3D 底唇**（`box-shadow: 0 X 0 深一档同色`，无模糊）+ 品牌碧绿 `#18B690` 主色 + 明亮糖果色。按压反馈 = 糖果按下去（`translateY` 下沉 + 底唇收没）。全站含暗场沉浸页一并改亮（暗场词汇已退役）。红线不变：成交/支付/资质区克制——白卡灰边照用，糖果杂色不进。旧令牌名全部保留、只换值，历史页面零改动焕肤。

---

## 颜色令牌

### 中性色阶（多邻国灰阶：文字四级 + 线/边/填充）

| 语义 | 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|---|
| 主文字 | `--ink` / `ink` | `#4B4B4B` | 标题、正文（⚠ 不再兼任边框/投影色） |
| 次级文字 | `--ink-2` / `ink2` | `#6F6F6F` | 副标题、正文辅助 |
| 三级文字 | `--sub` / `sub` | `#999999` | 说明、占位、非活跃 |
| 四级文字 | `--faint` / `faint` | `#BCBCBC` | 时间戳、极弱信息 |
| 发丝线 | `--line` / `line` | `#F0F0F0` | 卡片内部弱分隔 |
| 边框 | `--border` / `border` | `#E5E5E5` | 卡片/输入统一浅灰边，写法 `var(--bw-thin) solid var(--border)` |
| 底唇/强边 | `--border-strong` | `#D8D8D8` | 卡片 3D 底唇、按压态边 |
| 描边宽 | `--bw` / `--bw-thin` | `3rpx` / `2rpx` | 保留阶值（多数场景用 thin） |
| 灰填充 | `--fill` / `fill` | `#F7F7F7` | 标签底、头像回退底、进度条轨 |
| 按下填充 | `--fill-pressed` / `fillPressed` | `#EFEFEF` | 单元格按下底 |

### 表面与强调

| 语义 | 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|---|
| 页面底色 | `--bg` / `bg` | `#FFFFFF` | scaffold 背景（纯白） |
| 暖纸底 | `--paper` / `paper` | `#FFFBF0` | 学习区页面级皮肤（learn-map / 学习日记）整页背景专用 |
| 卡片表面 | `--surface` / `surface` | `#FFFFFF` | 卡片靠 border+底唇与页面区分 |
| 凹陷面 | `--surface-sunken` / `surfaceSunken` | `#F7F7F7` | 内嵌区 / 输入底 |
| **强调色（品牌碧绿）** | `--accent` / `accent` | `#18B690` | 主 CTA、活跃态、点亮态 |
| 强调按下/底唇 | `--accent-strong` / `accentStrong` | `#0E9C7A` | 绿按钮的 3D 底唇色 |
| 强调深绿 | `--accent-deep` / `accentDeep` | `#0F766E` | soft 底上的图标 / 次级按钮文字 |
| 浅薄荷底 | `--accent-soft` / `accentSoft` | `#DCF6EC` | 推荐位/强调标签背景 |
| 绿浅底上文字 | `--accent-ink` / `accentInk` | `#0C5A48` | accent-soft 上的文字 |
| **accent 实底上文字** | `--on-accent` / `onAccent` | `#FFFFFF` | ⚠ 糖果风：绿底配白粗字（改自墨字） |
| 马克笔高亮 | `--hl` / `hl` | `#A9F0D8` | 标题关键词底条（`.hl` 类） |
| 次级 CTA 蓝 | `--cand-blue` / `-strong` | `#1CB0F6` / `#1899D6` | 次级行动、信息态 |
| 金冠 | `--gold` / `--gold-strong` | `#FFD335` / `#E6A817` | 已掌握 / 通关 / 成就 |
| 糖果色 | `--pop-butter/coral/mint/lilac/sky`（各配 `-ink`） | `#FFC800` `#FF9600` `#A8F0D4` `#E6CCFF` `#DDF4FF` | 角色/学习/庆祝层；成交区禁入 |
| 成功 / 警示 / 危险 | `--success(-strong)` `--warn` `--danger(-strong)` | `#58CC02/#46A302` `#FFC800` `#FF4B4B/#EA2B2B` | 状态 |
| 深林（**logo 专用**） | `--forest` | `#08312A` | 仅 logo 底色，**不进 UI**：满屏深色会稀释图标在微信列表里的辨识度 |

### 品牌标志

图形标是一株小盆栽：不对称碧绿树冠 + 珊瑚陶盆，立在深林底上。几何真源 `assets/brand/logo.svg`，
PNG 由 `node scripts/gen-logo.mjs` 生成（零依赖光栅器，iOS 图标输出无 alpha、无圆角）。

| 部位 | 令牌 | 值 |
|---|---|---|
| 底 | `--forest` | `#08312A` |
| 树冠 | `--accent` | `#18B690` |
| 陶盆 | `--pop-coral` | `#FF9600` |
| 掌握态金花 | `--gold` | `#FFD335` |

三条不可破的规则：**冠三圆不对称**（右肩高于左肩，这点歪就是指纹）；**冠底与盆沿之间留 3 单位缝**（单色/刻章只剩剪影时靠它读出"一棵东西长在盆里"，否则糊成葫芦）；**单色场合整体填 `--accent-ink`、绿底场合整体反白**，永不出现绿冠白盆这种半调子。
logo 层永远无脸、永远不动；眨眼、呼吸、打蔫、开花属于产品内的 IP 层与成长态。吉祥物不上用户名片——名片传播的是用户本人的脸。

### 高度 / 3D 底唇（垂直 offset 无模糊；层级=下沉量）

| 令牌 | 值 | 用途 |
|---|---|---|
| `--lip` / `--lip-lg` | `6rpx` / `8rpx` | 标准 / 大按钮下沉距离 |
| `--shadow-hair` | `0 3rpx 0 border` | chip / 气泡 / 小徽章 |
| `--shadow-card` | `0 5rpx 0 border` | **默认卡片** |
| `--shadow-lift` | `0 8rpx 0 border-strong` | hero 级大卡 |
| `--shadow-soft` | `0 5rpx 0 #CDEEE2` | 薄荷软唇（推荐位） |
| `--shadow-pop` | `0 16rpx 48rpx rgba(0,0,0,.14)` | 浮层（糖果风允许软阴影） |
| `--shadow-accent` | `0 8rpx 0 accent-strong` | 绿 CTA 底唇 |

**交互纪律**：按压 = 糖果下沉 —— `:active { transform: translateY(4~8rpx); box-shadow: none; }`（下沉量=底唇量）。hover-class 场景用全局 `.card-press`；通用封装 `.press-3d`。3D 按钮不描边（实色底+底唇即是形体）；白/灰件底唇用 `--border`，彩色件底唇用同色深一档。

**强调色使用**：绿是主色可放开当 CTA；糖果杂色（butter/coral/lilac/sky）只进角色/学习/庆祝层，一屏 ≤2 种。金 `--gold` 只表「已掌握/通关」成就语义。

> **会员/成交区**：白卡灰边 + 绿 CTA，禁糖果杂色与金色滥用。
>
> **暖纸底 `--paper`**：只做学习区整页背景（"翻开冒险手账"的世界感），护栏三条——控件层仍白底浅灰边+底唇、不加深描边；纸底不进成交区与全站框架；不成套引入棕色系。

### 空态 / 骨架屏 / 全局类

- 空态三层：`.empty-wrap` + `.empty-ic`（薄荷圆底）+ `.empty-title` / `.empty-sub`（+ 可选 `.empty-cta` 绿糖按钮）。
- 骨架屏：`.skel`（灰底 + 流光扫过）。
- `.hl` 马克笔高亮；`.eyebrow` 区段眉标；`.sticker`（-mint/-coral 变体）糖果胶囊徽章（不再旋转）；`.pill-gold` 金冠胶囊。

---

## 间距 / 圆角 / 字阶

**间距**（4 基数，`--sp-1..7`）：`8 / 16 / 24 / 32 / 40 / 56 / 72 rpx`（未变）。

**圆角阶**（糖果圆润，全面加大）：

| 令牌 | 值（rpx） | 用途 |
|---|---|---|
| `--r-sm` | 16 | 小标签、嵌套气泡角 |
| `--r-md` | 24 | 按钮、输入框、图标贴片 |
| `--r-lg` | 28 | 列表卡、气泡 |
| `--r-xl` | 32 | 标准卡片 |
| `--r-2xl` | 40 | 大容器 |
| `--r-3xl` | 48 | hero 大卡 / 底部抽屉顶角 |
| `--r-full` | 999 | 胶囊 / 头像 / chip |

**字阶**：`.t-hero`(56/800) · `.t-display`(46/800) · `.t-section`(32/800) · `.t-body`(28) · `.t-sub`(26, ink-2) · `.t-caption`(24, sub) · `.t-micro`(22, faint)。糖果风标题统一 800 大字重、不收字距。

按钮三类：
- `.btn-primary` = `.btn-accent` —— 绿底白 800 字 + 深绿底唇（糖果风主按钮就是绿）
- `.btn-ghost` —— 白底灰细边 + 灰底唇，文字 `--accent-deep` 700
- 大圆角矩形（`--r-md`），不再是胶囊

---

## 图标系统

线性图标（24 网格 / 2px 描边 / 圆头圆角 = Lucide 气质），**彻底取代 emoji 当功能图标**。

- 真源：`weifou-miniapp/styles/icons.wxss`，已在 `app.wxss` 顶部 `@import`。
- 实现：CSS `mask` + `background-color: currentColor`。
- 用法：`<view class="ic ic-bot" />`；尺寸用 `font-size`，颜色用 `color`。
- 自定义组件（如 `custom-tab-bar`）默认 `styleIsolation:isolated`，需单独 `@import`，且其色值为字面量——改令牌时必须手动同步该文件。
- 已有字形：`bot/chat/user/search/bell/crown/inbox/phone/edit/compose/stats/settings/sparkle/apps/mic/chevron/back/arrow/plus/close/check`。

> 例外：闯关地图节点（🔒⭐👑✨）与空态大 emoji 属插画性质，保留 emoji。

---

## 沉浸件（亮场玻璃）

原暗场玻璃令牌名保留、值已反转为亮场：`--glass-light` `rgba(255,255,255,.86)`、`--glass-dark` `rgba(255,255,255,.72)`（⚠ 名为 dark 实为白玻璃，历史消费方免改）、`--glass-hair` `rgba(75,75,75,.10)`、`--blur-glass` `blur(28rpx) saturate(160%)`。立绘兜底渐变（`styles/stage.wxss` `.art*`）已改糖果亮渐变（warm 绿 / cool 蓝 / lively 紫）。

## 温度档

跨行业沟通风格 3 档不变（cool/warm/lively），实现见 `weifou-miniapp/utils/avatars.js`。全局强调色不随档变（始终碧绿 #18B690）。档位差异体现在头像气质 + 开场白语气 + 立绘渐变色族。

---

## 维护约定

1. 改色值：先改本文件表格 → 再改 `app.wxss`（App 端 `app_theme.dart` 糖果化排期后同步），两端值必须一致。
2. 新增语义色 / 间距 / 圆角 / 阴影令牌：三处同步。
3. `custom-tab-bar/index.wxss` 色值是字面量（组件样式隔离），改令牌必须手动同步。
4. 新增图标：只动 `styles/icons.wxss`。
5. 头像资产升级（toon → image/lottie）：保持 preset `id` 不变即可无缝迁移。
