// PGIS 地图容器:OpenLayers + 可切换瓦片源(light/dark/PMTiles)+ GeoJSON 点位图层。
// 接收 points/theme/tileUrl + onSelect + onViewportChange;mount 一次,
// points/theme 变化复用 map 实例。viewport 回调在 moveend 时触发,
// 业务页用于"视域内点位"实时统计(B3 消费)。
// jsdom 单测跳过(WebGL 缺失),vitest e2e 用真实浏览器兜底。

import { useEffect, useRef } from 'react'
import 'ol/ol.css'
import Map from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import VectorLayer from 'ol/layer/Vector'
import VectorSource from 'ol/source/Vector'
import { fromLonLat, transformExtent } from 'ol/proj'
import { unByKey } from 'ol/Observable'
import { GeoJSON } from 'ol/format'
import { Style, Circle, Fill, Stroke, Text } from 'ol/style'
import { pointRadius, pointColor, pointsToFeatureCollection, type GisPoint } from './point-layer'
import { DEFAULT_VIEW } from './bbox-utils'
import { makeTileLayer, readDocumentTheme, type Theme } from './tile-source'

/** 视域范围(WGS84 经纬度),格式与 /gis/points 的 bbox 入参兼容。 */
export interface ViewportBbox { minLng: number; minLat: number; maxLng: number; maxLat: number }

export function PgisMap({
  points, onSelect, onViewportChange, theme, tileUrl,
}: {
  points: GisPoint[]
  onSelect?: (p: GisPoint) => void
  /** 视域范围回调(moveend/zoomend 时调用);不传则不订阅。 */
  onViewportChange?: (b: ViewportBbox) => void
  /** 不传时按 [data-theme] 自动切换;传了以传入为准。 */
  theme?: Theme
  /** 不传时按 theme 选默认 URL(light→高德,dark→CartoDB)。 */
  tileUrl?: string
}) {
  const ref = useRef<HTMLDivElement | null>(null)
  const mapRef = useRef<Map | null>(null)
  const tileLayerRef = useRef<TileLayer | null>(null)
  const vectorLayerRef = useRef<VectorLayer | null>(null)

  // 初始化地图(只一次);主题变化只换 tileLayer。
  useEffect(() => {
    if (!ref.current) return
    const map = new Map({
      target: ref.current,
      layers: [],
      view: new View({
        center: fromLonLat(DEFAULT_VIEW.center),
        zoom: DEFAULT_VIEW.zoom,
      }),
    })
    mapRef.current = map
    const initial = makeTileLayer(theme ?? readDocumentTheme(), tileUrl)
    map.addLayer(initial)
    tileLayerRef.current = initial
    return () => {
      map.setTarget(undefined)
      mapRef.current = null
      tileLayerRef.current = null
      vectorLayerRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // 主题/tileUrl 变化 → 替换 tileLayer。
  useEffect(() => {
    const map = mapRef.current
    if (!map || !tileLayerRef.current) return
    const newLayer = makeTileLayer(theme ?? readDocumentTheme(), tileUrl)
    map.addLayer(newLayer)
    const old = tileLayerRef.current
    map.removeLayer(old)
    tileLayerRef.current = newLayer
  }, [theme, tileUrl])

  // points 或 theme 变化 → 重贴 vectorLayer(主题影响点位颜色)。
  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    const effTheme = theme ?? readDocumentTheme()
    const max = Math.max(1, ...points.map((p) => p.count))
    const source = new VectorSource({
      features: new GeoJSON().readFeatures(pointsToFeatureCollection(points), {
        dataProjection: 'EPSG:4326', featureProjection: 'EPSG:3857',
      }),
    })
    const layer = new VectorLayer({ source, style: (f) => pointStyle(f, max, effTheme) })
    map.addLayer(layer)
    const handler = map.on('singleclick', (evt) => {
      const f = map.forEachFeatureAtPixel(evt.pixel, (x) => x)
      if (f && onSelect) {
        const props = f.getProperties() as unknown as GisPoint
        onSelect(props)
      }
    })
    const old = vectorLayerRef.current
    vectorLayerRef.current = layer
    return () => {
      map.removeLayer(layer)
      unByKey(handler)
      if (vectorLayerRef.current === layer) vectorLayerRef.current = old
    }
  }, [points, theme, onSelect])

  // 视域变化(moveend/zoomend)→ 计算 bbox 转 WGS84 → 回调。
  useEffect(() => {
    const map = mapRef.current
    if (!map || !onViewportChange) return
    const handler = () => {
      const ext = map.getView().calculateExtent(map.getSize() ?? [0, 0])
      const [minX, minY, maxX, maxY] = transformExtent(ext, 'EPSG:3857', 'EPSG:4326')
      onViewportChange({ minLng: minX, minLat: minY, maxLng: maxX, maxLat: maxY })
    }
    const listener = map.on('moveend', handler)
    // mount 后立即触发一次(初始视域)
    handler()
    return () => unByKey(listener)
  }, [onViewportChange])

  return <div ref={ref} className="h-full min-h-96 w-full rounded-md border border-[var(--shell-card-border)]" data-testid="pgis-map" />
}

/** 单点 style:圆形 + 主题感知品牌色填充 + 白色描边 + 中心数字。 */
function pointStyle(feat: import('ol/Feature').FeatureLike, max: number, theme: Theme): Style {
  const p = feat.getProperties() as unknown as GisPoint
  return new Style({
    image: new Circle({
      radius: pointRadius(p.count, max),
      fill: new Fill({ color: pointColor(p.status, theme) }),
      stroke: new Stroke({ color: '#FFFFFF', width: 1.5 }),
    }),
    text: new Text({
      text: p.count > 0 ? String(p.count) : '',
      fill: new Fill({ color: '#FFFFFF' }),
      font: '11px sans-serif',
    }),
  })
}