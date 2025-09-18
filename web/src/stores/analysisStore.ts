import { create } from 'zustand'

export interface Analysis {
  id: string
  filename: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  progress: number
  createdAt: Date
  completedAt?: Date
  results?: {
    duration: string
    sampleRate: number
    frequencies: Array<{ frequency: number; amplitude: number }>
  }
}

interface AnalysisStore {
  analyses: Analysis[]
  currentAnalysis: Analysis | null
  setCurrentAnalysis: (analysis: Analysis | null) => void
  addAnalysis: (analysis: Analysis) => void
  updateAnalysis: (id: string, updates: Partial<Analysis>) => void
  getAnalysis: (id: string) => Analysis | undefined
}

export const useAnalysisStore = create<AnalysisStore>((set, get) => ({
  analyses: [],
  currentAnalysis: null,

  setCurrentAnalysis: (analysis) => set({ currentAnalysis: analysis }),

  addAnalysis: (analysis) => set((state) => ({
    analyses: [...state.analyses, analysis]
  })),

  updateAnalysis: (id, updates) => set((state) => ({
    analyses: state.analyses.map(analysis =>
      analysis.id === id ? { ...analysis, ...updates } : analysis
    ),
    currentAnalysis: state.currentAnalysis?.id === id
      ? { ...state.currentAnalysis, ...updates }
      : state.currentAnalysis
  })),

  getAnalysis: (id) => get().analyses.find(analysis => analysis.id === id),
}))