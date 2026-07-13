// 预设动态形象库。
// type:
//   'css'(默认)  — 纯样式渐变 + 首字 + WXSS 动效（零资源、合规）
//   'image'      — 静态图形象（如古风插画），支持按对话状态切表情图；图加载失败回退 css
// css 动效 anim: 'flow'(渐变流动) | 'pulse'(呼吸缩放) | 'shine'(高光扫过)
// 注意：colors 对 image 预设仍要填，作为未配置/加载失败时的回退色。
//
// image 形象示例（古风男女）——把图放 assets/avatars/ 或 COS，按下方格式加进 PRESETS：
//   { id:'gf-her', name:'青衣', type:'image',
//     images:{ idle:'/assets/avatars/her_idle.png',
//              thinking:'/assets/avatars/her_think.png',   // 选填，无则回落 idle
//              speaking:'/assets/avatars/her_speak.png' },  // 选填，AI 回复时显示
//     colors:['#8b5cf6','#c4b5fd'] }
//   组件会用 idle 图常驻、按 state 在各表情图间淡入淡出切换；只给 idle 一张也可用（纯静态 + CSS 呼吸漂浮）。
//   详见 assets/avatars/README.md。
const PRESETS = [
  { id: 'aurora', name: '极光', colors: ['#6366f1', '#a855f7', '#ec4899'], anim: 'flow' },
  { id: 'ocean', name: '深海', colors: ['#0ea5e9', '#2563eb'], anim: 'flow' },
  { id: 'mint', name: '薄荷', colors: ['#10b981', '#34d399'], anim: 'pulse' },
  { id: 'sunset', name: '日落', colors: ['#f59e0b', '#ef4444'], anim: 'flow' },
  { id: 'graphite', name: '石墨', colors: ['#374151', '#111827'], anim: 'shine' },
  { id: 'lavender', name: '薰衣草', colors: ['#8b5cf6', '#c4b5fd'], anim: 'pulse' },
  { id: 'coral', name: '珊瑚', colors: ['#fb7185', '#f43f5e'], anim: 'flow' },
  { id: 'forest', name: '森林', colors: ['#16a34a', '#065f46'], anim: 'shine' },

  // ↓ 全屏立绘形象（type:'image'，fal 生成，见 scripts/gen-avatars.mjs）；对话页据此走"星野式全屏立绘"模式。
  //   ⚠️ 单图 ~2.8MB，上线务必改 COS（images 里写 https 链接）；本地测试用包内路径即可。
  { id: 'gf-meinv', name: '古风美女', type: 'image', images: { idle: '/assets/avatars/gf-meinv_idle.webp' }, colors: ['#9aa7c4', '#d8c7e0'] },

  // ↓ 历史 Lottie 示例槽位（Lottie 运行时已移除）。id 可能已被用户选中存库，
  //   保留条目按 css 渐变渲染（此前因无人加载动画数据，实际也一直走 css 回退，视觉无变化）。
  { id: 'lottie-spark', name: '闪耀', colors: ['#f59e0b', '#ec4899'], anim: 'shine' },
  { id: 'lottie-wave', name: '波动', colors: ['#0ea5e9', '#22d3ee'], anim: 'flow' },
];

// 默认全屏立绘（所有场景统一的立绘背景）。暂时全员共用 gf-meinv 这一张；
// 将来做成可选立绘库时，每人 profile 自带 image 形象即覆盖此默认——只改这一处。
const _meinv = PRESETS.find((p) => p.id === 'gf-meinv');
const DEFAULT_LIHE = (_meinv && _meinv.images && _meinv.images.idle) || '/assets/avatars/gf-meinv_idle.webp';

// 温度档（弹性体系核心，定义见 docs/design-tokens.md）。
// 把 4 种沟通风格归到 3 档；一档同时建议 头像气质(look/渐变) + 文案语气(tone)。
// 注意：全局强调色固定为雾蓝紫 #7772c8（见 app.wxss --accent），不随档变，保证品牌一致；
//       档位差异体现在「头像 + 开场白语气」上，这是低风险、可感知的升温点。
const TONES = {
  cool: {
    id: 'cool', name: '专业冷静',
    styles: ['steady', 'sharp'],     // 顾问 / 律师 / 财务 / 医美
    look: 'steady', avatars: ['graphite', 'ocean'],
    tone: '严谨克制，先结论后依据，不寒暄',
  },
  warm: {
    id: 'warm', name: '中性亲和',
    styles: ['warm'],                 // 大多数 / 生活服务（默认档）
    look: 'warm', avatars: ['coral', 'sunset'],
    tone: '友好专业，口语化，先共情再答',
  },
  lively: {
    id: 'lively', name: '活泼年轻',
    styles: ['humorous'],             // 创作者 / 网红 / IP
    look: 'humorous', avatars: ['aurora', 'mint'],
    tone: '轻松有趣，可适度玩笑，不油腻',
  },
};
const DEFAULT_TONE = 'warm';

// 由沟通风格取温度档；风格为空/未知时回落到默认档（中性亲和）。
function toneForStyle(style) {
  const hit = Object.keys(TONES).find((k) => TONES[k].styles.indexOf(style) >= 0);
  return TONES[hit || DEFAULT_TONE];
}

// 由形象预设 id 反推温度档（对话舞台氛围用）：
// 预设带 look → 复用 toneForStyle；渐变预设按 TONES.avatars 命中表反查；都落空回退默认档 warm。
function tierForPreset(presetId, seed) {
  const p = getPreset(presetId, seed);
  if (p.look) return toneForStyle(p.look);
  const hit = Object.keys(TONES).find((k) => TONES[k].avatars.indexOf(p.id) >= 0);
  return TONES[hit || DEFAULT_TONE];
}

function hashStr(s) {
  let h = 0;
  const str = String(s || '');
  for (let i = 0; i < str.length; i++) {
    h = (h * 31 + str.charCodeAt(i)) >>> 0;
  }
  return h;
}

// 取预设；id 为空或非法时用 seed 确定性兜底
function getPreset(id, seed) {
  const found = PRESETS.find((p) => p.id === id);
  if (found) return found;
  return PRESETS[hashStr(seed) % PRESETS.length];
}

// 由 seed 选默认形象 id
function pickDefault(seed) {
  return PRESETS[hashStr(seed) % PRESETS.length].id;
}

// 取姓名首字符（中文取首字，英文取首字母大写）
function initial(name) {
  const n = (name || '').trim();
  if (!n) return '微';
  return n[0].toUpperCase();
}

module.exports = { PRESETS, TONES, DEFAULT_TONE, DEFAULT_LIHE, toneForStyle, tierForPreset, getPreset, pickDefault, initial, hashStr };
