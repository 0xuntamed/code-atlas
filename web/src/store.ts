import { create } from 'zustand'
import type { ArchitectureCrumb, ImpactFilter } from './types'

interface AtlasState {
  projectId: string
  selectedEntityId: string
  // Selecting any node reveals its blast radius; this chooses which side to show.
  impactFilter: ImpactFilter
  // Drill-down trail for the structural map (overview → module → file). Lives in
  // the store so it survives remounts and stays consistent with the selection.
  scopePath: ArchitectureCrumb[]
  showTests: boolean
  showReferences: boolean
  setProject: (id: string) => void
  selectEntity: (id: string) => void
  setImpactFilter: (filter: ImpactFilter) => void
  openScope: (crumb: ArchitectureCrumb) => void
  navigateScope: (index: number) => void
  toggleTests: () => void
  toggleReferences: () => void
  resetSelection: () => void
}

export const useAtlasStore = create<AtlasState>((set) => ({
  projectId: '',
  selectedEntityId: '',
  impactFilter: 'both',
  scopePath: [],
  showTests: false,
  showReferences: false,
  setProject: (projectId) =>
    set({ projectId, selectedEntityId: '', impactFilter: 'both', scopePath: [] }),
  selectEntity: (selectedEntityId) => set({ selectedEntityId }),
  setImpactFilter: (impactFilter) => set({ impactFilter }),
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
