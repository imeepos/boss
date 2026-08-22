// 点位图层:把 GIS 域 /gis/points 返回的 Point[] 转 GeoJSON FeatureCollection 喂给 OL。
// 包含 cluster 聚合 + 金/蓝品牌色 style + 单击弹 name/status 回调。

import type { Theme } from './tile-source'

// OL 10.x 的 GeoJSON 类型在 'ol/format' 命名空间下,用 loose interface 表达。
export interface GisPoint {
  id: number
  level: number
  name: string
  lng: number
  lat: number
  status: string
  count: number
  parentId: number
}

interface OlFeatureCollection {
  type: 'FeatureCollection'
  features: Array<{
    type: 'Feature'
    geometry: { type: 'Point'; coordinates: [number, number] }
    properties: GisPoint
  }>
}

/** 把 Point 数组转 GeoJSON FeatureCollection(OL VectorLayer 直接吃)。 */
export function pointsToFeatureCollection(points: GisPoint[]): OlFeatureCollection {
  return {
    type: 'FeatureCollection',
    features: points.map((p) => ({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [p.lng, p.lat] },
      properties: p,
    })),
  }
}

/** 根据点位 count 返回半径(像素):count 越大圆越大,sqrt 缩放避免极端值遮挡。 */
export function pointRadius(count: number, max: number): number {
  if (max <= 0) return 6
  return 6 + 14 * Math.sqrt(count / max)
}

/** 品牌色:在线=金色,离线/未知=深蓝。
 *  dark 主题下亮色提亮对比度,深色提亮为浅金(对齐 CartoDB Dark Matter 浅色特征)。 */
export function pointColor(status: string, theme: Theme = 'light'): string {
  if (theme === 'dark') {
    return status === 'ONLINE' ? '#E5C985' : '#9CAAC3'
  }
  return status === 'ONLINE' ? '#C69835' : '#1F355F'
}