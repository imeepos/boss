// 师傅选择器:/workers keyword 服务端检索;详情走 GET /workers/{id};管理页 /boss/worker。
import { useT } from '../../i18n'
import { EntityPicker } from './EntityPicker'
import { getWorkerDetail, searchWorkers, type WorkerItem } from '../../api/pickers'
import { workerDetailItems } from './detailItems'

export function WorkerPicker({ value, onChange, disabled }: {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}) {
  const p = useT().pages.pickers
  return (
    <EntityPicker<WorkerItem>
      value={value}
      onChange={onChange}
      disabled={disabled}
      search={searchWorkers}
      toOption={(w) => ({ value: String(w.id), label: `${w.name} · ${w.staffNo}` })}
      fetchDetail={() => getWorkerDetail(value)}
      clearable
      clearLabel={p.common.clear}
      detailItems={(w) => workerDetailItems(w, p.worker)}
      detailTitle={p.worker.title}
      listPath="/boss/worker"
      texts={{
        aria: p.worker.aria,
        placeholder: p.common.placeholder,
        loadFail: p.common.loadFail,
        viewDetail: p.common.viewDetail,
        detailFail: p.common.detailFail,
        close: p.common.close,
        jumpToList: p.common.jumpToList,
      }}
    />
  )
}
