// 经营区域纯逻辑:视图构建(级别名/下级数派生)+ 三条件筛选 + 下钻前缀。

export interface RegionRow {
  id: number
  path: string
  level: number
  name: string
  parent: string
}

export interface RegionView extends RegionRow {
  levelName: string
  childCount: number
}

/** 全量 → 视图:挂级别名,统计每个 path 的直接子级数。 */
export function buildRegionView(rows: RegionRow[], levelNames: string[]): RegionView[] {
  const childCountByParent = new Map<string, number>()
  for (const r of rows) childCountByParent.set(r.parent, (childCountByParent.get(r.parent) ?? 0) + 1)
  return rows.map((r) => ({
    ...r,
    levelName: levelNames[r.level - 1] ?? String(r.level),
    childCount: childCountByParent.get(r.path) ?? 0,
  }))
}

/** 筛选:关键字(名称/路径) + 级别 + 下钻(parentPath 前缀限定子树)。 */
export function filterRegions(
  view: RegionView[],
  keyword: string,
  level: string,
  drillPath: string,
): RegionView[] {
  const kw = keyword.trim().toLowerCase()
  return view.filter((r) => {
    const hitKw = !kw || r.name.toLowerCase().includes(kw) || r.path.toLowerCase().includes(kw)
    const hitLevel = !level || String(r.level) === level
    const hitDrill = !drillPath || (r.path.startsWith(drillPath + '.') && r.path !== drillPath)
    return hitKw && hitLevel && hitDrill
  })
}

/** 分页切片(list 量级小,前端分页)。 */
export function pageSlice<T>(list: T[], page: number, pageSize: number): T[] {
  const safe = Math.max(1, page)
  return list.slice((safe - 1) * pageSize, safe * pageSize)
}
