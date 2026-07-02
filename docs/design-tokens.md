# 微否设计令牌 · 双端真源

本文件是 **App（Flutter）+ 小程序** 视觉令牌的唯一真源。改令牌先改这里，再同步两端实现：

- 小程序：`weifou-miniapp/app.wxss` 顶部 `page { --* }`
- App：`weifou-app/lib/core/theme/app_theme.dart` 的 `AppTheme` 常量

设计原则（2026-07-02 用户拍板）：**新粗野 Neo-brutalist**。奶油纸底 `#FDFBF2` + 墨青粗描边（`--bw` 3rpx）+ 硬偏移投影（offset 无模糊）+ 薄荷 `#18B690` 点睛 + 糖果贴纸色点缀。一切"柔"的手段（渐变 / 模糊阴影 / 半透明浮层）在浅色底盘退场，深度只靠 offset 硬投影表达；按压反馈 = 图章按下去（translate 位移 + 投影收没）。红线不变：成交/支付/资质区克制——描边照用，但贴纸 / 旋转 / 糖果色不进；暗场沉浸皮肤（登录/创建/立绘对话/回放）是独立词汇，不参与粗野化。

---

## 颜色令牌

### 中性色阶（青温灰：同一绿相递进，文字四级 + 线/填充）

| 语义 | 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|---|
| 主文字/描边/投影 | `--ink` / `ink` | `#16241E` | 标题、粗描边、硬投影、墨青主按钮 |
| 次级文字 | `--ink-2` / `ink2` | `#4A5A53` | 副标题、正文辅助 |
| 三级文字 | `--sub` / `sub` | `#8A9A93` | 说明、占位、非活跃 |
| 四级文字 | `--faint` / `faint` | `#AAB4AF` | 时间戳、极弱信息 |
| 纸感发丝线 | `--line` / `line` | `#EFEDE2` | 卡片内部弱分隔 |
| 边框 | `--border` / `border` | `#16241E` | ⚠ 语义=墨青描边色；写法用 `var(--bw) solid var(--ink)` |
| 描边宽 | `--bw` / `--bw-thin` | `3rpx` / `2rpx` | 卡片·按钮·输入框 / chip·徽章·头像框 |
| 奶油填充 | `--fill` / `fill` | `#F4F1E4` | 标签底、头像回退底 |
| 按下填充 | `--fill-pressed` / `fillPressed` | `#F7F5EA` | 单元格按下底 |

> 旧版冷蓝灰阶（`#1F2330 / #8A8F9C / #B4B9C4 / #8a90a0 / #9aa0ad …`）已于 2026-07 整体收敛到本青温阶；新代码禁止再引入近似灰，一律取上表四级。

### 表面与强调

| 语义 | 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|---|
| 页面底色 | `--bg` / `bg` | `#FDFBF2` | scaffold 背景（奶油纸） |
| 卡片表面 | `--surface` / `surface` | `#FFFFFF` | 卡片/浮层 |
| 凹陷面 | `--surface-sunken` / `surfaceSunken` | `#F7F5EA` | 内嵌区 / 输入底 |
| **强调色（薄荷）** | `--accent` / `accent` | `#18B690` | **仅** CTA 高亮、活跃态、强调图标 |
| 强调按下态 | `--accent-strong` / `accentStrong` | `#0E9C7A` | accent 元素 pressed / 浅底小字 |
| 强调深绿 | `--accent-deep` / `accentDeep` | `#0F766E` | accent-soft 底上的图标 / 强化小字 |
| 浅薄荷底 | `--accent-soft` / `accentSoft` | `#E8F5EF` | 推荐位/数据片/强调标签背景 |
| 绿底上文字 | `--accent-ink` / `accentInk` | `#0C5A48` | accent-soft 上的文字 |
| **accent 实底上墨字** | `--on-accent` / `onAccent` | `#06251C` | 薄荷实底配墨字，不配白字（粗野惯例） |
| 马克笔高亮 | `--hl` / `hl` | `#7CE3C4` | 标题关键词底条（`.hl` 类） |
| 糖果贴纸色 | `--pop-butter/coral/mint/lilac`（各配 `-ink` 文字色） | `#FFD666` `#FFC9B5` `#B7EBD9` `#D8CBF6` | **仅角色/学习/庆祝层**；成交区禁入 |
| 成功 / 警示 / 危险 | `--success` `--warn` `--danger` | `#16A34A` `#F59E0B` `#E0404B` | 状态 |

### 高度 / 硬投影（offset 无模糊；层级=位移量）

| 令牌（小程序 / App） | 值 | 用途 |
|---|---|---|
| `--shadow-hair` | `2rpx 2rpx 0 ink` | chip / 气泡 / 小徽章 |
| `--shadow-card` / `cardShadow` | `5rpx 5rpx 0 ink` | **默认卡片 / 按钮** |
| `--shadow-lift` | `9rpx 9rpx 0 ink` | hero 级大卡 |
| `--shadow-soft` / `softShadow` | `5rpx 5rpx 0 #CBE7DA` | 薄荷软投影（少用） |
| `--shadow-pop` | `12rpx 12rpx 0 ink` | 浮层 / 弹层 |
| `--shadow-accent` | `5rpx 5rpx 0 ink` | accent CTA（同吃墨投影） |

**交互纪律**：按压 = 图章落下 —— `:active { transform: translate(4~5rpx, 4~5rpx); box-shadow: 0 0 0 var(--ink); }`（位移量=投影量，视觉上是"按进纸里"）。hover-class 场景用全局 `.card-press`。禁止带模糊的 box-shadow 和 linear-gradient 出现在浅色底盘（暗场沉浸皮肤除外）。

**强调色使用红线**：accent 是「点睛」不是「主调」，一屏 accent 实心块 ≤1 个。accent 实底必配 `--on-accent` 墨字 + 墨描边。贴纸 `.sticker`（微旋转描边胶囊）一屏 ≤2 张。

> **会员页**：付费场景描边+硬投影照用，但禁贴纸/旋转/糖果色；会员卡主视觉 = accent 实底（不渐变）+ 描边 + `--shadow-lift`。

### 空态 / 骨架屏 / 创意原子（app.wxss 全局类）

- 空态三层：`.empty-wrap` + `.empty-ic`（描边圆底、微旋转）+ `.empty-title` / `.empty-sub`（+ 可选 `.empty-cta`）。
- 骨架屏：`.skel`（奶油底 + 流光扫过），与真实卡片同构占位。
- `.hl` 马克笔高亮包标题关键词；`.eyebrow` 区段眉标；`.cutline` / `dashed rgba(22,36,30,0.2)` 剪裁虚线做内部分隔；`.sticker`（-mint/-coral 变体）贴纸徽章。

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
| `--r-3xl` | 36 / 36 | 首页 hero 大卡 |
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
