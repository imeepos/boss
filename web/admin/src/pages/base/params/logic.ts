// 业务参数页纯逻辑:筛选(关键字命中 参数/说明/key;状态=已修改/未修改)。

export interface BizParam {
  key: string
  label: string
  value: string
  desc: string
}

export function filterParams(
  all: BizParam[],
  draft: Record<string, string>,
  keyword: string,
  status: string,
): BizParam[] {
  const kw = keyword.trim()
  return all.filter((p) => {
    const hitKw = !kw || [p.label, p.desc, p.key].some((s) => String(s).includes(kw))
    const dirty = (draft[p.key] ?? p.value) !== p.value
    const hitSt = !status || (status === 'changed' ? dirty : !dirty)
    return hitKw && hitSt
  })
}
