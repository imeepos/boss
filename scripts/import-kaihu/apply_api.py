# -*- coding: utf-8 -*-
# API 段:型号/批次/产品 查重建档 + 客户/资产逐行建档(自然键幂等,40900 冲突=已存在) + import-tasks 登记。
# 前置:BOSS_ADMIN_KEY 对应账号需持有 menu:asset/menu:customer/menu:importer 权限,数据范围需覆盖平台总公司(建议全集团)。
from api_client import ApiError, AdminClient, items_of
from normalize import DEFAULTS

MODEL_VENDOR = '(存量未登记)'
MODEL_CATEGORY = 'ONU'
ASSET_TYPE = 'ONU'
BATCH_NAME = '存量开户导入'
PRODUCT_ENTITY_ID = 1
PRODUCT_PREFIX = '存量宽带'
IMPORT_KIND = 'entity:kaihu_legacy'
CODE_CONFLICT = 40900


def _norm(value):
    return (value or '').strip() if isinstance(value, str) else value


def ensure_asset_models(client, records):
    # 返回 model->id 映射;已有按 model 名幂等跳过,新建取 POST 返回 id。
    wanted = sorted(set(r['model'] for r in records if r['model']))
    items = items_of(client.request('GET', '/asset-models'))
    id_map = {}
    for it in items:
        if _norm(it.get('model')):
            id_map[_norm(it.get('model'))] = it.get('id')
    for model in wanted:
        if model in id_map:
            continue
        data = client.request('POST', '/asset-models', {'vendor': MODEL_VENDOR, 'model': model, 'category': MODEL_CATEGORY})
        id_map[model] = data.get('id')
    return id_map


def ensure_batch(client):
    # 建一个导入专用入库批次(资产建档 batchId 必填);按名称幂等。
    # POST 撞重名 409(并行/残留)时重新按名 list 取 id,幂等语义以服务端唯一约束为权威。
    items = items_of(client.request('GET', '/assets/batches'))
    for it in items:
        if _norm(it.get('name')) == BATCH_NAME:
            return it.get('id')
    try:
        data = client.request('POST', '/asset-batches', {'name': BATCH_NAME, 'legalEntityId': DEFAULTS['legal_entity_id']})
        return data.get('id')
    except ApiError as exc:
        if exc.code != CODE_CONFLICT:
            raise
        items = items_of(client.request('GET', '/assets/batches'))
        for it in items:
            if _norm(it.get('name')) == BATCH_NAME:
                return it.get('id')
        raise ApiError('GET', '/assets/batches', 'missing', 'batch conflict but name not found after re-list')


def ensure_products(client, records, fee_20m, fee_50m):
    # 20M/50M 两档新建(挂家庭宽带产品同法人,名字带 存量 前缀标记来源);100M/200M 用既有档,缺失显式报错。
    result = {}
    items = items_of(client.request('GET', '/products', params={'legalEntityId': PRODUCT_ENTITY_ID}))
    have = {}
    for it in items:
        if _norm(it.get('name')):
            have[_norm(it.get('name'))] = it.get('id')
    fees = {20: fee_20m, 50: fee_50m}
    for mbps in sorted(set(r['bandwidth_mbps'] for r in records if r['bandwidth_mbps'] in (20, 50))):
        name = '%s%dM' % (PRODUCT_PREFIX, mbps)
        if name in have:
            result[name] = 'existing'
            continue
        payload = {'name': name, 'bandwidth': '%dM' % mbps, 'monthlyFee': fees[mbps],
                   'legalEntityId': PRODUCT_ENTITY_ID, 'category': 'broadband'}
        client.request('POST', '/products', payload)
        result[name] = 'created'
    all_items = items_of(client.request('GET', '/products'))
    have_bw = set(_norm(it.get('bandwidth')) for it in all_items)
    for mbps in sorted(set(r['bandwidth_mbps'] for r in records if r['bandwidth_mbps'] is not None and r['bandwidth_mbps'] not in (20, 50))):
        tag = '%dM' % mbps
        if tag not in have_bw:
            raise ApiError('GET', '/products', 'missing', 'no existing product with bandwidth ' + tag)
        result['existing-' + tag] = 'verified'
    return result


