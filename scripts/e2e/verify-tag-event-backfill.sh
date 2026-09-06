#!/usr/bin/env bash
# e2e-tag-event-backfill.sh -- 标签事件流历史回填端到端校验(P3-T4/迁移 000189,102 真库)。
# 断言: B1 零事件资产数=0;B2 在绑标签缺当前绑定对 BIND 事件数=0;
#       B3 回填行带 backfill 标记且 CREATE 行 tag_id=0 哨兵;
#       B4 幂等: 重放迁移体(与 000189 up 相同的 NOT EXISTS 守卫语句)
#       零新增,重复执行本脚本不产生重复事件。
# 只读为主: 唯一写动作是 B4 的幂等重放(NOT EXISTS 守卫,重跑零新增),不造数不清理。
# 信号: 收尾 "E2E-TAG-EVENT-BACKFILL RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-tag-event-backfill.sh   环境: SSH_HOST(默认 102) 依赖: ssh
set -u

env_or() { local v; v=$(printenv "$1" 2>/dev/null); if [ -n "$v" ]; then echo "$v"; else echo "$2"; fi; }
SSH_HOST=$(env_or SSH_HOST "imeepos@192.168.0.102")

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi }

# 幂等重放体: 必须与 migrations/000189_tag_event_backfill.up.sql 的两条 INSERT 保持一致
# (NOT EXISTS 守卫 = 天然幂等,重放零新增;此处是机械断言,不靠自觉)。
replay() {
  sql <<SQL
INSERT INTO tag_events(tag_id, asset_id, action, detail, changed)
SELECT 0, a.id, 'CREATE', 'backfill: asset created before event stream',
       jsonb_build_object('backfill', '000189', 'source', 'migration')
  FROM assets a
 WHERE NOT EXISTS (SELECT 1 FROM tag_events e WHERE e.asset_id = a.id);
INSERT INTO tag_events(tag_id, asset_id, action, detail, changed)
SELECT t.id, t.bound_asset_id, 'BIND', 'backfill: binding predates event stream',
       jsonb_build_object('bound_asset_id', t.bound_asset_id,
                          'backfill', '000189', 'source', 'migration')
  FROM tags t
 WHERE t.status = 'BOUND' AND t.bound_asset_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM tag_events e
                    WHERE e.tag_id = t.id AND e.asset_id = t.bound_asset_id
                      AND e.action = 'BIND');
SQL
}

stats() { # 总事件数|回填 CREATE 数|回填 BIND 数
  sql <<SQL
SELECT count(*), count(*) FILTER (WHERE changed->>'backfill' = '000189' AND action = 'CREATE'),
       count(*) FILTER (WHERE changed->>'backfill' = '000189' AND action = 'BIND') FROM tag_events;
SQL
}

echo "标签事件流历史回填校验(P3-T4/000189) @ $SSH_HOST"
stats_out=$(stats) || { echo "E2E-TAG-EVENT-BACKFILL RESULT: FAIL (db unreachable)"; exit 1; }
total0=$(echo "$stats_out" | cut -d"|" -f1); create0=$(echo "$stats_out" | cut -d"|" -f2); bind0=$(echo "$stats_out" | cut -d"|" -f3)
echo "基线: 总事件=$total0 回填CREATE=$create0 回填BIND=$bind0"

# B1 零事件资产数=0
b1=$(sql <<SQL
SELECT count(*) FROM assets a WHERE NOT EXISTS (SELECT 1 FROM tag_events e WHERE e.asset_id = a.id);
SQL
)
assert_eq "B1" "0" "$b1" "零事件资产数(回填后必须为 0)"

# B2 在绑标签缺当前绑定对 BIND 事件数=0
b2=$(sql <<SQL
SELECT count(*) FROM tags t WHERE t.status = 'BOUND' AND t.bound_asset_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM tag_events e WHERE e.tag_id = t.id AND e.asset_id = t.bound_asset_id AND e.action = 'BIND');
SQL
)
assert_eq "B2" "0" "$b2" "在绑标签无 BIND 事件数"

# B3 回填行形态: CREATE 行 tag_id=0 哨兵、标记齐全
b3=$(sql <<SQL
SELECT count(*) FROM tag_events WHERE changed->>'backfill' = '000189' AND action = 'CREATE' AND tag_id <> 0;
SQL
)
assert_eq "B3" "0" "$b3" "CREATE 回填行 tag_id 非哨兵 0 的行数"

# B4 幂等重放: 零新增(重复执行本脚本不产生重复事件的机械证明)
replay || { bad "B4" "重放 SQL 执行失败"; }
stats2=$(stats)
total1=$(echo "$stats2" | cut -d"|" -f1); create1=$(echo "$stats2" | cut -d"|" -f2); bind1=$(echo "$stats2" | cut -d"|" -f3)
assert_eq "B4-total" "$total0" "$total1" "重放后总事件数不变"
assert_eq "B4-create" "$create0" "$create1" "重放后回填 CREATE 行数不变"
assert_eq "B4-bind" "$bind0" "$bind1" "重放后回填 BIND 行数不变"

echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ]; then
  echo "E2E-TAG-EVENT-BACKFILL RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-TAG-EVENT-BACKFILL RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1
