// 产品资费行类型:字段权威 docs/contract/fields.md §2.2(Go customer.ProductOffer)。

export interface ProductRow {
  id: number
  legalEntityId: number
  name: string
  bandwidth: string
  monthlyFee: number
  category: string // broadband/fusion/addon(空回退 broadband)
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

// 产品↔下发模板绑定行(GET/PUT/DELETE /products/{id}/provision-binding)。
export interface OfferBindingRow {
  id: number
  legalEntityId: number
  offerId: number
  templateId: number
  templateCode: string
  templateName: string
  remark: string
}

export const PRODUCT_STATUSES = ['DRAFT', 'PUBLISHED', 'OFFLINE'] as const
export type ProductStatus = (typeof PRODUCT_STATUSES)[number]

export const PRODUCT_CATEGORIES = ['broadband', 'fusion', 'addon'] as const
export type ProductCategory = (typeof PRODUCT_CATEGORIES)[number]
