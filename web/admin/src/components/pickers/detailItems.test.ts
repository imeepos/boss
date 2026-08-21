// 详情字段映射单测:确认 k-v 顺序与取值、状态与缺省回退。
import { describe, expect, it } from 'vitest'
import { customerDetailItems, userDetailItems, workerDetailItems } from './detailItems'
import type { CustomerItem, UserDetail, WorkerItem } from '../../api/pickers'

const wTexts = {
  name: '姓名', staffNo: '工号', phone: '手机号', group: '班组', region: '负责区域',
  status: '状态', onDuty: '在职', left: '离职', joinedAt: '入职时间',
}
const cTexts = {
  code: '客户编码', name: '姓名', phone: '手机号', idType: '证件类型', idNo: '证件号',
  realNameStatus: '实名状态', serviceStatus: '服务状态', region: '所在区域', createdAt: '建档时间',
}
const uTexts = {
  name: '姓名', phone: '手机号', plan: '当前套餐', balance: '余额', arrears: '欠费金额',
  activeOrders: '在途业务', realNameStatus: '实名状态', serviceStatus: '服务状态',
  autoPay: '自动缴费', on: '已开启', off: '未开启',
}

const worker: WorkerItem = {
  id: 6, staffNo: 'WK-1', name: '王测试', groupId: 11, regionId: 4,
  phone: '13800001001', status: 1, joinedAt: '2026-08-19T10:31:35Z',
}

describe('workerDetailItems', () => {
  it('映射主档字段并翻译在职状态', () => {
    const items = workerDetailItems(worker, wTexts)
    expect(items[0]).toEqual({ k: '姓名', v: '王测试' })
    expect(items.find((i) => i.k === '状态')?.v).toBe('在职')
    expect(items.find((i) => i.k === '班组')?.v).toBe('#11')
  })
  it('离职状态与无班组回退', () => {
    const items = workerDetailItems({ ...worker, status: 0, groupId: 0 }, wTexts)
    expect(items.find((i) => i.k === '状态')?.v).toBe('离职')
    expect(items.find((i) => i.k === '班组')?.v).toBe('—')
  })
})

describe('customerDetailItems', () => {
  const customer: CustomerItem = {
    id: 213, customerCode: 'C-00000213', name: '采购经理·王', phone: '13900001234',
    idType: '身份证', idNo: '110101***', realNameStatus: 'VERIFIED', serviceStatus: 'ACTIVE',
    addressId: 288, legalEntityId: 1, regionId: 4, regionName: '马尼拉市', createdAt: '2026-08-19T10:26:16Z',
  }
  it('映射档案字段,区域名缺失回退 #id', () => {
    const items = customerDetailItems(customer, cTexts)
    expect(items.find((i) => i.k === '客户编码')?.v).toBe('C-00000213')
    expect(items.find((i) => i.k === '所在区域')?.v).toBe('马尼拉市')
    const bare = customerDetailItems({ ...customer, regionName: '' }, cTexts)
    expect(bare.find((i) => i.k === '所在区域')?.v).toBe('#4')
  })
})

describe('userDetailItems', () => {
  const user: UserDetail = {
    customerId: 213, loginName: '', name: '采购经理·王', phone: '13900001234',
    planName: '家庭宽带100M', realNameStatus: 'VERIFIED', serviceStatus: 'ACTIVE',
    balance: 92, arrearsAmount: 0, activeOrders: 35, autoPay: false,
  }
  it('映射用户摘要字段并翻译自动缴费开关', () => {
    const items = userDetailItems(user, uTexts)
    expect(items.find((i) => i.k === '当前套餐')?.v).toBe('家庭宽带100M')
    expect(items.find((i) => i.k === '自动缴费')?.v).toBe('未开启')
    expect(userDetailItems({ ...user, autoPay: true }, uTexts).find((i) => i.k === '自动缴费')?.v).toBe('已开启')
  })
})
