#!/usr/bin/env python3
# W8 ODN 资产化转固 102 真实环境验收(P-INFRA-1 W8;审查 F7)。
# 前置:102 已部署含 W8 的 main(CI deploy-102);本机 ssh 免密 imeepos@192.168.0.102。
# 断言:就绪探针(新端点 /odn/assets/registrations 旧代次 404 新代次 200+code0)/出库台账连续性
#       (OPEN 不动状态→CONFIRMED 置 IN_TRANSIT→取消退库回 IN_STOCK)/活跃单重复占用 40900/
#       转固凭证(CONSTRUCTION 须项目 ACCEPTED→资产 DEPLOYED+设施列表 assetReg 可见)/
#       重复登记 40900/冲销回 IN_STOCK/盘点 scope=ODN 只盘有凭证资产/审计留痕;
#       造数即清(finally 兜底)+quad 孤儿基线对比(存量 313 开户导入债待负责人批处置,不新增即过)。
import json
import random
import shlex
import subprocess
import sys
import time
import urllib.error
import urllib.request

BASE = "http://192.168.0.102:28080/api/admin/v1"
HEALTH = "http://192.168.0.102:28080/healthz"
SSH_HOST = "imeepos@192.168.0.102"
PASS = 0
FAIL = 0
KEY = ""
TASK_ID = 0


def ok(name):
    global PASS
    PASS += 1
    print("PASS: " + name)


def bad(name, detail=""):
    global FAIL
    FAIL += 1
    print("FAIL: " + name + ("  [" + detail + "]" if detail else ""))


def http(method, path, body=None, timeout=20):
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


def wait_ready(seconds=300):
    # 二进制就绪探针:W8 新端点在新二进制 200+code0,旧二进制 404(路由未注册);
    # 401 只说明鉴权不通过分不清代次,故带 admin key 断言业务码。
    last = ""
    deadline = time.time() + seconds
    while time.time() < deadline:
        st, body = http("GET", "/odn/assets/registrations")
        last = "status=" + str(st) + " code=" + str((body or {}).get("code") if body else None)
        if st == 200 and body and body.get("code") == 0:
            return True
        time.sleep(5)
    print("ready probe timeout; last=" + last)
    return False


def quad_orphans():
    st, body = http("GET", "/db-patrol/orphans")
    if st == 200 and body and body.get("code") == 0:
        for it in body.get("data", {}).get("items", []):
            if "LINKED but asset not DEPLOYED" in str(it.get("check")):
                return int(it.get("orphans", 0))
    return -1


def asset_status(asset_id):
    st, body = http("GET", "/assets/" + str(asset_id))
    if st == 200 and body and body.get("code") == 0:
        return (body.get("data") or {}).get("status")
    return None


def batch_items(body):
    d = (body or {}).get("data")
    if isinstance(d, dict):
        return d.get("items", [])
    if isinstance(d, list):
        return d
    return []


def ensure_batch():
    name = "W8ACC 转固验收批次"
    st, body = http("GET", "/assets/batches?limit=500")
    for it in batch_items(body):
        if it.get("name") == name:
            return it.get("id")
    st, body = http("POST", "/asset-batches", {"name": name, "legalEntityId": 6})
    if st != 200 or not body or body.get("code") != 0:
        bad("create batch", str(body))
        return None
    st, body = http("GET", "/assets/batches?limit=500")
    for it in batch_items(body):
        if it.get("name") == name:
            return it.get("id")
    return None


def create_asset(batch_id, tag):
    sn = "W8ACCSN" + str(random.randint(1000000, 9999999)) + tag
    loid = "W8ACCLO" + str(random.randint(1000000, 9999999)) + tag
    st, body = http("POST", "/assets", {"batchId": batch_id, "type": "ONU", "sn": sn, "loid": loid})
    if st != 200 or not body or body.get("code") != 0:
        bad("create asset " + tag, str(body))
        return None
    st2, body2 = http("GET", "/assets?keyword=" + loid + "&limit=5")
    for it in ((body2 or {}).get("data") or {}).get("items", []):
        if it.get("loid") == loid:
            return it.get("assetId")
    bad("locate asset " + tag, str((st2, body2))[:200])
    return None


