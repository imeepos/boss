// 重置密码结果弹层:随机明文一次性展示(后端仅本次响应返回,库内只有密文)+ 复制 + 语义提示。
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../../../components/ui/dialog'
import { CopyButton } from '../../../components/business'
import { useT } from '../../../i18n'

interface ResetPasswordDialogProps {
  loid: string
  password: string
  onClose: () => void
}

export function ResetPasswordDialog({ loid, password, onClose }: ResetPasswordDialogProps) {
  const t = useT()
  const l = t.pages.loAccountPage
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-sm border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-0" data-testid="reset-password-dialog">
        <DialogHeader className="space-y-0 border-b border-[var(--shell-side-border)] px-5 py-4">
          <DialogTitle className="text-[15px] font-semibold text-[var(--shell-heading)]">{l.resetTitle}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 px-5 py-4">
          <DialogDescription className="text-[13px] leading-6 text-[var(--shell-content-text)]">
            <span className="font-medium text-[var(--shell-heading)]">{loid}</span>
          </DialogDescription>
          <div className="flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-2 text-[13px] tracking-wide text-[var(--shell-heading)]" data-testid="reset-password-value">{password}</code>
            <CopyButton text={password} className="shrink-0" />
          </div>
          <p className="text-[12px] leading-5 text-[var(--shell-group-title)]">{l.resetOnceHint}</p>
        </div>
        <DialogFooter className="gap-2 border-t border-[var(--shell-side-border)] px-5 py-3 sm:space-x-0">
          <button
            type="button"
            className="h-8 min-w-20 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
            onClick={onClose}
          >{l.resetClose}</button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
