#!/usr/bin/env python3
"""吉祥物母版后处理：清背景 → 缩放 → 出 webp。

为什么需要它：fal 的 edit 端点偶发吐出「假透明」——alpha 不是 0 而是 100~160 的
半透明灰底（think 姿势稳定复现，重跑两次都脏）。直接进包会在角色四周渲染出一圈灰框。
这里从四角 flood fill 清掉与边界连通的半透明背景；对本来就干净的图，BFS 起点 alpha=0
即刻停止，等于空操作，所以对全部姿势无差别跑一遍是安全的。

不做 autocrop：六个姿势共享 1024 见方画布的居中锚点，裁切会让走路/装死切换时角色跳动。
装死图主体扁横、显得矮小——那是对的，它本来就躺下了。

用法：python3 scripts/pack-mascot.py [--size 256]
"""
import sys
from collections import deque
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
# 母版在包外，成品进包（见 gen-mascot.mjs 同款注释：1024px 母版进包会顶爆主包）
RAW = ROOT / "assets-src" / "mascot" / "raw"
OUT = ROOT / "weifou-miniapp" / "assets" / "mascot"

# 低于此 alpha 且与画布边界连通的像素判为背景。取 190：假透明脏底最高约 160，
# 主体粗描边为 255，只会削掉主体最外圈 1px 抗锯齿——缩放到 256 时被重新平滑，看不出。
BG_ALPHA_MAX = 190


def clear_background(im: Image.Image) -> int:
    w, h = im.size
    px = im.load()
    seen = bytearray(w * h)
    q = deque()
    for x in range(w):
        for y in (0, h - 1):
            q.append((x, y))
    for y in range(h):
        for x in (0, w - 1):
            q.append((x, y))
    cleared = 0
    while q:
        x, y = q.popleft()
        i = y * w + x
        if seen[i]:
            continue
        seen[i] = 1
        r, g, b, a = px[x, y]
        if a > BG_ALPHA_MAX:
            continue  # 撞到主体，这条路径到此为止
        if a:
            px[x, y] = (r, g, b, 0)
            cleared += 1
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            nx, ny = x + dx, y + dy
            if 0 <= nx < w and 0 <= ny < h and not seen[ny * w + nx]:
                q.append((nx, ny))
    return cleared


def main() -> int:
    size = 256
    if "--size" in sys.argv:
        size = int(sys.argv[sys.argv.index("--size") + 1])
    masters = sorted(RAW.glob("possum_*.png"))
    if not masters:
        print(f"✗ {RAW} 下没有 possum_*.png 母版，先跑 gen-mascot.mjs")
        return 1
    for p in masters:
        im = Image.open(p).convert("RGBA")
        cleared = clear_background(im)
        im = im.resize((size, size), Image.LANCZOS)
        dst = OUT / f"{p.stem}.webp"
        im.save(dst, "WEBP", quality=86, method=6, exact=False)
        kb = dst.stat().st_size / 1024
        note = f"清背景 {cleared} px" if cleared else "本就干净"
        print(f"  {p.stem:14} {note:16} → {dst.name} {kb:.0f}KB")
    print(f"\n完成：{len(masters)} 张 @{size}px")
    return 0


if __name__ == "__main__":
    sys.exit(main())
