// @vitest-environment jsdom
// 导入面板 JSON 粘贴校验回归:blur 时解析,失败给行:列定位 ErrorBanner;
// 输入过程中(未失焦)不报错。
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { ImportPanel } from './ImportPanel'
import type { Translations } from '../../../i18n/types'

vi.mock('../../../api/client', () => ({ apiFetch: vi.fn() }))
vi.mock('../../../i18n', () => ({
  useT: () => ({
    common: { copy: '复制', copied: '已复制', loading: '加载中', statusTags: {}, confirmDialog: { title: '确认', ok: '确定', cancel: '取消' } },
    pages: {
      importer: new Proxy({ pickTitle: '选择附件', pickUse: '使用', pickFetching: '获取中', pickFetchFail: '获取失败' } as Record<string, unknown>, { get: (t, k) => (k in t ? t[k as string] : k) }),
    },
  }),
}))

const text = {
  title: '地址导入', addrTitle: '地址层级', addrHint: '', geoTitle: '地理', geoHint: '',
  importBtn: '执行导入', parseFail: 'JSON 解析失败',
  parseFailAt: 'JSON 解析失败(第 {line} 行)',
  parseFailAtCol: 'JSON 解析失败(第 {line} 行 第 {col} 列)',
  reasonNotArray: '顶层应为 JSON 数组', reasonNotObject: '顶层应为对象', reasonBadRow: '缺字段',
  previewOf: '预览 · 共 {count} 行', previewTruncated: '仅预览前', clear: '清空',
  template: 'JSON 模板', templateExcel: 'Excel 模板', pasteToggle: '粘贴 JSON',
  pickFromAttachments: '从附件选择', pickTitle: '', pickUse: '', pickFetching: '', pickFetchFail: '',
  fileButton: '选择文件', dropHint: '拖拽', onlyJson: '仅 JSON', fileTooLarge: '过大', readFail: '读取失败',
  imported: '导入 {count} 行', importedGeo: '导入完成', importing: '导入中…',
  addrColumns: ['path', 'name'], geoSections: [], excelNoSheet: '', excelBadHeader: '', excelBadRow: '',
  pastePlaceholder: '',
} as unknown as Translations['pages']['importer']

let root: Root | null = null

afterEach(() => {
  act(() => { root?.unmount() })
  document.body.innerHTML = ''
  root = null
})

async function mount() {
  const host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  await act(async () => {
    root?.render(<ImportPanel kind="addr" title={text.addrTitle} hint={text.addrHint} endpoint="/geo/addresses:import" text={text} onImported={() => {}} />)
  })
}

function setTextarea(value: string) {
  const ta = document.body.querySelector('textarea') as HTMLTextAreaElement
  const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set
  setter?.call(ta, value)
  ta.dispatchEvent(new Event('input', { bubbles: true }))
  return ta
}

describe('ImportPanel JSON blur 校验', () => {
  it('失焦后解析失败渲染行:列 ErrorBanner', async () => {
    await mount()
    act(() => {
      [...document.body.querySelectorAll('button')].find((b) => b.textContent === '粘贴 JSON')?.click()
    })
    const ta = setTextarea('{foo}')
    // React onBlur 经原生 focusout(冒泡)实现,直接派发 focusout 触发失焦校验
    await act(async () => { ta.dispatchEvent(new FocusEvent('focusout', { bubbles: true })) })
    const banner = [...document.body.querySelectorAll('div')].find((d) => typeof d.className === 'string' && d.className.includes('color-danger'))
    expect(banner?.textContent).toContain('JSON 解析失败(第 1 行 第 2 列)')
  })

  it('输入过程中未失焦不报错;修正后失焦横幅清除', async () => {
    await mount()
    act(() => {
      [...document.body.querySelectorAll('button')].find((b) => b.textContent === '粘贴 JSON')?.click()
    })
    setTextarea('{bad}')
    expect(document.body.textContent).not.toContain('JSON 解析失败')
    const ta = setTextarea('[{"path":"cn","name":"China","countryCode":"CN"}]')
    await act(async () => { ta.dispatchEvent(new FocusEvent('focusout', { bubbles: true })) })
    const banner = [...document.body.querySelectorAll('div')].find((d) => typeof d.className === 'string' && d.className.includes('color-danger'))
    expect(banner).toBeUndefined()
    expect(document.body.textContent).toContain('预览 · 共 1 行')
  })
})
