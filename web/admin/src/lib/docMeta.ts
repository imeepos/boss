// SPA 文档元数据直操(无 helmet 依赖):setMeta 按 name/property 定位或创建
// <meta>,卸载还原;title 置空触发页面级重设。SEO 够用,不引库(adopted 取舍)。

/** 设置 <title> 与多个 meta(name 或 property 键)。返回还原函数。 */
export function setDocMeta(title: string, metas: Record<string, string>): () => void {
  const prevTitle = document.title
  document.title = title
  const touched: Array<{ el: HTMLMetaElement; had: boolean; prev?: string }> = []
  for (const [key, content] of Object.entries(metas)) {
    let el = document.head.querySelector<HTMLMetaElement>(`meta[name="${key}"], meta[property="${key}"]`)
    if (!el) {
      el = document.createElement('meta')
      el.setAttribute(key.startsWith('og:') ? 'property' : 'name', key)
      document.head.appendChild(el)
      touched.push({ el, had: false })
    } else {
      touched.push({ el, had: true, prev: el.content })
    }
    el.content = content
  }
  return () => {
    document.title = prevTitle
    for (const t of touched) {
      if (t.had && t.prev !== undefined) t.el.content = t.prev
      else t.el.remove()
    }
  }
}
