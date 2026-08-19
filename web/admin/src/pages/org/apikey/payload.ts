// 创建 API key 的请求体构造:契约(internal/app/http_apikey.go)要求主体三态,
// 本页只签发账号主体。历史 bug:曾发 {accountId,name} 导致线上创建必 42200。
export function buildCreatePayload(accountId: number, name: string): {
  subjectType: 'account'
  subjectRef: number
  name: string
} {
  return { subjectType: 'account', subjectRef: accountId, name: name.trim() }
}
