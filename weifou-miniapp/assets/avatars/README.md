# 静态图形象（古风等插画）

存放 `type:'image'` 形象的插画 PNG，支持按对话状态切表情图。

## 怎么加一个图形象

1. 准备图片（建议 **正方形、半身居中、透明或纯色背景、统一风格**，如 1024×1024 古风男/女）。
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
       thinking:'/assets/avatars/her_think.png',
       speaking:'/assets/avatars/her_speak.png',
     },
     colors:['#8b5cf6','#c4b5fd'] }   // colors 必填：图加载失败时回退渐变形象
   ```

## 行为
- 组件 `components/avatar` 对 image 形象：各状态图**叠放 + opacity 淡入淡出**切换；`state` 由调用方传入（聊天页 AI 回复时传 `speaking`）。
- 整体仍套用 CSS **呼吸/漂浮**活人感动效。
- 任一图加载失败 → 自动回退 css 渐变形象，**绝不留白/破图**。
- 只给 `idle` 一张也能用（纯静态半身 + CSS 微动）。

## 关于"衣服头发飘动"
状态切图给不了连续飘动。要"仙气飘动"，把该形象做成 **Lottie**（设计师在 AE 给头发/飘带 K 飘动帧），放 `assets/lottie/` 并改用 `type:'lottie'`——通路已就绪，无需改组件。

## 来源与合规
- 离线用绘图工具/模型产出固定形象库、当静态图打包 = 预制素材，**不触发运行时 AIGC 的备案/标识**；但要遵守所用工具的商用授权，确保你拥有图片使用权。
- 真人肖像需授权；避免名人/受版权保护的角色形象。
