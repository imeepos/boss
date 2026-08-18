// 账号表单纯逻辑:校验 + 提交负载组装(口径:fields.md 1.1;规则与后端 user.AccountRules 一致)。

export interface AccountFormValues {
  id?: number
  username: string
  password: string
  realName: string
  phone: string
  roleCode: string
  legalEntityId: number
  deptId: number
  postId: number
  regionScope: string
}

export const USERNAME_RE = /^[A-Za-z0-9_\-.]{3,64}$/

/** 逐字段校验;编辑模式 password 留空=不修改,跳过密码校验。返回错误 key 数组(空=通过)。 */
export function validateAccount(v: AccountFormValues, isEdit: boolean): string[] {
  const errs: string[] = []
  if (!USERNAME_RE.test(v.username)) errs.push('invalidUsername')
  if (!isEdit && v.password.length < 6) errs.push('shortPassword')
  if (isEdit && v.password !== '' && v.password.length < 6) errs.push('shortPassword')
  if (!v.realName.trim() || v.realName.trim().length > 64) errs.push('invalidRealName')
  if (v.phone && !/^[0-9+\-\s]{3,32}$/.test(v.phone)) errs.push('invalidPhone')
  if (!v.roleCode) errs.push('roleRequired')
  return errs
}

/** 组装 POST /accounts 或 PUT /accounts/{id} 负荷:0=不限组织,空 regionScope=全集团。 */
export function buildAccountPayload(v: AccountFormValues, isEdit: boolean): Record<string, unknown> {
  const body: Record<string, unknown> = {
    username: v.username.trim(),
    realName: v.realName.trim(),
    phone: v.phone.trim(),
    roleCode: v.roleCode,
    legalEntityId: v.legalEntityId || null,
    deptId: v.deptId || null,
    postId: v.postId || null,
    regionScope: v.regionScope || null,
  }
  if (!isEdit || v.password !== '') body.password = v.password
  return body
}

/** 级联重置:公司变更清空部门/岗位;部门变更清空岗位。 */
export function cascadeReset(
  v: AccountFormValues,
  changed: 'legalEntityId' | 'deptId',
): AccountFormValues {
  if (changed === 'legalEntityId') {
    return { ...v, deptId: 0, postId: 0 }
  }
  return { ...v, postId: 0 }
}
