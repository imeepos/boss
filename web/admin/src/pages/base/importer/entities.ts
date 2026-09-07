// 业务数据批量导入实体定义(审计结论:仅收录"主档型 + 已有 admin POST 创建端点"的项目;
// 逐行调用既有创建端点,后端零改动,对接 102 部署环境)。
// 契约出处:internal/httpapi/admin/org.go(账号/法人/部门/岗位)、customer.go(产品)。
export type ColumnType = 'string' | 'number' | 'list'

export interface EntityColumn {
  key: string
  required: boolean
  type: ColumnType
}

export interface EntityDef {
  /** i18n key(pages.importer.entityNames.<kind>)与前端选择器值。 */
  kind: string
  /** 既有 admin 创建端点(逐行 POST)。 */
  endpoint: string
  /** 创建端点所需菜单权限码(权限前置:无权限置灰禁止导入)。 */
  perm: string
  /** 走 query string 而非 body 的列(如 ODN 的 prvCode/cityPrefix;仍须登记在 columns 中)。 */
  queryColumns?: string[]
  /** 自然唯一键(导入前去重:文件内先到先得 + 与现有数据比对);缺省=不去重。 */
  uniqueKey?: string[]
  /** 现有数据清单端点(去重数据源;返回数组或 {items} 数组;缺省=仅文件内去重)。 */
  listEndpoint?: string
  columns: EntityColumn[]
  /** 模板样例行(与 columns 同序的字段集)。 */
  samples: Array<Record<string, string | number | string[]>>
}

