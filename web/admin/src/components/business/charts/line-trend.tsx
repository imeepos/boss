// 趋势折线:N 个时间点(横轴)× M 条指标线(纵轴);SVG <polyline> + 圆点节点。
// 每条折线 max 独立归一(避免跨指标量纲差异);legend 在下方。
// 用于"四周期/多快照"趋势对比(报告中心 commit B6 消费)。

const PALETTE = ['#C69835', '#1F355F', '#BE8D25', '#273F70', '#9CAAC3']

export interface LineTrendSeries {
  /** 指标名(图例);与 dots 等长 */
  name: string
  /** N 个时间点的指标值(顺序=labels 顺序) */
  values: number[]
}

export function LineTrend({
  labels, series, height = 200,
}: {
  labels: string[]
  series: LineTrendSeries[]
  /** 画布高(像素);宽自适应容器宽度。 */
  height?: number
}) {
  if (series.length === 0 || labels.length === 0) return null
  const W = 720
  const H = height
  const padX = 32
  const padY = 24
  const innerW = W - padX * 2
  const innerH = H - padY * 2
  const stepX = innerW / Math.max(1, labels.length - 1)
  const xAt = (i: number) => padX + i * stepX
  // 每条线各自 max 归一(避免跨指标量纲)
  const maxes = series.map((s) => Math.max(1, ...s.values))
  return (
    <div className="flex flex-col gap-3">
      <svg viewBox={`0 0 ${W} ${H}`} className="h-48 w-full" role="img" aria-label="趋势折线">
        {/* y 轴基线 */}
        {[0, 0.5, 1].map((p) => (
          <line key={p}
            x1={padX} y1={padY + innerH * (1 - p)}
            x2={W - padX} y2={padY + innerH * (1 - p)}
            stroke="var(--shell-side-border)" strokeDasharray="2 4" strokeWidth="1"
          />
        ))}
        {/* x 轴标签 */}
        {labels.map((lab, i) => (
          <text key={lab + i}
            x={xAt(i)} y={H - 6}
            textAnchor="middle" fontSize="10"
            className="fill-[var(--shell-group-title)]">{lab}</text>
        ))}
        {/* 每条指标线 */}
        {series.map((s, k) => {
          const max = maxes[k] ?? 1
          const pts = s.values.map((v, i) => {
            const y = padY + innerH * (1 - Math.max(0, v) / max)
            return `${xAt(i)},${y}`
          }).join(' ')
          return (
            <g key={s.name}>
              <polyline points={pts} fill="none"
                stroke={PALETTE[k % PALETTE.length]} strokeWidth="2" strokeLinejoin="round" />
              {s.values.map((v, i) => {
                const y = padY + innerH * (1 - Math.max(0, v) / max)
                return <circle key={i} cx={xAt(i)} cy={y} r="3"
                  fill={PALETTE[k % PALETTE.length]} stroke="#FFFFFF" strokeWidth="1" />
              })}
            </g>
          )
        })}
      </svg>
      <ul className="flex flex-wrap items-center gap-3 text-[12px]">
        {series.map((s, k) => (
          <li key={s.name} className="flex items-center gap-1.5">
            <span className="inline-block h-3 w-3 rounded-sm" style={{ background: PALETTE[k % PALETTE.length] }} />
            <span className="text-[var(--shell-content-text)]">{s.name}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}