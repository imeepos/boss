#!/usr/bin/env python3
# W4 ROW 路权与 PECE 许可工作流 102 真实环境验收(P-INFRA-1 W4;审查 F3/F6)。
# 前置:102 已部署含 W4 的 main(CI deploy-102);本机 ssh 免密 imeepos@192.168.0.102。
# 断言:二进制就绪探针(新端点 /odn/permits)/门控默认关存量项目可开工/门控开 无许可开工被拒
#       (40900+ROW/PECE 缺失明细)/建许可-流转获批盖章-开工成功/竣工覆盖 PENDING-SERVED 联动+审计
#       coverageServed/过期许可阻断+自动 EXPIRED+复验重走获批/门控恢复关;造数即清(finally 兜底)+孤儿巡检。
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
COMPOSE_PATH = "/home/imeepos/boss/deployments/docker-compose.102.app.yml"
PROJ_PREFIX = "w4acc-"
FAC_PREFIX = "W4ACC-"
PASS = 0
FAIL = 0
KEY = ""


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


def wait_health(seconds=120):
    deadline = time.time() + seconds
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(HEALTH, timeout=5) as resp:
                if resp.status == 200:
                    return True
        except Exception:
            pass
        time.sleep(4)
    return False


def wait_ready(seconds=300):
    # 二进制就绪探针:W4 新端点 /odn/permits 可用(404=旧二进制;部署窗口竞态防护)
    deadline = time.time() + seconds
    while time.time() < deadline:
        st, body = http("GET", "/odn/permits")
        if st == 200 and body and body.get("code") == 0:
            return True
        time.sleep(5)
    return False


def remote_rewrite_gate(on):
    # 经 stdin 传 python 到 102 宿主改写运维副本 compose(规避 ssh 叠引号;脚本内禁反斜杠转义)
    q = chr(34)
    py = []
    py.append("import io")
    py.append("p = " + q + COMPOSE_PATH + q)
    py.append("s = io.open(p, encoding=" + q + "utf-8" + q + ").read()")
    py.append("lines = [ln for ln in s.split(chr(10)) if " + q + "BOSS_ODN_PERMIT_GATE" + q + " not in ln]")
    py.append("if " + ("True" if on else "False") + ":")
    py.append("    out = []")
    py.append("    for ln in lines:")
    py.append("        if ln.startswith(" + q + "      BOSS_DEV_MODE:" + q + "):")
    py.append("            out.append(" + q + "      BOSS_ODN_PERMIT_GATE: " + q + " + chr(34) + " + q + "on" + q + " + chr(34))")
    py.append("        out.append(ln)")
    py.append("    lines = out")
    py.append("io.open(p, " + q + "w" + q + ", encoding=" + q + "utf-8" + q + ").write(chr(10).join(lines))")
    rc, out, err = ssh("python3 -", stdin_text=py.join if False else chr(10).join(py))
    return rc == 0, (out + err)[-200:]


def set_gate(on):
    good, detail = remote_rewrite_gate(on)
    if not good:
        bad("set_gate rewrite on=" + str(on), detail)
        return False
    cmd = "cd /home/imeepos/boss && docker-compose -f deployments/docker-compose.102.app.yml -p boss-app up -d --force-recreate server"
    rc, out, err = ssh(cmd, timeout=300)
    if rc != 0:
        bad("set_gate recreate", (out + err)[-300:])
        return False
    if not wait_health():
        bad("set_gate healthz", "server not healthy after recreate")
        return False
    time.sleep(2)
    return True


def gate_state():
    rc, out, err = ssh("docker exec boss-server printenv BOSS_ODN_PERMIT_GATE")
    return out.strip()

