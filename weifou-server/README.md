# weifou-server

微否后端（**Go**：Gin + GORM + go-redis + DeepSeek + 微信支付V3 + 腾讯云TRTC）。

> 本项目由早期 NestJS 版重写为 Go（已替代，NestJS 版已删除）。重写动因与权衡见 `~/.claude/plans/ai-aipage-snappy-beacon.md` 附录。

## 技术栈

| 层 | 选型 |
|----|------|
| Web | Gin v1.10 |
| ORM | GORM + AutoMigrate（Postgres） |
| 缓存/限流 | go-redis v9 |
| JWT | golang-jwt v5 |
| 定时任务 | robfig/cron v3 |
| 配置 | godotenv + 环境变量 |
| 微信支付V3 / TRTC UserSig | 标准库 crypto/zlib 自实现 |

## 启动

```bash
cp .env.example .env   # 填 WX/DeepSeek/支付/TRTC 等
go mod tidy
go run ./cmd/server
# http://localhost:3000/api ；健康检查 /healthz
```

首次启动自动 `AutoMigrate` 建表。证书放 `certs/`（已 gitignore）。

## 目录结构

```
cmd/server/main.go            # 入口：加载配置→连库→启动 cron→注册路由
internal/
  config/                     # 环境变量
  database/                   # GORM 连接 + AutoMigrate
  models/                     # 全部表模型（GORM）
  redisx/                     # Redis 客户端
  httpx/                      # 统一响应 {success,code,message,data} + 错误类型
  middleware/                 # JWTAuth / OptionalJWT
  idgen/                      # 主键/单号生成
  wechat/                     # code2session + access_token + msg_sec_check
  deepseek/                   # LLM 客户端
  trtc/                       # UserSig（TLSSigAPIv2）
  wxpay/                      # 微信支付V3：签名/验签/AES-GCM/下单/退款/关单/分账
  auth user profile persona chat visit share consult payment rtc tasks
  app/                        # 组装所有 handler + 路由 + cron
```

每个业务包暴露 `NewHandler(...)` + `Register(rg *gin.RouterGroup)`，由 `internal/app` 统一装配。

## 约定

- **响应格式**：成功 `{success:true,data}`，失败 `{success:false,code,message}`，401 → `UNAUTHORIZED`，与小程序 `utils/request.js` 完全兼容。
- **鉴权**：JWT claims `{sub,openid}`，HS256。
- **路由**：见下（`internal/app/route_test.go` 做注册冲突回归）。

## API

| Method | Path |
|--------|------|
| POST | /api/auth/login |
| GET | /api/user/me |
| POST | /api/profile · /api/profile/regenerate |
| GET | /api/profile/mine · /api/profile/:id · /api/profile/:id/contact |
| PATCH | /api/profile/contact |
| POST | /api/chat/:profileId/ask |
| POST/GET | /api/visit/:profileId · /api/visit/stats/mine |
| GET | /api/share/bundle/:profileId |
| GET/PATCH | /api/consult/setting/mine · /api/consult/setting |
| GET | /api/consult/pricing/:profileId · /api/consult/sessions/mine |
| POST/GET/DELETE | /api/consult/slots · /slots/mine · /slots/:slotId · /slots/public/:profileId |
| POST | /api/payment/tip · /consult · /refund · /notify · /refund-notify |
| GET | /api/payment/orders/:id |
| POST | /api/rtc/consult/:sessionId/join · /start · /end |

## 已实现的付费/通话能力

- 打赏（赠予，不解锁权益）、付费咨询（选档预约 → 支付 → 进房）
- 微信支付V3：JSAPI 下单、回调验签+AES-GCM 解密+幂等+金额校验、退款及回调、关单
- 分账：通话结束触发给 host 分账并完结解冻（`PROFIT_SHARING_ENABLED`）
- TRTC：UserSig 签发 + 进房时间窗校验 + 通话计时
- cron：超时关单、过期档期清理、爽约自动退款

## 待完善

- 订阅消息提醒、host 收入明细页、退款幂等分布式锁
- DeepSeek 流式（当前同步返回）
- 第三方 AI 头像 / 腾讯云 COS 上传（ENV 已预留）

## 校验

```bash
go build ./...      # 编译
go vet ./...        # 静态检查
go test ./...       # 路由注册冒烟
gofmt -l internal cmd   # 格式（应无输出）
```
