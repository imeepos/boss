// DataTable: generic list page pattern (table + pagination + empty state)
// Consolidates the pattern repeated across ~30 pages.
import type { ReactNode } from 'react'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../ui/table'
import { Card, CardFooter } from '../ui/card'
import { Pagination } from '../Pagination'
import { EmptyState } from './page-head'

export interface ColumnDef {
  key: string
  label: string
  /** Optional render function; defaults to rendering row[key] */
  render?: (row: Record<string, unknown>) => ReactNode
}

export function DataTable({
  columns, rows, emptyText, onRowHover,
}: {
  columns: ColumnDef[]
  rows: Record<string, unknown>[]
  emptyText: string
  onRowHover?: (row: Record<string, unknown>) => void
}) {
  if (!rows.length) {
    return (
      <Card>
        <div className="px-4 py-8">
          <EmptyState text={emptyText} />
        </div>
      </Card>
    )
  }
  return (
    <div className="w-full overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((c) => (
              <TableHead key={c.key}>{c.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, i) => (
            <TableRow
              key={(row.id as number) ?? i}
              onMouseEnter={() => onRowHover?.(row)}
            >
              {columns.map((c) => (
                <TableCell key={c.key}>
                  {c.render ? c.render(row) : (row[c.key] as ReactNode) ?? '—'}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

export function TableFooter({
  total, page, pageSize, onPage, onSize, pagerProps,
}: {
  total: number
  page: number
  pageSize: number
  onPage: (p: number) => void
  onSize: (s: number) => void
  pagerProps: ReturnType<typeof import('./page-head').pagerTexts>
}) {
  return (
    <CardFooter>
      <Pagination
        total={total}
        page={page}
        pageSize={pageSize}
        onPage={onPage}
        onSize={onSize}
        {...pagerProps}
      />
    </CardFooter>
  )
}