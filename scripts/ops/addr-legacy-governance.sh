#!/usr/bin/env bash
# addr-legacy-governance: user_addresses.address_path 旧格式一次性治理(苏晚裁定,2026-08-28 议题收口)。
# 旧格式=人类可读地名词路径(如 ncr.manila.barangay_6);新 scheme 见迁移 000153。
# 用法: scripts/ops/addr-legacy-governance.sh [--apply]   # 默认 dry-run 绝不写库
# apply 需主持人审批后另行执行。环境: SSH_HOST 可覆盖(缺省 imeepos@192.168.0.102);
# psql 容器 boss-infra-postgres-1,SQL 一律单引号 heredoc 走 stdin(禁叠引号)。
#
# 【判定规则 1: 旧格式识别】(依据 000153_materialize_ncr_address_tree.up.sql + 102 实测)
#   新 scheme path 标签 = PSGC 码去横杠小写,层级树 L1-L3 为锚定段:
#   L1 大区(如 ph1300000000)、L2 市(ph1380600000)、L3 Barangay(ph1380601006)。
#   L4/L5 是服务节点自定义标签(实测 dmv_sunset/dmv_b8),不以 ph 开头,但整条路径
#   首段恒为 ph+纯数字。故旧格式判定:
#       address_path <> '' AND address_path !~ '^ph[0-9]+(\.|$)'
#   空串 = 历史自由文本地址(000152 注释),不在本治理范围。
#   树内另有 demo_root/idem*/w8c* 等测试根(首段非 ph),同样按旧格式扫出,如实 unmatched。
#
# 【判定规则 2: 旧路径→新 path 匹配】
#   按 '.' 拆段;每段归一化: trim → 下划线转空格 → 连续空格合一 → 小写。
#   自顶向下: 第 i 段在 addresses level=i 匹配 name(归一化后严格等值),
#   level=1 限定 parent_id IS NULL,i>1 限定 parent_id=上段候选(逐层收敛)。
#   末段候选数 0/1/>1 → unmatched/matched/ambiguous;候选以 EXISTS(path::text=) 显式验证。
#   不做别名/子串/模糊匹配: ncr≠国家首都区、manila≠City of Manila 归一化后不等,
#   宁缺毋滥,如实标 unmatched 交人工复核(实测全树无 ncr/manila 同名节点)。
#
# 【人工复核步骤】
#   1. 跑 dry-run,对每行核对 candidate 与业务语义(面包屑名链)。
#   2. unmatched 行: 结合 user_addresses.detail/addr_code 与地理证据人工定位节点;
#      ambiguous 行: 逐个候选核对。
#   3. 人工裁定后,对零星行用显式模板(先 SELECT 验证 path 存在再 UPDATE):
#        UPDATE user_addresses SET address_path='<已验证新path>'
#         WHERE id=<id> AND address_path='<原值>';
#   4. --apply 仅更新 matched 单候选且验证通过的行;先 pg_dump 整表备份,
#      单事务 ON_ERROR_STOP,UPDATE 带乐观锁谓词 AND address_path=旧值 防并发漂移。
#   5. apply 属写库操作,须主持人审批后另行执行。
set -u
MODE="dry-run"
case "${1:-}" in
  "") ;;
  --apply) MODE="apply" ;;
  *) echo "用法: $0 [--apply]"; exit 1 ;;
esac
SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
PGC="boss-infra-postgres-1"
psql_run() { ssh "$SSH_HOST" "docker exec -i $PGC psql -U boss -d boss -v ON_ERROR_STOP=1"; }

echo "== addr-legacy-governance ($MODE) =="
echo "-- 概览(判定: 非空 且 首段非 ph+数字) --"
psql_run <<'SQL' || { echo "FAIL: dry-run SQL 失败或 DB 不可达(见上方 ERROR)"; exit 2; }
SELECT 'user_addresses 总行数: ' || count(*) FROM user_addresses;
SELECT 'legacy 行数: ' || count(*) FROM user_addresses
 WHERE address_path <> '' AND address_path !~ '^ph[0-9]+(\.|$)';