def create_facility(suffix):
    code = "CLS9" + str(random.randint(1000, 9999))
    st, body = http("POST", "/odn/facilities", {"code": code, "kind": "CLS", "name": "W8ACC-" + suffix, "prvCode": "PHL001", "cityPrefix": "MNL"})
    if st != 200 or not body or body.get("code") != 0:
        bad("create facility " + suffix, str(body))
        return None
    # 新建设施缺省 IN_SERVICE,入施工单须 PLANNED(1.5.8 前置);验收造数经 psql 直设,
    # 开工/竣工翻转仍走真实 API(W4 同款)。
    psql("UPDATE odn_facility SET lifecycle_status = " + q("PLANNED") + " WHERE code = " + q(code))
    return code


def create_project(suffix):
    proj_no = "w8acc-" + suffix + "-" + str(random.randint(1000, 9999))
    st, body = http("POST", "/odn/constructions", {"projNo": proj_no, "name": "W8 验收 " + suffix})
    if st != 200 or not body or body.get("code") != 0:
        bad("create project " + suffix, str(body))
        return None
    st2, body2 = http("GET", "/odn/constructions?limit=500")
    for p in (body2 or {}).get("data", []):
        if p.get("projNo") == proj_no:
            return p["id"]
    bad("locate project " + suffix, str((st2, body2))[:200])
    return None


def cleanup():
    steps = [
        "DELETE FROM odn_asset_registrations WHERE asset_id IN (SELECT id FROM assets WHERE sn LIKE " + q("W8ACCSN%") + ")",
        "DELETE FROM odn_material_issue_items WHERE issue_id IN (SELECT id FROM odn_material_issues WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w8acc-%") + "))",
        "DELETE FROM odn_material_issues WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w8acc-%") + ")",
        "DELETE FROM construction_items WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w8acc-%") + ")",
        "DELETE FROM construction_projects WHERE proj_no LIKE " + q("w8acc-%"),
        "DELETE FROM stocktake_items WHERE task_id = " + str(TASK_ID),
        "DELETE FROM stocktakes WHERE id = " + str(TASK_ID),
        "DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM assets WHERE sn LIKE " + q("W8ACCSN%") + ")",
        "DELETE FROM assets WHERE sn LIKE " + q("W8ACCSN%"),
        "DELETE FROM asset_batches WHERE name = " + q("W8ACC 转固验收批次"),
        "DELETE FROM odn_facility WHERE name LIKE " + q("W8ACC-%"),
    ]
    failed = 0
    for sql in steps:
        if TASK_ID == 0 and ("stocktake_items" in sql or "FROM stocktakes" in sql):
            continue
        try:
            psql(sql)
        except Exception as e:
            failed += 1
            print("[cleanup] FAILED: " + sql[:70] + " -> " + str(e))
    if failed == 0:
        print("[cleanup] data removed OK")
    return failed == 0


