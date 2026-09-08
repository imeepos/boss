#!/usr/bin/env python3
# W6 工程预算里程碑+应付台账 102 真实环境验收(P-INFRA-1 W6;审查 F8/G2)。
# 前置:102 已部署含 W6 的 main(CI deploy-102);本机 ssh 免密 imeepos@192.168.0.102。
# 断言:就绪探针(新端点 /odn/payables 旧代次 404 新代次 200+code0;401 分不清代次)/
#       预算挂接与执行进度(SETTLED 合计派生)/里程碑编辑窗口(PENDING 可改,BUILDING 锁清单留状态,ACCEPTED 全锁)/
#       SETTLED 同事务生成应付(金额=结算应付,AP- 单号)/部分付款与分期(不超余额 40900)/
#       核减(原因必填,净应付同步,防超付 40900)/发票登记(重复 40900)/
#       VOIDED 冲销应付(原因同源)/PENDING 作废不产生应付/审计留痕;
#       造数即清(finally 兜底)+quad 孤儿基线对比不新增。
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


def data_of(body):
    return (body or {}).get("data")


def code_of(body):
    return (body or {}).get("code")


def wait_ready(seconds=300):
    # 二进制就绪探针:W6 新端点在新二进制 200+code0,旧二进制 404(路由未注册);
    # 401 只说明鉴权不通过分不清代次,故带 admin key 断言业务码。
    last = ""
    deadline = time.time() + seconds
    while time.time() < deadline:
        st, body = http("GET", "/odn/payables")
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


def get_project(pid):
    st, body = http("GET", "/odn/constructions/" + str(pid))
    if st == 200 and code_of(body) == 0:
        return (data_of(body) or {}).get("project")
    return None


def list_payables(pid=None):
    path = "/odn/payables?limit=500"
    if pid:
        path += "&projectId=" + str(pid)
    st, body = http("GET", path)
    if st == 200 and code_of(body) == 0:
        return data_of(body) or []
    return []


def create_supplier():
    code = "W6ACC-S" + str(random.randint(10000, 99999))
    st, body = http("POST", "/procurement/suppliers", {
        "code": code, "name": "W6ACC 施工供应商", "legalEntityId": 6,
        "contractorType": "CONSTRUCTION", "status": "ENABLED", "remark": "W6ACC"})
    if st == 200 and code_of(body) == 0:
        return (data_of(body) or {}).get("id")
    bad("create supplier", str((st, body))[:200])
    return None


def create_facility(suffix):
    code = "CLS9" + str(random.randint(1000, 9999))
    # T2 起 PLANNED 是创建 API 显式入参,不再 psql 直设;入施工单须 PLANNED(W8 同款)。
    st, body = http("POST", "/odn/facilities", {"code": code, "kind": "CLS", "name": "W6ACC-" + suffix, "prvCode": "PHL001", "cityPrefix": "MNL", "lifecycleStatus": "PLANNED"})
    if st == 200 and code_of(body) == 0:
        return code
    bad("create facility " + suffix, str((st, body))[:200])
    return None


def create_project(suffix):
    proj_no = "w6acc-" + suffix + "-" + str(random.randint(1000, 9999))
    st, body = http("POST", "/odn/constructions", {"projNo": proj_no, "name": "W6 验收 " + suffix})
    if st != 200 or code_of(body) != 0:
        bad("create project " + suffix, str((st, body))[:200])
        return None
    st2, body2 = http("GET", "/odn/constructions?limit=500")
    for p in data_of(body2) or []:
        if p.get("projNo") == proj_no:
            return p["id"]
    bad("locate project " + suffix, str((st2, body2))[:200])
    return None


