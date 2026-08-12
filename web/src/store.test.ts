import { beforeEach, describe, expect, it } from 'vitest'
import { useAtlasStore } from './store'

describe('atlas workspace state', () => {
  beforeEach(() =>
    useAtlasStore.setState({
      projectId: '',
      selectedEntityId: '',
      lens: 'structure',
      scopePath: [],
      showTests: false,
      showReferences: false,
    }),
  )

  it('resets graph context when switching projects', () => {
    const store = useAtlasStore.getState()
    store.setProject('project-a')
    store.selectEntity('entity-a')
    store.setLens('impact')
    store.openScope({ id: 'mod', name: 'mod', kind: 'module' })
    store.setProject('project-b')
    expect(useAtlasStore.getState()).toMatchObject({
      projectId: 'project-b',
      selectedEntityId: '',
      lens: 'structure',
      scopePath: [],
    })
  })

  it('keeps explicit signal filters across lenses', () => {
    const store = useAtlasStore.getState()
    store.toggleTests()
    store.toggleReferences()
    store.setLens('flow')
    expect(useAtlasStore.getState()).toMatchObject({
      lens: 'flow',
      showTests: true,
      showReferences: true,
    })
  })

  it('tracks and rewinds the drill-down scope path', () => {
    const store = useAtlasStore.getState()
    store.openScope({ id: 'a', name: 'a', kind: 'module' })
    store.openScope({ id: 'b', name: 'b', kind: 'file' })
    expect(useAtlasStore.getState().scopePath.map((crumb) => crumb.id)).toEqual(['a', 'b'])

    // Re-opening an ancestor truncates the trail instead of duplicating it.
    store.openScope({ id: 'a', name: 'a', kind: 'module' })
    expect(useAtlasStore.getState().scopePath.map((crumb) => crumb.id)).toEqual(['a'])
    expect(useAtlasStore.getState().selectedEntityId).toBe('a')

    store.navigateScope(-1)
    expect(useAtlasStore.getState().scopePath).toEqual([])
    expect(useAtlasStore.getState().selectedEntityId).toBe('')
  })
})
