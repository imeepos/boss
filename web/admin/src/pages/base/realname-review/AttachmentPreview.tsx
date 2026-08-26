// 证件照预览:走 attachments/content 拉 blob,转 Object URL 在 modal 中渲染。
import { useEffect, useState } from 'react'
import { fetchAttachmentFile } from '../../../api/attachments'

interface Props {
  attachmentId: number
  onClose: () => void
}

export function AttachmentPreview({ attachmentId, onClose }: Props) {
  const [src, setSrc] = useState<string | null>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    let url: string | null = null
    fetchAttachmentFile(attachmentId)
      .then((file) => { url = URL.createObjectURL(file); setSrc(url) })
      .catch((e) => setError(e instanceof Error ? e.message : 'load failed'))
    return () => { if (url) URL.revokeObjectURL(url) }
  }, [attachmentId])
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div className="max-h-[90vh] max-w-[90vw] rounded-lg bg-white p-4 shadow-xl" onClick={(e) => e.stopPropagation()}>
        {error
          ? <div className="px-4 py-8 text-sm text-red-600">{error}</div>
          : src
            ? <img src={src} alt="attachment" className="max-h-[80vh] max-w-[88vw] object-contain" />
            : <div className="px-12 py-12 text-sm text-gray-500">…</div>}
      </div>
    </div>
  )
}