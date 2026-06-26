# 微否设计令牌 · 双端真源

本文件是 **App（Flutter）+ 小程序** 视觉令牌的唯一真源。改令牌先改这里，再同步两端实现：

- 小程序：`weifou-miniapp/app.wxss` 顶部 `page { --* }`
- App：`weifou-app/lib/core/theme/app_theme.dart` 的 `AppTheme` 常量

设计原则：**冷热分区**。墨黑 `#1F2330` 仍是主色与底盘色调；碧绿 `#18B690` 只做强调，绝不替换主色。成交关键区（支付 / 预约确认 / 主页资质）保持克制，不上卡通、不滥用强调色。

---

## 颜色令牌

### 中性色阶（同一冷蓝色相递进，文字四级 + 线/填充）

| 语义 | 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|---|
| 主文字 | `--ink` / `ink` | `#1F2330` | 标题、墨黑主按钮、用户气泡 |
| 次级文字 | `--ink-2` / `ink2` | `#4F5564` | 副标题、正文辅助 |
| 三级文字 | `--sub` / `sub` | `#8A8F9C` | 说明、占位、非活跃 |
| 四级文字 | `--faint` / `faint` | `#B4B9C4` | 时间戳、极弱信息 |
| 发丝线 | `--line` / `line` | `#EEF0F4` | 分隔线、内部描边 |
| 边框 | `--border` / `border` | `#E5E7EC` | 卡片/输入框边框 |
| 中性填充 | `--fill` / `fill` | `#F0F1F5` | 标签底、头像回退底 |
| 按下填充 | `--fill-pressed` / `fillPressed` | `#F6F7F9` | 单元格按下底 |

> 旧版散落的 `#8a90a0 / #9aa0ad / #6b7180 / #b3b8c2 / #c4c9d4 …` 一律收敛到上表四级文字；新代码禁止再引入近似灰。

### 表面与强调

| 语义 | 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|---|
| 页面底色 | `--bg` / `bg` | `#F5F6FA` | scaffold 背景 |
| 卡片表面 | `--surface` / `surface` | `#FFFFFF` | 卡片/浮层 |
| 凹陷面 | `--surface-sunken` / `surfaceSunken` | `#EEF0F5` | 内嵌区 |
| **强调色（碧绿）** | `--accent` / `accent` | `#18B690` | **仅** CTA 高亮、活跃态、强调图标 |
| 强调按下态 | `--accent-strong` / `accentStrong` | `#0E9C7A` | accent 元素 pressed |
| 浅绿底 | `--accent-soft` / `accentSoft` | `#E2F5EF` | 推荐位/强调标签背景、活跃指示胶囊 |
| 绿底上文字 | `--accent-ink` / `accentInk` | `#0C5A48` | accent-soft 上的文字 |
| 成功 / 警示 / 危险 | `--success` `--warn` `--danger` | `#16A34A` `#F59E0B` `#E0404B` | 状态（success 用草绿，与碧绿 accent 拉开） |

### 高度 / 阴影（分层、极淡、按需微暖）

| 令牌（小程序 / App） | 用途 |
|---|---|
| `--shadow-hair` | 列表项的极轻接触阴影 |
| `--shadow-card` / `cardShadow` | **默认卡片**：双层近中性，干净不脏 |
| `--shadow-soft` / `softShadow` | 绿柔阴影：推荐位 / hero 卡 |
| `--shadow-pop` | 浮层 / 弹层 |
| `--shadow-accent` / `accentShadow` | accent CTA 光晕（绿） |

**强调色使用红线**：accent 是「点睛」不是「主调」。一屏中带 accent 的实心块通常 ≤1 个（主 CTA / 发送键 / 活跃 tab / 选中分类）。底盘按钮、导航、普通卡片一律保持墨黑+灰。碧绿阴影只跟随 accent 实体，普通卡片用近中性 `--shadow-card`。

> **会员页例外**：付费/会员相关界面（`membership` / `agents` 的会员 banner）用「翡翠玉绿」高级渐变（`#2BC79E→#0E9C7A`）+ 深绿文字 `#0C5A48`，是 accent 同族的加深变体，做付费层的高级感差异；不要回退成暖金。

---

## 间距 / 圆角 / 字阶（弹性体系）

**间距**（4 基数，小程序 `--sp-1..7`）：`8 / 16 / 24 / 32 / 40 / 56 / 72 rpx`。新代码用阶值，不再随手写 18/22/26。

**圆角阶**（小程序 `--r-*` / App `AppTheme.r*`）：

| 令牌 | 值（rpx / dp） | 用途 |
|---|---|---|
| `--r-sm` | 12 / 12 | 小标签、嵌套气泡角 |
| `--r-md` | 16 / 16 | 图标贴片、输入框 |
| `--r-lg` | 20 / 20 | 列表卡、通知卡 |
| `--r-xl` | 24 / 24 | 标准卡片 |
| `--r-2xl` | 32 / 32 | 大容器 |
| `--r-full` | 999 | 胶囊按钮 / 头像 / chip |

**字阶**（小程序辅助类）：`.t-display`(46/700, 负字距) · `.t-section`(32/700) · `.t-body`(28) · `.t-sub`(26, ink-2) · `.t-caption`(24, sub) · `.t-micro`(22, faint)。大号粗标题统一 `letter-spacing:-0.5rpx`，更紧凑高级。

按钮三类：
- `.btn-primary` / 默认 `ElevatedButton` —— 墨黑，底盘通用主按钮；按下 `scale(0.975)`
- `.btn-accent` / `AppTheme.accentButton` —— 碧绿 + `--shadow-accent` 光晕，**仅关键转化点**（预约、立即开聊）
- `.btn-ghost` / `OutlinedButton` —— 白底描边，次级操作

---

## 图标系统

线性图标（24 网格 / 2px 描边 / 圆头圆角 = Lucide 气质），**彻底取代 emoji 当功能图标**。

- 真源：`weifou-miniapp/styles/icons.wxss`，已在 `app.wxss` 顶部 `@import`。
- 实现：CSS `mask` + `background-color: currentColor` —— 单字形随 `color` 重着色，任意尺寸清晰。
- 用法：`<view class="ic ic-bot" />`；尺寸用 `font-size`（图标取 `1em`），颜色用 `color`。
- 自定义组件（如 `custom-tab-bar`）默认 `styleIsolation:isolated`，需在组件 wxss 内单独 `@import "../styles/icons.wxss"`。
- 已有字形：`bot/chat/user/search/bell/crown/inbox/phone/edit/compose/stats/settings/sparkle/apps/mic/chevron/back/arrow/plus/close/check`。新增图标在此文件按同样 24 网格补一个 mask 即可。

> 例外：空状态里纯插画性质的大 emoji 可保留；凡是「可点/表意/导航」的图标一律用本系统。

---

## 温度档（弹性体系）

跨行业用户「都要覆盖」，故把沟通风格归到 3 档温度。**一档同时建议 头像气质 + 文案语气**；全局强调色不随档变（始终碧绿 #18B690），保证品牌一致——档位差异体现在「头像 + 开场白语气」这两个低风险、强可感知的点上。

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
2. 新增语义色 / 间距 / 圆角 / 阴影令牌：三处同步（本文件 + 两端）。
3. 新增图标：只动 `styles/icons.wxss`（小程序），App 端用 Flutter `Icon`/SVG 对应同名字形。
4. 头像资产升级（toon → image/lottie）：保持 preset `id` 不变即可无缝迁移，见 `utils/avatars.js` 注释与 `assets/avatars/README.md`。
