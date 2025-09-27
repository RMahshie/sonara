import React, { useState, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { analysisService } from '../services/analysisService'
import ProgressBar from './ProgressBar'

// Test signal configuration
const TEST_SIGNAL = "sine_sweep_20_20k"

export const LiveRecorder: React.FC = () => {
  const [isRecording, setIsRecording] = useState(false)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [phase, setPhase] = useState<'ready' | 'room-input' | 'recording' | 'processing'>('ready')
  const [roomDimensions, setRoomDimensions] = useState({
    length: '',
    width: '',
    height: ''
  })

  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const navigate = useNavigate()

  // Form validation for room dimensions
  const isRoomFormValid = () => {
    // All empty = valid (skip room analysis)
    if (!roomDimensions.length && !roomDimensions.width && !roomDimensions.height) {
      return true
    }
    // All filled with valid numbers = valid
    return [roomDimensions.length, roomDimensions.width, roomDimensions.height]
      .every(dim => dim === '' || (parseFloat(dim) > 0 && parseFloat(dim) < 50))
  }

  // Go to room input phase
  const handleAnalyzeRoom = useCallback(() => {
    setError(null)
    setPhase('room-input')
  }, [])

  // Start recording logic (shared between entry points)
  const startRecording = useCallback(async () => {
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

        // Validate recording before upload
        if (audioBlob.size < 1000) {
          setError('Recording failed. Please check your microphone.')
          setPhase('ready')
          stream.getTracks().forEach(track => track.stop())
          return
        }

        if (audioBlob.size > 20 * 1024 * 1024) {
          setError('Recording too large. Please try a shorter recording.')
          setPhase('ready')
          stream.getTracks().forEach(track => track.stop())
          return
        }

        await uploadRecording(audioBlob, mimeType)
        stream.getTracks().forEach(track => track.stop())
      }

      // Play test signal and record simultaneously
      const testSignal = new Audio('/test-signals/sweep-20-20k-10s.wav')
      testSignal.volume = 1.0

      // Add error handlers for test signal
      testSignal.onerror = () => {
        setError('Test signal failed to load. Please check your setup.')
        setPhase('ready')
        setIsRecording(false)
        mediaRecorder.stop()
        stream.getTracks().forEach(track => track.stop())
        return
      }

      testSignal.oncanplaythrough = () => {
        // Signal loaded successfully
      }

      testSignal.ontimeupdate = () => {
        const currentProgress = (testSignal.currentTime / testSignal.duration) * 100
        setProgress(Math.round(currentProgress))
      }

      testSignal.onended = () => {
        mediaRecorder.stop()
        setIsRecording(false)
        setPhase('processing')
      }

      // Start recording then play test signal
      mediaRecorder.start()
      setIsRecording(true)

      try {
        await testSignal.play()
      } catch (playError) {
        setError('Cannot play test signal. Check audio permissions.')
        setPhase('ready')
        setIsRecording(false)
        mediaRecorder.stop()
        stream.getTracks().forEach(track => track.stop())
        return
      }

    } catch (err: any) {
      setError(err.message || 'Failed to access microphone. Please check permissions.')
      setPhase('ready')
      setIsRecording(false)
    }
  }, [navigate])


  // Entry point from room dimensions form
  const startRecordingWithRoomInfo = useCallback(() => {
    setError(null)
    setPhase('recording')
    startRecording()
  }, [startRecording])

  const uploadRecording = async (audioBlob: Blob, mimeType: string) => {
    try {
      // Get or create session ID
      let sessionId = localStorage.getItem('sonara_session_id')
      if (!sessionId) {
        sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
        localStorage.setItem('sonara_session_id', sessionId)
      }

      // Convert blob to file for the service
      const file = new File([audioBlob], 'recording', { type: mimeType })

      // Phase 1: Create analysis and get upload URL
      const { id: analysisId, upload_url: uploadUrl } = await analysisService.createAnalysis(sessionId, file, TEST_SIGNAL)

      // Send room dimensions if provided
      if (roomDimensions.length || roomDimensions.width || roomDimensions.height) {
        try {
          await analysisService.addRoomInfo(analysisId, {
            room_length: roomDimensions.length ? parseFloat(roomDimensions.length) : undefined,
            room_width: roomDimensions.width ? parseFloat(roomDimensions.width) : undefined,
            room_height: roomDimensions.height ? parseFloat(roomDimensions.height) : undefined,
            room_size: 'medium', // Default values for required fields
            ceiling_height: 'standard',
            floor_type: 'hardwood',
            features: [],
            speaker_placement: 'desk',
            additional_notes: ''
          })
        } catch (err: any) {
          console.warn('Failed to save room info, continuing with analysis:', err)
          // Don't block the analysis if room info fails
        }
      }

      // Upload to S3 with progress (0-50% of total progress)
      await analysisService.uploadToS3(uploadUrl, file, (uploadProgress) => {
        const totalProgress = Math.round(uploadProgress * 0.5) // 50% of total
        setProgress(totalProgress)
      })

      // Start backend processing
      setProgress(50) // Start processing phase at 50%

      await analysisService.startProcessing(analysisId)

      // Navigate to analysis page (processing will continue in background)
      navigate(`/analysis/${analysisId}`)

    } catch (err: any) {
      // Use backend-provided error messages when available
      if (err.response?.data?.message) {
        setError(err.response.data.message)
      } else if (err.response?.status === 413) {
        setError('Recording file is too large for upload.')
      } else if (!navigator.onLine) {
        setError('No internet connection. Please check your network.')
      } else {
        setError('Upload failed. Please try again.')
      }
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
                onClick={handleAnalyzeRoom}
                className="btn-primary mt-4"
                disabled={isRecording || phase !== 'ready'}
              >
                Analyze Room
              </button>
            </div>
          </div>
        )}

        {phase === 'room-input' && (
          <div className="space-y-4">
            <div className="text-6xl text-racing-green/40">📏</div>
            <div>
              <p className="text-xl font-medium text-racing-green mb-2">
                Room Dimensions (Optional)
              </p>
              <p className="text-racing-green/60 mb-4">
                Enter your room dimensions for enhanced resonance analysis
              </p>

              {/* Form fields */}
              <div className="space-y-3 mb-6">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                  <div>
                    <label className="block text-sm text-racing-green/70 mb-1">
                      Length (m)
                    </label>
                    <input
                      type="number"
                      step="0.1"
                      min="0.1"
                      max="50"
                      placeholder="e.g. 4.5"
                      className="w-full px-3 py-2 border border-racing-green/30 rounded-md focus:outline-none focus:ring-2 focus:ring-racing-green/50"
                      value={roomDimensions.length}
                      onChange={(e) => setRoomDimensions(prev => ({...prev, length: e.target.value}))}
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-racing-green/70 mb-1">
                      Width (m)
                    </label>
                    <input
                      type="number"
                      step="0.1"
                      min="0.1"
                      max="50"
                      placeholder="e.g. 3.2"
                      className="w-full px-3 py-2 border border-racing-green/30 rounded-md focus:outline-none focus:ring-2 focus:ring-racing-green/50"
                      value={roomDimensions.width}
                      onChange={(e) => setRoomDimensions(prev => ({...prev, width: e.target.value}))}
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-racing-green/70 mb-1">
                      Height (m)
                    </label>
                    <input
                      type="number"
                      step="0.1"
                      min="0.1"
                      max="10"
                      placeholder="e.g. 2.4"
                      className="w-full px-3 py-2 border border-racing-green/30 rounded-md focus:outline-none focus:ring-2 focus:ring-racing-green/50"
                      value={roomDimensions.height}
                      onChange={(e) => setRoomDimensions(prev => ({...prev, height: e.target.value}))}
                    />
                  </div>
                </div>
                <p className="text-xs text-racing-green/50">
                  Leave blank to skip enhanced analysis
                </p>
              </div>

              <div className="flex gap-3">
                <button
                  onClick={() => setPhase('ready')}
                  className="px-4 py-2 text-racing-green/70 hover:text-racing-green border border-racing-green/30 rounded-md transition-colors"
                >
                  Back
                </button>
                <button
                  onClick={startRecordingWithRoomInfo}
                  className="btn-primary flex-1"
                  disabled={!isRoomFormValid()}
                >
                  Start Analysis
                </button>
              </div>
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

        {phase === 'processing' && (
          <div className="space-y-4">
            <div className="text-6xl text-racing-green/40">⏳</div>
            <div className="text-racing-green font-medium">
              Processing your recording...
            </div>
            <ProgressBar progress={progress} />
            <p className="text-sm text-racing-green/60">
              Analyzing frequency response and room characteristics...
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
