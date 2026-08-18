// 403/404 错误页。
export function ForbiddenPage() {
  return (
    <div style={{ textAlign: 'center', padding: 64 }}>
      <h1>403</h1>
      <p>当前角色无权访问该页面(接口级权限由后端兜底拦截)。</p>
    </div>
  )
}

export function NotFoundPage() {
  return (
    <div style={{ textAlign: 'center', padding: 64 }}>
      <h1>404</h1>
      <p>页面不存在。</p>
    </div>
  )
}
