// 个人工作台各分区共享的样式常量与展示型小组件。
// 样式:tailwind 原子类,令牌走 shell-* 体系,双主题自动切换。
// 重合件复用设计系统(page-patterns.md):SectionTitle=PageHead 包装。
import { Link } from 'react-router-dom'
import { PageHead } from '../../components/business/page-head'

export const PAGE = 'border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-6 shadow-[var(--shell-card-shadow)] md:p-8'
export const BOX = 'grid gap-[7px] border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-4.5'
export const AVATAR = 'grid h-[52px] w-[52px] flex-none place-items-center rounded-full bg-[var(--color-brand-gold-300)] text-[21px] font-semibold text-[var(--color-brand-navy-950)]'
export const CHEVRON = 'absolute right-4 top-10 inline-block h-2 w-2 rotate-45 border-t-[1.5px] border-r-[1.5px] border-current text-[var(--color-brand-gold-600)]'
export const FORM_LABEL = 'grid gap-[7px] text-[13px] text-[var(--shell-content-text)]'
export const READONLY_INPUT = 'read-only:bg-[var(--shell-input-disabled-bg)] read-only:text-[var(--shell-crumb-text)]'
export const MSG = (ok: boolean) => ({ color: ok ? 'var(--color-success)' : 'var(--color-danger)' })
export const TIP = 'mt-4 bg-[var(--shell-menu-hover-bg)] px-3.5 py-2.5 text-xs leading-relaxed text-[var(--shell-content-text)]'
export const LIST_BTN = 'flex w-full min-h-16 cursor-pointer items-center justify-between border-0 border-b border-[var(--shell-side-border)] bg-transparent px-1 py-3 text-left text-[var(--shell-heading)] hover:text-[var(--color-brand-gold-600)]'

export function SectionTitle({ title, desc }: { title: string; desc: string }) {
  return <PageHead title={title} desc={desc} />
}

export function OverviewLink({ title, desc, href }: { title: string; desc: string; href: string }) {
  return (
    <Link className="relative grid min-h-[92px] gap-[7px] border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-4 text-[var(--shell-heading)] no-underline hover:border-[var(--color-brand-gold-500)] hover:bg-[var(--shell-menu-hover-bg)]" to={href}>
      <strong className="text-sm font-semibold">{title}</strong><span className="text-xs leading-relaxed text-[var(--shell-content-text)]">{desc}</span><i className={CHEVRON} />
    </Link>
  )
}

export function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return <div className="grid gap-1.5"><span className="text-xs text-[var(--shell-content-text)]">{label}</span><strong className="text-[13px] font-medium text-[var(--shell-heading)]">{value}</strong></div>
}

export function SecurityRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex min-h-12 items-center gap-3.5 border-b border-[var(--shell-side-border)] text-[13px] text-[var(--shell-heading)]">
      <span>{label}</span><em className="ml-auto text-xs font-normal text-[var(--shell-crumb-text)]">{value}</em>
      <button aria-label={label} className="h-7 w-7 cursor-pointer border-0 bg-transparent p-0 text-[var(--shell-crumb-text)]"><i className={CHEVRON + ' static'} /></button>
    </div>
  )
}

export function Summary({ value, label }: { value: string; label: string }) {
  return (
    <div className="border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-4">
      <strong className="block text-2xl font-semibold text-[var(--shell-heading)]">{value}</strong>
      <span className="mt-1.5 block text-xs text-[var(--shell-content-text)]">{label}</span>
    </div>
  )
}
