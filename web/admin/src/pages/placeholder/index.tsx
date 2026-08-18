// A1 前占位页:菜单全集路由可直达,A1 批次逐页替换为正式实现。
export function PlaceholderPage({ title }: { title: string }) {
  return (
    <div>
      <h2>{title}</h2>
      <p style={{ color: '#888' }}>建设中(A1 批次接入,列名对照 docs/contract/fields.md)。</p>
    </div>
  )
}
