// 换新单操作显隐:取消仅对 PENDING 开放(契约 POST /replacements/{id}/cancel;
// 状态机 adopted note 2026-08-27-replacement-ticket-flow:终态不可流转,DOING 起不可取消)。
export const canCancelReplacement = (status: string): boolean => status === 'PENDING'

export function cancelPath(id: number): string {
  return '/replacements/' + id + '/cancel'
}
