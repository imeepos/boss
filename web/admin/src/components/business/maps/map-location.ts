// 选点纯逻辑:坐标校验/换算/回显文案/受控状态机。
// node(jsdom) 环境无 WebGL,组件本体不可测;交互规则全部收敛在此文件,vitest 直接覆盖。

import { toLonLat } from 'ol/proj'
import type { Theme } from './tile-source'

/** WGS84 十进制经纬度。 */
export interface LatLng { lat: number; lng: number }

/** 选点后的定位缩放级别(楼栋级);无值回退 bbox-utils DEFAULT_VIEW。 */
export const PICK_ZOOM = 16

/** 坐标小数位:6 位约 0.1m,与落库 float8 语义匹配。 */
export const COORD_DECIMALS = 6

/** 合法选点:范围同后端 PUT /addresses/:id/geom 的 422 前置校验;0,0(几内亚湾)同 binding:required 语义拒绝。 */
export function isValidLatLng(lat: number, lng: number): boolean {
  if (!Number.isFinite(lat) || !Number.isFinite(lng)) return false
  if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return false
  return lat !== 0 || lng !== 0
}

/** 四舍五入到 COORD_DECIMALS 位:拖拽微调时降噪,受控值稳定不抖。 */
export function roundCoord(v: number): number {
  const f = 10 ** COORD_DECIMALS
  return Math.round(v * f) / f
}

/** 受控值规范化:非法/空归一为 null,合法取整;状态机与回显共用同一入口,杜绝双轨。 */
export function normalizeLatLng(v: LatLng | null | undefined): LatLng | null {
  if (!v || !isValidLatLng(v.lat, v.lng)) return null
  return { lat: roundCoord(v.lat), lng: roundCoord(v.lng) }
}

/** Web 墨卡托(EPSG:3857) → WGS84;OL 点击/拖拽坐标出口统一走此换算。 */
export function mercatorToLatLng(xy: [number, number]): LatLng {
  const [lng, lat] = toLonLat(xy)
  return { lat, lng }
}

/** 回显文案 "14.599000, 120.984000";无值返回空串(占位文案由消费方传)。 */
export function formatLatLng(v: LatLng | null): string {
  const n = normalizeLatLng(v)
  if (!n) return ''
  return n.lat.toFixed(COORD_DECIMALS) + ', ' + n.lng.toFixed(COORD_DECIMALS)
}

/** 视域决策:有值定位到该点(PICK_ZOOM),无值回退默认视野(既有 DEFAULT_VIEW)。 */
export function viewForLatLng(
  v: LatLng | null,
  fallback: { center: [number, number]; zoom: number },
): { center: [number, number]; zoom: number } {
  const n = normalizeLatLng(v)
  if (!n) return fallback
  return { center: [n.lng, n.lat], zoom: PICK_ZOOM }
}

/** 受控状态机动作:pick=点击落点,drag=拖拽微调,clear=清空按钮。 */
export type PickerAction =
  | { type: 'pick'; value: LatLng }
  | { type: 'drag'; value: LatLng }
  | { type: 'clear' }

/** 状态机:clear 恒清空;非法落点保持原值(宁可不改也不产非法坐标)。 */
export function nextPickerValue(current: LatLng | null, action: PickerAction): LatLng | null {
  if (action.type === 'clear') return null
  const next = normalizeLatLng(action.value)
  if (next) return next
  return normalizeLatLng(current)
}

/** 标记主题色:与 point-layer 品牌金一致,dark 下提亮(对齐 CartoDB 暗色瓦片)。 */
export function markerColor(theme: Theme): string {
  return theme === 'dark' ? '#E5C985' : '#C69835'
}
