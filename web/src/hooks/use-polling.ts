import { useEffect, useCallback } from 'react'

export function usePolling(fn: () => Promise<void>, intervalMs = 3000) {
  const stableFn = useCallback(fn, [fn])

  useEffect(() => {
    stableFn()
    const id = setInterval(stableFn, intervalMs)
    return () => clearInterval(id)
  }, [stableFn, intervalMs])
}
