import { api } from './api'

export interface UploadResponse {
  analysisId: string
  uploadUrl: string
}

export interface AnalysisStatus {
  id: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  progress: number
  filename: string
  createdAt: string
  completedAt?: string
}

export interface AnalysisResult {
  id: string
  filename: string
  duration: string
  sampleRate: number
  frequencies: Array<{ frequency: number; amplitude: number }>
  createdAt: string
  completedAt: string
}

export const analysisService = {
  // Upload audio file
  uploadFile: async (file: File): Promise<UploadResponse> => {
    const formData = new FormData()
    formData.append('file', file)

    const response = await api.post<UploadResponse>('/analyses', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })

    return response.data
  },

  // Get analysis status
  getAnalysisStatus: async (id: string): Promise<AnalysisStatus> => {
    const response = await api.get<AnalysisStatus>(`/analyses/${id}/status`)
    return response.data
  },

  // Get analysis results
  getAnalysisResults: async (id: string): Promise<AnalysisResult> => {
    const response = await api.get<AnalysisResult>(`/analyses/${id}/results`)
    return response.data
  },

  // Get all analyses
  getAnalyses: async (): Promise<AnalysisStatus[]> => {
    const response = await api.get<AnalysisStatus[]>('/analyses')
    return response.data
  },
}

