import { beforeEach, describe, expect, it } from 'vitest'
import { useAtlasStore } from './store'

describe('atlas view state', () => {
  beforeEach(() =>
    useAtlasStore.setState({ projectId: '', selectedEntityId: '', graphView: 'architecture' }),
  )

  it('resets graph context when switching projects', () => {
    useAtlasStore.getState().setProject('project-a')
    useAtlasStore.getState().selectEntity('entity-a')
    useAtlasStore.getState().setGraphView('impact')
    useAtlasStore.getState().setProject('project-b')

    expect(useAtlasStore.getState()).toMatchObject({
      projectId: 'project-b',
      selectedEntityId: '',
      graphView: 'architecture',
    })
  })

  it('clears the selected entity without changing the active graph mode', () => {
    useAtlasStore.getState().setProject('project-a')
    useAtlasStore.getState().selectEntity('entity-a')
    useAtlasStore.getState().setGraphView('impact')
    useAtlasStore.getState().resetSelection()

    expect(useAtlasStore.getState()).toMatchObject({
      projectId: 'project-a',
      selectedEntityId: '',
      graphView: 'impact',
    })
  })
})
