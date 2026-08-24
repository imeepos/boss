// 客户端下载区:公开接口 /site/downloads 拉双端最新 PUBLISHED,失败/为空整块隐藏
// (与 NewsSection 同策略,不为未发版留空区)。
import { useEffect, useState } from 'react'
import { apiFetch, apiBaseUrl } from '../../api/client'
import { SectionHead } from './Features'

interface DownloadItem { app: string; version: string; sha256: string; size: number; downloadPath: string }

export interface DownloadSectionProps {
  title: string
  subtitle: string
  userApp: string
  workerApp: string
  userDesc: string
  workerDesc: string
  button: string
  shaPrefix: string
}

const APP_LABEL: Record<string, { name: string; desc: string; glyph: string }> = {
  user: { name: '', desc: '', glyph: 'M' },
  worker: { name: '', desc: '', glyph: 'W' },
}

export function DownloadSection(p: DownloadSectionProps) {
  const [items, setItems] = useState<DownloadItem[]>([])
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let alive = true
    apiFetch<{ items: DownloadItem[] }>('/site/downloads')
      .then((d) => { if (alive) setItems(d?.items ?? []) })
      .catch(() => { if (alive) setFailed(true) })
    return () => { alive = false }
  }, [])

  if (failed || !items.length) return null
  APP_LABEL.user = { name: p.userApp, desc: p.userDesc, glyph: 'C' }
  APP_LABEL.worker = { name: p.workerApp, desc: p.workerDesc, glyph: 'S' }

  const CARD = 'flex flex-col gap-3 rounded-xl border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-7 shadow-[var(--shell-card-shadow)] transition-all hover:-translate-y-1 hover:border-[var(--home-hero-badge-border)] hover:shadow-lg'
  const BADGE = 'flex h-10 w-10 items-center justify-center rounded-full border border-[var(--home-hero-badge-border)] bg-[var(--home-hero-badge-bg)] font-brand text-base font-semibold text-[var(--home-gold-fg)]'
  const BTN = 'inline-flex h-9 items-center justify-center rounded-md bg-[var(--color-brand-solid)] px-5 text-[13px] font-medium text-white transition-opacity hover:opacity-90'

  return (
    <section id="download">
      <div className="mx-auto max-w-6xl scroll-mt-20 px-4 py-16">
        <SectionHead title={p.title} subtitle={p.subtitle} />
        <div className="mt-12 grid gap-6 md:grid-cols-2">
          {items.map((d) => {
            const meta = APP_LABEL[d.app] ?? APP_LABEL.user
            const sizeMb = d.size > 0 ? `${(d.size / 1048576).toFixed(1)}MB` : ''
            return (
              <div key={d.app} className={CARD}>
                <div className="flex items-center gap-4">
                  <span className={BADGE}>{meta.glyph}</span>
                  <div>
                    <h3 className="font-brand text-base font-semibold tracking-tight text-[var(--shell-heading)]">{meta.name}</h3>
                    <p className="text-xs text-[var(--shell-group-title)]">v{d.version}{sizeMb ? ` · ${sizeMb}` : ''} · Android</p>
                  </div>
                </div>
                <p className="text-sm leading-6 text-[var(--shell-content-text)]">{meta.desc}</p>
                <div className="mt-auto flex items-center justify-between gap-3 pt-2">
                  <span className="truncate font-mono text-xs text-[var(--shell-group-title)]" title={d.sha256}>{p.shaPrefix}{d.sha256.slice(0, 12)}</span>
                  <a className={BTN} href={`${apiBaseUrl()}${d.downloadPath}`} download>{p.button}</a>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
