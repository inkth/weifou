#!/usr/bin/env bash
#
# 一次性运维:删除「找对象/择偶测试」退役功能残留的三张孤儿表。
#
# 背景:2026-07-05 找对象功能整包退役(前端+后端 internal/dating/ + models.go 模型均删),
#       但 AutoMigrate 不删表,dating_quizzes / dating_results / compat_results 一直留在生产库。
#       models.go 里对应模型已不存在,故删表后重启不会被 AutoMigrate 重建 —— 删掉即永久干净。
#
# 用法(与 deploy.sh 一样在【本地】跑,自动 SSH 进服务器):
#   ./ops-drop-dating-tables.sh            # 交互:先备份 → 打印行数 → 要你确认 → 删表 → 复核
#   ./ops-drop-dating-tables.sh --yes      # 跳过确认(仍会先备份),用于非交互
#
# ⚠️ DROP TABLE 不可逆。脚本会在删除前把三张表 pg_dump 到本地备份文件;要回滚就把它 psql 回去。
#
set -euo pipefail

# ─── 配置(与 deploy.sh 对齐) ───────────────────────────────────
REMOTE_USER="ubuntu"
REMOTE_HOST="101.35.215.28"
PG_CONTAINER="weifou-postgres"
TABLES=(dating_quizzes dating_results compat_results)

LOCAL_DIR="$(cd "$(dirname "$0")" && pwd)"
SSH_KEY="$(cd "$LOCAL_DIR/.." && pwd)/server.pem"   # 密钥在 weifou 项目根目录
SSH="ssh -i $SSH_KEY -o StrictHostKeyChecking=accept-new $REMOTE_USER@$REMOTE_HOST"

BACKUP_FILE="$LOCAL_DIR/dating-tables-backup-$(date +%Y%m%d-%H%M%S).sql"

ASSUME_YES=0
[[ "${1:-}" == "--yes" ]] && ASSUME_YES=1

# 容器内用 POSTGRES_USER/POSTGRES_DB(compose 注入),不硬编码库名密码。
pg_psql() { $SSH "docker exec $PG_CONTAINER sh -c 'psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -v ON_ERROR_STOP=1 $1'"; }

# 拼 -t 参数
DUMP_TARGS=""; DROP_LIST=""
for t in "${TABLES[@]}"; do
  DUMP_TARGS="$DUMP_TARGS -t $t"
  DROP_LIST="${DROP_LIST:+$DROP_LIST, }$t"
done

echo "==> 目标服务器:$REMOTE_USER@$REMOTE_HOST  容器:$PG_CONTAINER"
echo "==> 待删表:$DROP_LIST"
echo

echo "==> [1/4] 删除前行数:"
pg_psql "-c \"SELECT relname, n_live_tup FROM pg_stat_user_tables WHERE relname IN ('${TABLES[0]}','${TABLES[1]}','${TABLES[2]}') ORDER BY relname;\""
echo

echo "==> [2/4] 备份三张表到本地:$BACKUP_FILE"
$SSH "docker exec $PG_CONTAINER sh -c 'pg_dump -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" $DUMP_TARGS'" > "$BACKUP_FILE"
if [[ ! -s "$BACKUP_FILE" ]]; then
  echo "  [!] 备份文件为空,中止(未删任何东西)。请检查表名/连接。" >&2
  exit 1
fi
echo "  备份完成($(wc -c < "$BACKUP_FILE" | tr -d ' ') 字节)。回滚方式见文件末尾提示。"
echo

if [[ "$ASSUME_YES" -ne 1 ]]; then
  echo "==> [3/4] 即将 DROP:$DROP_LIST (不可逆)"
  read -r -p "    确认删除?输入大写 YES 继续:" ans
  [[ "$ans" == "YES" ]] || { echo "  已取消(备份保留:$BACKUP_FILE)。"; exit 0; }
else
  echo "==> [3/4] --yes 已跳过确认,执行 DROP:$DROP_LIST"
fi
pg_psql "-c \"DROP TABLE IF EXISTS $DROP_LIST;\""
echo "  DROP 完成。"
echo

echo "==> [4/4] 复核(应无残留 dating_/compat_ 表):"
pg_psql "-c \"\\\\dt dating_quizzes|dating_results|compat_results\"" || true
echo
echo "完成。回滚:  cat '$BACKUP_FILE' | $SSH \"docker exec -i $PG_CONTAINER sh -c 'psql -U \\\"\\\$POSTGRES_USER\\\" -d \\\"\\\$POSTGRES_DB\\\"'\""
