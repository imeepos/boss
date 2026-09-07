def cleanup():
    ids = ",".join(str(i) for i in TRACK["surveys"]) or "0"
    pid = str(TRACK["project"])
    fac = TRACK["facility"]
    lines = [
        "BEGIN;",
        "DELETE FROM survey_task_reports WHERE task_id IN (" + ids + ");",
        "DELETE FROM survey_tasks WHERE id IN (" + ids + ");",
    ]
    if TRACK["project"] > 0:
        lines.append("DELETE FROM construction_progress WHERE project_id = " + pid + ";")
        lines.append("DELETE FROM construction_items WHERE project_id = " + pid + ";")
        lines.append("DELETE FROM construction_projects WHERE id = " + pid + ";")
    if fac:
        lines.append("DELETE FROM construction_items WHERE facility_code = '" + fac + "';")
        lines.append("DELETE FROM odn_facility WHERE code = '" + fac + "';")
    lines.append("COMMIT;")
    try:
        cmd = ["ssh", "-o", "ConnectTimeout=10", "-o", "BatchMode=yes", SSH_HOST,
               "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"]
        r = subprocess.run(cmd, input="\n".join(lines), capture_output=True, text=True, timeout=60)
        if r.returncode != 0:
            print("[cleanup] FAILED: " + r.stderr.strip()[:300])
            return False
        print("[cleanup] PASS 造数已清 surveys=" + ids + " project=" + pid + " facility=" + fac)
        return True
    except Exception as e:
        print("[cleanup] FAILED: " + str(e))
        return False


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

    # B 施工进度(F5a):建 PLANNED 设施->建项目->挂明细->开工->师傅上报->admin 可见
    nc, _ = api("GET", "/api/admin/v1/odn/facility-next-code?kind=P&gridCode=91")
    fac = (nc or {}).get("code") or (nc or {}).get("nextCode") or ""
    fac = str(fac) or ("ACCW7P" + STAMP[-4:])
    TRACK["facility"] = fac
    r, _ = api("POST", "/api/admin/v1/odn/facilities", {"code": fac, "kind": "P",
        "prvCode": "PHL001", "cityPrefix": "MNL", "gridCode": 91,
        "name": "W7验收杆", "lat": 14.606, "lng": 120.991})
    check("B1 创建 PLANNED 验收设施 " + fac, r is not None, r)

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
    gate = subprocess.run(["bash", os.path.join(ROOT, "scripts", "ops", "db-patrol-gate.sh")],
                          capture_output=True, text=True, timeout=300)
    check("E1 孤儿巡检门禁归零", gate.returncode == 0, gate.stdout.strip()[-200:])

    print("")
    print("==== W7 验收汇总 PASS=" + str(OK) + " FAIL=" + str(FAIL) + " ====")
    if FAIL > 0 or not clean_ok:
        sys.exit(1)


if __name__ == "__main__":
    main()