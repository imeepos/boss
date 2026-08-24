// 动态与新闻区块:从公开接口 /site/posts 拉最新已发布内容,失败/为空时整块隐藏
// (官网首页不为 CMS 未使用而留空白区)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../api/client'
import { SectionHead } from './Features'

interface NewsItem { slug: string; title: string; category: string; summary: string; publishedAt: string }

export interface NewsSectionProps {
  title: string
  subtitle: string
  empty: string
  catNews: string
  catArticle: string
}

export function NewsSection(p: NewsSectionProps) {
  const [items, setItems] = useState<NewsItem[]>([])
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let alive = true
    apiFetch<{ items: NewsItem[] }>('/site/posts', { query: { limit: 3 } })
      .then((d) => { if (alive) setItems(d?.items ?? []) })
      .catch(() => { if (alive) setFailed(true) })
    return () => { alive = false }
  }, [])

  if (failed || !items.length) return null
  const CARD = 'flex flex-col gap-3 rounded-xl border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-7 shadow-[var(--shell-card-shadow)] transition-all hover:-translate-y-1 hover:border-[var(--home-hero-badge-border)] hover:shadow-lg'
  const TAG =
    'flex h-6 items-center rounded-full border border-[var(--home-hero-badge-border)] bg-[var(--home-hero-badge-bg)] px-3 text-xs font-medium text-[var(--home-gold-fg)]'
  return (
    <section id="news">
      <div className="mx-auto max-w-6xl scroll-mt-20 px-4 py-16">
        <SectionHead title={p.title} subtitle={p.subtitle} />
        <div className="mt-12 grid gap-6 lg:grid-cols-3">
          {items.map((n) => (
            <article key={n.slug} className={CARD}>
              <div className="flex items-center justify-between gap-3">
                <span className={TAG}>{n.category === 'ARTICLE' ? p.catArticle : p.catNews}</span>
                <time className="flex-none text-xs text-[var(--shell-group-title)]">{n.publishedAt}</time>
              </div>
              <h3 className="font-brand text-base font-semibold tracking-tight text-[var(--shell-heading)]">{n.title}</h3>
              <p className="line-clamp-3 text-sm leading-6 text-[var(--shell-content-text)]">{n.summary || n.title}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}
