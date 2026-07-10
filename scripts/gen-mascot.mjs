#!/usr/bin/env node
/**
 * 学习区吉祥物「负鼠」资产管线。两阶段，别混着跑：
 *
 *   ① node scripts/gen-mascot.mjs candidates      出 N 张母版候选 → assets/mascot/candidates/
 *   ② 挑一张，记下它的编号
 *   ③ node scripts/gen-mascot.mjs poses --from c3 从该母版派生全部姿势 → assets/mascot/
 *
 * 铁律：母版只 T2I 一次。所有姿势/装备一律走 EDIT 从母版派生（「只改动作、其余锁死」），
 * 绝不用 T2I 重新生成第二只负鼠——那样必然出现两张脸，风格漂移一次就毁掉整套资产。
 * 这套打法与 gen-avatars.mjs 的 STATE_EDITS 同源，只是那边锁表情、这边锁动作。
 */
import { writeFile, mkdir, rename, readdir } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const FAL_KEY = process.env.FALAI_API_KEY;
if (!FAL_KEY) { console.error('✗ 缺少 FALAI_API_KEY'); process.exit(1); }

const T2I_MODEL = process.env.FAL_IMAGE_MODEL || 'fal-ai/gpt-image-1/text-to-image';
const EDIT_MODEL = process.env.FAL_EDIT_MODEL || 'fal-ai/gpt-image-1/edit';
// 吉祥物是方形贴纸，不用立绘的竖版
const IMAGE_SIZE = process.env.FAL_IMAGE_SIZE || '1024x1024';
const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT_DIR = join(__dirname, '..', 'weifou-miniapp', 'assets', 'mascot');
const CAND_DIR = join(OUT_DIR, 'candidates');
const RAW_DIR = join(OUT_DIR, 'raw');
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const FORCE = process.argv.includes('--force');
const DRY_RUN = process.argv.includes('--dry-run');

// 与糖果风 UI 同语言：扁平粗描边、平涂、无渐变无写实。不写长相细节——那些留给候选变体去分化。
const STYLE = [
  '扁平卡通贴纸风格：均匀的深色粗描边、纯色平涂、无渐变无阴影无纹理、无写实毛发；',
  '2 头身超变形（chibi）比例，大头圆身、四肢短粗，可爱亲人、绝不惊悚；',
  '正面站立、全身入画、居中，四周留少量空白；画面很小时剪影仍清晰可辨；',
  '背景纯透明（PNG 透明通道，无背景/地面/投影）；不要任何文字、水印、logo、边框、签名；',
  '不是写实动物、不是真实照片、非任何已有版权角色。',
].join('');

// 负鼠的不变式（每张候选都必须满足，靠它保住「一眼是同一个物种设定」）
const CANON = '一只拟人化的小负鼠：圆脸、圆耳朵、粉色小鼻头、毛茸茸的尾巴（不要写实的裸尾）、常年一副没睡醒的死鱼眼。';

// 母版候选：只分化「长相取向」，风格常量不动。挑脸就是在这几种取向里选。
const CANDIDATES = [
  { id: 'c1', desc: '经典摆烂', look: '灰白配色，眼皮半耷拉的死鱼眼，嘴角平直，双手插在肚子前，一副「又要学习了」的无语表情。' },
  { id: 'c2', desc: '圆滚团子', look: '身体极圆几乎是个球，米白色，腮帮鼓鼓，眼睛是两个小圆点，尾巴卷成一圈贴在身侧，憨得不像话。' },
  { id: 'c3', desc: '呆毛傲娇', look: '浅灰色，头顶一根翘起的呆毛，眉毛微微下压显得不服气，一只耳朵是深灰的（记忆点），眼神斜睨。' },
  { id: 'c4', desc: '眼袋社畜', look: '灰褐色，眼下有明显的深色眼袋，眼神空洞放空，肩膀微塌，尾巴无力地垂着，写满「困」字。' },
  { id: 'c5', desc: '叼叶行者', look: '奶白色带浅灰耳朵，嘴里横叼一片翠绿的叶子，眼神慵懒但脚步稳，尾巴俏皮上翘。' },
];

