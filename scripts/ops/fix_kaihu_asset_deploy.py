#!/usr/bin/env python3
# fix_kaihu_asset_deploy: 开户导入资产归属修复(W8;负责人批复 2026-09-07 方案 B)。
# 圈定指纹:批次=存量开户导入 + loid 非空 + 存在 LINKED 四码 + 资产仍 IN_STOCK。
# 动作:置 DEPLOYED + address_id=四码行级地址 + asset_lifecycles 轨迹(note=fix-kaihu-deploy)。
# 幂等:仅翻 IN_STOCK,轨迹行 NOT EXISTS 去重;CLOSED 拆机行无四码天然不在圈定集。
# 用法: fix_kaihu_asset_deploy.py backup|dryrun|apply|verify
#       backup: 受影响行 COPY CSV 经 ssh 落 102:~/backups/kaihu-fix-<ts>/(assets/lifecycles/quad)。
import shlex
import subprocess
import sys
import time

SSH_HOST = "imeepos@192.168.0.102"
STAMP = time.strftime("%Y%m%d_%H%M%S")
REMOTE_DIR = "backups/kaihu-fix-" + STAMP
NOTE = "fix-kaihu-deploy " + STAMP[:8]
Q = chr(39)


def ssh(cmd, timeout=120, stdin_text=None):
    r = subprocess.run(["ssh", "-o", "BatchMode=yes", SSH_HOST, cmd], capture_output=True, text=True, timeout=timeout, input=stdin_text)
    return r.returncode, r.stdout.strip(), r.stderr.strip()


def psql(sql):
    cmd = "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q"
    rc, out, err = ssh(cmd, stdin_text=sql)
    if rc != 0:
        raise RuntimeError("psql failed: " + out + err)
    return out


def psql_scalar(sql):
    cmd = "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc " + shlex.quote(sql)
    rc, out, err = ssh(cmd)
    if rc != 0:
        raise RuntimeError("psql failed: " + out + err)
    return out.strip()


TARGET_WHERE = "ql.status = " + Q + "LINKED" + Q + " AND a.status = " + Q + "IN_STOCK" + Q + " AND a.loid IS NOT NULL AND b.name = " + Q + "存量开户导入" + Q


def target_sql(select_cols, extra=""):
    return "SELECT " + select_cols + " FROM quad_links ql JOIN assets a ON a.id = ql.asset_id JOIN asset_batches b ON b.id = a.batch_id WHERE " + TARGET_WHERE + extra


def stage_backup():
    mkdir = "mkdir -p ~/" + REMOTE_DIR
    rc, out, err = ssh(mkdir)
    if rc != 0:
        raise RuntimeError("mkdir failed: " + err)
    tables = {
        "assets": "SELECT a.* FROM assets a JOIN quad_links ql ON ql.asset_id = a.id AND ql.status = " + Q + "LINKED" + Q + " JOIN asset_batches b ON b.id = a.batch_id WHERE " + TARGET_WHERE.replace("a.status = " + Q + "IN_STOCK" + Q, "a.status IN (" + Q + "IN_STOCK" + Q + "," + Q + "DEPLOYED" + Q + ")"),
        "asset_lifecycles": "SELECT al.* FROM asset_lifecycles al WHERE al.asset_id IN (" + target_sql("ql.asset_id") + ")",
        "quad_links": "SELECT ql.* FROM quad_links ql JOIN assets a ON a.id = ql.asset_id JOIN asset_batches b ON b.id = a.batch_id WHERE " + TARGET_WHERE,
    }
    for name, sel in tables.items():
        copy_sql = "COPY (" + sel + ") TO STDOUT WITH CSV HEADER"
        remote = "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -c " + shlex.quote(copy_sql) + " > ~/" + REMOTE_DIR + "/" + name + ".csv"
        rc, out, err = ssh(remote)
        if rc != 0:
            raise RuntimeError("backup " + name + " failed: " + err)
        cnt = ssh("wc -l < ~/" + REMOTE_DIR + "/" + name + ".csv")[1].strip()
        print("[backup] " + name + ".csv rows=" + cnt + " -> ~/" + REMOTE_DIR)
    print("[backup] done dir=~/" + REMOTE_DIR)


