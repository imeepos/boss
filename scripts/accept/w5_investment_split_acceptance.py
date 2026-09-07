#!/usr/bin/env python3
# W5 投资分析深化与分光比建模 102 真实环境验收(P-INFRA-1 W5;审查 F1 四点)。
# 前置:102 已部署含 W5 的 main(CI deploy-102);本机 ssh 免密 imeepos@192.168.0.102。
# 断言:就绪探针(新端点 /odn/city-investment 旧代次 404 新代次 200+code0;401 分不清代次)/
#       规划成本(项目预算按明细金额占比分摊 0.25/0.75)/材料成本(CONFIRMED 出库采购价链合计同比例分摊,与结算成本分列)/
#       城市卷积(城市行=网格行同源卷积,逐格一致)/SQL 直查对照(接口值与权威表直算逐格一致)/
#       分光比回写建模(backfill-split 幂等,城市域同名唯一优先,导入域兜底,全网 summary 潜在/已接/可扩)/
#       容量不进网格行不摊/审计留痕;造数即清(finally 兜底)+quad 孤儿基线对比不新增。
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
TAG = "W5ACC" + str(random.randint(10000, 99999))


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
    r = subprocess.run(["ssh", "-o", "BatchMode=yes", SSH_HOST, cmd], capture_output=True, text=True, timeout=timeout, input=stdin_text)
    return r.returncode, r.stdout.strip(), r.stderr.strip()


def psql(sql):
    cmd = "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc " + shlex.quote(sql)
    rc, out, err = ssh(cmd)
    if rc != 0:
        raise RuntimeError("psql failed: " + out + err)
    return out


def q(v):
    # SQL 字面量单引号(脚本内禁反斜杠转义)
    return chr(39) + str(v) + chr(39)


def data_of(body):
    return (body or {}).get("data")


def code_of(body):
    return (body or {}).get("code")


def near(a, b, tol=0.02):
    return a is not None and b is not None and abs(a - b) <= tol


def wait_ready(seconds=300):
    # 二进制就绪探针:W5 新端点在新二进制 200+code0,旧二进制 404(路由未注册)。
    last = ""
    deadline = time.time() + seconds
    while time.time() < deadline:
        st, body = http("GET", "/odn/city-investment")
        last = "status=" + str(st) + " code=" + str(code_of(body))
        if st == 200 and code_of(body) == 0:
            return True
        time.sleep(5)
    print("ready probe timeout; last=" + last)
    return False


def quad_orphans():
    st, body = http("GET", "/db-patrol/orphans")
    if st == 200 and code_of(body) == 0:
        for it in data_of(body).get("items", []):
            if "LINKED but asset not DEPLOYED" in str(it.get("check")):
                return int(it.get("orphans", 0))
    return -1


def create_grid(grid_code):
    # prvCode/cityPrefix 是 query 参数(yaml /odn/grids post),body 只带 gridCode/name/coverage/status
    st, body = http("POST", "/odn/grids?prvCode=PHL001&cityPrefix=MNL",
        {"gridCode": grid_code, "name": TAG + "-G" + str(grid_code), "coverage": TAG, "status": "ACTIVE"})
    if st == 200 and code_of(body) == 0:
        return grid_code
    bad("create grid " + str(grid_code), str((st, body))[:200])
    return None


def pick_free_grids(need):
    # 102 网格码可能被占用(如 91),先查现役网格再从 60~99 选空位
    st, body = http("GET", "/odn/grids?prvCode=PHL001&cityPrefix=MNL")
    taken = set()
    if st == 200 and code_of(body) == 0:
        for r in data_of(body) or []:
            taken.add(int(r.get("gridCode")))
    free = [c for c in range(60, 100) if c not in taken]
    if len(free) < need:
        raise RuntimeError("no free grid codes, taken=" + str(sorted(taken)))
    return free[:need]


