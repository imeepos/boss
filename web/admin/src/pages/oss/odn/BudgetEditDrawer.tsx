// 预算编辑抽屉:预算卡内联表单的容器层重构,路径与载荷不变
// (PUT /odn/constructions/{id}/budget {budgetAmount});留空=NULL(未登记)与非负校验同原内联实现。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'

export function BudgetEditDrawer({ projectId, current, onClose, onSaved }: {
  projectId: number
  current: number | null | undefined
  onClose: () => void
  onSaved: () => void
}) {
  const [budget, setBudget] = useState(current != null ? String(current) : '')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const save = async () => {
    const raw = budget.trim()
    const v = raw === '' ? null : Number(raw)
    if (raw !== '' && (!Number.isFinite(v) || (v as number) < 0)) { setError('预算金额须为非负数或留空清除'); return }
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/constructions/' + projectId + '/budget', { method: 'PUT', body: { budgetAmount: v } })
      toast.success('预算已保存')
      onSaved()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存失败'
      setError(msg)
      toast.error('预算保存失败', { description: msg })
    } finally { setBusy(false) }
  }

  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'

  return (
    <Drawer title="编辑预算" onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>取消</ToolbarButton>
        <SubmitButton state={submitState} disabled={busy} onClick={() => void save()}
          labels={{ idle: '保存', loading: '保存中…', success: '已保存', failed: '重试保存' }} />
      </>}>
      <div className="flex flex-col gap-3.5">
        {error && <ErrorBanner message={error} className="mx-0" />}
        <FormField label="预算金额" hint="仅待开工期可改;留空=清除(未登记)">
          <Input value={budget} inputMode="decimal" placeholder="留空=清除"
            onChange={(e) => setBudget(e.target.value.replace(/[^0-9.]/g, ''))} />
        </FormField>
      </div>
    </Drawer>
  )
}