def stage_dryrun():
    total = psql_scalar(target_sql("count(DISTINCT ql.asset_id)"))
    print("[dryrun] 命中资产数 = " + total + " (预期 313)")
    if total != "313":
        print("[dryrun] WARN: 命中数不等于 313,人工核对后再 apply")
    closed = psql_scalar("SELECT count(*) FROM assets a JOIN asset_batches b ON b.id = a.batch_id WHERE b.name = " + Q + "存量开户导入" + Q + " AND a.loid = " + Q + "OWPAL51531" + Q + " AND a.status = " + Q + "IN_STOCK" + Q)
    print("[dryrun] CLOSED 拆机资产(应保持 IN_STOCK,不在圈定集)命中检查 loid=OWPAL51531 rows=" + closed)
    samples = psql_scalar(target_sql("ql.id, ql.asset_id, a.asset_code, a.status", " ORDER BY ql.id LIMIT 5"))
    print("[dryrun] 修复前样本 5 条 (ql_id,asset_id,asset_code,status):")
    print(samples)


def stage_apply():
    sql = chr(10).join([
        "BEGIN;",
        "UPDATE assets a SET status = " + Q + "DEPLOYED" + Q + ", address_id = t.address_id, updated_at = now()"
        " FROM (" + target_sql("DISTINCT ql.asset_id AS asset_id, ql.address_id") + ") t",
        " WHERE a.id = t.asset_id AND a.status = " + Q + "IN_STOCK" + Q + ";",
        "INSERT INTO asset_lifecycles(asset_id, status, address_id, note, changed_at)",
        "SELECT t.asset_id, " + Q + "DEPLOYED" + Q + ", t.address_id, " + Q + NOTE + Q + ", now()",
        " FROM (" + target_sql("DISTINCT ql.asset_id AS asset_id, ql.address_id") + ") t",
        " WHERE NOT EXISTS (SELECT 1 FROM asset_lifecycles al WHERE al.asset_id = t.asset_id AND al.note = " + Q + NOTE + Q + ");",
        "COMMIT;",
    ])
    psql(sql)
    flipped = psql_scalar("SELECT count(*) FROM asset_lifecycles WHERE note = " + Q + NOTE + Q)
    print("[apply] 轨迹修复行 = " + flipped + " (预期 313)")
    if flipped != "313":
        print("[apply] WARN: 修复行数不等于 313,人工核对")


def stage_verify():
    remain = psql_scalar(target_sql("count(DISTINCT ql.asset_id)"))
    print("[verify] 指纹圈定集剩余 IN_STOCK = " + remain + " (预期 0)")
    if remain != "0":
        raise RuntimeError("repair incomplete")
    note_rows = psql_scalar("SELECT count(*) FROM asset_lifecycles WHERE note LIKE " + Q + "fix-kaihu-deploy%" + Q)
    print("[verify] asset_lifecycles fix 标记行 = " + note_rows)
    deployed_where = TARGET_WHERE.replace("a.status = " + Q + "IN_STOCK" + Q, "a.status = " + Q + "DEPLOYED" + Q)
    addr_ok = psql_scalar("SELECT count(*) FROM quad_links ql JOIN assets a ON a.id = ql.asset_id JOIN asset_batches b ON b.id = a.batch_id WHERE " + deployed_where + " AND a.address_id IS NOT NULL AND a.address_id = ql.address_id")
    print("[verify] 抽样全量:DEPLOYED 且地址=四码地址 = " + addr_ok)
    closed = psql_scalar("SELECT a.status FROM assets a JOIN asset_batches b ON b.id = a.batch_id WHERE b.name = " + Q + "存量开户导入" + Q + " AND a.loid = " + Q + "OWPAL51531" + Q + " LIMIT 1")
    print("[verify] CLOSED 拆机资产状态(预期 IN_STOCK) = " + closed)


def main():
    stage = sys.argv[1] if len(sys.argv) > 1 else "dryrun"
    if stage == "backup":
        stage_backup()
    elif stage == "dryrun":
        stage_dryrun()
    elif stage == "apply":
        stage_apply()
    elif stage == "verify":
        stage_verify()
    else:
        print("unknown stage: " + stage + " (backup|dryrun|apply|verify)")
        sys.exit(2)


if __name__ == "__main__":
    main()
