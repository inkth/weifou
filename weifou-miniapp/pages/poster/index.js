const { request } = require('../../utils/request');
const { getPreset, initial } = require('../../utils/avatars');

Page({
  data: {
    profileId: '',
    ready: false,
    bundle: null,
    tempFilePath: '',
  },

  async onLoad(query) {
    this.setData({ profileId: query.profileId });
    try {
      const bundle = await request({ url: `/share/bundle/${query.profileId}` });
      this.setData({ bundle });
      await this.draw();
      this.setData({ ready: true });
    } catch (e) {
      wx.showToast({ title: e.message || '加载失败', icon: 'none' });
    }
  },

  draw() {
    return new Promise((resolve, reject) => {
      const query = wx.createSelectorQuery();
      query
        .select('#poster')
        .fields({ node: true, size: true })
        .exec(async (res) => {
          // canvas 未就绪（慢首屏布局）时 res[0] 可能为空：显式 reject 交给 onLoad 兜底，
          // 否则 await this.draw() 永挂、页面永远停在"海报合成中…"。
          const canvas = res[0] && res[0].node;
          if (!canvas) { reject(new Error('海报画布未就绪，请重试')); return; }
          const ctx = canvas.getContext('2d');
          const dpr = wx.getWindowInfo().pixelRatio;
          const cssW = 300;
          const cssH = 450;
          canvas.width = cssW * dpr;
          canvas.height = cssH * dpr;
          ctx.scale(dpr, dpr);

          // 背景
          ctx.fillStyle = '#ffffff';
          ctx.fillRect(0, 0, cssW, cssH);

          // 顶部色块
          ctx.fillStyle = '#18b690';
          ctx.fillRect(0, 0, cssW, 100);
          // 绿底场合 logo 整体反白（见 assets/brand/logo.svg 的单色规则）
          this.drawMark(ctx, 20, 14, 24, '#ffffff');
          ctx.fillStyle = '#ffffff';
          ctx.font = '14px sans-serif';
          ctx.fillText('微否 · 我的 AI 主页', 52, 31);
          ctx.font = 'bold 22px sans-serif';
          ctx.fillText(this.data.bundle.realName || '', 20, 70);

          // 形象（静态渐变圆 + 首字，与所选动态形象同色）
          this.drawAvatar(ctx, cssW - 50, 50, 28);

          // 一句话介绍
          ctx.fillStyle = '#4b4b4b';
          ctx.font = 'bold 18px sans-serif';
          const oneLiner = this.data.bundle.oneLiner || '';
          this.wrapText(ctx, oneLiner, 20, 150, cssW - 40, 26);

          // 标签
          const tags = this.data.bundle.tags || [];
          let tx = 20;
          let ty = 240;
          ctx.font = '12px sans-serif';
          tags.forEach((t) => {
            const tw = ctx.measureText(t).width + 20;
            if (tx + tw > cssW - 20) { tx = 20; ty += 28; }
            ctx.fillStyle = '#f7f7f7';
            this.roundRect(ctx, tx, ty - 14, tw, 22, 11);
            ctx.fill();
            ctx.fillStyle = '#6f6f6f';
            ctx.fillText(t, tx + 10, ty + 1);
            tx += tw + 8;
          });

          // 小程序码
          if (this.data.bundle.wxacodeBase64) {
            try {
              const img = canvas.createImage();
              await new Promise((res2, rej2) => {
                img.onload = res2;
                img.onerror = rej2;
                img.src = this.data.bundle.wxacodeBase64;
              });
              ctx.drawImage(img, cssW - 110, cssH - 130, 90, 90);
            } catch (e) {
              // ignore
            }
          }

          ctx.fillStyle = '#999999';
          ctx.font = '11px sans-serif';
          ctx.fillText('长按识别 · 和 TA 的 AI 聊聊', 20, cssH - 30);

          resolve();
        });
    });
  },

  /**
   * 微否图形标（小盆栽）。几何真源 assets/brand/logo.svg，坐标系 140×140、中心原点。
   * 这里只画冠与盆、不画底色方块 —— 底色由调用方所在的色块提供。
   * @param x,y 左上角  @param size 边长  @param color 单色填充（绿底反白 / 浅底墨绿）
   */
  drawMark(ctx, x, y, size, color) {
    const s = size / 140;
    const px = (u) => x + (u + 70) * s;
    const py = (u) => y + (u + 70) * s;

    ctx.fillStyle = color;

    // 树冠：三圆不对称，右肩高于左肩
    ctx.beginPath();
    [
      [-2, -17, 25],
      [-20, -4, 13],
      [17, -10, 16.5],
    ].forEach(([cx, cy, r]) => {
      ctx.moveTo(px(cx) + r * s, py(cy));
      ctx.arc(px(cx), py(cy), r * s, 0, Math.PI * 2);
    });
    ctx.fill();

    // 盆沿
    this.roundRect(ctx, px(-24), py(12), 48 * s, 9.5 * s, 3.5 * s);
    ctx.fill();

    // 盆身
    ctx.beginPath();
    ctx.moveTo(px(-20), py(21));
    ctx.lineTo(px(-15), py(46));
    ctx.quadraticCurveTo(px(-14), py(50), px(-9), py(50));
    ctx.lineTo(px(9), py(50));
    ctx.quadraticCurveTo(px(14), py(50), px(15), py(46));
    ctx.lineTo(px(20), py(21));
    ctx.closePath();
    ctx.fill();
  },

  drawAvatar(ctx, cx, cy, r) {
    const b = this.data.bundle || {};
    const preset = getPreset(b.avatarStyle, b.profileId);
    const grad = ctx.createLinearGradient(cx - r, cy - r, cx + r, cy + r);
    const cs = preset.colors;
    cs.forEach((c, i) => grad.addColorStop(cs.length === 1 ? 1 : i / (cs.length - 1), c));
    ctx.save();
    ctx.beginPath();
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.closePath();
    ctx.fillStyle = grad;
    ctx.fill();
    ctx.clip();
    ctx.fillStyle = '#ffffff';
    ctx.font = `bold ${Math.round(r * 0.9)}px sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(initial(b.realName), cx, cy + 1);
    ctx.restore();
    // 还原默认对齐，避免影响后续文本
    ctx.textAlign = 'left';
    ctx.textBaseline = 'alphabetic';
  },

  wrapText(ctx, text, x, y, maxWidth, lineHeight) {
    const chars = text.split('');
    let line = '';
    let yy = y;
    chars.forEach((ch) => {
      const test = line + ch;
      if (ctx.measureText(test).width > maxWidth) {
        ctx.fillText(line, x, yy);
        line = ch;
        yy += lineHeight;
      } else {
        line = test;
      }
    });
    if (line) ctx.fillText(line, x, yy);
  },

  roundRect(ctx, x, y, w, h, r) {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.arcTo(x + w, y, x + w, y + h, r);
    ctx.arcTo(x + w, y + h, x, y + h, r);
    ctx.arcTo(x, y + h, x, y, r);
    ctx.arcTo(x, y, x + w, y, r);
    ctx.closePath();
  },

  saveToAlbum() {
    wx.createSelectorQuery()
      .select('#poster')
      .fields({ node: true, size: true })
      .exec((res) => {
        const canvas = res[0] && res[0].node;
        if (!canvas) { wx.showToast({ title: '海报未就绪，请稍候', icon: 'none' }); return; }
        wx.canvasToTempFilePath({
          canvas,
          success: (r) => {
            wx.saveImageToPhotosAlbum({
              filePath: r.tempFilePath,
              success: () => wx.showToast({ title: '已保存', icon: 'success' }),
              fail: (e) => {
                // 用户拒过一次后系统不再弹授权框：必须引导去设置手动开，否则点"保存"永远没反应。
                if (e.errMsg && e.errMsg.includes('auth deny')) {
                  wx.showModal({
                    title: '需要相册权限',
                    content: '请在设置中允许"保存到相册"，再回来重试',
                    confirmText: '去设置',
                    success: (m) => { if (m.confirm) wx.openSetting(); },
                  });
                } else {
                  wx.showToast({ title: '保存失败', icon: 'none' });
                }
              },
            });
          },
          fail: () => wx.showToast({ title: '生成失败', icon: 'none' }),
        });
      });
  },
});