def cleanup(supplier_id):
    steps = [
        "DELETE FROM construction_payable_invoices WHERE payable_id IN (SELECT id FROM construction_payables WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%") + "))",
        "DELETE FROM construction_payable_deductions WHERE payable_id IN (SELECT id FROM construction_payables WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%") + "))",
        "DELETE FROM construction_payable_payments WHERE payable_id IN (SELECT id FROM construction_payables WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%") + "))",
        "DELETE FROM construction_payables WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%") + ")",
        "DELETE FROM construction_settlements WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%") + ")",
        "DELETE FROM construction_milestones WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%") + ")",
        "DELETE FROM construction_items WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%") + ")",
        "DELETE FROM construction_projects WHERE proj_no LIKE " + q("w6acc-%"),
        "DELETE FROM odn_facility WHERE name LIKE " + q("W6ACC-%"),
    ]
    if supplier_id:
        steps.append("DELETE FROM procurement_suppliers WHERE id = " + str(supplier_id))
    failed = 0
    for sql in steps:
        try:
            psql(sql)
        except Exception as e:
            failed += 1
            print("[cleanup] FAILED: " + sql[:70] + " -> " + str(e))
    if failed == 0:
        print("[cleanup] data removed OK")
    return failed == 0


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
        print("FAIL: ready probe /odn/payables not ready (旧二进制或鉴权失败)")
        sys.exit(2)
    ok("ready probe: new binary deployed (/odn/payables 200 code=0)")
    base_quad = quad_orphans()
    print("[baseline] quad LINKED-but-not-DEPLOYED = " + str(base_quad))
    supplier_id = None
    try:
        # ---- Phase A: 造数:供应商+设施+项目(PENDING)+预算+里程碑+清单 ----
        supplier_id = create_supplier()
        fac = create_facility("f")
        proj = create_project("p")
        if not supplier_id or not fac or not proj:
            raise RuntimeError("setup failed")
        st, body = http("PUT", "/odn/constructions/" + str(proj) + "/contractor", {"contractorId": supplier_id})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("assign contractor failed: " + str(body))
        ok("supplier assigned to project")
        st, body = http("PUT", "/odn/constructions/" + str(proj) + "/budget", {"budgetAmount": 1000.0})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("set budget failed: " + str(body))
        p = get_project(proj)
        if p and p.get("budgetAmount") == 1000 and p.get("settledAmount") == 0:
            ok("budget set 1000, settledAmount derived 0 (PENDING)")
        else:
            bad("budget read-back", str(p)[:300])
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/milestones", {"name": "主干光缆敷设", "plannedDate": "2026-12-31"})
        m1 = (data_of(body) or {}) if code_of(body) == 0 else None
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/milestones", {"name": "光交箱安装"})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("add milestone failed: " + str(body))
        st, body = http("GET", "/odn/constructions/" + str(proj) + "/milestones")
        ms = data_of(body) or []
        if len(ms) == 2 and ms[0].get("plannedDate") == "2026-12-31":
            ok("milestones listed 2 with planned date")
        else:
            bad("milestone list", str(ms)[:200])
        if m1:
            st, body = http("PUT", "/odn/milestones/" + str(m1.get("id")), {"name": "主干光缆敷设(改)", "plannedDate": "2027-01-15"})
            if st == 200 and code_of(body) == 0:
                ok("milestone editable at PENDING")
            else:
                bad("milestone edit at PENDING", str((st, body))[:200])
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/items", {"facilityCode": fac, "quantity": 10, "unitPrice": 100})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("add item failed: " + str(body))
        p = get_project(proj)
        if p and p.get("itemsAmount") == 1000:
            ok("item amount computed server-side (10x100=1000)")
        else:
            bad("itemsAmount", str(p)[:300])
        # ---- Phase B: 开工 BUILDING:清单与预算/里程碑清单锁定,状态标记仍可 ----
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/start")
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("start failed: " + str(body))
        st, body = http("PUT", "/odn/constructions/" + str(proj) + "/budget", {"budgetAmount": 2000.0})
        if code_of(body) == 40900:
            ok("budget edit locked at BUILDING (40900)")
        else:
            bad("budget lock", str((st, code_of(body)))[:200])
        ms = data_of(http("GET", "/odn/constructions/" + str(proj) + "/milestones")[1]) or []
        m1 = ms[0] if ms else None
        if m1:
            st, body = http("PUT", "/odn/milestones/" + str(m1.get("id")), {"name": "越权改", "plannedDate": ""})
            if code_of(body) == 40900:
                ok("milestone edit locked at BUILDING (40900)")
            else:
                bad("milestone edit lock", str((st, code_of(body)))[:200])
            st, body = http("POST", "/odn/milestones/" + str(m1.get("id")) + "/complete")
            if st == 200 and code_of(body) == 0:
                ok("milestone mark allowed at BUILDING")
            else:
                bad("milestone mark at BUILDING", str((st, code_of(body)))[:200])
        # ---- Phase C: 竣工 ACCEPTED:里程碑全锁;结算 SETTLED 同事务生成应付 ----
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/accept", {"note": "W6acc"})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("accept failed: " + str(body))
        if m1:
            st, body = http("POST", "/odn/milestones/" + str(m1.get("id")) + "/reopen")
            if code_of(body) == 40900:
                ok("milestone locked at ACCEPTED (40900)")
            else:
                bad("milestone ACCEPTED lock", str((st, code_of(body)))[:200])
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/settlements")
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("create settlement failed: " + str(body))
        settle = data_of(body)
        st, body = http("POST", "/odn/settlements/" + str(settle.get("id")) + "/settle")
        sd = data_of(body) or {}
        if st == 200 and code_of(body) == 0 and sd.get("payableId") and str(sd.get("payableNo", "")).startswith("AP-"):
            ok("settle returns payable (" + str(sd.get("payableNo")) + ")")
        else:
            raise RuntimeError("settle payable missing: " + str((st, body))[:250])
        aps = list_payables(proj)
        ap = None
        for a in aps:
            if a.get("id") == sd.get("payableId"):
                ap = a
        if ap and ap.get("payableAmount") == 1000 and ap.get("status") == "OPEN" and ap.get("balance") == 1000 and ap.get("settlementNo") == settle.get("settlementNo"):
            ok("payable generated with amount=1000 OPEN, settlement ref replayable")
        else:
            bad("payable generated", str(ap)[:300])
        p = get_project(proj)
        if p and p.get("settledAmount") == 1000:
            ok("project settledAmount derived 1000 (预算执行 100%)")
        else:
            bad("settledAmount", str(p)[:300])
        # ---- Phase D: 付款/核减/发票/余额断言 ----
        apid = str(sd.get("payableId"))
        st, body = http("POST", "/odn/payables/" + apid + "/payments", {"amount": 400, "method": "TRANSFER", "reference": "W6ACC-P1"})
        if st == 200 and code_of(body) == 0:
            ok("partial payment 400 registered")
        else:
            bad("partial payment", str((st, code_of(body)))[:200])
        st, body = http("POST", "/odn/payables/" + apid + "/payments", {"amount": 700, "method": "CASH"})
        if code_of(body) == 40900:
            ok("over-balance payment rejected 40900 (余额 600 < 700)")
        else:
            bad("over-balance guard", str((st, code_of(body)))[:200])
        st, body = http("POST", "/odn/payables/" + apid + "/deductions", {"amount": 200})
        if code_of(body) in (42200, 40900):
            ok("deduction without reason rejected (" + str(code_of(body)) + ")")
        else:
            bad("deduction reason guard", str((st, code_of(body)))[:200])
        st, body = http("POST", "/odn/payables/" + apid + "/deductions", {"amount": 200, "reason": "复审工程量核减"})
        if st == 200 and code_of(body) == 0:
            ok("deduction 200 registered with reason")
        else:
            bad("deduction", str((st, code_of(body)))[:200])
        st, body = http("GET", "/odn/payables/" + apid)
        d = data_of(body) or {}
        ap2 = d.get("payable") or {}
        if ap2.get("deductedAmount") == 200 and ap2.get("balance") == 400 and ap2.get("status") == "PARTIAL":
            ok("net payable synced after deduction (1000-200, balance=800-400=400 PARTIAL)")
        else:
            bad("deduction sync", str(ap2)[:300])
        st, body = http("POST", "/odn/payables/" + apid + "/deductions", {"amount": 700, "reason": "超限核减"})
        if code_of(body) == 40900:
            ok("over-deduction rejected 40900 (核减后净应付 100 < 已付 400)")
        else:
            bad("over-deduction guard", str((st, code_of(body)))[:200])
        st, body = http("POST", "/odn/payables/" + apid + "/invoices", {"invoiceNo": "INV-W6ACC-1", "amount": 800, "invoicedAt": "2026-09-07"})
        if st == 200 and code_of(body) == 0:
            ok("invoice registered")
        else:
            bad("invoice", str((st, code_of(body)))[:200])
        st, body = http("POST", "/odn/payables/" + apid + "/invoices", {"invoiceNo": "INV-W6ACC-1", "amount": 100})
        if code_of(body) == 40900:
            ok("duplicate invoice rejected 40900")
        else:
            bad("duplicate invoice guard", str((st, code_of(body)))[:200])
        st, body = http("POST", "/odn/payables/" + apid + "/payments", {"amount": 400, "method": "CHEQUE", "reference": "W6ACC-P2"})
        if st == 200 and code_of(body) == 0:
            ok("installment payment 400 registered")
        else:
            bad("installment payment", str((st, code_of(body)))[:200])
        st, body = http("GET", "/odn/payables/" + apid)
        d = data_of(body) or {}
        ap3 = d.get("payable") or {}
        if ap3.get("status") == "PAID" and ap3.get("balance") == 0 and len(d.get("payments", [])) == 2:
            ok("paid-in-full after installments (PAID, balance 0, 2 payments)")
        else:
            bad("paid state", str(ap3)[:300])
        st, body = http("POST", "/odn/payables/" + apid + "/payments", {"amount": 1, "method": "OTHER"})
        if code_of(body) == 40900:
            ok("payment after PAID rejected 40900 (余额耗尽)")
        else:
            bad("exhausted guard", str((st, code_of(body)))[:200])
        # ---- Phase E: VOIDED 冲销路径 ----
        st, body = http("POST", "/odn/settlements/" + str(settle.get("id")) + "/void", {"reason": "W6ACC 结算冲销"})
        if not (st == 200 and code_of(body) == 0):
            raise RuntimeError("void settlement failed: " + str(body))
        st, body = http("GET", "/odn/payables/" + apid)
        ap4 = (data_of(body) or {}).get("payable") or {}
        if ap4.get("status") == "VOIDED" and ap4.get("voidReason") == "W6ACC 结算冲销" and len(data_of(body).get("payments", [])) == 2:
            ok("payable VOIDED with same-source reason, payments kept (流水留历史)")
        else:
            bad("payable void", str(ap4)[:300])
        st, body = http("POST", "/odn/payables/" + apid + "/payments", {"amount": 1, "method": "CASH"})
        if code_of(body) == 40900:
            ok("payment on VOIDED payable rejected 40900")
        else:
            bad("voided payable guard", str((st, code_of(body)))[:200])
        before = len(list_payables(proj))
        st, body = http("POST", "/odn/constructions/" + str(proj) + "/settlements")
        s2 = data_of(body)
        st, body = http("POST", "/odn/settlements/" + str(s2.get("id")) + "/void", {"reason": "PENDING 期作废不生成应付"})
        if st == 200 and code_of(body) == 0:
            after = len(list_payables(proj))
            if before == after == 1:
                ok("PENDING-void settlement generates no payable (" + str(after) + " payable)")
            else:
                bad("PENDING void payable count", str(before) + "->" + str(after))
        else:
            bad("PENDING void", str((st, code_of(body)))[:200])
        # ---- 审计 ----
        for action, minimum in (("odn.settlement.settle", 1), ("odn.payable.payment", 2), ("odn.payable.deduct", 1), ("odn.budget.set", 1)):
            cnt = psql("SELECT count(*) FROM audit_logs WHERE action = " + q(action))
            if int(cnt.strip() or "0") >= minimum:
                ok("audit " + action + " >= " + str(minimum) + " (got " + cnt.strip() + ")")
            else:
                bad("audit " + action, cnt)
        # ---- 失败路径留痕抽查(域层 [odn-payable] 日志可 grep 由部署侧 docker logs 复核,这里验证审计即可) ----
    finally:
        cleanup(supplier_id)
        after_quad = quad_orphans()
        if after_quad == base_quad:
            ok("quad orphans unchanged after acceptance (" + str(after_quad) + ")")
        else:
            bad("quad orphan delta", str(base_quad) + " -> " + str(after_quad))
    print("")
    print("=== FINAL SUMMARY ===")
    print("PASS=" + str(PASS) + " FAIL=" + str(FAIL))
    if FAIL == 0:
        print("W6 ACCEPTANCE PASS")
        sys.exit(0)
    print("W6 ACCEPTANCE FAIL")
    sys.exit(1)


if __name__ == "__main__":
    main()