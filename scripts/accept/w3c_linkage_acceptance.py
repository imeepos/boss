#!/usr/bin/env python3
# W3 收口 102 真实环境验收(P-INFRA-1 顺序五 W3 收口;审查 F2 + W8 遗留①).
# 前置:102 已部署含本收口的 main(CI deploy-102);本机 ssh 免密 imeepos@192.168.0.102.
# 断言:就绪探针(POST /assets/region-backfill 旧二进制 404 新二进制 200+code0,首次调用即回补)/
#       资源链导入网格归属联动(带归属链行导入->odn_grid 备案+MH 锚点设施+链行归属快照)/
#       投资测算网格行非空(API 设施计数与 SQL 直查一致)/覆盖模型可见(挂锚点设施码进覆盖页+网格行覆盖计数)/
#       资产区域快照回补(347 台回补 root 集团,幂等重跑零更新,前后样本)/审计留痕;
#       造数即清(finally 兜底)+孤儿巡检归零+quad 孤儿基线对比不新增.
import json
import random
import shlex
import subprocess
import sys
import time
import urllib.error
import urllib.request

BASE = "http://192.168.0.102:28080/api/admin/v1"
SSH_HOST = "imeepos@192.168.0.102"
PASS = 0
FAIL = 0
KEY = ""
TAG = "W3C" + str(random.randint(10000, 99999))


def ok(name):
    global PASS
    PASS += 1
    print("PASS: " + name)


def bad(name, detail=""):
    global FAIL
    FAIL += 1
    print("FAIL: " + name + ("  [" + detail + "]" if detail else ""))


def http(method, path, body=None, timeout=30):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header("Content-Type", "application/json")
    req.add_header("X-API-Key", KEY)
    data = json.dumps(body).encode() if body is not None else None
    try:
        with urllib.request.urlopen(req, data=data, timeout=timeout) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read().decode())
        except Exception:
            return e.code, None


def ssh(cmd, timeout=180, stdin_text=None):
    r = subprocess.run(["ssh", "-o", "BatchMode=yes", SSH_HOST, cmd],
                       capture_output=True, text=True, timeout=timeout, input=stdin_text)
    return r.returncode, r.stdout.strip(), r.stderr.strip()


def psql(sql):
    cmd = "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc " + shlex.quote(sql)
    rc, out, err = ssh(cmd)
    if rc != 0:
        raise RuntimeError("psql failed: " + out + err)
    return out


def q(v):
    return chr(39) + str(v) + chr(39)


def data_of(body):
    return (body or {}).get("data")


def code_of(body):
    return (body or {}).get("code")


def wait_ready(seconds=300):
    # 就绪探针:POST /assets/region-backfill 旧二进制 404(路由未注册),新二进制 200+code0.
    last = ""
    deadline = time.time() + seconds
    while time.time() < deadline:
        st, body = http("POST", "/assets/region-backfill", {})
        last = "status=" + str(st) + " code=" + str(code_of(body))
        if st == 200 and code_of(body) == 0:
            return body
        time.sleep(5)
    print("ready probe timeout; last=" + last)
    return None


def quad_orphans():
    st, body = http("GET", "/db-patrol/orphans")
    if st == 200 and code_of(body) == 0:
        for it in data_of(body).get("items", []):
            if "LINKED but asset not DEPLOYED" in str(it.get("check")):
                return int(it.get("orphans", 0))
    return -1


def pick_free_grid():
    st, body = http("GET", "/odn/grids?prvCode=PHL001&cityPrefix=MNL")
    taken = set()
    if st == 200 and code_of(body) == 0:
        for r in data_of(body) or []:
            taken.add(int(r.get("gridCode")))
    for c in range(60, 100):
        if c not in taken:
            return c
    raise RuntimeError("no free grid code, taken=" + str(sorted(taken)))


def create_address(label):
    st, body = http("POST", "/addresses", {"parentID": 0, "label": label.lower().replace("-", ""), "name": label})
    if st == 200 and code_of(body) == 0:
        aid = (data_of(body) or {}).get("id") or (data_of(body) or {}).get("addressId")
        if aid:
            return int(aid)
    row = psql("SELECT id FROM addresses WHERE name = " + q(label) + " ORDER BY id DESC LIMIT 1")
    return int(row)


