#!/usr/bin/env node
/** gen-avatars 迁移版：①防同款脸 ②--force/--dry-run+幂等 ③png母版转raw/ */
import { writeFile, mkdir, rename } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const FAL_KEY = process.env.FALAI_API_KEY;
if (!FAL_KEY) { console.error('✗ 缺少 FALAI_API_KEY'); process.exit(1); }

const T2I_MODEL = process.env.FAL_IMAGE_MODEL || 'fal-ai/gpt-image-1/text-to-image';
// 端点是 /edit-image，不是 /edit（后者 404：Path /edit not found，--states 分支曾因此全灭）
const EDIT_MODEL = process.env.FAL_EDIT_MODEL || 'fal-ai/gpt-image-1/edit-image';
const IMAGE_SIZE = process.env.FAL_IMAGE_SIZE || '1024x1536';
const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT_DIR = join(__dirname, '..', 'weifou-miniapp', 'assets', 'avatars');
const RAW_DIR = join(OUT_DIR, 'raw');
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const FORCE = process.argv.includes('--force');
const DRY_RUN = process.argv.includes('--dry-run');

// ① 只放与长相无关的不变量；具体长相写进各角色 prompt，避免 AI 同款脸
const STYLE = [
  '精致中式国风人物立绘（七分身偏全身），细节丰富、柔和厚涂、高质感、电影感打光；',
  '竖版全屏构图，人物居中略偏上、视线朝向观者，画面下方留出干净空间以便叠加对话气泡；',
  '背景纯透明（PNG 透明通道，无背景/地面/投影），主体干净抠像；不要任何文字、水印、logo、边框、签名；',
  '五官清晰、有个人辨识度，避免程式化的 AI 通用脸；',
  '非真人写实、非名人、无版权角色。',
].join('');

// ① 各角色 prompt 尽量覆盖：年龄感+脸型+眼型+眉形+瞳色+发型发色+妆容+服饰气质+可选标记
const CHARACTERS = [
  { id: 'gf-meinv', name: '古风美女', prompt: `一位中式古风绝色美人的竖版全屏立绘（七分身）：乌发高髻斜簪步摇珠钗，淡施粉黛、眉目如画、眼神温柔含情；身着月白配淡青齐胸襦裙、轻纱披帛随风轻扬，身姿优雅、亭亭玉立、自然浅笑、目视观者。${STYLE}`, colors: ['#9aa7c4', '#d8c7e0'] },
];

const STATE_EDITS = {
  speaking: '在保持人物五官、发型、服饰、配色、画风、构图与透明背景完全一致的前提下，只把表情改为「正在说话」：嘴自然张开、眼神看向观众、神态生动。',
  thinking: '在保持人物五官、发型、服饰、配色、画风、构图与透明背景完全一致的前提下，只把表情改为「正在思考」：目光微微上瞟、嘴唇轻抿、若有所思。',
};

async function falRun(model, input, { tries = 2 } = {}) {
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
        if (res.status === 429 || res.status >= 500) throw Object.assign(new Error(`HTTP ${res.status}`), { retry: true });
        throw new Error(`HTTP ${res.status}: ${body.slice(0, 400)}`);
      }
      const data = await res.json();
      const url = data?.images?.[0]?.url;
      if (!url) throw new Error('返回里没有图片');
      return url;
    } catch (e) {
      lastErr = e;
      if (!e.retry || i === tries) break;
      console.warn(`  …第 ${i} 次失败，重试`);
      await sleep(1500 * i);
    }
  }
  throw lastErr;
}

async function download(url) {
  const r = await fetch(url);
  if (!r.ok) throw new Error(`下载失败 HTTP ${r.status}`);
  return Buffer.from(await r.arrayBuffer());
}

const HAS_CWEBP = (() => { try { execFileSync('cwebp', ['-version'], { stdio: 'ignore' }); return true; } catch (e) { return false; } })();
const EXT = HAS_CWEBP ? 'webp' : 'png';

// ③ 压 webp 后 png 母版移到 raw/ 留底（不再 unlink 删）
async function saveImg(buf, id, state) {
  const png = join(OUT_DIR, `${id}_${state}.png`);
  await writeFile(png, buf);
  if (!HAS_CWEBP) return png;
  const webp = join(OUT_DIR, `${id}_${state}.webp`);
  execFileSync('cwebp', ['-q', '82', '-alpha_q', '100', png, '-o', webp], { stdio: 'ignore' });
  await mkdir(RAW_DIR, { recursive: true });
  await rename(png, join(RAW_DIR, `${id}_${state}.png`));
  return webp;
}

async function genCharacter(ch, withStates) {
  console.log(`\n=== ${ch.name}（${ch.id}）===`);
  const idleOut = join(OUT_DIR, `${ch.id}_idle.${EXT}`);
  // ② 幂等：产物已存在就跳过
  if (!FORCE && existsSync(idleOut)) { console.log(`  ↩ 跳过：已存在（--force 重出）`); return; }
  if (DRY_RUN) { console.log(`  🔍 dry-run：将生成 ${ch.id}`); return; }
  process.stdout.write('  生成 idle … ');
  const idleUrl = await falRun(T2I_MODEL, { prompt: ch.prompt, image_size: IMAGE_SIZE, background: 'transparent', quality: 'high', output_format: 'png', num_images: 1 });
  await saveImg(await download(idleUrl), ch.id, 'idle');
  console.log(`✓ ${ch.id}_idle.${EXT}`);
  if (!withStates) return;
  for (const [state, instr] of Object.entries(STATE_EDITS)) {
    process.stdout.write(`  生成 ${state} … `);
    const url = await falRun(EDIT_MODEL, { prompt: instr, image_urls: [idleUrl], background: 'transparent', quality: 'high', output_format: 'png', num_images: 1 });
    await saveImg(await download(url), ch.id, state);
    console.log(`✓ ${ch.id}_${state}.${EXT}`);
  }
}

async function main() {
  const args = process.argv.slice(2);
  const withStates = args.includes('--states');
  const only = args.find((a) => !a.startsWith('--'));
  const list = only ? CHARACTERS.filter((c) => c.id === only) : CHARACTERS;
  if (!list.length) { console.error(`✗ 找不到角色「${only}」`); process.exit(1); }
  await mkdir(OUT_DIR, { recursive: true });
  console.log(`模型: ${T2I_MODEL} · ${IMAGE_SIZE}${DRY_RUN ? ' · dry-run' : ''}${FORCE ? ' · force' : ''}`);
  const ok = [];
  for (const ch of list) {
    try { await genCharacter(ch, withStates); ok.push(ch); }
    catch (e) { console.error(`  ✗ ${ch.id} 失败：${e.message}`); }
  }
  if (!ok.length) { console.error('全部失败'); process.exit(1); }
  console.log('\n完成，粘进 avatars.js 的 PRESETS：');
  for (const ch of ok) {
    const imgs = withStates
      ? `{ idle:'/assets/avatars/${ch.id}_idle.${EXT}', speaking:'/assets/avatars/${ch.id}_speaking.${EXT}', thinking:'/assets/avatars/${ch.id}_thinking.${EXT}' }`
      : `{ idle:'/assets/avatars/${ch.id}_idle.${EXT}' }`;
    console.log(`  { id:'${ch.id}', name:'${ch.name}', type:'image', images:${imgs}, colors:${JSON.stringify(ch.colors)} },`);
  }
}

main().catch((e) => { console.error('脚本异常：', e); process.exit(1); });
