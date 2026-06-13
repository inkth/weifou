# weifou-server 生产部署说明

线上：**https://api.weifou.com**（腾讯云 101.35.215.28，Ubuntu 24.04，多项目共享宿主，与 jifou/oneclaw/deepclaw 并存）。

## 拓扑

```
公网 → oneclaw-nginx(80/443,全机唯一反代/SSL终止)
        ├─ test.oneclaw.club  → oneclaw
        ├─ api.jifou.com      → jifou
        └─ api.weifou.com     → weifou-server:8080   ← 本项目
weifou 栈(docker compose,/opt/weifou-server):
  weifou-server (host 8083→8080)  weifou-postgres  weifou-redis   网络 weifou-net
```
> 80/443 只有 oneclaw-nginx 一个持有者，故 weifou 的 server 块并入它的 `nginx.conf`，
> 并把 oneclaw-nginx `docker network connect weifou-net` 以解析 `weifou-server`。

## 首次部署

```bash
# 1) 上传源码
rsync -az --exclude='.git' --exclude='*.pem' --exclude='.env' \
  -e "ssh -i server.pem" ./ ubuntu@101.35.215.28:/opt/weifou-server/
# 2) 服务器上写 .env(见 .deploy.env.example,填 DB_PASSWORD/JWT_SECRET 等)
# 3) 构建启动
ssh -i server.pem ubuntu@101.35.215.28 \
  'cd /opt/weifou-server && sudo docker compose -f docker-compose.prod.yml up -d --build'
# 健康检查:curl http://127.0.0.1:8083/healthz  → {"ok":true}(首次自动 AutoMigrate)
```

## 域名 + HTTPS（HTTP-01 webroot，Let's Encrypt 自动续期）

证书 webroot 复用 oneclaw 已挂载的 certs 目录：`/opt/oneclaw-server/server/certs/acme-webroot`。

```bash
# 共享 nginx 加 weifou 的 :80(含 ACME 验证) + :443 块(见 nginx-weifou.conf),并:
sudo docker network connect weifou-net oneclaw-nginx
# 签发(acme.sh 容器,EC-256):
sudo docker run --rm -v /opt/weifou-server/acme:/acme.sh \
  -v /opt/oneclaw-server/server/certs:/certs neilpang/acme.sh:latest \
  --issue -d api.weifou.com -w /certs/acme-webroot --server letsencrypt --keylength ec-256
# 安装到 certs 目录(被 nginx 当 fullchain/privkey 读):
sudo docker run --rm -v /opt/weifou-server/acme:/acme.sh \
  -v /opt/oneclaw-server/server/certs:/certs neilpang/acme.sh:latest \
  --install-cert -d api.weifou.com --ecc \
  --key-file /certs/api.weifou.com.key.pem \
  --fullchain-file /certs/api.weifou.com.fullchain.pem --reloadcmd ":"
# 自动续期:root crontab `17 4 * * * /opt/weifou-server/renew-cert.sh`(--cron + nginx -s reload)
```

## ⚠️ 改 oneclaw-nginx 的注意点（inode 陷阱）

`oneclaw-nginx` 把 `nginx.conf` 以**单文件 bind-mount** 挂载。任何「换 inode」式写入
（`sed -i`、`cp`、编辑器另存、本项目/jifou 的 deploy 脚本）都会让运行中的容器与新文件脱钩，
此后 `nginx -s reload` 读到的是**旧文件**。改完 `nginx.conf` 必须：

```bash
# 先用一次性容器在两张网络上校验完整文件(避免重启后起不来拖垮三个产品):
cid=$(sudo docker create --network oneclaw-server_oneclaw-net \
  -v /opt/oneclaw-server/server/nginx.conf:/etc/nginx/nginx.conf:ro \
  -v /opt/oneclaw-server/server/certs:/etc/nginx/certs:ro nginx:1.27-alpine nginx -t)
sudo docker network connect weifou-net "$cid"; sudo docker start -a "$cid"; sudo docker rm "$cid"
# 校验 exit 0 后,重启使其重挂当前文件:
sudo docker restart oneclaw-nginx
```
> 仅证书 `.pem` 更新走目录挂载，不受 inode 影响，`nginx -s reload` 即可（故续期无需 restart）。
> 改前务必备份：`cp -a nginx.conf nginx.conf.bak.$(date +%s)`。

## 日常运维 —— 用一键脚本 `weifou-server/deploy.sh`

```bash
./deploy.sh            # 全量:rsync 源码 + 重建 server + 等健康 + reload 共享 nginx
./deploy.sh --quick    # 只重建&重启(跳过 rsync)
./deploy.sh --status   # 状态(含公网 HTTPS 健康检查)
./deploy.sh --logs     # tail 日志
```
> 脚本自动:用项目根 `server.pem` 登录、排除 `.env`/`certs`/`acme`(服务器端真源,防 `--delete` 误删)、
> 健康轮询、**部署末尾 reload oneclaw-nginx**(weifou-server 重建后 IP 可能变,否则上游 502)。
> 改密钥直接编辑服务器 `/opt/weifou-server/.env` 后 `./deploy.sh --quick`。

手动等价命令(脚本内部即这些):
```bash
ssh -i server.pem ubuntu@101.35.215.28 'cd /opt/weifou-server && \
  docker compose -f docker-compose.prod.yml build server && \
  docker compose -f docker-compose.prod.yml up -d server && \
  docker exec oneclaw-nginx nginx -s reload'
```

## 上线前仍需补的密钥（当前为空，服务可启动但对应功能未启用）

`WX_APPID/SECRET`、`WX_MOBILE_APPID/SECRET`、`DEEPSEEK_API_KEY`、`WXPAY_*`(+证书放 certs/)、`TRTC_*`、`COS_*`。
客户端 API base 也需切到 `https://api.weifou.com/api`（小程序 `utils/config.js` 仍是 localhost；
Flutter 用 `--dart-define=API_BASE=https://api.weifou.com/api`）。
