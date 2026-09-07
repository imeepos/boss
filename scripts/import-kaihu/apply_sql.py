# -*- coding: utf-8 -*-
# SQL 段:生成幂等 upsert 脚本(占位地址/OLT 资源/端口/LO 账号/四码绑定)经 ssh psql 执行,并对账复核。
# 前置:迁移 000202(ports VLAN 四列+legacy_path,lo_accounts.contract_months)已生效;API 段 customers/assets 已建档。
# 引号红线:SQL 经 stdin 传 psql(ssh 远程命令单引号包裹),字面量单引号由 _q 双写转义,禁止嵌套叠引号。
import os
import subprocess

from normalize import DEFAULTS

SQL_NAME = 'apply.sql'
RECON_ITEMS = ('resources_olt', 'ports', 'lo_accounts', 'lo_closed', 'quad_links', 'customers', 'customers_phone')


def _q(text):
    q = chr(39)
    return q + str(text).replace(q, q + q) + q


def _int_or_null(value):
    return 'NULL' if value is None else str(int(value))


def _offer_subq(rec):
    # 20/50 -> 导入新建档按名取;100/200 -> 既有档按带宽取最小 id;缺带宽行 -> 默认 50M 档(报告已列 needsReview)。
    mbps = rec['bandwidth_mbps'] or 50
    if mbps == 20:
        return '(SELECT id FROM product_offers WHERE name = %s ORDER BY id LIMIT 1)' % _q('存量宽带20M')
    if mbps == 50:
        return '(SELECT id FROM product_offers WHERE name = %s ORDER BY id LIMIT 1)' % _q('存量宽带50M')
    return '(SELECT id FROM product_offers WHERE bandwidth = %s ORDER BY id LIMIT 1)' % _q('%dM' % mbps)


def _sql_address():
    out = ['-- 占位地址(单一节点 needs_review=true,治理队列消化;自然键 path 幂等)']
    out.append('INSERT INTO addresses (path, level, name, needs_review)')
    out.append('VALUES (%s, 1, %s, true)' % (_q(DEFAULTS['address_path']), _q(DEFAULTS['address_name'])))
    out.append('ON CONFLICT (path) DO NOTHING;')
    return out


def _sql_resources(olts):
    out = ['-- OLT 资源登记(自然键 code 幂等;地址挂占位节点满足 NOT NULL)']
    for code in olts:
        out.append('INSERT INTO resources (legal_entity_id, code, name, type, address_id, status)')
        out.append('SELECT %d, %s, %s, %s,' % (DEFAULTS['legal_entity_id'], _q(code), _q(code), _q('OLT')))
        out.append('       (SELECT id FROM addresses WHERE path = %s), %s;' % (_q(DEFAULTS['address_path']), _q('ONLINE')))
    return out


def _sql_ports(records):
    out = ['-- 端口(自然键 port_code upsert;pon 三段+onu_no+VLAN 四列+legacy_path)']
    for rec in records:
        if not rec['port_code']:
            continue
        out.append('INSERT INTO ports (port_code, quad_code, resource_id, legal_entity_id, legal_entity_name,')
        out.append('  address_id, region_id, region_name, status, pon_frame, pon_slot, pon_port, onu_no,')
        out.append('  svlan, cvlan, internet_cvlan, tr069_cvlan, legacy_path)')
        out.append('VALUES (%s, %s,' % (_q(rec['port_code']), _q(rec['port_code'])))
        out.append('  (SELECT id FROM resources WHERE code = %s), %d, %s,' % (_q(rec['olt']), DEFAULTS['legal_entity_id'], _q(DEFAULTS['legal_entity_name'])))
        out.append('  (SELECT id FROM addresses WHERE path = %s), %d, %s, %s,' % (_q(DEFAULTS['address_path']), DEFAULTS['region_id'], _q(DEFAULTS['region_name']), _q('USED')))
        out.append('  %s, %s, %s, %s,' % (rec['pon_frame'], rec['pon_slot'], rec['pon_port'], rec['onu_no']))
        out.append('  %s, %s, %s, %s, %s)' % (_int_or_null(rec['svlan']), _int_or_null(rec['cvlan']), _int_or_null(rec['internet_cvlan']), _int_or_null(rec['tr069_cvlan']), _int_or_null(rec['legacy_path']) if rec['legacy_path'] is None else _q(rec['legacy_path'])))
        out.append('ON CONFLICT (port_code) DO UPDATE SET')
        out.append('  pon_frame = EXCLUDED.pon_frame, pon_slot = EXCLUDED.pon_slot, pon_port = EXCLUDED.pon_port, onu_no = EXCLUDED.onu_no,')
        out.append('  svlan = EXCLUDED.svlan, cvlan = EXCLUDED.cvlan, internet_cvlan = EXCLUDED.internet_cvlan, tr069_cvlan = EXCLUDED.tr069_cvlan,')
        out.append('  legacy_path = EXCLUDED.legacy_path;')
    return out


