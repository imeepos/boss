// 待办主题拆分:把首段业务编号(ALM-xxx/DT-xxx 等)与描述正文分离,
// 长编号不再淹没正文;编号无匹配模式时整体作为正文返回。
const ID_PATTERN = /^[A-Z][A-Z0-9]*(-[0-9]+)+$/

export function splitSubject(subject: string): { id: string; text: string } {
  const trimmed = subject.trim()
  const spaceAt = trimmed.indexOf(' ')
  if (spaceAt <= 0) return { id: '', text: trimmed }
  const id = trimmed.slice(0, spaceAt)
  if (!ID_PATTERN.test(id)) return { id: '', text: trimmed }
  return { id, text: trimmed.slice(spaceAt + 1).trim() }
}