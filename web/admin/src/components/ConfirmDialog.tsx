// 全局确认弹窗:替代原生 window.confirm,基于 Radix Dialog + 项目 shell 令牌。
import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog'
import { useT } from '../i18n'

export interface ConfirmOptions {
  message: string
  title?: string
  danger?: boolean
}

type ConfirmFn = (message: string, opts?: Omit<ConfirmOptions, 'message'>) => Promise<boolean>

const ConfirmCtx = createContext<ConfirmFn | null>(null)

/** 页面内用法:const confirmDialog = useConfirm(); if (!(await confirmDialog(msg))) return */
export function useConfirm(): ConfirmFn {
  const fn = useContext(ConfirmCtx)
  if (!fn) throw new Error('useConfirm 必须在 ConfirmProvider 内使用')
  return fn
}

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const t = useT()
  const c = t.common.confirmDialog
  const [opts, setOpts] = useState<ConfirmOptions | null>(null)
  const resolver = useRef<((v: boolean) => void) | null>(null)

  const close = useCallback((v: boolean) => {
    resolver.current?.(v)
    resolver.current = null
    setOpts(null)
  }, [])

  const confirm = useCallback<ConfirmFn>((message, extra) => {
    // 同一时刻只保留最新一次请求,旧请求直接视为取消
    resolver.current?.(false)
    return new Promise<boolean>((resolve) => {
      resolver.current = resolve
      setOpts({ message, ...extra })
    })
  }, [])

  return (
    <ConfirmCtx.Provider value={confirm}>
      {children}
      <Dialog open={!!opts} onOpenChange={(open) => { if (!open) close(false) }}>
        <DialogContent className="max-w-sm border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-0">
          <DialogHeader className="space-y-0 border-b border-[var(--shell-side-border)] px-5 py-4">
            <DialogTitle className="text-[15px] font-semibold text-[var(--shell-heading)]">
              {opts?.title ?? c.title}
            </DialogTitle>
          </DialogHeader>
          <div className="px-5 py-4">
            <DialogDescription className="text-[13px] leading-6 whitespace-pre-line text-[var(--shell-content-text)]">
              {opts?.message}
            </DialogDescription>
          </div>
          <DialogFooter className="gap-2 border-t border-[var(--shell-side-border)] px-5 py-3 sm:space-x-0">
            <button
              className="h-8 min-w-20 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
              onClick={() => close(false)}
            >
              {c.cancel}
            </button>
            <button
              className={`h-8 min-w-20 cursor-pointer rounded-sm border-none px-4 text-[13px] ${opts?.danger
                ? 'bg-[var(--color-danger)] text-white hover:opacity-90'
                : 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]'}`}
              onClick={() => close(true)}
            >
              {c.ok}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ConfirmCtx.Provider>
  )
}