export const IMPORT_ENTITIES: EntityDef[] = [
  {
    kind: 'account',
    perm: 'menu:account',
    endpoint: '/accounts',
    uniqueKey: ['username'],
    listEndpoint: '/accounts',
    columns: [
      { key: 'username', required: true, type: 'string' },
      { key: 'password', required: true, type: 'string' },
      { key: 'realName', required: true, type: 'string' },
      { key: 'roleCode', required: true, type: 'string' },
      { key: 'phone', required: false, type: 'string' },
      { key: 'legalEntityId', required: false, type: 'number' },
      { key: 'deptId', required: false, type: 'number' },
      { key: 'postId', required: false, type: 'number' },
      { key: 'regionScope', required: false, type: 'string' },
    ],
    samples: [
      { username: 'op102', password: 'Init@1024', realName: '运营一号', roleCode: 'ops', phone: '09170000001', legalEntityId: 1, deptId: 1, postId: 1, regionScope: '' },
    ],
  },
  {
    kind: 'legal_entity',
    perm: 'menu:company',
    endpoint: '/legal-entities',
    uniqueKey: ['code'],
    listEndpoint: '/legal-entities',
    columns: [
      { key: 'code', required: true, type: 'string' },
      { key: 'name', required: true, type: 'string' },
      { key: 'taxJurisdiction', required: false, type: 'string' },
      { key: 'taxChannel', required: false, type: 'string' },
    ],
    samples: [
      { code: 'LE-SUB-01', name: '示例子公司', taxJurisdiction: 'CN', taxChannel: 'manual' },
    ],
  },
  {
    kind: 'department',
    perm: 'menu:department',
    endpoint: '/departments',
    uniqueKey: ['legalEntityId', 'name'],
    listEndpoint: '/departments',
    columns: [
      { key: 'legalEntityId', required: true, type: 'number' },
      { key: 'name', required: true, type: 'string' },
    ],
    samples: [
      { legalEntityId: 1, name: '示例部门' },
    ],
  },
  {
    kind: 'post',
    perm: 'menu:post',
    endpoint: '/posts',
    uniqueKey: ['deptId', 'code'],
    listEndpoint: '/posts',
    columns: [
      { key: 'deptId', required: true, type: 'number' },
      { key: 'code', required: true, type: 'string' },
      { key: 'name', required: true, type: 'string' },
      { key: 'roles', required: false, type: 'list' },
    ],
    samples: [
      { deptId: 1, code: 'post_imp_01', name: '示例岗位', roles: 'ops,analyst' },
    ],
  },
  {
    kind: 'product',
    perm: 'menu:product',
    endpoint: '/products',
    uniqueKey: ['legalEntityId', 'name'],
    listEndpoint: '/products',
    columns: [
      { key: 'legalEntityId', required: true, type: 'number' },
      { key: 'name', required: true, type: 'string' },
      { key: 'monthlyFee', required: true, type: 'number' },
      { key: 'bandwidth', required: false, type: 'string' },
      { key: 'category', required: false, type: 'string' },
      { key: 'status', required: false, type: 'string' },
    ],
    samples: [
      { legalEntityId: 1, name: '示例套餐 100M', monthlyFee: 99, bandwidth: '100M', category: 'broadband', status: 'DRAFT' },
    ],
  },
  {
    kind: 'odn_site',
    perm: 'menu:odn',
    endpoint: '/odn/sites',
    uniqueKey: ['prvCode', 'cityPrefix', 'siteNo'],
    listEndpoint: '/odn/sites',
    queryColumns: ['prvCode', 'cityPrefix'],
    columns: [
      { key: 'prvCode', required: true, type: 'string' },
      { key: 'cityPrefix', required: true, type: 'string' },
      { key: 'siteNo', required: true, type: 'number' },
      { key: 'name', required: false, type: 'string' },
      { key: 'lat', required: false, type: 'number' },
      { key: 'lng', required: false, type: 'number' },
    ],
    samples: [
      { prvCode: 'PHL001', cityPrefix: 'MNL', siteNo: 88, name: '示例局点', lat: 14.6, lng: 121.0 },
    ],
  },
  {
    kind: 'odn_grid',
    perm: 'menu:odn',
    endpoint: '/odn/grids',
    uniqueKey: ['prvCode', 'cityPrefix', 'gridCode'],
    listEndpoint: '/odn/grids',
    queryColumns: ['prvCode', 'cityPrefix'],
    columns: [
      { key: 'prvCode', required: true, type: 'string' },
      { key: 'cityPrefix', required: true, type: 'string' },
      { key: 'gridCode', required: true, type: 'number' },
      { key: 'name', required: false, type: 'string' },
      { key: 'coverage', required: false, type: 'string' },
      { key: 'status', required: true, type: 'string' },
    ],
    samples: [
      { prvCode: 'PHL001', cityPrefix: 'MNL', gridCode: 12, name: '示例网格', coverage: 'Makati', status: 'ACTIVE' },
    ],
  },
  {
    kind: 'customer',
    perm: 'menu:customer',
    endpoint: '/customers',
    uniqueKey: ['phone'],
    listEndpoint: '/customers',
    columns: [
      { key: 'name', required: true, type: 'string' },
      { key: 'phone', required: true, type: 'string' },
      { key: 'legalEntityId', required: true, type: 'number' },
      { key: 'addressId', required: true, type: 'number' },
      { key: 'regionId', required: true, type: 'number' },
      { key: 'idType', required: false, type: 'string' },
      { key: 'idNo', required: false, type: 'string' },
    ],
    samples: [
      { name: '示例客户', phone: '09170000001', legalEntityId: 1, addressId: 1, regionId: 1, idType: '身份证', idNo: '' },
    ],
  },
  {
    kind: 'odn_device',
    perm: 'menu:odn',
    endpoint: '/odn/devices',
    uniqueKey: ['prvCode', 'cityPrefix', 'code'],
    listEndpoint: '/odn/devices',
    queryColumns: ['prvCode', 'cityPrefix'],
    columns: [
      { key: 'code', required: true, type: 'string' },
      { key: 'kind', required: true, type: 'string' },
      { key: 'prvCode', required: true, type: 'string' },
      { key: 'cityPrefix', required: true, type: 'string' },
      { key: 'siteNo', required: false, type: 'number' },
      { key: 'parentId', required: false, type: 'number' },
      { key: 'name', required: false, type: 'string' },
    ],
    samples: [
      // 示例行必须能通过后端校验真实入库:OLT 顶层设备可不挂 parent(禁填内部 id 0)。
      { code: 'OLT001', kind: 'OLT', prvCode: 'PHL001', cityPrefix: 'MNL', siteNo: 88, name: '示例核心设备' },
    ],
  },
  {
    kind: 'odn_facility',
    perm: 'menu:odn',
    endpoint: '/odn/facilities',
    uniqueKey: ['code'],
    listEndpoint: '/odn/facilities',
    queryColumns: ['prvCode', 'cityPrefix'],
    columns: [
      { key: 'code', required: true, type: 'string' },
      { key: 'kind', required: true, type: 'string' },
      { key: 'prvCode', required: true, type: 'string' },
      { key: 'cityPrefix', required: true, type: 'string' },
      { key: 'gridCode', required: false, type: 'number' },
      { key: 'name', required: false, type: 'string' },
      { key: 'lat', required: false, type: 'number' },
      { key: 'lng', required: false, type: 'number' },
    ],
    samples: [
      // 编码须符合 DB CHECK:^(P|MH|TW|CLS|TBX)[0-9]{5}$;MH+网格 12 → MH12001。
      { code: 'MH12001', kind: 'MH', prvCode: 'PHL001', cityPrefix: 'MNL', gridCode: 12, name: '示例设施', lat: 14.6, lng: 121.0 },
    ],
  },
]

export function findEntity(kind: string): EntityDef | undefined {
  return IMPORT_ENTITIES.find((e) => e.kind === kind)
}
