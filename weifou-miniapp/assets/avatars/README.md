# 静态图形象（古风等插画）

存放 `type:'image'` 形象的插画 PNG，支持按对话状态切表情图。

当前主推名片分身：

- `healing-boy_idle.webp`：治愈男孩，暖棕发、苔绿色外套。
- `healing-girl_idle.webp`：治愈女孩，暖棕发、杏粉色针织外套。

两张均为 540×810 透明背景 WebP，来自同一张原创角色设定图；高分辨率生成母版保存在顶层 `assets-src/avatars/raw/`，不会进入小程序主包。

## 怎么加一个图形象

1. 准备图片（建议 **540×810、半身居中、透明背景、统一风格**），单帧 WebP 尽量控制在 100KB 内。
   - 至少 1 张 `idle`（常驻）。
   - 想要对话感，再出 `speaking`（AI 回复时显示）、`thinking`（思考时，选填）。
   - 同一形象各状态图务必**同一张脸/服饰/配色**（用图生图从 idle 改神情，保一致）。
2. 二选一放置：
   - **打包**：放本目录（如 `her_idle.png`），预设里写 `/assets/avatars/her_idle.png`（注意主包 2MB 上限，多/大图建议走 COS 或分包）。
   - **COS**：传腾讯云 COS，预设里写 `https://<bucket>.cos.../her_idle.png`（域名加进小程序 `request` / `downloadFile` 合法域名白名单）。
3. 在 `utils/avatars.js` 的 `PRESETS` 里新增：
   ```js
   { id:'gf-her', name:'青衣', type:'image',
     images:{
       idle:'/assets/avatars/her_idle.png',
       blink:'/assets/avatars/her_blink.png',       // 名片/对话待机可用
       glance:'/assets/avatars/her_glance.png',     // 名片/对话待机可用
       thinking:'/assets/avatars/her_think.png',
       speaking:'/assets/avatars/her_speak.png',
     },
     colors:['#8b5cf6','#c4b5fd'] }   // colors 必填：图加载失败时回退渐变形象
   ```

## 行为
- 舞台对 image 形象：各状态图**叠放 + opacity 淡入淡出**切换；名片待机只用 `idle / blink / glance`，对话或课程才使用 `thinking / speaking`。
- 新用户不需要先判断，默认使用 `healing-girl`；用户主动切换后以所选形象为准。
- 非法或已下线的预设 id 只从非人脸气场中降级，不会随机套用另一个人物。
- 无 image 立绘时，身份舞台优先显示用户头像，否则显示姓名首字 + 个人气场。
- 整体仍套用 CSS **呼吸/漂浮**活人感动效。
- 任一图加载失败 → 自动回退 css 渐变形象，**绝不留白/破图**。
- 只给 `idle` 一张也能用（纯静态半身 + CSS 微动）。

## 关于"衣服头发飘动"
状态切图给不了连续飘动。要"仙气飘动"需要矢量补间动画（设计师在 AE 给头发/飘带 K 飘动帧，走 Lottie 一类方案）——项目曾内置 lottie-miniprogram 运行时，因长期未实际使用已移除；真有此需求时再引入，且建议放分包或 COS 远程加载，不占主包。

## 来源与合规
- 离线用绘图工具/模型产出固定形象库、当静态图打包 = 预制素材，**不触发运行时 AIGC 的备案/标识**；但要遵守所用工具的商用授权，确保你拥有图片使用权。
- 真人肖像需授权；避免名人/受版权保护的角色形象。
