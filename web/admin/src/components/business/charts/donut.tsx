// 占比环形:多段 N 等分,SVG <circle stroke-dasharray> 实现,中心显示总数。
// 颜色按 --color-text-link 金 → --color-brand-navy-950 深蓝 → 灰阶 6 段。
// 顶部 label + 副标 + legend 格式化值都通过 formatValue 回调传入(对齐 commit F1 format-value.ts)。

const PALETTE = ['#C69835', '#1F355F', '#BE8D25', '#273F70', '#9CAAC3', '#A6B1C3']

export interface DonutSegment { label: string; value: number }

export function Donut({
  segments, total, label = '', sublabel, formatValue, formatTotal,
}: {
  segments: DonutSegment[]
  /** 显式给定时覆盖 sum,避免精度问题 */
  total?: number
  /** 顶部主标(默认 "指标构成");短文案,1-6 字 */
  label?: string
  /** 顶部副标(可选,如 "x 项"),fontSize 11 */
  sublabel?: string
  /** 单段值的格式化函数(返回字符串);不传则显示原始浮点。
   * 第二参 segment 便于业务页用 closure 把 indicator/key 透传过来。 */
  formatValue?: (v: number, segment: DonutSegment) => string
  /** 顶部总数格式化函数(默认千分位整数) */
  formatTotal?: (v: number) => string
}) {
  const sum = total ?? segments.reduce((s, x) => s + Math.max(0, x.value), 0)
  if (sum <= 0 || segments.length === 0) {
    return <div className="flex h-40 items-center justify-center text-[13px] text-[var(--shell-group-title)]">—</div>
  }
  const r = 50
  const c = 2 * Math.PI * r
  const fmtVal = formatValue ?? ((v: number) => v.toString())
  // 实际调用通过 closure 传 segment:在 map 内部用 (v) => formatValue(v, s) 包装。
  const fmtTot = formatTotal ?? ((v: number) => v.toLocaleString('en-US', { maximumFractionDigits: 0 }))
  let acc = 0
  return (
    <div className="flex items-center gap-4 py-2">
      <svg width="140" height="140" viewBox="0 0 120 120" role="img" aria-label="占比环形">
        <g transform="rotate(-90 60 60)">
          {segments.map((s, i) => {
            const v = Math.max(0, s.value)
            const len = (v / sum) * c
            const dash = `${len} ${c - len}`
            const offset = -(acc / sum) * c
            acc += v
            return (
              <circle key={s.label + i} cx="60" cy="60" r={r} fill="none"
                stroke={PALETTE[i % PALETTE.length]} strokeWidth="18"
                strokeDasharray={dash} strokeDashoffset={offset} />
            )
          })}
        </g>
        <text x="60" y="55" textAnchor="middle" className="fill-[var(--shell-heading)]" fontSize="20" fontWeight="600">{fmtTot(sum)}</text>
        {label && <text x="60" y="72" textAnchor="middle" className="fill-[var(--shell-group-title)]" fontSize="10">{label}</text>}
        {sublabel && <text x="60" y="88" textAnchor="middle" className="fill-[var(--shell-group-title)]" fontSize="9">{sublabel}</text>}
      </svg>
      <ul className="flex flex-col gap-1.5 text-[13px]">
        {segments.map((s, i) => {
          const pct = sum > 0 ? ((s.value / sum) * 100).toFixed(1) : '0.0'
          return (
            <li key={s.label + i} className="flex items-center gap-2">
              <span className="inline-block h-3 w-3 rounded-sm" style={{ background: PALETTE[i % PALETTE.length] }} />
              <span className="text-[var(--shell-content-text)]">{s.label}</span>
              <span className="text-[var(--shell-group-title)]">· {fmtVal(s.value, s)} ({pct}%)</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}