// 预设动态形象库。
// type:
//   'css'(默认)  — 纯样式渐变 + 首字 + WXSS 动效（零资源、合规）
//   'image'      — 静态图形象（如古风插画），支持按舞台状态切表情图；图加载失败回退 css
// css 动效 anim: 'flow'(渐变流动) | 'pulse'(呼吸缩放) | 'shine'(高光扫过)
// 注意：colors 对 image 预设仍要填，作为未配置/加载失败时的回退色。
//
// image 形象示例（古风男女）——把图放 assets/avatars/ 或 COS，按下方格式加进 PRESETS：
//   { id:'gf-her', name:'青衣', type:'image',
//     images:{ idle:'/assets/avatars/her_idle.png',
//              blink:'/assets/avatars/her_blink.png',       // 选填，名片/对话待机共用
//              glance:'/assets/avatars/her_glance.png',     // 选填，名片/对话待机共用
//              thinking:'/assets/avatars/her_think.png',    // 选填，无则回落 idle
//              speaking:'/assets/avatars/her_speak.png' },  // 选填，AI 回复时显示
//     colors:['#8b5cf6','#c4b5fd'] }
//   组件会用 idle 图常驻、按 state 在各表情图间淡入淡出切换；只给 idle 一张也可用（纯静态 + CSS 呼吸漂浮）。
//   详见 assets/avatars/README.md。
const PRESETS = [
  // 主推虚拟分身：名片与课程舞台共用同一套角色身份。
  // 先用 idle + CSS 呼吸建立稳定识别，后续再按同一张脸补 blink / glance / speaking。
  { id: 'healing-boy', name: '治愈男孩', type: 'image', look: 'warm', images: { idle: '/assets/avatars/healing-boy_idle.webp' }, colors: ['#60705c', '#d6bd84'] },
  { id: 'healing-girl', name: '治愈女孩', type: 'image', look: 'warm', images: { idle: '/assets/avatars/healing-girl_idle.webp' }, colors: ['#b87468', '#edc8a5'] },

  { id: 'aurora', name: '极光', colors: ['#6366f1', '#a855f7', '#ec4899'], anim: 'flow' },
  { id: 'ocean', name: '深海', colors: ['#0ea5e9', '#2563eb'], anim: 'flow' },
  { id: 'mint', name: '薄荷', colors: ['#10b981', '#34d399'], anim: 'pulse' },
  { id: 'sunset', name: '日落', colors: ['#f59e0b', '#ef4444'], anim: 'flow' },
  { id: 'graphite', name: '石墨', colors: ['#374151', '#111827'], anim: 'shine' },
  { id: 'lavender', name: '薰衣草', colors: ['#8b5cf6', '#c4b5fd'], anim: 'pulse' },
  { id: 'coral', name: '珊瑚', colors: ['#fb7185', '#f43f5e'], anim: 'flow' },
  { id: 'forest', name: '森林', colors: ['#16a34a', '#065f46'], anim: 'shine' },

  // ↓ 历史可明确选中的虚拟立绘（type:'image'，见 scripts/gen-avatars.mjs）。
  //   包内帧应控制在 540px 宽、100KB 左右；表情帧增多后转 COS/分包，不挤占主包。
  { id: 'gf-meinv', name: '古风分身', type: 'image', images: { idle: '/assets/avatars/gf-meinv_idle.webp' }, colors: ['#9aa7c4', '#d8c7e0'] },

  // ↓ 历史 Lottie 示例槽位（Lottie 运行时已移除）。id 可能已被用户选中存库，
  //   保留条目按 css 渐变渲染（此前因无人加载动画数据，实际也一直走 css 回退，视觉无变化）。
  { id: 'lottie-spark', name: '闪耀', colors: ['#f59e0b', '#ec4899'], anim: 'shine' },
  { id: 'lottie-wave', name: '波动', colors: ['#0ea5e9', '#22d3ee'], anim: 'flow' },
];

// 新用户无需判断，直接看到完成度最高的治愈女孩；之后仍可主动切换其他形象。
// 非法/下线 id 仍只回退非人脸气场，避免把错误历史值误认成某个共享人物。
const DEFAULT_PRESET_ID = 'healing-girl';
const AUTO_PRESETS = PRESETS.filter((p) => p.type !== 'image');

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

// 取预设；空值代表尚未选择，使用产品默认；非法 id 只从非人脸气场中确定性兜底。
function getPreset(id, seed) {
  const found = PRESETS.find((p) => p.id === id);
  if (found) return found;
  if (!id) return PRESETS.find((p) => p.id === DEFAULT_PRESET_ID) || PRESETS[0];
  return AUTO_PRESETS[hashStr(seed) % AUTO_PRESETS.length] || PRESETS[0];
}

// 同一身份立绘的共享舞台协议。缺图帧一律返空，页面据此降级到用户头像/首字；
// 能力由数据明示，页面不再靠猜测文件名决定是否可以眨眼或说话。
function portraitStage(presetId, seed) {
  const preset = getPreset(presetId, seed);
  const images = preset.type === 'image' && preset.images ? preset.images : {};
  const frames = {
    idle: images.idle || '',
    blink: images.blink || '',
    glance: images.glance || '',
    thinking: images.thinking || '',
    speaking: images.speaking || '',
  };
  return {
    presetId: preset.id,
    kind: frames.idle ? 'illustration' : 'identity',
    label: frames.idle ? '虚拟分身形象' : '个人身份气场',
    frames,
    capabilities: {
      blink: !!frames.blink,
      glance: !!frames.glance,
      thinking: !!frames.thinking,
      speaking: !!frames.speaking,
    },
  };
}

function portraitFrames(presetId, seed) {
  return portraitStage(presetId, seed).frames;
}

// 新用户统一使用产品默认；保留 seed 参数以兼容已有调用方。
function pickDefault(seed) {
  void seed;
  return DEFAULT_PRESET_ID;
}

// 取姓名首字符（中文取首字，英文取首字母大写）
function initial(name) {
  const n = (name || '').trim();
  if (!n) return '微';
  return n[0].toUpperCase();
}

module.exports = {
  PRESETS, TONES, DEFAULT_TONE, DEFAULT_PRESET_ID,
  toneForStyle, tierForPreset, getPreset, portraitStage, portraitFrames, pickDefault, initial, hashStr,
};
