// 采购域组件级用例:必填校验、载荷组装、状态显隐(仅 DRAFT 可编辑/驳回、仅 DISABLED 显启用)。
import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { LocaleProvider } from '../../../i18n/context'
import { OrderItemsEditor } from './OrderItemsEditor'
import {
  buildOrderEditPayload, buildRejectPayload, buildSupplierPayload, canEditOrder,
  canEnableSupplier, canRejectReceipt, emptySupplierForm, orderEditErr,
  rejectReasonErr, supplierFormErr, type OrderEditFormState, type SupplierFormState,
} from './purchaseLogic'

const editForm: OrderEditFormState = {
  supplierId: 3, legalEntityId: 1, remark: '加急',
  items: [
    { materialCode: 'MI-ONU', spec: '1GE', quantity: 10, unitAmount: 80, receivedQty: 4 },
  ],
}

const supForm: SupplierFormState = {
  code: ' S-2 ', name: ' 华光 ', contactName: ' 李四 ', contactPhone: ' 138 ',
  legalEntityId: 2, remark: ' 备注 ',
}

describe('purchaseLogic 显隐规则', () => {
  it('订单编辑仅 DRAFT 可见;入库驳回仅 DRAFT 可见', () => {
    for (const s of ['DRAFT', 'SUBMITTED', 'PARTIAL', 'RECEIVED', 'CANCELLED']) {
      expect(canEditOrder(s)).toBe(s === 'DRAFT')
      expect(canRejectReceipt(s)).toBe(s === 'DRAFT')
    }
  })
  it('供应商启用仅 DISABLED 可见', () => {
    expect(canEnableSupplier('DISABLED')).toBe(true)
    expect(canEnableSupplier('ENABLED')).toBe(false)
  })
})

describe('purchaseLogic 载荷组装', () => {
  it('PUT 订单载荷整体替换明细并剥掉只读 receivedQty', () => {
    const payload = buildOrderEditPayload(editForm)
    expect(payload).toEqual({
      supplierId: 3, legalEntityId: 1, remark: '加急',
      items: [{ materialCode: 'MI-ONU', spec: '1GE', quantity: 10, unitAmount: 80 }],
    })
    expect(JSON.stringify(payload)).not.toContain('receivedQty')
  })
  it('供应商载荷去空白且六键齐备', () => {
    expect(buildSupplierPayload(supForm)).toEqual({
      code: 'S-2', name: '华光', contactName: '李四', contactPhone: '138',
      legalEntityId: 2, remark: '备注',
    })
    expect(supplierFormErr({ ...emptySupplierForm })).toBe('supErrName')
    expect(supplierFormErr(supForm)).toBe('')
  })
  it('驳回原因必填且 255 字内,载荷去空白', () => {
    expect(rejectReasonErr({ reason: '' })).toBe('eRejectReason')
    expect(rejectReasonErr({ reason: ' '.repeat(3) })).toBe('eRejectReason')
    expect(rejectReasonErr({ reason: 'x'.repeat(256) })).toBe('eRejectReason')
    expect(rejectReasonErr({ reason: ' 外包装破损 ' })).toBe('')
    expect(buildRejectPayload({ reason: ' 外包装破损 ' })).toEqual({ reason: '外包装破损' })
  })
  it('订单编辑校验:缺供应商/明细缺物料或数量报错', () => {
    expect(orderEditErr({ ...editForm, supplierId: 0 })).toBe('errSupplier')
    expect(orderEditErr({ ...editForm, items: [{ materialCode: '', spec: '', quantity: 1, unitAmount: 0 }] })).toBe('errItems')
    expect(orderEditErr({ ...editForm, items: [{ materialCode: 'MI-ONU', spec: '', quantity: 0, unitAmount: 0 }] })).toBe('errItems')
    expect(orderEditErr(editForm)).toBe('')
  })
})

describe('OrderItemsEditor 组件', () => {
  it('渲染明细行值与添加明细按钮', () => {
    const html = renderToStaticMarkup(
      <LocaleProvider>
        <OrderItemsEditor items={[{ materialCode: 'MI-ONU', spec: '1GE', quantity: 2, unitAmount: 9 }]}
          onChange={() => {}} />
      </LocaleProvider>,
    )
    expect(html).toContain('MI-ONU')
    expect(html).toContain('1GE')
    expect(html).toContain('添加明细')
  })
})
