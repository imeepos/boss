// 子公司/法人筛选纯函数(契约: GET /legal-entities,字段口径 fields.md 1.4)。

export interface LegalEntityRow {
  id: number
  code: string
  name: string
  taxJurisdiction?: string
  taxChannel?: string
}

/** 关键词命中 code/name(大小写不敏感)。 */
export function filterLegalEntities(list: LegalEntityRow[], keyword: string): LegalEntityRow[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return list
  return list.filter((e) => e.code.toLowerCase().includes(kw) || e.name.toLowerCase().includes(kw))
}

/** 分页切片(前端分页,列表量级小)。 */
export function pageSlice<T>(list: T[], page: number, pageSize: number): T[] {
  const safe = Math.max(1, page)
  return list.slice((safe - 1) * pageSize, safe * pageSize)
}
