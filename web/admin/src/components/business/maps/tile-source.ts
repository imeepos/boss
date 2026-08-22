// 瓦片源工厂:把"亮/暗主题 + 多种瓦片协议"统一为 OL TileLayer 构造参数。
// 当前实现用 OSM/CartoDB 公开瓦片 + pmtiles 协议占位;
// 下期 tileserver-gl 上线后改指向 https://boss.ymm.cn/tiles/{z}/{x}/{y}.pbf。

import TileLayer from 'ol/layer/Tile'
import OSM from 'ol/source/OSM'
import XYZ from 'ol/source/XYZ'

export type Theme = 'light' | 'dark'

/** 亮色瓦片源(默认 OSM,生产建议自建 PMTiles)。 */
export const LIGHT_TILE_URL = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
/** 暗色瓦片源(CartoDB Dark Matter 免费层)。 */
export const DARK_TILE_URL = 'https://{a-c}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png'
/** 自建 PMTiles 单文件切片(tileserver-gl 反代;下期切)。 */
export const PMTILES_TILE_URL = '/tiles/{z}/{x}/{y}.pbf'

/** 构造瓦片 TileLayer(工厂封装,避免业务页重复写 OL 配置)。 */
export function makeTileLayer(theme: Theme = 'light', url?: string): TileLayer<OSM | XYZ> {
  const finalUrl = url ?? (theme === 'dark' ? DARK_TILE_URL : LIGHT_TILE_URL)
  if (finalUrl.includes('{a-c}') || finalUrl.includes('{a-b}') || finalUrl.includes('{s}')) {
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