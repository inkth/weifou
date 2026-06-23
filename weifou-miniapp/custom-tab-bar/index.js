// 自定义底部 tabBar（毛玻璃悬浮）。
// 微信只在 app.json tabBar.list 内的页面渲染该组件——沉浸/详情页（chat、agent-chat、
// profile…）天然不显示底栏，「沉浸页隐藏 tab」由此自动成立，无需逐页隐藏逻辑。
// 各 tab 页在 onShow 里 this.getTabBar().setData({ selected }) 同步高亮态。
Component({
  data: {
    selected: 0,
    list: [
      { pagePath: '/pages/discover/index', icon: '🤖', text: '分身' },
      { pagePath: '/pages/messages/index', icon: '💬', text: '消息' },
      { pagePath: '/pages/me/index', icon: '👤', text: '我的' },
    ],
  },
  methods: {
    onTap(e) {
      const idx = e.currentTarget.dataset.index;
      if (idx === this.data.selected) return;
      wx.switchTab({ url: this.data.list[idx].pagePath });
    },
  },
});
