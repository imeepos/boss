// 表单字段区组件级用例:必填标记与错误提示、型号选定后类型只读、非 IN_STOCK 批次禁用。
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { LocaleProvider } from '../../../i18n/context'
import { AssetFormFields } from './AssetFormFields'
import type { AssetFormFieldsProps } from './AssetFormFields'

const base = {
  batchOptions: [{ value: '9', label: '#9 B-001 一批' }],
  modelOptions: [{ value: '7', label: '华为 · ONU' }],
  tagOptions: [{ value: '3', label: 'T-0001' }],
  onBatch: () => {}, onModel: () => {}, onType: () => {}, onTag: () => {},
}

const render = (over: Partial<AssetFormFieldsProps>) => renderToStaticMarkup(
  <LocaleProvider>
    <AssetFormFields {...base} form={{ batchId: 9, modelId: 0, type: '', tagId: 0 }} typeValue=""
      typeReadonly={false} batchDisabled={false} batchHint="" error="" {...over} />
  </LocaleProvider>,
)

describe('AssetFormFields', () => {
  it('批次为必填项(红星标记),校验失败展示 eBatch 文案', () => {
    const html = render({ form: { batchId: 0, modelId: 0, type: '', tagId: 0 }, error: 'batch' })
    expect(html).toContain('*')
    expect(html).toContain('入库批次必填')
  })
  it('型号选定后类型只读并回显 category', () => {
    const html = render({ form: { batchId: 9, modelId: 7, type: '', tagId: 0 }, typeValue: 'ONU', typeReadonly: true })
    expect(html).toContain('readonly')
    expect(html).toContain('ONU')
  })
  it('未选型号时类型可编辑', () => {
    expect(render({})).not.toContain('readonly')
  })
  it('非 IN_STOCK 批次控件禁用并展示业务流转提示', () => {
    const html = render({ batchDisabled: true, batchHint: '仅入库(IN_STOCK)状态可更换批次,其余状态请走业务流转' })
    expect(html).toContain('disabled')
    expect(html).toContain('cursor-not-allowed')
    expect(html).toContain('业务流转')
  })
})