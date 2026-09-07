// 设备编码域规则镜像(internal/domain/odn device.go / nextcode.go):
// 表单预览用纯前端推导,入库权威仍由后端唯一索引与容量校验兜底。

// legalParentKind 设备归属链要求(规范 2.1+2.4);''=顶层(SNW/OLT/ODF/OCC)。
export function legalParentKind(kind: string): string {
  const map: Record<string, string> = { ODB: 'OCC', OBD: 'ODB', SDB: 'ODB', SBD: 'SDB', PRT: 'SDB', TBP: 'PRT' }
  return map[kind] ?? ''
}

// nextDeviceCode 下一可用设备编码建议:同 kind 现存最大 3 位序号+1,
// 与服务端 MAX+1 同规则;扩容后缀行(-N)不参与计数;满 999 返回 null(禁用自动建议)。
export function nextDeviceCode(kind: string, codes: string[]): string | null {
  const re = new RegExp('^' + kind + '([0-9]{3})(?:-[0-9]+)?$')
  let max = 0
  for (const c of codes) {
    const m = re.exec(c)
    if (m) {
      const seq = parseInt(m[1], 10)
      if (seq > max) max = seq
    }
  }
  if (max >= 999) return null
  return kind + String(max + 1).padStart(3, '0')
}
