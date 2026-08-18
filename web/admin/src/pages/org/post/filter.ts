// 岗位筛选纯函数(契约: GET /posts,字段口径 fields.md 1.4)。

export interface PostRow {
  id: number
  code: string
  name: string
  deptId: number
  deptName: string
  roles: string[]
}

/** 关键字命中 岗位代码/岗位名/部门名(大小写不敏感)。 */
export function filterPosts(list: PostRow[], keyword: string): PostRow[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return list
  return list.filter((p) =>
    p.code.toLowerCase().includes(kw)
    || p.name.toLowerCase().includes(kw)
    || p.deptName.toLowerCase().includes(kw))
}

/** 分页切片(list 量级小,前端分页)。 */
export function pageSlice<T>(list: T[], page: number, pageSize: number): T[] {
  const safe = Math.max(1, page)
  return list.slice((safe - 1) * pageSize, safe * pageSize)
}