def chain_row(prv, city, grid, occ, odb, obd, sdb, sbd):
    return {"rowNo": 0, "resourceStatus": "规划", "prvCode": prv, "cityPrefix": city, "gridCode": str(grid),
        "siteCode": TAG + "-SITE", "siteName": TAG, "oltCode": TAG + "-OLT", "odfCode": "", "odfPort": "",
        "occCode": occ, "odbCode": odb, "obdCode": obd, "split1Ratio": "1:8", "split1Port": "P01",
        "sdbCode": sdb, "sbdCode": sbd, "split2Ratio": "1:8", "split2Port": "P01", "totalSplit": "64",
        "fiberCode": TAG + "-F", "frTo": "", "portStatus": "可用", "layingMethod": "架空",
        "rowStatus": "不适用", "peceStatus": "不适用", "remark": TAG}


def import_chains(rows):
    st, body = http("POST", "/odn/resource-chains/import", {"exampleRowCount": 0, "rows": rows})
    if st == 200 and code_of(body) == 0:
        res = (data_of(body) or {}).get("result") or {}
        return res
    raise RuntimeError("import: " + str((st, body))[:300])


def grid_rows():
    st, body = http("GET", "/odn/grid-investment")
    if st == 200 and code_of(body) == 0:
        return (data_of(body) or {}).get("items") or []
    raise RuntimeError("grid-investment: " + str((st, body))[:200])


def grid_row_for(rows, grid_code):
    for r in rows:
        if r.get("prvCode") == "PHL001" and r.get("cityPrefix") == "MNL" and r.get("gridCode") == grid_code:
            return r
    return None


