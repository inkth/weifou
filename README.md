# 微否 · weifou

> 每个人的 AI 助理 —— 别人加你微信前，先和你的 AI 聊聊。

一个 24 小时在线、越来越懂你的 AI 助理主页。微信小程序 MVP。

## 工程结构

```
weifou/
├─ weifou-miniapp/   # 微信小程序（原生 WXML/WXSS/JS）
├─ weifou-server/    # Go 后端（Gin + GORM + Redis + DeepSeek）
└─ README.md
```

## MVP 功能（已落地）

1. **AI 生成主页** — 填表 → DeepSeek 生成「一句话介绍 / 完整介绍 / 人格标签」
2. **AI 问答** — 访客向主页 AI 提问；Redis 滑窗限流；内容安全双向过滤
3. **分享卡片** — `onShareAppMessage` / `onShareTimeline`
4. **分享海报** — canvas 端合成（头像 + 一句话 + 小程序码）
5. **访客统计** — 仅本人可见的 PV / UV / 问答数
6. **联系本人** — 微信号 / 手机号，本人决定是否公开

## 付费功能（已落地）

7. **打赏** — 访客自愿赠予（6/18/66/88 元预设 + 留言）。**定性为赠予，不解锁任何权益**，iOS/安卓均可。
8. **付费语音/视频咨询** — 本人开启并设 30/60 分钟价格 → 访客下单支付 → 进入腾讯云 TRTC 通话。**定性为真人服务（非虚拟商品），iOS 亦可收款**。
   - 微信支付 V3（JSAPI 下单 + 回调验签 + AES-GCM 解密，纯 Go 标准库 crypto 实现）
   - TRTC UserSig 后端签发（TLSSigAPIv2），通话计时与时长记账

> ⚠️ **iOS 合规红线**：打赏文案与逻辑绝不可绑定权益解锁（会员/AI 次数/付费内容），否则会被认定为「虚拟支付」遭驳回甚至封支付权限。虚拟权益类（会员/Pro）若要做，iOS 端只能隐藏入口、仅安卓开放。

## 快速开始（开发）

### 后端

需要本地运行 PostgreSQL 和 Redis。推荐用 docker：

```bash
docker run -d --name weifou-pg -e POSTGRES_USER=weifou -e POSTGRES_PASSWORD=weifou -e POSTGRES_DB=weifou -p 5432:5432 postgres:16
docker run -d --name weifou-redis -p 6379:6379 redis:7
```

启动后端：

```bash
cd weifou-server
cp .env.example .env  # 填入 WX_APPID/WX_APPSECRET/DEEPSEEK_API_KEY
go mod tidy
go run ./cmd/server   # 首次启动自动 AutoMigrate 建表
# http://localhost:3000/api ；健康检查 /healthz
```

### 小程序

