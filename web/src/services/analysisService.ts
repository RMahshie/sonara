import { api } from './api'
import axios from 'axios'

export interface CreateAnalysisResponse {
  id: string
  upload_url: string
  expires_in: number
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
  // Create analysis and get upload URL
  createAnalysis: async (sessionId: string, file: File): Promise<CreateAnalysisResponse> => {
    const response = await api.post<CreateAnalysisResponse>('/analyses', {
      session_id: sessionId,
      file_size: file.size,
      mime_type: file.type
    })

    return response.data
  },

  // Upload file directly to S3
  uploadToS3: async (uploadUrl: string, file: File, onProgress?: (progress: number) => void): Promise<void> => {
    await axios.put(uploadUrl, file, {
      headers: {
        'Content-Type': file.type,
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total)
          onProgress(percent)
        }
      },
    })
  },

  // Start processing the uploaded file
  startProcessing: async (analysisId: string): Promise<void> => {
    await api.post(`/analyses/${analysisId}/process`)
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

