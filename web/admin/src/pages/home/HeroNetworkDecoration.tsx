// Hero 右侧抽象网络装饰图:渐变光晕 + 连线 + 节点,SVG 内联零资源请求。
export function HeroNetworkDecoration() {
  return (
    <svg
      viewBox="0 0 480 360"
      className="h-full w-full"
      aria-hidden
      preserveAspectRatio="xMidYMid meet"
    >
      <defs>
        <radialGradient id="heroGlow" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#D5A63A" stopOpacity="0.18" />
          <stop offset="100%" stopColor="#D5A63A" stopOpacity="0" />
        </radialGradient>
        <linearGradient id="heroLine" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#31568F" stopOpacity="0.5" />
          <stop offset="100%" stopColor="#D5A63A" stopOpacity="0.6" />
        </linearGradient>
      </defs>
      <circle cx="320" cy="160" r="160" fill="url(#heroGlow)" />
      <g stroke="url(#heroLine)" strokeWidth="1" fill="none" opacity="0.55">
        {CONNECTIONS.map(([x1, y1, x2, y2], i) => (
          <line key={i} x1={x1} y1={y1} x2={x2} y2={y2} />
        ))}
      </g>
      <g fill="#31568F" opacity="0.7">
        {NODES.map(([x, y], i) => (
          <circle key={i} cx={x} cy={y} r="3" />
        ))}
      </g>
      <g fill="#D5A63A">
        {HIGHLIGHTS.map(([x, y], i) => (
          <circle key={i} cx={x} cy={y} r="4" />
        ))}
      </g>
    </svg>
  )
}

const CONNECTIONS: Array<[number, number, number, number]> = [
  [80, 90, 200, 50], [120, 220, 240, 130], [60, 280, 180, 200],
  [280, 60, 400, 130], [220, 300, 360, 250], [340, 200, 420, 290],
]

const NODES: Array<[number, number]> = [
  [80, 90], [200, 50], [120, 220], [240, 130], [60, 280], [180, 200],
  [280, 60], [400, 130], [220, 300], [360, 250], [340, 200], [420, 290],
]

const HIGHLIGHTS: Array<[number, number]> = [[200, 50], [400, 130], [240, 130], [420, 290]]