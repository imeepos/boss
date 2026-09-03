// 地图选点选择器:点击落点、标记拖拽微调;受控 value(WGS84 十进制经纬度)+ onChange。
// 瓦片源与双主题复用 tile-source 工厂,视域决策复用 bbox-utils/map-location 纯函数;
// 文案一律由消费方注入(clearLabel/hint/placeholder),组件本体不做 i18n 耦合。
// node 环境无 WebGL,交互规则在 map-location.test.ts 覆盖,组件本体不写单测。

import { useEffect, useRef } from 'react'
import 'ol/ol.css'
import Map from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import VectorLayer from 'ol/layer/Vector'
import VectorSource from 'ol/source/Vector'
import OSM from 'ol/source/OSM'
import XYZ from 'ol/source/XYZ'
import Collection from 'ol/Collection'
import Feature from 'ol/Feature'
import Point from 'ol/geom/Point'
import Translate from 'ol/interaction/Translate'
import { fromLonLat } from 'ol/proj'
import { unByKey } from 'ol/Observable'
import { Circle, Fill, Stroke, Style } from 'ol/style'
import { DEFAULT_VIEW } from './bbox-utils'
import { makeTileLayer, readDocumentTheme, type Theme } from './tile-source'
import {
  markerColor, mercatorToLatLng, nextPickerValue, normalizeLatLng,
  viewForLatLng, formatLatLng, type LatLng,
} from './map-location'
import { ToolbarButton } from '../page-head'

export type { LatLng }

