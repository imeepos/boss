// point-layer 单测:GeoJSON 转换与样式参数(纯函数,不依赖 OL 渲染)。
import { describe, expect, it } from 'vitest'
import { pointsToFeatureCollection, pointRadius, pointColor } from './point-layer'
import type { GisPoint } from './point-layer'

const sample: GisPoint[] = [
  { id: 1, level: 1, name: '市', lng: 121.5, lat: 31.2, status: 'AREA', count: 3, parentId: 0 },
  { id: 2, level: 6, name: 'OLT-01', lng: 121.4, lat: 31.1, status: 'ONLINE', count: 16, parentId: 5 },
  { id: 3, level: 6, name: 'OLT-02', lng: 121.6, lat: 31.3, status: 'OFFLINE', count: 0, parentId: 5 },
]

describe('point-layer', () => {
  it('pointsToFeatureCollection 转 GeoJSON', () => {
    const fc = pointsToFeatureCollection(sample)
    expect(fc.type).toBe('FeatureCollection')
    expect(fc.features).toHaveLength(3)
    expect(fc.features[0]?.geometry).toMatchObject({ type: 'Point', coordinates: [121.5, 31.2] })
    expect(fc.features[0]?.properties).toMatchObject({ id: 1, name: '市', status: 'AREA' })
  })
  it('空数组返回空 FeatureCollection', () => {
    expect(pointsToFeatureCollection([]).features).toEqual([])
  })
  it('pointRadius:count=0 或 max=0 兜底为 6', () => {
    expect(pointRadius(0, 0)).toBe(6)
    expect(pointRadius(0, 100)).toBeGreaterThanOrEqual(6)
  })
  it('pointRadius:sqrt 缩放(大 count 不应线性爆炸)', () => {
    expect(pointRadius(100, 100)).toBeGreaterThan(pointRadius(50, 100))
    expect(pointRadius(100, 100)).toBeLessThanOrEqual(20) // 6 + 14*1 = 20
  })
  it('pointColor:ONLINE 金色,其它深蓝(light)', () => {
    expect(pointColor('ONLINE')).toBe('#C69835')
    expect(pointColor('OFFLINE')).toBe('#1F355F')
    expect(pointColor('UNKNOWN')).toBe('#1F355F')
  })
  it('pointColor:dark 主题提亮对比度', () => {
    expect(pointColor('ONLINE', 'dark')).toBe('#E5C985')
    expect(pointColor('OFFLINE', 'dark')).toBe('#9CAAC3')
  })
  it('pointColor:theme 默认 light', () => {
    expect(pointColor('ONLINE')).toBe(pointColor('ONLINE', 'light'))
  })
})