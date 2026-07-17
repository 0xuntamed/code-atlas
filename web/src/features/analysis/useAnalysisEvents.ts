import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { AnalysisRun, Project } from '../../types'

export function useAnalysisEvents(project: Project): AnalysisRun | undefined {
  const queryClient = useQueryClient()
  const [liveRun, setLiveRun] = useState<AnalysisRun>()

  useEffect(() => {
    if (project.status === 'ready' || project.latestRun?.status === 'failed') {
      return
    }

    const events = new EventSource(`/api/v1/projects/${project.id}/events`)
    const handleProgress = (event: Event) => {
      const nextRun = JSON.parse((event as MessageEvent).data) as AnalysisRun
      setLiveRun(nextRun)
      if (nextRun.status === 'ready' || nextRun.status === 'failed') {
        void queryClient.invalidateQueries({ queryKey: ['projects'] })
        events.close()
      }
    }

    events.addEventListener('progress', handleProgress)
    return () => events.close()
  }, [project.id, project.latestRun?.status, project.status, queryClient])

  return liveRun?.id === project.latestRun?.id ? liveRun : project.latestRun
}
