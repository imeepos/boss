// 瓦片源工厂:把"亮/暗主题 + 多种瓦片协议"统一为 OL TileLayer 构造参数。
// 当前对接高德地图(AMap)栅格瓦片,亮/暗主题共用同一套高德瓦片;
// 高德无原生暗色瓦片,后续可自建 tileserver 时切换 PMTiles。

import TileLayer from 'ol/layer/Tile'
import OSM from 'ol/source/OSM'
import XYZ from 'ol/source/XYZ'

export type Theme = 'light' | 'dark'

/** 高德地图栅格瓦片(普通地图,中文标注,4 子域取 01)。 */
export const LIGHT_TILE_URL = 'https://webrd01.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}'
/** 暗色主题暂用同一套高德瓦片(高德无原生暗色瓦片)。 */
export const DARK_TILE_URL = 'https://webrd01.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}'
/** 自建 PMTiles 单文件切片(tileserver-gl 反代;后续切换)。 */
export const PMTILES_TILE_URL = '/tiles/{z}/{x}/{y}.pbf'

/** 构造瓦片 TileLayer(工厂封装,避免业务页重复写 OL 配置)。 */
export function makeTileLayer(theme: Theme = 'light', url?: string): TileLayer<OSM | XYZ> {
  const finalUrl = url ?? (theme === 'dark' ? DARK_TILE_URL : LIGHT_TILE_URL)
  if (finalUrl.includes('{a-c}') || finalUrl.includes('{a-b}') || finalUrl.includes('{s}')) {
    return new TileLayer({ source: new XYZ({ url: finalUrl, crossOrigin: 'anonymous' }) })
  }
  // 高德瓦片含子域 01-04,但以固定 01 子域 URL 传入,仍用 XYZ 源。
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