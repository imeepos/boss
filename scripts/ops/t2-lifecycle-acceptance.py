#!/usr/bin/env python3
# T2 生命周期选择器验收(P-INFRA-1 T2,2026-09-08 裁定):对真实 102 环境逐断言打点。
# 前提:feat/t2-lifecycle-selector 已合入 main 并经 deploy-102 滚出(新二进制 POST
# /odn/facilities 响应含 lifecycleStatus 回显,旧二进制无此字段,即探针)。
# 造数即清:验收自建设施/施工项目,收尾按特征模式化删除+孤儿巡检。
# 用法: python3 scripts/ops/t2-lifecycle-acceptance.py
# 环境: BASE_URL/ADMIN_API_KEY/SSH_HOST 可覆盖。
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
BASE = os.environ.get("BASE_URL", "http://192.168.0.102:28080")
ACCOUNTS = os.path.join(ROOT, ".agents", "skills", "bossctl-cli", "test-accounts.json")
with open(ACCOUNTS) as f:
    acc = json.load(f)
AKEY = os.environ.get("ADMIN_API_KEY", acc["admin"]["apiKeys"][0]["key"])
SSH_HOST = os.environ.get("SSH_HOST", "imeepos@192.168.0.102")
STAMP = str(int(time.time()))
OK = 0
FAIL = 0


