#!/usr/bin/env node
/** 学习区贴纸图标批量生成（照 gen-avatars.mjs 范式）：
 *  深描边贴纸风只进插画层（控件层仍白底浅灰边，见 docs/design-tokens.md 暖纸底护栏）。
 *  默认 fal gpt-image-1（真透明）；gpt-image-2 假透明勿用，除非自己加抠图后处理。
 *  用法：FALAI_API_KEY=xxx node scripts/gen-learn-icons.mjs [--dry-run|--force|<id>]
 */
import { writeFile, mkdir, rename } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const FAL_KEY = process.env.FALAI_API_KEY;
if (!FAL_KEY) { console.error('✗ 缺少 FALAI_API_KEY'); process.exit(1); }

const T2I_MODEL = process.env.FAL_IMAGE_MODEL || 'fal-ai/gpt-image-1/text-to-image';
const IMAGE_SIZE = process.env.FAL_IMAGE_SIZE || '1024x1024';
const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT_DIR = join(__dirname, '..', 'weifou-miniapp', 'assets', 'icons', 'learn');
const RAW_DIR = join(OUT_DIR, 'raw');
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const FORCE = process.argv.includes('--force');
const DRY_RUN = process.argv.includes('--dry-run');

// 统一成长插画：深描边留在插画内部；以雾蓝紫、低饱和金与路径辅助色为主。
const STYLE = [
  '游戏贴纸风小图标：单个物件居中、饱满圆润卡通造型、均匀的深棕色粗描边（描边只属于这枚贴纸本身）、',
  '明亮糖果色扁平填充+一处简单高光、微弱的同色深一档底部投影表现体积；',
  '正方形构图、主体占画面约 70%、背景纯透明（PNG 透明通道，无背景无地面无外框）；',
  '不要任何文字、数字、水印、logo；风格统一、像同一套贴纸里的一枚。',
].join('');

const ICONS = [
  { id: 'flame',   name: '连学火焰',  prompt: `一枚燃烧的小火苗贴纸，珊瑚橙#FF9600为主、内焰奶油黄#FFC800，元气十足。${STYLE}` },
  { id: 'star',    name: '点亮星星',  prompt: `一枚五角星贴纸，金黄色#FFD335、边角圆润、闪亮饱满。${STYLE}` },
  { id: 'crown',   name: '掌握金冠',  prompt: `一顶三尖小王冠贴纸，金色#FFD335、冠尖各嵌一颗小圆宝石、庄重又可爱。${STYLE}` },
  { id: 'sparkle', name: '下一关星芒', prompt: `一簇四角星芒贴纸（一大两小），雾蓝#B9DDED与蓝紫#7772C8搭配、轻盈闪烁感。${STYLE}` },
  { id: 'lock',    name: '未解锁挂锁', prompt: `一把圆润的小挂锁贴纸，暖灰色锁体、锁孔清晰、憨态可掬不冷硬。${STYLE}` },
  { id: 'review',  name: '复习循环',  prompt: `两支首尾相接的环形循环箭头贴纸，天蓝色#1CB0F6、箭头圆头粗壮。${STYLE}` },
  { id: 'medal',   name: '段位奖牌',  prompt: `一枚挂着绶带的圆形奖牌贴纸，牌面蓝紫#7772C8、绶带低饱和金#C7A45D、中央一颗小星。${STYLE}` },
  { id: 'speak',   name: '流利开口',  prompt: `一张侧面张开说话的可爱嘴巴加三道声波弧线的贴纸，珊瑚橙#FF9600声波、活力开朗。${STYLE}` },
  { id: 'target',  name: '准确靶心',  prompt: `一个三环圆靶正中插着一支圆头飞镖的贴纸，靶环蓝紫#7772C8与白相间、飞镖低饱和金#C7A45D。${STYLE}` },
  { id: 'bubble',  name: '表达气泡',  prompt: `一只圆润对话气泡里排着三个小圆点的贴纸，天蓝色#DDF4FF气泡、深蓝#1CB0F6圆点。${STYLE}` },
];

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

// 图标实际显示 ≤60rpx(≈30px@3x=90px),160px 足够;webp 3-8KB 不压主包
const ICON_SIZE = process.env.ICON_SIZE || '160';

async function saveImg(buf, id) {
  const png = join(OUT_DIR, `${id}.png`);
  await writeFile(png, buf);
  if (!HAS_CWEBP) return png;
  const small = join(OUT_DIR, `${id}.${ICON_SIZE}.png`);
  execFileSync('sips', ['-Z', ICON_SIZE, png, '--out', small], { stdio: 'ignore' });
  const webp = join(OUT_DIR, `${id}.webp`);
  execFileSync('cwebp', ['-q', '82', '-alpha_q', '100', small, '-o', webp], { stdio: 'ignore' });
  execFileSync('rm', [small]);
  await mkdir(RAW_DIR, { recursive: true });
  await rename(png, join(RAW_DIR, `${id}.png`));
  return webp;
}

async function main() {
  const only = process.argv.slice(2).find((a) => !a.startsWith('--'));
  const list = only ? ICONS.filter((c) => c.id === only) : ICONS;
  if (!list.length) { console.error(`✗ 找不到图标「${only}」`); process.exit(1); }
  await mkdir(OUT_DIR, { recursive: true });
  console.log(`模型: ${T2I_MODEL} · ${IMAGE_SIZE}${DRY_RUN ? ' · dry-run' : ''}${FORCE ? ' · force' : ''}`);
  const ok = [];
  for (const ic of list) {
    const out = join(OUT_DIR, `${ic.id}.${EXT}`);
    if (!FORCE && existsSync(out)) { console.log(`↩ ${ic.id} 跳过：已存在（--force 重出）`); ok.push(ic); continue; }
    if (DRY_RUN) { console.log(`🔍 dry-run：将生成 ${ic.id}（${ic.name}）`); continue; }
    process.stdout.write(`生成 ${ic.id}（${ic.name}）… `);
    try {
      const url = await falRun(T2I_MODEL, { prompt: ic.prompt, image_size: IMAGE_SIZE, background: 'transparent', quality: 'high', output_format: 'png', num_images: 1 });
      await saveImg(await download(url), ic.id);
      console.log(`✓ ${ic.id}.${EXT}`);
      ok.push(ic);
    } catch (e) { console.error(`✗ 失败：${e.message}`); }
  }
  if (!DRY_RUN) console.log(`\n完成 ${ok.length}/${list.length}，产物在 weifou-miniapp/assets/icons/learn/`);
}

main().catch((e) => { console.error('脚本异常：', e); process.exit(1); });
