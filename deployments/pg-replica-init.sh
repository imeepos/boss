#!/bin/bash
# PostgreSQL 从节点初始化脚本
# 从主节点执行基础备份并配置流复制

set -e

PG_MASTER_HOST="192.168.0.102"
PG_MASTER_PORT="25432"
PG_USER="boss"
PG_PASSWORD="boss"

# 等待主节点就绪
until pg_isready -h $PG_MASTER_HOST -p $PG_MASTER_PORT -U $PG_USER; do
  echo "等待主节点就绪..."
  sleep 2
done

# 如果数据目录为空,从主节点执行基础备份
if [ -z "$(ls -A /var/lib/postgresql/data)" ]; then
  echo "从主节点执行基础备份..."
  PGPASSWORD=$PG_PASSWORD pg_basebackup \
    -h $PG_MASTER_HOST \
    -p $PG_MASTER_PORT \
    -U $PG_USER \
    -D /var/lib/postgresql/data \
    -Fp -Xs -P -R

  # 配置从节点
  cat >> /var/lib/postgresql/data/postgresql.auto.conf <<EOF
primary_conninfo = 'host=$PG_MASTER_HOST port=$PG_MASTER_PORT user=$PG_USER password=$PG_PASSWORD application_name=$(hostname)'
primary_slot_name = '$(hostname)_slot'
hot_standby = on
EOF

  # 创建恢复信号文件
  touch /var/lib/postgresql/data/standby.signal
fi

echo "从节点初始化完成"
