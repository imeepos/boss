import {
  Area,
  CartesianGrid,
  Line,
  LabelList,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

interface OrderTrendPoint {
  label: string
  value: number
}

interface OrderTrendProps {
  labels: string[]
  values: number[]
  valueUnit: string
  tooltipLabel: string
  emptyText: string
}

function points(labels: string[], values: number[]): OrderTrendPoint[] {
  return labels.map((label, index) => ({ label, value: values[index] ?? 0 }))
}

function formatValue(value: number, unit: string): string {
  return `${value}${unit}`
}

export function OrderTrend({ labels, values, valueUnit, tooltipLabel, emptyText }: OrderTrendProps) {
  const data = points(labels, values)
  if (data.length === 0) {
    return <div className="flex h-64 items-center justify-center text-[13px] text-[var(--shell-group-title)]">{emptyText}</div>
  }

  return (
    <div className="h-80 w-full" role="img" aria-label={tooltipLabel}>
      <ResponsiveContainer width="100%" height="100%" minWidth={280}>
        <LineChart data={data} margin={{ top: 16, right: 12, left: -12, bottom: 4 }}>
          <defs>
            <linearGradient id="order-trend-fill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--color-brand-gold-500)" stopOpacity={0.28} />
              <stop offset="100%" stopColor="var(--color-brand-gold-500)" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid vertical={false} stroke="var(--shell-side-border)" strokeDasharray="3 5" />
          <XAxis
            dataKey="label"
            interval={data.length > 14 ? Math.ceil(data.length / 12) - 1 : 0}
            minTickGap={16}
            axisLine={false}
            tickLine={false}
            tick={{ fill: 'var(--shell-group-title)', fontSize: 11 }}
            dy={8}
          />
          <YAxis
            allowDecimals={false}
            axisLine={false}
            tickLine={false}
            width={30}
            tick={{ fill: 'var(--shell-group-title)', fontSize: 11 }}
          />
          <Tooltip
            cursor={{ stroke: 'var(--color-brand-gold-500)', strokeDasharray: '4 4', strokeOpacity: 0.65 }}
            contentStyle={{
              background: 'var(--shell-card-bg)',
              border: '1px solid var(--shell-card-border)',
              borderRadius: '8px',
              boxShadow: 'var(--shell-card-shadow)',
              color: 'var(--shell-heading)',
              fontSize: '12px',
            }}
            labelStyle={{ color: 'var(--shell-heading)', fontWeight: 600, marginBottom: 4 }}
            formatter={(value) => [formatValue(Number(value), valueUnit), tooltipLabel]}
          />
          <Area type="monotone" dataKey="value" stroke="none" fill="url(#order-trend-fill)" />
          <Line
            type="monotone"
            dataKey="value"
            stroke="var(--color-brand-gold-500)"
            strokeWidth={3}
            dot={{ r: 5, fill: 'var(--shell-card-bg)', stroke: 'var(--color-brand-gold-500)', strokeWidth: 2 }}
            activeDot={{ r: 7, fill: 'var(--color-brand-gold-500)', stroke: 'var(--shell-card-bg)', strokeWidth: 2 }}
            connectNulls
          >
            <LabelList
              dataKey="value"
              position="top"
              offset={10}
              fill="var(--shell-heading)"
              fontSize={12}
              fontWeight={600}
              formatter={(value: number) => formatValue(value, valueUnit)}
            />
          </Line>
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
