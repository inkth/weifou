Component({
  // 让导航根节点直接参与页面的 flex 布局，避免 glass-easel 下组件宿主
  // 没有把内部高度传给父级，导致课程舞台顶进状态栏。
  options: {
    virtualHost: true,
  },

  properties: {
    title: { type: String, value: '' },
    subtitle: { type: String, value: '' },
    theme: { type: String, value: 'light' },
    transparent: { type: Boolean, value: false },
  },

  data: {
    statusBarHeight: 20,
    navigationHeight: 44,
    sideWidth: 88,
    isRoot: false,
  },

  lifetimes: {
    attached() {
      this._measure();
      this.setData({ isRoot: getCurrentPages().length <= 1 });
    },
  },

  methods: {
    _measure() {
      let info = {};
      try {
        info = typeof wx.getWindowInfo === 'function'
          ? wx.getWindowInfo()
          : wx.getSystemInfoSync();
      } catch (e) {}

      const statusBarHeight = info.statusBarHeight || 20;
      const windowWidth = info.windowWidth || 375;
      let navigationHeight = 44;
      let sideWidth = 88;

      try {
        const capsule = wx.getMenuButtonBoundingClientRect();
        if (capsule && capsule.height && capsule.top) {
          navigationHeight = capsule.height + 2 * Math.max(capsule.top - statusBarHeight, 4);
          // 标题左右保留对称空间，避免与微信右上角胶囊相撞。
          sideWidth = Math.max(windowWidth - capsule.left + 8, 76);
        }
      } catch (e) {}

      this.setData({ statusBarHeight, navigationHeight, sideWidth });
    },

    onBack() {
      if (getCurrentPages().length > 1) {
        wx.navigateBack();
        return;
      }
      wx.switchTab({
        url: '/pages/discover/index',
        fail: () => wx.reLaunch({ url: '/pages/discover/index' }),
      });
    },
  },
});
