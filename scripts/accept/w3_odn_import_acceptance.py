#!/usr/bin/env python3
# W3 ODN 资源表批量导入 102 真实环境验收(P-INFRA-1 W3)。
# 前置:102 已部署含 W3 的 main(CI deploy-102),本机 ssh 免密 imeepos@192.168.0.102。
# 断言:模板真实导入示例行被滤/六规则校验(总分光比不一致报行号、层级断裂、枚举非法)/
#       链展开为设备(父链+PLANNED)/批次清单/指纹去重/孤儿半链巡检归零;造数即清(trap 兜底)。
import json
import re
import shlex
import subprocess
import sys
import urllib.request
import zipfile
import xml.etree.ElementTree as ET

BASE = "http://192.168.0.102:28080/api/admin/v1"
SSH = "imeepos@192.168.0.102"
TEMPLATE = "docs/ODN网络资源表模板.xlsx"
BATCH_PREFIX = "w3acc-"
CODES = ["OCC901", "ODB901", "OBD901", "SDB901", "SBD901",
         "ODB902", "OBD902", "SDB902", "SBD902"]

PASS = 0
FAIL = 0
TOKEN = ""


def ok(name):
    global PASS
    PASS += 1
    print("PASS: " + name)


def bad(name, detail=""):
    global FAIL
    FAIL += 1
    print("FAIL: " + name + ("  [" + detail + "]" if detail else ""))


