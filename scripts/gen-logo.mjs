#!/usr/bin/env node
/**
 * 微否 logo 生成器 · 零依赖
 *
 * 真源是 assets/brand/logo.svg，本脚本把同一组几何光栅化成 PNG：
 *   - iOS AppIcon 全尺寸（无圆角、无 alpha —— Apple 自己切角，带 alpha 会被拒）
 *   - 微信小程序头像 144（方形，后台上传时被圆裁）
 *   - 文档/README 预览（圆角 + alpha）、单色版、反白版
 *
 * 机器上没有 rsvg-convert / ImageMagick / sharp，sips 也不吃 SVG，
 * 所以这里自带 4x4 超采样光栅器 + PNG 编码器。改几何只需动下面的 GEOM。
 *
 *   node scripts/gen-logo.mjs
 */
import { deflateSync } from 'node:zlib';
import { writeFileSync, mkdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');

// ─── 设计令牌（与 logo.svg 的 <style> 同步）───────────────────────────
const C = {
  forest: [0x08, 0x31, 0x2a], // --forest  logo 专用底色，不进 UI
  accent: [0x18, 0xb6, 0x90], // --accent  碧绿树冠
  coral: [0xff, 0x96, 0x00], // --pop-coral 珊瑚陶盆
  gold: [0xff, 0xd3, 0x35], // --gold    掌握态金花
  ink: [0x0c, 0x5a, 0x48], // --accent-ink 单色版
  white: [0xff, 0xff, 0xff],
};

/**
 * 几何定义在 140×140 的画布上，中心为原点（与 SVG viewBox "-70 -70 140 140" 一致）。
 * 冠三圆刻意不对称：右肩(17,-10)高于左肩(-20,-4)。这点歪就是指纹，别抹平。
 * 冠底最低 y=9，盆沿顶 y=12 —— 那 3 个单位的缝是剪影可读性的命根子。
 */
const GEOM = {
  viewBox: 140,
  bgRadius: 33, // 仅用于圆角预览；iOS/微信输出恒为 0
  crown: [
    { cx: -2, cy: -17, r: 25 },
    { cx: -20, cy: -4, r: 13 },
    { cx: 17, cy: -10, r: 16.5 },
  ],
  shine: { cx: 14, cy: -27, r: 4.4, alpha: 0.85 },
  potRim: { x: -24, y: 12, w: 48, h: 9.5, r: 3.5 },
  // M-20 21 L-15 46 Q-14 50 -9 50 L9 50 Q14 50 15 46 L20 21 Z
  // 起点 y=21 顶住盆沿底边：盆沿不能浮空。唯一该留的缝在冠(底 y=9)与盆沿(顶 y=12)之间。
  potBody: {
    start: [-20, 21],
    line1: [-15, 46],
    quad1: { c: [-14, 50], to: [-9, 50] },
    line2: [9, 50],
    quad2: { c: [14, 50], to: [15, 46] },
    line3: [20, 21],
  },
  // 掌握态金花：开在冠内左上，不是插在冠尖上（顶一颗星会读成圣诞树）。
  bloom:
    'M-14 -38 L-12 -32.75 L-6.39 -32.47 L-10.77 -28.95 L-9.3 -23.53 L-14 -26.6 L-18.7 -23.53 L-17.23 -28.95 L-21.61 -32.47 L-16 -32.75 Z',
};

// ─── 形状：coverage(x, y) 返回该点是否落在形内 ──────────────────────
const inCircle = ({ cx, cy, r }) => {
  const f = (x, y) => (x - cx) ** 2 + (y - cy) ** 2 <= r * r;
  f.bbox = [cx - r, cy - r, cx + r, cy + r];
  return f;
};

const inRoundRect = ({ x: rx, y: ry, w, h, r }) => {
  const f = (x, y) => {
    if (x < rx || x > rx + w || y < ry || y > ry + h) return false;
    if (r <= 0) return true;
    const cx = Math.min(Math.max(x, rx + r), rx + w - r);
    const cy = Math.min(Math.max(y, ry + r), ry + h - r);
    return (x - cx) ** 2 + (y - cy) ** 2 <= r * r;
  };
  f.bbox = [rx, ry, rx + w, ry + h];
  return f;
};

const quadPoints = (p0, c, p1, n = 12) => {
  const pts = [];
  for (let i = 1; i <= n; i++) {
    const t = i / n;
    const u = 1 - t;
    pts.push([u * u * p0[0] + 2 * u * t * c[0] + t * t * p1[0], u * u * p0[1] + 2 * u * t * c[1] + t * t * p1[1]]);
  }
  return pts;
};

const potPolygon = () => {
  const g = GEOM.potBody;
  return [
    g.start,
    g.line1,
    ...quadPoints(g.line1, g.quad1.c, g.quad1.to),
    g.line2,
    ...quadPoints(g.line2, g.quad2.c, g.quad2.to),
    g.line3,
  ];
};

const parsePath = (d) =>
  d
    .trim()
    .replace(/[MLZ]/g, ' ')
    .trim()
    .split(/\s+/)
    .reduce((acc, _, i, arr) => (i % 2 === 0 ? [...acc, [+arr[i], +arr[i + 1]]] : acc), []);

const inPolygon = (pts) => {
  const f = (x, y) => {
    let hit = false;
    for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) {
      const [xi, yi] = pts[i];
      const [xj, yj] = pts[j];
      if (yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) hit = !hit;
    }
    return hit;
  };
  const xs = pts.map((p) => p[0]);
  const ys = pts.map((p) => p[1]);
  f.bbox = [Math.min(...xs), Math.min(...ys), Math.max(...xs), Math.max(...ys)];
  return f;
};

