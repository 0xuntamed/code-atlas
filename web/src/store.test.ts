import { beforeEach, describe, expect, it } from 'vitest'
import { useAtlasStore } from './store'

describe('atlas workspace state', () => {
  beforeEach(() =>
    useAtlasStore.setState({
      projectId: '',
      selectedEntityId: '',
      graphView: 'architecture',
      showTests: false,
      showReferences: false,
    }),
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

  it('keeps explicit signal filters across graph modes', () => {
    useAtlasStore.getState().toggleTests()
    useAtlasStore.getState().toggleReferences()
    useAtlasStore.getState().setGraphView('flow')
    expect(useAtlasStore.getState()).toMatchObject({
      graphView: 'flow',
      showTests: true,
      showReferences: true,
    })
  })
})