def http(method, path, token, body=None):
    req = urllib.request.Request(BASE + path, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    data = json.dumps(body).encode() if body is not None else None
    with urllib.request.urlopen(req, data=data, timeout=15) as resp:
        return json.loads(resp.read().decode())


def psql(sql):
    cmd = "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc " + shlex.quote(sql)
    out = subprocess.run(["ssh", "-o", "BatchMode=yes", SSH, cmd],
                         capture_output=True, text=True, timeout=30)
    if out.returncode != 0:
        raise RuntimeError("psql failed: " + out.stderr.strip())
    return out.stdout.strip()


def cleanup(token, pre_chains):
    try:
        psql("DELETE FROM odn_resource_chain WHERE occ_code='OCC901'"
             " OR odb_code IN ('ODB902','ODB903')")
        psql("DELETE FROM odn_device WHERE prv_code IS NULL AND code IN (" +
             ",".join("'" + c + "'" for c in CODES) + ")")
        p = http("GET", "/odn/resource-chains/patrol", token)
        d = p.get("data") or {}
        okc = d.get("orphanBoxCode") == 0 and d.get("brokenAncestor") == 0 and d.get("chains", 1) == pre_chains
        if okc:
            ok("清理后巡检归零(chains=%s)" % d.get("chains"))
        else:
            bad("清理后巡检归零", json.dumps(d))
    except Exception as e:  # noqa: BLE001 清理失败必须大声报
        bad("清理", str(e))


def xlsx_parse(path):
    ns = {"m": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}
    z = zipfile.ZipFile(path)
    wb = ET.fromstring(z.read("xl/workbook.xml"))
    names = [s.get("name") for s in wb.findall(".//m:sheet", ns)]
    sst = []
    if "xl/sharedStrings.xml" in z.namelist():
        root = ET.fromstring(z.read("xl/sharedStrings.xml"))
        for si in root.findall("m:si", ns):
            sst.append("".join(t.text or "" for t in si.findall(".//m:t", ns)))

    def sheet_rows(idx):
        ws = ET.fromstring(z.read("xl/worksheets/sheet%d.xml" % idx))
        rows = []
        for row in ws.findall(".//m:row", ns):
            cells = []
            for c in row.findall("m:c", ns):
                t = c.get("t")
                v = c.find("m:v", ns)
                if t == "s" and v is not None:
                    cells.append(sst[int(v.text)])
                elif v is not None:
                    cells.append(v.text or "")
                else:
                    cells.append("")
            rows.append(cells)
        return rows

    guide = sheet_rows(names.index("说明") + 1)
    example_n = 4
    for cells in guide:
        for i, text in enumerate(cells):
            if text == "示例说明" and i + 1 < len(cells):
                m = re.search(r"前(\d+)行", cells[i + 1])
                if m:
                    example_n = int(m.group(1))
    header_map = {
        "资源状态": "resourceStatus", "机房编码": "siteCode", "机房名称": "siteName",
        "OLT设备编号": "oltCode", "ODF编号": "odfCode", "ODF端口": "odfPort",
        "OCC编号": "occCode", "ODB编号": "odbCode", "OBD编号": "obdCode",
        "一级分光比": "split1Ratio", "一级分光端口": "split1Port",
        "SDB编号": "sdbCode", "SBD编号": "sbdCode",
        "二级分光比": "split2Ratio", "二级分光端口": "split2Port",
        "总分光比": "totalSplit", "光缆/纤芯编号": "fiberCode", "FR/TO标签": "frTo",
        "端口状态": "portStatus", "敷设方式": "layingMethod", "ROW状态": "rowStatus",
        "PECE状态": "peceStatus", "备注": "remark",
    }
    body = sheet_rows(names.index("ODN资源") + 1)
    headers = body[0]
    keys = [header_map.get(h) for h in headers]
    if len(headers) != 23 or any(k is None for k in keys):
        raise RuntimeError("模板列数/表头异常: %d 列" % len(headers))
    rows = []
    for n, cells in enumerate(body[1:], start=2):
        row = {"rowNo": n}
        for k, v in zip(keys, cells):
            row[k] = (v or "").strip()
        rows.append(row)
    return example_n, rows


def chain1(row_no):
    return {"rowNo": row_no, "resourceStatus": "规划", "siteCode": "SITE001",
            "siteName": "验收机房", "oltCode": "SITE001_OLT001",
            "odfCode": "", "odfPort": "ODF-A-01",
            "occCode": "OCC901", "odbCode": "ODB901", "obdCode": "OBD901",
            "split1Ratio": "1:8", "split1Port": "P01",
            "sdbCode": "SDB901", "sbdCode": "SBD901",
            "split2Ratio": "1:8", "split2Port": "P01", "totalSplit": "64",
            "fiberCode": "F01", "frTo": "", "portStatus": "可用",
            "layingMethod": "架空", "rowStatus": "待处理", "peceStatus": "待签署",
            "remark": "W3 验收"}


def main():
    global TOKEN
    tok = http("POST", "/auth/login", "", {"username": "admin", "password": "admin123"})
    TOKEN = (tok.get("data") or {}).get("token", "")
    if not TOKEN:
        print("FAIL: admin 登录")
        sys.exit(1)
    ok("admin 登录")
    pre = http("GET", "/odn/resource-chains/patrol", TOKEN)
    pre_chains = (pre.get("data") or {}).get("chains", 0)
    try:
        run(pre_chains)
    finally:
        cleanup(TOKEN, pre_chains)
    print("===== 结果: PASS=%d FAIL=%d =====" % (PASS, FAIL))
    sys.exit(1 if FAIL else 0)


def run(pre_chains):
    # 1. 模板文件真实导入:示例行按说明页约定过滤
    example_n, rows = xlsx_parse(TEMPLATE)
    r = http("POST", "/odn/resource-chains/import", TOKEN,
             {"exampleRowCount": example_n, "rows": rows})
    d = (r.get("data") or {}).get("result") or {}
    if r.get("code") == 0 and d.get("skipped") == example_n and d.get("imported") == 0 and d.get("failed") == 0:
        ok("模板导入: %d 条示例行全部过滤,零入库" % example_n)
    else:
        bad("模板导入示例行过滤", json.dumps(d, ensure_ascii=False)[:300])

    # 2. 真实记录导入:2 行有效 + 1 重复 + 3 违规(总分光比/层级/枚举)
    valid2 = {"rowNo": 902, "resourceStatus": "规划", "siteCode": "SITE001",
              "siteName": "验收机房", "oltCode": "SITE001_OLT001",
              "occCode": "OCC901", "odbCode": "ODB902", "obdCode": "OBD902",
              "split1Ratio": "1:4", "split1Port": "P01",
              "sdbCode": "SDB902", "sbdCode": "SBD902",
              "split2Ratio": "", "split2Port": "", "totalSplit": "",
              "fiberCode": "", "frTo": "", "portStatus": "", "layingMethod": "",
              "rowStatus": "", "peceStatus": "", "remark": ""}
    bad_total = chain1(904)
    bad_total["totalSplit"] = "32"
    bad_hier = {"rowNo": 905, "resourceStatus": "", "odbCode": "ODB903"}
    bad_enum = chain1(906)
    bad_enum["occCode"] = "OCC901"
    bad_enum["odbCode"] = "ODB901"
    bad_enum["obdCode"] = "OBD901"
    bad_enum["sdbCode"] = "SDB901"
    bad_enum["sbdCode"] = "SBD901"
    bad_enum["portStatus"] = "已烧毁"
    payload = [chain1(901), valid2, chain1(907), bad_total, bad_hier, bad_enum]
    r = http("POST", "/odn/resource-chains/import", TOKEN,
             {"exampleRowCount": 0, "rows": payload})
    d = (r.get("data") or {}).get("result") or {}
    rows_res = {x.get("rowNo"): x for x in d.get("rows") or []}
    if r.get("code") == 0 and d.get("imported") == 2 and d.get("duplicate") == 1 and d.get("failed") == 3:
        ok("六规则计数: 导入 2 / 重复 1 / 拒绝 3")
    else:
        bad("六规则计数", json.dumps(d, ensure_ascii=False)[:300])
    r904 = rows_res.get(904) or {}
    if "总分光比不一致" in (r904.get("reason") or ""):
        ok("总分光比不一致拒绝并报行号 904")
    else:
        bad("总分光比不一致拒绝", json.dumps(r904, ensure_ascii=False))
    if "层级断裂" in (rows_res.get(905, {}).get("reason") or ""):
        ok("层级断裂拒绝并报行号 905")
    else:
        bad("层级断裂拒绝", json.dumps(rows_res.get(905), ensure_ascii=False))
    if "端口状态非法" in (rows_res.get(906, {}).get("reason") or ""):
        ok("枚举非法拒绝并报行号 906")
    else:
        bad("枚举非法拒绝", json.dumps(rows_res.get(906), ensure_ascii=False))
    if (rows_res.get(907, {}).get("status")) == "duplicate":
        ok("重复行去重并报告(907)")
    else:
        bad("重复行去重", json.dumps(rows_res.get(907), ensure_ascii=False))

    # 3. 展开断言:先查库(权威表),再接口复核
    sql_cnt = ("SELECT count(*) FROM odn_device WHERE prv_code IS NULL AND lifecycle_status='PLANNED' "
               "AND code IN (" + ",".join("'" + c + "'" for c in CODES) + ")")
    cnt = psql(sql_cnt)
    if cnt == "9":
        ok("展开落库: 9 台导入域箱体设备且 PLANNED(规则④)")
    else:
        bad("展开落库设备数", "count=%s" % cnt)
    sql_parent = ("SELECT count(*) FROM odn_device child JOIN odn_device p ON child.parent_id=p.id "
                  "WHERE (child.code='ODB901' AND p.code='OCC901')"
                  " OR (child.code='OBD901' AND p.code='ODB901')"
                  " OR (child.code='SDB901' AND p.code='ODB901')"
                  " OR (child.code='SBD901' AND p.code='SDB901')"
                  " OR (child.code='ODB902' AND p.code='OCC901')")
    pc = psql(sql_parent)
    if pc == "5":
        ok("父链展开: ODB←OCC、OBD←ODB、SDB←ODB、SBD←SDB 五条边全部成立")
    else:
        bad("父链展开", "edges=%s" % pc)
    lst = http("GET", "/odn/resource-chains?limit=50", TOKEN)
    items = (lst.get("data") or {}).get("items") or []
    mine = [x for x in items if str(x.get("batchNo", "")).startswith(BATCH_PREFIX)]
    if len(mine) == 2:
        ok("链行清单: 批次内 2 行(接口复核)")
    else:
        bad("链行清单", "items=%d" % len(mine))
    dev = http("GET", "/odn/devices?kind=OBD", TOKEN)
    devs = (dev.get("data") or {}).get("items") or []
    codes = sorted(x.get("code") for x in devs if str(x.get("code")).startswith("OBD9"))
    if codes == ["OBD901", "OBD902"]:
        ok("设备接口复核: OBD901/OBD902 在册")
    else:
        bad("设备接口复核", json.dumps(codes))

    # 4. 巡检:链上引用资源必须存在,孤儿/断祖/半链归零
    p = http("GET", "/odn/resource-chains/patrol", TOKEN).get("data") or {}
    if p.get("orphanBoxCode") == 0 and p.get("brokenAncestor") == 0 and p.get("splitPortGap") == 0 and p.get("chains", 0) >= 2:
        ok("巡检: 孤儿 0 / 断祖 0 / 半链 0(链完整)")
    else:
        bad("巡检归零", json.dumps(p, ensure_ascii=False)[:300])

    # 5. 幂等去重:同两行重放 → 全部 duplicate
    r = http("POST", "/odn/resource-chains/import", TOKEN,
             {"exampleRowCount": 0, "rows": [chain1(911), valid2]})
    d = (r.get("data") or {}).get("result") or {}
    if d.get("duplicate") == 2 and d.get("imported") == 0:
        ok("重复导入幂等: 2 行全部去重")
    else:
        bad("重复导入幂等", json.dumps(d, ensure_ascii=False)[:200])


if __name__ == "__main__":
    main()
