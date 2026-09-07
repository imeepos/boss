import { describe, expect, it } from 'vitest'
import { legalParentKind, nextDeviceCode } from './nextcode'

describe('legalParentKind', () => {
  it('归属链镜像域层 parentKind(规范 2.1+2.4)', () => {
    expect(legalParentKind('ODB')).toBe('OCC')
    expect(legalParentKind('OBD')).toBe('ODB')
    expect(legalParentKind('SDB')).toBe('ODB')
    expect(legalParentKind('SBD')).toBe('SDB')
    expect(legalParentKind('PRT')).toBe('SDB')
    expect(legalParentKind('TBP')).toBe('PRT')
  })
  it('顶层设备无上级要求', () => {
    for (const k of ['SNW', 'OLT', 'ODF', 'OCC']) expect(legalParentKind(k)).toBe('')
    expect(legalParentKind('XX')).toBe('')
  })
})

describe('nextDeviceCode', () => {
  it('空列表从 001 起', () => {
    expect(nextDeviceCode('OLT', [])).toBe('OLT001')
  })
  it('现存最大序号+1,跨序号稀疏取最大', () => {
    expect(nextDeviceCode('ODB', ['ODB001', 'ODB013', 'ODB007'])).toBe('ODB014')
  })
  it('扩容后缀行不参与计数', () => {
    expect(nextDeviceCode('ODB', ['ODB001', 'ODB001-2', 'ODB001-9'])).toBe('ODB002')
  })
  it('忽略其他 kind 编码', () => {
    expect(nextDeviceCode('OLT', ['SDB999', 'PRT123'])).toBe('OLT001')
  })
  it('满 999 返回 null', () => {
    expect(nextDeviceCode('OLT', ['OLT999'])).toBeNull()
  })
})
