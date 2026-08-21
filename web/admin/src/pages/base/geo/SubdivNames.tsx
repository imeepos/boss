// SubdivNames 区划译名维护抽屉:列表 + locale/name 新增 + 删除确认。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TAG_OFF, type NameRow } from './subdiv-shared'

export function SubdivNames({ code, onClose }: { code: string; onClose: () => void }) {
  const t = useT()
  const g = t.pages.geo
  const confirmDialog = useConfirm()
  const [names, setNames] = useState<NameRow[]>([])
  const [locale, setLocale] = useState('zh-Hans')
  const [name, setName] = useState('')

  const load = useCallback(() => {
    apiFetch<NameRow[]>(`/geo/subdivisions/${code}/names`)
      .then((d) => setNames(d ?? []))
      .catch(() => setNames([]))
  }, [code])

  useEffect(load, [load])

  const add = async () => {
    if (!name.trim()) return
    await apiFetch(`/geo/subdivisions/${code}/names`, {
      method: 'POST', body: { locale, name, nameType: 'STANDARD' },
    }).catch(() => undefined)
    setName('')
    load()
  }

  const remove = async (loc: string, nameType: string) => {
    if (!(await confirmDialog(g.deleteNameConfirm, { danger: true }))) return
    await apiFetch(`/geo/subdivisions/${code}/names/${loc}/${nameType}`, { method: 'DELETE' })
    load()
  }

  return (
    <Drawer title={`${g.names} · ${code}`} onClose={onClose}>
      {names.map((n) => (
        <div key={n.locale + n.nameType} className={TAG_OFF}>
          <span>{n.locale} · {n.nameType} · {n.name}</span>
          <button className="cursor-pointer border-none bg-none text-[var(--color-danger)]"
            onClick={() => remove(n.locale, n.nameType)}>×</button>
        </div>
      ))}
      <div className="mt-3 flex gap-2">
        <Input className="w-27" value={locale}
          onChange={(e) => setLocale(e.target.value)} placeholder="locale" />
        <Input value={name}
          onChange={(e) => setName(e.target.value)} placeholder="name" />
        <ToolbarButton onClick={add}>{g.addName}</ToolbarButton>
      </div>
    </Drawer>
  )
}
