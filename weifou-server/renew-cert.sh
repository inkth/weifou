#!/usr/bin/env bash
#
# weifou HTTPS 证书自动续期(acme.sh + webroot)
#
# 由服务器 root crontab 每天调用一次:
#   17 4 * * * /opt/weifou-server/renew-cert.sh >> /var/log/weifou-cert-renew.log 2>&1
#
# 【为什么这个文件必须留在仓库里】
#   它原本只存在于服务器上,先后两次被 deploy.sh 的 rsync --delete 清掉
#   (2026-06-15 那次断了 25 天没人发现,直到 07-10 排查磁盘时才暴露)。
#   现在纳入版本控制并由 deploy.sh 正常同步,--delete 不会再误删它。
#
# 【链路】
#   acme.sh 容器 --cron
#     ├─ 数据目录 /acme.sh      ← 宿主 /opt/weifou-server/acme      (账号 + 域名 conf)
#     └─ 证书目录 /certs        ← 宿主 /opt/oneclaw-server/server/certs
#          ├─ acme-webroot/               HTTP-01 挑战目录(oneclaw-nginx 对外暴露)
#          ├─ api.weifou.com.fullchain.pem
#          └─ api.weifou.com.key.pem
#   证书目录以只读挂进共享容器 oneclaw-nginx 的 /etc/nginx/certs,
#   域名 conf 里 Le_ReloadCmd 是空操作(:),所以 reload 由本脚本负责。
#
#   acme.sh 只在过了 Le_NextRenewTime 才真正续期,平时输出 "Skipping" 直接返回,
#   因此本脚本每天跑是安全的、幂等的。
#
set -euo pipefail

ACME_HOME="/opt/weifou-server/acme"           # acme.sh 数据目录(账号/域名 conf,勿动)
CERT_DIR="/opt/oneclaw-server/server/certs"   # 共享证书目录(ro 挂进 oneclaw-nginx)
ACME_IMAGE="neilpang/acme.sh"
NGINX_CONTAINER="oneclaw-nginx"               # 全机 80/443 持有者,weifou 经它反代
FULLCHAIN="$CERT_DIR/api.weifou.com.fullchain.pem"
KEY="$CERT_DIR/api.weifou.com.key.pem"

fingerprint() {
  # 证书+私钥的联合指纹;文件不存在时回落到 none(首次签发场景)
  cat "$FULLCHAIN" "$KEY" 2>/dev/null | sha256sum | awk '{print $1}' || echo none
}

[ -d "$ACME_HOME" ] || { echo "[✗] acme 数据目录不存在: $ACME_HOME"; exit 1; }
[ -d "$CERT_DIR" ]  || { echo "[✗] 证书目录不存在: $CERT_DIR"; exit 1; }

before="$(fingerprint)"

docker run --rm \
  -v "$ACME_HOME:/acme.sh" \
  -v "$CERT_DIR:/certs" \
  "$ACME_IMAGE" --cron --home /acme.sh

after="$(fingerprint)"

# 证书没变(绝大多数天)就不打扰 nginx;真续期了才 reload,让新证书生效。
if [ "$before" = "$after" ]; then
  echo "[=] 证书未变化,跳过 nginx reload"
  exit 0
fi

echo "[!] 证书已更新,reload $NGINX_CONTAINER"
docker exec "$NGINX_CONTAINER" nginx -t          # 先验配置,坏配置绝不 reload
docker exec "$NGINX_CONTAINER" nginx -s reload
echo "[✓] reload 完成,新证书已生效"
