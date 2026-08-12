import { create } from 'zustand'
import type { ArchitectureCrumb, Lens } from './types'

interface AtlasState {
  projectId: string
  selectedEntityId: string
  lens: Lens
  // Drill-down trail for the structural map (overview → module → file). Lives in
  // the store so it survives remounts and stays consistent with the selection.
  scopePath: ArchitectureCrumb[]
  showTests: boolean
  showReferences: boolean
  setProject: (id: string) => void
  selectEntity: (id: string) => void
  setLens: (lens: Lens) => void
  openScope: (crumb: ArchitectureCrumb) => void
  navigateScope: (index: number) => void
  toggleTests: () => void
  toggleReferences: () => void
  resetSelection: () => void
}

export const useAtlasStore = create<AtlasState>((set) => ({
  projectId: '',
  selectedEntityId: '',
  lens: 'structure',
  scopePath: [],
  showTests: false,
  showReferences: false,
  setProject: (projectId) =>
    set({ projectId, selectedEntityId: '', lens: 'structure', scopePath: [] }),
  selectEntity: (selectedEntityId) => set({ selectedEntityId }),
  setLens: (lens) => set({ lens }),
  openScope: (crumb) =>
    set((state) => {
      const existing = state.scopePath.findIndex((entry) => entry.id === crumb.id)
      const scopePath =
        existing >= 0 ? state.scopePath.slice(0, existing + 1) : [...state.scopePath, crumb]
      return { scopePath, selectedEntityId: crumb.id }
    }),
  navigateScope: (index) =>
    set((state) => {
      if (index < 0) return { scopePath: [], selectedEntityId: '' }
      return {
        scopePath: state.scopePath.slice(0, index + 1),
        selectedEntityId: state.scopePath[index].id,
      }
    }),
  toggleTests: () => set((state) => ({ showTests: !state.showTests })),
  toggleReferences: () => set((state) => ({ showReferences: !state.showReferences })),
  resetSelection: () => set({ selectedEntityId: '' }),
}))