SELECT '树内 ph 锚定根: ' || count(*) FROM addresses WHERE level = 1 AND path::text ~ '^ph[0-9]+$';

WITH RECURSIVE
legacy AS (
  SELECT u.id, u.customer_id, u.addr_code, u.contact, u.detail, u.address_path
  FROM user_addresses u
  WHERE u.address_path <> '' AND u.address_path !~ '^ph[0-9]+(\.|$)'
),
seg AS (
  SELECT l.id AS ua_id, l.address_path, s.seg, s.seg_idx,
         lower(regexp_replace(regexp_replace(trim(s.seg), '_', ' ', 'g'), ' +', ' ', 'g')) AS nseg
  FROM legacy l
  CROSS JOIN LATERAL unnest(string_to_array(l.address_path, '.')) WITH ORDINALITY AS s(seg, seg_idx)
),
tot AS (SELECT ua_id, max(seg_idx) AS n_segs FROM seg GROUP BY ua_id),
match AS (
  SELECT g.ua_id, g.seg_idx, a.id AS addr_id, a.path::text AS cand
  FROM seg g JOIN addresses a
    ON a.level = 1 AND a.parent_id IS NULL
   AND lower(regexp_replace(regexp_replace(trim(a.name), '_', ' ', 'g'), ' +', ' ', 'g')) = g.nseg
  WHERE g.seg_idx = 1
  UNION ALL
  SELECT g.ua_id, g.seg_idx, a.id, a.path::text
  FROM match m
  JOIN seg g ON g.ua_id = m.ua_id AND g.seg_idx = m.seg_idx + 1
  JOIN addresses a
    ON a.parent_id = m.addr_id
   AND lower(regexp_replace(regexp_replace(trim(a.name), '_', ' ', 'g'), ' +', ' ', 'g')) = g.nseg
),
reach AS (SELECT ua_id, max(seg_idx) AS deepest FROM match GROUP BY ua_id),
final AS (
  SELECT m.ua_id, m.cand FROM match m JOIN tot t ON t.ua_id = m.ua_id
  WHERE m.seg_idx = t.n_segs
),
verify AS (
  SELECT f.ua_id, f.cand,
         EXISTS(SELECT 1 FROM addresses a WHERE a.path::text = f.cand) AS ok
  FROM final f
),
fs AS (
  SELECT ua_id, count(*) AS n_final, bool_and(ok) AS ok_all,
         string_agg(cand || ' [' || (SELECT string_agg(a.name, ' > ' ORDER BY a.level)
                                     FROM addresses a WHERE a.path @> cand::ltree) || ']',
                    E'\n    或 ' ORDER BY cand) AS cands
  FROM verify GROUP BY ua_id
)
SELECT l.id   AS "id",
       l.customer_id,
       l.addr_code,
       l.contact,
       l.address_path AS "旧path",
       CASE WHEN COALESCE(fs.n_final, 0) = 0 THEN 'unmatched'
            WHEN fs.n_final = 1 AND fs.ok_all THEN 'matched'
            WHEN fs.n_final = 1 THEN 'matched(验证失败)'
            ELSE 'ambiguous' END AS "状态",
       COALESCE(fs.cands, '-') AS "候选新path[各级name]",
       CASE
         WHEN COALESCE(fs.n_final, 0) = 0 THEN
           '中断于第 ' || (COALESCE(r.deepest, 0) + 1) || ' 段 ''' ||
           COALESCE(fb.seg, '?') || ''' (已匹配到 level ' || COALESCE(r.deepest, 0) ||
           ',该层无归一化同名节点)'
         WHEN fs.n_final = 1 AND fs.ok_all THEN '单候选且 EXISTS 验证通过'
         WHEN fs.n_final = 1 THEN 'EXISTS 验证未通过,不可用'
         ELSE '末段 ' || fs.n_final || ' 个候选,需人工裁定'
       END AS "说明"
FROM legacy l
JOIN tot t ON t.ua_id = l.id
LEFT JOIN fs ON fs.ua_id = l.id
LEFT JOIN reach r ON r.ua_id = l.id
LEFT JOIN LATERAL (SELECT g.seg FROM seg g
                    WHERE g.ua_id = l.id AND g.seg_idx = COALESCE(r.deepest, 0) + 1) fb ON TRUE
ORDER BY l.id;
SQL

if [ "$MODE" != "apply" ]; then
  echo "dry-run 完成,未写库。apply 需主持人审批,确认后执行: $0 --apply"
  exit 0
fi

STAMP=$(date +%Y%m%d-%H%M%S)
echo "== 备份 user_addresses 到 $SSH_HOST:/tmp/boss_addr_gov_backup_$STAMP.sql =="
ssh "$SSH_HOST" "docker exec -i $PGC pg_dump -U boss -d boss -t user_addresses \
  > /tmp/boss_addr_gov_backup_$STAMP.sql" || { echo "FAIL: backup failed, abort"; exit 2; }

echo "== 单事务更新(仅 matched 单候选;乐观锁谓词防并发漂移) =="
psql_run <<'SQL'
BEGIN;
CREATE TEMP TABLE gov_applied AS
WITH RECURSIVE
legacy AS (
  SELECT u.id, u.address_path
  FROM user_addresses u
  WHERE u.address_path <> '' AND u.address_path !~ '^ph[0-9]+(\.|$)'
),
seg AS (
  SELECT l.id AS ua_id, s.seg_idx,
         lower(regexp_replace(regexp_replace(trim(s.seg), '_', ' ', 'g'), ' +', ' ', 'g')) AS nseg
  FROM legacy l
  CROSS JOIN LATERAL unnest(string_to_array(l.address_path, '.')) WITH ORDINALITY AS s(seg, seg_idx)
),
tot AS (SELECT ua_id, max(seg_idx) AS n_segs FROM seg GROUP BY ua_id),
match AS (
  SELECT g.ua_id, g.seg_idx, a.id AS addr_id, a.path::text AS cand
  FROM seg g JOIN addresses a
    ON a.level = 1 AND a.parent_id IS NULL
   AND lower(regexp_replace(regexp_replace(trim(a.name), '_', ' ', 'g'), ' +', ' ', 'g')) = g.nseg
  WHERE g.seg_idx = 1
  UNION ALL
  SELECT g.ua_id, g.seg_idx, a.id, a.path::text
  FROM match m
  JOIN seg g ON g.ua_id = m.ua_id AND g.seg_idx = m.seg_idx + 1
  JOIN addresses a
    ON a.parent_id = m.addr_id
   AND lower(regexp_replace(regexp_replace(trim(a.name), '_', ' ', 'g'), ' +', ' ', 'g')) = g.nseg
),
final AS (
  SELECT m.ua_id, m.cand FROM match m JOIN tot t ON t.ua_id = m.ua_id
  WHERE m.seg_idx = t.n_segs
),
plan AS (
  SELECT f.ua_id, f.cand, l.address_path AS old_path
  FROM final f
  JOIN legacy l ON l.id = f.ua_id
  WHERE (SELECT count(*) FROM final x WHERE x.ua_id = f.ua_id) = 1
    AND EXISTS(SELECT 1 FROM addresses a WHERE a.path::text = f.cand)
)
SELECT p.ua_id AS id, p.old_path, p.cand FROM plan p;

WITH upd AS (
  UPDATE user_addresses u SET address_path = g.cand
  FROM gov_applied g
  WHERE u.id = g.id AND u.address_path = g.old_path
  RETURNING u.id
)
SELECT '计划更新: ' || (SELECT count(*) FROM gov_applied) ||
       ' 行, 实际更新: ' || count(*) || ' 行(差额=并发漂移被谓词跳过)' FROM upd;
COMMIT;

SELECT g.id, u.customer_id, g.old_path, g.cand AS new_path, u.address_path AS now_value
FROM gov_applied g JOIN user_addresses u ON u.id = g.id ORDER BY g.id;
SQL
rc=$?
[ $rc -eq 0 ] && echo "apply 完成(备份: /tmp/boss_addr_gov_backup_$STAMP.sql)" \
              || echo "apply 失败,事务已回滚(备份仍在)"
exit $rc
