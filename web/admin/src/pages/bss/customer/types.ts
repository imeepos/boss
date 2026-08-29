// 客户档案行类型:字段权威 docs/contract/fields.md §2.1(Go customer.Customer)。

export interface CustomerRow {
  id: number
  name: string
  phone: string
  idType: string // 身份证/护照/营业执照/无
  idNo: string
  realNameStatus: string // VERIFIED / PENDING
  serviceStatus: string // ACTIVE / ARREARS / SUSPENDED
  addressId: number
  legalEntityId: number
  regionId: number
  regionName: string
  createdAt: string
}

// 实名核验记录行(Go customer.RealNameVerification)。
export interface VerifyLogRow {
  id: number
  customerId: number
  method: string // 人脸/证件OCR/人工/第三方
  verifiedAt: string
  result: string // PASS / FAIL
  operatorAccountId: number
  operatorName: string
}

// 自助注册申请行(fields.md §7.6 customer_registrations,脱敏字段)。
export interface RegistrationRow {
  id: number
  name: string
  phone: string
  idCardNo: string
  legalEntityId: number
  addressId: number
  regionId: number
  source: string
  status: string // PENDING / APPROVED / REJECTED
  reviewNote: string
  reviewerAccountId: number
  customerId: number // 0=未建主档
  submittedAt: string
  reviewedAt: string
}
