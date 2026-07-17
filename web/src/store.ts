import { create } from 'zustand'
import type { GraphView } from './types'

interface AtlasState {
  projectId: string
  selectedEntityId: string
  graphView: GraphView
  setProject: (id: string) => void
  selectEntity: (id: string) => void
  setGraphView: (view: GraphView) => void
  resetSelection: () => void
}

export const useAtlasStore = create<AtlasState>((set) => ({
  projectId: '',
  selectedEntityId: '',
  graphView: 'architecture',
  setProject: (projectId) => set({ projectId, selectedEntityId: '', graphView: 'architecture' }),
  selectEntity: (selectedEntityId) => set({ selectedEntityId }),
  setGraphView: (graphView) => set({ graphView }),
  resetSelection: () => set({ selectedEntityId: '' }),
}))
