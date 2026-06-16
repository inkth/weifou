#!/usr/bin/env node
/**
 * 用 fal.ai 批量产出「猫箱/星野式全屏角色立绘」（竖版高清、透明背景）。
 * 模型：默认 fal-ai/gpt-image-1（实测能出真 alpha 透明）；openai/gpt-image-2 会"假透明"(RGB 画个底)，故不用。fal 计费，仅需 FALAI_API_KEY。
 * 透明：传 background:"transparent"（实测 gpt-image-1 能出真 alpha）。若某模型不认（出 RGB 假透明）→ 回退 gpt-image-1 或接 remove-bg。
 *
 * 用法（key 在 weifou-server/.env，用 Node 的 --env-file 注入）：
 *   node --env-file=weifou-server/.env scripts/gen-avatars.mjs            # 全部角色，仅 idle
 *   node --env-file=weifou-server/.env scripts/gen-avatars.mjs --states   # 含说话/思考（图生图编辑，保一致）
 *   node --env-file=weifou-server/.env scripts/gen-avatars.mjs gf-meinv
 *
 * 环境变量：FALAI_API_KEY（必需）；FAL_IMAGE_MODEL / FAL_EDIT_MODEL / FAL_IMAGE_SIZE 可覆盖。
 * ⚠️ 全屏大图勿入 2MB 主包，建议改存腾讯云 COS（预设里写 https 链接 + 加 downloadFile 合法域名）。
 */
import { writeFile, mkdir, unlink } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const FAL_KEY = process.env.FALAI_API_KEY;
if (!FAL_KEY) {
  console.error('✗ 缺少 FALAI_API_KEY。用法: node --env-file=weifou-server/.env scripts/gen-avatars.mjs [角色id] [--states]');
  process.exit(1);
}

// fal 上的 GPT Image 模型（用户要 gpt-image-2；404 时自动回退 gpt-image-1）
const T2I_MODEL = process.env.FAL_IMAGE_MODEL || 'fal-ai/gpt-image-1/text-to-image'; // 出真 alpha 透明
const EDIT_MODEL = process.env.FAL_EDIT_MODEL || 'fal-ai/gpt-image-1/edit';
const IMAGE_SIZE = process.env.FAL_IMAGE_SIZE || '1024x1536'; // gpt-image-1 竖版枚举，长边 1536
const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT_DIR = join(__dirname, '..', 'weifou-miniapp', 'assets', 'avatars');
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// 全屏立绘的统一构图/画风（透明背景由 background 参数实现，prompt 只管人物与构图）
const STYLE = [
  '精致中式国风人物立绘（七分身偏全身），细节丰富、柔和厚涂、高质感、电影感打光；',
  '竖版全屏构图，人物居中略偏上、视线朝向观者，画面下方留出干净空间以便叠加对话气泡；',
  '背景纯透明（PNG 透明通道，无背景/地面/投影），主体干净抠像；不要任何文字、水印、logo、边框、签名；',
  '非真人写实、非名人、无版权角色。',
].join('');

const CHARACTERS = [
  { id: 'gf-meinv', name: '古风美女', prompt: `一位中式古风绝色美人的竖版全屏立绘（七分身）：乌发高髻斜簪步摇珠钗，淡施粉黛、眉目如画、眼神温柔含情；身着月白配淡青齐胸襦裙、轻纱披帛随风轻扬，身姿优雅、亭亭玉立、自然浅笑、目视观者。${STYLE}`, colors: ['#9aa7c4', '#d8c7e0'] },
  // 需要更多角色解开即可：
  // { id: 'gf-shutong', name: '书童', prompt: `一位中式古风少年「书童」：束发小髻，月白色交领长衫，眉清目秀，平和浅笑。${STYLE}`, colors: ['#7c8b9c', '#aebccd'] },
  // { id: 'gf-zhanggui', name: '掌柜', prompt: `一位中式古风「掌柜」中年男子：圆脸和善，深蓝对襟长褂，拱手作揖、温暖的笑。${STYLE}`, colors: ['#8a5a3c', '#caa07a'] },
];

// 状态变体：基于 idle 图生图（edit），强约束「只改表情、其余完全一致」
const STATE_EDITS = {
  speaking: '在保持人物五官、发型、服饰、配色、画风、构图与透明背景完全一致的前提下，只把表情改为「正在说话」：嘴自然张开、眼神看向观众、神态生动。',
  thinking: '在保持人物五官、发型、服饰、配色、画风、构图与透明背景完全一致的前提下，只把表情改为「正在思考」：目光微微上瞟、嘴唇轻抿、若有所思。',
};

