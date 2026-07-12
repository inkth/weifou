// 自定义底部 tabBar（毛玻璃悬浮）。
// 微信只在 app.json tabBar.list 内的页面渲染该组件——沉浸/详情页（chat、agent-chat、
// profile…）天然不显示底栏，「沉浸页隐藏 tab」由此自动成立，无需逐页隐藏逻辑。
// 各 tab 页在 onShow 里 this.getTabBar().setData({ selected }) 同步高亮态。
Component({
  data: {
    selected: 0,
    // icon = styles/icons.wxss 里的字形后缀（.ic-bot / .ic-sparkle / .ic-inbox / .ic-user）
    list: [
      { pagePath: '/pages/discover/index', icon: 'bot', text: '首页' },
      { pagePath: '/pages/explore/index', icon: 'sparkle', text: '技能' },
      { pagePath: '/pages/inbox/index', icon: 'inbox', text: '消息' },
      { pagePath: '/pages/me/index', icon: 'user', text: '我的' },
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
