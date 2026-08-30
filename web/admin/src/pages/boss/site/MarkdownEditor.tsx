// Markdown 富编辑器:工具栏(加粗/斜体/标题/链接/代码/图片上传) + 左右双栏实时预览。
// 预览基于 react-markdown;正文内附件引用 ](att/N) 由 AttImg 走登录态附件端点转 blob 渲染,
// 公开侧由后端详情端点重写为 /site/posts/:slug/img/:id?lang=(见 site_img.go)。
// 文案走 i18n(sitePage.md*);工具栏按钮/编辑框接入 ui Button / ui Textarea,紧凑形态用 className 覆盖。
import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { fetchAttachmentFile } from '../../../api/attachments'
import { AttachmentPickerDialog } from '../../../components/AttachmentManager/PickerDialog'
import { Button } from '../../../components/ui/button'
import { Textarea } from '../../../components/ui/textarea'
import { useT } from '../../../i18n'

// att/N 引用 → 附件 id;非该形态返回 null。
const attId = (src: string): number | null => {
  const m = /^att\/(\d+)$/.exec(src)
  return m ? Number(m[1]) : null
}

// blob 缓存:同一附件在预览里反复渲染不重复拉取。
const blobCache = new Map<number, string>()

function AttImg({ src, alt }: { src?: string; alt?: string }) {
  const t = useT()
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
  if (url === null) return <span className="text-xs text-[var(--shell-group-title)]">[{alt || `att/${id}`} {t.pages.sitePage.mdLoading}]</span>
  return <img src={url} alt={alt} />
}

// 工具栏紧凑按钮:DS Button outline+sm 打底,覆盖为 7px 高微型工具钮。
const tbBtn = 'h-7 min-w-7 px-2 text-xs font-normal'

export function MarkdownEditor({
  value, onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  const t = useT(); const s = t.pages.sitePage
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
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdBold} onClick={() => apply(['**', '**'], '', s.mdBold)}><b>B</b></Button>
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdItalic} onClick={() => apply(['*', '*'], '', s.mdItalic)}><i>I</i></Button>
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdH2} onClick={() => apply(null, '## ', s.mdH2)}>H2</Button>
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdH3} onClick={() => apply(null, '### ', s.mdH3)}>H3</Button>
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdCode} onClick={() => apply(['`', '`'], '', s.mdCode)}>{'<>'}</Button>
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdLink} onClick={() => apply(['[', '](https://)'], '', s.mdLink)}>Link</Button>
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdList} onClick={() => apply(null, '- ', s.mdList)}>List</Button>
      <Button type="button" variant="outline" size="sm" className={tbBtn} title={s.mdImage} onClick={() => setPickImg(true)}>Image</Button>
    </div>
    <div className="grid gap-3 md:grid-cols-2">
      <Textarea
        ref={ref}
        className="min-h-96 rounded-sm p-3 font-mono text-[13px]"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={s.mdPlaceholder}
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
