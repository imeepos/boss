// 服务端检索数据流 hook:关键字防抖 commit + 纯状态机(pickerSearchReducer)+ seq 防竞态 + 失败重试。
// SimplePicker 服务端模式唯一数据源;防抖毫秒与 DialogPicker 关键字检索统一用 PICKER_DEBOUNCE_MS。
import { useCallback, useEffect, useReducer, useRef, useState } from 'react'
import { PICKER_DEBOUNCE_MS, initialPickerSearchState, pickerSearchReducer } from './pickerCore'

export interface UseServerPickerSearchArgs<T> {
  /** 服务端关键字检索;undefined 时 hook 空转(静态源场景)。 */
  fetcher?: (keyword: string) => Promise<T[] | null>
  debounceMs?: number
}

export function useServerPickerSearch<T>({ fetcher, debounceMs }: UseServerPickerSearchArgs<T>) {
  const [state, dispatch] = useReducer(pickerSearchReducer<T>, undefined, initialPickerSearchState<T>)
  const [keyword, setKeyword] = useState('')
  const [committed, setCommitted] = useState('')
  const [retryTick, setRetryTick] = useState(0)
  const fetcherRef = useRef(fetcher)
  fetcherRef.current = fetcher

  // 关键字防抖:停顿超过 debounceMs 才 commit 发请求;卸载/重打字即取消。
  useEffect(() => {
    const timer = setTimeout(() => setCommitted(keyword), debounceMs ?? PICKER_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [keyword, debounceMs])

  // committed/retry 变化即发请求;seq 单增,过期响应由 reducer 丢弃(防竞态)。
  useEffect(() => {
    const req = fetcherRef.current
    if (!req) return undefined
    const seq = state.reqSeq + 1
    dispatch({ type: 'request', seq })
    let alive = true
    req(committed)
      .then((items) => { if (alive) dispatch({ type: 'ok', seq, items: items ?? [] }) })
      .catch(() => { if (alive) dispatch({ type: 'fail', seq }) })
    return () => { alive = false }
    // state.reqSeq 是已发出的最新请求序号,仅在本效应重放时读取。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [committed, retryTick])

  const retry = useCallback(() => setRetryTick((n) => n + 1), [])
  return { keyword, setKeyword, retry, items: state.items, loading: state.loading, error: state.error }
}