def main():
    global KEY, TASK_ID
    try:
        import os
        ta_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", ".agents", "skills", "bossctl-cli", "test-accounts.json")
        ta = json.load(open(ta_path))
        KEY = ta["admin"]["apiKeys"][0]["key"]
    except Exception as e:
        print("FAIL: load api key: " + str(e))
        sys.exit(1)
    ok("api key loaded")
    if not wait_ready():
        print("FAIL: ready probe /odn/assets/registrations not ready (旧二进制或鉴权失败)")
        sys.exit(2)
    ok("ready probe: new binary deployed (/odn/assets/registrations 200 code=0)")
    base_quad = quad_orphans()
    print("[baseline] quad LINKED-but-not-DEPLOYED = " + str(base_quad) + " (存量开户导入债,处置待负责人批)")
    try:
        # ---- Phase A: 造数:批次+2 资产+设施+项目(BUILDING) ----
        batch_id = ensure_batch()
        if not batch_id:
            raise RuntimeError("batch setup failed")
        a1 = create_asset(batch_id, "a")
        a2 = create_asset(batch_id, "b")
        if not a1 or not a2:
            raise RuntimeError("asset create failed")
        if asset_status(a1) == "IN_STOCK":
            ok("asset A1 created IN_STOCK")
        else:
            bad("asset A1 initial status", str(asset_status(a1)))
        fac = create_facility("f")
        proj = create_project("p")
        if not fac or not proj:
            raise RuntimeError("facility/project setup failed")
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/items", {"facilityCode": fac})
        if not (st == 200 and body and body.get("code") == 0):
            raise RuntimeError("add item failed: " + str(body))
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/start")
        if st == 200 and body and body.get("code") == 0:
            ok("project started BUILDING (permit gate default off)")
        else:
            bad("project start", str((st, body)))
        # ---- Phase B: 出库台账连续性 ----
        st, body = http("POST", "/odn/material-issues", {"projectId": proj, "assetIds": [a1, a2], "remark": "W8ACC 出库"})
        issue = (body or {}).get("data") if body else None
        if st == 200 and body and body.get("code") == 0 and issue and issue.get("status") == "OPEN":
            ok("issue created OPEN, assets untouched")
        else:
            raise RuntimeError("issue create failed: " + str((st, body)))
        issue_id = issue.get("id")
        if asset_status(a1) == "IN_STOCK":
            ok("OPEN issue keeps assets IN_STOCK (台账不静默漂移)")
        else:
            bad("OPEN issue changed asset status", str(asset_status(a1)))
        st, body = http("POST", "/odn/material-issues", {"projectId": proj, "assetIds": [a1]})
        if st == 200 and body and body.get("code") == 40900:
            ok("duplicate asset on active issue rejected 40900")
        else:
            bad("duplicate active-issue guard", str((st, body))[:200])
        st, body = http("POST", "/odn/material-issues/" + str(issue_id) + "/confirm")
        if st == 200 and body and body.get("code") == 0 and asset_status(a1) == "IN_TRANSIT" and asset_status(a2) == "IN_TRANSIT":
            ok("issue CONFIRMED flips assets IN_TRANSIT (出库在途不消失)")
        else:
            bad("issue confirm transit", str((asset_status(a1), asset_status(a2))))
        st, body = http("POST", "/odn/material-issues/" + str(issue_id) + "/cancel")
        if st == 200 and body and body.get("code") == 0 and asset_status(a2) == "IN_STOCK":
            ok("issue CANCELLED returns assets IN_STOCK (退库闭环)")
        else:
            bad("issue cancel restock", str((st, asset_status(a2))))
        # ---- Phase C: 转固凭证 ----
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/accept", {"note": "W8acc"})
        if not (st == 200 and body and body.get("code") == 0):
            raise RuntimeError("project accept failed: " + str(body))
        st, body = http("POST", "/odn/assets/registrations", {"entityKind": "FACILITY", "facilityCode": fac, "assetId": a1, "sourceKind": "CONSTRUCTION", "projectId": proj, "valueAmount": 123.45, "remark": "W8ACC"})
        reg = (body or {}).get("data") if body else None
        if st == 200 and body and body.get("code") == 0 and reg and reg.get("status") == "ACTIVE" and reg.get("registrationNo", "").startswith("ZG-"):
            ok("registration created ACTIVE, no ZG-... (凭证号规则)")
        else:
            raise RuntimeError("registration failed: " + str((st, body)))
        if asset_status(a1) == "DEPLOYED":
            ok("registered asset flipped DEPLOYED (施工建成转固闭环)")
        else:
            bad("asset not DEPLOYED after reg", str(asset_status(a1)))
        st, body = http("GET", "/odn/facilities?prvCode=PHL001&cityPrefix=MNL&limit=500")
        found = None
        for f in (body or {}).get("data", []):
            if f.get("code") == fac:
                found = f.get("assetReg")
                break
        if found and found.get("assetId") == a1 and found.get("registrationNo", "").startswith("ZG-"):
            ok("facility list carries assetReg (资产身份在设施视图可见)")
        else:
            bad("facility assetReg visible", str(found))
        st, body = http("POST", "/odn/assets/registrations", {"entityKind": "FACILITY", "facilityCode": fac, "assetId": a2, "sourceKind": "DIRECT"})
        if st == 200 and body and body.get("code") == 40900:
            ok("duplicate ACTIVE registration on same facility rejected 40900")
        else:
            bad("duplicate registration guard", str((st, body))[:200])
        st, body = http("POST", "/odn/assets/registrations", {"entityKind": "FACILITY", "facilityCode": fac, "assetId": a1, "sourceKind": "DIRECT"})
        if st == 200 and body and body.get("code") == 40900:
            ok("registration on non IN_STOCK/IN_TRANSIT asset rejected 40900")
        else:
            bad("asset state guard", str((st, body))[:200])
        st, body = http("POST", "/odn/assets/registrations/" + str(reg.get("id")) + "/reverse", {"reason": "W8ACC 冲销"})
        if st == 200 and body and body.get("code") == 0 and asset_status(a1) == "IN_STOCK":
            ok("reverse returns asset IN_STOCK (冲销留痕+资产归位)")
        else:
            bad("reverse", str((st, asset_status(a1))))
        # ---- Phase D: 盘点 scope=ODN ----
        st, body = http("POST", "/odn/assets/registrations", {"entityKind": "FACILITY", "facilityCode": fac, "assetId": a1, "sourceKind": "DIRECT", "valueAmount": 99.5})
        if not (st == 200 and body and body.get("code") == 0):
            raise RuntimeError("re-register after reverse failed: " + str(body))
        st, body = http("POST", "/stocktakes", {"legalEntityId": 6, "scope": "ODN"})
        if st == 200 and body and body.get("code") == 0:
            TASK_ID = int((body.get("data") or {}).get("id", 0))
            ok("stocktake scope=ODN created task " + str(TASK_ID))
        else:
            bad("stocktake ODN create", str((st, body))[:200])
        if TASK_ID:
            cnt1 = psql("SELECT count(*) FROM stocktake_items WHERE task_id = " + str(TASK_ID) + " AND asset_id = " + str(a1))
            cnt2 = psql("SELECT count(*) FROM stocktake_items WHERE task_id = " + str(TASK_ID) + " AND asset_id = " + str(a2))
            if cnt1.strip() == "1" and cnt2.strip() == "0":
                ok("ODN scope snapshots only ACTIVE-registered assets (a1 in, a2 out)")
            else:
                bad("ODN snapshot range", cnt1.strip() + "/" + cnt2.strip())
        # ---- 审计 ----
        rcnt = psql("SELECT count(*) FROM audit_logs WHERE action = " + q("odn.assetreg.create"))
        icnt = psql("SELECT count(*) FROM audit_logs WHERE action = " + q("odn.issue.confirm"))
        if int(rcnt.strip() or "0") >= 2:
            ok("audit odn.assetreg.create >= 2 (got " + rcnt.strip() + ")")
        else:
            bad("assetreg audit", rcnt)
        if int(icnt.strip() or "0") >= 1:
            ok("audit odn.issue.confirm >= 1 (got " + icnt.strip() + ")")
        else:
            bad("issue confirm audit", icnt)
        # ---- 收尾:造数即清 + quad 孤儿基线不新增 ----
        cleanup()
        after_quad = quad_orphans()
        if after_quad == base_quad:
            ok("quad orphans unchanged after acceptance (" + str(after_quad) + ", 存量债未新增)")
        else:
            bad("quad orphan delta", str(base_quad) + " -> " + str(after_quad))
    finally:
        cleanup()
    print("")
    print("=== FINAL SUMMARY ===")
    print("PASS=" + str(PASS) + " FAIL=" + str(FAIL))
    if FAIL == 0:
        print("W8 ACCEPTANCE PASS")
        sys.exit(0)
    print("W8 ACCEPTANCE FAIL")
    sys.exit(1)


if __name__ == "__main__":
    main()
