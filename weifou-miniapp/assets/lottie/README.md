# Lottie 动态形象资源

存放设计师从 AE 导出的 Lottie `.json` 动画，作为"预制动态形象库"的素材。

## 怎么加一个 Lottie 形象

1. 准备 `.json`（AE + Bodymovin/Lottie 导出，**纯形状/矢量**，不要内嵌位图，体积更小、可循环）。
2. 二选一提供数据来源：
   - **打包进小程序**：把 json 放到本目录（如 `spark.json`），然后在 `utils/avatars.js` 顶部的 `LOTTIE_DATA` 里静态登记：
     ```js
     const LOTTIE_DATA = {
       'lottie-spark': require('../assets/lottie/spark.json'),
     };
     ```
     （小程序不支持动态 require，本地 json 必须这样显式登记。注意主包体积上限 2MB，多/大动画建议走分包或 COS。）
   - **放腾讯云 COS**：把 json 传到 COS，在 `utils/avatars.js` 的预设里把 `lottie` 写成 `https://<bucket>.cos.../spark.json`（域名需加入小程序 `request` 合法域名白名单）。运行时 `wx.request` 拉取。
3. 在 `utils/avatars.js` 的 `PRESETS` 里确认/新增对应预设项：
   ```js
   { id:'lottie-spark', name:'闪耀', type:'lottie', lottie:'/assets/lottie/spark.json', colors:['#f59e0b','#ec4899'], anim:'shine' }
   ```
   `colors` 必填——加载失败或未登记时，组件按它回退为 css 渐变形象，**绝不留白屏**。

## 行为说明
- 组件 `components/avatar` 对 `type:'lottie'` 的预设：可播放时用 canvas 播 Lottie；拿不到数据时自动回退 css 形象。
- 设置页的形象网格：仅**当前选中**的那个播放 Lottie，其余渲染静态 css 预览（省性能）。
- 主页 hero / 海报：海报为静态图，对 lottie 形象用其 `colors` 画静态渐变圆（与动态版同色）。

## 当前已内置的演示动画
- `spark.json` / `wave.json` 是**原创手写**的简易几何 Lottie（脉冲光环 + 呼吸光点 / 挤压水滴 + 高光），仅用于验证 Lottie 通路与观感，无第三方版权。已在 `utils/avatars.js` 的 `LOTTIE_DATA` 登记，对应预设 `lottie-spark` / `lottie-wave` 即时可播。
- 正式上线请替换为设计师产出的高质量动画（保持 200x200、纯矢量形状、可循环）。
