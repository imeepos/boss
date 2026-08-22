// 组织树纯逻辑单测:结构组装/成员归属/关键字过滤。
import { describe, expect, it } from 'vitest'
import { buildOrgTree, filterTree, membersOf } from './tree'

const entities = [{ id: 1, code: 'LEG-A', name: '总公司' }]
const departments = [
  { id: 10, legalEntityId: 1, name: '运维部' },
  { id: 11, legalEntityId: 1, name: '客服部' },
]
const posts = [
  { id: 100, code: 'dispatcher', name: '调度员', deptId: 10 },
  { id: 101, code: 'agent', name: '坐席', deptId: 11 },
]
const members = [
  { id: 1001, username: 'a1', realName: '张三', legalEntityId: 1, deptId: 10, postId: 100 },
  { id: 1002, username: 'a2', realName: '李四', legalEntityId: 1, deptId: 10 },
  { id: 1003, username: 'a3', realName: '王五', legalEntityId: 1, deptId: 11, postId: 101 },
]

describe('buildOrgTree', () => {
  it('三层挂靠正确且 memberCount 按直接挂靠统计', () => {
    const tree = buildOrgTree(entities, departments, posts, members)
    expect(tree).toHaveLength(1)
    const e = tree[0]
    expect(e.memberCount).toBe(3)
    expect(e.depts.map((d) => d.name)).toEqual(['运维部', '客服部'])
    expect(e.depts[0].memberCount).toBe(2)
    expect(e.depts[0].posts[0].memberCount).toBe(1)
    expect(e.depts[1].posts[0].memberCount).toBe(1)
  })
})

describe('membersOf', () => {
  const tree = buildOrgTree(entities, departments, posts, members)
  it('企业选中=全部归属成员', () => {
    expect(membersOf(members, { kind: 'entity', id: 1 })).toHaveLength(3)
  })
  it('部门选中=该部门成员(含未挂岗)', () => {
    const rows = membersOf(members, { kind: 'dept', id: 10 })
    expect(rows.map((r) => r.username)).toEqual(['a1', 'a2'])
  })
  it('岗位选中=挂该岗成员', () => {
    expect(membersOf(members, { kind: 'post', id: 100 })).toHaveLength(1)
  })
  it('空选择返回空', () => {
    expect(membersOf(members, null)).toEqual([])
  })
  expect(tree).toBeDefined()
})

describe('filterTree', () => {
  const tree = buildOrgTree(entities, departments, posts, members)
  it('命中部门名保留整支', () => {
    const out = filterTree(tree, '运维')
    expect(out[0].depts).toHaveLength(1)
    expect(out[0].depts[0].posts).toHaveLength(1)
  })
  it('命中岗位保留并带出祖先', () => {
    const out = filterTree(tree, 'agent')
    expect(out[0].depts[0].name).toBe('客服部')
  })
  it('空关键字原样返回', () => {
    expect(filterTree(tree, '  ')).toBe(tree)
  })
})