// ─── 光栅化 ──────────────────────────────────────────────────────────
const SS = 4; // 每像素 4×4 超采样

/** @returns Float64Array RGBA（premultiplied 无关，直接 alpha 合成） */
function rasterize(size, { radius, bloom = false, variant = 'color' }) {
  const scale = GEOM.viewBox / size;
  const px = new Float64Array(size * size * 4);

  const layers = [];
  const bgFill = variant === 'invert' ? C.accent : variant === 'mono' ? null : C.forest;
  if (bgFill) {
    layers.push({
      test: inRoundRect({ x: -70, y: -70, w: 140, h: 140, r: radius }),
      color: bgFill,
      alpha: 1,
      fast: radius <= 0,
    });
  }

  const fg = (kind) => {
    if (variant === 'mono') return C.ink;
    if (variant === 'invert') return C.white;
    return kind;
  };

  for (const c of GEOM.crown) layers.push({ test: inCircle(c), color: fg(C.accent), alpha: 1 });
  if (variant === 'color') layers.push({ test: inCircle(GEOM.shine), color: C.white, alpha: GEOM.shine.alpha });
  if (bloom) layers.push({ test: inPolygon(parsePath(GEOM.bloom)), color: fg(C.gold), alpha: 1 });
  layers.push({ test: inRoundRect(GEOM.potRim), color: fg(C.coral), alpha: 1 });
  layers.push({ test: inPolygon(potPolygon()), color: fg(C.coral), alpha: 1 });

  // painter's algorithm：逐层合成，每层只走自己包围盒内的像素
  const toPx = (u) => (u + 70) / scale;
  for (const L of layers) {
    const [bx0, by0, bx1, by1] = L.test.bbox;
    const x0 = Math.max(0, Math.floor(toPx(bx0)));
    const y0 = Math.max(0, Math.floor(toPx(by0)));
    const x1 = Math.min(size - 1, Math.ceil(toPx(bx1)));
    const y1 = Math.min(size - 1, Math.ceil(toPx(by1)));
    for (let py = y0; py <= y1; py++) {
      for (let pxi = x0; pxi <= x1; pxi++) {
        let cov;
        if (L.fast) {
          cov = 1;
        } else {
          let hits = 0;
          for (let sy = 0; sy < SS; sy++) {
            for (let sx = 0; sx < SS; sx++) {
              const ux = -70 + (pxi + (sx + 0.5) / SS) * scale;
              const uy = -70 + (py + (sy + 0.5) / SS) * scale;
              if (L.test(ux, uy)) hits++;
            }
          }
          cov = hits / (SS * SS);
        }
        if (cov === 0) continue;
        const o = (py * size + pxi) * 4;
        const a = cov * L.alpha;
        px[o] = px[o] * (1 - a) + L.color[0] * a;
        px[o + 1] = px[o + 1] * (1 - a) + L.color[1] * a;
        px[o + 2] = px[o + 2] * (1 - a) + L.color[2] * a;
        px[o + 3] = px[o + 3] * (1 - a) + 255 * a;
      }
    }
  }
  return px;
}

