// 官网新闻/文章详情页(公开路由 /news/:slug):匿名读已发布内容(按界面语言,
// 后端缺变体回退默认语言),Markdown 安全渲染(react-markdown,无 dangerouslySetInnerHTML);
// 非已发布 404 → 友好空态。
import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import { apiFetch, apiBaseUrl } from '../../api/client'
import { setDocMeta } from '../../lib/docMeta'
import { useT, useLang } from '../../i18n'
import { useTheme } from '../../theme/context'
import { TopNav } from '../home/TopNav'
import { Footer } from '../home/Footer'
import '../home/home.css'

interface PostDTO {
  slug: string; title: string; category: string; categoryName?: string
  summary: string; coverAttachmentId: number; content: string; publishedAt: string
}

export default function NewsDetailPage() {
  const t = useT(); const h = t.pages.home
  const { slug } = useParams()
  const nav = useNavigate()
  const { locale, setLocale } = useLang()
  const { theme, toggleTheme } = useTheme()
  const [post, setPost] = useState<PostDTO | null>(null)
  const [miss, setMiss] = useState(false)

  useEffect(() => {
    let alive = true
    apiFetch<PostDTO>(`/site/posts/${slug}`, { query: { lang: locale } })
      .then((d) => { if (alive) setPost(d) })
      .catch(() => { if (alive) setMiss(true) })
    window.scrollTo(0, 0)
    return () => { alive = false }
  }, [slug, locale])

  // SEO 元数据:title/description/og 分享卡;封面走公开流绝对地址(带 lang 取同变体封面)。
  useEffect(() => {
    if (!post) return
    const base = apiBaseUrl().replace(/\/api\/admin\/v1$/, '')
    const img = post.coverAttachmentId > 0 ? `${base}/api/admin/v1/site/posts/${post.slug}/cover?lang=${locale}` : undefined
    return setDocMeta(post.title, {
      description: post.summary || post.title,
      'og:title': post.title,
      'og:description': post.summary || post.title,
      'og:type': 'article',
      ...(img ? { 'og:image': img } : {}),
    })
  }, [post, locale])

  const MD_H = 'font-brand font-semibold tracking-tight text-[var(--shell-heading)] mt-8 mb-3'
  const md = {
    h1: (p: object) => <h1 {...p} className={MD_H + ' text-2xl'} />,
    h2: (p: object) => <h2 {...p} className={MD_H + ' text-xl'} />,
    h3: (p: object) => <h3 {...p} className={MD_H + ' text-lg'} />,
    p: (p: object) => <p {...p} className="mb-4 text-[15px] leading-7 text-[var(--shell-content-text)]" />,
    ul: (p: object) => <ul {...p} className="mb-4 list-disc pl-6 text-[15px] leading-7 text-[var(--shell-content-text)]" />,
    ol: (p: object) => <ol {...p} className="mb-4 list-decimal pl-6 text-[15px] leading-7 text-[var(--shell-content-text)]" />,
    a: (p: object) => <a {...p} className="text-[var(--home-gold-fg)] underline underline-offset-2" />,
    code: (p: object) => <code {...p} className="rounded bg-[var(--shell-input-bg)] px-1.5 py-0.5 text-[13px]" />,
    blockquote: (p: object) => <blockquote {...p} className="my-4 border-l-2 border-[var(--home-hero-badge-border)] pl-4 text-[var(--shell-group-title)]" />,
  }

  return (
    <div className="min-h-screen bg-[var(--shell-content-bg)]">
      <TopNav
        locale={locale} setLocale={(v) => setLocale(v as typeof locale)}
        theme={theme} toggleTheme={toggleTheme}
        ctaLabel={h.bookDemo} onCtaClick={() => nav('/login')}
        t={{ heroBadge: h.heroBadge, navFeatures: h.navFeatures, navSolutions: h.navSolutions, navContact: h.navContact }}
        shellT={t.shell}
      />
      <main className="mx-auto max-w-3xl px-4 py-14">
        {post ? (
          <article>
            <div className="mb-3 flex items-center gap-3">
              <span className="flex h-6 items-center rounded-full border border-[var(--home-hero-badge-border)] bg-[var(--home-hero-badge-bg)] px-3 text-xs font-medium text-[var(--home-gold-fg)]">
                {post.categoryName || (post.category === 'ARTICLE' ? h.newsCatArticle : h.newsCatNews)}
              </span>
              <time className="text-xs text-[var(--shell-group-title)]">{post.publishedAt}</time>
            </div>
            <h1 className="font-brand text-3xl font-bold tracking-tight text-[var(--shell-heading)]">{post.title}</h1>
            {post.coverAttachmentId > 0 && (
              <img className="mt-6 w-full rounded-xl border border-[var(--shell-card-border)] object-cover" src={`${apiBaseUrl()}/site/posts/${post.slug}/cover?lang=${locale}`} alt={post.title} />
            )}
            <div className="mt-6">
              <ReactMarkdown components={md}>{post.content}</ReactMarkdown>
            </div>
          </article>
        ) : miss ? (
          <div className="py-24 text-center">
            <p className="text-sm text-[var(--shell-group-title)]">{h.newsNotFound}</p>
            <Link className="mt-4 inline-block text-sm text-[var(--home-gold-fg)] underline underline-offset-2" to="/home">{h.newsBack}</Link>
          </div>
        ) : (
          <div className="py-24 text-center text-sm text-[var(--shell-group-title)]">{t.common.loading}</div>
        )}
      </main>
      <Footer
        t={{
          tagline: h.footerTagline, taglineDesc: h.footerTaglineDesc,
          productTitle: h.footerProductTitle, solutionsTitle: h.footerSolutionsTitle,
          quickTitle: h.footerQuickTitle, navContact: h.navContact,
          bookDemo: h.bookDemo, login: h.login, copyright: h.footerCopyright,
        }}
        featureLinks={h.features.slice(0, 3)}
        caseLinks={h.cases}
      />
    </div>
  )
}
