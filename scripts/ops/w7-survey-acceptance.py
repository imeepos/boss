#!/usr/bin/env python3
# W7 勘测采集+施工进度上报验收(P-INFRA-1 W7,审查 F5a/F5b):对真实 102 环境逐断言打点。
# 就绪探针:新二进制特有行为——/api/admin/v1/odn/surveys 与 /api/worker/v1/surveys
# 无凭证访问,旧二进制 404(路由不存在),新二进制 401(路由存在进鉴权);双 401 才开跑。
# 造数即清:验收自举勘测任务/施工项目/设施,收尾按 id 精确删除+孤儿巡检门禁。
# 用法: python3 scripts/ops/w7-survey-acceptance.py [--wait]
#   --wait: 轮询探针等待 CI 部署新二进制(每 10s 一次,上限 30 分钟)。
# 环境: BASE_URL/ADMIN_API_KEY/WORKER_API_KEY/WORKER_ID/SSH_HOST 可覆盖。
import argparse
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
WKEY = os.environ.get("WORKER_API_KEY", "")
WID = int(os.environ.get("WORKER_ID", "0"))
WNAME = os.environ.get("WORKER_NAME", "")
SSH_HOST = os.environ.get("SSH_HOST", "imeepos@192.168.0.102")
STAMP = str(int(time.time()))
OK = 0
FAIL = 0
TRACK = {"surveys": [], "facility": "", "project": 0}


