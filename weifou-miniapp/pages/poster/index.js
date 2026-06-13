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
    return new Promise((resolve) => {
      const query = wx.createSelectorQuery();
      query
        .select('#poster')
        .fields({ node: true, size: true })
        .exec(async (res) => {
          const canvas = res[0].node;
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
          ctx.fillStyle = '#1f2330';
          ctx.fillRect(0, 0, cssW, 100);
          ctx.fillStyle = '#ffffff';
          ctx.font = '14px sans-serif';
          ctx.fillText('微否 · 我的 AI 主页', 20, 30);
          ctx.font = 'bold 22px sans-serif';
          ctx.fillText(this.data.bundle.realName || '', 20, 70);

          // 形象（静态渐变圆 + 首字，与所选动态形象同色）
          this.drawAvatar(ctx, cssW - 50, 50, 28);

          // 一句话介绍
          ctx.fillStyle = '#1f2330';
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
            ctx.fillStyle = '#f0f1f5';
            this.roundRect(ctx, tx, ty - 14, tw, 22, 11);
            ctx.fill();
            ctx.fillStyle = '#4a5070';
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

          ctx.fillStyle = '#8a8f9c';
          ctx.font = '11px sans-serif';
          ctx.fillText('长按识别 · 和 TA 的 AI 聊聊', 20, cssH - 30);

          resolve();
        });
    });
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
        const canvas = res[0].node;
        wx.canvasToTempFilePath({
          canvas,
          success: (r) => {
            wx.saveImageToPhotosAlbum({
              filePath: r.tempFilePath,
              success: () => wx.showToast({ title: '已保存', icon: 'success' }),
              fail: (e) => {
                if (e.errMsg.includes('auth deny')) {
                  wx.showToast({ title: '需要相册权限', icon: 'none' });
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