// 调 fal 同步接口（fal.run 会阻塞到出图）。404 且用的是 gpt-image-2 时自动回退 gpt-image-1。
async function falRun(model, input, { tries = 2, allowFallback = true } = {}) {
  let lastErr;
  for (let i = 1; i <= tries; i++) {
    try {
      const res = await fetch(`https://fal.run/${model}`, {
        method: 'POST',
        headers: { Authorization: `Key ${FAL_KEY}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        const body = await res.text();
        if (res.status === 429 || res.status >= 500) throw Object.assign(new Error(`HTTP ${res.status}: ${body.slice(0, 200)}`), { retry: true });
        throw new Error(`HTTP ${res.status}: ${body.slice(0, 400)}`);
      }
      const data = await res.json();
      const url = data?.images?.[0]?.url;
      if (!url) throw new Error('返回里没有图片，片段: ' + JSON.stringify(data).slice(0, 400));
      return url;
    } catch (e) {
      lastErr = e;
      if (!e.retry || i === tries) break;
      console.warn(`  …第 ${i} 次失败（${e.message}），重试`);
      await sleep(1500 * i);
    }
  }
  throw lastErr;
}

async function download(url) {
  const r = await fetch(url);
  if (!r.ok) throw new Error(`下载图片失败 HTTP ${r.status}`);
  return Buffer.from(await r.arrayBuffer());
}

// 有 cwebp（brew install webp）则把出图压成 webp（保 alpha、约 1/7 体积）；否则留 png。
const HAS_CWEBP = (() => { try { execFileSync('cwebp', ['-version'], { stdio: 'ignore' }); return true; } catch (e) { return false; } })();
const EXT = HAS_CWEBP ? 'webp' : 'png';

// 存图：先落 fal 的 png，有 cwebp 则压成 webp 并删掉中间 png（保 alpha）
async function saveImg(buf, id, state) {
  const png = join(OUT_DIR, `${id}_${state}.png`);
  await writeFile(png, buf);
  if (!HAS_CWEBP) return png;
  const webp = join(OUT_DIR, `${id}_${state}.webp`);
  execFileSync('cwebp', ['-q', '82', '-alpha_q', '100', png, '-o', webp], { stdio: 'ignore' });
  await unlink(png);
  return webp;
}

async function genCharacter(ch, withStates) {
  console.log(`\n=== ${ch.name}（${ch.id}）===`);
  process.stdout.write('  生成 idle … ');
  const idleUrl = await falRun(T2I_MODEL, {
    prompt: ch.prompt, image_size: IMAGE_SIZE, background: 'transparent',
    quality: 'high', output_format: 'png', num_images: 1,
  });
  await saveImg(await download(idleUrl), ch.id, 'idle');
  console.log(`✓ ${ch.id}_idle.${EXT}`);

  if (!withStates) return;
  for (const [state, instr] of Object.entries(STATE_EDITS)) {
    process.stdout.write(`  生成 ${state} … `);
    const url = await falRun(EDIT_MODEL, {
      prompt: instr, image_urls: [idleUrl], background: 'transparent',
      quality: 'high', output_format: 'png', num_images: 1,
    });
    await saveImg(await download(url), ch.id, state);
    console.log(`✓ ${ch.id}_${state}.${EXT}`);
  }
}

async function main() {
  const args = process.argv.slice(2);
  const withStates = args.includes('--states');
  const only = args.find((a) => !a.startsWith('--'));
  const list = only ? CHARACTERS.filter((c) => c.id === only) : CHARACTERS;
  if (!list.length) {
    console.error(`✗ 找不到角色「${only}」。可用：${CHARACTERS.map((c) => c.id).join(', ')}`);
    process.exit(1);
  }

  await mkdir(OUT_DIR, { recursive: true });
  console.log(`模型: ${T2I_MODEL} · 尺寸: ${IMAGE_SIZE} · 透明背景\n输出: ${OUT_DIR}\n角色: ${list.map((c) => c.id).join(', ')}${withStates ? '（含 speaking/thinking）' : '（仅 idle）'}`);

  const ok = [];
  for (const ch of list) {
    try { await genCharacter(ch, withStates); ok.push(ch); }
    catch (e) { console.error(`  ✗ ${ch.id} 失败：${e.message}`); }
  }
  if (!ok.length) { console.error('\n全部失败。检查 FALAI_API_KEY / 模型 slug / fal 余额。'); process.exit(1); }

  console.log('\n完成。确认图后粘进 weifou-miniapp/utils/avatars.js 的 PRESETS（全屏大图建议改 COS）：\n');
  for (const ch of ok) {
    const imgs = withStates
      ? `{ idle:'/assets/avatars/${ch.id}_idle.${EXT}', speaking:'/assets/avatars/${ch.id}_speaking.${EXT}', thinking:'/assets/avatars/${ch.id}_thinking.${EXT}' }`
      : `{ idle:'/assets/avatars/${ch.id}_idle.${EXT}' }`;
    console.log(`  { id:'${ch.id}', name:'${ch.name}', type:'image', images:${imgs}, colors:${JSON.stringify(ch.colors)} },`);
  }
}

main().catch((e) => { console.error('脚本异常：', e); process.exit(1); });
