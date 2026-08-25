// 瓦片源工厂:把"亮/暗主题 + 多种瓦片协议"统一为 OL TileLayer 构造参数。
// 亮色=高德地图(AMap)栅格瓦片;暗色=CartoDB Dark Matter(高德无原生暗色瓦片)。
// AMAP_KEY 通过 VITE_AMAP_KEY 环境变量注入,见 .env.example。

import TileLayer from 'ol/layer/Tile'
import OSM from 'ol/source/OSM'
import XYZ from 'ol/source/XYZ'

export type Theme = 'light' | 'dark'

/** 高德地图栅格瓦片(普通地图,中文标注,4 子域取 01)。 */
export const LIGHT_TILE_URL = 'https://webrd01.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}'
/** 暗色主题用 CartoDB Dark Matter(高德无原生暗色瓦片;自建 tileserver 后可切 PMTiles)。 */
export const DARK_TILE_URL = 'https://{a-c}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png'
/** 自建 PMTiles 单文件切片(tileserver-gl 反代;后续切换)。 */
export const PMTILES_TILE_URL = '/tiles/{z}/{x}/{y}.pbf'

/** 读取高德地图 key(从 VITE_AMAP_KEY 环境变量),未配置则返回空串。 */
function amapKey(): string {
  return (typeof import.meta !== 'undefined' && (import.meta.env as Record<string, string>).VITE_AMAP_KEY) || ''
}

/** 在 URL 尾部追加 key 参数(仅 AMap 瓦片需要)。 */
function appendKey(url: string, key: string): string {
  if (!key || !url.includes('is.autonavi.com')) return url
  const sep = url.includes('?') ? '&' : '?'
  return url + sep + 'key=' + key
}

/** 解析最终瓦片 URL:主题默认值 → AMap key 注入(纯函数,供单测传 key 断言)。 */
export function resolveTileUrl(theme: Theme = 'light', url?: string, key?: string): string {
  return appendKey(url ?? (theme === 'dark' ? DARK_TILE_URL : LIGHT_TILE_URL), key ?? amapKey())
}

/** 构造瓦片 TileLayer(工厂封装,避免业务页重复写 OL 配置)。 */
export function makeTileLayer(theme: Theme = 'light', url?: string): TileLayer<OSM | XYZ> {
  const finalUrl = resolveTileUrl(theme, url)
  if (finalUrl.includes('{a-c}') || finalUrl.includes('{a-b}') || finalUrl.includes('{s}')) {
    return new TileLayer({ source: new XYZ({ url: finalUrl, crossOrigin: 'anonymous' }) })
  }
  if (finalUrl.includes('is.autonavi.com')) {
    return new TileLayer({ source: new XYZ({ url: finalUrl, crossOrigin: 'anonymous' }) })
  }
  return new TileLayer({ source: new OSM({ url: finalUrl, crossOrigin: 'anonymous' }) })
}

/** 从当前 document.theme 属性读主题(兼容项目 next-themes / 自定义 data-theme)。 */
export function readDocumentTheme(): Theme {
  if (typeof document === 'undefined') return 'light'
  const t = document.documentElement.getAttribute('data-theme')
  return t === 'dark' ? 'dark' : 'light'
}