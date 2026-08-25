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
  columns: EntityColumn[]
  /** 模板样例行(与 columns 同序的字段集)。 */
  samples: Array<Record<string, string | number | string[]>>
}

export const IMPORT_ENTITIES: EntityDef[] = [
  {
    kind: 'account',
    endpoint: '/accounts',
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
      { username: 'op102', password: 'Init@1024', realName: '运营一号', roleCode: 'operator', phone: '09170000001', legalEntityId: 1, deptId: 1, postId: 1, regionScope: '' },
    ],
  },
  {
    kind: 'legalEntity',
    endpoint: '/legal-entities',
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
    endpoint: '/departments',
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
    endpoint: '/posts',
    columns: [
      { key: 'deptId', required: true, type: 'number' },
      { key: 'code', required: true, type: 'string' },
      { key: 'name', required: true, type: 'string' },
      { key: 'roles', required: false, type: 'list' },
    ],
    samples: [
      { deptId: 1, code: 'POST-IMP-01', name: '示例岗位', roles: 'operator,worker' },
    ],
  },
  {
    kind: 'product',
    endpoint: '/products',
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
]

export function findEntity(kind: string): EntityDef | undefined {
  return IMPORT_ENTITIES.find((e) => e.kind === kind)
}
