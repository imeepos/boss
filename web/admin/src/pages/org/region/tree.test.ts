// 经营区域树构建与菜单权限矩阵筛选用例。
import { describe, expect, it } from 'vitest'
import { buildRegionView, filterRegions, type RegionRow } from './tree'
import { filterMatrixRows, type MenuPermRow } from '../menuperm/matrix'

const regions: RegionRow[] = [
  { id: 1, path: 'root', level: 1, name: '集团', parent: '' },
  { id: 2, path: 'root.luzon', level: 2, name: '吕宋大区', parent: 'root' },
  { id: 3, path: 'root.luzon.manila', level: 3, name: '马尼拉', parent: 'root.luzon' },
]

describe('region tree', () => {
  it('buildRegionView 派生级别名与直接子级数', () => {
    const view = buildRegionView(regions, ['集团', '大区', '省', '城市'])
    expect(view[0]).toMatchObject({ levelName: '集团', childCount: 1 })
    expect(view[1]).toMatchObject({ levelName: '大区', childCount: 1 })
    expect(view[2]).toMatchObject({ levelName: '省', childCount: 0 })
  })
  it('下钻限定子树(不含自身),级别过滤正交', () => {
    const view = buildRegionView(regions, ['集团', '大区', '省', '城市'])
    expect(filterRegions(view, '', '', 'root')).toHaveLength(2)
    expect(filterRegions(view, '', '', 'root.luzon')).toHaveLength(1)
    expect(filterRegions(view, '', '2', 'root')).toHaveLength(1)
    expect(filterRegions(view, '马尼拉', '', '')).toHaveLength(1)
  })
})

const menuRows: MenuPermRow[] = [
  { code: 'menu:billing', name: '计费与账务', roles: ['ops', 'sysadmin'] },
  { code: 'menu:ams', name: '资产与标签', roles: ['asset_admin'] },
]

describe('filterMatrixRows', () => {
  it('关键字命中菜单名/code,角色命中=持权', () => {
    expect(filterMatrixRows(menuRows, '计费', '')).toHaveLength(1)
    expect(filterMatrixRows(menuRows, 'ams', '')).toHaveLength(1)
    expect(filterMatrixRows(menuRows, '', 'ops')).toHaveLength(1)
    expect(filterMatrixRows(menuRows, '', 'analyst')).toHaveLength(0)
  })
})
