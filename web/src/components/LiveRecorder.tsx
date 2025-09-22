import React, { useState, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import axios from 'axios'
import { api } from '../services/api'
import ProgressBar from './ProgressBar'

export const LiveRecorder: React.FC = () => {
  const [isRecording, setIsRecording] = useState(false)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [phase, setPhase] = useState<'ready' | 'recording' | 'uploading'>('ready')

  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const navigate = useNavigate()

  const startAnalysis = useCallback(async () => {
    setError(null)
    setPhase('recording')

    try {
      // Request microphone access
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: false,
          noiseSuppression: false,
          autoGainControl: false,
          sampleRate: 48000
        }
      })
      streamRef.current = stream

      // Setup MediaRecorder
      const mimeType = MediaRecorder.isTypeSupported('audio/webm')
        ? 'audio/webm'
        : 'audio/ogg'

      const mediaRecorder = new MediaRecorder(stream, { mimeType })
      mediaRecorderRef.current = mediaRecorder

      const chunks: Blob[] = []

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          chunks.push(event.data)
        }
      }

      mediaRecorder.onstop = async () => {
        const audioBlob = new Blob(chunks, { type: mimeType })
        await uploadRecording(audioBlob, mimeType)
        stream.getTracks().forEach(track => track.stop())
      }

      // Play test signal and record simultaneously
      const testSignal = new Audio('/test-signals/pink-noise-10s.wav')
      testSignal.volume = 1.0

      testSignal.ontimeupdate = () => {
        const currentProgress = (testSignal.currentTime / testSignal.duration) * 100
        setProgress(Math.round(currentProgress))
      }

      testSignal.onended = () => {
        mediaRecorder.stop()
        setIsRecording(false)
        setPhase('uploading')
      }

      // Start recording then play test signal
      mediaRecorder.start()
      setIsRecording(true)
      await testSignal.play()

    } catch (err: any) {
      setError(err.message || 'Failed to access microphone. Please check permissions.')
      setPhase('ready')
      setIsRecording(false)
    }
  }, [navigate])

  const uploadRecording = async (audioBlob: Blob, mimeType: string) => {
    try {
      // Get or create session ID
      let sessionId = localStorage.getItem('sonara_session_id')
      if (!sessionId) {
        sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
        localStorage.setItem('sonara_session_id', sessionId)
      }

      // Create analysis
      const { data } = await api.post('/analyses', {
        session_id: sessionId,
        file_size: audioBlob.size,
        mime_type: mimeType
      })

      const { id: analysisId, upload_url } = data

      // Upload to S3
      await axios.put(upload_url, audioBlob, {
        headers: {
          'Content-Type': mimeType
        },
        onUploadProgress: (progressEvent) => {
          const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total!)
          setProgress(percent)
        }
      })

      // Navigate to analysis page
      navigate(`/analysis/${analysisId}`)

    } catch (err: any) {
      setError('Upload failed. Please try again.')
      setPhase('ready')
    }
  }

  const stopAnalysis = useCallback(() => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop()
      setIsRecording(false)
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(track => track.stop())
    }
  }, [isRecording])

  return (
    <div className="w-full">
      <div
        className={`
          border-2 border-dashed rounded-xl p-12 text-center transition-all duration-200
          ${error ? 'border-red-300 bg-red-50' : 'border-racing-green/30 hover:border-racing-green/60'}
          ${isRecording ? 'pointer-events-none opacity-50' : ''}
        `}
      >
        {phase === 'ready' && !isRecording && (
          <div className="space-y-4">
            <div className="text-6xl text-racing-green/40">🎤</div>
            <div>
              <p className="text-xl font-medium text-racing-green mb-2">
                Room Acoustic Analysis
              </p>
              <p className="text-racing-green/60 mb-4">
                Click to analyze your room acoustics
              </p>
              <button
                onClick={startAnalysis}
                className="btn-primary mt-4"
                disabled={isRecording || phase !== 'ready'}
              >
                Analyze Room
              </button>
            </div>
          </div>
        )}

        {phase === 'recording' && isRecording && (
          <div className="space-y-4">
            <div className="text-6xl text-racing-green/40 animate-pulse">🔴</div>
            <div className="text-racing-green font-medium">
              Recording in progress...
            </div>
            <ProgressBar progress={progress} />
            <p className="text-sm text-racing-green/60">
              Please remain quiet during the measurement
            </p>
            <button
              onClick={stopAnalysis}
              className="text-red-600 hover:text-red-700 text-sm"
            >
              Cancel Recording
            </button>
          </div>
        )}

        {phase === 'uploading' && (
          <div className="space-y-4">
            <div className="text-6xl text-racing-green/40">⏳</div>
            <div className="text-racing-green font-medium">
              Analyzing your recording...
            </div>
            <ProgressBar progress={progress} />
            <p className="text-sm text-racing-green/60">
              This may take a moment
            </p>
          </div>
        )}

        {error && (
          <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-red-600 text-sm">{error}</p>
            <button
              onClick={() => {
                setError(null)
                setPhase('ready')
              }}
              className="mt-2 text-red-600 hover:text-red-700 text-sm underline"
            >
              Try Again
            </button>
          </div>
        )}

        {/* Setup instructions */}
        {phase === 'ready' && !error && (
          <div className="mt-8 text-xs text-racing-green/60 max-w-md mx-auto">
            <p className="mb-2 font-medium text-base">Quick Setup:</p>
            <ul className="space-y-1">
              <li>• Set speakers to normal listening volume</li>
              <li>• Position microphone at listening position</li>
              <li>• Minimize background noise</li>
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}

export default LiveRecorder
