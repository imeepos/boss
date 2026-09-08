// 新建勘测任务抽屉(W7 勘测页签):与全站表单口径一致——右侧 Drawer + FormField,
// 枚举走 Dropdown(禁原生 select);师傅列表由面板一次拉取经 props 注入,不重复请求。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'

export interface WorkerLite { id: number; name: string }

export function SurveyCreateDrawer({ workers, onClose, onCreated }: {
  workers: WorkerLite[]
  onClose: () => void
  onCreated: () => void
}) {
  const [title, setTitle] = useState('')
  const [desc, setDesc] = useState('')
  const [grid, setGrid] = useState('')
  const [assignee, setAssignee] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const save = async () => {
    if (!title.trim()) { setError('任务标题必填'); return }
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/surveys', { method: 'POST', body: {
        title: title.trim(), description: desc.trim(), gridCode: Number(grid) || 0,
        assignedWorkerId: Number(assignee) || 0 } })
      toast.success('勘测任务已创建')
      onCreated()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存失败'
      setError(msg)
      toast.error('勘测任务创建失败', { description: msg })
    } finally { setBusy(false) }
  }
  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'

  return (
    <Drawer title='新建勘测任务' onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>取消</ToolbarButton>
        <SubmitButton state={submitState} disabled={busy || !title.trim()} onClick={() => void save()}
          labels={{ idle: '保存', loading: '保存中…', success: '已创建', failed: '重试保存' }} />
      </>}>
      <div className='flex flex-col gap-3.5'>
        {error && <ErrorBanner message={error} className='mx-0' />}
        <FormField label='任务标题' required>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="主干光缆段现场勘测" />
        </FormField>
        <FormField label='任务说明'>
          <Input value={desc} onChange={(e) => setDesc(e.target.value)} placeholder="目标区域/网格、勘测要点" />
        </FormField>
        <FormField label='目标网格'>
          <Input value={grid} onChange={(e) => setGrid(e.target.value)} inputMode="numeric" placeholder="0=不限" />
        </FormField>
        <FormField label='指派师傅'>
          <Dropdown value={assignee} ariaLabel="选择指派师傅" placeholder="不指派(进抢单池)" searchable searchPlaceholder="搜索师傅"
            options={workers.map((w) => ({ value: String(w.id), label: w.name + ' (#' + w.id + ')' }))}
            onChange={(v) => setAssignee(v)} />
        </FormField>
      </div>
    </Drawer>
  )
}
