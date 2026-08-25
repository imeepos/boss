// tile-source 单测:主题派生 + URL 默认值 + key 注入 + readDocumentTheme(mock document)。
import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { LIGHT_TILE_URL, DARK_TILE_URL, readDocumentTheme, resolveTileUrl } from './tile-source'

describe('tile-source', () => {
  describe('URL 常量', () => {
    it('亮色默认走高德地图栅格瓦片', () => {
      expect(LIGHT_TILE_URL).toContain('is.autonavi.com')
      expect(LIGHT_TILE_URL).toContain('style=8')
      expect(LIGHT_TILE_URL).toMatch(/[?&]x=\{x\}&y=\{y\}&z=\{z\}/)
    })
    it('暗色默认走 CartoDB Dark Matter(高德无原生暗色)', () => {
      expect(DARK_TILE_URL).toContain('basemaps.cartocdn.com')
      expect(DARK_TILE_URL).toContain('dark_all')
    })
  })
  describe('resolveTileUrl', () => {
    it('light 默认=高德,暗色默认=CartoDB Dark Matter', () => {
      expect(resolveTileUrl('light')).toBe(LIGHT_TILE_URL)
      expect(resolveTileUrl('dark')).toBe(DARK_TILE_URL)
    })
    it('显式传 URL 优先于主题默认值', () => {
      expect(resolveTileUrl('dark', 'https://example.com/{z}/{x}/{y}.png')).toBe('https://example.com/{z}/{x}/{y}.png')
    })
    it('key 存在时拼接 key 参数(仅高德瓦片)', () => {
      const got = resolveTileUrl('light', undefined, 'test-key')
      expect(got).toContain('is.autonavi.com')
      expect(got).toContain('key=test-key')
      // 非高德 URL 不追加 key
      const carto = resolveTileUrl('dark', undefined, 'test-key')
      expect(carto).not.toContain('key=')
    })
    it('无 key 时不加参数', () => {
      const got = resolveTileUrl('light', undefined, '')
      expect(got).not.toContain('key=')
    })
  })
  describe('readDocumentTheme', () => {
    let fakeDoc: { documentElement: { getAttribute: (n: string) => string | null } }
    beforeEach(() => {
      fakeDoc = { documentElement: { getAttribute: vi.fn() } }
      vi.stubGlobal('document', fakeDoc)
    })
    afterEach(() => { vi.unstubAllGlobals() })
    it('无 data-theme 返回 light', () => {
      vi.mocked(fakeDoc.documentElement.getAttribute).mockReturnValue(null)
      expect(readDocumentTheme()).toBe('light')
    })
    it('data-theme=dark 返回 dark', () => {
      vi.mocked(fakeDoc.documentElement.getAttribute).mockReturnValue('dark')
      expect(readDocumentTheme()).toBe('dark')
    })
    it('data-theme=其它值兜底为 light', () => {
      vi.mocked(fakeDoc.documentElement.getAttribute).mockReturnValue('sepia')
      expect(readDocumentTheme()).toBe('light')
    })
  })
})