export function MapLocationPicker({
  value, onChange, theme, height = 320, clearLabel, hint, placeholder,
}: {
  value: LatLng | null
  onChange: (v: LatLng | null) => void
  /** 不传时挂载时按 [data-theme] 读一次;传入则随 prop 切换瓦片与标记色。 */
  theme?: Theme
  /** 地图高度 px;抽屉内默认 320。 */
  height?: number
  /** 清空按钮文案(消费方 i18n)。 */
  clearLabel?: string
  /** 操作提示文案(消费方 i18n)。 */
  hint?: string
  /** 无值时的回显占位文案(消费方 i18n)。 */
  placeholder?: string
}) {
  const ref = useRef<HTMLDivElement | null>(null)
  const mapRef = useRef<Map | null>(null)
  const tileRef = useRef<TileLayer<OSM | XYZ> | null>(null)
  const vecLayerRef = useRef<VectorLayer | null>(null)
  const markerRef = useRef<Feature<Point> | null>(null)
  const draggingRef = useRef(false)
  // 内部交互回声:onChange 后父组件回流的同值不重复定位视域。
  const lastEmittedRef = useRef<LatLng | null>(normalizeLatLng(value))
  const valueRef = useRef(value)
  const onChangeRef = useRef(onChange)
  valueRef.current = value
  onChangeRef.current = onChange

  // 挂载一次:地图 + 瓦片 + 标记层 + 点击/拖拽交互;theme 变化走下方独立 effect。
  useEffect(() => {
    if (!ref.current) return
    const start = viewForLatLng(valueRef.current, { center: DEFAULT_VIEW.center, zoom: DEFAULT_VIEW.zoom })
    const map = new Map({
      target: ref.current,
      view: new View({ center: fromLonLat(start.center), zoom: start.zoom }),
    })
    mapRef.current = map
    const tiles = makeTileLayer(theme ?? readDocumentTheme())
    map.addLayer(tiles)
    tileRef.current = tiles

    const effTheme = theme ?? readDocumentTheme()
    const marker = new Feature<Point>(new Point(fromLonLat(start.center)))
    marker.setStyle(markerStyle(effTheme))
    const vecLayer = new VectorLayer({ source: new VectorSource({ features: [marker] }) })
    map.addLayer(vecLayer)
    markerRef.current = marker
    vecLayerRef.current = vecLayer

    const translate = new Translate({ features: new Collection([marker]) })
    translate.on('translatestart', () => { draggingRef.current = true })
    translate.on('translateend', (evt) => {
      draggingRef.current = false
      const geom = (evt.features.getArray()[0] as Feature<Point> | undefined)?.getGeometry()
      if (!geom) return
      emit(nextPickerValue(valueRef.current, {
        type: 'drag', value: mercatorToLatLng(geom.getCoordinates() as [number, number]),
      }))
    })
    map.addInteraction(translate)

    const click = map.on('singleclick', (evt) => {
      if (draggingRef.current) return
      // 命中标记本体不重复落点(微调走拖拽),防误触把点重置。
      if (map.hasFeatureAtPixel(evt.pixel, { layerFilter: (l) => l === vecLayer })) return
      emit(nextPickerValue(valueRef.current, {
        type: 'pick', value: mercatorToLatLng(evt.coordinate as [number, number]),
      }))
    })
    const move = map.on('pointermove', (evt) => {
      setCursor(map, vecLayer, evt.pixel as [number, number], draggingRef)
    })

    return () => {
      unByKey(click)
      unByKey(move)
      map.removeInteraction(translate)
      map.setTarget(undefined)
      mapRef.current = null
      tileRef.current = null
      vecLayerRef.current = null
      markerRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // 受控 value → 标记几何 + 视域;内部交互回声(lastEmitted 同值)不重复定位。
  useEffect(() => {
    const map = mapRef.current
    const marker = markerRef.current
    if (!map || !marker) return
    const n = normalizeLatLng(value)
    marker.setGeometry(n ? new Point(fromLonLat([n.lng, n.lat])) : undefined)
    const e = lastEmittedRef.current
    const echo = n === null ? e === null : e !== null && n.lat === e.lat && n.lng === e.lng
    if (echo) return
    const v = viewForLatLng(n, { center: DEFAULT_VIEW.center, zoom: DEFAULT_VIEW.zoom })
    map.getView().setCenter(fromLonLat(v.center))
    map.getView().setZoom(v.zoom)
  }, [value])

  // theme prop 变化 → 换瓦片层 + 标记色(与 PgisMap 同节奏)。
  useEffect(() => {
    const map = mapRef.current
    const marker = markerRef.current
    if (!map || !marker) return
    const tiles = makeTileLayer(theme ?? readDocumentTheme())
    map.addLayer(tiles)
    if (tileRef.current) map.removeLayer(tileRef.current)
    tileRef.current = tiles
    marker.setStyle(markerStyle(theme ?? readDocumentTheme()))
  }, [theme])

  const emit = (next: LatLng | null) => {
    if (!next) return
    lastEmittedRef.current = next
    onChangeRef.current(next)
  }

  const clear = () => {
    lastEmittedRef.current = null
    onChangeRef.current(null)
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div ref={ref} style={{ height }} data-testid="map-location-picker"
        className="w-full rounded-md border border-[var(--shell-card-border)]" />
      <div className="flex flex-wrap items-center gap-2 text-xs text-[var(--shell-content-text)]">
        <span className="text-[var(--shell-input-placeholder)]">WGS84</span>
        <span data-testid="location-coords" className="font-mono">
          {formatLatLng(value) || placeholder}
        </span>
        {hint && <span className="text-[var(--shell-input-placeholder)]">{hint}</span>}
        <span className="flex-1" />
        <ToolbarButton disabled={!value} onClick={clear}>{clearLabel}</ToolbarButton>
      </div>
    </div>
  )
}

/** 拖拽把手光标:悬停标记 grab,其余还原;拖拽中不打扰。 */
function setCursor(map: Map, vecLayer: VectorLayer, pixel: [number, number] | undefined, dragging: { current: boolean }) {
  if (dragging.current || !pixel) return
  const el = map.getTargetElement()
  if (!el) return
  const hit = map.hasFeatureAtPixel(pixel, { layerFilter: (l) => l === vecLayer })
  el.style.cursor = hit ? 'grab' : ''
}

/** 标记样式:品牌金圆点 + 白描边(主题感知)。 */
function markerStyle(theme: Theme): Style {
  return new Style({
    image: new Circle({
      radius: 9,
      fill: new Fill({ color: markerColor(theme) }),
      stroke: new Stroke({ color: '#FFFFFF', width: 2 }),
    }),
  })
}
