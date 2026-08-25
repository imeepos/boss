// BatchImportEntry 渲染回归:未知 kind 渲染 null;实体/addr/geo 按钮文案与标题;
// 无权限时按钮 title 提示缺失权限码(不允许静默灰按钮)。
import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { BatchImportEntry } from './BatchImportEntry'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
vi.mock('../../../layouts/profile', () => ({
  useProfile: () => ({ permissionCodes: ['menu:account', 'menu:geo'] as string[] }),
}))
vi.mock('../../../i18n', () => ({
  useT: () => ({
    pages: {
      importer: {
        title: '数据导入中心',
        importBtn: '导入',
        entryAddr: '导入地址',
        entryGeo: '导入地理数据',
        addrTitle: '地址层级导入',
        geoTitle: 'ISO 地理数据导入',
        entityTitle: '业务数据批量导入',
        entityNoPerm: '当前账号缺少 {perm},该目标禁止导入',
        entityNames: { account: '账号', legal_entity: '法人公司' },
      },
    },
  }),
}))

describe('BatchImportEntry', () => {
  it('未知 kind 渲染 null', () => {
    expect(renderToStaticMarkup(<BatchImportEntry kind="nope" onImported={() => undefined} />)).toBe('')
  })

  it('实体 kind:按钮文案=导入+实体名,无权限时 title 提示缺失权限码', () => {
    const html = renderToStaticMarkup(<BatchImportEntry kind="legal_entity" onImported={() => undefined} />)
    expect(html).toContain('导入 法人公司')
    // legal_entity 需要 menu:company,mock 档案未持有 → title 须含权限码
    expect(html).toContain('menu:company')
  })

  it('实体 kind 持有权限时 title 不含权限提示', () => {
    const html = renderToStaticMarkup(<BatchImportEntry kind="account" onImported={() => undefined} />)
    expect(html).toContain('导入 账号')
    expect(html).not.toContain('menu:account')
  })

  it('addr/geo kind 复用基础导入入口文案', () => {
    expect(renderToStaticMarkup(<BatchImportEntry kind="addr" onImported={() => undefined} />)).toContain('导入地址')
    expect(renderToStaticMarkup(<BatchImportEntry kind="geo" onImported={() => undefined} />)).toContain('导入地理数据')
  })

  it('自定义 label 覆盖缺省按钮文案', () => {
    expect(renderToStaticMarkup(<BatchImportEntry kind="account" onImported={() => undefined} label="批量导入账号" />))
      .toContain('批量导入账号')
  })
})
