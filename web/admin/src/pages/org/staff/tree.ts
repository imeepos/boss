// 组织树纯逻辑:企业→部门→岗位三层模型 + 成员数统计 + 选中节点成员过滤。
// 口径:docs/contract/fields.md 1.1/1.4;成员=accounts(在职+停用都计数,状态由列表展示)。

export interface EntityInput { id: number; code: string; name: string }
export interface DeptInput { id: number; legalEntityId: number; name: string }
export interface PostInput { id: number; code: string; name: string; deptId: number; roles?: string[] }
export interface MemberInput {
  id: number
  username: string
  realName: string
  legalEntityId?: number
  deptId?: number
  postId?: number
}

export interface PostNode extends PostInput { memberCount: number }
export interface DeptNode extends DeptInput { posts: PostNode[]; memberCount: number }
export interface EntityNode extends EntityInput { depts: DeptNode[]; memberCount: number }

export type Selection =
  | { kind: 'entity'; id: number }
  | { kind: 'dept'; id: number }
  | { kind: 'post'; id: number }

/** 组装组织树:部门挂企业、岗位挂部门,memberCount=直接挂靠该节点的成员数。 */
export function buildOrgTree(
  entities: EntityInput[],
  departments: DeptInput[],
  posts: PostInput[],
  members: MemberInput[],
): EntityNode[] {
  return entities.map((e) => {
    const depts: DeptNode[] = departments
      .filter((d) => d.legalEntityId === e.id)
      .map((d) => {
        const postNodes: PostNode[] = posts
          .filter((p) => p.deptId === d.id)
          .map((p) => ({ ...p, memberCount: members.filter((m) => m.postId === p.id).length }))
        return { ...d, posts: postNodes, memberCount: members.filter((m) => m.deptId === d.id).length }
      })
    return { ...e, depts, memberCount: members.filter((m) => m.legalEntityId === e.id).length }
  })
}

/** 选中节点命中的成员列表:企业=全部归属,部门=该部门(含未挂岗),岗位=挂该岗。 */
export function membersOf<T extends MemberInput>(members: T[], sel: Selection | null): T[] {
  if (!sel) return []
  switch (sel.kind) {
    case 'entity':
      return members.filter((m) => m.legalEntityId === sel.id)
    case 'dept':
      return members.filter((m) => m.deptId === sel.id)
    case 'post':
      return members.filter((m) => m.postId === sel.id)
  }
}

/** 树内关键字过滤:命中节点保留并带上全部祖先(名称/编码不敏感)。 */
export function filterTree(tree: EntityNode[], kw: string): EntityNode[] {
  const q = kw.trim().toLowerCase()
  if (!q) return tree
  const hitPost = (p: PostNode) => p.name.toLowerCase().includes(q) || p.code.toLowerCase().includes(q)
  return tree
    .map((e) => {
      const depts = e.depts
        .map((d) => {
          const nameHit = d.name.toLowerCase().includes(q)
          const posts = d.posts.filter(hitPost)
          return (nameHit || posts.length) ? { ...d, posts: nameHit ? d.posts : posts } : null
        })
        .filter((d): d is DeptNode => d !== null)
      const selfHit = e.name.toLowerCase().includes(q) || e.code.toLowerCase().includes(q)
      return (selfHit || depts.length) ? { ...e, depts: selfHit ? e.depts : depts } : null
    })
    .filter((e): e is EntityNode => e !== null)
}
