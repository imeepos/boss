// 标签页组件级用例:必填校验、载荷组装、操作列状态显隐(仅 DISABLED 显启用、仅 BOUND 显解绑)。
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { LocaleProvider } from '../../../i18n/context'
import { TagFormFields } from './TagFormFields'
import { buildTagPayload, emptyTagForm, tagActionsOf, tagFormErr } from './logic'

const renderFields = (over: { tagNo?: string; epcCode?: string; error?: string }) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <TagFormFields
        form={{ tagNo: over.tagNo ?? '', epcCode: over.epcCode ?? '', band: 'UHF' }}
        error={over.error ?? ''}
        onTagNo={() => {}} onEpc={() => {}} onBand={() => {}} />
    </LocaleProvider>,
  )

describe('TagFormFields', () => {
  it('标签编号与 EPC 为必填(红星标记),校验失败展示 eTagNo 文案', () => {
    const html = renderFields({ error: 'eTagNo' })
    expect(html).toContain('*')
    expect(html).toContain('标签编号与 EPC 码必填')
  })
  it('频段为选填:必填星标仅两处(编号/EPC)', () => {
    const html = renderFields({})
    expect((html.match(/text-\[var\(--color-danger\)\]/g) ?? []).length).toBe(2)
  })
})

describe('tagFormErr / buildTagPayload', () => {
  it('缺标签编号或 EPC 时报必填,齐备返回空串', () => {
    expect(tagFormErr({ ...emptyTagForm })).toBe('eTagNo')
    expect(tagFormErr({ tagNo: 'T-1', epcCode: '', band: '' })).toBe('eTagNo')
    expect(tagFormErr({ tagNo: 'T-1', epcCode: 'E200', band: '' })).toBe('')
  })
  it('载荷组装去首尾空白且仅含三个契约字段', () => {
    const payload = buildTagPayload({ tagNo: ' T-9 ', epcCode: ' E200 ', band: ' UHF ' })
    expect(payload).toEqual({ tagNo: 'T-9', epcCode: 'E200', band: 'UHF' })
    expect(Object.keys(payload).sort()).toEqual(['band', 'epcCode', 'tagNo'])
  })
})

describe('tagActionsOf 状态显隐', () => {
  it('UNBOUND:禁用+事件流,无启用/解绑', () => {
    expect(tagActionsOf({ status: 'UNBOUND' })).toEqual(['disable', 'events'])
  })
  it('BOUND:禁用+解绑+事件流', () => {
    expect(tagActionsOf({ status: 'BOUND' })).toEqual(['disable', 'unbind', 'events'])
  })
  it('DISABLED:仅启用+事件流,不再提供禁用', () => {
    expect(tagActionsOf({ status: 'DISABLED' })).toEqual(['enable', 'events'])
  })
})
