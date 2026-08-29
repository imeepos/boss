// 待治理黄标:与地址管理页治理队列同口径的视觉标记;警示不阻断。
export function ReviewBadge({ text }: { text: string }) {
  return (
    <span className="inline-flex shrink-0 items-center rounded-sm border border-[color-mix(in_srgb,var(--color-warning)_40%,transparent)] bg-[color-mix(in_srgb,var(--color-warning)_12%,transparent)] px-1 py-px text-[10px] leading-none text-[var(--color-warning)]">
      {text}
    </span>
  )
}