def main():
    global KEY
    try:
        import os
        ta_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", ".agents", "skills", "bossctl-cli", "test-accounts.json")
        ta = json.load(open(ta_path))
        KEY = ta["admin"]["apiKeys"][0]["key"]
    except Exception as e:
        print("FAIL: load api key: " + str(e))
        sys.exit(1)
    base_quad = quad_orphans()
    print("[baseline] quad LINKED-but-not-DEPLOYED = " + str(base_quad))
    # ---- 回补前样本(存量开户导入批空快照基线)----
    before_cnt = psql("SELECT count(*) FROM assets a JOIN asset_batches b ON b.id=a.batch_id WHERE b.name=" + q("存量开户导入") + " AND (a.region_id IS NULL OR a.region_id=0)").strip()
    before_sql = psql("SELECT a.asset_code||'|'||COALESCE(a.region_id::text,'0')||'|'||COALESCE(a.region_name,'') FROM assets a JOIN asset_batches b ON b.id=a.batch_id WHERE b.name=" + q("存量开户导入") + " AND (a.region_id IS NULL OR a.region_id=0) ORDER BY a.id LIMIT 3")
    print("[before] 空区域快照资产数=" + before_cnt)
    print("[before] 样本3条 (code|regionId|regionName):")
    print(before_sql)
    # ---- 就绪探针:新二进制特有行为(首次调用即执行回补)----
    probe = wait_ready()
    if probe is None:
        print("FAIL: ready probe /assets/region-backfill not ready (旧二进制或鉴权失败)")
        sys.exit(2)
    ok("ready probe: new binary deployed (POST /assets/region-backfill 200 code=0)")
    grid_code = None
    occ = odb = obd = sdb = sbd = ""
    anchor = ""
    address_id = None
    try:
        # ================= Phase A: 资源链导入网格归属联动(F2) =================
        grid_code = pick_free_grid()
        sfx = str(random.randint(300, 599))
        occ, odb, obd, sdb, sbd = "OCC" + sfx, "ODB" + sfx, "OBD" + sfx, "SDB" + sfx, "SBD" + sfx
        row = chain_row("PHL001", "MNL", grid_code, occ, odb, obd, sdb, sbd)
        res = import_chains([row])
        if res.get("imported") == 1 and res.get("failed") == 0:
            ok("链行导入: 带归属真实链 imported=1 (grid=" + str(grid_code) + ")")
        else:
            bad("链行导入", json.dumps(res, ensure_ascii=False)[:300])
        snap = psql("SELECT prv_code||'/'||city_prefix||'/'||grid_code FROM odn_resource_chain WHERE occ_code=" + q(occ) + " LIMIT 1").strip()
        if snap == "PHL001/MNL/" + str(grid_code):
            ok("链行归属快照落库: " + snap)
        else:
            bad("链行归属快照", snap)
        gcnt = psql("SELECT count(*) FROM odn_grid WHERE prv_code=" + q("PHL001") + " AND city_prefix=" + q("MNL") + " AND grid_code=" + str(grid_code)).strip()
        if gcnt == "1":
            ok("导入备案网格 odn_grid: " + str(grid_code))
        else:
            bad("网格备案", "grid count=" + gcnt)
        anchor = psql("SELECT code FROM odn_facility WHERE prv_code=" + q("PHL001") + " AND city_prefix=" + q("MNL") + " AND grid_code=" + str(grid_code) + " AND kind=" + q("MH") + " LIMIT 1").strip()
        if anchor and anchor.startswith("MH"):
            ok("锚点设施(编码含网格段): " + anchor)
        else:
            bad("锚点设施", anchor)
        acnt = psql("SELECT count(*) FROM odn_facility WHERE prv_code=" + q("PHL001") + " AND city_prefix=" + q("MNL") + " AND grid_code=" + str(grid_code) + " AND lifecycle_status=" + q("PLANNED")).strip()
        if acnt == "1":
            ok("锚点设施 lifecycle=PLANNED (规划链落规划态)")
        else:
            bad("锚点设施生命周期", "count=" + acnt)
        lst = http("GET", "/odn/resource-chains?limit=50")[1]
        mine = [x for x in ((data_of(lst) or {}).get("items") or []) if x.get("occCode") == occ]
        if mine and mine[0].get("prvCode") == "PHL001" and mine[0].get("gridCode") == grid_code:
            ok("链行清单 API 回带归属字段 (prvCode/gridCode)")
        else:
            bad("链行清单归属字段", json.dumps(mine[:1], ensure_ascii=False)[:200])
        # 投资测算网格行非空 + SQL 直查一致
        grows = grid_rows()
        gr = grid_row_for(grows, grid_code)
        sql_planned = psql("SELECT count(*) FROM odn_facility WHERE prv_code=" + q("PHL001") + " AND city_prefix=" + q("MNL") + " AND grid_code=" + str(grid_code) + " AND lifecycle_status=" + q("PLANNED")).strip()
        if gr is not None and int(gr.get("facilitiesPlanned") or 0) == int(sql_planned) and int(sql_planned) >= 1:
            ok("投资测算网格行非空: facilitiesPlanned=" + str(gr.get("facilitiesPlanned")) + " == SQL " + sql_planned)
        else:
            bad("投资测算网格行", "api=" + json.dumps(gr, ensure_ascii=False)[:200] + " sql=" + sql_planned)
        # 覆盖登记联动:挂锚点设施码进覆盖页 + 网格行覆盖计数
        address_id = create_address(TAG + "-ADDR")
        st, body = http("POST", "/odn/coverage", {"addressId": address_id, "facilityCode": anchor, "status": "SERVED", "note": TAG})
        if st == 200 and code_of(body) == 0:
            ok("覆盖登记挂锚点设施码 SERVED")
        else:
            bad("覆盖登记", str((st, body))[:200])
        sql_cov = psql("SELECT count(*) FROM address_coverage cv JOIN odn_facility fl ON fl.code=cv.facility_code WHERE fl.prv_code=" + q("PHL001") + " AND fl.city_prefix=" + q("MNL") + " AND fl.grid_code=" + str(grid_code) + " AND cv.status=" + q("SERVED")).strip()
        grows2 = grid_rows()
        gr2 = grid_row_for(grows2, grid_code)
        if gr2 is not None and int(gr2.get("coverageServed") or 0) == int(sql_cov) and int(sql_cov) == 1:
            ok("网格行覆盖计数: coverageServed=1 == SQL " + sql_cov + " (覆盖模型可见)")
        else:
            bad("网格行覆盖计数", "api=" + str(gr2 and gr2.get("coverageServed")) + " sql=" + sql_cov)
        covlist = http("GET", "/odn/coverage/list?limit=50")[1]
        covrows = [x for x in ((data_of(covlist) or {}).get("items") or []) if x.get("facilityCode") == anchor]
        if covrows and covrows[0].get("status") == "SERVED":
            ok("覆盖页可见: /odn/coverage/list 含锚点设施行 SERVED")
        else:
            bad("覆盖页可见", json.dumps(covrows[:1], ensure_ascii=False)[:200])
        # 巡检:链完整,孤儿/断祖/半链归零
        p = http("GET", "/odn/resource-chains/patrol")[1]
        pd = data_of(p) or {}
        if pd.get("orphanBoxCode") == 0 and pd.get("brokenAncestor") == 0 and pd.get("splitPortGap") == 0:
            ok("巡检: 孤儿 0 / 断祖 0 / 半链 0 (链完整)")
        else:
            bad("巡检归零", json.dumps(pd, ensure_ascii=False)[:300])
        # 幂等:同链同归属重导 -> duplicate(归属并入指纹)
        res2 = import_chains([chain_row("PHL001", "MNL", grid_code, occ, odb, obd, sdb, sbd)])
        if res2.get("duplicate") == 1 and res2.get("imported") == 0:
            ok("同链同归属重导幂等: duplicate=1 (归属并入指纹)")
        else:
            bad("重导幂等", json.dumps(res2, ensure_ascii=False)[:200])
        # ================= Phase B: 资产区域快照回补(W8 遗留①) =================
        d = data_of(probe) or {}
        back = (d.get("result") or d) if isinstance(d, dict) else {}
        got = int(back.get("backfilled") or 0)
        if got == int(before_cnt or 0) and got >= 300:
            ok("区域快照回补: backfilled=" + str(got) + " == 空快照基线 " + before_cnt)
        else:
            bad("区域快照回补数", "backfilled=" + str(got) + " baseline=" + before_cnt)
        after_cnt = psql("SELECT count(*) FROM assets a JOIN asset_batches b ON b.id=a.batch_id WHERE b.name=" + q("存量开户导入") + " AND (a.region_id IS NULL OR a.region_id=0)").strip()
        after_all = psql("SELECT count(*) FROM assets a JOIN asset_batches b ON b.id=a.batch_id WHERE b.name=" + q("存量开户导入") + " AND a.region_id>0 AND a.region_name=" + q("集团")).strip()
        if after_cnt == "0" and after_all == before_cnt:
            ok("回补后: 空快照 0,全部 " + after_all + " 台 = root 集团")
        else:
            bad("回补后状态", "empty=" + after_cnt + " 集团=" + after_all)
        after_sql = psql("SELECT a.asset_code||'|'||a.region_id||'|'||a.region_name FROM assets a JOIN asset_batches b ON b.id=a.batch_id WHERE b.name=" + q("存量开户导入") + " AND a.region_id>0 ORDER BY a.id LIMIT 3")
        print("[after] 样本3条 (code|regionId|regionName):")
        print(after_sql)
        probe2 = http("POST", "/assets/region-backfill", {})[1]
        d2 = data_of(probe2) or {}
        back2 = (d2.get("result") or d2) if isinstance(d2, dict) else {}
        if int(back2.get("backfilled") or 0) == 0:
            ok("回补幂等: 二次执行 backfilled=0 (重跑零更新)")
        else:
            bad("回补幂等", json.dumps(back2, ensure_ascii=False)[:200])
        acnt2 = psql("SELECT count(*) FROM audit_logs WHERE action=" + q("asset.region-backfill")).strip()
        if int(acnt2 or 0) >= 2:
            ok("审计 asset.region-backfill >= 2 次留痕")
        else:
            bad("审计留痕", acnt2)
    finally:
        cleanup_all(grid_code, occ, odb, obd, sdb, sbd, anchor, address_id)
        after_quad = quad_orphans()
        if after_quad == base_quad:
            ok("quad orphans unchanged after acceptance (" + str(after_quad) + ")")
        else:
            bad("quad orphan delta", str(base_quad) + " -> " + str(after_quad))
    print("")
    print("=== FINAL SUMMARY ===")
    print("PASS=" + str(PASS) + " FAIL=" + str(FAIL))
    if FAIL == 0:
        print("W3C ACCEPTANCE PASS")
        sys.exit(0)
    print("W3C ACCEPTANCE FAIL")
    sys.exit(1)


