// @vitest-environment jsdom
// Drawer × 共享 Dialog 层级回归:导入抽屉里打开附件选择器,Dialog(z-dialog=130)必须浮于
// 抽屉遮罩(z-drawer-mask=100)/面板(z-drawer=101)之上——Portal 挂 body 后与 Drawer 同层叠上下文。
// 语义类数值从 styles.css --z-* 令牌解析对账,令牌改值/类名漂移都会红。
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { BatchImportEntry } from './BatchImportEntry'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn(() => Promise.resolve({ items: [] })) }))
vi.mock('../../../layouts/profile', () => ({
  useProfile: () => ({ permissionCodes: ['menu:account'] as string[] }),
}))
// 附件管理器内部实现与层级断言无关,桩掉以隔离 i18n/api 面
vi.mock('../../../components/AttachmentManager', () => ({
  AttachmentManager: () => <div data-testid="attachment-manager-stub" />,
}))
vi.mock('../../../i18n', () => ({
  useT: () => ({
    pages: {
      importer: {
        importBtn: '导入', entityTitle: '业务数据批量导入',
        entityNoPerm: '当前账号缺少 {perm},该目标禁止导入',
        entityNames: { account: '账号' },
        fileButton: '选择文件', dropHint: '或拖入', templateExcel: 'Excel 模板', template: 'JSON 模板',
        pasteToggle: '粘贴', pickFromAttachments: '从附件选择', clear: '清空',
        pickTitle: '选择附件', pickFetching: '拉取中', pickUse: '使用', pickFetchFail: '拉取失败',
      },
    },
    common: { confirmDialog: { cancel: '取消' } },
  }),
}))

const mockFetch = (await import('../../../api/client')).apiFetch as ReturnType<typeof vi.fn>

afterEach(() => {
  mockFetch.mockClear()
  document.body.innerHTML = ''
})

/** styles.css --z-* 令牌表(层级契约唯一事实源):--z-fab: 60 → { fab: 60 }。 */
const Z_TOKENS = Object.fromEntries(
  [...readFileSync(join(process.cwd(), 'src/styles.css'), 'utf8').matchAll(/--z-([a-z-]+):\s*(\d+)/g)]
    .map((m) => [m[1], Number(m[2])]),
)

/** 从 className 解析语义 z 类(z-dialog 等)并经令牌表换算数值;未标注视为 0。 */
function zOf(el: Element): number {
  const m = /\bz-([a-z][a-z-]*)/.exec(el.className)
  return m && m[1] in Z_TOKENS ? Z_TOKENS[m[1]] : 0
}

function findByText(root: ParentNode, text: string): HTMLElement | null {
  return Array.from(root.querySelectorAll<HTMLElement>('button, [role="button"]'))
    .find((el) => el.textContent?.includes(text)) ?? null
}

describe('z-index ladder contract', () => {
  it('令牌阶梯:fab < 抽屉遮罩 < 抽屉 < 页面弹窗 < Dialog < 弹层', () => {
    expect(Object.keys(Z_TOKENS).sort()).toEqual(['dialog', 'drawer', 'drawer-mask', 'fab', 'page-modal', 'popover'])
    expect(Z_TOKENS.fab).toBeLessThan(Z_TOKENS['drawer-mask'])
    expect(Z_TOKENS['drawer-mask']).toBeLessThan(Z_TOKENS.drawer)
    expect(Z_TOKENS.drawer).toBeLessThan(Z_TOKENS['page-modal'])
    expect(Z_TOKENS['page-modal']).toBeLessThan(Z_TOKENS.dialog)
    expect(Z_TOKENS.dialog).toBeLessThan(Z_TOKENS.popover)
  })
})

describe('Drawer × Dialog layering', () => {
  it('导入抽屉内打开附件选择器:Dialog 层级须高于抽屉面板', async () => {
    const host = document.createElement('div')
    document.body.appendChild(host)
    const root = createRoot(host)
    await act(async () => { root.render(<BatchImportEntry kind="account" onImported={() => undefined} />) })

    // 1. 打开导入抽屉
    const importBtn = findByText(host, '导入 账号')
    expect(importBtn).not.toBeNull()
    await act(async () => { importBtn!.click() })
    const drawer = host.querySelector('aside[role="dialog"]')
    expect(drawer).not.toBeNull()
    expect(zOf(drawer!)).toBe(Z_TOKENS.drawer)

    // 2. 抽屉内点「从附件选择」,Radix Dialog 应 Portal 到 body 且可交互
    const pickBtn = findByText(drawer!, '从附件选择')
    expect(pickBtn).not.toBeNull()
    await act(async () => { pickBtn!.click() })
    await act(async () => { await Promise.resolve() })

    const portal = document.body.querySelector('[data-testid="attachment-manager-stub"]')
    expect(portal).not.toBeNull()
    const dialogContent = portal!.closest('[role="dialog"]')
    expect(dialogContent).not.toBeNull()
    expect(zOf(dialogContent!)).toBe(Z_TOKENS.dialog)

    // 3. 层级契约:Dialog 严格高于抽屉面板,抽屉面板高于自身遮罩
    expect(zOf(dialogContent!)).toBeGreaterThan(zOf(drawer!))
    const mask = Array.from(host.querySelectorAll('div')).find((el) => zOf(el) === Z_TOKENS['drawer-mask'])
    expect(mask).not.toBeNull()

    root.unmount()
  })
})
