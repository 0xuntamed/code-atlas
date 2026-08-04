import { create } from 'zustand'
import type { GraphView } from './types'

interface AtlasState {
  projectId: string
  selectedEntityId: string
  graphView: GraphView
  showTests: boolean
  showReferences: boolean
  setProject: (id: string) => void
  selectEntity: (id: string) => void
  setGraphView: (view: GraphView) => void
  toggleTests: () => void
  toggleReferences: () => void
  resetSelection: () => void
}

export const useAtlasStore = create<AtlasState>((set) => ({
  projectId: '',
  selectedEntityId: '',
  graphView: 'architecture',
  showTests: false,
  showReferences: false,
  setProject: (projectId) => set({ projectId, selectedEntityId: '', graphView: 'architecture' }),
  selectEntity: (selectedEntityId) => set({ selectedEntityId }),
  setGraphView: (graphView) => set({ graphView }),
  toggleTests: () => set((state) => ({ showTests: !state.showTests })),
  toggleReferences: () => set((state) => ({ showReferences: !state.showReferences })),
  resetSelection: () => set({ selectedEntityId: '' }),
}))
