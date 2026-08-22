// PGIS 地图容器:OpenLayers + OSM 瓦片 + GeoJSON 点位图层。
// 接收 points 与 onSelect 回调;mount 时初始化,unmount 时销毁。
// jsdom 单测跳过(WebGL 缺失),vitest e2e 用真实浏览器兜底。

import { useEffect, useRef } from 'react'
import 'ol/ol.css'
import Map from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import VectorLayer from 'ol/layer/Vector'
import OSM from 'ol/source/OSM'
import VectorSource from 'ol/source/Vector'
import { fromLonLat } from 'ol/proj'
import { GeoJSON } from 'ol/format'
import { Style, Circle, Fill, Stroke, Text } from 'ol/style'
import { pointRadius, pointColor, pointsToFeatureCollection, type GisPoint } from './point-layer'
import { DEFAULT_VIEW } from './bbox-utils'

export function PgisMap({
  points, onSelect,
}: {
  points: GisPoint[]
  onSelect?: (p: GisPoint) => void
}) {
  const ref = useRef<HTMLDivElement | null>(null)
  const mapRef = useRef<Map | null>(null)

  useEffect(() => {
    if (!ref.current) return
    const map = new Map({
      target: ref.current,
      layers: [
        new TileLayer({ source: new OSM() }),
      ],
      view: new View({
        center: fromLonLat(DEFAULT_VIEW.center),
        zoom: DEFAULT_VIEW.zoom,
      }),
    })
    mapRef.current = map
    return () => { map.setTarget(undefined); mapRef.current = null }
  }, [])

  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    const max = Math.max(1, ...points.map((p) => p.count))
    const source = new VectorSource({
      features: new GeoJSON().readFeatures(pointsToFeatureCollection(points), {
        dataProjection: 'EPSG:4326', featureProjection: 'EPSG:3857',
      }),
    })
    const layer = new VectorLayer({ source, style: (f) => pointStyle(f, max) })
    map.addLayer(layer)
    const handler = map.on('singleclick', (evt) => {
      const feat = map.forEachFeatureAtPixel(evt.pixel, (x) => x)
      if (feat && onSelect) {
        const props = feat.getProperties() as unknown as GisPoint
        onSelect(props)
      }
    })
    return () => { map.removeLayer(layer); map.un('singleclick', handler as never) }
  }, [points, onSelect])

  return <div ref={ref} className="h-full min-h-96 w-full rounded-md border border-[var(--shell-card-border)]" data-testid="pgis-map" />
}

/** 单点 style:圆形 + 品牌色填充 + 白色描边 + 中心数字。 */
function pointStyle(feat: import('ol/Feature').FeatureLike, max: number): Style {
  const p = feat.getProperties() as unknown as GisPoint
  return new Style({
    image: new Circle({
      radius: pointRadius(p.count, max),
      fill: new Fill({ color: pointColor(p.status) }),
      stroke: new Stroke({ color: '#FFFFFF', width: 1.5 }),
    }),
    text: new Text({
      text: p.count > 0 ? String(p.count) : '',
      fill: new Fill({ color: '#FFFFFF' }),
      font: '11px sans-serif',
    }),
  })
}