def create_project(name_suffix):
    proj_no = PROJ_PREFIX + name_suffix + "-" + str(random.randint(1000, 9999))
    st, body = http("POST", "/odn/constructions", {"projNo": proj_no, "name": "W4 验收 " + name_suffix})
    if st != 200 or not body or body.get("code") != 0:
        bad("create_project " + name_suffix, str(body))
        return None
    st2, body2 = http("GET", "/odn/constructions?limit=500")
    if st2 == 200 and body2 and body2.get("code") == 0:
        for p in body2.get("data", []):
            if p.get("projNo") == proj_no:
                return p["id"], proj_no
    return None


def create_facility(suffix):
    code = "CLS9" + str(random.randint(1000, 9999))
    st, body = http("POST", "/odn/facilities", {"code": code, "kind": "CLS", "name": FAC_PREFIX + suffix})
    if st != 200 or not body or body.get("code") != 0:
        bad("create_facility", str(body))
        return None
    st2, body2 = http("PUT", "/odn/facilities/" + code + "/lifecycle", {"lifecycleStatus": "PLANNED"})
    if st2 != 200 or not body2 or body2.get("code") != 0:
        bad("set lifecycle PLANNED", str(body2))
        return None
    return code


def add_item(proj_id, fac_code):
    st, body = http("POST", "/odn/constructions/" + str(proj_id) + "/items", {"facilityCode": fac_code})
    return st == 200 and body and body.get("code") == 0, body


def create_permit(kind, proj_id):
    st, body = http("POST", "/odn/permits", {"kind": kind, "title": "W4 验收 " + kind, "projectId": proj_id})
    if st != 200 or not body or body.get("code") != 0:
        bad("create_permit " + kind, str(body))
        return None
    return body.get("data", {})


def transition(permit_id, to, extra=None):
    body = {"to": to}
    if extra:
        body.update(extra)
    st, resp = http("POST", "/odn/permits/" + str(permit_id) + "/transition", body)
    return st, resp


def link_permit(permit_id, proj_id):
    st, body = http("POST", "/odn/permits/" + str(permit_id) + "/link", {"projectId": proj_id})
    return st == 200 and body and body.get("code") == 0, body


def start_project(proj_id):
    st, body = http("POST", "/odn/constructions/" + str(proj_id) + "/start")
    return st, body


def accept_project(proj_id):
    st, body = http("POST", "/odn/constructions/" + str(proj_id) + "/accept", {"note": "W4 acc"})
    return st, body


def create_address():
    label = "w4acc" + str(random.randint(100000, 999999))
    st, body = http("POST", "/addresses", {"parentID": 0, "label": label, "name": ADDR_PREFIX + label})
    if st != 200 or not body or body.get("code") != 0:
        bad("create_address", str(body))
        return None, None
    # 响应形状:Data 信封内 data 或裸 id;兜底按 label 查库
    data = body.get("data") or {}
    aid = data.get("id") if isinstance(data, dict) else None
    if not aid:
        aid = psql("SELECT id FROM addresses WHERE label = " + shlex.quote(label) + " ORDER BY id DESC LIMIT 1")
    return aid, label


def set_coverage(address_id, fac_code):
    st, body = http("POST", "/odn/coverage", {"addressId": int(address_id), "facilityCode": fac_code, "status": "PENDING"})
    return st == 200 and body and body.get("code") == 0, body


def get_coverage(address_id):
    st, body = http("GET", "/odn/coverage?addressId=" + str(address_id))
    if st == 200 and body and body.get("code") == 0:
        return body.get("data")
    return None

def cleanup():
    # 造数即清:许可(先,因 FK 项目/设施)/明细/项目/覆盖/地址/设施(按验收前缀)
    steps = [
        '''DELETE FROM odn_permits WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'w4acc-%')''',
        '''DELETE FROM construction_items WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'w4acc-%')''',
        '''DELETE FROM construction_projects WHERE proj_no LIKE 'w4acc-%' ''',
        '''DELETE FROM address_coverage WHERE address_id IN (SELECT id FROM addresses WHERE name LIKE 'W4ACC-%')''',
        '''DELETE FROM addresses WHERE name LIKE 'W4ACC-%' ''',
        '''DELETE FROM odn_facility WHERE name LIKE 'W4ACC-%' ''',
    ]
    failed = 0
    for sql in steps:
        try:
            psql(sql)
        except Exception as e:
            failed += 1
            print("[cleanup] FAILED: " + sql[:60] + " -> " + str(e))
    if failed == 0:
        print("[cleanup] data removed OK")
    return failed == 0