// ─── PNG 编码 ────────────────────────────────────────────────────────
const CRC_TABLE = (() => {
  const t = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    t[n] = c >>> 0;
  }
  return t;
})();

const crc32 = (buf) => {
  let c = 0xffffffff;
  for (const b of buf) c = CRC_TABLE[(c ^ b) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
};

const chunk = (type, data) => {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const body = Buffer.concat([Buffer.from(type, 'ascii'), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
};

/** @param alpha false → colorType 2 (RGB)，iOS AppIcon 必须无 alpha 通道 */
function encodePNG(px, size, alpha) {
  const ch = alpha ? 4 : 3;
  const raw = Buffer.alloc(size * (size * ch + 1));
  let p = 0;
  for (let y = 0; y < size; y++) {
    raw[p++] = 0; // filter: none
    for (let x = 0; x < size; x++) {
      const o = (y * size + x) * 4;
      const a = px[o + 3] / 255;
      // 无 alpha 输出时把未覆盖处按背景合成为不透明（iOS 圆角为 0，实际不会触发）
      for (let i = 0; i < 3; i++) raw[p++] = Math.round(alpha ? px[o + i] : px[o + i] * a + 255 * (1 - a));
      if (alpha) raw[p++] = Math.round(px[o + 3]);
    }
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8;
  ihdr[9] = alpha ? 6 : 2;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk('IHDR', ihdr),
    chunk('IDAT', deflateSync(raw, { level: 9 })),
    chunk('IEND', Buffer.alloc(0)),
  ]);
}

const write = (rel, size, opts) => {
  const alpha = opts.alpha ?? false;
  const buf = encodePNG(rasterize(size, opts), size, alpha);
  const abs = resolve(ROOT, rel);
  mkdirSync(dirname(abs), { recursive: true });
  writeFileSync(abs, buf);
  console.log(`  ${rel}  ${size}×${size}  ${(buf.length / 1024).toFixed(1)}KB`);
};

// ─── 输出 ────────────────────────────────────────────────────────────
const IOS_DIR = 'weifou-app/ios/Runner/Assets.xcassets/AppIcon.appiconset';
const IOS = {
  'Icon-App-20x20@1x.png': 20,
  'Icon-App-20x20@2x.png': 40,
  'Icon-App-20x20@3x.png': 60,
  'Icon-App-29x29@1x.png': 29,
  'Icon-App-29x29@2x.png': 58,
  'Icon-App-29x29@3x.png': 87,
  'Icon-App-40x40@1x.png': 40,
  'Icon-App-40x40@2x.png': 80,
  'Icon-App-40x40@3x.png': 120,
  'Icon-App-60x60@2x.png': 120,
  'Icon-App-60x60@3x.png': 180,
  'Icon-App-76x76@1x.png': 76,
  'Icon-App-76x76@2x.png': 152,
  'Icon-App-83.5x83.5@2x.png': 167,
  'Icon-App-1024x1024@1x.png': 1024,
};

console.log('iOS AppIcon（方形无圆角、无 alpha）');
for (const [name, size] of Object.entries(IOS)) write(`${IOS_DIR}/${name}`, size, { radius: 0 });

console.log('\n微信小程序头像（后台上传，系统圆裁）');
write('assets/brand/wechat-avatar-144.png', 144, { radius: 0 });

console.log('\n预览与变体（圆角 + alpha）');
write('assets/brand/logo-512.png', 512, { radius: GEOM.bgRadius, alpha: true });
write('assets/brand/logo-bloom-512.png', 512, { radius: GEOM.bgRadius, alpha: true, bloom: true });
write('assets/brand/logo-mono-512.png', 512, { radius: 0, alpha: true, variant: 'mono' });
write('assets/brand/logo-invert-512.png', 512, { radius: GEOM.bgRadius, alpha: true, variant: 'invert' });
write('assets/brand/logo-16.png', 16, { radius: 4, alpha: true });
write('assets/brand/logo-32.png', 32, { radius: 8, alpha: true });
