import { useEffect, useState } from 'react'

/** useDebouncedValue:检索输入防抖,默认 300ms(对齐 pickerCore.PICKER_DEBOUNCE_MS)。 */
export function useDebouncedValue<T>(value: T, delayMs = 300): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(t)
  }, [value, delayMs])
  return debounced
}