// 姿势派生：全部「只改动作、锁死其余」——这句话是一致性的全部秘密。
const LOCK = '在保持这只负鼠的长相、五官、配色、描边粗细、画风、2头身比例与透明背景完全一致的前提下，只改变它的姿势与表情：';
const POSES = {
  idle:  `${LOCK}正面站立待机，双手自然垂放，死鱼眼直视前方，神情平静地摆烂。`,
  walk:  `${LOCK}侧身（面朝画面右方）迈步行走，一条腿抬起、身体微微前倾，尾巴随步伐甩起，一副被迫赶路的样子。`,
  atk:   `${LOCK}面朝画面右方向前猛冲出击，双手前伸、身体前倾成冲刺姿态，眼睛第一次睁大、气势汹汹。`,
  dead:  `${LOCK}四脚朝天躺平装死：整个身体侧倒在地、肚皮朝上，四肢僵直朝天，眼睛变成两个「X」，舌头微微吐出，尾巴瘫直。`,
  win:   `${LOCK}原地高高跳起欢呼庆祝：双手举过头顶、身体腾空，眼睛变成两道开心的弯月，嘴巴大张笑开，尾巴兴奋翘起。`,
  think: `${LOCK}歪头思考：一只手托着下巴，眼睛向上瞟，嘴角困惑地撇着，头顶微微歪斜。`,
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

// png 母版留底到 raw/，产物出 webp（小程序包体敏感）
async function saveImg(buf, dir, name) {
  await mkdir(dir, { recursive: true });
  const png = join(dir, `${name}.png`);
  await writeFile(png, buf);
  if (!HAS_CWEBP) return png;
  const webp = join(dir, `${name}.webp`);
  execFileSync('cwebp', ['-q', '82', '-alpha_q', '100', png, '-o', webp], { stdio: 'ignore' });
  await mkdir(RAW_DIR, { recursive: true });
  await rename(png, join(RAW_DIR, `${name}.png`));
  return webp;
}

// 阶段①：出母版候选，人工挑脸
async function genCandidates() {
  console.log(`\n=== 母版候选（${CANDIDATES.length} 张）===\n挑中哪张，记住它的编号，再跑 poses --from <编号>\n`);
  for (const c of CANDIDATES) {
    const out = join(CAND_DIR, `${c.id}.${EXT}`);
    if (!FORCE && existsSync(out)) { console.log(`  ↩ ${c.id} 已存在（--force 重出）`); continue; }
    if (DRY_RUN) { console.log(`  🔍 dry-run：${c.id}「${c.desc}」`); continue; }
    process.stdout.write(`  ${c.id}「${c.desc}」… `);
    try {
      const url = await falRun(T2I_MODEL, {
        prompt: `${CANON}${c.look}${STYLE}`,
        image_size: IMAGE_SIZE, background: 'transparent', quality: 'high', output_format: 'png', num_images: 1,
      });
      await saveImg(await download(url), CAND_DIR, c.id);
      console.log(`✓ candidates/${c.id}.${EXT}`);
    } catch (e) { console.error(`✗ ${e.message}`); }
  }
}

// 阶段②：从选定母版派生全部姿势。母版必须上传成 URL 给 EDIT 模型——
// fal 的 edit 接口收 image_urls，本地文件需先转 data URI。
async function genPoses(fromId) {
  const src = join(CAND_DIR, `${fromId}.png`);
  const srcRaw = join(RAW_DIR, `${fromId}.png`);
  const path = existsSync(src) ? src : (existsSync(srcRaw) ? srcRaw : null);
  if (!path) {
    const have = existsSync(CAND_DIR) ? (await readdir(CAND_DIR)).join(' ') : '（还没跑 candidates）';
    console.error(`✗ 找不到母版 ${fromId} 的 png。现有：${have}`);
    process.exit(1);
  }
  const { readFile } = await import('node:fs/promises');
  const dataUri = `data:image/png;base64,${(await readFile(path)).toString('base64')}`;
  console.log(`\n=== 从母版 ${fromId} 派生姿势（${Object.keys(POSES).length} 个）===\n`);
  for (const [pose, instr] of Object.entries(POSES)) {
    const out = join(OUT_DIR, `possum_${pose}.${EXT}`);
    if (!FORCE && existsSync(out)) { console.log(`  ↩ ${pose} 已存在（--force 重出）`); continue; }
    if (DRY_RUN) { console.log(`  🔍 dry-run：${pose}`); continue; }
    process.stdout.write(`  ${pose} … `);
    try {
      const url = await falRun(EDIT_MODEL, {
        prompt: instr, image_urls: [dataUri],
        background: 'transparent', quality: 'high', output_format: 'png', num_images: 1,
      });
      await saveImg(await download(url), OUT_DIR, `possum_${pose}`);
      console.log(`✓ possum_${pose}.${EXT}`);
    } catch (e) { console.error(`✗ ${e.message}`); }
  }
  console.log(`\n完成。舞台接线：把 agent-chat 的 .hero-face emoji 换成 <image src="/assets/mascot/possum_idle.${EXT}">，`);
  console.log(`动作层 class（walk/atk/dead/win）改为切 src 或叠加 CSS 动画。`);
}

async function main() {
  const args = process.argv.slice(2);
  const cmd = args.find((a) => !a.startsWith('--'));
  console.log(`模型: ${cmd === 'poses' ? EDIT_MODEL : T2I_MODEL} · ${IMAGE_SIZE}${DRY_RUN ? ' · dry-run' : ''}${FORCE ? ' · force' : ''}`);
  if (cmd === 'candidates') return genCandidates();
  if (cmd === 'poses') {
    const i = args.indexOf('--from');
    const from = i >= 0 ? args[i + 1] : '';
    if (!from) { console.error('✗ poses 需要 --from <母版编号，如 c3>'); process.exit(1); }
    return genPoses(from);
  }
  console.error('用法：\n  gen-mascot.mjs candidates            出母版候选\n  gen-mascot.mjs poses --from c3       从母版派生姿势\n可选：--force 重出  --dry-run 只看不生成');
  process.exit(1);
}

main().catch((e) => { console.error('脚本异常：', e); process.exit(1); });
