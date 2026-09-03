// map-location 纯函数单测:坐标校验/规范化/回显文案/视域决策/受控状态机/换算。
// 组件本体依赖 WebGL,jsdom(node) 不可测;本文件是选点交互的唯一测试面。
import { describe, it, expect } from 'vitest'
import { fromLonLat } from 'ol/proj'
import { DEFAULT_VIEW } from './bbox-utils'
import {
  isValidLatLng, normalizeLatLng, roundCoord, formatLatLng, viewForLatLng,
  mercatorToLatLng, nextPickerValue, markerColor, PICK_ZOOM, type LatLng,
} from './map-location'

const BOUND: { center: [number, number]; zoom: number } = { center: DEFAULT_VIEW.center, zoom: DEFAULT_VIEW.zoom }

describe('isValidLatLng 范围校验(对齐后端 422 前置)', () => {
  it('合法十进制经纬度通过', () => {
    expect(isValidLatLng(14.599, 120.984)).toBe(true)
    expect(isValidLatLng(-45.5, -170.2)).toBe(true)
  })
  it('边界值 ±90/±180 合法', () => {
    expect(isValidLatLng(90, 180)).toBe(true)
    expect(isValidLatLng(-90, -180)).toBe(true)
  })
  it('越界/非有限值拒绝', () => {
    expect(isValidLatLng(91, 120.984)).toBe(false)
    expect(isValidLatLng(14.599, 181)).toBe(false)
    expect(isValidLatLng(NaN, 120)).toBe(false)
    expect(isValidLatLng(14, Infinity)).toBe(false)
  })
  it('0,0(几内亚湾) 与后端 binding:required 同语义拒绝', () => {
    expect(isValidLatLng(0, 0)).toBe(false)
  })
})

describe('normalizeLatLng / roundCoord 受控值规范化', () => {
  it('非法与空归一为 null', () => {
    expect(normalizeLatLng(null)).toBeNull()
    expect(normalizeLatLng(undefined)).toBeNull()
    expect(normalizeLatLng({ lat: 91, lng: 0 })).toBeNull()
  })
  it('合法值取整到 6 位', () => {
    expect(normalizeLatLng({ lat: 14.599123456, lng: 120.984987654 }))
      .toEqual({ lat: 14.599123, lng: 120.984988 })
  })
  it('roundCoord 不放大浮点误差', () => {
    expect(roundCoord(14.1 + 2.2)).toBe(16.3)
  })
})

describe('formatLatLng 回显文案', () => {
  it('有值输出 6 位小数 "lat, lng"', () => {
    expect(formatLatLng({ lat: 14.599, lng: 120.984 })).toBe('14.599000, 120.984000')
    expect(formatLatLng({ lat: -0.5, lng: 179.123456789 })).toBe('-0.500000, 179.123457')
  })
  it('无值/非法返回空串(占位交给消费方)', () => {
    expect(formatLatLng(null)).toBe('')
    expect(formatLatLng({ lat: 91, lng: 0 })).toBe('')
  })
})

describe('viewForLatLng 视域决策', () => {
  it('有值定位到该点,经纬度换 center 顺序,zoom 取 PICK_ZOOM', () => {
    expect(viewForLatLng({ lat: 14.599, lng: 120.984 }, BOUND))
      .toEqual({ center: [120.984, 14.599], zoom: PICK_ZOOM })
  })
  it('无值回退既有默认视野(DEFAULT_VIEW 原样透出)', () => {
    expect(viewForLatLng(null, BOUND)).toBe(BOUND)
    expect(BOUND.center).toEqual(DEFAULT_VIEW.center)
    expect(BOUND.zoom).toBe(DEFAULT_VIEW.zoom)
  })
})

describe('mercatorToLatLng 墨卡托换算', () => {
  it('与 fromLonLat 往返一致(WGS84 域内)', () => {
    const p: LatLng = { lat: 14.599, lng: 120.984 }
    const back = mercatorToLatLng(fromLonLat([p.lng, p.lat]) as [number, number])
    expect(back.lat).toBeCloseTo(p.lat, 9)
    expect(back.lng).toBeCloseTo(p.lng, 9)
  })
})

describe('nextPickerValue 受控状态机', () => {
  const cur: LatLng = { lat: 14.599, lng: 120.984 }
  it('pick 落点替换当前值', () => {
    expect(nextPickerValue(cur, { type: 'pick', value: { lat: 14.6, lng: 120.9 } }))
      .toEqual({ lat: 14.6, lng: 120.9 })
  })
  it('drag 微调更新当前值(含取整)', () => {
    expect(nextPickerValue(cur, { type: 'drag', value: { lat: 14.5991234567, lng: 120.984 } }))
      .toEqual({ lat: 14.599123, lng: 120.984 })
  })
  it('clear 恒清空,与当前值无关', () => {
    expect(nextPickerValue(cur, { type: 'clear' })).toBeNull()
    expect(nextPickerValue(null, { type: 'clear' })).toBeNull()
  })
  it('非法落点保持原值,不产非法坐标', () => {
    expect(nextPickerValue(cur, { type: 'pick', value: { lat: 91, lng: 0 } })).toEqual(cur)
    expect(nextPickerValue(null, { type: 'pick', value: { lat: 0, lng: 0 } })).toBeNull()
  })
})

describe('markerColor 主题色', () => {
  it('light 品牌金,dark 提亮', () => {
    expect(markerColor('light')).toBe('#C69835')
    expect(markerColor('dark')).toBe('#E5C985')
  })
})
