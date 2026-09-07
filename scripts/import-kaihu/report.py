# -*- coding: utf-8 -*-
# 校验报告:硬指标计数 + needsReview 明细 + 白名单 + 30M 归档 + VLAN 离群(按 OLT+PON板卡 分组众数偏离)。
# 硬指标口径:VLAN 四元组缺失按外层 svlan 缺失计(=35,与 T4 基线 svlan 非空 312 对齐)。
from collections import Counter

HARD_METRICS = {
    'rows_valid': 347,
    'accounts_unique': 347,
    'sn_nonempty': 330,
    'sn_unique': 330,
    'missing_pon': 16,
    'missing_sn': 17,
    'vlan_quad_missing': 35,
    'closed_lines': 1,
}

CLOSED_EXPECTED = {'account': 'OWPAL51531', 'date': '2023-12-19'}
SECOND_LINE_EXPECTED = 'OWPAL19516_02'
WHITELIST_EXPECTED = ('POWC_PALYYT_001', 'OWC_Office_GF', 'Office301')


def _mode(values):
    # 众数;并列取较小值,保证确定性输出。
    counts = Counter(values)
    best = max(counts.values())
    return min(v for v, n in counts.items() if n == best)


def vlan_outliers(records):
    groups = {}
    for rec in records:
        if rec['olt'] and rec['pon_frame'] is not None and rec['svlan'] is not None:
            key = '%s-%d/%d' % (rec['olt'], rec['pon_frame'], rec['pon_slot'])
            groups.setdefault(key, []).append(rec)
    outliers = []
    for key in sorted(groups):
        recs = groups[key]
        mode = _mode([r['svlan'] for r in recs])
        for rec in recs:
            if rec['svlan'] != mode:
                outliers.append({
                    'row': rec['row'],
                    'account': rec['account'],
                    'group': key,
                    'svlan': rec['svlan'],
                    'group_mode': mode,
                })
    return outliers


def _dist(values):
    pairs = sorted(Counter(str(v) for v in values).items())
    return dict(pairs)


def build_report(records, skipped_empty, sheet_names, source_name):
    accounts = [r['account'] for r in records]
    sns = [r['sn'] for r in records if r['sn']]
    closed = [r for r in records if r['closed_date']]
    counts = {
        'rows_valid': len(records),
        'rows_skipped_empty': skipped_empty,
        'accounts_unique': len(set(accounts)),
        'sn_nonempty': len(sns),
        'sn_unique': len(set(sns)),
        'missing_pon': sum(1 for r in records if r['pon_frame'] is None),
        'missing_sn': sum(1 for r in records if not r['sn']),
        'vlan_quad_missing': sum(1 for r in records if r['svlan'] is None),
        'closed_lines': len(closed),
        'missing_olt': sum(1 for r in records if not r['olt']),
        'missing_onu': sum(1 for r in records if r['onu_no'] is None),
        'missing_model': sum(1 for r in records if not r['model']),
        'missing_months': sum(1 for r in records if r['months'] is None),
        'missing_bandwidth': sum(1 for r in records if r['bandwidth_mbps'] is None),
        'vlan_partial': sum(1 for r in records if 'vlan_partial' in r['needs_review']),
        'ports_planned': sum(1 for r in records if r['port_code']),
        'assets_planned': len(records),
        'customers_planned': len(records),
        'phone_placeholder_rows': len(records),
        'lo_accounts_planned': len(records),
        'quad_links_planned': sum(1 for r in records if r['port_code']),
    }
    needs = []
    for rec in records:
        if rec['needs_review']:
            needs.append({
                'row': rec['row'],
                'account': rec['account'],
                'reasons': sorted(rec['needs_review']),
                'remark': rec['remark'],
            })
    whitelist = [
        {'row': r['row'], 'account': r['account'], 'reason': r['remark'] or '非标账号白名单放行(设计文档 §5 裁定)'}
        for r in records if 'nonstandard_account_whitelisted' in r['needs_review']
    ]
    adjustments = [
        {'row': r['row'], 'account': r['account'], 'original_mbps': r['bandwidth_original'], 'archived_mbps': r['bandwidth_mbps']}
        for r in records if 'bandwidth_30_archived_as_50' in r['needs_review']
    ]
    second = [r for r in records if 'second_line_same_customer' in r['needs_review']]
    return {
        'tool': 'import-kaihu',
        'source': {'file': source_name, 'sheets': sheet_names, 'rows_skipped_empty': skipped_empty},
        'counts': counts,
        'bandwidth_distribution': _dist([r['bandwidth_original'] for r in records]),
        'months_distribution': _dist([r['months'] for r in records]),
        'needs_review': needs,
        'whitelist': whitelist,
        'bandwidth_adjustments': adjustments,
        'second_line_accounts': [
            {'row': r['row'], 'account': r['account'], 'note': '同客户二次线路:独立 lo_account,quad_link 1:N'}
            for r in second
        ],
        'closed_lines': [
            {'row': r['row'], 'account': r['account'], 'closed_date': r['closed_date'], 'note': '拆机以 lo_accounts.status=CLOSED 表达'}
            for r in closed
        ],
        'vlan_outliers': vlan_outliers(records),
        'assumptions': [
            '产品月费为导入假设值: 20M 默认 49.00, 50M 默认 69.00(--fee-20m/--fee-50m 可调), T4 前负责人复核',
            '缺带宽行 offer 落默认 50M 档; 30M 行按 50M 归档(原始值见 bandwidth_adjustments)',
            'customers phone=PENDING 哨兵, regionId=1(集团)代理不限, 归属法人=平台总公司(id=6)',
            '导入产品档挂法人 1(与家庭宽带100M/200M 同法人); 100M/200M 复用既有档按带宽取最小 id',
            'VLAN 四元组缺失口径=外层 svlan 缺失(=35); 全四空 34 行, vlan_partial 单列',
            '端口/四码绑定仅为 OLT+PON+ONU 三要素齐备行建(实测 313), 与预估 ~331/+346 的差异见 README §6',
            'customers.phone 为伪登录号段 0999000xxxx(行序 1 起四位零填充, 唯一由构造保证, 对齐 uq_customers_app_login_phone); 档案占位, 真实号注册撞号时走 409 人工处理',
        ],
    }


def selfcheck(rep):
    # 硬指标逐项断言 + 设计文档点名的三个事实;返回失败清单(空=通过)。
    failures = []
    counts = rep['counts']
    for key in sorted(HARD_METRICS):
        expected = HARD_METRICS[key]
        if counts.get(key) != expected:
            failures.append('metric %s expected %s got %s' % (key, expected, counts.get(key)))
    closed = rep['closed_lines']
    if len(closed) == 1:
        row = closed[0]
        if row['account'] != CLOSED_EXPECTED['account'] or row['closed_date'] != CLOSED_EXPECTED['date']:
            failures.append('closed line mismatch: %s' % row)
    second = [e['account'] for e in rep['second_line_accounts']]
    if second != [SECOND_LINE_EXPECTED]:
        failures.append('second line accounts expected [%s] got %s' % (SECOND_LINE_EXPECTED, second))
    wl = sorted(e['account'] for e in rep['whitelist'])
    if wl != sorted(WHITELIST_EXPECTED):
        failures.append('whitelist expected %s got %s' % (sorted(WHITELIST_EXPECTED), wl))
    return failures