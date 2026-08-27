// 动态与新闻区块:从公开接口 /site/posts 拉最新已发布内容(按界面语言 lang 过滤,
// 后端缺变体回退默认语言),失败/为空时整块隐藏(官网首页不为 CMS 未使用而留空白区)。
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiFetch, apiBaseUrl } from '../../api/client'
import { useLang } from '../../i18n'
import { SectionHead } from './Features'

interface NewsItem {
  slug: string; title: string; category: string; categoryName?: string
  summary: string; coverAttachmentId: number; publishedAt: string
}

export interface NewsSectionProps {
  title: string
  subtitle: string
  empty: string
  catNews: string
  catArticle: string
}

export function NewsSection(p: NewsSectionProps) {
  const { locale } = useLang()
  const [items, setItems] = useState<NewsItem[]>([])
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let alive = true
    apiFetch<{ items: NewsItem[] }>('/site/posts', { query: { limit: 3, lang: locale } })
      .then((d) => { if (alive) setItems(d?.items ?? []) })
      .catch(() => { if (alive) setFailed(true) })
    return () => { alive = false }
  }, [locale])

  if (failed || !items.length) return null
  const catLabel = (n: NewsItem) => n.categoryName || (n.category === 'ARTICLE' ? p.catArticle : p.catNews)
  const CARD = 'flex flex-col gap-3 rounded-xl border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-7 shadow-[var(--shell-card-shadow)] transition-all hover:-translate-y-1 hover:border-[var(--home-hero-badge-border)] hover:shadow-lg'
  const TAG =
    'flex h-6 items-center rounded-full border border-[var(--home-hero-badge-border)] bg-[var(--home-hero-badge-bg)] px-3 text-xs font-medium text-[var(--home-gold-fg)]'
  return (
    <section id="news">
      <div className="mx-auto max-w-6xl scroll-mt-20 px-4 py-16">
        <SectionHead title={p.title} subtitle={p.subtitle} />
        <div className="mt-12 grid gap-6 lg:grid-cols-3">
          {items.map((n) => (
            <Link key={n.slug} to={`/news/${n.slug}`} className={CARD}>
              {n.coverAttachmentId > 0 && (
                <img className="h-36 w-full rounded-lg border border-[var(--shell-card-border)] object-cover" src={`${apiBaseUrl()}/site/posts/${n.slug}/cover?lang=${locale}`} alt={n.title} />
              )}
              <div className="flex items-center justify-between gap-3">
                <span className={TAG}>{catLabel(n)}</span>
                <time className="flex-none text-xs text-[var(--shell-group-title)]">{n.publishedAt}</time>
              </div>
              <h3 className="font-brand text-base font-semibold tracking-tight text-[var(--shell-heading)]">{n.title}</h3>
              <p className="line-clamp-3 text-sm leading-6 text-[var(--shell-content-text)]">{n.summary || n.title}</p>
            </Link>
          ))}
        </div>
      </div>
    </section>
  )
}