def create_facility(kind, grid_code, seq):
    code = kind + ("%02d" % grid_code) + ("%03d" % seq)
    st, body = http("POST", "/odn/facilities", {"code": code, "kind": kind, "name": TAG + "-" + code,
        "prvCode": "PHL001", "cityPrefix": "MNL", "gridCode": grid_code})
    if st == 200 and code_of(body) == 0:
        psql("UPDATE odn_facility SET lifecycle_status = " + q("PLANNED") + " WHERE code = " + q(code))
        return code
    raise RuntimeError("create facility " + code + ": " + str((st, body))[:200])


def create_address(label):
    # label 契约限制:路径段=小写字母数字(name 无限制,仍用 TAG 前缀供清理定位)
    st, body = http("POST", "/addresses", {"parentID": 0, "label": label.lower().replace("-", ""), "name": label})
    if st == 200 and code_of(body) == 0:
        aid = (data_of(body) or {}).get("id") or (data_of(body) or {}).get("addressId")
        if aid:
            return int(aid)
        row = psql("SELECT id FROM addresses WHERE label = " + shlex.quote(label) + " ORDER BY id DESC LIMIT 1")
        return int(row)
    raise RuntimeError("create address " + label + ": " + str((st, body))[:200])


def set_coverage(address_id, fac_code):
    st, body = http("POST", "/odn/coverage", {"addressId": address_id, "facilityCode": fac_code, "status": "SERVED"})
    if st == 200 and code_of(body) == 0:
        return True
    raise RuntimeError("coverage: " + str((st, body))[:200])


def create_city_device(code, kind, parent_code):
    payload = {"code": code, "kind": kind, "prvCode": "PHL001", "cityPrefix": "MNL", "name": TAG + "-" + code}
    if parent_code:
        payload["parentCode"] = parent_code
    st, body = http("POST", "/odn/devices", payload)
    if st == 200 and code_of(body) == 0:
        return True
    raise RuntimeError("create device " + code + ": " + str((st, body))[:200])


def chain_row(occ, odb, obd, s1r, s1p, sdb, sbd, s2r, s2p):
    return {"rowNo": 0, "resourceStatus": "", "siteCode": TAG + "-SITE", "siteName": TAG,
        "oltCode": TAG + "-OLT", "odfCode": "", "odfPort": "",
        "occCode": occ, "odbCode": odb, "obdCode": obd,
        "split1Ratio": s1r, "split1Port": s1p,
        "sdbCode": sdb, "sbdCode": sbd,
        "split2Ratio": s2r, "split2Port": s2p,
        "totalSplit": "", "fiberCode": TAG + "-F", "frTo": TAG + "-FRTO",
        "portStatus": "可用", "layingMethod": "架空", "rowStatus": "不适用", "peceStatus": "不适用",
        "remark": TAG}

def import_chains(rows):
    st, body = http("POST", "/odn/resource-chains/import", {"exampleRowCount": 0, "rows": rows})
    if st == 200 and code_of(body) == 0:
        res = (data_of(body) or {}).get("result") or {}
        if res.get("failed"):
            raise RuntimeError("import rows failed: " + str([r for r in res.get("rows", []) if r.get("status") == "failed"])[:400])
        return res
    raise RuntimeError("import: " + str((st, body))[:300])


def backfill():
    st, body = http("POST", "/odn/resource-chains/backfill-split", {})
    if st == 200 and code_of(body) == 0:
        return (data_of(body) or {}).get("result") or {}
    raise RuntimeError("backfill: " + str((st, body))[:300])


def grid_rows():
    st, body = http("GET", "/odn/grid-investment")
    if st == 200 and code_of(body) == 0:
        return (data_of(body) or {}).get("items") or []
    raise RuntimeError("grid-investment: " + str((st, body))[:200])


def city_rows():
    st, body = http("GET", "/odn/city-investment")
    if st == 200 and code_of(body) == 0:
        return (data_of(body) or {}).get("items") or []
    raise RuntimeError("city-investment: " + str((st, body))[:200])


