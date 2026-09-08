// T2 生命周期选择器回归:建设施默认 PLANNED、可作态只含 PLANNED/IN_SERVICE、
// 提示文案随表单渲染;局点/设备建单不出现该选择器(口径仅设施)。
// Dropdown 选中值经 effect 回显,SSR 标记断言不可见,故默认态断言走 initialFormState。
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { CREATE_LIFECYCLE, initialFormState, ResourceDrawer, type DrawerTarget } from './ResourceDrawer'
import type { Facility, Site } from './forms'

const g = {
  cancel: '取消', save: '保存', saving: '保存中', saveOk: '已保存', saveFail: '保存失败', empty: '无数据',
  prvLabel: '省份', cityLabel: '城市', kind: '类型', name: '名称', coverage: '覆盖区域', status: '状态',
  lat: '纬度', lng: '经度', gridCode: '网格码', relGrid: '所属网格', relSite: '归属局点', parentDevice: '上级设备',
  autoCode: '设施编码(自动顺延)', autoSiteNo: '局点序号(自动)',
  drawer: {
    createTitle: '新增资源', editTitle: '编辑资源', hintOverride: '编码自动顺延,可覆盖',
    lifecycle: '生命周期', lifecycleHint: '新建设施默认规划态(可入施工单);登记既有在网设施选 IN_SERVICE',
    edit: '编辑', missingRequired: '请补全必填项',
    kindP: '电杆 P', kindMH: '人井 MH', kindTW: '铁塔 TW', kindCLS: '接头盒 CLS', kindTBX: '终端盒 TBX',
  },
}

const props = (target: DrawerTarget) => ({
  target, prv: 'PHL001', city: 'MNL', regions: [], grids: [], sites: [], devices: [], g, onClose: () => {}, onSaved: () => {},
})

describe('CREATE_LIFECYCLE 可作生态', () => {
  it('只含 PLANNED/IN_SERVICE,PLANNED 为首项默认', () => {
    expect(CREATE_LIFECYCLE).toEqual(['PLANNED', 'IN_SERVICE'])
  })
})

describe('initialFormState 生命周期初值', () => {
  it('create 默认 PLANNED(T2 规划新建口径)', () => {
    expect(initialFormState(null, 'PHL001', 'MNL').lifecycle).toBe('PLANNED')
  })
  it('edit 行 lifecycleStatus 透传', () => {
    const fac: Facility = { code: 'P01001', kind: 'P', prvCode: 'PHL001', cityPrefix: 'MNL', gridCode: 1, name: '', lat: null, lng: null, status: 'IN_USE', lifecycleStatus: 'IN_BUILD' }
    expect(initialFormState(fac, '', '').lifecycle).toBe('IN_BUILD')
  })
  it('存量行缺 lifecycleStatus:RETIRED→RETIRED,其余→IN_SERVICE(存量语义不变)', () => {
    const st: Site = { prvCode: 'PHL001', cityPrefix: 'MNL', siteNo: 1, name: '', lat: null, lng: null, status: 'ACTIVE' }
    expect(initialFormState(st, '', '').lifecycle).toBe('IN_SERVICE')
    expect(initialFormState({ ...st, status: 'RETIRED' }, '', '').lifecycle).toBe('RETIRED')
  })
})

describe('ResourceDrawer 建设施生命周期选择器', () => {
  it('create 模式渲染生命周期字段 + 口径提示', () => {
    const html = renderToStaticMarkup(<ResourceDrawer {...props({ tab: 'facilities', editing: null })} />)
    expect(html).toContain('生命周期')
    expect(html).toContain('新建设施默认规划态(可入施工单);登记既有在网设施选 IN_SERVICE')
  })

  it('create 局点不渲染生命周期选择器(口径仅设施)', () => {
    const html = renderToStaticMarkup(<ResourceDrawer {...props({ tab: 'sites', editing: null })} />)
    expect(html).not.toContain('生命周期')
    expect(html).not.toContain('新建设施默认规划态')
  })

  it('edit 模式保留四态转移选择器(不带 create 提示)', () => {
    const fac: Facility = { code: 'P01001', kind: 'P', prvCode: 'PHL001', cityPrefix: 'MNL', gridCode: 1, name: '', lat: null, lng: null, status: 'IN_USE', lifecycleStatus: 'PLANNED' }
    const html = renderToStaticMarkup(<ResourceDrawer {...props({ tab: 'facilities', editing: fac })} />)
    expect(html).toContain('生命周期')
    expect(html).not.toContain('新建设施默认规划态')
  })
})
