// API 在线文档页:拉取后端聚合契约(GET /docs/openapi)交 Swagger UI 渲染。
// Try-it-out 经 requestInterceptor 改写到 apiBaseUrl 并注入当前登录态,
// 管理员可直接在页面真实调用四端接口(权限仍由后端逐接口 RBAC 拦截)。
import { useCallback, useEffect, useRef, useState } from 'react'
import SwaggerUI from 'swagger-ui-react'
import 'swagger-ui-react/swagger-ui.css'
import './apidocs.css'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '../../../components/ui/tabs'
import { apiFetch, apiBaseUrl, getAuthToken } from '../../../api/client'

const PORTALS = ['admin', 'user', 'worker', 'open'] as const
type Portal = (typeof PORTALS)[number]

// 请求改写:契约 servers 为相对前缀(如 /api/admin/v1),try-it-out 需指向
// 服务端配置的基址(默认 102 直连);Authorization 注入当前 admin 登录态。
function rewriteRequest(req: Record<string, unknown> & { url: string; headers: Record<string, string> }) {
  const base = apiBaseUrl().replace(/\/$/, '')
  const u = new URL(req.url, window.location.origin)
  req.url = base + u.pathname + u.search
  const token = getAuthToken()
  if (token) req.headers.Authorization = `Bearer ${token}`
  return req
}

export default function ApiDocsPage() {
  const t = useT()
  const [portal, setPortal] = useState<Portal>('admin')
  const [spec, setSpec] = useState<Record<string, unknown> | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const cache = useRef(new Map<Portal, Record<string, unknown>>())

  const load = useCallback(async (p: Portal) => {
    const hit = cache.current.get(p)
    if (hit) {
      setSpec(hit)
      setError('')
      return
    }
    setLoading(true)
    setError('')
    try {
      const data = await apiFetch<Record<string, unknown>>('/docs/openapi', { query: { portal: p } })
      if (!data) throw new Error('empty spec')
      cache.current.set(p, data)
      setSpec(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.apidocs.loadFailed)
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load(portal)
  }, [portal, load])

  const portalLabel = (p: Portal) =>
    p === 'admin' ? t.pages.apidocs.portalAdmin
      : p === 'user' ? t.pages.apidocs.portalUser
        : p === 'worker' ? t.pages.apidocs.portalWorker
          : t.pages.apidocs.portalOpen

  return (
    <div className="apidocs-root">
      <PageHead title={t.pages.apidocs.title} desc={t.pages.apidocs.desc} />
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <Tabs value={portal} onValueChange={(v) => setPortal(v as Portal)}>
          <TabsList>
            {PORTALS.map((p) => (
              <TabsTrigger key={p} value={p} className="px-4">{portalLabel(p)}</TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <span className="text-xs text-[var(--shell-crumb-text)]">{t.pages.apidocs.tryHint}</span>
      </div>
      {error && <div className="mb-3"><ErrorBanner message={error} /></div>}
      <Card className="p-4">
        {loading && <div className="flex min-h-100 items-center justify-center text-sm text-[var(--shell-crumb-text)]">{t.common.loading}</div>}
        {!loading && !error && spec && (
          <SwaggerUI
            spec={spec as never}
            docExpansion="none"
            defaultModelsExpandDepth={0}
            tryItOutEnabled
            persistAuthorization
            displayOperationId={false}
            requestInterceptor={rewriteRequest as never}
          />
        )}
      </Card>
    </div>
  )
}
