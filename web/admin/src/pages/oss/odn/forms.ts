// ODN 页共享类型与样式常量。内联表单已移除(oss-odn v2 spec §0.1):
// 新增/编辑一律经 ResourceDrawer 抽屉,详情经 DetailDrawer。

export const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
export const FIELD = 'flex flex-col gap-1'
export const LABEL = 'text-xs text-[var(--shell-content-text)]'

export type Tab = 'grids' | 'facilities' | 'sites' | 'devices' | 'coverage' | 'constructions' | 'surveys' | 'assets'
export type Region = { prvCode: string; name: string }
export type City = { cityPrefix: string; name: string }
export type Grid = { prvCode: string; cityPrefix: string; gridCode: number; name: string; coverage: string; status: string; facilities: number; warn: boolean }
export type AssetReg = { registrationNo: string; assetId: number; assetCode: string; assetStatus: string }
export type Facility = { code: string; kind: string; prvCode: string; cityPrefix: string; gridCode: number; name: string; lat: number | null; lng: number | null; status: string; lifecycleStatus?: string; assetReg?: AssetReg | null }
export type Site = { prvCode: string; cityPrefix: string; siteNo: number; name: string; lat: number | null; lng: number | null; status: string; lifecycleStatus?: string }
export type Device = { id: number; code: string; kind: string; prvCode: string; cityPrefix: string; siteNo: number; parentId: number; name: string; lat: number | null; lng: number | null; status: string; lifecycleStatus?: string; assetReg?: AssetReg | null }