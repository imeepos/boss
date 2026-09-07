# -*- coding: utf-8 -*-
# 开户记录 xlsx 导入工具(CLI)。
# dry-run(默认): --source <xlsx> --outdir <dir> [--selfcheck]  -> 产出 plan.json + report.json
# apply(负责人 T4): --apply --plan-file <plan.json> [--outdir <dir>]
#   环境变量 BOSS_ADMIN_KEY 必填;API 段 http://192.168.0.102:28080/api/admin/v1;SQL 段 ssh 102 psql。
import argparse
import hashlib
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import apply_api
import apply_sql
import normalize
import report as report_mod
import xlsx_reader
from api_client import AdminClient

DEFAULT_API_BASE = 'http://192.168.0.102:28080/api/admin/v1'
DEFAULT_OUTDIR = '/tmp/kaihu-dry'
PLAN_NAME = 'plan.json'
REPORT_NAME = 'report.json'
RESULT_NAME = 'apply-result.json'


def _dump_json(path, obj):
    with open(path, 'w', encoding='utf-8') as fh:
        json.dump(obj, fh, ensure_ascii=False, indent=2, sort_keys=True)
        fh.write(chr(10))


def _build_plan(records, source_path, sheet_names):
    with open(source_path, 'rb') as fh:
        digest = hashlib.sha256(fh.read()).hexdigest()
    return {
        'meta': {
            'tool': 'import-kaihu',
            'plan_version': 1,
            'source_file': os.path.basename(source_path),
            'source_sha256': digest,
            'sheet': sheet_names[0] if sheet_names else '',
            'rows': len(records),
        },
        'defaults': normalize.DEFAULTS,
        'rows': records,
    }


def _cmd_dry_run(args):
    names, header, rows, skipped = xlsx_reader.read_rows(args.source)
    records = normalize.build_records(rows)
    rep = report_mod.build_report(records, skipped, names, os.path.basename(args.source))
    plan = _build_plan(records, args.source, names)
    os.makedirs(args.outdir, exist_ok=True)
    plan_path = os.path.join(args.outdir, PLAN_NAME)
    report_path = os.path.join(args.outdir, REPORT_NAME)
    _dump_json(plan_path, plan)
    _dump_json(report_path, rep)
    counts = rep['counts']
    print('[import-kaihu] source=%s sheet=%s rows=%d skipped_empty=%d' % (os.path.basename(args.source), names[0] if names else '', counts['rows_valid'], skipped))
    print('[import-kaihu] plan=%s report=%s' % (plan_path, report_path))
    print('[import-kaihu] counts: rows_valid=%(rows_valid)d accounts_unique=%(accounts_unique)d sn_nonempty=%(sn_nonempty)d missing_pon=%(missing_pon)d missing_sn=%(missing_sn)d vlan_quad_missing=%(vlan_quad_missing)d closed_lines=%(closed_lines)d' % counts)
    print('[import-kaihu] planned: ports=%(ports_planned)d customers=%(customers_planned)d lo_accounts=%(lo_accounts_planned)d quad_links=%(quad_links_planned)d' % counts)
    if args.selfcheck:
        failures = report_mod.selfcheck(rep)
        if failures:
            for line in failures:
                print('[import-kaihu] SELFCHECK FAILED %s' % line)
            return 1
        print('[import-kaihu] selfcheck: PASS (%d metrics)' % len(report_mod.HARD_METRICS))
    return 0


def _cmd_apply(args):
    if not args.plan_file:
        print('[import-kaihu] APPLY FAILED --plan-file required')
        return 2
    api_key = os.environ.get('BOSS_ADMIN_KEY', '')
    if not api_key:
        print('[import-kaihu] APPLY FAILED env BOSS_ADMIN_KEY not set')
        return 2
    with open(args.plan_file, 'r', encoding='utf-8') as fh:
        plan = json.load(fh)
    records = plan.get('rows', [])
    if not records:
        print('[import-kaihu] APPLY FAILED plan has no rows')
        return 2
    client = AdminClient(args.api_base, api_key)
    api_result = apply_api.run(client, records, args)
    sql_result = apply_sql.run(args, records)
    if sql_result is None or not sql_result.get('ok'):
        print('[import-kaihu] APPLY FAILED at SQL segment (see above)')
        return 1
    result = {'api': api_result, 'sql': sql_result}
    os.makedirs(args.outdir, exist_ok=True)
    out_path = os.path.join(args.outdir, RESULT_NAME)
    _dump_json(out_path, result)
    print('[import-kaihu] apply complete, result=%s' % out_path)
    return 0


def _parse_args(argv):
    parser = argparse.ArgumentParser(description='legacy kaihu xlsx import tool')
    parser.add_argument('--dry-run', dest='dry_run', action='store_true', default=True)
    parser.add_argument('--apply', action='store_true')
    parser.add_argument('--selfcheck', action='store_true')
    parser.add_argument('--source', default='')
    parser.add_argument('--outdir', default=DEFAULT_OUTDIR)
    parser.add_argument('--plan-file', default='')
    parser.add_argument('--api-base', default=DEFAULT_API_BASE)
    parser.add_argument('--fee-20m', dest='fee_20m', type=float, default=49.0)
    parser.add_argument('--fee-50m', dest='fee_50m', type=float, default=69.0)
    parser.add_argument('--client-key', dest='client_key', default='kaihu-legacy-20240106')
    parser.add_argument('--ssh-host', dest='ssh_host', default='imeepos@192.168.0.102')
    parser.add_argument('--ssh-container', dest='ssh_container', default='boss-infra-postgres-1')
    parser.add_argument('--db-user', dest='db_user', default='boss')
    parser.add_argument('--db-name', dest='db_name', default='boss')
    return parser.parse_args(argv)


def main(argv):
    args = _parse_args(argv)
    if args.apply:
        return _cmd_apply(args)
    if not args.source:
        print('[import-kaihu] dry-run requires --source')
        return 2
    return _cmd_dry_run(args)


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))