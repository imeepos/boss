// Hero 右侧控制台模拟视图:纯图形骨架(无文字/无假数据),应用顶栏 + 侧栏 +
// KPI 卡片行 + 趋势折线图 + 状态环形图,颜色全部走 --home-mock-* 令牌随主题切换。

export function HeroConsoleMock() {
  return (
    <div className="relative h-full w-full" aria-hidden>
      <div className="absolute -right-10 -top-8 h-56 w-56 rounded-full bg-[var(--home-cta-glow)] blur-3xl" />
      <div className="absolute -bottom-12 -left-6 h-48 w-48 rounded-full bg-[var(--home-stats-glow)] blur-3xl" />
      <div className="relative flex h-full min-h-[400px] flex-col overflow-hidden rounded-2xl border border-[var(--home-mock-border)] bg-[var(--home-mock-bg)] shadow-2xl">
        <MockTopBar />
        <div className="flex min-h-0 flex-1">
          <MockSidebar />
          <div className="flex min-w-0 flex-1 flex-col gap-3 p-4">
            <KpiRow />
            <div className="flex min-h-0 flex-1 gap-3">
              <TrendPanel />
              <DonutPanel />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function MockTopBar() {
  return (
    <div className="flex h-11 flex-none items-center gap-2 border-b border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] px-4">
      <span className="h-5 w-5 rounded-md bg-gradient-to-br from-[var(--home-mock-bar-accent)] to-[var(--color-brand-gold-600)]" />
      <span className="h-2 w-20 rounded-full bg-[var(--home-mock-panel-2)]" />
      <span className="ml-auto h-6 w-28 rounded-full border border-[var(--home-mock-border)] bg-[var(--home-mock-bg)]" />
      <span className="h-3.5 w-3.5 rounded-full bg-[var(--home-mock-panel-2)]" />
      <span className="h-3.5 w-3.5 rounded-full bg-[var(--home-mock-panel-2)]" />
      <span className="h-6 w-6 rounded-full bg-[var(--home-mock-bar-accent)]/70" />
    </div>
  )
}

function MockSidebar() {
  return (
    <div className="flex w-16 flex-none flex-col items-center gap-3 border-r border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] py-4">
      <span className="flex h-8 w-10 items-center justify-center rounded-lg bg-[var(--home-gold-bg)]">
        <span className="h-4 w-4 rounded-sm bg-[var(--home-mock-bar-accent)]" />
      </span>
      {[0, 1, 2, 3, 4].map((i) => (
        <span key={i} className="flex h-8 w-10 items-center justify-center rounded-lg">
          <span className="h-4 w-4 rounded-sm bg-[var(--home-mock-panel-2)]" />
        </span>
      ))}
    </div>
  )
}

function KpiRow() {
  return (
    <div className="grid flex-none grid-cols-4 gap-3">
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="rounded-lg border border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] p-3">
          <div className="flex items-center gap-1.5">
            <span className="flex h-5 w-5 items-center justify-center rounded-md bg-[var(--home-gold-bg)]">
              <span className="h-2.5 w-2.5 rounded-sm bg-[var(--home-mock-bar-accent)]" />
            </span>
            <span className="h-1.5 w-8 rounded-full bg-[var(--home-mock-panel-2)]" />
          </div>
          <span className="mt-2.5 block h-3.5 w-14 rounded-md bg-[var(--home-mock-bar)]/60" />
          <div className="mt-2 flex items-center gap-1">
            <span className="h-1.5 w-5 rounded-full bg-[var(--home-case-fg)]/70" />
            <span className="h-1.5 flex-1 rounded-full bg-[var(--home-mock-grid)]" />
          </div>
        </div>
      ))}
    </div>
  )
}

