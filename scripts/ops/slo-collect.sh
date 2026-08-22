#!/usr/bin/env bash
# SLO 采集脚本:在 102 上聚合 S4/S5/S6/S7 的 DB 侧 SLI,输出 JSON。
# 用法: ./scripts/ops/slo-collect.sh [ssh-host]   (默认 imeepos@192.168.0.102)
# 指标定义见 docs/ops/slo.md;本脚本只做采集,不判达标(由看板/报告消费)。
set -uo pipefail
HOST="${1:-imeepos@192.168.0.102}"

SQL=$(cat <<'EOF'
-- S4 订单完成率:近7天创建的订单中终态占比
SELECT json_build_object(
 's4_orders', (SELECT json_build_object(
   'window_7d_created', count(*),
   'done', count(*) FILTER (WHERE status = 'DONE'),
   'cancelled', count(*) FILTER (WHERE status IN ('CANCELLED','REJECTED'))
 ) FROM orders WHERE created_at > now() - interval '7 days'),
 's5_push', (SELECT json_build_object(
   'by_status', COALESCE(json_object_agg(status, c), '{}'::json),
   'window_7d', sum(c)
 ) FROM (SELECT status, count(*) c FROM push_records
        WHERE created_at > now() - interval '7 days' GROUP BY status) t),
 's5_admin_todo', (SELECT json_build_object(
   'open', count(*), 'older_24h', count(*) FILTER (WHERE created_at < now() - interval '24 hours')
 ) FROM admin_notifications WHERE category='todo' AND NOT resolved),
 's6_cdrs', (SELECT json_build_object(
   'total', count(*),
   'kafka_by_status', COALESCE(json_object_agg(kafka_status, c), '{}'::json),
   'billing_unbilled', count(*) FILTER (WHERE billing_status = 'UNBILLED')
 ) FROM (SELECT kafka_status, billing_status, count(*) OVER (PARTITION BY kafka_status) c
        FROM cdrs WHERE started_at > now() - interval '7 days') t),
 's7_reports', (SELECT json_build_object(
   'latest_daily_created', max(created_at)::text,
   'latest_daily_window', max(window_end)::text,
   'lag_seconds', EXTRACT(EPOCH FROM max(created_at) - max(window_end))::bigint,
   'on_time_cnt', count(*) FILTER (WHERE created_at < window_end + interval '8 hours')
 ) FROM (SELECT created_at, window_end FROM report_snapshots
        WHERE period='daily' AND window_end > now() - interval '7 days' ORDER BY window_end DESC LIMIT 3) r)
);
EOF
)

ssh -o BatchMode=yes "$HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atq" <<< "$SQL"
