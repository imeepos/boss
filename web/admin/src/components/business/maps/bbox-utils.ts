// bbox 工具:OL 地图视域与后端接口共享。
// 格式:"minLng,minLat,maxLng,maxLat"(WGS84,逗号分隔,无空格);空=不限。

export interface Bbox { minLng: number; minLat: number; maxLng: number; maxLat: number }

/** 解析 bbox 字符串;空串返回 null。失败抛错(供上层捕获)。 */
export function parseBbox(s: string): Bbox | null {
  if (!s || !s.trim()) return null
  const parts = s.split(',').map((p) => p.trim())
  if (parts.length !== 4) {
    throw new Error(`bbox 需 4 段 (minLng,minLat,maxLng,maxLat),收到 ${parts.length}`)
  }
  const [minLng, minLat, maxLng, maxLat] = parts.map(Number)
  for (const v of [minLng, minLat, maxLng, maxLat]) {
    if (!Number.isFinite(v)) throw new Error('bbox 段含非数值')
  }
  if (minLng > maxLng || minLat > maxLat) {
    throw new Error('bbox 顺序错(minLng<=maxLng,minLat<=maxLat)')
  }
  return { minLng, minLat, maxLng, maxLat }
}

/** 序列化为后端期望的 "minLng,minLat,maxLng,maxLat"。 */
export function formatBbox(b: Bbox): string {
  return `${b.minLng},${b.minLat},${b.maxLng},${b.maxLat}`
}

/** 默认视域:马尼拉大区(13 站 PSGC 起手),zoom 适中。 */
export const DEFAULT_VIEW = { center: [121.0, 14.6] as [number, number], zoom: 10 }