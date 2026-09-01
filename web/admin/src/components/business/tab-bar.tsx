// TabBar: 页内平铺页签(顶边高亮)。收敛 6 处内联 style 页签(设计系统 §2.1)。
// extra: 页签行右侧插槽(antd Tabs tabBarExtraContent 语义),常放刷新/新建等动作。
import type { ReactNode } from 'react'

interface TabItem<K extends string> {
  key: K
  label: string
}

export function TabBar<K extends string>({ tabs, value, onChange, extra }: {
  tabs: TabItem<K>[]
  value: K
  onChange: (key: K) => void
  extra?: ReactNode
}) {
  return (
    <div
      className="flex items-center gap-1 border-b border-[var(--shell-side-border)]"
      style={{ marginBottom: 12 }}
      role="tablist"
    >
      <div className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto">
        {tabs.map((t) => {
          const active = t.key === value
          return (
            <button
              key={t.key}
              role="tab"
              aria-selected={active}
              onClick={() => onChange(t.key)}
              className="cursor-pointer border-none bg-none px-4 py-2 text-sm"
              style={{
                borderBottom: active ? '2px solid var(--color-border-focus)' : '2px solid transparent',
                color: active ? 'var(--color-border-focus)' : 'var(--shell-group-title)',
                fontWeight: active ? 600 : 400,
              }}
            >
              {t.label}
            </button>
          )
        })}
      </div>
      {extra != null ? <div className="shrink-0 pb-1">{extra}</div> : null}
    </div>
  )
}
