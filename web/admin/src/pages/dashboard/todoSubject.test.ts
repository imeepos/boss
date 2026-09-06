import { describe, it, expect } from 'vitest'
import { splitSubject } from './todoSubject'

describe('splitSubject 待办主题拆分', () => {
  it('告警编号拆出编号与正文', () => {
    expect(splitSubject('ALM-1788709925884642555 容量预警: 端口使用率 100%')).toEqual({
      id: 'ALM-1788709925884642555',
      text: '容量预警: 端口使用率 100%',
    })
  })

  it('工单编号(双段数字)拆分', () => {
    expect(splitSubject('DT-20260903-000637 待指派师傅')).toEqual({
      id: 'DT-20260903-000637',
      text: '待指派师傅',
    })
  })

  it('非编号前缀不拆分', () => {
    expect(splitSubject('O20240001 待指派师傅')).toEqual({ id: '', text: 'O20240001 待指派师傅' })
  })

  it('无空格原样返回', () => {
    expect(splitSubject('纯文本待办')).toEqual({ id: '', text: '纯文本待办' })
  })

  it('空串安全', () => {
    expect(splitSubject('')).toEqual({ id: '', text: '' })
  })
})