def _customer_exists(client, account):
    data = client.request('GET', '/customers', params={'keyword': account, 'limit': 50})
    for it in items_of(data):
        if _norm(it.get('name')) == account:
            return True
    return False


def ensure_customers(client, records):
    # 客户直建:name=宽带账号(可追溯),phone=PENDING 哨兵,idType=无,realName 服务端默认 PENDING;按 name 幂等。
    created = 0
    skipped = 0
    for rec in records:
        account = rec['account']
        if _customer_exists(client, account):
            skipped += 1
            continue
        phone = rec.get('phone', '')
        if not phone:
            raise ApiError('GET', '/customers', 'missing', 'plan row lacks phone: stale plan, re-run dry-run')
        payload = {
            'name': account,
            'phone': phone,
            'legalEntityId': DEFAULTS['legal_entity_id'],
            'regionId': DEFAULTS['region_id'],
            'idType': DEFAULTS['customer_id_type'],
        }
        try:
            client.request('POST', '/customers', payload)
        except ApiError as exc:
            if exc.code != CODE_CONFLICT:
                print('[import-kaihu] API FAILED customer row=%d account=%s %s' % (rec['row'], account, exc))
                raise
            # 40900=唯一约束(phone/name 残留)已存在,幂等权威在服务端,计 skipped 不中断。
            skipped += 1
            continue
        created += 1
    return created, skipped


def create_assets(client, records, batch_id, model_ids):
    # 资产逐行建档:loid=账号(000188 唯一索引兜底),SN 全网唯一;冲突 40900 = 已存在跳过。
    created = 0
    skipped = 0
    for rec in records:
        if rec['model'] and rec['model'] not in model_ids:
            raise ApiError('GET', '/asset-models', 'missing', 'model not registered: ' + rec['model'])
        payload = {
            'batchId': batch_id,
            'modelId': model_ids.get(rec['model'], 0) if rec['model'] else 0,
            'type': ASSET_TYPE,
            'sn': rec['sn'] or '',
            'loid': rec['account'],
        }
        try:
            client.request('POST', '/assets', payload)
            created += 1
        except ApiError as exc:
            if exc.code == CODE_CONFLICT:
                skipped += 1
                continue
            print('[import-kaihu] API FAILED asset row=%d account=%s %s' % (rec['row'], rec['account'], exc))
            raise
    return created, skipped


def register_import_task(client, records, imported, skipped, args):
    # clientKey 全局幂等:重复登记覆盖统计数(2026-09-03 裁定)。
    detail = {'source': 'legacy-kaihu-20240106', 'mode': 'apply', 'segments': ['asset-models', 'asset-batches', 'assets', 'products', 'customers', 'sql']}
    payload = {
        'kind': IMPORT_KIND,
        'total': len(records),
        'imported': imported,
        'failed': 0,
        'skipped': skipped,
        'detail': detail,
        'clientKey': args.client_key,
    }
    client.request('POST', '/import-tasks', payload)


def run(client, records, args):
    result = {}
    id_map = ensure_asset_models(client, records)
    result['models'] = {k: 'id:%s' % v for k, v in sorted(id_map.items())}
    print('[import-kaihu] api: asset-models ready %s' % sorted(id_map))
    batch_id = ensure_batch(client)
    result['batch_id'] = batch_id
    print('[import-kaihu] api: batch id=%s' % batch_id)
    result['products'] = ensure_products(client, records, args.fee_20m, args.fee_50m)
    print('[import-kaihu] api: products %s' % sorted(result['products'].items()))
    cust_created, cust_skipped = ensure_customers(client, records)
    result['customers'] = {'created': cust_created, 'skipped': cust_skipped}
    print('[import-kaihu] api: customers created=%d skipped=%d' % (cust_created, cust_skipped))
    asset_created, asset_skipped = create_assets(client, records, batch_id, id_map)
    result['assets'] = {'created': asset_created, 'skipped_conflict': asset_skipped}
    print('[import-kaihu] api: assets created=%d skipped_conflict=%d' % (asset_created, asset_skipped))
    register_import_task(client, records, len(records) - cust_skipped, cust_skipped, args)
    result['import_task'] = {'kind': IMPORT_KIND, 'clientKey': args.client_key}
    print('[import-kaihu] api: import-tasks registered clientKey=%s' % args.client_key)
    return result