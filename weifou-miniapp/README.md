# weifou-miniapp

原生微信小程序（不使用 npm 包，不依赖 Taro/uni-app），便于审核与维护。

## 打开

1. 微信开发者工具 → 导入项目 → 选择本目录
2. AppID 改为你自己的（`project.config.json` 的 `appid` 字段）
3. 编辑 `utils/config.js` 的 `API_BASE` 指向你的后端

## 页面

| 路径 | 说明 |
|------|------|
| `pages/index/index` | 首页：未创建引导创建，已创建跳转主页 |
| `pages/create/index` | 创建/编辑表单（基础信息 + 3 问） |
| `pages/generating/index` | 生成过场（路由占位，目前 create 同步阻塞） |
| `pages/profile/index` | 主页（自/他视图共用） |
| `pages/chat/index` | AI 问答 |
| `pages/poster/index` | 海报合成（canvas） |
| `pages/settings/index` | 联系方式 / 重新生成 / 退出登录 |

## 网络层

`utils/request.js` 统一注入 `Authorization: Bearer <token>`，401 自动清 token。

`utils/auth.js` 提供 `ensureLogin` 静默登录。`app.js` 启动即调用。

## 真机调试

- 开发阶段：开发者工具勾选「不校验合法域名」可直连本地后端
- 提审/真机：后端必须 HTTPS + ICP 备案，并在小程序后台「服务器域名」中加入 `request` 白名单

## 设计风格

- 主色 `#1f2330`，背景 `#f5f6fa`，强调圆角 24rpx
- 卡片均使用 `.card`，主按钮 `.btn-primary`，次按钮 `.btn-ghost`
- 字号体系：`title 36`、`subtitle 28`、`muted 26/24`
