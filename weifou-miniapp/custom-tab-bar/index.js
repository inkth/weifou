// 自定义底部 tabBar（毛玻璃悬浮）。
// 微信只在 app.json tabBar.list 内的页面渲染该组件——沉浸/详情页（chat、agent-chat、
// profile…）天然不显示底栏，「沉浸页隐藏 tab」由此自动成立，无需逐页隐藏逻辑。
// 各 tab 页在 onShow 里 this.getTabBar().setData({ selected }) 同步高亮态。
const { fetchTodoCount } = require('../utils/badge');

Component({
  data: {
    selected: 0,
    dotUser: false, // 「我的」红点：有待办（待答提问/新线索/知识缺口）时亮
    // icon = styles/icons.wxss 里的字形后缀（.ic-bot / .ic-chat / .ic-user）
    list: [
      { pagePath: '/pages/discover/index', icon: 'bot', text: '分身' },
      { pagePath: '/pages/explore/index', icon: 'sparkle', text: '发现' },
      { pagePath: '/pages/me/index', icon: 'user', text: '我的' },
    ],
  },
  // 每个 tab 页 onShow 时该组件的 pageLifetimes.show 都会触发——借此刷新红点。
  // fetchTodoCount 自带 15s 缓存，频繁切 tab 不会反复打接口。
  pageLifetimes: {
    show() { this.refreshBadge(); },
  },
  methods: {
    onTap(e) {
      const idx = e.currentTarget.dataset.index;
      if (idx === this.data.selected) return;
      wx.switchTab({ url: this.data.list[idx].pagePath });
    },
    // force=true 绕过 15s 缓存拉最新（页面完成鉴权加载后调用）；缺省走缓存，切 tab 轻量。
    async refreshBadge(force = false) {
      const count = await fetchTodoCount({ force }).catch(() => 0);
      const dotUser = count > 0;
      if (dotUser !== this.data.dotUser) this.setData({ dotUser });
    },
  },
});
