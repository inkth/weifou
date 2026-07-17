# 微否 · weifou

> 每个人的 AI 分身 —— 别人加你微信前，先和你的 AI 聊聊。

一个 24 小时在线、越来越懂你的 AI 分身主页。项目包含微信小程序、Flutter 客户端与 Go 服务端。

## 工程结构

```
weifou/
├─ weifou-miniapp/   # 微信小程序（原生 WXML/WXSS/JS）
├─ weifou-app/       # Flutter 客户端
└─ weifou-server/    # Go 后端（Gin + GORM + Redis + DeepSeek）
```

## 已有能力

1. **AI 生成主页** — 根据用户资料生成一句话介绍、完整介绍、人格标签和访客引导问题。
2. **AI 分身问答** — 访客与分身对话；Redis 滑窗限流；输入输出双向内容安全。
3. **知识库成长** — 主人补充资料，分身答不上来的问题自动形成知识缺口。
4. **问答箱** — 分身即时回答，访客可请本人补充；不涉及付费。
5. **访客线索** — 访客可交换名片、留下意向或获取主人公开的联系方式。
6. **分享传播** — 微信分享卡片、朋友圈和带小程序码的分享海报。
7. **访问洞察** — 主人查看 PV、UV、问答数和访客列表。
8. **能力课程会员** — 会员解锁平台自营的 AI 能力课程；与个人分身问答相互独立。

项目不提供打赏、付费真人咨询、音视频通话、咨询预约或相关分账能力。

## 快速开始

### 后端

本地需要 PostgreSQL 和 Redis：

```bash
docker run -d --name weifou-pg -e POSTGRES_USER=weifou -e POSTGRES_PASSWORD=weifou -e POSTGRES_DB=weifou -p 5432:5432 postgres:16
docker run -d --name weifou-redis -p 6379:6379 redis:7

cd weifou-server
cp .env.example .env
go mod tidy
go run ./cmd/server
```

API 默认位于 `http://localhost:3000/api`，健康检查为 `/healthz`。

### 小程序

1. 用微信开发者工具打开 `weifou-miniapp`。
2. 将 `project.config.json` 中的 `appid` 改为自己的小程序 AppID。
3. 修改 `utils/config.js` 的 `API_BASE`。
4. 真机和提审环境需配置 HTTPS 后端域名及小程序服务器域名白名单。

## 关键配置

| 变量 | 说明 |
|------|------|
| `WX_APPID` / `WX_APPSECRET` | 小程序凭证 |
| `JWT_SECRET` | JWT 签名密钥，生产环境必须使用强随机值 |
| `DATABASE_URL` | PostgreSQL 连接串 |
| `REDIS_URL` | Redis 连接串 |
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥 |
| `CHAT_FREE_QUOTA_PER_DAY` | 每位访客对每个主页的每日免费问答上限 |
| `WXV_*` | 小程序会员虚拟支付配置 |
| `WXPAY_*` | 会员备用支付通道配置 |

## 主要 API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/auth/login` | 微信登录 |
| GET | `/api/user/me` | 当前用户与主页状态 |
| POST | `/api/profile` | 创建或更新主页并生成分身 |
| GET | `/api/profile/mine` · `/api/profile/:id` | 本人/公开主页 |
| POST | `/api/chat/:profileId/ask` | 向分身提问 |
| POST | `/api/chat/:profileId/lead` | 留下访客意向 |
| POST | `/api/async-question/qabox` | 问答箱提问 |
| POST | `/api/connection/:profileId` | 交换名片 |
| POST | `/api/visit/:profileId` | 记录访问 |
| GET | `/api/visit/stats/mine` | 查看访问统计 |
| GET | `/api/share/bundle/:profileId` | 获取海报数据与小程序码 |
| GET | `/api/membership/status` | 查询会员状态 |
| GET · POST | `/api/agency/application` | 查询或提交代理商注册资料（自动通过） |
| GET | `/api/agency/dashboard` | 代理商邀请码、邀请与付费转化数据 |
| POST | `/api/agency/bind` | 用户首次绑定代理商邀请码 |
| GET | `/api/agency/qrcode` | 获取代理商专属小程序码 |

代理商入口为独立小程序页面 `pages/agency-register/index`，不出现在普通用户 Tab 中；招商二维码可通过 `inviteCode` 或小程序码 `scene` 参数预填推荐编号。
代理商推广邀请码采用从 `1112` 开始递增的 4 位数字，由数据库事务锁保证并发分配唯一。

## 提审提示

- 生成式 AI 服务需按实际业务完成内容安全、算法备案或登记等合规要求。
- 隐私协议应覆盖微信用户标识、访客 IP 哈希、问答内容、上传语音和会员订单记录。
- 会员属于虚拟权益，各端必须使用符合平台规则的支付方式与入口控制。

## 许可证

私有项目，未授权请勿外传。
