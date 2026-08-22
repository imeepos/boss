// Hero 右侧控制台模拟视图:纯图形(无文字/无假数据),窗口卡片 + 侧栏色块 +
// KPI 骨架 + 柱状图/折线图,颜色全部走 --home-mock-* 令牌随主题切换。
export function HeroConsoleMock() {
  return (
    <div className="relative h-full w-full" aria-hidden>
      <div className="absolute right-[-40px] top-[-30px] h-56 w-56 rounded-full bg-[var(--home-cta-glow)] blur-3xl" />
      <div className="relative flex h-full min-h-[340px] flex-col overflow-hidden rounded-xl border border-[var(--home-mock-border)] bg-[var(--home-mock-bg)] shadow-2xl">
        <WindowChrome />
        <div className="flex min-h-0 flex-1">
          <MockSidebar />
          <div className="flex min-w-0 flex-1 flex-col gap-3 p-4">
            <KpiRow />
            <BarChart />
            <LineChart />
          </div>
        </div>
      </div>
    </div>
  )
}

function WindowChrome() {
  return (
    <div className="flex h-9 flex-none items-center gap-1.5 border-b border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] px-3">
      <span className="h-2.5 w-2.5 rounded-full bg-[var(--home-mock-dot-r)]" />
      <span className="h-2.5 w-2.5 rounded-full bg-[var(--home-mock-dot-y)]" />
      <span className="h-2.5 w-2.5 rounded-full bg-[var(--home-mock-dot-g)]" />
      <span className="ml-3 h-2.5 w-24 rounded-full bg-[var(--home-mock-panel-2)]" />
    </div>
  )
}

function MockSidebar() {
  return (
    <div className="flex w-14 flex-none flex-col items-center gap-3 border-r border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] py-4">
      <span className="h-7 w-7 rounded-md bg-[var(--home-mock-bar-accent)]/80" />
      {[0, 1, 2, 3, 4].map((i) => (
        <span key={i} className="h-6 w-6 rounded-md bg-[var(--home-mock-panel-2)]" />
      ))}
    </div>
  )
}

function KpiRow() {
  return (
    <div className="grid flex-none grid-cols-3 gap-3">
      {[0, 1, 2].map((i) => (
        <div key={i} className="rounded-lg border border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] p-3">
          <span className="block h-2 w-10 rounded-full bg-[var(--home-mock-panel-2)]" />
          <span className="mt-2.5 block h-4 w-14 rounded-md bg-[var(--home-mock-bar)]/50" />
          <div className="mt-2 flex items-end gap-1">
            <span className="block h-1.5 flex-1 rounded-full bg-[var(--home-mock-grid)]" />
            <span className="block h-1.5 w-6 rounded-full bg-[var(--home-mock-bar-accent)]/70" />
          </div>
        </div>
      ))}
    </div>
  )
}

const BARS = [42, 68, 51, 84, 60, 95, 74]

function BarChart() {
  return (
    <div className="flex min-h-0 flex-1 items-end gap-2 rounded-lg border border-[var(--home-mock-border)] bg-[var(--home-mock-panel)] p-3">
      <div className="flex h-full w-2/3 items-end gap-2">
        {BARS.map((h, i) => (
          <span
            key={i}
            style={{ height: `${h}%` }}
            className={'flex-1 rounded-t-sm ' + (i === 5 ? 'bg-[var(--home-mock-bar-accent)]' : 'bg-[var(--home-mock-bar)]/60')}
          />
        ))}
      </div>
      <div className="flex h-full flex-1 flex-col justify-between py-1 pl-2">
        <span className="h-1.5 w-full rounded-full bg-[var(--home-mock-grid)]" />
        <span className="h-1.5 w-3/4 rounded-full bg-[var(--home-mock-grid)]" />
        <span className="h-1.5 w-full rounded-full bg-[var(--home-mock-grid)]" />
      </div>
    </div>
  )
}

function LineChart() {
  return (
    <div className="relative h-20 flex-none overflow-hidden rounded-lg border border-[var(--home-mock-border)] bg-[var(--home-mock-panel)]">
      <svg viewBox="0 0 300 80" preserveAspectRatio="none" className="absolute inset-0 h-full w-full">
        <path d="M0 62 L40 54 L80 58 L120 44 L160 48 L200 34 L240 38 L300 20" fill="none" stroke="var(--home-mock-grid)" strokeWidth="2" />
        <path
          d="M0 66 L40 60 L80 50 L120 52 L160 38 L200 40 L240 26 L300 12"
          fill="none"
          stroke="var(--home-mock-line)"
          strokeWidth="2.5"
          strokeLinecap="round"
        />
        <circle cx="240" cy="26" r="3.5" fill="var(--home-mock-bar-accent)" />
      </svg>
    </div>
  )
}
