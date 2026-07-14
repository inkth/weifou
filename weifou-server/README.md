# weifou-server

微否 Go 后端：Gin + GORM/PostgreSQL + Redis + DeepSeek。

## 技术栈

| 层 | 选型 |
|----|------|
| Web | Gin |
| ORM | GORM + AutoMigrate（PostgreSQL） |
| 缓存/限流 | go-redis |
| JWT | golang-jwt |
| 定时任务 | robfig/cron |
| AI | DeepSeek Chat API |
| 会员支付 | 微信小程序虚拟支付；微信支付 V3 作为现有备用通道 |

## 启动

```bash
cp .env.example .env
go mod tidy
go run ./cmd/server
```

API 默认位于 `http://localhost:3000/api`，健康检查为 `/healthz`。首次启动自动执行 `AutoMigrate`。

## 主要业务包

```
internal/
  auth user profile persona     # 登录与 AI 分身资料
  answer chat asyncq            # 分身对话与免费问答箱
  connection visit share        # 名片关系、访问与分享
  toolagent membership referral # 能力课程、会员和邀请奖励
  payment wxpay wxvpay          # 会员订单支付基础设施
  wechat                        # 微信登录、内容安全与订阅消息
  app tasks                     # 依赖装配、路由与定时任务
```

服务端不包含打赏、付费真人咨询、档期预约、音视频通话或咨询分账接口。

## 约定

- 成功响应：`{success:true,data}`。
- 失败响应：`{success:false,code,message}`。
- JWT claims 使用 `{sub,openid}`。
- `internal/app/route_test.go` 覆盖全部路由注册，防止 Gin 通配路由冲突。

## 校验

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l internal cmd
```
