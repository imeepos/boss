// 部门筛选纯函数(契约: GET /departments,字段口径 fields.md 1.4)。

export interface DepartmentRow {
  id: number
  legalEntityId: number
  legalEntity: string
  name: string
}

/** 关键字命中 部门名/所属子公司名(大小写不敏感)。 */
export function filterDepartments(list: DepartmentRow[], keyword: string): DepartmentRow[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return list
  return list.filter((d) => d.name.toLowerCase().includes(kw) || d.legalEntity.toLowerCase().includes(kw))
}

/** 分页切片(list 量级小,前端分页)。 */
export function pageSlice<T>(list: T[], page: number, pageSize: number): T[] {
  const safe = Math.max(1, page)
  return list.slice((safe - 1) * pageSize, safe * pageSize)
}
