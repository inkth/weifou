#!/usr/bin/env bash
#
# weifou-server — 一键部署脚本(模仿 deepclaw/server/deploy.sh,适配 weifou)
#
# 用法:
#   ./deploy.sh            # 全量:rsync 源码 + 重建 server 镜像 + 重启 + reload 共享 nginx
#   ./deploy.sh --quick    # 快速:只重建&重启(跳过 rsync)
#   ./deploy.sh --logs     # tail server 日志
#   ./deploy.sh --status   # 看状态
#
# 与 deepclaw 的关键差异(务必注意):
#   * weifou 的 .env 以【服务器端】为唯一真源(密钥只在 /opt/weifou-server/.env)。
#     故 rsync 排除 .env/certs/acme,--delete 不会动它们。改密钥请直接编辑服务器上的 .env。
#   * 公网 HTTPS 由共享 oneclaw-nginx 反代;weifou-server 重建后容器 IP 可能变,
#     oneclaw-nginx 缓存旧 IP,故部署末尾 reload 一次让它重新解析(不改 nginx.conf,reload 安全)。
#
set -euo pipefail

# ─── 配置 ──────────────────────────────────────────────────────
REMOTE_USER="ubuntu"
REMOTE_HOST="101.35.215.28"
REMOTE_DIR="/opt/weifou-server"
COMPOSE="docker-compose.prod.yml"
CONTAINER="weifou-server"
SHARED_NGINX="oneclaw-nginx"          # 全机 80/443 持有者,weifou 经它反代
HOST_HEALTH_PORT="8083"               # host 端口(→容器 8080)

LOCAL_DIR="$(cd "$(dirname "$0")" && pwd)"
SSH_KEY="$(cd "$LOCAL_DIR/.." && pwd)/server.pem"   # 密钥在 weifou 项目根目录

SSH_CMD="ssh -i $SSH_KEY -o StrictHostKeyChecking=no -o ConnectTimeout=10"

# ─── 颜色 ──────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log()  { echo -e "${GREEN}[✓]${NC} $*"; }
warn() { echo -e "${YELLOW}[!]${NC} $*"; }
err()  { echo -e "${RED}[✗]${NC} $*" >&2; }
info() { echo -e "${CYAN}[→]${NC} $*"; }

# ─── 函数 ──────────────────────────────────────────────────────
check_ssh() {
    [ -f "$SSH_KEY" ] || { err "找不到 SSH 私钥: $SSH_KEY"; exit 1; }
    info "测试 SSH 连接..."
    $SSH_CMD "$REMOTE_USER@$REMOTE_HOST" "echo ok" &>/dev/null \
        || { err "无法连接 $REMOTE_HOST,检查密钥/网络"; exit 1; }
    log "SSH 连接正常"
}

show_status() {
    info "$REMOTE_HOST 上的 weifou 状态:"
    $SSH_CMD "$REMOTE_USER@$REMOTE_HOST" "
        echo '── 容器 ──'
        docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E 'NAMES|weifou'
        echo ''
        echo '── 磁盘/内存 ──'; df -h / | tail -1; free -h | head -2
        echo ''
        echo '── 本地健康检查 ──'
        curl -s http://localhost:$HOST_HEALTH_PORT/healthz || echo '健康检查失败'
    "
    echo ''
    info "公网 HTTPS 健康检查:"
    curl -s -m10 https://api.weifou.com/healthz -w '  → HTTP %{http_code}\n' || true
}

sync_files() {
    info "同步源码 $LOCAL_DIR/ → $REMOTE_HOST:$REMOTE_DIR ..."
    # 排除:服务器端真源(.env/备份)、密钥、TLS 证书、acme 续期状态、构建产物。
    # 这些被 --delete 误删会导致掉密钥/证书无法续期,务必排除。
    # 注意 renew-cert.sh(root crontab 每天调用)【现在在仓库里】,必须让 rsync 正常同步它:
    # 它曾两次只存在于服务器上而被 --delete 清掉(2026-06-15 那次续期静默断了 25 天)。
    # 别再给它加 --exclude —— basename 模式会连仓库里的副本一起挡掉,等于永远发不上去。
    rsync -avz --delete \
        --exclude '.git' \
        --exclude '.env' \
        --exclude '.env.bak.*' \
        --exclude 'server.pem' \
        --exclude '*.pem' \
        --exclude 'certs/' \
        --exclude 'acme/' \
        --exclude 'bin/' \
        --exclude 'tmp/' \
        --exclude '*.test' \
        -e "$SSH_CMD" \
        "$LOCAL_DIR/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/"
    log "源码已同步(.env/certs/acme 保持服务器端不动)"
}

rebuild_and_restart() {
    info "重建镜像并重启 server(postgres/redis 不动)..."
    $SSH_CMD "$REMOTE_USER@$REMOTE_HOST" "
        set -e
        cd $REMOTE_DIR
        test -f .env || { echo '[✗] 服务器缺少 .env(/opt/weifou-server/.env),先创建并填密钥'; exit 1; }

        echo '── 构建 server 镜像 ──'
        docker compose -f $COMPOSE build server

        echo '── 重启 server 容器 ──'
        docker compose -f $COMPOSE up -d server

        echo '── 等待健康检查(最多 60s) ──'
        HEALTHY=0
        for i in \$(seq 1 12); do
            sleep 5
            S=\$(docker inspect --format='{{.State.Health.Status}}' $CONTAINER 2>/dev/null || echo unknown)
            echo \"  尝试 \$i/12: \$S\"
            [ \"\$S\" = healthy ] && { HEALTHY=1; break; }
        done
        if [ \"\$HEALTHY\" -ne 1 ]; then
            echo '── 错误:60s 内未健康,最近日志: ──'
            docker logs --tail 30 $CONTAINER 2>&1
            exit 1
        fi

        # weifou-server 容器 IP 可能变,reload 共享 nginx 让它重新解析上游(不改配置,reload 安全)。
        echo '── reload 共享 nginx($SHARED_NGINX) ──'
        docker exec $SHARED_NGINX nginx -s reload 2>/dev/null && echo '  reloaded' || echo '  [!] reload 跳过(容器不在?手动检查)'

        echo ''
        docker ps --format 'table {{.Names}}\t{{.Status}}' | grep -E 'NAMES|weifou'
    "
    log "部署完成"
}

tail_logs() {
    info "tail server 日志(Ctrl+C 退出)..."
    $SSH_CMD "$REMOTE_USER@$REMOTE_HOST" "docker logs -f --tail 50 $CONTAINER"
}

# ─── 主流程 ────────────────────────────────────────────────────
main() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║   weifou-server 一键部署             ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════╝${NC}"
    echo ""
    check_ssh
    case "${1:-full}" in
        --status|-s) show_status ;;
        --logs|-l|--tail-only|-t) tail_logs ;;
        --quick|-q) warn "快速模式:跳过 rsync"; rebuild_and_restart ;;
        *)
            sync_files
            rebuild_and_restart
            echo ""; log "🎉 部署成功!  https://api.weifou.com/healthz"
            echo ""; info "常用:"
            echo "  ./deploy.sh --status   # 状态"
            echo "  ./deploy.sh --logs     # 日志"
            echo "  ./deploy.sh --quick    # 不重新 rsync,只重建重启"
            ;;
    esac
}
main "$@"
