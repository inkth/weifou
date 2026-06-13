#!/bin/bash
set -e
cd /Users/zhangkun/Project/weifou
OUT="软著申请材料/源代码-完整版.txt"
: > "$OUT"

emit() { # $1=base dir, $2=relative path
  {
    echo ""
    echo "//=================================================================="
    echo "// 文件: $2"
    echo "//=================================================================="
    cat -s "$1/$2"
  } >> "$OUT"
}

echo "/* ========== 微否软件 V1.0 客户端源代码（Flutter / Dart） ========== */" >> "$OUT"
for f in $( (cd weifou-app && { echo lib/main.dart; echo lib/app.dart; find lib/core lib/data lib/features -name "*.dart" | sort; } | awk '!seen[$0]++') ); do
  emit weifou-app "$f"
done

total=$(wc -l < "$OUT")
echo "DONE lines=$total pages~$((total/50))"
