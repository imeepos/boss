// Markdown 富编辑器:工具栏(加粗/斜体/标题/链接/代码/图片上传) + 左右双栏实时预览。
// 预览基于 react-markdown;正文内附件引用 ](att/N) 由 AttImg 走登录态附件端点转 blob 渲染,
// 公开侧由后端详情端点重写为 /site/posts/:slug/img/:id(见 site_img.go)。
import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { fetchAttachmentFile } from '../../../api/attachments'
import { AttachmentPickerDialog } from '../../../components/AttachmentManager/PickerDialog'

// att/N 引用 → 附件 id;非该形态返回 null。
const attId = (src: string): number | null => {
  const m = /^att\/(\d+)$/.exec(src)
  return m ? Number(m[1]) : null
}

// blob 缓存:同一附件在预览里反复渲染不重复拉取。
const blobCache = new Map<number, string>()

function AttImg({ src, alt }: { src?: string; alt?: string }) {
  const id = src ? attId(src) : null
  const [url, setUrl] = useState<string | null>(id ? blobCache.get(id) ?? null : null)
  useEffect(() => {
    if (!id || url) return
    let alive = true
    fetchAttachmentFile(id)
      .then((f) => {
        const u = URL.createObjectURL(f)
        blobCache.set(id, u)
        if (alive) setUrl(u)
      })
      .catch(() => { if (alive) setUrl(null) })
    return () => { alive = false }
  }, [id, url])
  if (!id) return <img src={src} alt={alt} />
  if (url === null) return <span className="text-xs text-[var(--shell-group-title)]">[{alt || `att/${id}`} 加载中…]</span>
  return <img src={url} alt={alt} />
}

const tbBtn = 'h-7 min-w-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-xs text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]'

export function MarkdownEditor({
  value, onChange, texts,
}: {
  value: string
  onChange: (v: string) => void
  texts: { uploadImg: string; uploadFail: string }
}) {
  const ref = useRef<HTMLTextAreaElement>(null)

  // 用选区包裹/行前缀插入;无选区时插入占位符并选中,便于直接键入。
  const apply = (wrap: [string, string] | null, prefix: string, placeholder: string) => {
    const el = ref.current
    if (!el) return
    const { selectionStart: s, selectionEnd: e } = el
    const sel = value.slice(s, e) || placeholder
    const next = wrap
      ? value.slice(0, s) + wrap[0] + sel + wrap[1] + value.slice(e)
      : value.slice(0, s) + prefix + sel + value.slice(e)
    onChange(next)
    requestAnimationFrame(() => {
      el.focus()
      el.setSelectionRange(s + prefix.length, s + prefix.length + sel.length)
    })
  }

  const insert = (text: string) => {
    const el = ref.current
    const s = el?.selectionStart ?? value.length
    onChange(value.slice(0, s) + text + value.slice(s))
  }

  const [pickImg, setPickImg] = useState(false)

  // 选择器(可多选)回填:每张图插入一行 ](att/N) 引用,alt 用文件名去括号。
  const insertPicked = (items: { id: number; fileName: string }[]) => {
    for (const at of items) insert(`![${at.fileName.replace(/[[\]]/g, '')}](att/${at.id})`)
  }

  return <div>
    <div className="mb-1 flex flex-wrap items-center gap-1.5">
      <button type="button" className={tbBtn} title="Bold" onClick={() => apply(['**', '**'], '', '粗体')}><b>B</b></button>
      <button type="button" className={tbBtn} title="Italic" onClick={() => apply(['*', '*'], '', '斜体')}><i>I</i></button>
      <button type="button" className={tbBtn} title="H2" onClick={() => apply(null, '## ', '标题')}>H2</button>
      <button type="button" className={tbBtn} title="H3" onClick={() => apply(null, '### ', '标题')}>H3</button>
      <button type="button" className={tbBtn} title="Code" onClick={() => apply(['`', '`'], '', 'code')}>{'<>'}</button>
      <button type="button" className={tbBtn} title="Link" onClick={() => apply(['[', '](https://)'], '', '链接文字')}>Link</button>
      <button type="button" className={tbBtn} title="List" onClick={() => apply(null, '- ', '列表项')}>List</button>
      <button type="button" className={tbBtn} title={texts.uploadImg} onClick={() => setPickImg(true)}>Image</button>
      </div>
    <div className="grid gap-3 md:grid-cols-2">
      <textarea
        ref={ref}
        className="min-h-96 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] p-3 font-mono text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="Markdown 正文,图片经 Image 按钮上传后以 ](att/N) 引用"
      />
      <div className="min-h-96 overflow-auto rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-card-bg)] p-3 text-[13px] text-[var(--shell-content-text)] [&_h2]:mt-3 [&_h2]:text-base [&_h3]:mt-2 [&_h3]:text-sm [&_img]:max-w-full [&_p]:my-2 [&_pre]:my-2 [&_pre]:rounded-sm [&_pre]:bg-[var(--shell-input-bg)] [&_pre]:p-2">
        <ReactMarkdown components={{ img: ({ src, alt }) => <AttImg src={src} alt={alt} /> }}>
          {value}
        </ReactMarkdown>
      </div>
    </div>
    <AttachmentPickerDialog open={pickImg} onClose={() => setPickImg(false)} multiple imageOnly onPick={insertPicked} />
  </div>
}
