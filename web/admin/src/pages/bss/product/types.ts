// 产品资费行类型:字段权威 docs/contract/fields.md §2.2(Go customer.ProductOffer)。

export interface ProductRow {
  id: number
  legalEntityId: number
  name: string
  bandwidth: string
  monthlyFee: number
  effectiveAt: string
  status: string // DRAFT/PUBLISHED/OFFLINE
}

// 产品调价台账行(Go customer.ProductPriceHistory)。
export interface PriceHistoryRow {
  id: number
  offerId: number
  oldMonthlyFee: number
  newMonthlyFee: number
  effectiveAt: string
  reason: string
  operatorAccountId: number
}

export const PRODUCT_STATUSES = ['DRAFT', 'PUBLISHED', 'OFFLINE'] as const
export type ProductStatus = (typeof PRODUCT_STATUSES)[number]
