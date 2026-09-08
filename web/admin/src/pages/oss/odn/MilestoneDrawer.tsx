// 里程碑新增/编辑抽屉(双模式):面板内联表单的容器层重构,接口路径与载荷字段
// 与原内联表单一致(POST /odn/constructions/{id}/milestones、PUT /odn/milestones/{id})。
// 文案沿用施工详情面板字面量先例,日历结构性词条内联同源 zh 值。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { DatePicker } from '../../../components/DatePicker'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'

export interface Milestone {
  id: number
  projectId: number
  name: string
  plannedDate?: string
  status: string
  doneAt?: string
  createdAt: string
}

export type MilestoneDrawerTarget = { projectId: number; editing: Milestone | null }

const CAL_LABELS = {
  prevMonth: '上一月', nextMonth: '下一月', today: '今天', clear: '清除',
  weekdays: ['日', '一', '二', '三', '四', '五', '六'],
}

// initialMilestoneForm 抽屉表单初值(纯函数,便于回归):create 空表单;edit 回填行值,plannedDate 缺省补空。
export function initialMilestoneForm(editing: Milestone | null): { name: string; plannedDate: string } {
  return editing
    ? { name: editing.name, plannedDate: editing.plannedDate ?? '' }
    : { name: '', plannedDate: '' }
}

export function MilestoneDrawer({ target, onClose, onSaved }: {
  target: MilestoneDrawerTarget
  onClose: () => void
  onSaved: () => void
}) {
  const { projectId, editing } = target
  const isEdit = editing !== null
  const [form, setForm] = useState(initialMilestoneForm(editing))
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    setForm(initialMilestoneForm(editing))
    setError('')
  }, [target]) // eslint-disable-line react-hooks/exhaustive-deps

  const save = async () => {
    if (!form.name.trim()) { setError('里程碑名称必填'); return }
    setBusy(true); setError('')
    try {
      const body = { name: form.name.trim(), plannedDate: form.plannedDate.trim() }
      const path = editing
        ? '/odn/milestones/' + editing.id
        : '/odn/constructions/' + projectId + '/milestones'
      await apiFetch(path, { method: isEdit ? 'PUT' : 'POST', body })
      toast.success(isEdit ? '里程碑已保存' : '里程碑已追加')
      onSaved()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存失败'
      setError(msg)
      toast.error(isEdit ? '里程碑保存失败' : '里程碑追加失败', { description: msg })
    } finally { setBusy(false) }
  }

  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'

  return (
    <Drawer title={isEdit ? '编辑里程碑' : '追加里程碑'} onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>取消</ToolbarButton>
        <SubmitButton state={submitState} disabled={busy || !form.name.trim()} onClick={() => void save()}
          labels={{ idle: '保存', loading: '保存中…', success: isEdit ? '已保存' : '已追加', failed: '重试保存' }} />
      </>}>
      <div className="flex flex-col gap-3.5">
        {error && <ErrorBanner message={error} className="mx-0" />}
        <FormField label="名称" required>
          <Input value={form.name} placeholder="如 主干光缆敷设完成"
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
        </FormField>
        <FormField label="计划完成日" hint="可空">
          <DatePicker value={form.plannedDate} labels={CAL_LABELS} ariaLabel="计划完成日" placeholder="YYYY-MM-DD(可空)"
            onChange={(v) => setForm((f) => ({ ...f, plannedDate: v }))} />
        </FormField>
      </div>
    </Drawer>
  )
}
