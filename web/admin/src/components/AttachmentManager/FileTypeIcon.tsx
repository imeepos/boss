// 文件类型彩色徽章组件:按 category 渲染 32×32 圆角浅底 + 18px 描边 SVG 图标。
// 颜色由 src/theme/tokens.css 的 .afti-<category> 类定义,自动按 [data-theme] 切换。
import type { ComponentType } from 'react'
import {
  Archive, File, FileAudio, FileCode, FileImage, FileSpreadsheet,
  FileText, FileVideo,
} from 'lucide-react'
import type { FileCategoryKey } from './logic'

export interface CategoryStyle {
  key: FileCategoryKey
  Icon: ComponentType<{ className?: string }>
}

export const CATEGORY_STYLES: Record<FileCategoryKey, CategoryStyle> = {
  image:       { key: 'image',       Icon: FileImage },
  audio:       { key: 'audio',       Icon: FileAudio },
  video:       { key: 'video',       Icon: FileVideo },
  pdf:         { key: 'pdf',         Icon: FileText },
  document:    { key: 'document',    Icon: FileText },
  spreadsheet: { key: 'spreadsheet', Icon: FileSpreadsheet },
  archive:     { key: 'archive',     Icon: Archive },
  code:        { key: 'code',        Icon: FileCode },
  other:       { key: 'other',       Icon: File },
}

export function FileTypeIcon({ category, size = 32 }: { category: FileCategoryKey; size?: number }) {
  const style = CATEGORY_STYLES[category]
  return (
    <span
      className={`afti-${category} inline-flex shrink-0 items-center justify-center rounded-md`}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      <style.Icon className="h-[18px] w-[18px]" />
    </span>
  )
}