def _sql_phone_fix(records):
    # 收敛已落库的 PENDING 哨兵行(apply 实跑撞 uq_customers_app_login_phone 前落库的 id=443);
    # 伪号与 plan 行序一致,幂等:仅当仍为 PENDING 才改,第二遍 0 行受影响。
    out = ['-- customers.phone 收敛(伪登录号段 0999000xxxx;历史 PENDING 哨兵行修正)']
    for rec in records:
        out.append('UPDATE customers SET phone = %s WHERE name = %s AND phone = %s;' % (_q(rec['phone']), _q(rec['account']), _q('PENDING')))
    return out


def _sql_lo_accounts(records):
    out = ['-- LO 账号(自然键 loid upsert;customer 按 name=账号+法人解析;月数入 contract_months;拆机 CLOSED)']
    for rec in records:
        status = 'CLOSED' if rec['closed_date'] else 'ACTIVE'
        out.append('INSERT INTO lo_accounts (loid, customer_id, legal_entity_id, legal_entity_name, region_id, region_name,')
        out.append('  offer_id, qos_template_id, status, contract_months)')
        out.append('SELECT %s, c.id, %d, %s, %d, %s,' % (_q(rec['account']), DEFAULTS['legal_entity_id'], _q(DEFAULTS['legal_entity_name']), DEFAULTS['region_id'], _q(DEFAULTS['region_name'])))
        out.append('  %s, (SELECT id FROM qos_templates ORDER BY id LIMIT 1), %s, %s' % (_offer_subq(rec), _q(status), _int_or_null(rec['months'])))
        out.append('FROM (SELECT id FROM customers WHERE name = %s AND legal_entity_id = %d ORDER BY id LIMIT 1) c' % (_q(rec['account']), DEFAULTS['legal_entity_id']))
        out.append('ON CONFLICT (loid) DO UPDATE SET')
        out.append('  offer_id = EXCLUDED.offer_id, status = EXCLUDED.status, contract_months = EXCLUDED.contract_months;')
    return out


def _sql_quad_links(records):
    out = ['-- 四码绑定(端口唯一槽位:同 port 已有 LINKED/CONFLICT 则跳过,幂等;asset 按 loid 解析)']
    for rec in records:
        if not rec['port_code']:
            continue
        out.append('INSERT INTO quad_links (asset_id, customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status)')
        out.append('SELECT a.id, c.id, p.id, (SELECT id FROM addresses WHERE path = %s), %d, %s, %s' % (_q(DEFAULTS['address_path']), DEFAULTS['legal_entity_id'], _q(DEFAULTS['legal_entity_name']), _q('LINKED')))
        out.append('FROM (SELECT id FROM assets WHERE loid = %s ORDER BY id LIMIT 1) a,' % _q(rec['account']))
        out.append('     (SELECT id FROM customers WHERE name = %s AND legal_entity_id = %d ORDER BY id LIMIT 1) c,' % (_q(rec['account']), DEFAULTS['legal_entity_id']))
        out.append('     (SELECT id FROM ports WHERE port_code = %s) p' % _q(rec['port_code']))
        out.append("WHERE NOT EXISTS (SELECT 1 FROM quad_links q WHERE q.port_id = p.id AND q.status IN ('LINKED', 'CONFLICT'));")
    return out


def _in_list(values):
    return ', '.join(_q(v) for v in values)


