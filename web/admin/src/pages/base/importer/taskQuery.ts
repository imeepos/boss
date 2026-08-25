export interface TaskQueryFilters {
  kind: string
  operator: string
  from: string
  to: string
}

export function buildTaskQuery(filters: TaskQueryFilters): string {
  const params = new URLSearchParams()
  if (filters.kind) params.set('kind', filters.kind)
  if (filters.operator) params.set('operator', filters.operator)
  if (filters.from) params.set('from', new Date(`${filters.from}T00:00:00Z`).toISOString())
  if (filters.to) params.set('to', new Date(`${filters.to}T00:00:00Z`).toISOString())
  const query = params.toString()
  return query ? `?${query}` : ''
}
