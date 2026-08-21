// 行政区划面板共享:行类型/空模板/译名标签样式。
export interface SubdivRow {
  code: string
  countryCode: string
  parentCode: string
  level: number
  category: string
  osmAdminLevel: number
  geonameId: number
  isActive: boolean
  displayName: string
}

export interface NameRow { locale: string; name: string; nameType: string }

export const EMPTY: SubdivRow = {
  code: '', countryCode: '', parentCode: '', level: 1, category: 'region',
  osmAdminLevel: 4, geonameId: 0, isActive: true, displayName: '',
}

export const TAG_OFF = 'mb-1.5 inline-flex items-center justify-between rounded-[4px] border border-[color-mix(in_srgb,var(--shell-group-title)_35%,transparent)] bg-[color-mix(in_srgb,var(--shell-group-title)_10%,transparent)] px-2 py-0.5 text-xs leading-[22px] text-[var(--shell-group-title)]'
