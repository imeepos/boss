// tile-source 单测:主题派生 + URL 默认值 + readDocumentTheme(mock document)。
import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { LIGHT_TILE_URL, DARK_TILE_URL, readDocumentTheme } from './tile-source'

describe('tile-source', () => {
  describe('URL 常量', () => {
    it('亮色默认走 OSM 公开瓦片', () => {
      expect(LIGHT_TILE_URL).toContain('openstreetmap.org')
      expect(LIGHT_TILE_URL).toMatch(/\{z\}\/\{x\}\/\{y\}\.png/)
    })
    it('暗色默认走 CartoDB Dark Matter', () => {
      expect(DARK_TILE_URL).toContain('basemaps.cartocdn.com')
      expect(DARK_TILE_URL).toContain('dark_all')
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