def _sql_verify(records):
    pcs = sorted(set(r['port_code'] for r in records if r['port_code']))
    loids = sorted(set(r['account'] for r in records))
    olts = sorted(set(r['olt'] for r in records if r['olt']))
    out = ['-- 对账复核(工具解析 CSV 输出,任一不符即整体失败)']
    out.append("SELECT 'resources_olt' AS item, count(*) AS n FROM resources WHERE code IN (%s);" % _in_list(olts))
    out.append("SELECT 'ports' AS item, count(*) AS n FROM ports WHERE port_code IN (%s);" % _in_list(pcs))
    out.append("SELECT 'lo_accounts' AS item, count(*) AS n FROM lo_accounts WHERE loid IN (%s);" % _in_list(loids))
    out.append("SELECT 'lo_closed' AS item, count(*) AS n FROM lo_accounts WHERE status = 'CLOSED' AND loid IN (%s);" % _in_list(loids))
    out.append("SELECT 'quad_links' AS item, count(*) AS n FROM quad_links WHERE status = 'LINKED' AND port_id IN (SELECT id FROM ports WHERE port_code IN (%s));" % _in_list(pcs))
    out.append("SELECT 'customers' AS item, count(*) AS n FROM customers WHERE name IN (%s);" % _in_list(loids))
    out.append("SELECT 'customers_phone' AS item, count(*) AS n FROM customers WHERE phone LIKE '0999000%%' AND name IN (%s);" % _in_list(loids))
    return out


def build_sql(records):
    lines = ['-- 存量开户记录导入 SQL 段(幂等 upsert;T4 执行;前置见文件头注释)']
    lines.append('BEGIN;')
    lines.extend(_sql_address())
    lines.extend(_sql_resources(sorted(set(r['olt'] for r in records if r['olt']))))
    lines.extend(_sql_ports(records))
    lines.extend(_sql_phone_fix(records))
    lines.extend(_sql_lo_accounts(records))
    lines.extend(_sql_quad_links(records))
    lines.extend(_sql_verify(records))
    lines.append('COMMIT;')
    return chr(10).join(lines) + chr(10)


def _expected(records):
    pcs = set(r['port_code'] for r in records if r['port_code'])
    return {
        'resources_olt': len(set(r['olt'] for r in records if r['olt'])),
        'ports': len(pcs),
        'lo_accounts': len(records),
        'lo_closed': sum(1 for r in records if r['closed_date']),
        'quad_links': len(pcs),
        'customers': len(records),
        'customers_phone': len(records),
    }


def _parse_verify(stdout):
    got = {}
    for line in stdout.splitlines():
        parts = line.split(',')
        if len(parts) == 2 and parts[0] in RECON_ITEMS:
            got[parts[0]] = int(parts[1])
    return got


def run(args, records):
    sql = build_sql(records)
    os.makedirs(args.outdir, exist_ok=True)
    path = os.path.join(args.outdir, SQL_NAME)
    with open(path, 'w', encoding='utf-8') as fh:
        fh.write(sql)
    remote = 'docker exec -i %s psql -U %s -d %s -v ON_ERROR_STOP=1 --csv' % (args.ssh_container, args.db_user, args.db_name)
    print('[import-kaihu] sql: script=%s rows=%d via ssh %s' % (path, len(records), args.ssh_host))
    with open(path, 'r', encoding='utf-8') as fh:
        proc = subprocess.run(['ssh', args.ssh_host, remote], stdin=fh, capture_output=True, text=True)
    if proc.returncode != 0:
        print('[import-kaihu] SQL APPLY FAILED rc=%d stderr_tail=%s' % (proc.returncode, proc.stderr[-800:]))
        return None
    expected = _expected(records)
    got = _parse_verify(proc.stdout)
    if got != expected:
        print('[import-kaihu] RECONCILIATION FAILED expected=%s got=%s' % (sorted(expected.items()), sorted(got.items())))
        return {'reconciliation': {'expected': expected, 'got': got}, 'ok': False}
    print('[import-kaihu] sql: reconciliation PASS %s' % sorted(got.items()))
    return {'reconciliation': {'expected': expected, 'got': got}, 'ok': True, 'sql_file': path}