function TrendPanel() {
  return (
    <div className="flex min-w-0 flex-1 flex-col rounded-lg border border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] p-3">
      <div className="flex flex-none items-center gap-2">
        <span className="h-2 w-16 rounded-full bg-[var(--home-mock-panel-2)]" />
        <span className="ml-auto flex items-center gap-1">
          <span className="h-1.5 w-1.5 rounded-full bg-[var(--home-mock-bar-accent)]" />
          <span className="h-1 w-6 rounded-full bg-[var(--home-mock-grid)]" />
        </span>
        <span className="flex items-center gap-1">
          <span className="h-1.5 w-1.5 rounded-full bg-[var(--home-mock-bar)]" />
          <span className="h-1 w-6 rounded-full bg-[var(--home-mock-grid)]" />
        </span>
      </div>
      <div className="relative mt-2 min-h-0 flex-1">
        <svg viewBox="0 0 300 120" preserveAspectRatio="none" className="absolute inset-0 h-full w-full">
          <defs>
            <linearGradient id="homeTrendFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--home-mock-bar-accent)" stopOpacity="0.28" />
              <stop offset="100%" stopColor="var(--home-mock-bar-accent)" stopOpacity="0" />
            </linearGradient>
          </defs>
          {[24, 48, 72, 96].map((y) => (
            <line key={y} x1="0" y1={y} x2="300" y2={y} stroke="var(--home-mock-grid)" strokeWidth="1" />
          ))}
          <path
            d="M0 92 L40 80 L80 86 L120 66 L160 72 L200 52 L240 58 L300 34 L300 120 L0 120 Z"
            fill="url(#homeTrendFill)" stroke="none"
          />
          <path
            d="M0 92 L40 80 L80 86 L120 66 L160 72 L200 52 L240 58 L300 34"
            fill="none" stroke="var(--home-mock-line)" strokeWidth="2.5" strokeLinecap="round"
          />
          <path
            d="M0 100 L40 96 L80 100 L120 90 L160 94 L200 82 L240 86 L300 70"
            fill="none" stroke="var(--home-mock-bar)" strokeWidth="2" strokeLinecap="round" opacity="0.7"
          />
          <circle cx="240" cy="58" r="3.5" fill="var(--home-mock-bar-accent)" />
        </svg>
      </div>
    </div>
  )
}

const DONUT_R = 28
const DONUT_C = 2 * Math.PI * DONUT_R
// 已完成/处理中/待处理/已取消 四段占比 45/28/17/10。
const DONUT_SEGMENTS = [
  { frac: 0.45, color: 'var(--home-mock-bar-accent)' },
  { frac: 0.28, color: 'var(--home-mock-bar)' },
  { frac: 0.17, color: 'var(--home-case-fg)' },
  { frac: 0.10, color: 'var(--home-mock-panel-2)' },
]

function DonutPanel() {
  let acc = 0
  return (
    <div className="flex w-36 flex-none flex-col rounded-lg border border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] p-3">
      <span className="h-2 w-14 flex-none rounded-full bg-[var(--home-mock-panel-2)]" />
      <div className="flex min-h-0 flex-1 items-center justify-center py-2">
        <svg viewBox="0 0 72 72" className="h-full max-h-28 w-auto">
          {DONUT_SEGMENTS.map((s, i) => {
            const offset = -acc * DONUT_C
            acc += s.frac
            return (
              <circle
                key={i} cx="36" cy="36" r={DONUT_R} fill="none"
                stroke={s.color} strokeWidth="10"
                strokeDasharray={`${s.frac * DONUT_C} ${DONUT_C}`}
                strokeDashoffset={offset}
                transform="rotate(-90 36 36)"
              />
            )
          })}
        </svg>
      </div>
      <div className="flex flex-none flex-col gap-1.5">
        {DONUT_SEGMENTS.map((s, i) => (
          <span key={i} className="flex items-center gap-1.5">
            <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: s.color }} />
            <span className="h-1 flex-1 rounded-full bg-[var(--home-mock-grid)]" />
          </span>
        ))}
      </div>
    </div>
  )
}