def http(method, path, body=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("X-API-Key", AKEY)
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return json.loads(r.read().decode()), None
    except urllib.error.HTTPError as e:
        try:
            return json.loads(e.read().decode()), None
        except Exception:
            return None, str(e)
    except Exception as e:
        return None, str(e)


def check(name, cond, detail=None):
    global OK, FAIL
    tag = "OK " if cond else "FAIL"
    if cond:
        OK += 1
    else:
        FAIL += 1
    print(f"[{tag}] {name}" + ("" if cond else f"  detail={json.dumps(detail, ensure_ascii=False)[:300]}"))


def sql(query):
    cmd = ["ssh", "-o", "ConnectTimeout=10", "-o", "BatchMode=yes", SSH_HOST,
           "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"]
    return subprocess.run(cmd, input=query, capture_output=True, text=True, timeout=60)


def cleanup():
    # 模式化大扫除:按验收特征(proj_no/设施名)清除全部历史与当轮造数,可重入。
    lines = [
        "BEGIN;",
        "DELETE FROM construction_progress WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'ACC-T2LC-%');",
        "DELETE FROM construction_items WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'ACC-T2LC-%');",
        "DELETE FROM construction_settlements WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'ACC-T2LC-%');",
        "DELETE FROM construction_projects WHERE proj_no LIKE 'ACC-T2LC-%';",
        "DELETE FROM construction_progress WHERE facility_code IN (SELECT code FROM odn_facility WHERE name LIKE 'T2LC验收%');",
        "DELETE FROM construction_items WHERE facility_code IN (SELECT code FROM odn_facility WHERE name LIKE 'T2LC验收%');",
        "DELETE FROM odn_facility WHERE name LIKE 'T2LC验收%';",
        "COMMIT;",
    ]
    r = sql("\n".join(lines))
    if r.returncode != 0:
        print("[cleanup] FAILED: " + r.stderr.strip()[:300])
        return False
    vsql = (
        "SELECT 'proj', count(*) FROM construction_projects WHERE proj_no LIKE 'ACC-T2LC-%';"
        "SELECT 'fac', count(*) FROM odn_facility WHERE name LIKE 'T2LC验收%';"
    )
    v = sql(vsql)
    if v.returncode != 0 or "0" not in v.stdout.replace("\n", ""):
        print("[cleanup] VERIFY FAILED: " + v.stdout.strip() + v.stderr.strip()[:200])
        return False
    print("[cleanup] PASS 模式化大扫除:施工项目/明细/设施全清")
    return True


def next_fac_code():
    facs, _ = http("GET", "/api/admin/v1/odn/facilities?kind=P&gridCode=91")
    mx = 0
    for f in (facs or {}).get("data") or []:
        c = str(f.get("code", ""))
        if c.startswith("P91") and c[3:].isdigit():
            mx = max(mx, int(c[3:]))
    return "P91" + str(mx + 1).zfill(3)


def main():
    print("== T2 lifecycle acceptance @ " + BASE + " stamp=" + STAMP)
    cleanup()

    # 探针:新二进制创建响应回显 lifecycleStatus(旧二进制 data 仅 code)。
    fac_p = next_fac_code()
    body, err = http("POST", "/api/admin/v1/odn/facilities", {
        "kind": "P", "code": fac_p, "prvCode": "PHL001", "cityPrefix": "MNL",
        "gridCode": 91, "name": "T2LC验收-PLANNED", "lat": 14.606, "lng": 120.991})
    new_bin = isinstance(body, dict) and (body.get("data") or {}).get("lifecycleStatus") == "PLANNED"
    if not new_bin:
        print("[probe] 二进制非 T2 版(响应无 lifecycleStatus 回显),中止不造数。resp=" + json.dumps(body, ensure_ascii=False)[:200])
        sys.exit(2)
    check("P1 默认态:不传 lifecycleStatus 创建 → 回显 PLANNED", True, body)

    d, _ = http("GET", "/api/admin/v1/odn/facilities/" + fac_p)
    gd = (d or {}).get("data") or {}
    check("P2 默认态:详情 lifecycleStatus=PLANNED(可入施工单前提)", gd.get("lifecycleStatus") == "PLANNED", d)

    proj_no = "ACC-T2LC-" + STAMP
    p, _ = http("POST", "/api/admin/v1/odn/constructions", {"projNo": proj_no, "name": "T2LC验收项目"})
    pl, _ = http("GET", "/api/admin/v1/odn/constructions")
    hit = [x for x in ((pl or {}).get("data") or []) if x.get("projNo") == proj_no]
    check("P3 创建施工项目", len(hit) == 1, pl)
    pid = hit[0]["id"] if hit else 0

    r, _ = http("POST", f"/api/admin/v1/odn/constructions/{pid}/items",
                {"facilityCode": fac_p, "quantity": 2, "unitPrice": 100})
    check("P4 PLANNED 设施可入施工单(默认态语义闭环)", r is not None and (r or {}).get("code") == 0, r)

    fac_s = next_fac_code()
    body2, _ = http("POST", "/api/admin/v1/odn/facilities", {
        "kind": "P", "code": fac_s, "prvCode": "PHL001", "cityPrefix": "MNL",
        "gridCode": 91, "name": "T2LC验收-IN_SERVICE", "lifecycleStatus": "IN_SERVICE"})
    check("P5 显式 IN_SERVICE 创建 → 回显 IN_SERVICE", (body2 or {}).get("data", {}).get("lifecycleStatus") == "IN_SERVICE", body2)
    d2, _ = http("GET", "/api/admin/v1/odn/facilities/" + fac_s)
    gd2 = (d2 or {}).get("data") or {}
    check("P6 显式 IN_SERVICE 台账正确(详情四要素)", gd2.get("lifecycleStatus") == "IN_SERVICE"
          and gd2.get("code") == fac_s and gd2.get("kind") == "P" and gd2.get("gridCode") == 91, d2)
    r2, _ = http("POST", f"/api/admin/v1/odn/constructions/{pid}/items",
                 {"facilityCode": fac_s, "quantity": 1, "unitPrice": 1})
    check("P7 IN_SERVICE 设施不进施工单(CodeConflict 40900)", (r2 or {}).get("code") == 40900, r2)

    r3, _ = http("POST", "/api/admin/v1/odn/facilities", {
        "kind": "P", "code": "P91999", "prvCode": "PHL001", "cityPrefix": "MNL",
        "gridCode": 91, "lifecycleStatus": "RETIRED"})
    check("P8 RETIRED 出生态拒绝(CodeInvalidParam 42200)", (r3 or {}).get("code") == 42200, r3)

    ok = cleanup()
    print(f"== done ok={OK} fail={FAIL} cleanup={'PASS' if ok else 'FAIL'}")
    sys.exit(0 if (FAIL == 0 and ok) else 1)


if __name__ == "__main__":
    main()
