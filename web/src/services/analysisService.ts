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
  message?: string
  results_id?: string
}

export interface AnalysisResult {
  id: string
  frequency_data: Array<{ frequency: number; magnitude: number }>
  rt60?: number
  room_modes?: number[]
  room_info?: any
  created_at: string
}

export interface RoomInfoInput {
  room_length?: number
  room_width?: number
  room_height?: number
  room_size: string
  ceiling_height: string
  floor_type: string
  features: string[]
  speaker_placement: string
  additional_notes: string
}

export const analysisService = {
  // Create analysis and get upload URL
  createAnalysis: async (sessionId: string, file: File, signalId: string): Promise<CreateAnalysisResponse> => {
    const response = await api.post<CreateAnalysisResponse>('/analyses', {
      session_id: sessionId,
      file_size: file.size,
      mime_type: file.type,
      signal_id: signalId
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

  // Add room information to analysis
  addRoomInfo: async (analysisId: string, roomInfo: RoomInfoInput): Promise<void> => {
    await api.post(`/analyses/${analysisId}/room-info`, roomInfo)
  },

}