def capacity_report():
    st, body = http("GET", "/odn/split-capacity")
    if st == 200 and code_of(body) == 0:
        return data_of(body) or {}
    raise RuntimeError("split-capacity: " + str((st, body))[:200])


def grid_row_for(rows, grid_code):
    for r in rows:
        if r.get("prvCode") == "PHL001" and r.get("cityPrefix") == "MNL" and r.get("gridCode") == grid_code:
            return r
    return None


def city_row_for(rows, city):
    for r in rows:
        if r.get("prvCode") == "PHL001" and r.get("cityPrefix") == city:
            return r
    return None


def cap_item(items, code, level):
    for r in items:
        if r.get("code") == code and r.get("splitLevel") == level:
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
    if not wait_ready():
        print("FAIL: ready probe /odn/city-investment not ready (旧二进制或鉴权失败)")
        sys.exit(2)
    ok("ready probe: new binary deployed (/odn/city-investment 200 code=0)")
    base_quad = quad_orphans()
    print("[baseline] quad LINKED-but-not-DEPLOYED = " + str(base_quad))
    sfx = str(random.randint(700, 799))
    sfx2 = str(random.randint(800, 899))
    occ_s, odb_s, obd_s, sdb_s, sbd_s = "OCC" + sfx, "ODB" + sfx, "OBD" + sfx, "SDB" + sfx, "SBD" + sfx
    obd_i, odb_i, occ_i = "OBD" + sfx2, "ODB" + sfx2, "OCC" + sfx2
    batches = []
    proj = None
    supplier_id = None
    batch_id = None
    city_codes = [occ_s, odb_s, obd_s, sdb_s, sbd_s]
    import_codes = [occ_i, odb_i, obd_i]
    try:
        # ---- Phase A: 造数全链:网格→设施→覆盖→项目挂预算→材料出库→资源链分光比 ----
        free_grids = pick_free_grids(2)
        grid_a, grid_b = free_grids[0], free_grids[1]
        if not create_grid(grid_a) or not create_grid(grid_b):
            raise RuntimeError("grid create failed")
        ok("grids created " + str(grid_a) + "/" + str(grid_b))
        fac_a = create_facility("P", grid_a, 1)
        fac_b = create_facility("P", grid_b, 1)
        addr_a = create_address(TAG + "-A")
        addr_b = create_address(TAG + "-B")
        set_coverage(addr_a, fac_a)
        set_coverage(addr_b, fac_b)
        ok("facilities PLANNED + coverage SERVED on both grids")
        for c, k, pc in ((occ_s, "OCC", ""), (odb_s, "ODB", occ_s), (obd_s, "OBD", odb_s), (sdb_s, "SDB", odb_s), (sbd_s, "SBD", sdb_s)):
            create_city_device(c, k, pc)
        ok("city-domain splitter boxes created (" + obd_s + "/1:8, " + sbd_s + "/1:8)")
        st, body = http("POST", "/procurement/suppliers", {"code": TAG + "-S", "name": TAG + " 材料供应商",
            "legalEntityId": 6, "contractorType": "MATERIAL", "status": "ENABLED", "remark": TAG})
        if st == 200 and code_of(body) == 0:
            supplier_id = (data_of(body) or {}).get("id")
        else:
            raise RuntimeError("supplier: " + str((st, body))[:200])
        st, body = http("POST", "/procurement/orders", {"supplierId": supplier_id, "legalEntityId": 6,
            "items": [{"materialCode": TAG + "-MAT", "quantity": 2, "unitAmount": 10}]})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("order: " + str((st, body))[:200])
        order_id = (data_of(body) or {}).get("id")
        st, body = http("POST", "/procurement/receipts", {"orderId": order_id, "legalEntityId": 6})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("receipt: " + str((st, body))[:200])
        receipt_id = (data_of(body) or {}).get("id")
        st, body = http("POST", "/procurement/receipts/" + str(receipt_id) + "/confirm",
            {"items": [{"materialCode": TAG + "-MAT", "quantity": 2}]})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("receipt confirm: " + str((st, body))[:200])
        batch_id = psql("SELECT batch_id FROM procurement_receipts WHERE id = " + str(receipt_id)).strip()
        ok("procurement chain: order(2x10) -> receipt confirmed -> 2 assets IN_STOCK (batch " + batch_id + ")")
        st, body = http("POST", "/odn/constructions", {"projNo": TAG + "-P", "name": TAG + " 投资口径验收"})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("project: " + str((st, body))[:200])
        proj = int(psql("SELECT id FROM construction_projects WHERE proj_no = " + q(TAG + "-P") + " LIMIT 1"))
        st, body = http("PUT", "/odn/constructions/" + str(proj) + "/budget", {"budgetAmount": 1000.0})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("budget: " + str((st, body))[:200])
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/items", {"facilityCode": fac_a, "quantity": 1, "unitPrice": 100})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("item A: " + str((st, body))[:200])
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/items", {"facilityCode": fac_b, "quantity": 3, "unitPrice": 100})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("item B: " + str((st, body))[:200])
        ok("project " + str(proj) + " budget=1000, items 100(grid " + str(grid_a) + ")/300(grid " + str(grid_b) + ") shares 0.25/0.75")
        asset_rows = psql("SELECT id FROM assets WHERE type = " + q(TAG + "-MAT") + " ORDER BY id")
        asset_ids = [int(x) for x in asset_rows.split() if x]
        if len(asset_ids) != 2:
            raise RuntimeError("expect 2 receipt assets, got " + str(asset_ids))
        st, body = http("POST", "/odn/material-issues", {"projectId": proj, "assetIds": asset_ids, "remark": TAG})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("issue: " + str((st, body))[:200])
        issue_id = ((data_of(body) or {}).get("issue") or {}).get("id") or (data_of(body) or {}).get("id")
        st, body = http("POST", "/odn/material-issues/" + str(issue_id) + "/confirm", {})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("issue confirm: " + str((st, body))[:200])
        ok("material issue CONFIRMED (2 assets x 10 = 20 material cost)")
        batch_a = import_chains([
            chain_row(occ_s, odb_s, obd_s, "1:8", "P1", sdb_s, sbd_s, "1:8", "P1"),
            chain_row(occ_s, odb_s, obd_s, "1:8", "P1", sdb_s, sbd_s, "1:8", "P2"),
            chain_row(occ_s, odb_s, obd_s, "1:8", "P1", sdb_s, sbd_s, "1:8", "P3"),
        ])
        batches.append(batch_a.get("batchNo"))
        batch_b = import_chains([
            chain_row(occ_i, odb_i, obd_i, "1:4", "P1", "", "", "", ""),
            chain_row(occ_i, odb_i, obd_i, "1:4", "P2", "", "", "", ""),
        ])
        batches.append(batch_b.get("batchNo"))
        ok("chains imported: batch A 3 rows(city boxes), batch B 2 rows(import-domain)")
        res1 = backfill()
        res2 = backfill()
        if res1.get("devicesModeled") == 3 and res2.get("devicesModeled") == 3 and res2.get("unresolvedCodes") == 0:
            ok("backfill-split idempotent: devicesModeled=3 (level1=" + str(res2.get("level1")) + " level2=" + str(res2.get("level2")) + ")")
        else:
            bad("backfill-split result", str((res1, res2))[:300])
        cnt = psql("SELECT count(*) FROM odn_device_split_capacity")
        if cnt.strip() == "3":
            ok("capacity table has exactly 3 modeled devices after re-backfill")
        else:
            bad("capacity table count", cnt)
        # ---- Phase B: 断言:网格行新口径与 SQL 直查逐格一致 ----
        grows = grid_rows()
        ga = grid_row_for(grows, grid_a)
        gb = grid_row_for(grows, grid_b)
        # SQL 直查对照(与域层同形态 CTE):规划成本=预算x明细金额占比;材料成本=出库采购价合计x同占比
        pid = str(proj)
        sql_planned = psql("WITH item_grid AS (SELECT ci.project_id, fl.grid_code, SUM(ci.amount) AS part "
            "FROM construction_items ci JOIN odn_facility fl ON fl.code = ci.facility_code AND fl.grid_code IS NOT NULL "
            "WHERE ci.project_id = " + pid + " GROUP BY ci.project_id, fl.grid_code), "
            "proj_total AS (SELECT SUM(part) AS total FROM item_grid) "
            "SELECT i.grid_code, SUM(cp.budget_amount * i.part / p.total)::numeric(14,2) "
            "FROM item_grid i JOIN proj_total p ON true JOIN construction_projects cp ON cp.id = i.project_id "
            "WHERE cp.budget_amount IS NOT NULL AND p.total > 0 GROUP BY i.grid_code ORDER BY i.grid_code")
        sql_mat = psql("WITH asset_price AS (SELECT DISTINCT ON (a.id) a.id, oi.unit_amount FROM assets a "
            "JOIN procurement_receipts r ON r.batch_id = a.batch_id "
            "JOIN procurement_order_items oi ON oi.order_id = r.order_id AND oi.material_code = a.type), "
            "issue_cost AS (SELECT mi.project_id, SUM(p.unit_amount) AS cost FROM odn_material_issues mi "
            "JOIN odn_material_issue_items it ON it.issue_id = mi.id JOIN asset_price p ON p.id = it.asset_id "
            "WHERE mi.project_id = " + pid + " AND mi.status = " + q("CONFIRMED") + " GROUP BY mi.project_id), "
            "item_grid AS (SELECT ci.project_id, fl.grid_code, SUM(ci.amount) AS part FROM construction_items ci "
            "JOIN odn_facility fl ON fl.code = ci.facility_code AND fl.grid_code IS NOT NULL "
            "WHERE ci.project_id = " + pid + " GROUP BY ci.project_id, fl.grid_code), "
            "proj_total AS (SELECT SUM(part) AS total FROM item_grid) "
            "SELECT i.grid_code, SUM(ic.cost * i.part / t.total)::numeric(14,2) "
            "FROM item_grid i JOIN proj_total t ON true JOIN issue_cost ic ON ic.project_id = i.project_id "
            "GROUP BY i.grid_code ORDER BY i.grid_code")
        planned_map = {}
        for line in sql_planned.splitlines():
            parts_ = line.split("|")
            if len(parts_) == 2:
                planned_map[int(parts_[0])] = float(parts_[1])
        mat_map = {}
        for line in sql_mat.splitlines():
            parts_ = line.split("|")
            if len(parts_) == 2:
                mat_map[int(parts_[0])] = float(parts_[1])
        both_ok = True
        for gcode, row in ((grid_a, ga), (grid_b, gb)):
            if not row:
                bad("grid row missing " + str(gcode))
                both_ok = False
                continue
            exp_planned = planned_map.get(gcode)
            exp_mat = mat_map.get(gcode)
            if near(row.get("plannedCost"), exp_planned):
                ok("grid " + str(gcode) + " plannedCost=" + str(row.get("plannedCost")) + " == SQL " + str(exp_planned))
            else:
                bad("grid " + str(gcode) + " plannedCost", "api=" + str(row.get("plannedCost")) + " sql=" + str(exp_planned))
                both_ok = False
            if near(row.get("materialCost"), exp_mat):
                ok("grid " + str(gcode) + " materialCost=" + str(row.get("materialCost")) + " == SQL " + str(exp_mat))
            else:
                bad("grid " + str(gcode) + " materialCost", "api=" + str(row.get("materialCost")) + " sql=" + str(exp_mat))
                both_ok = False
            if row.get("settledCost") is None and row.get("costPerServed") is None:
                ok("grid " + str(gcode) + " settledCost/costPerServed null (无结算,未登记不显 0)")
            else:
                bad("grid " + str(gcode) + " settledCost should be null", str(row.get("settledCost")))
                both_ok = False
            if row.get("coverageServed") == 1:
                ok("grid " + str(gcode) + " coverageServed=1 (W2 口径回归)")
            else:
                bad("grid " + str(gcode) + " coverageServed", str(row.get("coverageServed")))
                both_ok = False
        if ga is not None and ga.get("potentialHomes") is None:
            ok("grid row has no capacity columns (容量住设备维度,不摊网格)")
        else:
            bad("grid capacity must be absent", str(ga and ga.get("potentialHomes")))
        # ---- Phase B2: 城市卷积行 ----
        crows = city_rows()
        mnl = city_row_for(crows, "MNL")
        if mnl:
            if near(mnl.get("plannedCost"), 1000.0) and near(mnl.get("materialCost"), 20.0):
                ok("city MNL plannedCost=1000 materialCost=20 (网格行卷积合计)")
            else:
                bad("city MNL costs", str(mnl.get("plannedCost")) + "/" + str(mnl.get("materialCost")))
            if mnl.get("coverageServed") == 2:
                ok("city MNL coverageServed=2 (两网格卷积)")
            else:
                bad("city MNL coverageServed", str(mnl.get("coverageServed")))
            if mnl.get("potentialHomes") == 8 and mnl.get("connectedHomes") == 3 and mnl.get("expandableHomes") == 5:
                ok("city MNL homes 8/3/5 (城市域 SBD 容量,一级器下挂二级不计户)")
            else:
                bad("city MNL homes", str((mnl.get("potentialHomes"), mnl.get("connectedHomes"), mnl.get("expandableHomes"))))
            if near(mnl.get("costPerPotential"), 1020.0 / 8):
                ok("city MNL costPerPotential=127.5 ((规划1000+材料20)/潜在8;分母不是覆盖户数)")
            else:
                bad("city MNL costPerPotential", str(mnl.get("costPerPotential")))
        else:
            bad("city row MNL missing")
        # ---- Phase B3: 设备容量视图与全网 summary ----
        rep = capacity_report()
        items = rep.get("items") or []
        obd_city = cap_item(items, obd_s, 1)
        sbd_city = cap_item(items, sbd_s, 2)
        obd_imp = cap_item(items, obd_i, 1)
        if obd_city and obd_city.get("hasSecondary") and obd_city.get("ratio") == 8:
            ok("capacity " + obd_s + " level1 ratio=8 hasSecondary=true (下挂二级,户级口径跳过)")
        else:
            bad("capacity " + obd_s, str(obd_city)[:200])
        if sbd_city and sbd_city.get("usedPorts") == 3 and sbd_city.get("expandable") == 5 and sbd_city.get("prvCode") == "PHL001":
            ok("capacity " + sbd_s + " level2 8 ports, used=3, expandable=5, scope=city (还能接 5 户)")
        else:
            bad("capacity " + sbd_s, str(sbd_city)[:200])
        if obd_imp and obd_imp.get("prvCode") is None and obd_imp.get("ratio") == 4 and obd_imp.get("usedPorts") == 2:
            ok("capacity " + obd_i + " import-domain (prv null) ratio=4 used=2 (城市行不含,全网含)")
        else:
            bad("capacity " + obd_i, str(obd_imp)[:200])
        sm = rep.get("summary") or {}
        if sm.get("devices") == 3 and sm.get("potentialHomes") == 12 and sm.get("connectedHomes") == 5 and sm.get("expandableHomes") == 7:
            ok("network summary devices=3 potential=12 connected=5 expandable=7 (含导入域;total_split 不重复计数)")
        else:
            bad("network summary", str(sm))
        # ---- Phase B4: 审计 ----
        cnt = psql("SELECT count(*) FROM audit_logs WHERE action = " + q("odn.resource-chain.backfill-split"))
        if int(cnt.strip() or "0") >= 2:
            ok("audit odn.resource-chain.backfill-split >= 2 (两次重建均留痕)")
        else:
            bad("audit backfill-split", cnt)
    finally:
        cleanup_all(TAG, proj, supplier_id, grid_a, grid_b, batches, city_codes, import_codes, batch_id)
        after_quad = quad_orphans()
        if after_quad == base_quad:
            ok("quad orphans unchanged after acceptance (" + str(after_quad) + ")")
        else:
            bad("quad orphan delta", str(base_quad) + " -> " + str(after_quad))
    print("")
    print("=== FINAL SUMMARY ===")
    print("PASS=" + str(PASS) + " FAIL=" + str(FAIL))
    if FAIL == 0:
        print("W5 ACCEPTANCE PASS")
        sys.exit(0)
    print("W5 ACCEPTANCE FAIL")
    sys.exit(1)


