import { useCallback, useEffect, useState } from 'react'
import App from './App'
import { Landing } from './features/landing/Landing'

// The app has exactly two top-level views, so it uses a minimal path-based
// router instead of a routing dependency — in keeping with the lean,
// single-binary ethos. Real URL paths (via the History API) mean deep links and
// refreshes work: the Go server already falls back to index.html for any
// non-file path (see internal/webui/webui.go).
const APP_PREFIX = '/app'

function useLocationPath(): string {
  const [path, setPath] = useState(() => window.location.pathname)
  useEffect(() => {
    const onPopState = () => setPath(window.location.pathname)
    window.addEventListener('popstate', onPopState)
    return () => window.removeEventListener('popstate', onPopState)
  }, [])
  return path
}

export function Root() {
  const path = useLocationPath()

  const navigate = useCallback((to: string) => {
    if (to === window.location.pathname) return
    window.history.pushState({}, '', to)
    // pushState doesn't emit popstate, so nudge listeners ourselves.
    window.dispatchEvent(new PopStateEvent('popstate'))
  }, [])

  if (path.startsWith(APP_PREFIX)) return <App />
  return <Landing onEnterDemo={() => navigate(APP_PREFIX)} />
}
