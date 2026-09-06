// 报废三要素确认弹窗用例:参考值展示(不可复制)、动态必填切换(有无 SN/有无绑标签)
// 与预校验错误文案。Radix Dialog 在 renderToStaticMarkup 下渲染为空,
// 故直测导出的 ScrapRefBlock/ScrapFields;纯规则在 logic.test.ts scrapConfirmErr 覆盖。
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { LocaleProvider, useT } from '../../../i18n'
import { ScrapFields, ScrapRefBlock, type ErrField } from './ScrapDialog'

const ref = (sn: string, tagNo: string) => renderToStaticMarkup(
  <ScrapRefBlock title='核对实物铭牌' codeLabel='资产编码' code='A-20260001' snLabel='SN' sn={sn} tagLabel='标签编号' tagNo={tagNo} />,
)

// useT 必须在 LocaleProvider 内调用:内层组件取文案,外层负责包 Provider。
const FieldsInner = (p: { hasSn: boolean; hasTag: boolean; errField: ErrField }) => {
  const t = useT()
  const a = t.pages.assetPage
  return (
    <ScrapFields a={a} hasSn={p.hasSn} hasTag={p.hasTag} errField={p.errField} clear={() => { }}
      reason='' onReason={() => { }} code='' onCode={() => { }} sn='' onSn={() => { }} tagNo='' onTagNo={() => { }} />
  )
}
const Fields = (p: { hasSn: boolean; hasTag: boolean; errField: ErrField }) => (
  <LocaleProvider><FieldsInner hasSn={p.hasSn} hasTag={p.hasTag} errField={p.errField} /></LocaleProvider>
)
const fields = (p: { hasSn: boolean; hasTag: boolean; errField: ErrField }) => renderToStaticMarkup(Fields(p))

describe('ScrapRefBlock 参考值展示', () => {
  it('三要素参考值齐全且 select-none 不可复制', () => {
    const html = ref('SN-ABC-123', 'T-0001')
    expect(html).toContain('select-none')
    expect(html).toContain('A-20260001')
    expect(html).toContain('SN-ABC-123')
    expect(html).toContain('T-0001')
  })
  it('无 SN/未绑标签:对应参考行不出现', () => {
    expect(ref('', '')).not.toContain('SN-ABC')
    const noTag = ref('SN-1', '')
    expect(noTag).toContain('SN-1')
    expect(noTag).not.toContain('T-0001')
  })
})

describe('ScrapFields 动态必填切换', () => {
  it('有 SN 有标签:三确认输入齐全,原因恒在', () => {
    const html = fields({ hasSn: true, hasTag: true, errField: '' })
    expect(html).toContain('报废原因')
    expect(html).toContain('输入资产编码确认')
    expect(html).toContain('输入 SN 确认')
    expect(html).toContain('输入标签编号确认')
  })
  it('无 SN:SN 确认输入不出现', () => {
    expect(fields({ hasSn: false, hasTag: true, errField: '' })).not.toContain('输入 SN 确认')
  })
  it('未绑标签:标签号确认输入不出现,编码确认恒在', () => {
    const html = fields({ hasSn: true, hasTag: false, errField: '' })
    expect(html).toContain('输入资产编码确认')
    expect(html).not.toContain('输入标签编号确认')
  })
  it('预校验错误文案按 errField 定位渲染', () => {
    expect(fields({ hasSn: true, hasTag: true, errField: 'code' })).toContain('请输入资产编码完成确认')
    expect(fields({ hasSn: true, hasTag: true, errField: 'sn' })).toContain('请输入 SN 完成确认')
    expect(fields({ hasSn: true, hasTag: true, errField: 'tagNo' })).toContain('请输入标签编号完成确认')
    expect(fields({ hasSn: true, hasTag: true, errField: 'reason' })).toContain('报废原因必填(64 字内)')
  })
})