def cleanup_all(grid_code, occ, odb, obd, sdb, sbd, anchor, address_id):
    # 造数即清:子->父逐层删,幂等可重入;失败打印 [cleanup] FAILED 不吞。
    codes = [c for c in (occ, odb, obd, sdb, sbd) if c]
    try:
        if address_id:
            psql("DELETE FROM address_coverage WHERE address_id = " + str(address_id))
            psql("DELETE FROM addresses WHERE id = " + str(address_id))
        if anchor:
            psql("DELETE FROM address_coverage WHERE facility_code = " + q(anchor))
            psql("DELETE FROM odn_facility WHERE code = " + q(anchor))
        if grid_code:
            psql("DELETE FROM odn_grid WHERE prv_code=" + q("PHL001") + " AND city_prefix=" + q("MNL") + " AND grid_code=" + str(grid_code) + " AND name LIKE " + q(TAG + "%"))
        if codes:
            allc = ",".join(q(c) for c in codes)
            psql("DELETE FROM odn_resource_chain WHERE occ_code IN (" + allc + ")")
            psql("DELETE FROM odn_device WHERE prv_code IS NULL AND code IN (" + allc + ")")
        left = psql("SELECT count(*) FROM odn_resource_chain WHERE remark = " + q(TAG)).strip()
        if left != "0":
            print("[cleanup] WARN chain rows left: " + left)
        print("[cleanup] W3C acceptance data removed (tag=" + TAG + ")")
    except Exception as e:
        print("[cleanup] FAILED: " + str(e))


if __name__ == "__main__":
    main()
