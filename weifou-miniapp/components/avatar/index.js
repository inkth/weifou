const { getPreset, initial, loadLottieData } = require('../../utils/avatars');

let lottie = null;
try {
  const mod = require('../../libs/lottie-miniprogram');
  if (mod && typeof mod.loadAnimation === 'function') {
    lottie = mod;
  } else if (mod && mod.default && typeof mod.default.loadAnimation === 'function') {
    lottie = mod.default;
  }
} catch (e) {
  lottie = null;
}

function lottieReady() {
  return !!lottie && typeof lottie.setup === 'function' && typeof lottie.loadAnimation === 'function';
}

Component({
  properties: {
    preset: { type: String, value: '' }, // 形象 id
    name: { type: String, value: '' }, // 用于取首字
    seed: { type: String, value: '' }, // 默认形象兜底种子（一般传 profileId）
    size: { type: Number, value: 120 }, // rpx
    static: { type: Boolean, value: false }, // true 时 lottie 不实例化播放器，渲染 css 回退（列表场景省性能）
    active: { type: Boolean, value: false }, // 活跃态（如 AI 回复时），idle 动效更强
    state: { type: String, value: 'idle' }, // 'idle' | 'thinking' | 'speaking'，image 形象据此切表情图
  },

  data: {
    mode: 'css', // 'css' | 'lottie' | 'image' | 'toon'
    boxStyle: '',
    sizeStyle: '',
    iniStyle: '',
    char: '微',
    anim: 'flow',
    look: 'steady', // toon 卡通形象的五官气质
    stateImages: [], // [{state, src}]，image 形象的各状态图
    curState: 'idle', // 当前生效状态（取不到对应图时回落 idle）
  },

  observers: {
    'preset, name, seed, size, static': function () {
      this.compute();
    },
    state: function () {
      this.updateState();
    },
  },

  lifetimes: {
    attached() {
      this.compute();
    },
    detached() {
      this._destroyLottie();
    },
  },

  methods: {
    compute() {
      const { preset, name, seed, size } = this.properties;
      const p = getPreset(preset, seed || name);
      const colors = p.colors.join(', ');

      const isImage = p.type === 'image' && p.images && p.images.idle;
      const wantLottie = !isImage && lottieReady() && !this.properties.static && p.type === 'lottie';

      let mode = 'css';
      let stateImages = [];
      if (isImage) {
        mode = 'image';
        this._imagesMap = p.images;
        stateImages = Object.keys(p.images)
          .filter((k) => p.images[k])
          .map((k) => ({ state: k, src: p.images[k] }));
      } else {
        this._imagesMap = null;
        if (p.type === 'toon') {
          mode = 'toon'; // CSS 卡通脸：眨眼/沉思/说话由 curState 驱动
        } else if (wantLottie) {
          mode = 'lottie';
        }
      }

      this.setData({
        boxStyle: `width:${size}rpx;height:${size}rpx;background:linear-gradient(135deg, ${colors});background-size:220% 220%;`,
        sizeStyle: `width:${size}rpx;height:${size}rpx;`,
        iniStyle: `font-size:${Math.round(size * 0.4)}rpx;line-height:${size}rpx;`,
        char: initial(name),
        anim: p.anim || 'flow',
        look: p.look || 'steady',
        mode,
        stateImages,
      });

      this.updateState();
      this._destroyLottie();
      if (mode === 'lottie') {
        this._initLottie(p);
      }
    },

    // 根据 state 选择生效的表情图；该状态无图时回落 idle
    updateState() {
      const map = this._imagesMap;
      let s = this.properties.state || 'idle';
      if (map && !map[s]) s = 'idle';
      this.setData({ curState: s });
    },

    // 任一状态图加载失败 → 回退 css 形象，绝不留白/破图
    _imgError() {
      this.setData({ mode: 'css' });
    },

    _initLottie(preset) {
      // 等待 canvas 渲染后再取节点
      wx.nextTick(() => {
        this.createSelectorQuery()
          .select('#lottie')
          .node()
          .exec((res) => {
            const node = res && res[0] && res[0].node;
            if (!node) {
              this.setData({ mode: 'css' });
              return;
            }
            const win = wx.getWindowInfo();
            const dpr = win.pixelRatio || 2;
            const px = this.properties.size * (win.windowWidth / 750);
            node.width = px * dpr;
            node.height = px * dpr;
            const ctx = node.getContext('2d');
            ctx.scale(dpr, dpr);
            try {
              lottie.setup(node);
            } catch (e) {
              this.setData({ mode: 'css' });
              return;
            }
            loadLottieData(preset)
              .then((animationData) => {
                if (this.data.mode !== 'lottie') return; // 期间已切走
                this._ani = lottie.loadAnimation({
                  loop: true,
                  autoplay: true,
                  animationData,
                  rendererSettings: { context: ctx },
                });
              })
              .catch(() => {
                // 没配/加载失败 → 回退 css 形象
                this.setData({ mode: 'css' });
              });
          });
      });
    },

    _destroyLottie() {
      if (this._ani) {
        try { this._ani.destroy(); } catch (e) {}
        this._ani = null;
      }
    },
  },
});