def cleanup_all(tag, proj, supplier_id, grid_a, grid_b, batches, city_codes, import_codes, batch_id):
    # 造数即清:子→父逐层删(显式 ID/编码清单,幂等可重入);失败打印 [cleanup] FAILED 不吞
    proj_s = str(proj or -1)
    sup_s = str(supplier_id or -1)
    codes_all = ",".join(q(c) for c in (city_codes + import_codes))
    try:
        psql("DELETE FROM odn_device_split_capacity WHERE device_id IN (SELECT id FROM odn_device WHERE code IN (" + codes_all + "))")
        psql("DELETE FROM odn_resource_chain WHERE remark = " + q(tag))
        psql("DELETE FROM odn_device WHERE code IN (" + codes_all + ")")
        psql("DELETE FROM odn_material_issue_items WHERE issue_id IN (SELECT id FROM odn_material_issues WHERE project_id = " + proj_s + ")")
        psql("DELETE FROM odn_material_issues WHERE project_id = " + proj_s)
        psql("DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM assets WHERE type = " + q(tag + "-MAT") + ")")
        psql("DELETE FROM assets WHERE type = " + q(tag + "-MAT"))
        if batch_id and batch_id not in ("None", ""):
            psql("UPDATE procurement_receipts SET batch_id = NULL WHERE batch_id = " + batch_id)
            psql("DELETE FROM asset_batches WHERE id = " + batch_id)
        psql("DELETE FROM procurement_receipts WHERE order_id IN (SELECT id FROM procurement_orders WHERE supplier_id = " + sup_s + ")")
        psql("DELETE FROM procurement_order_items WHERE order_id IN (SELECT id FROM procurement_orders WHERE supplier_id = " + sup_s + ")")
        psql("DELETE FROM procurement_orders WHERE supplier_id = " + sup_s)
        psql("DELETE FROM procurement_suppliers WHERE id = " + sup_s)
        psql("DELETE FROM construction_items WHERE project_id = " + proj_s)
        psql("DELETE FROM construction_milestones WHERE project_id = " + proj_s)
        psql("DELETE FROM construction_projects WHERE id = " + proj_s)
        psql("DELETE FROM address_coverage WHERE address_id IN (SELECT id FROM addresses WHERE name LIKE " + q(tag + "-%") + ")")
        psql("DELETE FROM addresses WHERE name LIKE " + q(tag + "-%"))
        psql("DELETE FROM odn_facility WHERE name LIKE " + q(tag + "-%"))
        psql("DELETE FROM odn_grid WHERE prv_code = " + q("PHL001") + " AND city_prefix = " + q("MNL") + " AND grid_code IN (" + str(grid_a) + "," + str(grid_b) + ")")
        left = psql("SELECT count(*) FROM odn_resource_chain WHERE remark = " + q(tag)).strip()
        if left != "0":
            print("[cleanup] WARN chain rows left: " + left)
        print("[cleanup] W5 acceptance data removed (tag=" + tag + ")")
    except Exception as e:
        print("[cleanup] FAILED: " + str(e))


if __name__ == "__main__":
    main()