1. 用 [微信开发者工具](https://developers.weixin.qq.com/miniprogram/dev/devtools/download.html) 打开 `weifou-miniapp` 目录
2. 修改 `project.config.json` 的 `appid` 为你的小程序 AppID
3. 开发阶段：
   - 工具中勾选「不校验合法域名」可以直接连本地后端
   - 提审/真机调试需要把后端域名加入小程序后台的「服务器域名」白名单（必须 HTTPS 且 ICP 备案）

## 关键配置

后端 `.env`：

| 变量 | 说明 |
|------|------|
| `WX_APPID` / `WX_APPSECRET` | 小程序凭证 |
| `JWT_SECRET` | 生产环境务必改成强随机串 |
| `DATABASE_URL` | PostgreSQL 连接串 |
| `REDIS_URL` | Redis 连接串 |
| `DEEPSEEK_API_KEY` | DeepSeek 控制台获取 |
| `CHAT_FREE_QUOTA_PER_DAY` | 每访客每主页每日免费问答上限（默认 10） |
| `AVATAR_API_*` | 第三方 AI 头像服务（占位，未接入） |

## API 速览

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/auth/login` | `wx.login` code 换 JWT |
| GET | `/api/user/me` | 当前用户信息 + 是否已有主页 |
| POST | `/api/profile` | 创建/更新主页并触发 AI 生成 |
| POST | `/api/profile/regenerate` | AI 重新生成主页 |
| GET | `/api/profile/mine` | 我的主页（含填写原文） |
| GET | `/api/profile/:id` | 公开主页（访客视图） |
| PATCH | `/api/profile/contact` | 设置联系方式 |
| GET | `/api/profile/:id/contact` | 公开联系方式（未开放则 403） |
| POST | `/api/chat/:profileId/ask` | 向主页 AI 提问 |
| POST | `/api/visit/:profileId` | 记录访问 |
| GET | `/api/visit/stats/mine` | 我的访客统计 |
| GET | `/api/share/bundle/:profileId` | 海报合成所需数据（含小程序码 base64） |
| POST | `/api/payment/tip` | 打赏下单，返回 JSAPI 支付参数 |
| POST | `/api/payment/consult` | 付费咨询下单 |
| POST | `/api/payment/notify` | 微信支付回调（验签 + 解密 + 置 paid） |
| GET | `/api/payment/orders/:id` | 订单状态（含 consultSessionId） |
| GET | `/api/consult/setting/mine` | 我的咨询开关/定价 |
| PATCH | `/api/consult/setting` | 设置咨询开关/定价 |
| GET | `/api/consult/pricing/:profileId` | 访客查询某主页咨询定价 |
| GET | `/api/consult/sessions/mine` | 我相关的通话记录 |
| POST | `/api/payment/refund` | 申请咨询退款（仅未开始通话） |
| POST | `/api/payment/refund-notify` | 退款回调 |
| POST | `/api/consult/slots` | host 批量新增可约档期 |
| GET | `/api/consult/slots/mine` | host 查看自己档期 |
| DELETE | `/api/consult/slots/:slotId` | host 删除档期（未被约） |
| GET | `/api/consult/slots/public/:profileId` | 访客查看可约档期 |
| POST | `/api/rtc/consult/:sessionId/join` | 签发 TRTC UserSig + 进房参数（含时间窗校验） |
| POST | `/api/rtc/consult/:sessionId/start` | 标记通话开始 |
| POST | `/api/rtc/consult/:sessionId/end` | 结束通话并记录时长（触发分账） |

> 付费咨询下单接口 `POST /api/payment/consult` 现需传 `{ profileId, slotId }`（按档期预约），不再传 durationMin。

## 付费/音视频配置

后端 `.env` 追加（见 `.env.example`）：

| 变量 | 说明 |
|------|------|
| `WXPAY_MCHID` | 微信支付商户号 |
| `WXPAY_API_V3_KEY` | APIv3 密钥（32 字节） |
| `WXPAY_CERT_SERIAL` | 商户 API 证书序列号 |
| `WXPAY_PRIVATE_KEY_PATH` | 商户私钥 PEM 路径（放 `certs/`，已 gitignore） |
| `WXPAY_PLATFORM_CERT_PATH` | 微信支付平台证书/公钥（回调验签） |
| `WXPAY_NOTIFY_URL` | 公网 HTTPS 回调地址 → `/api/payment/notify` |
| `TIP_MAX_AMOUNT` | 打赏金额上限（分） |
| `PLATFORM_FEE_RATE` | 平台抽成比例（0-1，分账用） |
| `PROFIT_SHARING_ENABLED` | 是否启用微信分账（需开通分账权限） |
| `ORDER_TIMEOUT_MIN` | 未支付订单超时关闭分钟数 |
| `CALL_EARLY_JOIN_MIN` / `CALL_GRACE_MIN` | 提前进房 / 迟到宽限分钟数 |
| `TRTC_SDK_APPID` | 腾讯云 TRTC 应用 SDKAppID |
| `TRTC_SECRET_KEY` | TRTC 密钥（用于签发 UserSig） |

### 退款 / 分账 / 排期 / 定时任务

- **排期**：host 在「设置 → 管理可约档期」开放时间段；访客在主页「选择时间预约」选档下单。下单即预占档期，支付超时由 cron 释放。
- **退款**：仅 `consult` 订单、且通话**未开始**（session=pending）可全额退款；打赏（赠予）不可退。退款成功自动取消会话并释放档期。
- **分账**：`PROFIT_SHARING_ENABLED=true` 时，咨询订单支付走资金冻结（`settle_info.profit_sharing`）；**通话真实结束后**才给 host 分账并完结解冻，平台按 `PLATFORM_FEE_RATE` 抽成。因此退款只可能发生在分账之前，逻辑互斥不冲突。
- **定时任务**（`robfig/cron`，见 `internal/tasks`）：
  - 每分钟关闭超时未支付订单（调微信关单）并释放档期
  - 每 5 分钟清理过期未约档期
  - 每 5 分钟对「host 爽约」（预约时间+宽限过后仍未开始）的会话自动全额退款

> ⚠️ 微信分账需在商户平台单独**开通分账权限**，且 host 作为个人接收方（`PERSONAL_OPENID`）须在同一 appid 下。未开通时置 `PROFIT_SHARING_ENABLED=false`，资金正常结算到商户号，由你线下与 host 结算。

小程序端：
- 通话页依赖 TRTC 小程序 SDK，需手动放入 `weifou-miniapp/libs/trtc-wx.js`（见该目录 README）
- 需在小程序后台开通**实时音视频类目**与 `live-pusher`/`live-player` 组件权限
- 微信支付需在小程序后台关联商户号并配置支付目录

## 提审准备清单

- 小程序 AppID + AppSecret
- 服务类目（涉及生成式 AI 通常需要「文娱-其他」或更具体类目，必要时备案算法/上传 ICP）
- 后端域名 HTTPS + ICP 备案
- 小程序后台「服务器域名」白名单：`request` 添加后端域名
- 内容安全 API 已默认接入（`msg_sec_check`）
- 隐私协议（含「微信用户标识」「访客 IP（哈希）」「问答内容」「通话/支付记录」等使用说明）
- **微信支付**：企业/个体工商户主体，开通微信支付并关联小程序，配置支付目录与 APIv3 密钥
- **实时音视频**：开通「实时音视频」服务类目，申请 `live-pusher`/`live-player` 组件权限
- **iOS 合规**：打赏不得绑定权益；虚拟权益类（会员/Pro）iOS 端不得出现付费入口

## 后续路线（不在 MVP）

- AI 头像生成（第三方 API 已预留 ENV）
- SSE 流式逐字渲染
- 9.9 元高级身份升级（虚拟权益，注意 iOS 限制）
- 多版本主页 / AI 访客分析
- 企业版

## 已落地（付费/通话进阶）

- ✅ **退款**：咨询未开始可全额退款 + 退款回调 + 释放档期
- ✅ **分账**：通话结束触发微信分账给 host + 完结解冻（需开通分账权限）
- ✅ **排期撮合**：host 开放档期 → 访客选档下单 → 按预约时间进房（含时间窗校验）
- ✅ **定时任务**：超时订单关闭、过期档期清理、爽约自动退款

## 已知待完善（付费/通话）

- **TRTC 房间策略**：当前用字符串房间 `consult_<orderId>`；若 SDK 仅支持数字房间需改造
- **订阅消息提醒**：未接入通话开始前的订阅消息提醒（建议后续加）
- **分账分润对账页**：host 端暂无收入明细页，仅记账于 `profit_shares` 表
- **重复退款幂等**：退款回调与同步返回并发时已做状态保护，但建议加分布式锁加固

## 许可证

私有项目，未授权请勿外传。
