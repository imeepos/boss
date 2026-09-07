# -*- coding: utf-8 -*-
# 行标准化:日期/带宽/月数/PON 三段/VLAN/port_code/legacy_path/归属缺省/needs_review 标记。
# 红线:缺字段留空不编造;30M 按 50M 归档(原始值入报告);非标账号白名单放行;日期按 Asia/Manila 自然日。
from datetime import date, timedelta

EXCEL_EPOCH = date(1899, 12, 30)

# 一线一地址节点(2026-09-07 裁定):单一父根 + 每行子节点,满足 uq_quad_links_address
# 每地址至多一条非空活跃链路,347 行共用单节点必撞。
ADDRESS_ROOT_PATH = 'legacy_import'
ADDRESS_ROOT_NAME = '存量开户导入占位地址'

DEFAULTS = {
    'legal_entity_id': 6,
    'legal_entity_name': '平台总公司',
    'region_id': 1,
    'region_name': '集团',
    'customer_id_type': '无',
    'address_root_path': ADDRESS_ROOT_PATH,
    'address_root_name': ADDRESS_ROOT_NAME,
}

STANDARD_ACCOUNT_PREFIXES = ('OWPAL', 'OWTAC')
SECOND_LINE_SUFFIX = '_02'


def parse_date(text):
    # yyyy/m/d 字符串或 Excel 序列天数(拆机 45279) -> ISO;空 -> None;其余格式显式报错。
    t = (text or '').strip()
    if not t:
        return None
    if t.isdigit():
        return (EXCEL_EPOCH + timedelta(days=int(t))).isoformat()
    parts = t.split('/')
    if len(parts) == 3 and all(p.isdigit() for p in parts):
        y, mo, d = [int(p) for p in parts]
        return date(y, mo, d).isoformat()
    raise ValueError('unsupported date: %r' % t)


def _int_or_none(text):
    t = (text or '').strip()
    if not t:
        return None
    return int(t)


def _parse_pon(text):
    # PON口 0/1/6 -> (0,1,6);空 -> None;非三段显式报错(数据画像核实全部三段)。
    t = (text or '').strip()
    if not t:
        return None
    parts = t.split('/')
    if len(parts) != 3 or not all(p.isdigit() for p in parts):
        raise ValueError('unsupported PON: %r' % t)
    return (int(parts[0]), int(parts[1]), int(parts[2]))


def _flag(rec, name):
    if name not in rec['needs_review']:
        rec['needs_review'].append(name)


def _build_one(raw):
    rec = {
        'row': raw['_row'],
        'account': raw.get('C', ''),
        'bandwidth_mbps': None,
        'bandwidth_original': None,
        'months': None,
        'olt': None,
        'pon_frame': None,
        'pon_slot': None,
        'pon_port': None,
        'pon_raw': None,
        'onu_no': None,
        'sn': None,
        'model': None,
        'svlan': None,
        'cvlan': None,
        'internet_cvlan': None,
        'tr069_cvlan': None,
        'legacy_path': None,
        'port_code': None,
        'opened_date': parse_date(raw.get('A', '')),
        'closed_date': parse_date(raw.get('S', '')),
        'remark': raw.get('T', ''),
        'ownership': dict(DEFAULTS),
        'needs_review': [],
    }
    account = rec['account']
    if not account.startswith(STANDARD_ACCOUNT_PREFIXES):
        _flag(rec, 'nonstandard_account_whitelisted')
    if account.endswith(SECOND_LINE_SUFFIX):
        _flag(rec, 'second_line_same_customer')
    if rec['closed_date']:
        _flag(rec, 'closed_line')
    bw = _int_or_none(raw.get('D', ''))
    rec['bandwidth_original'] = bw
    if bw is None:
        _flag(rec, 'bandwidth_missing')
    elif bw == 30:
        rec['bandwidth_mbps'] = 50
        _flag(rec, 'bandwidth_30_archived_as_50')
    else:
        rec['bandwidth_mbps'] = bw
    rec['months'] = _int_or_none(raw.get('E', ''))
    if rec['months'] is None:
        _flag(rec, 'months_missing')
    olt = raw.get('J', '') or None
    rec['olt'] = olt
    if not olt:
        _flag(rec, 'olt_missing')
    pon = _parse_pon(raw.get('K', ''))
    rec['pon_raw'] = raw.get('K', '') or None
    if pon is None:
        _flag(rec, 'pon_missing')
    else:
        rec['pon_frame'], rec['pon_slot'], rec['pon_port'] = pon
    onu = _int_or_none(raw.get('L', ''))
    rec['onu_no'] = onu
    if onu is None:
        _flag(rec, 'onu_missing')
    sn = raw.get('M', '') or None
    rec['sn'] = sn
    if not sn:
        _flag(rec, 'sn_missing')
    model = raw.get('R', '') or None
    rec['model'] = model
    if not model:
        _flag(rec, 'model_missing')
    vlan_vals = [_int_or_none(raw.get(k, '')) for k in ('N', 'O', 'P', 'Q')]
    rec['svlan'], rec['cvlan'], rec['internet_cvlan'], rec['tr069_cvlan'] = vlan_vals
    if rec['svlan'] is None:
        _flag(rec, 'vlan_quad_missing')
    present = [v for v in vlan_vals if v is not None]
    if present and len(present) < 4:
        _flag(rec, 'vlan_partial')
    if olt and pon is not None and onu is not None:
        rec['port_code'] = 'P-%s-%d/%d/%d-%d' % (olt, pon[0], pon[1], pon[2], onu)
    parts = []
    missing_seg = False
    for col, prefix in (('F', 'OCC'), ('G', 'ODB'), ('H', 'OBD'), ('I', 'P')):
        val = raw.get(col, '')
        if val:
            parts.append(prefix + val)
        else:
            missing_seg = True
    if parts:
        rec['legacy_path'] = '-'.join(parts)
    if missing_seg and parts:
        _flag(rec, 'legacy_path_partial')
    return rec


def build_records(rows):
    # phone 为确定性伪登录号段 0999000xxxx(行序 1 起四位零填充):customers.phone 是
    # App 登录名(uq_customers_app_login_phone 唯一),PENDING 哨兵第二行即撞 23505;
    # 唯一由构造保证,跨重跑确定(行序=xlsx 行序)。
    records = [_build_one(raw) for raw in rows]
    for idx, rec in enumerate(records):
        rec['phone'] = '0999000%04d' % (idx + 1)
        label = rec['account'].lower()
        if not label or not all(c.isascii() and (c.isalnum() or c == '_') for c in label):
            raise ValueError('account not ltree-label safe: %r' % rec['account'])
        rec['address_path'] = ADDRESS_ROOT_PATH + '.' + label
        rec['address_name'] = rec['account']
    return records