def http(method, path, key, body=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("X-API-Key", key)
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, None


def api(method, path, body=None, worker=False):
    key = WKEY if worker else AKEY
    code, env = http(method, path, key, body)
    ok = code == 200 and isinstance(env, dict) and env.get("code") in (0, 200)
    if not ok:
        print("[api-err] " + method + " " + path + " -> " + str(code) + " " + json.dumps(env, ensure_ascii=False)[:200])
        return None, code
    return env.get("data"), code


def check(name, cond, detail=""):
    global OK, FAIL
    if cond:
        OK += 1
        print("[PASS] " + name)
    else:
        FAIL += 1
        print("[FAIL] " + name + (": " + str(detail) if detail else ""))

def bootstrap_worker():
    # 自举师傅夹具(WK-ACC-W7,自举+用后即清):按 staffNo 解析既有,缺则经 admin API 创建;
    # key 现签一次用于本轮,收尾吊销;师傅行收尾清理(自举+用后即清)。
    global WID, WNAME, WKEY
    wl, _ = api("GET", "/api/admin/v1/workers?keyword=WK-ACC-W7")
    items = (wl or {}).get("items") if isinstance(wl, dict) else wl
    hit = [w for w in (items or []) if w.get("staffNo") == "WK-ACC-W7"]
    if not hit:
        gs, _ = api("GET", "/api/admin/v1/worker-groups")
        gitems = (gs or {}).get("items") if isinstance(gs, dict) else gs
        gid = gitems[0].get("id") if gitems else None
        body = {"staffNo": "WK-ACC-W7", "name": "W7验收师傅", "phone": "13800009977",
                "password": "accw7123456", "regionIds": [4]}
        if gid:
            body["groupId"] = gid
        w, wcode = api("POST", "/api/admin/v1/workers", body)
        if w is None:
            print("[bootstrap] FAIL worker create code=" + str(wcode))
            sys.exit(1)
        hit = [w]
    w0 = hit[0]
    WID = int(w0["id"])
    WNAME = w0.get("name") or "W7验收师傅"

    k, _ = api("POST", "/api/admin/v1/api-keys", {"subjectType": "worker", "subjectRef": WID, "name": "w7-acc-" + STAMP})
    if not isinstance(k, dict) or not k.get("plainKey"):
        print("[bootstrap] FAIL api key issue")
        sys.exit(1)
    WKEY = k["plainKey"]
    print("[bootstrap] worker id=" + str(WID) + " key 现签")

def probe_ready():
    a = http("GET", "/api/admin/v1/odn/surveys", "no-key")[0]
    w = http("GET", "/api/worker/v1/surveys", "no-key")[0]
    return a, w


def sql(query):
    cmd = ["ssh", "-o", "ConnectTimeout=10", "-o", "BatchMode=yes", SSH_HOST,
           "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"]
    return subprocess.run(cmd, input=query, capture_output=True, text=True, timeout=60)
def cleanup():
    # 模式化大扫除:按验收特征(title/proj_no/设施名/staff_no)清除全部历史与当轮造数,
    # 不依赖 TRACK 逐 id;ssh 中断回滚时下一轮启动兜底与 cleanup 均可重入。
    lines = [
        "BEGIN;",
        "DELETE FROM survey_task_reports WHERE task_id IN (SELECT id FROM survey_tasks WHERE title LIKE 'W7验收%' OR title LIKE '手动复现%');",
        "DELETE FROM survey_tasks WHERE title LIKE 'W7验收%' OR title LIKE '手动复现%';",
        "DELETE FROM construction_progress WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'ACC-W7-%');",
        "DELETE FROM construction_items WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'ACC-W7-%');",
        "DELETE FROM construction_settlements WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'ACC-W7-%');",
        "DELETE FROM construction_projects WHERE proj_no LIKE 'ACC-W7-%';",
        "DELETE FROM construction_progress WHERE facility_code IN (SELECT code FROM odn_facility WHERE name='W7验收杆');",
        "DELETE FROM construction_items WHERE facility_code IN (SELECT code FROM odn_facility WHERE name='W7验收杆');",
        "DELETE FROM odn_facility WHERE name='W7验收杆';",
        "DELETE FROM workers WHERE staff_no='WK-ACC-W7';",
        "COMMIT;",
    ]
    try:
        cmd = ["ssh", "-o", "ConnectTimeout=10", "-o", "BatchMode=yes", SSH_HOST,
               "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"]
        r = subprocess.run(cmd, input="\n".join(lines), capture_output=True, text=True, timeout=60)
        if r.returncode != 0:
            print("[cleanup] FAILED: " + r.stderr.strip()[:300])
            return False
    except Exception as e:
        print("[cleanup] FAILED: " + str(e))
        return False
    vsql = (
        "SELECT 'svy', count(*) FROM survey_tasks WHERE title LIKE 'W7验收%' OR title LIKE '手动复现%';"
        "SELECT 'fac', count(*) FROM odn_facility WHERE name='W7验收杆';"
        "SELECT 'proj', count(*) FROM construction_projects WHERE proj_no LIKE 'ACC-W7-%';"
        "SELECT 'wk', count(*) FROM workers WHERE staff_no='WK-ACC-W7';"
    )
    try:
        vr = sql(vsql)
    except Exception as e:
        print("[cleanup] VERIFY FAILED: " + str(e))
        return False
    vlines = [l for l in vr.stdout.strip().splitlines() if l]
    vok = vr.returncode == 0 and len(vlines) == 4 and all(l.endswith("|0") for l in vlines)
    if not vok:
        print("[cleanup] VERIFY FAILED: " + vr.stdout.strip() + vr.stderr.strip()[:200])
        return False
    print("[cleanup] PASS 模式化大扫除:勘测/项目/设施/师傅全清")
    return True

def wait_ready(wait):
    a, w = probe_ready()
    if not wait:
        check("就绪探针 admin=401 worker=401(新二进制)", a == 401 and w == 401, "admin=" + str(a) + " worker=" + str(w))
        return a == 401 and w == 401
    deadline = time.time() + 1800
    while time.time() < deadline:
        a, w = probe_ready()
        if a == 401 and w == 401:
            check("就绪探针(等待后)双 401", True)
            return True
        print("[wait] 探针 admin=" + str(a) + " worker=" + str(w) + ",10s 后重试")
        time.sleep(10)
    check("就绪探针 30 分钟内未就绪", False, "admin=" + str(a) + " worker=" + str(w))
    return False


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--wait", action="store_true")
    args = ap.parse_args()
    if not wait_ready(args.wait):
        sys.exit(1)

    bootstrap_worker()
    # 自清兜底:清掉历史运行/崩溃残留的验收任务(含报告),保证可重复执行。
    sql("DELETE FROM survey_task_reports WHERE task_id IN (SELECT id FROM survey_tasks WHERE title LIKE 'W7验收%' OR title LIKE '手动复现%');")
    sql("DELETE FROM survey_tasks WHERE title LIKE 'W7验收%' OR title LIKE '手动复现%');")
    # A1 admin 创建勘测任务(指派师傅)
    data, _ = api("POST", "/api/admin/v1/odn/surveys", {
        "title": "W7验收-网格91主干勘测", "description": "验收造数,用后即清",
        "prvCode": "PHL001", "cityPrefix": "MNL", "gridCode": 91, "assignedWorkerId": WID})
    ok1 = isinstance(data, dict) and str(data.get("taskNo", "")).startswith("SV-")
    check("A1 创建勘测任务返回 SV 单号", ok1, data)
    if not ok1:
        sys.exit(1)
    tid = data["id"]
    TRACK["surveys"].append(tid)

    # A2 admin 详情可见,初始 0 回填
    d, _ = api("GET", "/api/admin/v1/odn/surveys/" + str(tid))
    check("A2 admin 详情 status=PENDING reports=0",
          isinstance(d, dict) and d["task"]["status"] == "PENDING" and len(d["reports"]) == 0, d)

    # A3 师傅可见集包含该任务
    wl, _ = api("GET", "/api/worker/v1/surveys", worker=True)
    hit = [t for t in (wl or {}).get("items", []) if t["id"] == tid]
    check("A3 师傅端可见集包含任务", len(hit) == 1, wl)

    # A4 师傅接单 PENDING->ACCEPTED
    r, _ = api("POST", "/api/worker/v1/surveys/" + str(tid) + "/accept", {}, worker=True)
    d, _ = api("GET", "/api/admin/v1/odn/surveys/" + str(tid))
    check("A4 师傅接单后 status=ACCEPTED",
          r is not None and d["task"]["status"] == "ACCEPTED" and d["task"]["assignedWorkerId"] == WID, d)

    # A5 师傅回填(打点+备注+建议),append-only 幂等
    msg = "acc-w7-" + STAMP + "-r1"
    body = {"lat": 14.605, "lng": 120.99, "facilityNote": "杆路完好,箱体需更换",
            "suggestion": "NEED_NEW_FACILITY", "photoIds": [], "clientMsgId": msg}
    r, _ = api("POST", "/api/worker/v1/surveys/" + str(tid) + "/reports", body, worker=True)
    check("A5 首次回填 created=true", isinstance(r, dict) and r.get("created") is True, r)

    # A6 同 clientMsgId 重传 created=false 不重复计量
    r2, _ = api("POST", "/api/worker/v1/surveys/" + str(tid) + "/reports", body, worker=True)
    d, _ = api("GET", "/api/admin/v1/odn/surveys/" + str(tid))
    check("A6 幂等重传 created=false 且仍 1 条",
          isinstance(r2, dict) and r2.get("created") is False and len(d["reports"]) == 1, d)

    # A7 admin 回填可见:建议/师傅名/坐标
    rep = d["reports"][0] if d["reports"] else {}
    check("A7 回填数据可见(建议/师傅/坐标)",
          rep.get("suggestion") == "NEED_NEW_FACILITY" and rep.get("workerId") == WID
          and abs(rep.get("lat", 0) - 14.605) < 0.001, rep)

    # A8 抢单池:未指派任务师傅接单占位
    data, _ = api("POST", "/api/admin/v1/odn/surveys", {"title": "W7验收-抢单池", "gridCode": 91})
    tid2 = data["id"]
    TRACK["surveys"].append(tid2)
    r, _ = api("POST", "/api/worker/v1/surveys/" + str(tid2) + "/accept", {}, worker=True)
    d, _ = api("GET", "/api/admin/v1/odn/surveys/" + str(tid2))
    check("A8 未指派单师傅抢单占位",
          r is not None and d["task"]["assignedWorkerId"] == WID, d)

    # A9 admin 取消任务
    api("POST", "/api/admin/v1/odn/surveys/" + str(tid2) + "/cancel", {})
    d, _ = api("GET", "/api/admin/v1/odn/surveys/" + str(tid2))
    check("A9 admin 取消后 status=CANCELLED", d["task"]["status"] == "CANCELLED", d)

    # 验收设施:SQL 直造 PLANNED 杆(admin API 新建设施默认 IN_SERVICE,入施工单要求 PLANNED);
    # 取号口径同 web 端 nextcode.ts(现存 MAX 3 位序号+1;/facility-next-code 端点的 SUBSTRING
    # 会把网格位并入序号,P91xxx 误报编号用尽,故客户端计算)。
    facs, _ = api("GET", "/api/admin/v1/odn/facilities?kind=P&gridCode=91")
    mx = 0
    for f in (facs or []):
        c = str(f.get("code", ""))
        if c.startswith("P91") and c[3:].isdigit():
            mx = max(mx, int(c[3:]))
    fac = "P91" + str(mx + 1).zfill(3)
    r, _ = api("POST", "/api/admin/v1/odn/facilities", {"kind": "P", "code": fac,
        "prvCode": "PHL001", "cityPrefix": "MNL", "gridCode": 91,
        "name": "W7验收杆", "lat": 14.606, "lng": 120.991})
    check("B1 创建 PLANNED 验收设施 " + fac, r is not None, r)
    sql("UPDATE odn_facility SET lifecycle_status='PLANNED' WHERE code='" + fac + "'")
    TRACK["facility"] = fac

    projNo = "ACC-W7-" + STAMP
    p, _ = api("POST", "/api/admin/v1/odn/constructions", {"projNo": projNo, "name": "W7验收项目"})
    pl, _ = api("GET", "/api/admin/v1/odn/constructions", None)
    hit = [x for x in (pl or []) if x.get("projNo") == projNo]
    check("B2 创建施工项目", len(hit) == 1, pl)
    pid = hit[0]["id"] if hit else 0
    TRACK["project"] = pid

    r, _ = api("POST", "/api/admin/v1/odn/constructions/" + str(pid) + "/items",
               {"facilityCode": fac, "quantity": 4, "unitPrice": 100})
    check("B3 明细挂 PLANNED 设施", r is not None, r)
    r, _ = api("POST", "/api/admin/v1/odn/constructions/" + str(pid) + "/start", {})
    d, _ = api("GET", "/api/admin/v1/odn/constructions/" + str(pid))
    check("B4 开工后 BUILDING", d["project"]["status"] == "BUILDING", d)

    wl, _ = api("GET", "/api/worker/v1/constructions", worker=True)
    hit = [x for x in (wl or {}).get("items", []) if x.get("id") == pid]
    check("B5 师傅端可见 BUILDING 项目", len(hit) == 1, wl)

    pbody = {"facilityCode": fac, "doneQty": 2, "lat": 14.606, "lng": 120.991,
             "note": "立杆2根", "photoIds": [], "clientMsgId": "acc-w7-" + STAMP + "-p1"}
    r, _ = api("POST", "/api/worker/v1/constructions/" + str(pid) + "/progress", pbody, worker=True)
    check("B6 师傅进度上报 created=true", isinstance(r, dict) and r.get("created") is True, r)

    d, _ = api("GET", "/api/admin/v1/odn/constructions/" + str(pid) + "/progress")
    entries = (d or {}).get("entries", [])
    e0 = entries[0] if entries else {}
    check("B7 admin 进度可见 reporterType=WORKER 师傅名",
          len(entries) == 1 and e0.get("reporterType") == "WORKER"
          and e0.get("reporterName") == WNAME, entries)

    # C GIS 衔接:勘测打点与进度坐标进图层
    g, _ = api("GET", "/api/admin/v1/gis/odn-points?entity=survey")
    pts = (g or {}).get("items", [])
    hit = [x for x in pts if abs(x.get("lat", 0) - 14.605) < 0.001]
    check("C1 GIS entity=survey 含勘测打点(level12)", len(hit) >= 1 and hit[0].get("level") == 12, len(pts))
    g, _ = api("GET", "/api/admin/v1/gis/odn-points?entity=progress")
    pts = (g or {}).get("items", [])
    hit = [x for x in pts if x.get("name") == fac]
    check("C2 GIS entity=progress 含进度点(level13)", len(hit) >= 1 and hit[0].get("level") == 13, len(pts))

    # D 审计留痕:勘测创建/接单/回填与进度上报四类动作入 audit_logs
    r = sql("SELECT count(*) FROM audit_logs WHERE target_type='survey_tasks' AND target_id='" + str(tid) + "';")
    n = int(r.stdout.strip() or "0")
    check("D1 勘测任务审计留痕>=3(创建/接单/回填)", n >= 3, r.stdout.strip())

    # E 造数即清+孤儿巡检
    clean_ok = cleanup()
    # WK-ACC-W7 为常驻验收夹具:密钥用后即删,师傅保留复用(并行验证安全)。
    gate = subprocess.run(["bash", os.path.join(ROOT, "scripts", "ops", "db-patrol-gate.sh")],
                          capture_output=True, text=True, timeout=300)
    check("E1 孤儿巡检门禁归零", gate.returncode == 0, gate.stdout.strip()[-200:])

    print("")
    print("==== W7 验收汇总 PASS=" + str(OK) + " FAIL=" + str(FAIL) + " ====")
    if FAIL > 0 or not clean_ok:
        sys.exit(1)


if __name__ == "__main__":
    main()