def orphan_patrol():
    # 孤儿巡检门禁(0 孤儿):本机直接跑仓库内脚本(默认 102 + admin key)
    import os
    base = os.path.dirname(os.path.abspath(__file__))
    root = os.path.abspath(os.path.join(base, "..", ".."))
    cmd = ["bash", os.path.join(root, "scripts", "ops", "db-patrol-gate.sh")]
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if r.returncode != 0:
        bad("orphan patrol gate", (r.stdout + r.stderr)[-400:])
        return False
    ok("orphan patrol gate zero")
    return True


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
    ok("api key loaded")
    if not wait_ready():
        print("FAIL: ready probe /odn/permits not ready (旧二进制或鉴权失败), 不得在旧二进制上验收");
        sys.exit(2)
    ok("ready probe: new binary deployed (/odn/permits 200 code=0)")
    try:
        # ---- Phase A: 门控默认关,存量项目无许可可开工(兼容) ----
        pa = create_project("a")
        if pa is None:
            raise RuntimeError("phase A project create failed")
        pa_id, pa_no = pa
        fa = create_facility("a")
        if not fa or not add_item(pa_id, fa)[0]:
            raise RuntimeError("phase A facility/item failed")
        st, body = start_project(pa_id)
        if st == 200 and body and body.get("code") == 0:
            ok("gate off(default): start without permits succeeds (存量兼容)")
        else:
            bad("gate off start", str((st, body)))
        # ---- 开启门控 ----
        if not set_gate(True):
            raise RuntimeError("enable gate failed")
        if gate_state() != "on":
            bad("gate env on", gate_state())
        else:
            ok("gate env switched on")
        if not wait_ready():
            raise RuntimeError("ready after gate on")
        # ---- Phase B: 无许可开工被拒(40900+缺失明细) ----
        pg = create_project("g")
        if pg is None:
            raise RuntimeError("phase B project create failed")
        pg_id, pg_no = pg
        fg = create_facility("g")
        if not fg or not add_item(pg_id, fg)[0]:
            raise RuntimeError("phase B facility/item failed")
        st, body = start_project(pg_id)
        data = (body or {}).get("data", {}) if body else {}
        reason = str(data.get("reason", "")) if isinstance(data, dict) else ""
        if st == 200 and body and body.get("code") == 40900 and "ROW" in reason and "PECE" in reason:
            ok("gate on: start rejected 40900 with ROW+PECE missing detail")
        else:
            bad("gate on reject", str((st, body)))
        st, body = http("POST", "/odn/constructions/" + str(pg_id) + "/permits-check")
        data = (body or {}).get("data", {}) if body else {}
        if st == 200 and body and body.get("code") == 0 and data.get("ok") is False:
            ok("permits-check dry-run reports blocked before permits")
        else:
            bad("permits-check blocked", str((st, body)))
        # ---- 建许可并流转至获批/盖章 ----
        row = create_permit("ROW", pg_id)
        if not row:
            raise RuntimeError("ROW permit create failed")
        transition(row["id"], "PENDING")
        st, body = transition(row["id"], "APPROVED", {"approvalNo": "W4ACC-" + str(random.randint(1000, 9999)), "validUntil": "2027-12-31"})
        if st == 200 and body and body.get("code") == 0 and body.get("data", {}).get("status") == "APPROVED":
            ok("ROW transition NOT_STARTED-PENDING-APPROVED")
        else:
            bad("ROW approve", str((st, body)))
        pece = create_permit("PECE", pg_id)
        if not pece:
            raise RuntimeError("PECE permit create failed")
        transition(pece["id"], "SIGNED")
        st, body = transition(pece["id"], "STAMPED")
        if st == 200 and body and body.get("code") == 0 and body.get("data", {}).get("status") == "STAMPED":
            ok("PECE transition PENDING_SIGN-SIGNED-STAMPED")
        else:
            bad("PECE stamp", str((st, body)))
        # 关联/解除关联端点往返(施工项目挂许可入口的数据面)
        st, body = http("POST", "/odn/permits/" + str(pece["id"]) + "/unlink")
        if st == 200 and body and body.get("code") == 0:
            ok("permit unlink ok")
        else:
            bad("permit unlink", str((st, body)))
        st, body = http("POST", "/odn/permits/" + str(pece["id"]) + "/link", {"projectId": pg_id})
        if st == 200 and body and body.get("code") == 0:
            ok("permit link ok")
        else:
            bad("permit link", str((st, body)))
        st, body = http("POST", "/odn/constructions/" + str(pg_id) + "/permits-check")
        data = (body or {}).get("data", {}) if body else {}
        if st == 200 and body and body.get("code") == 0 and data.get("ok") is True:
            ok("permits-check satisfied after approve+stamp")
        else:
            bad("permits-check satisfied", str((st, body)))
        st, body = start_project(pg_id)
        if st == 200 and body and body.get("code") == 0:
            ok("start succeeds with approved ROW + stamped PECE")
        else:
            bad("start with permits", str((st, body)))
        # ---- Phase C: F6 竣工覆盖联动(PENDING-SERVED)+审计 ----
        addr_id, addr_label = create_address()
        if not addr_id:
            raise RuntimeError("address create failed")
        ok("address created id=" + str(addr_id))
        okcov, covbody = set_coverage(addr_id, fg)
        if okcov:
            ok("coverage PENDING created on facility")
        else:
            bad("coverage PENDING", str(covbody))
        cov = get_coverage(addr_id)
        if cov and cov.get("status") == "PENDING":
            ok("coverage status PENDING before accept")
        else:
            bad("coverage before accept", str(cov))
        st, body = accept_project(pg_id)
        data = (body or {}).get("data", {}) if body else {}
        if st == 200 and body and body.get("code") == 0 and data.get("coverageServed", 0) >= 1:
            ok("accept coverageServed=" + str(data.get("coverageServed")) + " (F6 linkage fired)")
        else:
            bad("accept coverage link", str((st, body)))
        cov2 = get_coverage(addr_id)
        if cov2 and cov2.get("status") == "SERVED":
            ok("coverage SERVED after accept (F6 assertion)")
        else:
            bad("coverage after accept", str(cov2))
        acc = psql("SELECT count(*) FROM audit_logs WHERE action = " + chr(39) + "odn.construction.accept" + chr(39) + " AND target_id = " + str(pg_id) + " AND detail::text LIKE " + chr(39) + "%coverageServed%" + chr(39))
        if acc.strip() == "1":
            ok("audit odn.construction.accept carries coverageServed detail")
        else:
            bad("accept audit", acc)
        # ---- Phase D: 过期许可阻断 + 自动 EXPIRED + 复验重走获批 ----
        pe = create_project("e")
        if pe is None:
            raise RuntimeError("phase D project create failed")
        pe_id, pe_no = pe
        fe = create_facility("e")
        if not fe or not add_item(pe_id, fe)[0]:
            raise RuntimeError("phase D facility/item failed")
        rowE = create_permit("ROW", pe_id)
        if not rowE:
            raise RuntimeError("ROW E permit create failed")
        transition(rowE["id"], "PENDING")
        st, body = transition(rowE["id"], "APPROVED", {"approvalNo": "W4ACC-E-" + str(random.randint(1000, 9999)), "validUntil": "2026-01-01"})
        if st == 200 and body and body.get("code") == 0:
            ok("ROW permit approved with past validUntil (for expire test)")
        else:
            bad("ROW E approve", str((st, body)))
        peceE = create_permit("PECE", pe_id)
        if not peceE:
            raise RuntimeError("PECE E permit create failed")
        transition(peceE["id"], "SIGNED")
        transition(peceE["id"], "STAMPED")
        st, body = start_project(pe_id)
        data = (body or {}).get("data", {}) if body else {}
        reason = str(data.get("reason", "")) if isinstance(data, dict) else ""
        if st == 200 and body and body.get("code") == 40900 and "ROW" in reason:
            ok("expired permit blocks start (40900)")
        else:
            bad("expired block", str((st, body)))
        st, body = http("GET", "/odn/permits/" + str(rowE["id"]))
        d = (body or {}).get("data", {}) if body else {}
        if st == 200 and body and body.get("code") == 0 and d.get("status") == "EXPIRED":
            ok("permit auto-expired to EXPIRED by gate")
        else:
            bad("auto expire", str((st, body)))
        st, body = transition(rowE["id"], "PENDING")
        if st == 200 and body and body.get("code") == 0:
            ok("EXPIRED-PENDING recheck (复验) allowed")
        else:
            bad("recheck", str((st, body)))
        st, body = transition(rowE["id"], "APPROVED", {"approvalNo": "W4ACC-E2-" + str(random.randint(1000, 9999)), "validUntil": "2027-06-30"})
        if st != 200 or not body or body.get("code") != 0:
            bad("reapprove after recheck", str((st, body)))
        st, body = start_project(pe_id)
        if st == 200 and body and body.get("code") == 0:
            ok("start succeeds after recheck + reapproval")
        else:
            bad("start after recheck", str((st, body)))
        # ---- Phase E: 门控恢复默认关 ----
        if not set_gate(False):
            raise RuntimeError("disable gate failed")
        if gate_state() == "":
            ok("gate env restored off (default)")
        else:
            bad("gate restored", gate_state())
        if not wait_ready():
            raise RuntimeError("ready after gate restore")
        pb = create_project("b")
        if pb is None:
            raise RuntimeError("phase E project create failed")
        pb_id, pb_no = pb
        fb = create_facility("b")
        if not fb or not add_item(pb_id, fb)[0]:
            raise RuntimeError("phase E facility/item failed")
        st, body = start_project(pb_id)
        if st == 200 and body and body.get("code") == 0:
            ok("gate off restored: start without permits succeeds again")
        else:
            bad("gate off restored start", str((st, body)))
        # ---- 审计断言(许可写操作全部入审计) ----
        pcnt = psql("SELECT count(*) FROM audit_logs WHERE action = " + chr(39) + "odn.permit.create" + chr(39))
        tcnt = psql("SELECT count(*) FROM audit_logs WHERE action = " + chr(39) + "odn.permit.transition" + chr(39))
        if int(pcnt.strip() or "0") >= 4:
            ok("audit odn.permit.create >= 4 (got " + pcnt.strip() + ")")
        else:
            bad("permit create audit", pcnt)
        if int(tcnt.strip() or "0") >= 8:
            ok("audit odn.permit.transition >= 8 (got " + tcnt.strip() + ")")
        else:
            bad("permit transition audit", tcnt)
        # ---- 收尾:造数即清 + 孤儿巡检归零 ----
        cleanup()
        orphan_patrol()
    finally:
        cleanup()
    print("")
    print("=== FINAL SUMMARY ===");
    print("PASS=" + str(PASS) + " FAIL=" + str(FAIL));
    if FAIL == 0:
        print("W4 ACCEPTANCE PASS");
        sys.exit(0)
    print("W4 ACCEPTANCE FAIL");
    sys.exit(1)


if __name__ == "__main__":
    main()